package main

import (
	"github.com/edwingeng/deque"
	"log"
	"math"
	"os"
)

type EdgeResult int8

const (
	kEdgeFailed    EdgeResult = 0
	kEdgeSucceeded EdgeResult = 1
)

// / Enumerate possible steps we want for an edge.
type Want int8

const (
	/// We do not want to build the edge, but we might want to build one of
	/// its dependents.
	kWantNothing Want = 0
	/// We want to build the edge, but have not yet scheduled it.
	kWantToStart Want = 1
	/// We want to build the edge, have scheduled it, and are waiting
	/// for it to complete.
	kWantToFinish Want = 2
)

type Verbosity int8

const (
	QUIET            Verbosity = 0
	NO_STATUS_UPDATE           = 1
	NORMAL                     = 2
	VERBOSE                    = 3
)

type BuildConfig struct {
	Verbosity       Verbosity
	DryRun          bool
	Parallelism     int
	FailuresAllowed int
	/// The maximum load average we must not exceed. A negative value
	/// means that we do not have any limit.
	MaxLoadAverage       float64
	DepfileParserOptions *DepfileParserOptions
	/// RBE Service
	RbeService string
	// RBE Instance
	RbeInstance string
}

func NewBuildConfig() *BuildConfig {
	ret := BuildConfig{Verbosity: NORMAL, DryRun: false,
		Parallelism: 1, FailuresAllowed: 1,
		MaxLoadAverage: -0.0, RbeInstance: "main",
	}
	return &ret
}

// / Map of running edge to time the edge started running.
type RunningEdgeMap map[*Edge]int

type Builder struct {
	state_          *State
	config_         *BuildConfig
	plan_           *Plan
	command_runner_ CommandRunner
	status_         Status

	running_edges_ RunningEdgeMap

	/// Time the build started.
	start_time_millis_ int64

	lock_file_path_ string
	disk_interface_ DiskInterface

	// Only create an Explanations class if '-d explain' is used.
	explanations_ Explanations

	scan_ *DependencyScan
}
type TopoSort struct {
	// Implementation note:
	//
	// This is the regular depth-first-search algorithm described
	// at https://en.wikipedia.org/wiki/Topological_sorting, except
	// that:
	//
	// - Edges are appended to the end of the list, for performance
	//   reasons. Hence the order used in result().
	//
	// - Since the graph cannot have any cycles, temporary marks
	//   are not necessary, and a simple set is used to record
	//   which edges have already been visited.
	//

	visited_set_  map[*Edge]bool //std::unordered_set<
	sorted_edges_ []*Edge
}

func (this *TopoSort) VisitTarget(target *Node) {
	producer := target.in_edge()
	if producer != nil {
		this.Visit(producer)
	}
}

func (this *TopoSort) result() []*Edge { return this.sorted_edges_ }

func (this *TopoSort) Visit(edge *Edge) {
	_, exists := this.visited_set_[edge]
	if exists {
		return
	}
	this.visited_set_[edge] = true
	for _, input := range edge.inputs_ {
		producer := input.in_edge()
		if producer != nil {
			this.Visit(producer)
		}
	}
	this.sorted_edges_ = append(this.sorted_edges_, edge)
}

// / The result of waiting for a command.
type Result struct {
	edge   *Edge
	status ExitStatus
	output string
}

func NewResult() *Result {
	ret := Result{}
	ret.edge = nil
	return &ret
}
func (this *Result) success() bool { return this.status == ExitSuccess }

type CommandRunner interface {
	//ReleaseCommandRunner()
	StartCommand(edge *Edge) bool
	WaitForCommand(result *Result) bool
	GetActiveEdges() []*Edge
	CanRunMore() int64
	Abort()
}

func CommandRunnerfactory(config *BuildConfig) CommandRunner {
	return NewRealCommandRunner(config)
}

type DryRunCommandRunner struct {
	CommandRunner
	finished_ deque.Deque // <Edge*>
}

