package gragent

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/rfratto/gragent/internal/dag"
)

// The component interface is an extension of a dag.Node used for gragent.
type component interface {
	dag.Node

	// Evaluate evaluates the HCL given the context provided. The component may
	// use Evaluate to update internal state; i.e., updating the config if an
	// underlying object.
	//
	// Evaluate must return a struct that corresponds to the parsed HCL. The
	// returned value must be consumably by go-cty; see config.EncodeCty for more
	// information.
	Evaluate(*hcl.EvalContext, hcl.Body) (interface{}, hcl.Diagnostics)

	// CurrentState should return the latest state of the component that can be
	// referenced by other component. If there is no state to return,
	// CurrentState must return nil.
	//
	// CurrentState should return an instance of the same type every time it is
	// called.
	//
	// The returned value must be consumable by go-cty; see config.EncodeCty for
	// more information.
	CurrentState() interface{}

	// TODO(rfratto): CurrentStatus for debug-only state that can't be referenced
	// by other objects.

	// Run runs the component until ctx is canceled. Implementations must call
	// onStateChange to signal that their state has changed. Callers may then
	// call CurrentState to retrieve the state.
	Run(ctx context.Context, onStateChange func())
}
