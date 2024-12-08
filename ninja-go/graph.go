package main

import (
	"fmt"
	"github.com/ahrtr/gocontainer/queue/priorityqueue"
	"github.com/edwingeng/deque"
	"github.com/fatih/color"
	"log"
	"runtime"
	"slices"
	"strings"
)

func NewNode(path string, slash_bits uint64) *Node {
	ret := Node{}
	ret.path_ = path
	ret.slash_bits_ = slash_bits
	return &ret
}

// / Return false on error.
func (this *Node) Stat(disk_interface DiskInterface, err *string) bool {
	mtime, notExist, err1 := disk_interface.StatNode(this)
	this.mtime_ = mtime // Node in_edge所有文件的Hash, path_也可能存在于远程，in_edge中的文件也可能存在于远程
	if err1 != nil {
		return false
	}
	if !notExist {
		this.exists_ = ExistenceStatusExists
	} else {
		this.exists_ = ExistenceStatusMissing
	}

	return true
}

// / If the file doesn't exist, set the mtime_ from its dependencies
func (this *Node) UpdatePhonyMtime(mtime TimeStamp) {}

// / Return false on error.
func (this *Node) StatIfNecessary(disk_interface DiskInterface, err *string) bool {
	if this.status_known() {
		return true
	}
	return this.Stat(disk_interface, err)
}

// / Mark as not-yet-stat()ed and not dirty.
func (this *Node) ResetState() {
	this.mtime_ = -1
	this.exists_ = ExistenceStatusUnknown
	this.dirty_ = false
}

// / Mark the Node as already-stat()ed and missing.
func (this *Node) MarkMissing() {
	if this.mtime_ == -1 {
		this.mtime_ = 0
	}
	this.exists_ = ExistenceStatusMissing
}

func (this *Node) exists() bool {
	return this.exists_ == ExistenceStatusExists
}

func (this *Node) status_known() bool {
	return this.exists_ != ExistenceStatusUnknown
}

func (this *Node) path() string {
	return this.path_
}

// / Get |path()| but use slash_bits to convert back to original slash styles.
func (this *Node) PathDecanonicalized() string {
	return PathDecanonicalized(this.path_, this.slash_bits_)
}
func PathDecanonicalized(path string, slash_bits uint64) string {
	result := path
	if runtime.GOOS == "windows" {
		mask := uint64(1)
		for i := 0; i < len(result); {
			if result[i] == '/' {
				if slash_bits&mask != 0 {
					result = strings.Replace(result, "/", "\\", 1)
				}
				mask <<= 1
			}
			i++
		}
	}
	return result
}
func (this *Node) slash_bits() uint64 { return this.slash_bits_ }

func (this *Node) mtime() TimeStamp { return this.mtime_ }

func (this *Node) dirty() bool          { return this.dirty_ }
func (this *Node) set_dirty(dirty bool) { this.dirty_ = dirty }
func (this *Node) MarkDirty()           { this.dirty_ = true }

func (this *Node) dyndep_pending() bool            { return this.dyndep_pending_ }
func (this *Node) set_dyndep_pending(pending bool) { this.dyndep_pending_ = pending }

func (this *Node) in_edge() *Edge         { return this.in_edge_ }
func (this *Node) set_in_edge(edge *Edge) { this.in_edge_ = edge }

// / Indicates whether this node was generated from a depfile or dyndep file,
// / instead of being a regular input or output from the Ninja manifest.
func (this *Node) generated_by_dep_loader() bool { return this.generated_by_dep_loader_ }

func (this *Node) set_generated_by_dep_loader(value bool) {
	this.generated_by_dep_loader_ = value
}

func (this *Node) id() int       { return this.id_ }
func (this *Node) set_id(id int) { this.id_ = id }

