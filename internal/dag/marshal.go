package dag

import (
	"bytes"
	"fmt"
)

// MarshalDOT marshals g into the DOT language defined by Graphviz.
func MarshalDOT(g *Graph) []byte {
	var buf bytes.Buffer

	fmt.Fprintln(&buf, "digraph {")
	fmt.Fprintf(&buf, "\trankdir=%q\n", "LR")

	fmt.Fprintf(&buf, "\n\t// Vertices:\n")
	for _, v := range g.Nodes() {
		fmt.Fprintf(&buf, "\t%q\n", v.Name())
	}

	fmt.Fprintf(&buf, "\n\t// Edges:\n")
	for _, edge := range g.Edges() {
		fmt.Fprintf(&buf, "\t%q -> %q\n", edge.From.Name(), edge.To.Name())
	}

	fmt.Fprintln(&buf, "}")
	return buf.Bytes()
}
