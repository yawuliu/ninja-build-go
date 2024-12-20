package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type StatusPrinter struct {
	Status
	config_ *BuildConfig

	started_edges_  int
	finished_edges_ int
	total_edges_    int
	running_edges_  int

	/// How much wall clock elapsed so far?
	time_millis_ int64

	/// How much cpu clock elapsed so far?
	cpu_time_millis_ int64

	/// What percentage of predicted total time have elapsed already?
	time_predicted_percentage_ float64

	/// Out of all the edges, for how many do we know previous time?
	eta_predictable_edges_total_ int
	/// And how much time did they all take?
	eta_predictable_cpu_time_total_millis_ int64

	/// Out of all the non-finished edges, for how many do we know previous time?
	eta_predictable_edges_remaining_ int
	/// And how much time will they all take?
	eta_predictable_cpu_time_remaining_millis_ int64

	/// For how many edges we don't know the previous run time?
	eta_unpredictable_edges_remaining_ int

	/// Prints progress output.
	printer_ LinePrinter

	/// An optional Explanations pointer, used to implement `-d explain`.
	explanations_ Explanations

	/// The custom progress status format to use.
	progress_status_format_ string

	current_rate_ *SlidingRateInfo
}

func (this *StatusPrinter) RecalculateProgressPrediction() {

}

func NewStatusPrinter(config *BuildConfig) *StatusPrinter {
	ret := StatusPrinter{}
	ret.config_ = config
	ret.started_edges_ = 0
	ret.finished_edges_ = 0
	ret.total_edges_ = 0
	ret.running_edges_ = 0
	ret.progress_status_format_ = ""
	ret.current_rate_ = NewSlidingRateInfo(config.Parallelism)
	// Don't do anything fancy in verbose mode.
	if ret.config_.Verbosity != NORMAL {
		ret.printer_.set_smart_terminal(false)
	}

	ret.progress_status_format_ = os.Getenv("NINJA_STATUS")
	if ret.progress_status_format_ == "" {
		ret.progress_status_format_ = "[%f/%t] "
	}
	return &ret
}

// / Callbacks for the Plan to notify us about adding/removing Edge's.
func (this *StatusPrinter) EdgeAddedToPlan(edge *Edge) {
	this.total_edges_++

	// Do we know how long did this edge take last time?
	if edge.prev_elapsed_time_millis != -1 {
		this.eta_predictable_edges_total_++
		this.eta_predictable_edges_remaining_++
		this.eta_predictable_cpu_time_total_millis_ += edge.prev_elapsed_time_millis
		this.eta_predictable_cpu_time_remaining_millis_ +=
			edge.prev_elapsed_time_millis
	} else {
		this.eta_unpredictable_edges_remaining_++
	}
}
func (this *StatusPrinter) EdgeRemovedFromPlan(edge *Edge) {
	this.total_edges_--

	// Do we know how long did this edge take last time?
	if edge.prev_elapsed_time_millis != -1 {
		this.eta_predictable_edges_total_--
		this.eta_predictable_edges_remaining_--
		this.eta_predictable_cpu_time_total_millis_ -= edge.prev_elapsed_time_millis
		this.eta_predictable_cpu_time_remaining_millis_ -=
			edge.prev_elapsed_time_millis
	} else {
		this.eta_unpredictable_edges_remaining_--
	}
}