func (this *DryRunCommandRunner) StartCommand(edge *Edge) bool {
	this.finished_.PushBack(edge)
	return true
}

func (this *DryRunCommandRunner) WaitForCommand(result *Result) bool {
	if this.finished_.Empty() {
		return false
	}

	result.status = ExitSuccess
	result.edge = this.finished_.Front().(*Edge)
	this.finished_.PopFront()
	return true
}

func (d *DryRunCommandRunner) GetActiveEdges() []*Edge {
	return []*Edge{}
}

func (d *DryRunCommandRunner) Abort() {}

func NewBuilder(state *State, config *BuildConfig, build_log *BuildLog,
	deps_log *DepsLog, disk_interface DiskInterface, status Status,
	start_time_millis int64, prefixDir string) *Builder {
	ret := Builder{}
	ret.state_ = state
	ret.config_ = config
	ret.plan_ = NewPlan(&ret)
	ret.status_ = status
	ret.start_time_millis_ = start_time_millis
	ret.disk_interface_ = disk_interface
	ret.running_edges_ = make(map[*Edge]int)
	if g_explaining {
		ret.explanations_ = NewOptionalExplanations()
	} else {
		ret.explanations_ = NewOptionalExplanations()
	}
	ret.scan_ = NewDependencyScan(state, build_log, deps_log, disk_interface,
		ret.config_.DepfileParserOptions, ret.explanations_, config, prefixDir)
	ret.lock_file_path_ = ".ninja_lock"
	build_dir := ret.state_.bindings_.LookupVariable("builddir")
	if build_dir != "" {
		ret.lock_file_path_ = build_dir + "/" + ret.lock_file_path_
	}
	ret.status_.SetExplanations(ret.explanations_)
	return &ret
}

func (this *Builder) RealeaseBuilder() {
	this.Cleanup()
	this.status_.SetExplanations(nil)
}

// / Clean up after interrupted commands by deleting output files.
func (this *Builder) Cleanup() {
	if this.command_runner_ != nil {
		active_edges := this.command_runner_.GetActiveEdges()
		this.command_runner_.Abort()

		for _, e := range active_edges {
			depfile := e.GetUnescapedDepfile()
			for _, o := range e.outputs_ {
				// Only delete this output if it was actually modified.  This is
				// important for things like the generator where we don't want to
				// delete the manifest file if we can avoid it.  But if the rule
				// uses a depfile, always delete.  (Consider the case where we
				// need to rebuild an output because of a modified header file
				// mentioned in a depfile, and the command touches its depfile
				// but is interrupted before it touches its output file.)
				new_mtime, _, err1 := this.disk_interface_.StatNode(o)
				if err1 != nil { // Log and ignore Stat() errors.
					this.status_.Error("%s", err1.Error())
				}
				if depfile != "" || o.mtime() != new_mtime {
					this.disk_interface_.RemoveFile(o.path())
				}
			}
			if depfile != "" {
				this.disk_interface_.RemoveFile(depfile)
			}
		}
	}

	if _, err1 := os.Stat(this.lock_file_path_); err1 == nil {
		this.disk_interface_.RemoveFile(this.lock_file_path_)
	}
}

func (this *Builder) AddTarget(name string, err *string) *Node {
	node := this.state_.LookupNode(name)
	if node == nil {
		*err = "unknown target: '" + name + "'"
		return nil
	}
	if !this.AddTarget2(node, err) {
		return nil
	}
	return node
}

// / Add a target to the build, scanning dependencies.
// / @return false on error.
func (this *Builder) AddTarget2(target *Node, err *string) bool {
	validation_nodes := []*Node{}
	if !this.scan_.RecomputeDirty(target, validation_nodes, err) {
		return false
	}

	in_edge := target.in_edge()
	if in_edge == nil || !in_edge.outputs_ready() {
		if !this.plan_.AddTarget(target, err) {
			return false
		}
	}

	// Also add any validation nodes found during RecomputeDirty as top level
	// targets.
	for _, n := range validation_nodes {
		validation_in_edge := n.in_edge()
		if validation_in_edge != nil {
			if !validation_in_edge.outputs_ready() && !this.plan_.AddTarget(n, err) {
				return false
			}
		}
	}

	return true
}

