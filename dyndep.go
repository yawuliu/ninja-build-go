package main

type DyndepLoader struct {
	state_          *State
	disk_interface_ DiskInterface
	explanations_   *OptionalExplanations
}

func NewDyndepLoader(state *State, disk_interface DiskInterface, explanations Explanations) *DyndepLoader {
	ret := DyndepLoader{}
	ret.state_ = state
	ret.disk_interface_ = disk_interface
	ret.explanations_ = explanations
	return &ret
}

// / Load a dyndep file from the given node's path and update the
// / build graph with the new information.  One overload accepts
// / a caller-owned 'DyndepFile' object in which to store the
// / information loaded from the dyndep file.
func (this *DyndepLoader) LoadDyndeps(node *Node, err *string) bool {
	ddf := DyndepFile{}
	return this.LoadDyndeps1(node, ddf, err)
}
func (this *DyndepLoader) LoadDyndeps1(node *Node, ddf DyndepFile, err *string) bool {
	// We are loading the dyndep file now so it is no longer pending.
	node.set_dyndep_pending(false)

	// Load the dyndep information from the file.
	this.explanations_.Record(node, "loading dyndep file '%s'", node.path())

	if !this.LoadDyndepFile(node, ddf, err) {
		return false
	}

	// Update each edge that specified this node as its dyndep binding.
	out_edges := node.out_edges()
	for _, edge := range out_edges {
		if edge.dyndep_ != node {
			continue
		}

		ddi_second, ok := ddf[edge]
		if !ok {
			*err = ("'" + edge.outputs_[0].path() + "' " +
				"not mentioned in its dyndep file " +
				"'" + node.path() + "'")
			return false
		}

		ddi_second.used_ = true
		dyndeps := ddi_second
		if !this.UpdateEdge(edge, &dyndeps, err) {
			return false
		}
	}

	// Reject extra outputs in dyndep file.
	for dyndep_output_first, dyndep_output_second := range ddf {
		if !dyndep_output_second.used_ {
			edge := dyndep_output_first
			*err = ("dyndep file '" + node.path() + "' mentions output " +
				"'" + edge.outputs_[0].path() + "' whose build statement " +
				"does not have a dyndep binding for the file")
			return false
		}
	}

	return true
}

func (this *DyndepLoader) LoadDyndepFile(file *Node, ddf DyndepFile, err *string) bool {
	parser := NewDyndepParser(this.state_, this.disk_interface_, ddf)
	return parser.Load(file.path(), err, nil)
}

func (this *DyndepLoader) UpdateEdge(edge *Edge, dyndeps *Dyndeps, err *string) bool {
	// Add dyndep-discovered bindings to the edge.
	// We know the edge already has its own binding
	// scope because it has a "dyndep" binding.
	if dyndeps.restat_ {
		edge.env_.AddBinding("restat", "1")
	}

	// Add the dyndep-discovered outputs to the edge.
	edge.outputs_ = append(edge.outputs_, dyndeps.implicit_outputs_...)
	edge.implicit_outs_ += len(dyndeps.implicit_outputs_)

	// Add this edge as incoming to each new output.
	for _, node := range dyndeps.implicit_outputs_ {
		if node.in_edge() != nil {
			// This node already has an edge producing it.
			*err = "multiple rules generate " + node.path()
			return false
		}
		node.set_in_edge(edge)
	}

	// Add the dyndep-discovered inputs to the edge.
	edge.inputs_ = append(edge.inputs_, dyndeps.implicit_inputs_...)
	edge.implicit_deps_ += len(dyndeps.implicit_inputs_)

	// Add this edge as outgoing from each new input.
	for _, node := range dyndeps.implicit_inputs_ {
		node.AddOutEdge(edge)
	}

	return true
}

type DyndepFile map[*Edge]Dyndeps

type Dyndeps struct {
	used_             bool
	restat_           bool
	implicit_inputs_  []*Node
	implicit_outputs_ []*Node
}

func NewDyndeps() *Dyndeps {
	ret := Dyndeps{}
	ret.used_ = false
	ret.restat_ = false
	return &ret
}