func (this *Node) out_edges() []*Edge            { return this.out_edges_ }
func (this *Node) validation_out_edges() []*Edge { return this.validation_out_edges_ }
func (this *Node) AddOutEdge(edge *Edge)         { this.out_edges_ = append(this.out_edges_, edge) }
func (this *Node) AddValidationOutEdge(edge *Edge) {
	this.validation_out_edges_ = append(this.validation_out_edges_, edge)
}

func (this *Node) Dump(prefix string) {
	fmt.Printf("%s <%s 0x%p> mtime: %d%s, (:%s), ",
		prefix,
		this.path(),
		this,
		this.mtime(),
		func() string {
			if this.exists() {
				return ""
			}
			return " (:missing)"
		}(),
		func() string {
			if this.dirty() {
				return " dirty"
			}
			return " clean"
		}())
	if this.in_edge() != nil {
		this.in_edge().Dump("in-edge: ")
	} else {
		fmt.Printf("no in-edge\n")
	}
	fmt.Printf(" out edges:\n")
	for _, e := range this.out_edges() {
		e.Dump(" +- ")
	}
	if len(this.validation_out_edges()) != 0 {
		fmt.Printf(" validation out edges:\n")
		for _, e := range this.validation_out_edges() {
			e.Dump(" +- ")
		}
	}
}

func (this *InputsCollector) VisitNode(node *Node) {
	edge := node.in_edge()

	if edge == nil { // A source file.
		return
	}
	// Add inputs of the producing edge to the result,
	// except if they are themselves produced by a phony
	// edge.
	for _, input := range edge.inputs_ {
		_, ok := this.visited_nodes_[input]
		if ok {
			continue
		}

		this.VisitNode(input)

		input_edge := input.in_edge()
		if !(input_edge != nil && input_edge.is_phony()) {
			this.inputs_ = append(this.inputs_, input)
		}
	}
}

// / Retrieve list of visited input nodes. A dependency always appears
// / before its dependents in the result, but final order depends on the
// / order of the VisitNode() calls performed before this.
func (this *InputsCollector) inputs() []*Node { return this.inputs_ }

// / Same as inputs(), but returns the list of visited nodes as a list of
// / strings, with optional shell escaping.
func (this *InputsCollector) GetInputsAsStrings(shell_escape bool) []string {
	result := make([]string, len(this.inputs_))
	for _, input := range this.inputs_ {
		unescaped := input.PathDecanonicalized()
		if shell_escape {
			path := ""
			GetWin32EscapedString(unescaped, &path)
			result = append(result, path)
		} else {
			result = append(result, unescaped)
		}
	}
	return result
}

// / Reset collector state.
func (this *InputsCollector) Reset() {
	this.inputs_ = []*Node{}
	this.visited_nodes_ = map[*Node]bool{}
}

func NewEdge() *Edge {
	ret := Edge{}
	return &ret
}

// / Return true if all inputs' in-edges are ready.
func (this *Edge) AllInputsReady() bool {
	for _, i := range this.inputs_ {
		if i.in_edge() != nil && !i.in_edge().outputs_ready() {
			return false
		}
	}
	return true
}

// / Expand all variables in a command and return it as a string.
// / If incl_rsp_file is enabled, the string will also contain the
// / full contents of a response file (if applicable)
func (this *Edge) EvaluateCommand(incl_rsp_file bool) string {
	command := this.GetBinding("command")
	if incl_rsp_file {
		rspfile_content := this.GetBinding("rspfile_content")
		if rspfile_content != "" {
			command += ";rspfile=" + rspfile_content
		}
	}
	return command
}

// / Returns the shell-escaped value of |key|.
func (this *Edge) GetBinding(key string) string {
	env := NewEdgeEnv(this, kShellEscape)
	return env.LookupVariable(key)
}

func (this *Edge) GetBindingBool(key string) bool {
	return this.GetBinding(key) != ""
}

// / Like GetBinding("depfile"), but without shell escaping.
func (this *Edge) GetUnescapedDepfile() string {
	env := NewEdgeEnv(this, kDoNotEscape)
	return env.LookupVariable("depfile")
}

