package main

type Verbosity int8

const (
	QUIET            Verbosity = 0
	NO_STATUS_UPDATE           = 1
	NORMAL                     = 2
	VERBOSE                    = 3
)

type BuildConfig struct {
	verbosity        Verbosity
	dry_run          bool
	parallelism      int
	failures_allowed int
	/// The maximum load average we must not exceed. A negative value
	/// means that we do not have any limit.
	max_load_average       float64
	depfile_parser_options DepfileParserOptions
}

func NewBuildConfig() *BuildConfig {
	ret := BuildConfig{verbosity: NORMAL, dry_run: false, parallelism: 1, failures_allowed: 1, max_load_average: -0.0}
	return &ret
}

// / Map of running edge to time the edge started running.
type RunningEdgeMap map[*Edge]int

type Builder struct {
	state_          *State
	config_         *BuildConfig
	plan_           Plan
	command_runner_ *CommandRunner
	status_         Status

	running_edges_ RunningEdgeMap

	/// Time the build started.
	start_time_millis_ int64

	lock_file_path_ string
	disk_interface_ DiskInterface

	// Only create an Explanations class if '-d explain' is used.
	explanations_ *Explanations

	scan_ DependencyScan
}
