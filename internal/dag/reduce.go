package dag

// Reduce performs a transitive reduction on g. A transitive reduction removes
// as many edges as possible while maintaining the same "reachability" as the
// original graph.
func Reduce(g *Graph) {
	// A direct edge between two vertices can be removed if that same target
	// vertex is indirectly reachable through another edge.
	//
	// Iterate through all the vertices in the graph, performing a depth-first
	// search at its dependencies. Remove any edge where the target vertex is
	// directly reachable from the starting vertex.
	for u := range g.nodes {
		Walk(g, g.Dependencies(u), func(v Node) error {
			// Remove any (u, v') edge where a (v, v') edge also exists.
			for vPrime := range g.outEdges[v] {
				g.RemoveEdge(Edge{From: u, To: vPrime})
			}
			return nil
		})
	}
}