// / Like GetBinding("dyndep"), but without shell escaping.
func (this *Edge) GetUnescapedDyndep() string {
	env := NewEdgeEnv(this, kDoNotEscape)
	return env.LookupVariable("dyndep")
}

// / Like GetBinding("rspfile"), but without shell escaping.
func (this *Edge) GetUnescapedRspfile() string {
	env := NewEdgeEnv(this, kDoNotEscape)
	return env.LookupVariable("rspfile")
}

func (this *Edge) Dump(prefix string) {
	fmt.Printf("%s[ ", prefix)
	for _, i := range this.inputs_ {
		fmt.Printf("%s ", (*i).path())
	}
	fmt.Printf("--%s. ", this.rule_.name())
	for _, i := range this.outputs_ {
		fmt.Printf("%s ", (*i).path())
	}
	if len(this.validations_) != 0 {
		fmt.Printf(" validations ")
		for _, i := range this.validations_ {
			fmt.Printf("%s ", i.path())
		}
	}
	if this.pool_ != nil {
		if this.pool_.name() != "" {
			fmt.Printf("(in pool '%s')", this.pool_.name())
		}
	} else {
		fmt.Printf("(null pool?)")
	}
	fmt.Printf("] 0x%p\n", this)
}

// critical_path_weight is the priority during build scheduling. The
// "critical path" between this edge's inputs and any target node is
// the path which maximises the sum oof weights along that path.
// NOTE: Defaults to -1 as a marker smaller than any valid weight
func (this *Edge) critical_path_weight() int64 { return this.critical_path_weight_ }
func (this *Edge) set_critical_path_weight(critical_path_weight int64) {
	this.critical_path_weight_ = critical_path_weight
}

func (this *Edge) rule() *Rule         { return this.rule_ }
func (this *Edge) pool() *Pool         { return this.pool_ }
func (this *Edge) weight() int         { return 1 }
func (this *Edge) outputs_ready() bool { return this.outputs_ready_ }

func (this *Edge) is_implicit(index int64) bool {
	return index >= int64(len(this.inputs_)-this.order_only_deps_-this.implicit_deps_) && !this.is_order_only(index)
}
func (this *Edge) is_order_only(index int64) bool {
	return index >= int64(len(this.inputs_)-this.order_only_deps_)
}

func (this *Edge) is_implicit_out(index int64) bool {
	return index >= int64(len(this.outputs_)-this.implicit_outs_)
}

var kDefaultPool = NewPool("", 0)
var kConsolePool = NewPool("console", 1)
var kPhonyRule = NewRule("phony")

func (this *Edge) is_phony() bool {
	return this.rule_ == kPhonyRule
}

func (this *Edge) use_console() bool {
	return this.pool() == kConsolePool
}

func (this *Edge) maybe_phonycycle_diagnostic() bool {
	// CMake 2.8.12.x and 3.0.x produced self-referencing phony rules
	// of the form "build a: phony ... a ...".   Restrict our
	// "phonycycle" diagnostic option to the form it used.
	return this.is_phony() && len(this.outputs_) == 1 && this.implicit_outs_ == 0 &&
		this.implicit_deps_ == 0
}

func NewDependencyScan(state *State, build_log *BuildLog, deps_log *DepsLog, disk_interface DiskInterface,
	depfile_parser_options *DepfileParserOptions, explanations Explanations, config *BuildConfig, prefixDir string) *DependencyScan {
	ret := DependencyScan{}
	ret.build_log_ = build_log
	ret.disk_interface_ = disk_interface
	ret.dep_loader_ = NewImplicitDepLoader(state, deps_log, disk_interface, depfile_parser_options, explanations)
	ret.dyndep_loader_ = NewDyndepLoader(state, disk_interface, nil)
	ret.explanations_ = explanations
	ret.Config_ = config
	ret.PrefixDir = prefixDir
	return &ret
}

