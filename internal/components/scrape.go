package components

import (
	"context"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	promcfg "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/scrape"
	"github.com/prometheus/prometheus/storage"
)

// ScrapeConfig configures a Scrape. Unlike Prometheus, a ScrapeConfig is bound
// tightly to the set of Targets that should be scraped.
type ScrapeConfig struct {
	Config  *promcfg.ScrapeConfig
	Targets []*targetgroup.Group
}

// Scrape scrapes metrics from targets.
type Scrape struct {
	logger log.Logger
	app    storage.Appendable

	configMut    sync.Mutex
	sc           ScrapeConfig
	reloadConfig chan struct{}
}

// NewScrape creates a new Scrape. When running, collected metrics will be sent
// to the given app.
func NewScrape(l log.Logger, app storage.Appendable) *Scrape {
	return &Scrape{
		logger: log.With(l, "component", "scrape"),
		app:    app,

		reloadConfig: make(chan struct{}, 1),
	}
}

// Configure sets the current ScrapeConfig to collect metrics from.
func (s *Scrape) Configure(c ScrapeConfig) {
	s.configMut.Lock()
	defer s.configMut.Unlock()

	// Store our most recent config
	s.sc = c

	select {
	case s.reloadConfig <- struct{}{}:
	default:
		// Something is already queued, don't need to do anything
	}
}

// Run runs Scrape until ctx is canceled. The updated function is unused;
// Scrape has no observable state that can be referenced.
func (s *Scrape) Run(ctx context.Context, updated func()) {
	sm := scrape.NewManager(&scrape.Options{}, s.logger, s.app)
	defer sm.Stop()

	tsets := make(chan map[string][]*targetgroup.Group, 1)
	go sm.Run(tsets)

	for {
		select {
		case <-ctx.Done():
			return

		case <-s.reloadConfig:
			// Grab the config out of the mutex
			s.configMut.Lock()
			conf := s.sc
			s.configMut.Unlock()

			// TODO(rfratto): this will be really expensive when only the set of
			// targets changes, we might want some way to avoid the ApplyConfig every
			// update.
			fullConf := &promcfg.Config{
				ScrapeConfigs: []*promcfg.ScrapeConfig{conf.Config},
			}
			if err := sm.ApplyConfig(fullConf); err != nil {
				level.Error(s.logger).Log("msg", "failed to apply scrape config", "err", err)
				continue
			}

			// Send the current set of targets out.
			tsets <- map[string][]*targetgroup.Group{conf.Config.JobName: conf.Targets}
		}
	}
}
