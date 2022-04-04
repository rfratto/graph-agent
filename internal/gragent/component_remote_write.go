package gragent

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/rfratto/gragent/internal/config"
)

type remoteWriteBlock struct {
	Name string `hcl:"name,label"`

	Body   hcl.Body `hcl:",body"`
	Remain hcl.Body `hcl:",remain"`
}

type remoteWriteComponent struct {
	id string
}

func newRemoteWriteComponent(id string) *remoteWriteComponent {
	return &remoteWriteComponent{
		id: id,
	}
}

func (c *remoteWriteComponent) Name() string { return c.id }

func (c *remoteWriteComponent) Evaluate(ectx *hcl.EvalContext, b hcl.Body) (interface{}, hcl.Diagnostics) {
	var cfg config.RemoteWrite

	diags := config.DecodeHCL(ectx, b, &cfg)
	if diags.HasErrors() {
		return nil, diags
	}

	// TODO(rfratto): do something

	return cfg, diags
}

func (c *remoteWriteComponent) CurrentState() interface{} {
	// There's no exposed state from remoteWriteComponent
	return nil
}

func (c *remoteWriteComponent) Run(ctx context.Context, onStateChange func()) {
	<-ctx.Done()
}