// / Update the |dirty_| state of the given nodes by transitively inspecting
// / their input edges.
// / Examine inputs, outputs, and command lines to judge whether an edge
// / needs to be re-run, and update outputs_ready_ and each outputs' |dirty_|
// / state accordingly.
// / Appends any validation nodes found to the nodes parameter.
// / Returns false on failure.
func (this *DependencyScan) RecomputeDirty(builder *Builder, initial_node *Node, validation_nodes []*Node, err *string) bool {
	stack := []*Node{}
	new_validation_nodes := []*Node{}
	nodes := deque.NewDeque() //(1, initial_node);
	nodes.PushBack(initial_node)

	// RecomputeNodeDirty might return new validation nodes that need to be
	// checked for dirty state, keep a queue of nodes to visit.
	for nodes.Len() != 0 {
		node := nodes.Front()
		nodes.PopFront()

		stack = []*Node{}
		new_validation_nodes = []*Node{}

		if !this.RecomputeNodeDirty(builder, node.(*Node), &stack, &new_validation_nodes, err) {
			return false
		}
		for _, i := range new_validation_nodes {
			nodes.PushBack(i)
		}

		if len(new_validation_nodes) != 0 {
			if validation_nodes == nil {
				panic(" validations require RecomputeDirty to be called with validation_nodes")
			}
			validation_nodes = append(validation_nodes, new_validation_nodes...)
		}
	}

	return true
}

// / Recompute whether any output of the edge is dirty, if so sets |*dirty|.
// / Returns false on failure.
func (this *DependencyScan) RecomputeOutputsDirty(edge *Edge, inputs []*Node, outputs_dirty *bool, err *string) bool {
	command := edge.EvaluateCommand( /*incl_rsp_file=*/ true)
	for _, o := range edge.outputs_ {
		if this.RecomputeOutputDirty(edge, inputs, command, o) {
			*outputs_dirty = true
			return true
		}
	}
	return true
}

func (this *DependencyScan) build_log() *BuildLog {
	return this.build_log_
}
func (this *DependencyScan) set_build_log(log *BuildLog) {
	this.build_log_ = log
}

func (this *DependencyScan) deps_log() *DepsLog {
	return this.dep_loader_.deps_log()
}

// / Load a dyndep file from the given node's path and update the
// / build graph with the new information.  One overload accepts
// / a caller-owned 'DyndepFile' object in which to store the
// / information loaded from the dyndep file.
func (this *DependencyScan) LoadDyndeps(node *Node, err *string) bool {
	return this.dyndep_loader_.LoadDyndeps(node, err)
}

func (this *DependencyScan) LoadDyndeps1(node *Node, ddf DyndepFile, err *string) bool {
	return this.dyndep_loader_.LoadDyndeps1(node, ddf, err)
}