// / Returns true if the build targets are already up to date.
func (this *Builder) AlreadyUpToDate() bool {
	return !this.plan_.more_to_do()
}

// / Run the build.  Returns false on error.
// / It is an error to call this function when AlreadyUpToDate() is true.
func (this *Builder) Build(err *string) bool {
	if this.AlreadyUpToDate() {
		panic("AlreadyUpToDate() ")
	}
	this.plan_.PrepareQueue()

	pending_commands := 0
	failures_allowed := this.config_.FailuresAllowed

	// Set up the command runner if we haven't done so already.
	if this.command_runner_ == nil {
		if this.config_.DryRun {
			this.command_runner_ = &DryRunCommandRunner{}
		} else {
			this.command_runner_ = CommandRunnerfactory(this.config_)
		}
	}

	// We are about to start the build process.
	this.status_.BuildStarted()

	// This main loop runs the entire build process.
	// It is structured like this:
	// First, we attempt to start as many commands as allowed by the
	// command runner.
	// Second, we attempt to wait for / reap the next finished command.
	for this.plan_.more_to_do() {
		// See if we can start any more commands.
		if failures_allowed != 0 {
			capacity := this.command_runner_.CanRunMore()
			for capacity > 0 {
				edge := this.plan_.FindWork()
				if edge == nil {
					break
				}

				if edge.GetBindingBool("generator") {
					this.scan_.build_log().Close()
				}

				if !this.StartEdge(edge, err) {
					this.Cleanup()
					this.status_.BuildFinished()
					return false
				}

				if edge.is_phony() {
					if !this.plan_.EdgeFinished(edge, kEdgeSucceeded, err) {
						this.Cleanup()
						this.status_.BuildFinished()
						return false
					}
				} else {
					pending_commands++

					capacity--

					// Re-evaluate capacity.
					current_capacity := this.command_runner_.CanRunMore()
					if current_capacity < capacity {
						capacity = current_capacity
					}
				}
			}

			// We are finished with all work items and have no pending
			// commands. Therefore, break out of the main loop.
			if pending_commands == 0 && !this.plan_.more_to_do() {
				break
			}
		}

		// See if we can reap any finished commands.
		if pending_commands != 0 {
			result := Result{}
			if !this.command_runner_.WaitForCommand(&result) ||
				result.status == ExitInterrupted {
				this.Cleanup()
				this.status_.BuildFinished()
				*err = "interrupted by user"
				return false
			}

			pending_commands--
			if !this.FinishCommand(&result, err) {
				this.Cleanup()
				this.status_.BuildFinished()
				return false
			}

			if !result.success() {
				if failures_allowed != 0 {
					failures_allowed--
				}
			}

			// We made some progress; start the main loop over.
			continue
		}

		// If we get here, we cannot make any more progress.
		this.status_.BuildFinished()
		if failures_allowed == 0 {
			if this.config_.FailuresAllowed > 1 {
				*err = "subcommands failed"
			} else {
				*err = "subcommand failed"
			}
		} else if failures_allowed < this.config_.FailuresAllowed {
			*err = "cannot make progress due to previous errors"
		} else {
			*err = "stuck [this is a bug]"
		}

		return false
	}

	this.status_.BuildFinished()
	return true
}

