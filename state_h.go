package main

type Pool struct {
	name_ string

	/// |current_use_| is the total of the weights of the edges which are
	/// currently scheduled in the Plan (i.e. the edges in Plan::ready_).
	current_use_ int
	depth_       int
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