func (this *DependencyScan) RecomputeNodeDirty(builder *Builder, node *Node,
	stack *[]*Node, validation_nodes *[]*Node, err *string) bool {
	edge := node.in_edge()
	if edge == nil {
		// If we already visited this leaf node then we are done.
		if node.status_known() {
			return true
		}
		// This node has no in-edge; it is dirty if it is missing.
		if !node.StatIfNecessary(this.disk_interface_, err) {
			return false
		}
		if !node.exists() {
			this.explanations_.Record(node, "%s has no in-edge and is missing",
				node.path())
		}
		node.set_dirty(!node.exists())
		return true
	}

	// If we already finished this edge then we are done.
	if edge.mark_ == VisitDone {
		return true
	}

	// If we encountered this edge earlier in the call stack we have a cycle.
	if !this.VerifyDAG(node, stack, err) {
		return false
	}

	// Mark the edge temporarily while in the call stack.
	edge.mark_ = VisitInStack
	*stack = append(*stack, node)

	dirty := false
	edge.outputs_ready_ = true
	edge.deps_missing_ = false

	if !edge.deps_loaded_ {
		// This is our first encounter with this edge.
		// If there is a pending dyndep file, visit it now:
		// * If the dyndep file is ready then load it now to get any
		//   additional inputs and outputs for this and other edges.
		//   Once the dyndep file is loaded it will no longer be pending
		//   if any other edges encounter it, but they will already have
		//   been updated.
		// * If the dyndep file is not ready then since is known to be an
		//   input to this edge, the edge will not be considered ready below.
		//   Later during the build the dyndep file will become ready and be
		//   loaded to update this edge before it can possibly be scheduled.
		if edge.dyndep_ != nil && edge.dyndep_.dyndep_pending() {
			if !this.RecomputeNodeDirty(builder, edge.dyndep_, stack, validation_nodes, err) {
				return false
			}

			if edge.dyndep_.in_edge() == nil || edge.dyndep_.in_edge().outputs_ready() {
				// The dyndep file is ready, so load it now.
				if !this.LoadDyndeps(edge.dyndep_, err) {
					return false
				}
			}
		}
	}

	// Load output mtimes so we can compare them to the most recent input below.
	for _, o := range edge.outputs_ {
		if !o.StatIfNecessary(this.disk_interface_, err) {
			return false
		}
	}

	if !edge.deps_loaded_ {
		// This is our first encounter with this edge.  Load discovered deps.
		edge.deps_loaded_ = true
		if !this.dep_loader_.LoadDeps(builder, edge, err) {
			if *err != "" {
				return false
			}
			// Failed to load dependency info: rebuild to regenerate it.
			// LoadDeps() did explanations_.Record() already, no need to do it here.
			edge.deps_missing_ = true
			dirty = true
		}
	}

	// Store any validation nodes from the edge for adding to the initial
	// nodes.  Don't recurse into them, that would trigger the dependency
	// cycle detector if the validation node depends on this node.
	// RecomputeDirty will add the validation nodes to the initial nodes
	// and recurse into them.
	*validation_nodes = append(*validation_nodes, edge.validations_...)

	// Visit all inputs; we're dirty if any of the inputs are dirty.
	var inputs []*Node = edge.inputs_
	for index, i := range edge.inputs_ {
		// Visit this input.
		if !this.RecomputeNodeDirty(builder, i, stack, validation_nodes, err) {
			return false
		}

		// If an input is not ready, neither are our outputs.
		in_edge := i.in_edge()
		if in_edge != nil {
			if !in_edge.outputs_ready_ {
				edge.outputs_ready_ = false
			}
		}

		if !edge.is_order_only(int64(index)) {
			// If a regular input is dirty (or missing), we're dirty.
			// Otherwise consider mtime.
			if i.dirty() {
				this.explanations_.Record(node, "%s is dirty", (*i).path())
				dirty = true
			}
		}
	}

	// We may also be dirty due to output state: missing outputs, out of
	// date outputs, etc.  Visit all outputs and determine whether they're dirty.
	if !dirty {
		if !this.RecomputeOutputsDirty(edge, inputs, &dirty, err) {
			return false
		}
	}

	// Finally, visit each output and update their dirty state if necessary.
	for _, o := range edge.outputs_ {
		if dirty {
			o.MarkDirty()
		}
	}

	// If an edge is dirty, its outputs are normally not ready.  (It's
	// possible to be clean but still not be ready in the presence of
	// order-only inputs.)
	// But phony edges with no inputs have nothing to do, so are always
	// ready.
	if dirty && !(edge.is_phony() && len(edge.inputs_) == 0) {
		edge.outputs_ready_ = false
	}

	// Mark the edge as finished during this walk now that it will no longer
	// be in the call stack.
	edge.mark_ = VisitDone
	if (*stack)[len(*stack)-1] != node {
		panic("stack.back() != node")
	}
	*stack = (*stack)[0 : len(*stack)-1]

	return true
}