func (this *Builder) StartEdge(edge *Edge, err *string) bool {
	METRIC_RECORD("StartEdge")
	if edge.is_phony() {
		return true
	}

	start_time_millis := GetTimeMillis() - this.start_time_millis_
	this.running_edges_[edge] = int(start_time_millis)

	this.status_.BuildEdgeStarted(edge, start_time_millis)

	//var build_start TimeStamp = -1
	//if this.config_.DryRun {
	//	build_start = 0
	//}
	// Create directories necessary for outputs and remember the current
	// filesystem mtime to record later
	// XXX: this will block; do we care?
	for _, o := range edge.outputs_ {
		ok := this.disk_interface_.MakeDirs(o.path(), err)
		if !ok {
			return false
		}
		//if build_start == -1 {
		//	this.disk_interface_.WriteFile(this.lock_file_path_, "")
		//	var err1 error
		//	build_start, _, err1 = this.disk_interface_.Stat(this.lock_file_path_)
		//	if err1 != nil {
		//		build_start = 0
		//	}
		//}
	}

	// edge.command_start_time_ = build_start

	// Create depfile directory if needed.
	// XXX: this may also block; do we care?
	depfile := edge.GetUnescapedDepfile()
	ok := this.disk_interface_.MakeDirs(depfile, err)
	if depfile != "" && !ok {
		return false
	}

	// Create response file, if needed
	// XXX: this may also block; do we care?
	rspfile := edge.GetUnescapedRspfile()
	if rspfile != "" {
		content := edge.GetBinding("rspfile_content")
		if !this.disk_interface_.WriteFile(rspfile, content) {
			return false
		}
	}

	// start command computing and run it
	if !this.command_runner_.StartCommand(edge) {
		*err = "command '" + edge.EvaluateCommand(false) + "' failed."
		return false
	}

	return true
}

// / Update status ninja logs following a command termination.
// / @return false if the build can not proceed further due to a fatal error.
func (this *Builder) FinishCommand(result *Result, err *string) bool {
	METRIC_RECORD("FinishCommand")

	edge := result.edge

	// First try to extract dependencies from the result, if any.
	// This must happen first as it filters the command output (we want
	// to filter /showIncludes output, even on compile failure) and
	// extraction itself can fail, which makes the command fail from a
	// build perspective.
	deps_nodes := []*Node{}
	deps_type := edge.GetBinding("deps")
	deps_prefix := edge.GetBinding("msvc_deps_prefix")
	if deps_type != "" {
		extract_err := ""
		if !this.ExtractDeps(result, deps_type, deps_prefix, deps_nodes,
			&extract_err) &&
			result.success() {
			if result.output != "" {
				result.output += "\n"
			}
			result.output += extract_err
			result.status = ExitFailure
		}
	}

	start_time_millis := int64(0)
	end_time_millis := int64(0)
	it_second, _ := this.running_edges_[edge]
	start_time_millis = int64(it_second)
	end_time_millis = GetTimeMillis() - this.start_time_millis_
	delete(this.running_edges_, edge)

	this.status_.BuildEdgeFinished(edge, start_time_millis, end_time_millis,
		result.success(), result.output)

	// The rest of this function only applies to successful commands.
	if !result.success() {
		return this.plan_.EdgeFinished(edge, kEdgeFailed, err)
	}

	// Restat the edge outputs
	var record_mtime TimeStamp = 0
	if !this.config_.DryRun {
		restat := edge.GetBindingBool("restat")
		generator := edge.GetBindingBool("generator")
		// node_cleaned := false
		// record_mtime = edge.command_start_time_

		// restat and generator rules must restat the outputs after the build
		// has finished. if record_mtime == 0, then there was an error while
		// attempting to touch/stat the temp file when the edge started and
		// we should fall back to recording the outputs' current mtime in the
		// log.
		if record_mtime == 0 || restat || generator {
			for _, o := range edge.outputs_ {
				new_mtime, _, err1 := this.disk_interface_.StatNode(o)
				if err1 != nil {
					return false
				}
				if new_mtime != record_mtime {
					record_mtime = new_mtime
				}
				if (*o).mtime() == new_mtime && restat {
					// The rule command did not change the output.  Propagate the clean
					// state through the build graph.
					// Note that this also applies to nonexistent outputs (mtime == 0).
					if !this.plan_.CleanNode(this.scan_, o, err) {
						return false
					}
					// node_cleaned = true
				}
			}
		}
		//if node_cleaned {
		//	record_mtime = edge.command_start_time_
		//}
	}

	if !this.plan_.EdgeFinished(edge, kEdgeSucceeded, err) {
		return false
	}

	// Delete any left over response file.
	rspfile := edge.GetUnescapedRspfile()
	if rspfile != "" && !g_keep_rsp {
		this.disk_interface_.RemoveFile(rspfile)
	}

	if this.scan_.build_log() != nil {
		if !this.scan_.build_log().RecordCommand(edge, int(start_time_millis),
			int(end_time_millis), record_mtime) {
			*err = string("Error writing to build log: ") + *err
			return false
		}
	}

	if deps_type != "" && !this.config_.DryRun {
		if len(edge.outputs_) == 0 {
			panic("should have been rejected by parser")
		}
		for _, o := range edge.outputs_ {
			deps_mtime, _, err1 := this.disk_interface_.StatNode(o)
			if err1 != nil {
				return false
			}
			if !this.scan_.deps_log().RecordDeps(o, deps_mtime, deps_nodes, err) {
				*err = "Error writing to deps log: " + *err
				return false
			}
		}
	}
	return true
}

