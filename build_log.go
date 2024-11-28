package main

import "os"

func NewBuildLog() *BuildLog {
	ret := BuildLog{}
	return &ret
}
func (this *BuildLog) ReleaseBuildLog() {

}

// / Prepares writing to the log file without actually opening it - that will
// / happen when/if it's needed
func (this *BuildLog) OpenForWrite(path string, user BuildLogUser, err *string) bool {}
func (this *BuildLog) RecordCommand(edge *Edge, start_time int, end_time int, mtime TimeStamp) bool {
}
func (this *BuildLog) Close() {}

// / Load the on-disk log.
func (this *BuildLog) Load(path string, err *string) LoadStatus {}

// / Lookup a previously-run command by its output path.
func (this *BuildLog) LookupByOutput(path string) *LogEntry {

}

// / Serialize an entry into a log file.
func (this *BuildLog) WriteEntry(f *os.File, entry *LogEntry) bool {}

// / Rewrite the known log entries, throwing away old data.
func (this *BuildLog) Recompact(path string, user *BuildLogUser, err *string) bool {}

// / Restat all outputs in the log
func (this *BuildLog) Restat(path string, disk_interface DiskInterface, outputs []string, err *string) bool {
}

type Entries map[string]*LogEntry

func (this *BuildLog) entries() *Entries { return this.entries_ }

// / Should be called before using log_file_. When false is returned, errno
// / will be set.
func (this *BuildLog) OpenForWriteIfNeeded() bool {}

func HashCommand(command StringPiece) uint64 {

}

// Used by tests.
func (this *LogEntry) CompareLogEntryEq(o *LogEntry) bool {
	return this.output == o.output && this.command_hash == o.command_hash &&
		this.start_time == o.start_time && this.end_time == o.end_time &&
		this.mtime == o.mtime
}

func NewLogEntry(output string) *LogEntry {

}
func NewLogEntry1(output string, command_hash uint64, start_time, end_time int, mtime TimeStamp) *LogEntry {

}