func (this *DependencyScan) VerifyDAG(node *Node, stack *[]*Node, err *string) bool {
	edge := node.in_edge()
	if edge == nil {
		*err = "assertion failed: edge is NULL"
		return false
	}

	// 如果边没有临时标记，则尚未发现循环
	if edge.mark_ != VisitInStack {
		return true
	}

	// 查找调用栈中该边的起始位置
	for i, n := range *stack {
		if n.in_edge() == edge {
			// 标记循环的开始为边的结束节点
			(*stack)[i] = node
			break
		}
	}

	// 构建错误信息，拒绝循环
	*err = "dependency cycle: "
	for _, n := range *stack {
		*err += n.path_ + " . "
	}
	*err += (*stack)[len(*stack)-1].path_

	if len(*stack) == 1 && edge.maybe_phonycycle_diagnostic() {
		*err += " [-w phonycycle=err]"
	}

	return false
}

// / Recompute whether a given single output should be marked dirty.
// / Returns true if so.
func (this *DependencyScan) RecomputeOutputDirty(edge *Edge, inputs []*Node, command string, output *Node) bool {
	/*if edge.is_phony() {
		// Phony edges don't write any output.  Outputs are only dirty if
		// there are no inputs and we're missing the output.
		if len(edge.inputs_) == 0 && !output.exists() {
			this.explanations_.Record(
				output, "output %s of phony edge with no inputs doesn't exist",
				output.path())
			return true
		}

		// Update the mtime with the newest input. Dependents can thus call mtime()
		// on the fake node and get the latest mtime of the dependencies
		if inputs != nil {
			output.UpdatePhonyMtime(inputs[0].mtime())
		}

		// Phony edges are clean, nothing to do
		return false
	}

	// Dirty if we're missing the output.
	if !output.exists() {
		this.explanations_.Record(output, "output %s doesn't exist",
			output.path())
		return true
	}*/

	var entry *LogEntry = nil

	// If this is a restat rule, we may have cleaned the output in a
	// previous run and stored the command start time in the build log.
	// We don't want to consider a restat rule's outputs as dirty unless
	// an input changed since the last run, so we'll skip checking the
	// output file's actual mtime and simply check the recorded mtime from
	// the log against the most recent input's mtime (see below)
	//used_restat := false
	//if edge.GetBindingBool("restat") && this.build_log() != nil {
	//	entry = this.build_log().LookupByOutput(this.Config_, output.path())
	//	if entry != nil {
	//		used_restat = true
	//	}
	//}

	// Dirty if the output is older than the input.
	//if !used_restat && most_recent_input != nil && output.mtime() != most_recent_input.mtime() {
	//	this.explanations_.Record(output,
	//		"output %s older than most recent input %s (%d vs %d)",
	//		output.path(),
	//		most_recent_input.path(), output.mtime(),
	//		most_recent_input.mtime())
	//	return true
	//}

	if this.build_log() != nil {
		generator := edge.GetBindingBool("generator")
		currentMtime, _, err := NodesHash(inputs, this.PrefixDir)
		currentHash := HashCommand(command)
		color.Blue("command: %s, currentHash: %x, currentMtime: %d", command, currentHash, currentMtime)

		if entry != nil || func() bool {
			entry = this.build_log().LookupByOutput(this.Config_, output.path(), currentHash, currentMtime)
			return entry != nil
		}() {
			if !generator && currentHash != entry.command_hash {
				// May also be dirty due to the command changing since the last build.
				// But if this is a generator rule, the command changing does not make us
				// dirty.
				this.explanations_.Record(output, "command line changed for %s", output.path())
				return true
			}
			if err == nil && entry.mtime != currentMtime {
				//	// May also be dirty due to the mtime in the log being older than the
				//	// mtime of the most recent input.  This can occur even when the mtime
				//	// on disk is newer if a previous run wrote to the output file but
				//	// exited with an error or was interrupted. If this was a restat rule,
				//	// then we only check the recorded mtime against the most recent input
				//	// mtime and ignore the actual output's mtime above.
				//this.explanations_.Record(
				//	output,
				//	"recorded mtime of %s older than most recent input %s (%d vs %d)",
				//	output.path(), most_recent_input.path(),
				//	entry.mtime, most_recent_input.mtime())
				return true
			}
		}
		if entry == nil && !generator {
			this.explanations_.Record(output, "command line not found in log for %s",
				output.path())
			return true
		}
	}

	return false
}