// / Used for tests.
func (this *Builder) SetBuildLog(log *BuildLog) {
	this.scan_.set_build_log(log)
}

func (this *Builder) ExtractDeps(result *Result, deps_type string, deps_prefix string, deps_nodes []*Node, err *string) bool {
	if deps_type == "msvc" {
		parser := NewCLParser()
		output := ""
		if !parser.Parse(&result.output, deps_prefix, &output, err) {
			return false
		}
		result.output = output
		for i, _ := range parser.includes_ {
			// ~0 is assuming that with MSVC-parsed headers, it's ok to always make
			// all backslashes (as some of the slashes will certainly be backslashes
			// anyway). This could be fixed if necessary with some additional
			// complexity in IncludesNormalize::Relativize.
			deps_nodes = append(deps_nodes, this.state_.GetNode(i, math.MaxUint64))
		}
	} else if deps_type == "gcc" {
		depfile := result.edge.GetUnescapedDepfile()
		if depfile == "" {
			*err = string("edge with deps=gcc but no depfile makes no sense")
			return false
		}

		// Read depfile content.  Treat a missing depfile as empty.
		content := ""
		switch this.disk_interface_.ReadFile(depfile, &content, err) {
		case NotFound:
			*err = ""
		case OtherError:
			return false
		case Okay:
		}
		if content == "" {
			return true
		}

		deps := NewDepfileParser(this.config_.DepfileParserOptions)
		if !deps.Parse([]byte(content), err) {
			return false
		}

		// XXX check depfile matches expected output.
		//deps_nodes.reserve(deps.ins_.size());
		for _, i := range deps.ins_ {
			var slash_bits uint64 = 0
			CanonicalizePath(&i, &slash_bits)
			deps_nodes = append(deps_nodes, this.state_.GetNode(i, slash_bits))
		}

		if !g_keep_depfile {
			if this.disk_interface_.RemoveFile(depfile) < 0 {
				*err = string("deleting depfile: ") + *err + string("\n")
				return false
			}
		}
	} else {
		log.Fatalf("unknown deps type '%s'", deps_type)
	}

	return true
}

// / Load the dyndep information provided by the given node.
func (this *Builder) LoadDyndeps(node *Node, err *string) bool {
	// Load the dyndep information provided by this node.
	ddf := DyndepFile{}
	if !this.scan_.LoadDyndeps1(node, ddf, err) {
		return false
	}

	// Update the build plan to account for dyndep modifications to the graph.
	if !this.plan_.DyndepsLoaded(this.scan_, node, ddf, err) {
		return false
	}

	return true
}
