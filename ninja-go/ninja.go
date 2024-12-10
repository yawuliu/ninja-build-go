package main

import (
	"fmt"
	"git.sr.ht/~sircmpwn/getopt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// / Command-line options.
type Options struct {
	/// Build file to load.
	InputFile string

	/// Directory to change into before running.
	WorkingDir string

	/// Tool to run rather than building.
	Tool *Tool

	/// Whether phony cycles should warn or print an error.
	PhonyCycleShouldErr bool
}

type When int8

const (
	/// Run after parsing the command-line flags and potentially changing
	/// the current working directory (as early as possible).
	RUN_AFTER_FLAGS When = 0

	/// Run after loading build.ninja.
	RUN_AFTER_LOAD = 1

	/// Run after loading the build/deps logs.
	RUN_AFTER_LOGS = 2
)

// / The type of functions that are the entry points to tools (subcommands).
type ToolFunc func(*Options, *[]string) int

// / Subtools, accessible via "-t foo".
type Tool struct {
	/// Short name of the tool.
	Name string

	/// Description (shown in "-t list").
	Desc string

	/// When to run the tool.
	When When

	/// Implementation of the tool.
	Func1 ToolFunc
}

type NinjaMain struct {
	BuildLogUser

	/// Command line used to run Ninja.
	NinjaCommand string

	/// Build configuration set from flags (e.g. parallelism).
	Config_ *BuildConfig

	/// Loaded state (rules, nodes).
	State_ *State

	/// Functions for accessing the disk.
	DiskInterface *RealDiskInterface

	/// The build directory, used for storing the build log etc.
	BuildDir string

	BuildLog *BuildLog
	DepsLog  DepsLog

	PrefixDir string

	StartTimeMillis int64
}

func NewNinjaMain(ninja_command, prefixDir string, config *BuildConfig) *NinjaMain {
	ret := NinjaMain{}
	ret.NinjaCommand = ninja_command
	ret.Config_ = config
	ret.StartTimeMillis = GetTimeMillis()
	ret.State_ = NewState()
	ret.BuildLog = NewBuildLog(config, prefixDir)
	ret.PrefixDir = prefixDir
	ret.DiskInterface = NewRealDiskInterface(prefixDir)
	return &ret
}

func (this *NinjaMain) Release() {
	this.BuildLog.ReleaseBuildLog()
	this.DiskInterface.ReleaseRealDiskInterface()
}

func (this *NinjaMain) EnsureBuildDirExists() bool {
	this.BuildDir = this.State_.bindings_.LookupVariable("builddir")
	if this.BuildDir != "" && !this.Config_.DryRun {
		err := ""
		if succ := this.DiskInterface.MakeDirs(this.BuildDir+"/.", &err); !succ {
			Error("creating build directory %s: %s", this.BuildDir, err)
			return false
		}
	}
	return true
}
func (this *NinjaMain) CollectTargetsFromArgs(args *[]string, targets *[]*Node, err *string) bool {
	if len(*args) == 0 {
		*targets = this.State_.DefaultNodes(err)
		return *err == ""
	}

	for i := 0; i < len(*args); i++ {
		node := this.CollectTarget((*args)[i], err)
		if node == nil {
			return false
		}
		*targets = append(*targets, node)
	}
	return true
}

func (this *NinjaMain) RunBuild(args *[]string, status Status) int {
	err := ""
	targets := []*Node{}
	if !this.CollectTargetsFromArgs(args, &targets, &err) {
		status.Error("%s", err)
		return 1
	}

	this.DiskInterface.AllowStatCache(g_experimental_statcache)

	builder := NewBuilder(this.State_, this.Config_, this.BuildLog, &this.DepsLog,
		this.DiskInterface, status, this.StartTimeMillis, this.PrefixDir)
	defer builder.RealeaseBuilder()
	for i := 0; i < len(targets); i++ {
		if !builder.AddTarget2(targets[i], &err) {
			if err != "" {
				status.Error("%s", err)
				return 1
			} else {
				// Added a target that is already up-to-date; not really
				// an error.
			}
		}
	}

	// Make sure restat rules do not see stale timestamps.
	this.DiskInterface.AllowStatCache(false)

	if builder.AlreadyUpToDate() {
		if this.Config_.Verbosity != NO_STATUS_UPDATE {
			status.Info("no work to do.")
		}
		return 0
	}

	if !builder.Build(&err) {
		status.Info("build stopped: %s.", err)
		if strings.Contains(err, "interrupted by user") {
			return 130
		}
		return 1
	}

	return 0
}

func (this *NinjaMain) OpenBuildLog(recompact_only bool) bool {
	log_path := ".ninja_log"
	if this.BuildDir != "" {
		log_path = this.BuildDir + "/" + log_path
	}

	err := ""
	status := this.BuildLog.Load(log_path, &err)
	if status == LOAD_ERROR {
		Error("loading build log %s: %s", log_path, err)
		return false
	}
	if err != "" {
		// Hack: Load() can return a warning via err by returning LOAD_SUCCESS.
		Warning("%s", err)
		err = ""
	}

	if recompact_only {
		if status == LOAD_NOT_FOUND {
			return true
		}
		success := this.BuildLog.Recompact(log_path, this, &err)
		if !success {
			Error("failed recompaction: %s", err)
		}
		return success
	}

	if !this.Config_.DryRun {
		if !this.BuildLog.OpenForWrite(log_path, this, &err) {
			Error("opening build log: %s", err)
			return false
		}
	}

	return true
}

// / Open the deps log: load it, then open for writing.
// / @return false on error.
func (this *NinjaMain) OpenDepsLog(recompact_only bool) bool {
	path := ".ninja_deps"
	if this.BuildDir != "" {
		path = this.BuildDir + "/" + path
	}

	err := ""
	status := this.DepsLog.Load(path, this.State_, &err)
	if status == LOAD_ERROR {
		Error("loading deps log %s: %s", path, err)
		return false
	}
	if err != "" {
		// Hack: Load() can return a warning via err by returning LOAD_SUCCESS.
		Warning("%s", err)
		err = ""
	}

	if recompact_only {
		if status == LOAD_NOT_FOUND {
			return true
		}
		success := this.DepsLog.Recompact(path, &err)
		if !success {
			Error("failed recompaction: %s", err)
		}
		return success
	}

	return true
}

// / Rebuild the build manifest, if necessary.
// / Returns true if the manifest was rebuilt.
func (this *NinjaMain) RebuildManifest(input_file string, err *string, status Status) bool {
	path := input_file
	if path == "" {
		*err = "empty path"
		return false
	}
	var slash_bits uint64 = 0 // Unused because this path is only used for lookup.
	CanonicalizePath(&path, &slash_bits)
	node := this.State_.LookupNode(path)
	if node == nil {
		return false
	}

	builder := NewBuilder(this.State_, this.Config_, this.BuildLog, &this.DepsLog,
		this.DiskInterface, status, this.StartTimeMillis, this.PrefixDir)
	defer builder.RealeaseBuilder()
	if !builder.AddTarget2(node, err) {
		return false
	}
	if builder.AlreadyUpToDate() {
		return false // Not an error, but we didn't rebuild.
	}
	if !builder.Build(err) {
		return false
	}

	// The manifest was only rebuilt if it is now dirty (it may have been cleaned
	// by a restat).
	if !node.dirty() {
		// Reset the state to prevent problems like
		// https://github.com/ninja-build/ninja/issues/874
		this.State_.Reset()
		return false
	}

	return true
}

func (this *NinjaMain) ParsePreviousElapsedTimes() {
	for _, edge := range this.State_.edges_ {
		//command := edge.EvaluateCommand( /*incl_rsp_file=*/ true)
		//currentMtime, _, _ := NodesHash(edge.inputs_, this.PrefixDir)
		for _, out := range edge.outputs_ {
			//currentHash := HashCommand(command)
			log_entry := this.BuildLog.LookupByOutput(this.Config_, out.path(), 0, 0)
			if log_entry == nil {
				continue // Maybe we'll have log entry for next output of this edge?
			}
			edge.prev_elapsed_time_millis = int64(log_entry.end_time - log_entry.start_time)
			break // Onto next edge.
		}
	}
}

func (this *NinjaMain) DumpMetrics() {
	GMetrics.Report()

	fmt.Printf("\n")
	count := int(len(this.State_.paths_))
	buckets := int(8)
	fmt.Printf("path.node hash load %.2f (%d entries / %d buckets)\n", float64(count)/float64(buckets), count, buckets)
}

// / Set a warning flag.  Returns false if Ninja should exit instead of
// / continuing.
func WarningEnable(name string, options *Options) bool {
	if name == "list" {
		fmt.Printf("warning flags:\n  phonycycle={err,warn}  phony build statement references itself\n")
		return false
	} else if name == "phonycycle=err" {
		options.PhonyCycleShouldErr = true
		return true
	} else if name == "phonycycle=warn" {
		options.PhonyCycleShouldErr = false
		return true
	} else if name == "depfilemulti=err" ||
		name == "depfilemulti=warn" {
		Warning("deprecated warning 'depfilemulti'")
		return true
	} else {
		suggestion := SpellcheckString(name, "phonycycle=err", "phonycycle=warn", "\000")
		if suggestion != "" {
			Error("unknown warning flag '%s', did you mean '%s'?", name, suggestion)
		} else {
			Error("unknown warning flag '%s'", name)
		}
		return false
	}
}

func (this *NinjaMain) CollectTarget(cpath string, err *string) *Node {
	path := cpath
	if path == "" {
		*err = "empty path"
		return nil
	}
	slash_bits := uint64(0)
	CanonicalizePath(&path, &slash_bits)

	// Special syntax: "foo.cc^" means "the first output of foo.cc".
	first_dependent := false
	if path != "" && path[len(path)-1] == '^' {
		path = path[0 : len(path)-1]
		first_dependent = true
	}

	node := this.State_.LookupNode(path)
	if node != nil {
		if first_dependent {
			if len(node.out_edges()) == 0 {
				rev_deps := this.DepsLog.GetFirstReverseDepsNode(node)
				if rev_deps == nil {
					*err = "'" + path + "' has no out edge"
					return nil
				}
				node = rev_deps
			} else {
				edge := node.out_edges()[0]
				if len(edge.outputs_) == 0 {
					edge.Dump("")
					log.Fatalln("edge has no outputs")
				}
				node = edge.outputs_[0]
			}
		}
		return node
	} else {
		*err = "unknown target '" + PathDecanonicalized(path, slash_bits) + "'"
		if path == "clean" {
			*err += ", did you mean 'ninja -t clean'?"
		} else if path == "help" {
			*err += ", did you mean 'ninja -h'?"
		} else {
			suggestion := this.State_.SpellcheckNode(path)
			if suggestion != nil {
				*err += ", did you mean '" + suggestion.path() + "'?"
			}
		}
		return nil
	}
}

// / Choose a default value for the -j (parallelism) flag.
func GuessParallelism() int {
	processors := GetProcessorCount()
	switch processors {
	case 0, 1:
		return 2
	case 2:
		return 3
	default:
		return processors + 2
	}
}

type DeferGuessParallelism struct {
	needGuess bool
	config    *BuildConfig
}

func NewDeferGuessParallelism(config *BuildConfig) *DeferGuessParallelism {
	ret := DeferGuessParallelism{}
	ret.needGuess = true
	ret.config = config
	return &ret
}

func (this *DeferGuessParallelism) Refresh() {
	if this.needGuess {
		this.needGuess = false
		this.config.Parallelism = GuessParallelism()
	}
}
func (this *DeferGuessParallelism) ReleaseDeferGuessParallelism() { this.Refresh() }

const (
	OPT_VERSION = 1
	OPT_QUIET   = 2
)

// / Parse argv for command-line options.
// / Returns an exit code, or -1 if Ninja should continue.
func ReadFlags(args *[]string, options *Options, config *BuildConfig) int {
	deferGuessParallelism := NewDeferGuessParallelism(config)
	defer deferGuessParallelism.ReleaseDeferGuessParallelism()
	//kLongOptions  :=  []option{
	//  { "help", no_argument, nil, 'h' },
	//  { "version", no_argument, nil, OPT_VERSION },
	//  { "verbose", no_argument, nil, 'v' },
	//  { "quiet", no_argument, nil, OPT_QUIET },
	//  { "", 0, nil, 0 },
	//}

	opts, optind, err := getopt.Getopts(*args, "d:f:j:k:l:nt:vw:C:h:r:R")
	if err != nil {
		log.Fatalln(err)
	}
	*args = (*args)[optind:]
	//for options.tool==nil && (opt = getopt_long(os.Args, "d:f:j:k:l:nt:vw:C:h", kLongOptions, nil)) != -1 {
	for _, optV := range opts {
		opt := optV.Option
		optarg := optV.Value
		switch opt {
		case 'd':
			if !DebugEnable(optarg) {
				return 1
			}
		case 'f':
			options.InputFile = optarg
		case 'r':
			config.RbeService = optarg
		case 'R':
			config.RbeInstance = optarg
		case 'j':
			{
				value, err := strconv.Atoi(optV.Value)
				if err != nil || value < 0 {
					log.Fatalln("invalid -j parameter")
				}

				// We want to run N jobs in parallel. For N = 0, INT_MAX
				// is close enough to infinite for most sane builds.
				if value > 0 {
					config.Parallelism = value
				} else {
					config.Parallelism = math.MaxInt
				}

				deferGuessParallelism.needGuess = false
			}
		case 'k':
			{
				value, err := strconv.Atoi(optV.Value)
				if err != nil {
					log.Fatalln("-k parameter not numeric; did you mean -k 0?")
				}
				// We want to go until N jobs fail, which means we should allow
				// N failures and then stop.  For N <= 0, INT_MAX is close enough
				// to infinite for most sane builds.
				if value > 0 {
					config.FailuresAllowed = value
				} else {
					config.FailuresAllowed = math.MaxInt
				}
			}
		case 'l':
			{
				value, err := strconv.ParseFloat(optV.Value, 32)
				if err != nil {
					log.Fatalln("-l parameter not numeric: did you mean -l 0.0?")
				}
				config.MaxLoadAverage = value
			}
		case 'n':
			config.DryRun = true
		case 't':
			options.Tool = ChooseTool(optarg)
			if options.Tool == nil {
				return 0
			}
		case 'v':
			config.Verbosity = VERBOSE
		case OPT_QUIET:
			config.Verbosity = NO_STATUS_UPDATE
		case 'w':
			if !WarningEnable(optarg, options) {
				return 1
			}
		case 'C':
			options.WorkingDir = optarg
		case OPT_VERSION:
			fmt.Printf("%s\n", kNinjaVersion)
		default: // case 'h':
			deferGuessParallelism.Refresh()
			UsageMain(config)
			return 1
		}
	}
	return -1
}

// / Print usage information.
func UsageMain(config *BuildConfig) {
	fmt.Fprintf(os.Stderr,
		"usage: ninja [options] [targets...]\n"+
			"\n"+
			"if targets are unspecified, builds the 'default' target (see manual).\n"+
			"\n"+
			"options:\n"+
			"  --version      print ninja version (\"%s\")\n"+
			"  -v, --verbose  show all command lines while building\n"+
			"  --quiet        don't show progress status, just command output\n"+
			"\n"+
			"  -C DIR   change to DIR before doing anything else\n"+
			"  -f FILE  specify input build file [default=build.ninja]\n"+
			"\n"+
			"  -j N     run N jobs in parallel (0 means infinity) [default=%d on this system]\n"+
			"  -k N     keep going until N jobs fail (0 means infinity) [default=1]\n"+
			"  -l N     do not start new jobs if the load average is greater than N\n"+
			"  -n       dry run (don't run commands but act like they succeeded)\n"+
			"\n"+
			"  -d MODE  enable debugging (use '-d list' to list modes)\n"+
			"  -t TOOL  run a subtool (use '-t list' to list subtools)\n"+
			"    terminates toplevel options; further flags are passed to the tool\n"+
			"  -w FLAG  adjust warnings (use '-w list' to list warnings)\n",
		kNinjaVersion, config.Parallelism)
}

func (this *NinjaMain) ToolBrowse(options *Options, args *[]string) int {
	RunBrowsePython(this.State_, this.NinjaCommand, options.InputFile, args)
	// If we get here, the browse failed.
	return 1
}

func (this *NinjaMain) ToolMSVC(options *Options, args *[]string) int {
	// Reset getopt: push one argument onto the front of argv, reset optind.
	optind = 0
	return MSVCHelperMain(args)
}

func (this *NinjaMain) ToolClean(options *Options, args *[]string) int {
	// The clean tool uses getopt, and expects argv[0] to contain the name of
	// the tool, i.e. "clean".

	generator := false
	clean_rules := false

	opts, optind, err := getopt.Getopts(*args, "hgr")
	if err != nil {
		log.Fatalln(err)
	}
	*args = (*args)[optind:]
	for _, optV := range opts {
		opt := optV.Option
		switch opt {
		case 'g':
			generator = true
		case 'r':
			clean_rules = true
		default: // case 'h':
			fmt.Printf("usage: ninja -t clean [options] [targets]\n" +
				"\n" +
				"options:\n" +
				"  -g     also clean files marked as ninja generator output\n" +
				"  -r     interpret targets as a list of rules to clean instead\n",
			)
			return 1
		}
	}

	if clean_rules && len(*args) == 0 {
		Error("expected a rule to clean")
		return 1
	}

	cleaner := NewCleaner(this.State_, this.Config_, this.DiskInterface)
	if len(*args) >= 1 {
		if clean_rules {
			return cleaner.CleanRules(*args)
		} else {
			return cleaner.CleanTargets(*args)
		}
	} else {
		return cleaner.CleanAll(generator)
	}
}

func (this *NinjaMain) ToolCleanDead(options *Options, args *[]string) int {
	cleaner := NewCleaner(this.State_, this.Config_, this.DiskInterface)
	return cleaner.CleanDead(this.BuildLog.entries())
}

type PrintCommandMode int8

const (
	PCM_Single PrintCommandMode = 0
	PCM_All    PrintCommandMode = 1
)

func PrintCommands(edge *Edge, seen EdgeSet, mode PrintCommandMode) {
	if edge == nil {
		return
	}
	if _, ok := seen[edge]; !ok {
		return
	}

	if mode == PCM_All {
		for _, in := range edge.inputs_ {
			PrintCommands(in.in_edge(), seen, mode)
		}
	}

	if !edge.is_phony() {
		fmt.Print(edge.EvaluateCommand(false))
	}
}

func (this *NinjaMain) ToolCommands(options *Options, args *[]string) int {
	// The commands tool uses getopt, and expects argv[0] to contain the name of
	// the tool, i.e. "commands".
	mode := PCM_All

	opts, optind, err1 := getopt.Getopts(*args, "hs")
	if err1 != nil {
		panic(err1)
	}
	*args = (*args)[optind:]
	for _, optV := range opts {
		opt := optV.Option
		switch opt {
		case 's':
			mode = PCM_Single
		default: //case 'h':
			fmt.Printf("usage: ninja -t commands [options] [targets]\n" +
				"\n" +
				"options:\n" +
				"  -s     only print the final command to build [target], not the whole chain\n",
			)
			return 1
		}
	}

	nodes := []*Node{}
	err := ""
	if !this.CollectTargetsFromArgs(args, &nodes, &err) {
		Error("%s", err)
		return 1
	}

	seen := EdgeSet{}
	for _, in := range nodes {
		PrintCommands(in.in_edge(), seen, mode)
	}

	return 0
}

func (this *NinjaMain) ToolInputs(options *Options, args *[]string) int {
	// The inputs tool uses getopt, and expects argv[0] to contain the name of
	// the tool, i.e. "inputs".
	print0 := false
	shell_escape := true
	dependency_order := false

	optind = 1
	//kLongOptions := []option{ { "help", no_argument, nil, 'h' },
	//                               { "no-shell-escape", no_argument, nil, 'E' },
	//                               { "print0", no_argument, nil, '0' },
	//                               { "dependency-order", no_argument, nil,
	//                                 'd' },
	//                               { "", 0, nil, 0 } }
	opts, optind, err1 := getopt.Getopts(*args, "h0Ed")
	if err1 != nil {
		panic(err1)
	}
	*args = (*args)[optind:]
	for _, optV := range opts {
		opt := optV.Option
		switch opt {
		case 'd':
			dependency_order = true
		case 'E':
			shell_escape = false
		case '0':
			print0 = true
		default: //case 'h':
			// clang-format off
			fmt.Print(
				"Usage '-t inputs [options] [targets]\n" +
					"\n" +
					"List all inputs used for a set of targets, sorted in dependency order.\n" +
					"Note that by default, results are shell escaped, and sorted alphabetically,\n" +
					"and never include validation target paths.\n\n" +
					"Options:\n" +
					"  -h, --help          Print this message.\n" +
					"  -0, --print0            Use \\0, instead of \\n as a line terminator.\n" +
					"  -E, --no-shell-escape   Do not shell escape the result.\n" +
					"  -d, --dependency-order  Sort results by dependency order.\n",
			)
			// clang-format on
			return 1
		}
	}
	nodes := []*Node{}
	err := ""
	if !this.CollectTargetsFromArgs(args, &nodes, &err) {
		Error("%s", err)
		return 1
	}

	collector := InputsCollector{}
	for _, node := range nodes {
		collector.VisitNode(node)
	}

	inputs := collector.GetInputsAsStrings(shell_escape)
	if !dependency_order {
		sort.Strings(inputs)
	}

	if print0 {
		for _, input := range inputs {
			fmt.Fprint(os.Stdout, input)
			fmt.Fprint(os.Stdout, 0)
		}
		fmt.Fprint(os.Stdout, "\n")
	} else {
		for _, input := range inputs {
			fmt.Printf(input)
		}
	}
	return 0
}

func (this *NinjaMain) ToolMultiInputs(options *Options, args *[]string) int {
	// The inputs tool uses getopt, and expects argv[0] to contain the name of
	// the tool, i.e. "inputs".

	optind = 1
	terminator := '\n'
	delimiter := "\t"
	//kLongOptions := []option { { "help", no_argument, nil, 'h' },
	//                                { "delimiter", required_argument, nil,
	//                                  'd' },
	//                                { "print0", no_argument, nil, '0' },
	//                                { "", 0, nil, 0 } }
	opts, optind, err := getopt.Getopts(*args, "d:h0")
	if err != nil {
		panic(err)
	}
	*args = (*args)[optind:]
	for _, optV := range opts {
		opt := optV.Option
		optarg := optV.Value
		switch opt {
		case 'd':
			delimiter = optarg
		case '0':
			terminator = 0
		default: // case 'h':
			// clang-format off
			fmt.Print(
				"Usage '-t multi-inputs [options] [targets]\n" +
					"\n" +
					"Print one or more sets of inputs required to build targets, sorted in dependency order.\n" +
					"The tool works like inputs tool but with addition of the target for each line.\n" +
					"The output will be a series of lines with the following elements:\n" +
					"<target> <delimiter> <input> <terminator>\n" +
					"Note that a given input may appear for several targets if it is used by more than one targets.\n" +
					"Options:\n" +
					"  -h, --help                   Print this message.\n" +
					"  -d  --delimiter=DELIM        Use DELIM instead of TAB for field delimiter.\n" +
					"  -0, --print0                 Use \\0, instead of \\n as a line terminator.\n",
			)
			// clang-format on
			return 1
		}
	}

	nodes := []*Node{}
	err1 := ""
	if !this.CollectTargetsFromArgs(args, &nodes, &err1) {
		Error("%s", err1)
		return 1
	}

	for _, node := range nodes {
		collector := InputsCollector{}

		collector.VisitNode(node)
		inputs := collector.GetInputsAsStrings(false)

		for _, input := range inputs {
			fmt.Printf("%s%s%s", node.path(), delimiter, input)
			fmt.Fprint(os.Stdout, terminator)

		}
	}

	return 0
}

type Action int8

const (
	kDisplayHelpAndExit Action = 0
	kEmitCommands       Action = 1
)

type EvaluateCommandMode int8

const (
	ECM_NORMAL         EvaluateCommandMode = 0
	ECM_EXPAND_RSPFILE                     = 1
)

type CompdbTargets struct {
	action    Action
	eval_mode EvaluateCommandMode

	targets []string
}

func EvaluateCommandWithRspfile(edge *Edge, mode EvaluateCommandMode) string {
	command := edge.EvaluateCommand(false)
	if mode == ECM_NORMAL {
		return command
	}

	rspfile := edge.GetUnescapedRspfile()
	if rspfile == "" {
		return command
	}

	index := strings.Index(command, rspfile)
	if index == 0 || index == -1 || (command[index-1] != '@' &&
		strings.Index(command, "--option-file=") != index-14 &&
		strings.Index(command, "-f ") != index-3) {
		return command
	}

	rspfile_content := edge.GetBinding("rspfile_content")
	newline_index := 0
	for {
		nextNewlineIndex := strings.IndexByte(rspfile_content[newline_index:], '\n')
		if nextNewlineIndex == -1 {
			break
		}
		newline_index += nextNewlineIndex
		rspfile_content = rspfile_content[:newline_index] + " " + rspfile_content[newline_index+1:]
		newline_index++
	}
	if command[index-1] == '@' {
		command = command[:index-1] + rspfile_content + command[index+len(rspfile):]
	} else if strings.Contains(command[:index-3], "-f ") {
		command = command[:index-3] + rspfile_content + command[index+len(rspfile)+3:]
	} else { // --option-file syntax
		command = command[:index-14] + rspfile_content + command[index+len(rspfile)+14:]
	}
	return command
}

func CreateFromArgs(args *[]string) CompdbTargets {
	//
	// grammar:
	//     ninja -t compdb-targets [-hx] target [targets]
	//
	ret := CompdbTargets{}

	// getopt_long() expects argv[0] to contain the name of
	// the tool, i.e. "compdb-targets".

	// Phase 1: parse options:
	// optind = 1  //see `man 3 getopt` for documentation on optind
	opts, optind, err := getopt.Getopts(*args, "hx")
	if err != nil {
		panic(err)
	}
	*args = (*args)[optind:]
	for _, optV := range opts {
		opt := optV.Option
		switch opt {
		case 'x':
			ret.eval_mode = ECM_EXPAND_RSPFILE

		default: //case 'h':
			ret.action = kDisplayHelpAndExit
			return ret
		}
	}

	// Phase 2: parse operands:
	targets_begin := optind
	targets_end := len(*args)

	if targets_begin == targets_end {
		Error("compdb-targets expects the name of at least one target")
		ret.action = kDisplayHelpAndExit
	} else {
		ret.action = kEmitCommands
		for i := targets_begin; i < targets_end; i++ {
			ret.targets = append(ret.targets, (*args)[i])
		}
	}

	return ret
}

func (this *NinjaMain) ToolCompilationDatabaseForTargets(options *Options, args *[]string) int {
	compdb := CreateFromArgs(args)

	switch compdb.action {
	case kDisplayHelpAndExit:
		{
			fmt.Printf(
				"usage: ninja -t compdb [-hx] target [targets]\n" +
					"\n" +
					"options:\n" +
					"  -h     display this help message\n" +
					"  -x     expand @rspfile style response file invocations\n")
			return 1
		}

	case kEmitCommands:
		{
			collector := CommandCollector{}

			for _, target_arg := range compdb.targets {
				err := ""
				node := this.CollectTarget(target_arg, &err)
				if node == nil {
					log.Fatalf("%s", err)
					return 1
				}
				if node.in_edge() == nil {
					log.Fatalf(
						"'%s' is not a target "+
							"(i.e. it is not an output of any `build` statement)",
						node.path())
				}
				collector.CollectFrom(node)
			}

			directory := GetWorkingDirectory()
			PrintCompdb(directory, collector.in_edges, compdb.eval_mode)
		}
	}

	return 0
}

func PrintCompdb(directory string, edges []*Edge, eval_mode EvaluateCommandMode) {
	fmt.Print('[')

	first := true
	for _, edge := range edges {
		if edge.is_phony() || len(edge.inputs_) == 0 {
			continue
		}
		if !first {
			fmt.Print(',')
		}
		PrintOneCompdbObject(directory, edge, eval_mode)
		first = false
	}

	fmt.Print("\n]")
}

func (this *NinjaMain) ToolUrtle(options *Options, args *[]string) int {
	// RLE encoded.
	urtle :=
		" 13 ,3;2!2;\n8 ,;<11!;\n5 `'<10!(2`'2!\n11 ,6;, `\\. `\\9 .,c13$ec,.\n6 " +
			",2;11!>; `. ,;!2> .e8$2\".2 \"?7$e.\n <:<8!'` 2.3,.2` ,3!' ;,(?7\";2!2'<" +
			"; `?6$PF ,;,\n2 `'4!8;<!3'`2 3! ;,`'2`2'3!;4!`2.`!;2 3,2 .<!2'`).\n5 3`5" +
			"'2`9 `!2 `4!><3;5! J2$b,`!>;2!:2!`,d?b`!>\n26 `'-;,(<9!> $F3 )3.:!.2 d\"" +
			"2 ) !>\n30 7`2'<3!- \"=-='5 .2 `2-=\",!>\n25 .ze9$er2 .,cd16$bc.'\n22 .e" +
			"14$,26$.\n21 z45$c .\n20 J50$c\n20 14$P\"`?34$b\n20 14$ dbc `2\"?22$?7$c" +
			"\n20 ?18$c.6 4\"8?4\" c8$P\n9 .2,.8 \"20$c.3 ._14 J9$\n .2,2c9$bec,.2 `?" +
			"21$c.3`4%,3%,3 c8$P\"\n22$c2 2\"?21$bc2,.2` .2,c7$P2\",cb\n23$b bc,.2\"2" +
			"?14$2F2\"5?2\",J5$P\" ,zd3$\n24$ ?$3?%3 `2\"2?12$bcucd3$P3\"2 2=7$\n23$P" +
			"\" ,3;<5!>2;,. `4\"6?2\"2 ,9;, `\"?2$\n"
	count := int32(0)
	for _, p := range urtle {
		if '0' <= p && p <= '9' {
			count = count*10 + p - '0'
		} else {
			for i := 0; i < int(math.Max(float64(count), 1)); i++ {
				fmt.Printf("%c", p)
			}
			count = 0
		}
	}
	return 0
}

func (this *NinjaMain) ToolDeps(options *Options, args *[]string) int {
	nodes := []*Node{}
	if len(*args) == 0 {
		for _, ni := range this.DepsLog.nodes() {
			if IsDepsEntryLiveFor(ni) {
				nodes = append(nodes, ni)
			}
		}
	} else {
		err := ""
		if !this.CollectTargetsFromArgs(args, &nodes, &err) {
			Error("%s", err)
			return 1
		}
	}

	var disk_interface RealDiskInterface
	for _, it := range nodes {
		deps := this.DepsLog.GetDeps(it)
		if deps == nil {
			fmt.Printf("%s: deps not found\n", (*it).path())
			continue
		}

		mtime, notExist, err := disk_interface.StatNode(it)
		if err != nil {
			Error("%s", err.Error()) // Log and ignore Stat() errors;
		}
		if notExist || mtime != deps.mtime {
			fmt.Printf("%s: #deps %d, deps mtime %d (%s)\n",
				it.path(), deps.node_count, deps.mtime,
				"STALE")
		} else {
			fmt.Printf("%s: #deps %d, deps mtime %d (%s)\n",
				it.path(), deps.node_count, deps.mtime,
				"VALID")
		}

		for i := 0; i < deps.node_count; i++ {
			fmt.Printf("    %s\n", deps.nodes[i].path())
		}
		fmt.Printf("\n")
	}

	return 0
}

func (this *NinjaMain) ToolMissingDeps(options *Options, args *[]string) int {
	nodes := []*Node{}
	err := ""
	if !this.CollectTargetsFromArgs(args, &nodes, &err) {
		Error("%s", err)
		return 1
	}
	disk_interface := RealDiskInterface{}
	printer := MissingDependencyPrinter{}
	scanner := NewMissingDependencyScanner(&printer, &this.DepsLog, this.State_, &disk_interface)
	for _, it := range nodes {
		scanner.ProcessNode(it)
	}
	scanner.PrintStats()
	if scanner.HadMissingDeps() {
		return 3
	}
	return 0
}

func ToolTargetsList3(nodes []*Node, depth, indent int) int {
	for _, n := range nodes {
		for i := 0; i < indent; i++ {
			fmt.Printf("  ")
			target := n.path()
			if n.in_edge() != nil {
				fmt.Printf("%s: %s\n", target, n.in_edge().rule_.name())
				if depth > 1 || depth <= 0 {
					ToolTargetsList3(n.in_edge().inputs_, depth-1, indent+1)
				}
			} else {
				fmt.Printf("%s\n", target)
			}
		}
	}
	return 0
}
func ToolTargetsList2(state *State, rule_name string) int {
	rules := map[string]bool{}

	// Gather the outputs.
	for _, e := range state.edges_ {
		if e.rule_.name() == rule_name {
			for _, out_node := range e.outputs_ {
				rules[out_node.path()] = true
			}
		}
	}

	// Print them.
	for i, _ := range rules {
		fmt.Printf("%s\n", i)
	}

	return 0
}
func ToolTargetsList1(state *State) int {
	for _, e := range state.edges_ {
		for _, out_node := range e.outputs_ {
			fmt.Printf("%s: %s\n", out_node.path(), e.rule_.name())
		}
	}
	return 0
}

func ToolTargetsSourceList(state *State) int {
	for _, e := range state.edges_ {
		for _, inps := range e.inputs_ {
			if inps.in_edge() == nil {
				fmt.Printf("%s\n", inps.path())
			}
		}
	}
	return 0
}

func (this *NinjaMain) ToolTargets(options *Options, args *[]string) int {
	depth := 1
	if len(*args) >= 1 {
		mode := (*args)[0]
		if mode == "rule" {
			rule := ""
			if len(*args) > 1 {
				rule = (*args)[1]
			}
			if rule == "" {
				return ToolTargetsSourceList(this.State_)
			} else {
				return ToolTargetsList2(this.State_, rule)
			}
		} else if mode == "depth" {
			if len(*args) > 1 {
				depth, _ = strconv.Atoi((*args)[1])
			}
		} else if mode == "all" {
			return ToolTargetsList1(this.State_)
		} else {
			suggestion := SpellcheckString(mode, "rule", "depth", "all", "\000")
			if suggestion != "" {
				Error("unknown target tool mode '%s', did you mean '%s'?",
					mode, suggestion)
			} else {
				Error("unknown target tool mode '%s'", mode)
			}
			return 1
		}
	}

	err := ""
	root_nodes := this.State_.RootNodes(&err)
	if err == "" {
		return ToolTargetsList3(root_nodes, depth, 0)
	} else {
		Error("%s", err)
		return 1
	}
}
func (this *NinjaMain) ToolWinCodePage(options *Options, args *[]string) int {
	if len(*args) != 0 {
		fmt.Printf("usage: ninja -t wincodepage\n")
		return 1
	}
	//if GetACP() == CP_UTF8 {
	fmt.Printf("Build file encoding: %s\n", "UTF-8")
	//} else {
	//	fmt.Printf("Build file encoding: %s\n", "ANSI")
	//}

	return 0
}
func (this *NinjaMain) ToolGraph(options *Options, args *[]string) int {
	nodes := []*Node{}
	err := ""
	if !this.CollectTargetsFromArgs(args, &nodes, &err) {
		Error("%s", err)
		return 1
	}

	graph := NewGraphViz(this.State_, this.DiskInterface)
	graph.Start()
	for _, n := range nodes {
		graph.AddTarget(n)
	}
	graph.Finish()

	return 0
}

func (this *NinjaMain) ToolQuery(options *Options, args *[]string) int {
	if len(*args) == 0 {
		Error("expected a target to query")
		return 1
	}

	dyndep_loader := NewDyndepLoader(this.State_, this.DiskInterface, nil)

	for i := 0; i < len(*args); i++ {
		err := ""
		node := this.CollectTarget((*args)[i], &err)
		if node == nil {
			Error("%s", err)
			return 1
		}

		fmt.Printf("%s:\n", node.path())
		if edge := node.in_edge(); edge != nil {
			if edge.dyndep_ != nil && edge.dyndep_.dyndep_pending() {
				if !dyndep_loader.LoadDyndeps(edge.dyndep_, &err) {
					Warning("%s\n", err)
				}
			}
			fmt.Printf("  input: %s\n", edge.rule_.name())
			for in := 0; in < int(len(edge.inputs_)); in++ {
				label := ""
				if edge.is_implicit(int64(in)) {
					label = "| "
				} else if edge.is_order_only(int64(in)) {
					label = "|| "
				}
				fmt.Printf("    %s%s\n", label, edge.inputs_[in].path())
			}
			if len(edge.validations_) != 0 {
				fmt.Printf("  validations:\n")
				for _, validation := range edge.validations_ {
					fmt.Printf("    %s\n", validation.path())
				}
			}
		}
		fmt.Printf("  outputs:\n")
		for _, edge := range node.out_edges() {
			for _, out := range edge.outputs_ {
				fmt.Printf("    %s\n", (*out).path())
			}
		}
		validation_edges := node.validation_out_edges()
		if len(validation_edges) != 0 {
			fmt.Printf("  validation for:\n")
			for _, edge := range validation_edges {
				for _, out := range edge.outputs_ {
					fmt.Printf("    %s\n", (*out).path())
				}
			}
		}
	}
	return 0
}
func PrintOneCompdbObject(directory string, edge *Edge, eval_mode EvaluateCommandMode) {
	fmt.Printf("\n  {\n    \"directory\": \"")
	PrintJSONString(directory)
	fmt.Printf("\",\n    \"command\": \"")
	PrintJSONString(EvaluateCommandWithRspfile(edge, eval_mode))
	fmt.Printf("\",\n    \"file\": \"")
	PrintJSONString(edge.inputs_[0].path())
	fmt.Printf("\",\n    \"output\": \"")
	PrintJSONString(edge.outputs_[0].path())
	fmt.Printf("\"\n  }")
}
func (this *NinjaMain) ToolCompilationDatabase(options *Options, args *[]string) int {
	// The compdb tool uses getopt, and expects argv[0] to contain the name of
	// the tool, i.e. "compdb".

	eval_mode := ECM_NORMAL

	opts, optind, err := getopt.Getopts(*args, "hx")
	if err != nil {
		panic(err)
	}
	*args = (*args)[optind:]
	for _, optV := range opts {
		opt := optV.Option
		switch opt {
		case 'x':
			eval_mode = ECM_EXPAND_RSPFILE
		default: //  case 'h':
			fmt.Print(
				"usage: ninja -t compdb [options] [rules]\n" +
					"\n" +
					"options:\n" +
					"  -x     expand @rspfile style response file invocations\n",
			)
			return 1
		}
	}

	first := true

	directory := GetWorkingDirectory()
	fmt.Print('[')
	for _, edge := range this.State_.edges_ {
		if len(edge.inputs_) == 0 {
			continue
		}
		if len(*args) == 0 {
			if !first {
				fmt.Print(',')
			}
			PrintOneCompdbObject(directory, edge, eval_mode)
			first = false
		} else {
			for i := 0; i != len(*args); i++ {
				if edge.rule_.name() == (*args)[i] {
					if !first {
						fmt.Print(',')
					}
					PrintOneCompdbObject(directory, edge, eval_mode)
					first = false
				}
			}
		}
	}

	fmt.Print("\n]")
	return 0
}

func (this *NinjaMain) ToolRecompact(options *Options, args *[]string) int {
	if !this.EnsureBuildDirExists() {
		return 1
	}

	if !this.OpenBuildLog( /*recompact_only=*/ true) || !this.OpenDepsLog( /*recompact_only=*/ true) {
		return 1
	}

	return 0
}

func (this *NinjaMain) ToolRestat(options *Options, args *[]string) int {
	// The restat tool uses getopt, and expects argv[0] to contain the name of the
	// tool, i.e. "restat"

	opts, optind, err1 := getopt.Getopts(*args, "h")
	if err1 != nil {
		panic(err1)
	}
	*args = (*args)[optind:]
	for _, optV := range opts {
		opt := optV.Option
		switch opt {
		case 'h':
		default:
			fmt.Printf("usage: ninja -t restat [outputs]\n")
			return 1
		}
	}

	if !this.EnsureBuildDirExists() {
		return 1
	}

	log_path := ".ninja_log"
	if this.BuildDir != "" {
		log_path = this.BuildDir + "/" + log_path
	}

	err := ""
	status := this.BuildLog.Load(log_path, &err)
	if status == LOAD_ERROR {
		Error("loading build log %s: %s", log_path, err)
		return EXIT_FAILURE
	}
	if status == LOAD_NOT_FOUND {
		// Nothing to restat, ignore this
		return EXIT_SUCCESS
	}
	if err != "" {
		// Hack: Load() can return a warning via err by returning LOAD_SUCCESS.
		Warning("%s", err)
		err = ""
	}

	success := this.BuildLog.Restat(log_path, this.DiskInterface, *args, &err)
	if !success {
		Error("failed recompaction: %s", err)
		return EXIT_FAILURE
	}

	if !this.Config_.DryRun {
		if !this.BuildLog.OpenForWrite(log_path, this, &err) {
			Error("opening build log: %s", err)
			return EXIT_FAILURE
		}
	}

	return EXIT_SUCCESS
}

func (this *NinjaMain) IsPathDead(s string) bool {
	n := this.State_.LookupNode(s)
	if n != nil && n.in_edge() != nil {
		return false
	}
	// Just checking n isn't enough: If an old output is both in the build log
	// and in the deps log, it will have a Node object in state_.  (It will also
	// have an in edge if one of its inputs is another output that's in the deps
	// log, but having a deps edge product an output that's input to another deps
	// edge is rare, and the first recompaction will delete all old outputs from
	// the deps log, and then a second recompaction will clear the build log,
	// which seems good enough for this corner case.)
	// Do keep entries around for files which still exist on disk, for
	// generators that want to use this information.
	_, notExist, err := this.DiskInterface.Stat(s)
	if err != nil {
		Error("%s", err.Error()) // Log and ignore Stat() errors.
	}
	return notExist
}

func (this *NinjaMain) ToolRules(options *Options, args *[]string) int {
	// Parse options.

	// The rules tool uses getopt, and expects argv[0] to contain the name of
	// the tool, i.e. "rules".

	print_description := false

	opts, optind, err := getopt.Getopts(*args, "hd")
	if err != nil {
		panic(err)
	}
	*args = (*args)[optind:]
	for _, optV := range opts {
		opt := optV.Option
		switch opt {
		case 'd':
			print_description = true
		default: // case 'h':
			fmt.Print("usage: ninja -t rules [options]\n" +
				"\n" +
				"options:\n" +
				"  -d     also print the description of the rule\n" +
				"  -h     print this message\n",
			)
			return 1
		}
	}

	// Print rules

	type Rules map[string]*Rule
	rules := this.State_.bindings_.GetRules()
	for key, val := range rules {
		fmt.Printf("%s", key)
		if print_description {
			rule := val
			description := rule.GetBinding("description")
			if description != nil {
				fmt.Printf(": %s", description.Unparse())
			}
		}
		fmt.Printf("\n")
		// fflush(stdout);
	}
	return 0
}

func ChooseTool(tool_name string) *Tool {
	this := &NinjaMain{}
	kTools := []Tool{
		{"browse", "browse dependency graph in a web browser",
			RUN_AFTER_LOAD, this.ToolBrowse},
		{"msvc", "build helper for MSVC cl.exe (DEPRECATED)",
			RUN_AFTER_FLAGS, this.ToolMSVC},
		{"clean", "clean built files",
			RUN_AFTER_LOAD, this.ToolClean},
		{"commands", "list all commands required to rebuild given targets",
			RUN_AFTER_LOAD, this.ToolCommands},
		{"inputs", "list all inputs required to rebuild given targets",
			RUN_AFTER_LOAD, this.ToolInputs},
		{"multi-inputs", "print one or more sets of inputs required to build targets",
			RUN_AFTER_LOAD, this.ToolMultiInputs},
		{"deps", "show dependencies stored in the deps log",
			RUN_AFTER_LOGS, this.ToolDeps},
		{"missingdeps", "check deps log dependencies on generated files",
			RUN_AFTER_LOGS, this.ToolMissingDeps},
		{"graph", "output graphviz dot file for targets",
			RUN_AFTER_LOAD, this.ToolGraph},
		{"query", "show inputs/outputs for a path",
			RUN_AFTER_LOGS, this.ToolQuery},
		{"targets", "list targets by their rule or depth in the DAG",
			RUN_AFTER_LOAD, this.ToolTargets},
		{"compdb", "dump JSON compilation database to stdout",
			RUN_AFTER_LOAD, this.ToolCompilationDatabase},
		{"compdb-targets",
			"dump JSON compilation database for a given list of targets to stdout",
			RUN_AFTER_LOAD, this.ToolCompilationDatabaseForTargets},
		{"recompact", "recompacts ninja-internal data structures",
			RUN_AFTER_LOAD, this.ToolRecompact},
		{"restat", "restats all outputs in the build log",
			RUN_AFTER_FLAGS, this.ToolRestat},
		{"rules", "list all rules",
			RUN_AFTER_LOAD, this.ToolRules},
		{"cleandead", "clean built files that are no longer produced by the manifest",
			RUN_AFTER_LOGS, this.ToolCleanDead},
		{"urtle", "",
			RUN_AFTER_FLAGS, this.ToolUrtle},
		{"wincodepage", "print the Windows code page used by ninja",
			RUN_AFTER_FLAGS, this.ToolWinCodePage},
		{"", "", RUN_AFTER_FLAGS, nil},
	}

	if tool_name == "list" {
		fmt.Printf("ninja subtools:\n")
		for _, tool := range kTools {
			if tool.Desc != "" {
				fmt.Printf("%11s  %s\n", tool.Name, tool.Desc)
			}
		}
		return nil
	}

	for _, tool := range kTools {
		if tool.Name == tool_name {
			return &tool
		}
	}

	words := []string{}
	for _, tool := range kTools {
		words = append(words, tool.Name)
	}
	suggestion := SpellcheckStringV(tool_name, words)
	if suggestion != "" {
		log.Fatalf("unknown tool '%s', did you mean '%s'?", tool_name, suggestion)
	} else {
		log.Fatalf("unknown tool '%s'", tool_name)
	}
	return nil // Not reached.
}

// / Enable a debugging mode.  Returns false if Ninja should exit instead
// / of continuing.
func DebugEnable(name string) bool {
	if name == "list" {
		fmt.Printf("debugging modes:\n" +
			"  stats        print operation counts/timing info\n" +
			"  explain      explain what caused a command to execute\n" +
			"  keepdepfile  don't delete depfiles after they're read by ninja\n" +
			"  keeprsp      don't delete @response files on success\n" +
			"  nostatcache  don't batch stat() calls per directory and cache them\n" +
			"multiple modes can be enabled via -d FOO -d BAR\n")
		return false
	} else if name == "stats" {
		GMetrics = &Metrics{}
		return true
	} else if name == "explain" {
		g_explaining = true
		return true
	} else if name == "keepdepfile" {
		g_keep_depfile = true
		return true
	} else if name == "keeprsp" {
		g_keep_rsp = true
		return true
	} else if name == "nostatcache" {
		g_experimental_statcache = false
		return true
	} else {
		suggestion := SpellcheckString(name, "stats", "explain", "keepdepfile", "keeprsp", "nostatcache", "\000")
		if suggestion != "" {
			Error("unknown debug setting '%s', did you mean '%s'?", name, suggestion)
		} else {
			Error("unknown debug setting '%s'", name)
		}
		return false
	}
}
