package gragent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/rfratto/gragent/internal/config"
	"github.com/rfratto/gragent/internal/dag"
	"github.com/rfratto/gragent/internal/dag/graphviz"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/zclconf/go-cty/cty/gocty"
)

type rootBlock struct {
	Discovery   []discoveryBlock   `hcl:"discovery,block"`
	Scrape      []scrapeBlock      `hcl:"scrape,block"`
	RemoteWrite []remoteWriteBlock `hcl:"remote_write,block"`

	Body   hcl.Body `hcl:",body"`
	Remain hcl.Body `hcl:",remain"`
}

// System represents the gragent system.
type System struct {
	log        log.Logger
	configFile string

	graphMut sync.RWMutex
	graph    *dag.Graph
}

func NewSystem(l log.Logger, configFile string) *System {
	s := &System{
		log:        l,
		configFile: configFile,
		graph:      &dag.Graph{},
	}
	s.graph.Add(s) // Add the system as the root node.
	return s
}

// Name implements dag.Node.
func (s *System) Name() string { return "<root>" }

// Load reads the config file and updates the system to reflect what was
// read.
func (s *System) Load() error {
	s.graphMut.Lock()
	defer s.graphMut.Unlock()

	// TODO(rfratto): this won't work yet for subseqent loads.

	bb, err := os.ReadFile(s.configFile)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	file, diags := hclsyntax.ParseConfig(bb, s.configFile, hcl.InitialPos)
	if diags.HasErrors() {
		return diags
	}

	var root rootBlock
	decodeDiags := gohcl.DecodeBody(file.Body, nil, &root)
	diags = diags.Extend(decodeDiags)
	if diags.HasErrors() {
		return diags
	}

	// TODO(rfratto): persist between reloads?
	var (
		idNodeMap    = make(map[string]dag.Node)
		referenceMap = make(map[dag.Node]reference)
		bodyLookup   = make(map[dag.Node]hcl.Body)
	)

	// TODO(rfratto): we need to evaluate the remainder of the root block here
	// for global settings.

	// Once we've parsed the config, we have to start creating components and
	// populating our DAG.
	//
	// TODO(rfratto): match to existing components based on ID. We only have to
	// make new components if the ID is new.
	for _, disc := range root.Discovery {
		var (
			id    = reference{"discovery", disc.Kind, disc.Name}
			idStr = id.String()
		)

		c := newDiscoveryComponent(disc.Kind, disc.Name)
		s.graph.Add(c)
		s.graph.AddEdge(dag.Edge{From: s, To: c})

		idNodeMap[idStr] = c
		referenceMap[c] = id
		bodyLookup[c] = disc.Body
	}

	for _, scrape := range root.Scrape {
		var (
			id    = reference{"scrape", scrape.Name}
			idStr = id.String()
		)

		c := newScrapeComponent(idStr)
		s.graph.Add(c)
		s.graph.AddEdge(dag.Edge{From: s, To: c})

		idNodeMap[idStr] = c
		referenceMap[c] = id
		bodyLookup[c] = scrape.Body
	}

	for _, rw := range root.RemoteWrite {
		var (
			id    = reference{"remote_write", rw.Name}
			idStr = id.String()
		)

		c := newRemoteWriteComponent(idStr)
		s.graph.Add(c)
		s.graph.AddEdge(dag.Edge{From: s, To: c})

		idNodeMap[idStr] = c
		referenceMap[c] = id
		bodyLookup[c] = rw.Body
	}

	for origin, body := range bodyLookup {
		traversals := expressionsFromSyntaxBody(body.(*hclsyntax.Body))
		for _, t := range traversals {
			lookup, pdiags := parseReference(t)
			diags.Extend(pdiags)
			if lookup == nil {
				continue
			}

			target := idNodeMap[lookup.String()]
			if target != nil {
				s.graph.AddEdge(dag.Edge{From: origin, To: target})
			}
		}
	}
	if diags.HasErrors() {
		return diags
	}

	// Wiring dependencies probably caused a mess. Reduce to the minimum set of
	// edges.
	dag.Reduce(s.graph)

	var (
		wctx walkContext
		ectx = hcl.EvalContext{
			Variables: make(map[string]cty.Value),
			Functions: map[string]function.Function{
				"concat": stdlib.ConcatFunc,
			},
		}
	)

	// At this point, our DAG is completely formed and we can start to evaluate
	// components. Perform a topological sort and evaluate everything.
	err = dag.WalkTopological(s.graph, func(n dag.Node) error {
		c, ok := n.(component)
		if !ok {
			// Not a component. Move on.
			return nil
		}

		body, ok := bodyLookup[c]
		if !ok {
			return fmt.Errorf("unexpected missing hcl.Body for %s", n.Name())
		}

		level.Debug(s.log).Log("msg", "evaluating node", "id", n.Name())

		inputVal, ediags := c.Evaluate(&ectx, body)
		if ediags.HasErrors() {
			return ediags
		}
		inputCtyVal, err := config.EncodeCty(inputVal)
		if err != nil {
			return err
		}

		cachedValue := inputCtyVal

		stateVal := c.CurrentState()
		if stateVal != nil {
			stateCtyVal, err := config.EncodeCty(stateVal)
			if err != nil {
				return err
			}
			cachedValue = mergeState(inputCtyVal, stateCtyVal)
		}

		wctx.vals = append(wctx.vals, referenceValue{
			Key:   referenceMap[n],
			Value: cachedValue,
		})
		wctx.FillEvalContext(&ectx)

		return nil
	})

	// TODO(rfratto): getting to the point now where we really need a /status
	// endpoint to write everything as HCL back to the user.
	//
	// While doing this, we'll want to query the most recent state just so
	// we get an up to date view, but we'll also want to include things like
	// last_eval_time or last_emit_time alongside the status so users can tell if
	// a component hasn't been updated just yet.

	return err
}

