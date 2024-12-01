package main

import (
	"strings"
)

type VisitMark int8

const (
	VisitNone    VisitMark = 0
	VisitInStack           = 1
	VisitDone              = 2
)

type Edge struct {
	rule_                    *Rule
	pool_                    *Pool
	inputs_                  []*Node
	outputs_                 []*Node
	validations_             []*Node
	dyndep_                  *Node
	env_                     *BindingEnv
	mark_                    VisitMark
	id_                      int
	critical_path_weight_    int64
	outputs_ready_           bool
	deps_loaded_             bool
	deps_missing_            bool
	generated_by_dep_loader_ bool
	command_start_time_      TimeStamp

	// There are three types of inputs.
	// 1) explicit deps, which show up as $in on the command line;
	// 2) implicit deps, which the target depends on implicitly (e.g. C headers),
	//                   and changes in them cause the target to rebuild;
	// 3) order-only deps, which are needed before the target builds but which
	//                     don't cause the target to rebuild.
	// These are stored in inputs_ in that order, and we keep counts of
	// #2 and #3 when we need to access the various subsets.
	implicit_deps_   int
	order_only_deps_ int

	// There are two types of outputs.
	// 1) explicit outs, which show up as $out on the command line;
	// 2) implicit outs, which the target generates but are not part of $out.
	// These are stored in outputs_ in that order, and we keep a count of
	// #2 to use when we need to access the various subsets.
	implicit_outs_ int

	// Historical info: how long did this edge take last time,
	// as per .ninja_log, if known? Defaults to -1 if unknown.
	prev_elapsed_time_millis int64
}

type ExistenceStatus int8

const (
	/// The file hasn't been examined.
	ExistenceStatusUnknown ExistenceStatus = 0
	/// The file doesn't exist. mtime_ will be the latest mtime of its dependencies.
	ExistenceStatusMissing = 1
	/// The path is an actual file. mtime_ will be the file's mtime.
	ExistenceStatusExists = 2
)

type Node struct {
	path_ string

	/// Set bits starting from lowest for backslashes that were normalized to
	/// forward slashes by CanonicalizePath. See |PathDecanonicalized|.
	slash_bits_ uint64

	/// Possible values of mtime_:
	///   -1: file hasn't been examined
	///   0:  we looked, and file doesn't exist
	///   >0: actual file's mtime, or the latest mtime of its dependencies if it doesn't exist
	mtime_ TimeStamp

	exists_ ExistenceStatus

	/// Dirty is true when the underlying file is out-of-date.
	/// But note that Edge::outputs_ready_ is also used in judging which
	/// edges to build.
	dirty_ bool

	/// Store whether dyndep information is expected from this node but
	/// has not yet been loaded.
	dyndep_pending_ bool

	/// Set to true when this node comes from a depfile, a dyndep file or the
	/// deps log. If it does not have a producing edge, the build should not
	/// abort if it is missing (as for regular source inputs). By default
	/// all nodes have this flag set to true, since the deps and build logs
	/// can be loaded before the manifest.
	generated_by_dep_loader_ bool

	/// The Edge that produces this Node, or NULL when there is no
	/// known edge to produce it.
	in_edge_ *Edge

	/// All Edges that use this Node as an input.
	out_edges_ []*Edge

	/// All Edges that use this Node as a validation.
	validation_out_edges_ []*Edge

	/// A dense integer id for the node, assigned and used by DepsLog.
	id_ int
}
type EdgeSet map[*Edge]bool //, EdgeCmp>

type InputsCollector struct {
	inputs_        []*Node
	visited_nodes_ map[*Node]bool
}

// / DependencyScan manages the process of scanning the files in a graph
// / and updating the dirty/outputs_ready state of all the nodes and edges.
type DependencyScan struct {
	build_log_      *BuildLog
	disk_interface_ DiskInterface
	dep_loader_     *ImplicitDepLoader
	dyndep_loader_  *DyndepLoader
	explanations_   Explanations
}

type ImplicitDepLoader struct {
	state_                  *State
	disk_interface_         DiskInterface
	deps_log_               *DepsLog
	depfile_parser_options_ *DepfileParserOptions
	explanations_           Explanations
}

func NewImplicitDepLoader(state *State, deps_log *DepsLog, disk_interface DiskInterface,
	depfile_parser_options *DepfileParserOptions, explanations Explanations) *ImplicitDepLoader {
	ret := ImplicitDepLoader{}
	ret.state_ = state
	ret.disk_interface_ = disk_interface
	ret.deps_log_ = deps_log
	ret.depfile_parser_options_ = depfile_parser_options
	ret.explanations_ = explanations
	return &ret
}