func (this *DependencyScan) RecordExplanation(node *Node, fmt string, args ...interface{}) {}

type EscapeKind int8

const (
	kShellEscape EscapeKind = 0
	kDoNotEscape EscapeKind = 1
)

type EdgeEnv struct {
	Env
	lookups_       []string
	edge_          *Edge
	escape_in_out_ EscapeKind
	recursive_     bool
}

func NewEdgeEnv(edge *Edge, escape EscapeKind) *EdgeEnv {
	ret := EdgeEnv{}
	ret.edge_ = edge
	ret.escape_in_out_ = escape
	ret.recursive_ = false
	return &ret
}
func (this *EdgeEnv) LookupVariable(var1 string) string {
	if var1 == "in" || var1 == "in_newline" {
		explicit_deps_count := len(this.edge_.inputs_) - this.edge_.implicit_deps_ -
			this.edge_.order_only_deps_
		if var1 == "in" {
			return this.MakePathList(this.edge_.inputs_, explicit_deps_count, ' ')
		} else {
			return this.MakePathList(this.edge_.inputs_, explicit_deps_count, '\n')
		}

	} else if var1 == "out" {
		explicit_outs_count := len(this.edge_.outputs_) - this.edge_.implicit_outs_
		return this.MakePathList(this.edge_.outputs_, explicit_outs_count, ' ')
	}

	// Technical note about the lookups_ vector.
	//
	// This is used to detect cycles during recursive variable expansion
	// which can be seen as a graph traversal problem. Consider the following
	// example:
	//
	//    rule something
	//      command = $foo $foo $var1
	//      var1 = $var2
	//      var2 = $var3
	//      var3 = $var1
	//      foo = FOO
	//
	// Each variable definition can be seen as a node in a graph that looks
	// like the following:
	//
	//   command -. foo
	//      |
	//      v
	//    var1 <-----.
	//      |        |
	//      v        |
	//    var2 --. var3
	//
	// The lookups_ vector is used as a stack of visited nodes/variables
	// during recursive expansion. Entering a node adds an item to the
	// stack, leaving the node removes it.
	//
	// The recursive_ flag is used as a small performance optimization
	// to never record the starting node in the stack when beginning a new
	// expansion, since in most cases, expansions are not recursive
	// at all.
	//
	if this.recursive_ {
		if slices.Contains(this.lookups_, var1) {
			cycle := ""

			for _, it := range this.lookups_ {
				cycle += it + " . "
			}
			cycle += var1
			log.Fatal("cycle in rule variables: " + cycle)
		}
	}

	// See notes on BindingEnv::LookupWithFallback.
	eval := this.edge_.rule_.GetBinding(var1)
	record_varname := this.recursive_ && eval != nil
	if record_varname {
		this.lookups_ = append(this.lookups_, var1)
	}

	// In practice, variables defined on rules never use another rule variable.
	// For performance, only start checking for cycles after the first lookup.
	this.recursive_ = true
	result := this.edge_.env_.LookupWithFallback(var1, eval, this)
	if record_varname {
		this.lookups_ = this.lookups_[0 : len(this.lookups_)-1]
	}
	return result
}

// / Given a span of Nodes, construct a list of paths suitable for a command
// / line.
func (this *EdgeEnv) MakePathList(spans []*Node, size int, sep uint8) string {
	result := ""
	for i := 0; i < size; i++ {
		if result != "" {
			result += string(sep)
		}
		path := spans[i].PathDecanonicalized()
		if this.escape_in_out_ == kShellEscape {
			GetWin32EscapedString(path, &result)
		} else {
			result += path
		}
	}
	return result
}

type EdgePriorityQueue priorityqueue.Interface
