package components

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/rfratto/gragent/internal/promutils"
)

// RemoteWrite is a thin wrapper around Prometheus remote_write that supports
// Gragent capabilities.
type RemoteWrite struct {
	logger       log.Logger
	reg          prometheus.Registerer
	name, walDir string

	configMut sync.Mutex
	rwc       *config.RemoteWriteConfig
	reloadRWC chan struct{}
}

// NewRemoteWrite creates a new, unstarted remote_write.
func NewRemoteWrite(l log.Logger, name, walDir string) *RemoteWrite {
	return &RemoteWrite{
		logger: log.With(l, "component", "remote_write"),
		reg:    prometheus.WrapRegistererWith(prometheus.Labels{"remote_write": name}, prometheus.DefaultRegisterer),

		name:   name,
		walDir: walDir,

		reloadRWC: make(chan struct{}, 1),
	}
}

func (rw *RemoteWrite) Configure(rwc *config.RemoteWriteConfig) {
	rw.configMut.Lock()
	defer rw.configMut.Unlock()

	// Update our most recent config
	rw.rwc = rwc

	select {
	case rw.reloadRWC <- struct{}{}:
	default:
		// Something is already queued, don't need to do anything
	}
}

// Run runs the RemoteWrite until ctx is canceled. The updated function is
// unused; RemoteWrite has no observable state that can be referenced.
func (rw *RemoteWrite) Run(ctx context.Context, updated func()) {
	// Because we might call Run multiple times during the lifecycle of the
	// application, we have to make sure that any metrics that get registered are
	// removed before the next invocation. This also prevents metrics from
	// leaking when remote_writes get taken away.
	ureg := promutils.WrapWithUnregisterer(rw.reg)
	defer ureg.UnregisterAll()

	rs := remote.NewStorage(rw.logger, ureg, fakeStartTimeCallback, rw.walDir, 30*time.Second, nil)
	defer rs.Close()

	for {
		select {
		case <-ctx.Done():
			return

		case <-rw.reloadRWC:
			// Grab the config out of the mutex
			rw.configMut.Lock()
			conf := rw.rwc
			rw.configMut.Unlock()

			fullConfig := &config.Config{
				RemoteWriteConfigs: []*config.RemoteWriteConfig{conf},
			}

			err := rs.ApplyConfig(fullConfig)
			if err != nil {
				level.Error(rw.logger).Log("msg", "failed to apply remote_write config", "err", err)
			}
		}
	}
}

func fakeStartTimeCallback() (int64, error) {
	return 0, nil
}
