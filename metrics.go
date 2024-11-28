package main

import (
	"time"
)

var g_metrics *Metrics = nil

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

}

// / Print a summary report to stdout.
func (this *Metrics) Report() {

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