func (this *StatusPrinter) BuildEdgeStarted(edge *Edge, start_time_millis int64) {
	this.started_edges_++
	this.running_edges_++
	this.time_millis_ = start_time_millis

	if edge.use_console() || this.printer_.is_smart_terminal() {
		this.PrintStatus(edge, start_time_millis)
	}

	if edge.use_console() {
		this.printer_.SetConsoleLocked(true)
	}
}
func (this *StatusPrinter) BuildEdgeFinished(edge *Edge, start_time_millis int64, end_time_millis int64, success bool, output string) {
	this.time_millis_ = end_time_millis
	this.finished_edges_++

	elapsed := end_time_millis - start_time_millis
	this.cpu_time_millis_ += elapsed

	// Do we know how long did this edge take last time?
	if edge.prev_elapsed_time_millis != -1 {
		this.eta_predictable_edges_remaining_--
		this.eta_predictable_cpu_time_remaining_millis_ -=
			edge.prev_elapsed_time_millis
	} else {
		this.eta_unpredictable_edges_remaining_--
	}

	if edge.use_console() {
		this.printer_.SetConsoleLocked(false)
	}
	if this.config_.Verbosity == QUIET {
		return
	}

	if !edge.use_console() {
		this.PrintStatus(edge, end_time_millis)
	}

	this.running_edges_--

	// Print the command that is spewing before printing its output.
	if !success {
		outputs := ""
		for _, o := range edge.outputs_ {
			outputs += o.path() + " "
		}

		if this.printer_.supports_color() {
			this.printer_.PrintOnNewLine("\x1B[31m" + "FAILED: " + "\x1B[0m" + outputs + "\n")
		} else {
			this.printer_.PrintOnNewLine("FAILED: " + outputs + "\n")
		}
		this.printer_.PrintOnNewLine(edge.EvaluateCommand(false) + "\n")
	}

	if output != "" {

		// Fix extra CR being added on Windows, writing out CR CR LF (#773)
		os.Stdout.Sync() // Begin Windows extra CR fix
		// _setmode(_fileno(stdout), _O_BINARY);

		// ninja sets stdout and stderr of subprocesses to a pipe, to be able to
		// check if the output is empty. Some compilers, e.g. clang, check
		// isatty(stderr) to decide if they should print colored output.
		// To make it possible to use colored output with ninja, subprocesses should
		// be run with a flag that forces them to always print color escape codes.
		// To make sure these escape codes don't show up in a file if ninja's output
		// is piped to a file, ninja strips ansi escape codes again if it's not
		// writing to a |smart_terminal_|.
		// (Launching subprocesses in pseudo ttys doesn't work because there are
		// only a few hundred available on some systems, and ninja can launch
		// thousands of parallel compile commands.)
		if this.printer_.supports_color() || !strings.Contains(output, string('\x1b')) {
			this.printer_.PrintOnNewLine(output)
		} else {
			final_output := StripAnsiEscapeCodes(output)
			this.printer_.PrintOnNewLine(final_output)
		}

		os.Stdout.Sync()
		//  this._setmode(_fileno(stdout), _O_TEXT);  // End Windows extra CR fix
	}
}
func (this *StatusPrinter) BuildStarted() {
	this.started_edges_ = 0
	this.finished_edges_ = 0
	this.running_edges_ = 0
}
func (this *StatusPrinter) BuildFinished() {
	this.printer_.SetConsoleLocked(false)
	this.printer_.PrintOnNewLine("")
}

func (this *StatusPrinter) Info(msg string, args ...interface{}) {
	Info(msg, args...)
}
func (this *StatusPrinter) Warning(msg string, args ...interface{}) {
	Warning(msg, args...)
}
func (this *StatusPrinter) Error(msg string, args ...interface{}) {
	Error(msg, args...)
}

func FORMAT_TIME_HMMSS(t int64) string {
	return fmt.Sprintf("%d:%02d:%02d", (t)/3600, ((t)%3600)/60, (t)%60)
}

func FORMAT_TIME_MMSS(t int64) string {
	return fmt.Sprintf("%02d:%02d", (t)/60, (t)%60)
}
func SnprintfRate(rate float64, format string) string {
	if rate == -1 {
		return fmt.Sprintf(format, "?")
	} else {
		return fmt.Sprintf(format, rate)
	}
}

