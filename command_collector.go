package main

type CommandCollector struct {
	visited_nodes_  map[*Node]bool
	svisited_edges_ map[*Node]bool

	/// we use a vector to preserve order from requisites to their dependents.
	/// This may help LSP server performance in languages that support modules,
	/// but it also ensures that the output of `-t compdb-targets foo` is
	/// consistent, which is useful in regression tests.
	in_edges []*Edge
}

func(this*CommandCollector) CollectFrom(node *Node) {
	if node==nil {
		panic("node==nil")
	}

	if (!this.visited_nodes_.insert(node).second) {
		return
	}

	edge := node.in_edge();
	if edge==nil || !this.visited_edges_.insert(edge).second {
		return
	}

	for (Node* input_node : edge.inputs_){
		this.CollectFrom(input_node)
	}

	if !edge.is_phony() {
		in_edges.push_back(edge)
	}
}