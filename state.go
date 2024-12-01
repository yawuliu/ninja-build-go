package main

import (
	"fmt"
	"slices"
)

// set
type DelayedEdges []*Edge // WeightedEdgeCmp

type Pool struct {
	name_ string

	/// |current_use_| is the total of the weights of the edges which are
	/// currently scheduled in the Plan (i.e. the edges in Plan::ready_).
	current_use_ int
	depth_       int
	delayed_     DelayedEdges
}

func NewPool(name string, depth int) *Pool {
	ret := Pool{}
	ret.name_ = name
	ret.current_use_ = 0
	ret.depth_ = depth
	// ret.delayed_()
	return &ret
}

// A depth of 0 is infinite
func (this *Pool) is_valid() bool   { return this.depth_ >= 0 }
func (this *Pool) depth() int       { return this.depth_ }
func (this *Pool) name() string     { return this.name_ }
func (this *Pool) current_use() int { return this.current_use_ }

// / true if the Pool might delay this edge
func (this *Pool) ShouldDelayEdge() bool { return this.depth_ != 0 }

// / informs this Pool that the given edge is committed to be run.
// / Pool will count this edge as using resources from this pool.
func (this *Pool) EdgeScheduled(edge *Edge) {
	if this.depth_ != 0 {
		this.current_use_ += edge.weight()
	}
}

// / informs this Pool that the given edge is no longer runnable, and should
// / relinquish its resources back to the pool
func (this *Pool) EdgeFinished(edge *Edge) {
	if this.depth_ != 0 {
		this.current_use_ -= edge.weight()
	}
}

// / adds the given edge to this Pool to be delayed.
func (this *Pool) DelayEdge(edge *Edge) {
	if this.depth_ != 0 {
		panic("this.depth_ != 0")
	}
	if !slices.Contains(this.delayed_, edge) {
		this.delayed_ = append(this.delayed_, edge)
	}
}

// / Pool will add zero or more edges to the ready_queue
func (this *Pool) RetrieveReadyEdges(ready_queue EdgePriorityQueue) {
	it := 0
	for it := range this.delayed_ {
		edge := this.delayed_[it]
		if this.current_use_+edge.weight() > this.depth_ {
			break
		}
		ready_queue.Add(edge)
		this.EdgeScheduled(edge)
		it++
	}
	this.delayed_ = this.delayed_[it:]
}

// / Dump the Pool and its edges (useful for debugging).
func (this *Pool) Dump() {
	fmt.Printf("%s (%d/%d) .\n", this.name_, this.current_use_, this.depth_)
	for _, it := range this.delayed_ {
		fmt.Printf("\t")
		it.Dump("")
	}
}

type Paths map[string]*Node

type State struct {
	paths_ Paths

	/// All the pools used in the graph.
	pools_ map[string]*Pool

	/// All the edges of the graph.
	edges_ []*Edge

	bindings_ BindingEnv
	defaults_ []*Node
}

func NewState() *State {
	ret := State{}
	ret.bindings_.AddRule(kPhonyRule)
	ret.AddPool(kDefaultPool)
	ret.AddPool(kConsolePool)
	return &ret
}

func (this *State) AddPool(pool *Pool) {
	if this.LookupPool(pool.name()) == nil {
		panic("AddPool")
	}
	this.pools_[pool.name()] = pool
}

func (this *State) LookupPool(pool_name string) *Pool {
	i, ok := this.pools_[pool_name]
	if !ok {
		return nil
	}
	return i
}

func (this *State) AddEdge(rule *Rule) *Edge {
	edge := NewEdge()
	edge.rule_ = rule
	edge.pool_ = kDefaultPool
	edge.env_ = &this.bindings_
	edge.id_ = len(this.edges_)
	this.edges_ = append(this.edges_, edge)
	return edge
}

func (this *State) GetNode(path string, slash_bits uint64) *Node {
	node := this.LookupNode(path)
	if node != nil {

		return node
	}
	node = NewNode(path, slash_bits)
	this.paths_[node.path()] = node
	return node
}

func (this *State) LookupNode(path string) *Node {
	i, ok := this.paths_[path]
	if ok {
		return i
	}
	return nil
}

func (this *State) SpellcheckNode(path string) *Node {
	kAllowReplacements := true
	kMaxValidEditDistance := 3

	min_distance := kMaxValidEditDistance + 1
	var result *Node = nil
	for first, second := range this.paths_ {
		distance := EditDistance(
			first, path, kAllowReplacements, kMaxValidEditDistance)
		if distance < min_distance && second != nil {
			min_distance = distance
			result = second
		}
	}
	return result
}

func (this *State) AddIn(edge *Edge, path string, slash_bits uint64) {
	node := this.GetNode(path, slash_bits)
	node.set_generated_by_dep_loader(false)
	edge.inputs_ = append(edge.inputs_, node)
	node.AddOutEdge(edge)
}

func (this *State) AddOut(edge *Edge, path string, slash_bits uint64, err *string) bool {
	node := this.GetNode(path, slash_bits)
	other := node.in_edge()
	if other != nil {
		if other == edge {
			*err = path + " is defined as an output multiple times"
		} else {
			*err = "multiple rules generate " + path
		}
		return false
	}
	edge.outputs_ = append(edge.outputs_, node)
	node.set_in_edge(edge)
	node.set_generated_by_dep_loader(false)
	return true
}

func (this *State) AddValidation(edge *Edge, path string, slash_bits uint64) {
	node := this.GetNode(path, slash_bits)
	edge.validations_ = append(edge.validations_, node)
	node.AddValidationOutEdge(edge)
	node.set_generated_by_dep_loader(false)
}

func (this *State) AddDefault(path string, err *string) bool {
	node := this.LookupNode(path)
	if node == nil {
		*err = "unknown target '" + path + "'"
		return false
	}
	this.defaults_ = append(this.defaults_, node)
	return true
}

func (this *State) RootNodes(err *string) []*Node {
	root_nodes := []*Node{}
	// Search for nodes with no output.
	for _, e := range this.edges_ {
		for _, out := range e.outputs_ {
			if len(out.out_edges()) == 0 {
				root_nodes = append(root_nodes, out)
			}
		}
	}

	if len(this.edges_) != 0 && len(root_nodes) == 0 {
		*err = "could not determine root nodes of build graph"
	}

	return root_nodes
}

func (this *State) DefaultNodes(err *string) []*Node {
	if len(this.defaults_) == 0 {
		return this.RootNodes(err)
	} else {
		return this.defaults_
	}

}

func (this *State) Reset() {
	for _, second := range this.paths_ {
		second.ResetState()
	}
	for _, e := range this.edges_ {
		e.outputs_ready_ = false
		e.deps_loaded_ = false
		e.mark_ = VisitNone
	}
}

func (this *State) Dump() {
	for _, second := range this.paths_ {
		node := second
		fmt.Printf("%s %s [id:%d]\n",
			node.path(),
			func() string {
				if node.status_known() {
					if node.dirty() {
						return "dirty"
					}
					return "clean"
				}
				return "unknown"
			}(),
			node.id())
	}
	if len(this.pools_) != 0 {
		fmt.Printf("resource_pools:\n")
		for _, second := range this.pools_ {
			if second.name() != "" {
				second.Dump()
			}
		}
	}
}
