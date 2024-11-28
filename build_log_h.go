package main

import "os"

type BuildLog struct {
	entries_            Entries
	log_file_           *os.File
	log_file_path_      string
	needs_recompaction_ bool
}

type LogEntry struct {
	output       string
	command_hash uint64
	start_time   int
	end_time     int
	mtime        TimeStamp
}
type BuildLogUser interface {
	IsPathDead(path string) bool
}
