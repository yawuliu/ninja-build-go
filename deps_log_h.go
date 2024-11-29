package main

import "os"

type DepsLog struct {
	needs_recompaction_ bool
	file_               *os.File
	file_path_          string

	/// Maps id . Node.
	nodes_ []*Node
	/// Maps id . deps of that id.
	deps_ []*Deps
}

type Deps struct {
	mtime      TimeStamp
	node_count int
	nodes      []*Node
}

func NewDeps(mtime int64, node_count int) *Deps {
	ret := Deps{}
	ret.mtime = TimeStamp(mtime)
	ret.node_count = node_count
	ret.nodes = make([]*Node, node_count)
	return &ret
}

func (this *Deps) ReleaseDeps() {
	this.nodes = []*Node{}
}
