package gragent

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/rfratto/gragent/internal/config"
)

type scrapeBlock struct {
	Name string `hcl:"name,label"`

	Body   hcl.Body `hcl:",body"`
	Remain hcl.Body `hcl:",remain"`
}

type scrapeComponent struct {
	id string
}

func newScrapeComponent(id string) *scrapeComponent {
	return &scrapeComponent{
		id: id,
	}
}

func (c *scrapeComponent) Name() string { return c.id }

func (c *scrapeComponent) Evaluate(ectx *hcl.EvalContext, b hcl.Body) (interface{}, hcl.Diagnostics) {
	var cfg config.MetricsScrape

	diags := config.DecodeHCL(ectx, b, &cfg)
	if diags.HasErrors() {
		return nil, diags
	}

	// TODO(rfratto): do something

	return cfg, diags
}

func (c *scrapeComponent) CurrentState() interface{} {
	// There's no exposed state from scrapeComponent
	return nil
}

func (c *scrapeComponent) Run(ctx context.Context, onStateChange func()) {
	<-ctx.Done()
}
