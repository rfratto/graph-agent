package gragent

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/rfratto/gragent/internal/config"
	"github.com/rfratto/gragent/internal/promutils/discoveryext"
	"github.com/zclconf/go-cty/cty"
)

var targetGroupCapsuleTy = cty.Capsule("targetgroup", reflect.TypeOf(targetgroup.Group{}))

type discoveryBlock struct {
	Kind string `hcl:"kind,label"`
	Name string `hcl:"name,label"`

	Body   hcl.Body `hcl:",body"`
	Remain hcl.Body `hcl:",remain"`
}

type discoveryComponent struct {
	kind, name string
}

func newDiscoveryComponent(kind, name string) *discoveryComponent {
	return &discoveryComponent{
		kind: kind,
		name: name,
	}
}

func (c *discoveryComponent) Name() string { return fmt.Sprintf("discovery.%s.%s", c.kind, c.name) }

func (c *discoveryComponent) Evaluate(ectx *hcl.EvalContext, b hcl.Body) (interface{}, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	switch c.kind {
	case "static":
		cfg, val, diags := readStaticSD(ectx, b)
		if diags.HasErrors() {
			return val, diags
		}
		_ = cfg
		return val, diags

	case "chain":
		cfg, val, diags := readChainSD(ectx, b)
		if diags.HasErrors() {
			return val, diags
		}
		_ = cfg
		return val, diags

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unknown discovery kind",
			Detail:   fmt.Sprintf("Block %s has unknown discovery kind %q", c.Name(), c.kind),
			Subject:  blockRange(b),
		})
		return cty.NilVal, diags
	}
}

func readStaticSD(ectx *hcl.EvalContext, b hcl.Body) (discovery.StaticConfig, interface{}, hcl.Diagnostics) {
	var cfg config.DiscoveryStatic

	diags := config.DecodeHCL(ectx, b, &cfg)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	// Convert into the upstream type.
	// TODO(rfratto): should this be a function somewhere else?
	group := targetgroup.Group{Labels: make(model.LabelSet)}
	for _, target := range cfg.Hosts {
		group.Targets = append(group.Targets, model.LabelSet{
			model.AddressLabel: model.LabelValue(target),
		})
	}
	for key, value := range cfg.Labels {
		group.Labels[model.LabelName(key)] = model.LabelValue(value)
	}

	return discovery.StaticConfig{&group}, cfg, diags
}

func readChainSD(ectx *hcl.EvalContext, b hcl.Body) (*discoveryext.ChainConfig, interface{}, hcl.Diagnostics) {
	var cfg config.DiscoveryChain

	diags := config.DecodeHCL(ectx, b, &cfg)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	// Convert into the upstream type.
	// TODO(rfratto): should this be a function somewhere else?
	var finalGroups []*targetgroup.Group
	for _, group := range cfg.Input {
		finalGroup := targetgroup.Group{
			Targets: make([]model.LabelSet, 0, len(group.Targets)),
			Labels:  make(model.LabelSet, len(group.Labels)),
		}

		for _, target := range group.Targets {
			finalTarget := make(model.LabelSet, len(target))
			for key, value := range target {
				finalTarget[model.LabelName(key)] = model.LabelValue(value)
			}
			finalGroup.Targets = append(finalGroup.Targets, finalTarget)
		}

		for key, value := range group.Labels {
			finalGroup.Labels[model.LabelName(key)] = model.LabelValue(value)
		}

		finalGroups = append(finalGroups, &finalGroup)
	}

	return &discoveryext.ChainConfig{Input: finalGroups}, cfg, diags
}

func (c *discoveryComponent) CurrentState() interface{} {
	state := struct {
		Targets []config.TargetGroup `hcl:"targets" cty:"targets"`
	}{
		Targets: make([]config.TargetGroup, 0),
		// TODO(rfratto): populate state
	}

	return &state
}

func (c *discoveryComponent) Run(ctx context.Context, onStateChange func()) {
	<-ctx.Done()
}

func blockRange(b hcl.Body) *hcl.Range {
	sb, ok := b.(*hclsyntax.Body)
	if !ok {
		return nil
	}
	return sb.SrcRange.Ptr()
}