// / Format the progress status string by replacing the placeholders.
// / See the user manual for more information about the available
// / placeholders.
// / @param progress_status_format The format of the progress status.
// / @param status The status of the edge.
func (this *StatusPrinter) FormatProgressStatus(progress_status_format string, time_millis int64) string {
	var out strings.Builder
	for s := 0; s < len(progress_status_format); s++ {
		if progress_status_format[s] == '\000' {
			break
		}
		if progress_status_format[s] == '%' {
			s++
			switch progress_status_format[s] {
			case '%':
				out.WriteByte('%')
				// Started edges.
			case 's':
				buf := fmt.Sprintf("%d", this.started_edges_)
				out.WriteString(buf)
				// Total edges.
			case 't':
				buf := fmt.Sprintf("%d", this.total_edges_)
				out.WriteString(buf)
				// Running edges.
			case 'r':
				{
					buf := fmt.Sprintf("%d", this.running_edges_)
					out.WriteString(buf)
				}

				// Unstarted edges.
			case 'u':
				buf := fmt.Sprintf("%d", this.total_edges_-this.started_edges_)
				out.WriteString(buf)

				// Finished edges.
			case 'f':
				buf := fmt.Sprintf("%d", this.finished_edges_)
				out.WriteString(buf)

				// Overall finished edges per second.
			case 'o':
				buf := SnprintfRate(float64(this.finished_edges_)/(float64(this.time_millis_)/1e3), "%.1f")
				out.WriteString(buf)

				// Current rate, average over the last '-j' jobs.
			case 'c':
				this.current_rate_.UpdateRate(this.finished_edges_, this.time_millis_)
				buf := SnprintfRate(this.current_rate_.rate(), "%.1f")
				out.WriteString(buf)

				// Percentage of edges completed
			case 'p':
				{
					percent := 0
					if this.finished_edges_ != 0 && this.total_edges_ != 0 {
						percent = (100 * this.finished_edges_) / this.total_edges_
					}
					buf := fmt.Sprintf("%3i%%", percent)
					out.WriteString(buf)
				}
				// Wall time
				// elapsed, seconds
				// elapsed, human-readable
				// ETA, seconds
			case 'e', 'E', 'w', 'W': // ETA, human-readable
				{
					elapsed_sec := float64(this.time_millis_ / 1e3)
					eta_sec := float64(-1) // To be printed as "?".
					if this.time_predicted_percentage_ != 0.0 {
						// So, we know that we've spent time_millis_ wall clock,
						// and that is time_predicted_percentage_ percent.
						// How much time will we need to complete 100%?
						total_wall_time := float64(this.time_millis_) / this.time_predicted_percentage_
						// Naturally, that gives us the time remaining.
						eta_sec = (total_wall_time - float64(this.time_millis_)) / 1e3
					}

					print_with_hours :=
						elapsed_sec >= 60*60 || eta_sec >= 60*60

					sec := float64(-1)
					switch progress_status_format[s] {
					case 'e', 'w': // elapsed, human-readable // elapsed, seconds
						sec = elapsed_sec
						break
					case 'E', 'W': //  // ETA, seconds ETA, human-readable
						sec = eta_sec
						break
					}
					buf := ""
					if sec < 0 {
						buf = fmt.Sprintf("?")
					} else {
						switch progress_status_format[s] {
						case 'e', 'E': // ETA, seconds // elapsed, seconds
							buf = fmt.Sprintf("%.3f", sec)
						case 'w', 'W': // ETA, human-readable // elapsed, human-readable
							if print_with_hours {
								buf = fmt.Sprintf(FORMAT_TIME_HMMSS(int64(sec)))
							} else {
								buf = fmt.Sprintf(FORMAT_TIME_MMSS(int64(sec)))
							}
						}
					}
					out.WriteString(buf)
					break
				}

			// Percentage of time spent out of the predicted time total
			case 'P':
				{
					buf := fmt.Sprintf("%3i%%", (int)(100.*this.time_predicted_percentage_))
					out.WriteString(buf)
				}

			default:
				log.Fatalln("unknown placeholder '%%%c' in $NINJA_STATUS", progress_status_format[s])
				return ""
			}
		} else {
			out.WriteByte(progress_status_format[s])
		}
	}

	return out.String()
}

// / Set the |explanations_| pointer. Used to implement `-d explain`.
func (this *StatusPrinter) SetExplanations(explanations Explanations) {
	this.explanations_ = explanations
}

func (this *StatusPrinter) PrintStatus(edge *Edge, time_millis int64) {
	if this.explanations_ != nil {
		// Collect all explanations for the current edge's outputs.
		explanations := []string{}
		for _, output := range edge.outputs_ {
			this.explanations_.LookupAndAppend(output, explanations)
		}
		if len(explanations) != 0 {
			// Start a new line so that the first explanation does not append to the
			// status line.
			this.printer_.PrintOnNewLine("")
			for _, exp := range explanations {
				fmt.Fprintf(os.Stderr, "ninja explain: %s\n", exp)
			}
		}
	}

	if this.config_.Verbosity == QUIET || this.config_.Verbosity == NO_STATUS_UPDATE {
		return
	}

	this.RecalculateProgressPrediction()

	force_full_command := this.config_.Verbosity == VERBOSE

	to_print := edge.GetBinding("description")
	if to_print == "" || force_full_command {
		to_print = edge.GetBinding("command")
	}

	to_print = this.FormatProgressStatus(this.progress_status_format_, time_millis) + to_print
	if force_full_command {
		this.printer_.Print(to_print, FULL)
	} else {
		this.printer_.Print(to_print, ELIDE)
	}

}

type SlidingRateInfo struct {
	rate_        float64
	N            int
	times_       []float64
	last_update_ int
}

func NewSlidingRateInfo(n int) *SlidingRateInfo {
	ret := SlidingRateInfo{}
	ret.rate_ = -1
	ret.N = n
	ret.last_update_ = -1
	return &ret
}

func (this *SlidingRateInfo) rate() float64 {
	return this.rate_
}

func (s *SlidingRateInfo) UpdateRate(update_hint int, time_millis int64) {
	if update_hint == s.last_update_ {
		return
	}
	s.last_update_ = update_hint

	if len(s.times_) == s.N {
		// 移除最旧的时间
		s.times_ = s.times_[1:]
	}
	// 添加新的时间
	s.times_ = append(s.times_, float64(time_millis))

	// 计算速率
	if len(s.times_) > 1 {
		interval := time.Duration(s.times_[len(s.times_)-1]-s.times_[0]) * time.Millisecond
		s.rate_ = float64(len(s.times_)) / interval.Seconds()
	} else {
		s.rate_ = -1
	}
}

func Statusfactory(config *BuildConfig) Status {
	return NewStatusPrinter(config)
}
