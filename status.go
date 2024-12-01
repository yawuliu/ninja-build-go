package main

type Status interface {
	EdgeAddedToPlan(edge *Edge)
	EdgeRemovedFromPlan(edge *Edge)
	BuildEdgeStarted(edge *Edge, start_time_millis int64)
	BuildEdgeFinished(edge *Edge, start_time_millis int64, end_time_millis int64, success bool, output string)
	BuildStarted()
	BuildFinished()

	/// Set the Explanations instance to use to report explanations,
	/// argument can be nullptr if no explanations need to be printed
	/// (which is the default).
	SetExplanations(Explanations)

	Info(msg string, args ...interface{})
	Warning(msg string, args ...interface{})
	Error(msg string, args ...interface{})

	ReleaseStatus()

	/// creates the actual implementation

}
