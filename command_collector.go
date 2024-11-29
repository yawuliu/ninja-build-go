package main

type CommandCollector struct {
	visited_nodes_ map[*Node]bool
	visited_edges_ map[*Edge]bool

	/// we use a vector to preserve order from requisites to their dependents.
	/// This may help LSP server performance in languages that support modules,
	/// but it also ensures that the output of `-t compdb-targets foo` is
	/// consistent, which is useful in regression tests.
	in_edges []*Edge
}

func (this *CommandCollector) CollectFrom(node *Node) {
	if node == nil {
		panic("node==nil")
	}
	if _, ok := this.visited_nodes_[node]; ok {
		return
	}
	this.visited_nodes_[node] = true

	edge := node.in_edge()
	if edge == nil {
		return
	}
	if _, ok := this.visited_edges_[edge]; ok {
		return
	}
	this.visited_edges_[edge] = true

	for _, input_node := range edge.inputs_ {
		this.CollectFrom(input_node)
	}

	if !edge.is_phony() {
		this.in_edges = append(this.in_edges, edge)
	}
}
