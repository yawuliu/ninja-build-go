package main

type DyndepLoader struct {
	state_          *State
	disk_interface_ DiskInterface
	explanations_   *OptionalExplanations
}

func NewDyndepLoader(state *State, disk_interface DiskInterface, explanations *Explanations) *DyndepLoader {
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
func (this *DyndepLoader) LoadDyndeps(node *Node, err *string) bool                   {}
func (this *DyndepLoader) LoadDyndeps1(node *Node, ddf *DyndepFile, err *string) bool {}

func (this *DyndepLoader) LoadDyndepFile(file *Node, ddf *DyndepFile, err *string) bool {}

func (this *DyndepLoader) UpdateEdge(edge *Edge, dyndeps *Dyndeps, err *string) bool {}

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