// Run runs the system. Run will block until there's an error or ctx is
// canceled. The returned error will only be non-nil when there was an
// error during running.
func (s *System) Run(ctx context.Context) error {
	// TODO(rfratto): magic
	<-ctx.Done()
	return nil
}

// GraphHandler returns an http.Handler that renders the system's DAG as an
// SVG.
func (s *System) GraphHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		s.graphMut.RLock()
		contents := dag.MarshalDOT(s.graph)
		s.graphMut.RUnlock()

		svgBytes, err := graphviz.Dot(contents, "svg")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		io.Copy(w, bytes.NewReader(svgBytes))
	}
}

// expressionsFromSyntaxBody returcses through body and finds all variable
// references.
func expressionsFromSyntaxBody(body *hclsyntax.Body) []hcl.Traversal {
	var exprs []hcl.Traversal

	for _, attrib := range body.Attributes {
		exprs = append(exprs, attrib.Expr.Variables()...)
	}
	for _, block := range body.Blocks {
		exprs = append(exprs, expressionsFromSyntaxBody(block.Body)...)
	}

	return exprs
}

// mergeState merges two the inputs of a component with its current state.
// mergeState panics if a key exits in both inputs and store or if neither
// argument is an object.
func mergeState(inputs, state cty.Value) cty.Value {
	if !inputs.Type().IsObjectType() {
		panic("component input must be object type")
	}
	if !state.Type().IsObjectType() {
		panic("component state must be object type")
	}

	var (
		inputMap = inputs.AsValueMap()
		stateMap = state.AsValueMap()
	)

	mergedMap := make(map[string]cty.Value, len(inputMap)+len(stateMap))
	for key, value := range inputMap {
		mergedMap[key] = value
	}
	for key, value := range stateMap {
		if _, exist := mergedMap[key]; exist {
			panic(fmt.Sprintf("component state overriding key %s", key))
		}
		mergedMap[key] = value
	}

	return cty.ObjectVal(mergedMap)
}

type walkContext struct {
	vals []referenceValue
}

type referenceValue struct {
	Key   reference
	Value cty.Value
}

// FillEvalContext fills an hcl.EvalContext with the referenceable values from
// wc.
func (wc *walkContext) FillEvalContext(ectx *hcl.EvalContext) {
	if ectx.Variables == nil {
		ectx.Variables = make(map[string]cty.Value)
	}

	// Create a tree of maps keyed by block label until we store the component
	// value.
	//
	// The value here is a set of nested map[string]interface{} (one per element
	// in the addressible name of a component), with the exception of the final
	// map, which is a map[string]cty.Value.
	//
	// This is an unfortunate hack, but is currently necessary due to a bug in
	// go-cty where gocty.ToCtyValue doesn't unwrap an interface{} holding a
	// cty.Value.
	componentValues := make(map[string]interface{})

	for _, val := range wc.vals {
		var currentLayer interface{} = componentValues

		for i, layerName := range val.Key {
			// We need to unfortunately be clever here. There's a bug in
			switch {
			case i+1 < len(val.Key)-1:
				// map[string]map[string]interface{}
				upperLayer := currentLayer.(map[string]interface{})
				next, ok := upperLayer[layerName].(map[string]interface{})
				if !ok {
					next = make(map[string]interface{})
					upperLayer[layerName] = next
				}
				currentLayer = next
			case i+1 == len(val.Key)-1:
				// map[string]map[string]cty.Value
				upperLayer := currentLayer.(map[string]interface{})
				next, ok := upperLayer[layerName].(map[string]cty.Value)
				if !ok {
					next = make(map[string]cty.Value)
					upperLayer[layerName] = next
				}
				currentLayer = next
			case i+1 == len(val.Key):
				// map[string]cty.Value
				upperLayer := currentLayer.(map[string]cty.Value)
				upperLayer[layerName] = val.Value
			default:
				panic("BUG: bad logic in switch statement")
			}
		}
	}

	// gocty.ImpliedTy won't work with map[string]interface{}, so we need to
	// recursively build up the type.
	componentTypes := make(map[string]cty.Type, len(componentValues))
	for cname, val := range componentValues {
		componentTypes[cname] = evalContextType(val)
	}

	for name := range componentValues {
		ctyVal, err := gocty.ToCtyValue(componentValues[name], componentTypes[name])
		if err != nil {
			panic(err)
		}
		ectx.Variables[name] = ctyVal
	}
}

// evalContextType builds a cty.Type for an *hcl.EvalContext variable. This is
// needed over gocty.ImpliedTy as ImpliedTy doesn't know what to do with
// interface{}.
func evalContextType(v interface{}) cty.Type {
	switch v := v.(type) {
	case map[string]interface{}:
		tys := make(map[string]cty.Type, len(v))
		for k, val := range v {
			tys[k] = evalContextType(val)
		}
		return cty.Object(tys)
	case map[string]cty.Value:
		tys := make(map[string]cty.Type, len(v))
		for k, val := range v {
			tys[k] = val.Type()
		}
		return cty.Object(tys)
	default:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
}
