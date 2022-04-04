package dag

type Node interface {
	// Name returns the display name of the Node.
	Name() string
}

type Edge struct{ From, To Node }

// Graph is a directed acyclic graph. The zero value is ready for use. Graphs
// cannot be used concurrently.
type Graph struct {
	nodes    nodeSet
	outEdges map[Node]nodeSet // Outgoing edges for a given Node
	inEdges  map[Node]nodeSet // Incoming edges for a given Node
}

type nodeSet map[Node]struct{}

// Add adds n into ns if it doesn't already exist.
func (ns nodeSet) Add(n Node) { ns[n] = struct{}{} }

// Has returns true if n is inside ns.
func (ns nodeSet) Has(n Node) bool {
	_, ok := ns[n]
	return ok
}

// init prepares g for writing.
func (g *Graph) init() {
	if g.nodes == nil {
		g.nodes = make(nodeSet)
	}
	if g.outEdges == nil {
		g.outEdges = make(map[Node]nodeSet)
	}
	if g.inEdges == nil {
		g.inEdges = make(map[Node]nodeSet)
	}
}

// Add adds n into g. Add will be a no-op if n already exists in g.
func (g *Graph) Add(n Node) {
	g.init()
	g.nodes.Add(n)
}

// AddEdge adds an edge e into the graph. AddEdge will be a no-op if e already
// exists in g.
//
// AddEdge will panic if the nodes from e don't exist in g.
func (g *Graph) AddEdge(e Edge) {
	g.init()

	if !g.nodes.Has(e.From) || !g.nodes.Has(e.To) {
		panic("adding edge with a node that doesn't exist in graph")
	}

	inSet, ok := g.inEdges[e.To]
	if !ok {
		inSet = make(nodeSet)
		g.inEdges[e.To] = inSet
	}
	inSet.Add(e.From)

	outSet, ok := g.outEdges[e.From]
	if !ok {
		outSet = make(nodeSet)
		g.outEdges[e.From] = outSet
	}
	outSet.Add(e.To)
}

// RemoveEdge removes an edge e into the graph. RemoveEdge is a no-op if e
// doesn't exist in g.
func (g *Graph) RemoveEdge(e Edge) {
	inSet, ok := g.inEdges[e.To]
	if ok {
		delete(inSet, e.From)
	}

	outSet, ok := g.outEdges[e.From]
	if ok {
		delete(outSet, e.To)
	}
}

// Nodes returns the set of nodes in g.
func (g *Graph) Nodes() []Node {
	nodes := make([]Node, 0, len(g.nodes))
	for n := range g.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// Edges returns the set of all edges in g.
func (g *Graph) Edges() []Edge {
	var edges []Edge
	for from, tos := range g.outEdges {
		for to := range tos {
			edges = append(edges, Edge{From: from, To: to})
		}
	}
	return edges
}

// Dependants returns the list of nodes which depend on n. (i.e., all nodes for
// which an edge to n is defined).
func (g *Graph) Dependants(n Node) []Node {
	sourceDependants := g.inEdges[n]

	dependants := make([]Node, 0, len(sourceDependants))
	for dep := range sourceDependants {
		dependants = append(dependants, dep)
	}
	return dependants
}

// Dependencies returns the list of nodes that n depends on. (i.e., all nodes
// for which an edge from n is defined).
func (g *Graph) Dependencies(n Node) []Node {
	sourceDependencies := g.outEdges[n]

	dependencies := make([]Node, 0, len(sourceDependencies))
	for dep := range sourceDependencies {
		dependencies = append(dependencies, dep)
	}
	return dependencies
}