// / Load implicit dependencies for \a edge.
// / @return false on error (without filling \a err if info is just missing
//
//	or out of date).
func (this *ImplicitDepLoader) LoadDeps(edge *Edge, err *string) bool {
	deps_type := edge.GetBinding("deps")
	if deps_type != "" {
		return this.LoadDepsFromLog(edge, err)
	}

	depfile := edge.GetUnescapedDepfile()
	if depfile != "" {
		return this.LoadDepFile(edge, depfile, err)
	}

	// No deps to load.
	return true
}

func (this *ImplicitDepLoader) deps_log() *DepsLog {
	return this.deps_log_
}

// / Process loaded implicit dependencies for \a edge and update the graph
// / @return false on error (without filling \a err if info is just missing)
func (this *ImplicitDepLoader) ProcessDepfileDeps(edge *Edge, depfile_ins []*string, err *string) bool {
	// Preallocate space in edge.inputs_ to be filled in below.
	implicit_dep := this.PreallocateSpace(edge, len(depfile_ins))

	// Add all its in-edges.
	for index, i := range depfile_ins { // .begin(); i != depfile_ins.end(); ++i, ++implicit_dep)
		slash_bits := uint64(0)
		CanonicalizePath(i, &slash_bits)
		node := this.state_.GetNode(*i, slash_bits)
		implicit_dep[index] = node
		node.AddOutEdge(edge)
	}

	return true
}

// / Load implicit dependencies for \a edge from a depfile attribute.
// / @return false on error (without filling \a err if info is just missing).
func (this *ImplicitDepLoader) LoadDepFile(edge *Edge, path string, err *string) bool {
	METRIC_RECORD("depfile load")
	// Read depfile content.  Treat a missing depfile as empty.
	content := ""
	switch this.disk_interface_.ReadFile(path, &content, err) {
	case Okay:
		break
	case NotFound:
		*err = ""
	case OtherError:
		*err = "loading '" + path + "': " + *err
		return false
	}
	// On a missing depfile: return false and empty *err.
	first_output := edge.outputs_[0]
	if content == "" {
		this.explanations_.Record(first_output, "depfile '%s' is missing", path)
		return false
	}

	depfile := NewDepfileParser(this.depfile_parser_options_)
	if this.depfile_parser_options_ != nil {
		depfile = NewDepfileParser(NewDepfileParserOptions())
	}
	depfile_err := ""
	if !depfile.Parse(content, &depfile_err) {
		*err = path + ": " + depfile_err
		return false
	}

	if len(depfile.outs_) == 0 {
		*err = path + ": no outputs declared"
		return false
	}

	unused := uint64(0)
	primary_out := depfile.outs_[0]
	CanonicalizePath(primary_out, &unused)

	// Check that this depfile matches the edge's output, if not return false to
	// mark the edge as dirty.
	opath := first_output.path()
	if opath != *primary_out {
		this.explanations_.Record(first_output,
			"expected depfile '%s' to mention '%s', got '%s'",
			path, first_output.path(),
			primary_out)
		return false
	}

	// Ensure that all mentioned outputs are outputs of the edge.
	for _, o := range depfile.outs_ {
		for _, node := range edge.outputs_ {
			if !strings.Contains(node.path(), *o) {
				*err = path + ": depfile mentions '" + *o + "' as an output, but no such output was declared"
				return false
			}
		}
	}

	return this.ProcessDepfileDeps(edge, depfile.ins_, err)
}

// / Load implicit dependencies for \a edge from the DepsLog.
// / @return false on error (without filling \a err if info is just missing).
func (this *ImplicitDepLoader) LoadDepsFromLog(edge *Edge, err *string) bool {
	// NOTE: deps are only supported for single-target edges.
	output := edge.outputs_[0]
	var deps *Deps = nil
	if this.deps_log_ != nil {
		deps = this.deps_log_.GetDeps(output)
	}
	if deps == nil {
		this.explanations_.Record(output, "deps for '%s' are missing",
			output.path())
		return false
	}

	// Deps are invalid if the output is newer than the deps.
	if output.mtime() > deps.mtime {
		this.explanations_.Record(output,
			"stored deps info out of date for '%s' (%"+PRId64+
				" vs %"+PRId64+")",
			output.path(), deps.mtime, output.mtime())
		return false
	}

	nodes := deps.nodes
	node_count := deps.node_count
	edge.inputs_ = append(edge.inputs_, nodes...)
	edge.implicit_deps_ += node_count
	for i := 0; i < node_count; i++ {
		nodes[i].AddOutEdge(edge)
	}
	return true
}

// / Preallocate \a count spaces in the input array on \a edge, returning
// / an iterator pointing at the first new space.
func (this *ImplicitDepLoader) PreallocateSpace(edge *Edge, count int) []*Node {
	edge.inputs_ = append(edge.inputs_, make([]*Node, count)...) // ,  count, 0
	edge.implicit_deps_ += count
	return edge.inputs_[edge.order_only_deps_ : edge.order_only_deps_+count]
}
