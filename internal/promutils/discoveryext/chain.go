package discoveryext

import (
	"context"

	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

// ChainConfig configures the Chain service discovery.
type ChainConfig struct {
	Input []*targetgroup.Group
}

var _ discovery.Config = (*ChainConfig)(nil)

// Name returns the name of the discovery.
func (c *ChainConfig) Name() string { return "chain" }

// NewDiscoverer converts the ChainConfig into a Discoverer.
func (c *ChainConfig) NewDiscoverer(o discovery.DiscovererOptions) (discovery.Discoverer, error) {
	return NewChainDiscoverer(c), nil
}

type ChainDiscoverer struct {
	c *ChainConfig
}

var _ discovery.Discoverer = (*ChainDiscoverer)(nil)

// NewChainDiscoverer returns a new ChainDiscoverer.
func NewChainDiscoverer(c *ChainConfig) *ChainDiscoverer {
	return &ChainDiscoverer{c: c}
}

// Run runs the ChainDiscoverer.
func (d *ChainDiscoverer) Run(ctx context.Context, up chan<- []*targetgroup.Group) {
	select {
	case <-ctx.Done():
	case up <- d.c.Input:
	}

	<-ctx.Done()
}
