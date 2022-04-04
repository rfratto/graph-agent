package dag

// WalkFunc is the type of function called by Walk* functions to visit a
// specific Node on a Graph.
//
// Walk* functions will abort if a WalkFunc return an error.
type WalkFunc func(n Node) error

// Walk performs a depth-first search of outgoing edges for all nodes in start.
// fn will be invoked for each node encountered. The order in which nodes are
// visited is not guaranteed.
//
// Walk does not visit nodes unreachable from start.
func Walk(g *Graph, start []Node, fn WalkFunc) error {
	var (
		visited   = make(nodeSet)
		unchecked = make([]Node, 0, len(start))
	)

	// Pre-fill the set of nodes to check from the start list.
	unchecked = append(unchecked, start...)

	// Iterate through each node in unchecked, visiting new nodes and adding
	// their outgoing edges to the unchecked list until we have processed all
	// reachable nodes.
	for len(unchecked) > 0 {
		check := unchecked[len(unchecked)-1]
		unchecked = unchecked[:len(unchecked)-1]

		if visited.Has(check) {
			continue
		}
		visited.Add(check)

		if err := fn(check); err != nil {
			return err
		}

		for n := range g.outEdges[check] {
			unchecked = append(unchecked, n)
		}
	}

	return nil
}

// WalkReverse performs a depth-first search of incoming edges for all nodes in
// start. fn will be invoked for each node encountered. The order in which
// nodes are visited is not guaranteed.
//
// WalkReverse does not visit nodes unreachable from start.
func WalkReverse(g *Graph, start []Node, fn WalkFunc) error {
	var (
		visited   = make(nodeSet)
		unchecked = make([]Node, 0, len(start))
	)

	// Pre-fill the set of nodes to check from the start list.
	unchecked = append(unchecked, start...)

	// Iterate through each node in unchecked, visiting new nodes and adding
	// their outgoing edges to the unchecked list until we have processed all
	// reachable nodes.
	for len(unchecked) > 0 {
		check := unchecked[len(unchecked)-1]
		unchecked = unchecked[:len(unchecked)-1]

		if visited.Has(check) {
			continue
		}
		visited.Add(check)

		if err := fn(check); err != nil {
			return err
		}

		for n := range g.inEdges[check] {
			unchecked = append(unchecked, n)
		}
	}

	return nil
}

// WalkTopological performs walks g topologically in dependency order: a node
// will not be visited until its outgoing edges are visited first.
func WalkTopological(g *Graph, fn WalkFunc) error {
	// NOTE(rfratto): WalkTopological is an implementation of Kahn's alogrithm
	// which leaves g unmodified.

	var (
		leaves = Leaves(g)

		visited   = make(nodeSet)
		unchecked = make([]Node, 0, len(leaves))

		remainingDeps = make(map[Node]int)
	)

	// Pre-fill the set of nodes to check from the start list.
	unchecked = append(unchecked, leaves...)

	for len(unchecked) > 0 {
		check := unchecked[len(unchecked)-1]
		unchecked = unchecked[:len(unchecked)-1]

		if visited.Has(check) {
			continue
		}
		visited.Add(check)

		if err := fn(check); err != nil {
			return err
		}

		// Iterate through the incoming edges to check and queue nodes if we're the
		// last edge to be walked.
		for n := range g.inEdges[check] {
			// remainingDeps starts with the number of edges, and we subtract one for
			// each outgoing edge that's visited.
			if _, ok := remainingDeps[n]; !ok {
				remainingDeps[n] = len(g.outEdges[n])
			}
			remainingDeps[n]--

			// Only enqueue the incoming edge once all of its outgoing edges have
			// been consumed. This prevents it from being visited before its
			// dependencies.
			if remainingDeps[n] == 0 {
				unchecked = append(unchecked, n)
			}
		}
	}

	return nil
}

// Leaves returns the set of Nodes in g which have no dependencies. This makes
// them safe to pass to WalkReverse to walk the graph in reverse-dependency
// order.
func Leaves(g *Graph) []Node {
	var res []Node

	for n := range g.nodes {
		if len(g.outEdges[n]) == 0 {
			res = append(res, n)
		}
	}

	return res
}
