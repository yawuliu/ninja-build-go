package main

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
	if g_metrics != nil {
		metrics_h_metric = g_metrics.NewMetric(name)
	} else {
		metrics_h_metric = nil
	}

	metrics_h_scoped = ScopedMetric(metrics_h_metric)
}

// / A variant of METRIC_RECORD that doesn't record anything if |condition|
// / is false.
func METRIC_RECORD_IF(name string, condition bool) {
	if g_metrics != nil {
		metrics_h_metric = g_metrics.NewMetric(name)
	} else {
		metrics_h_metric = nil
	}
	if condition {
		metrics_h_scoped = ScopedMetric(metrics_h_metric)
	} else {
		metrics_h_scoped = nil
	}
}
