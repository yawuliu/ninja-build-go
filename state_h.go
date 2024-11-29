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
