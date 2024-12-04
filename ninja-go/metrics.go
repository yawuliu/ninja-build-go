package main

import (
	"fmt"
	"time"
)

// / A simple stopwatch which returns the time
// / in seconds since Restart() was called.
type Stopwatch struct {
	started_ uint64
	// Return the current time using the native frequency of the high resolution
	// timer.
}

// / The primary interface to metrics.  Use METRIC_RECORD("foobar") at the top
// / of a function to get timing stats recorded for each call of the function.
var metrics_h_metric *Metric = nil

func METRIC_RECORD(name string) {
	if GMetrics != nil {
		metrics_h_metric = GMetrics.NewMetric(name)
	} else {
		metrics_h_metric = nil
	}

	// metrics_h_scoped = ScopedMetric(metrics_h_metric)
}

// / A variant of METRIC_RECORD that doesn't record anything if |condition|
// / is false.
func METRIC_RECORD_IF(name string, condition bool) {
	if GMetrics != nil {
		metrics_h_metric = GMetrics.NewMetric(name)
	} else {
		metrics_h_metric = nil
	}
	//if condition {
	//	metrics_h_scoped = ScopedMetric(metrics_h_metric)
	//} else {
	//	metrics_h_scoped = nil
	//}
}

var GMetrics *Metrics = nil

type Metric struct {
	name string
	/// Number of times we've hit the code path.
	count int
	/// Total time (in platform-dependent units) we've spent on the code path.
	sum int64
}

type Metrics struct {
	metrics_ []*Metric
}

func (this *Metrics) NewMetric(name string) *Metric {
	metric := Metric{}
	metric.name = name
	metric.count = 0
	metric.sum = 0
	this.metrics_ = append(this.metrics_, &metric)
	return &metric
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// / Print a summary report to stdout.
func (this *Metrics) Report() {
	width := 0
	for _, i := range this.metrics_ {
		width = max(len(i.name), width)
	}

	fmt.Printf("%-*s\t%-6s\t%-9s\t%s\n", width,
		"metric", "count", "avg (us)", "total (ms)")
	for _, i := range this.metrics_ {
		metric := i
		micros := TimerToMicrosInt64(metric.sum)
		total := float64(micros) / float64(1000)
		avg := float64(micros) / float64(metric.count)
		fmt.Printf("%-*s\t%-6d\t%-8.1f\t%.1f\n", width, metric.name, metric.count, avg, total)
	}
}

func NewStopwatch() *Stopwatch {
	ret := Stopwatch{}
	ret.started_ = 0
	return &ret
}

// / Compute a platform-specific high-res timer value that fits into an int64.
func HighResTimer() int64 {
	now := time.Now()
	duration := now.UnixNano()
	return duration
}

func TimerToMicrosInt64(dt int64) int64 {
	dtd := time.Duration(dt)           // 纳秒
	microseconds := dtd.Microseconds() // 转换为微秒
	return microseconds                // 输出结
}

func TimerToMicrosFloat64(dt float64) int64 {
	// dt is in ticks.  We want microseconds.
	dtd := time.Duration(dt) * time.Nanosecond // 假设以纳秒为单位
	microseconds := dtd.Microseconds()         // 直接转换为微秒
	return microseconds
}

// / Seconds since Restart() call.
func (this *Stopwatch) Elapsed() float64 {
	// Convert to micros after converting to double to minimize error.
	return 1e-6 * float64(TimerToMicrosFloat64(float64(this.NowRaw()-this.started_)))
}

func (this *Stopwatch) Restart() {
	this.started_ = this.NowRaw()
}

func (this *Stopwatch) NowRaw() uint64 {
	return uint64(HighResTimer())
}

func GetTimeMillis() int64 {
	return TimerToMicrosInt64(HighResTimer()) / 1000
}
