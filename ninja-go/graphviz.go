package ninja_go

import (
	"fmt"
	"strings"
)

// / Runs the process of creating GraphViz .dot file output.
type GraphViz struct {
	dyndep_loader_ *DyndepLoader
	visited_nodes_ map[*Node]bool
	visited_edges_ EdgeSet
}

func NewGraphViz(state *State, disk_interface DiskInterface) *GraphViz {
	ret := GraphViz{}
	ret.dyndep_loader_ = NewDyndepLoader(state, disk_interface, nil)
	return &ret
}

func (this *GraphViz) Start() {
	fmt.Printf("digraph ninja {\n")
	fmt.Printf("rankdir=\"LR\"\n")
	fmt.Printf("node [fontsize=10, shape=box, height=0.25]\n")
	fmt.Printf("edge [fontsize=10]\n")
}
func (this *GraphViz) AddTarget(node *Node) {
	if _, ok := this.visited_nodes_[node]; ok {
		return
	}

	pathstr := node.path()
	pathstr = strings.ReplaceAll(pathstr, "\\", "/")
	fmt.Printf("\"%p\" [label=\"%s\"]\n", node, pathstr)
	this.visited_nodes_[node] = true

	edge := node.in_edge()

	if edge == nil {
		// Leaf node.
		// Draw as a rect?
		return
	}

	if _, ok := this.visited_edges_[edge]; ok {
		return
	}
	this.visited_edges_[edge] = true

	if edge.dyndep_ != nil && edge.dyndep_.dyndep_pending() {
		err := ""
		if !this.dyndep_loader_.LoadDyndeps(edge.dyndep_, &err) {
			Warning("%s\n", err)
		}
	}

	if len(edge.inputs_) == 1 && len(edge.outputs_) == 1 {
		// Can draw simply.
		// Note extra space before label text -- this is cosmetic and feels
		// like a graphviz bug.
		fmt.Printf("\"%p\" . \"%p\" [label=\" %s\"]\n", edge.inputs_[0], edge.outputs_[0], edge.rule_.name())
	} else {
		fmt.Printf("\"%p\" [label=\"%s\", shape=ellipse]\n", edge, edge.rule_.name())
		for _, out := range edge.outputs_ {
			fmt.Printf("\"%p\" . \"%p\"\n", edge, *out)
		}
		for i, in := range edge.inputs_ {
			order_only := ""
			if edge.is_order_only(int64(i)) {
				order_only = " style=dotted"
			}
			fmt.Printf("\"%p\" . \"%p\" [arrowhead=none%s]\n", (*in), edge, order_only)
		}
	}

	for _, in := range edge.inputs_ {
		this.AddTarget(in)
	}
}
func (this *GraphViz) Finish() {
	fmt.Printf("}\n")
}
