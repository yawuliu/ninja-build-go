package main

func NewDepsLog() *DepsLog {
	ret := DepsLog{}
	ret.needs_recompaction_ = false
	ret.file_ = nil
	return &ret
}
func (this *DepsLog) ReleaseDepsLog() {

}

// Writing (build-time) interface.
func (this *DepsLog) OpenForWrite(path string, err *string) bool {

}
func (this *DepsLog) RecordDeps(node *Node, mtime TimeStamp, nodes []*Node) bool {

}
func (this *DepsLog) RecordDeps2(node *Node, mtime TimeStamp, node_count int, nodes *Node) bool {

}
func (this *DepsLog) Close() {}

func (this *DepsLog) Load(path string, state *State, err *string) LoadStatus {}
func (this *DepsLog) GetDeps(node *Node) *Deps                               {}
func (this *DepsLog) GetFirstReverseDepsNode(node *Node) *Node               {}

// / Rewrite the known log entries, throwing away old data.
func (this *DepsLog) Recompact(path string, err *string) bool {}

// / Returns if the deps entry for a node is still reachable from the manifest.
// /
// / The deps log can contain deps entries for files that were built in the
// / past but are no longer part of the manifest.  This function returns if
// / this is the case for a given node.  This function is slow, don't call
// / it from code that runs on every build.
func IsDepsEntryLiveFor(node *Node) bool {

}

// / Used for tests.
func (this *DepsLog) nodes() []*Node { return this.nodes_ }
func (this *DepsLog) deps() []*Deps  { return this.deps_ }

// Updates the in-memory representation.  Takes ownership of |deps|.
// Returns true if a prior deps record was deleted.
func (this *DepsLog) UpdateDeps(out_id int, deps *Deps) bool {

}

// Write a node name record, assigning it an id.
func (this *DepsLog) RecordId(node *Node) bool {

}

// / Should be called before using file_. When false is returned, errno will
// / be set.
func (this *DepsLog) OpenForWriteIfNeeded() bool {

}
