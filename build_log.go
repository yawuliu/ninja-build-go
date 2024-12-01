package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

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

func NewBuildLog() *BuildLog {
	ret := BuildLog{}
	return &ret
}
func (this *BuildLog) ReleaseBuildLog() {
	this.Close()
}

// / Prepares writing to the log file without actually opening it - that will
// / happen when/if it's needed
func (this *BuildLog) OpenForWrite(path string, user BuildLogUser, err *string) bool {
	if this.needs_recompaction_ {
		if !this.Recompact(path, user, err) {
			return false
		}
	}

	if this.log_file_ == nil {
		panic("!this.log_file_")
	}
	this.log_file_path_ = path // we don't actually open the file right now, but will
	// do so on the first write attempt
	return true
}

func (this *BuildLog) RecordCommand(edge *Edge, start_time int, end_time int, mtime TimeStamp) bool {
	command := edge.EvaluateCommand(true)
	command_hash := HashCommand(command)
	for _, out := range edge.outputs_ {
		path := out.path()
		second, ok := this.entries_[path]
		var log_entry *LogEntry = nil
		if ok {
			log_entry = second
		} else {
			log_entry = NewLogEntry(path)
			this.entries_[log_entry.output] = log_entry
		}
		log_entry.command_hash = command_hash
		log_entry.start_time = start_time
		log_entry.end_time = end_time
		log_entry.mtime = mtime

		if !this.OpenForWriteIfNeeded() {
			return false
		}
		if this.log_file_ != nil {
			_, err1 := this.WriteEntry(this.log_file_, log_entry)
			if err1 != nil {
				return false
			}
			err1 = this.log_file_.Sync()
			if err1 != nil {
				return false
			}
		}
	}
	return true
}
func (this *BuildLog) Close() {
	this.OpenForWriteIfNeeded() // create the file even if nothing has been recorded
	if this.log_file_ != nil {
		this.log_file_.Close()
	}
	this.log_file_ = nil
}

// / Load the on-disk log.
func (this *BuildLog) Load(path string, err1 *string) LoadStatus {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LOAD_NOT_FOUND
		}
		*err1 = err.Error()
		return LOAD_ERROR
	}
	defer file.Close()

	logVersion := 0
	uniqueEntryCount := 0
	totalEntryCount := 0

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			*err1 = err.Error()
			return LOAD_ERROR
		}
		line = strings.TrimSpace(line)

		if logVersion == 0 {
			const signaturePrefix = " ninja_log_v"
			if strings.HasPrefix(line, signaturePrefix) {
				versionStr := strings.TrimPrefix(line, signaturePrefix)
				logVersion, err = strconv.Atoi(versionStr)
				if err != nil {
					*err1 = err.Error()
					return LOAD_ERROR
				}

				if logVersion < kOldestSupportedVersion {
					*err1 = fmt.Errorf("build log version is too old; starting over").Error()
					return LOAD_NOT_FOUND
				} else if logVersion > kCurrentVersion {
					*err1 = fmt.Errorf("build log version is too new; starting over").Error()
					return LOAD_NOT_FOUND
				}
			}
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}

		startTime, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		endTime, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		mtime, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			continue
		}
		output := fields[3]

		entry, exists := this.entries_[output]
		if !exists {
			entry = &LogEntry{output: output}
			this.entries_[output] = entry
			uniqueEntryCount++
		}
		totalEntryCount++

		entry.start_time = startTime
		entry.end_time = endTime
		entry.mtime = TimeStamp(mtime)

		if len(fields) > 4 {
			commandHash, err := strconv.ParseUint(fields[4], 16, 64)
			if err != nil {
				continue
			}
			entry.command_hash = commandHash
		}
	}
	// Decide whether it's time to rebuild the log:
	// - if we're upgrading versions
	// - if it's getting large
	kMinCompactionEntryCount := 100
	kCompactionRatio := 3
	if logVersion < kCurrentVersion {
		this.needs_recompaction_ = true
	} else if totalEntryCount > kMinCompactionEntryCount && totalEntryCount > uniqueEntryCount*kCompactionRatio {
		this.needs_recompaction_ = true
	}

	if uniqueEntryCount == 0 {
		return LOAD_SUCCESS
	}

	return LOAD_SUCCESS
}

// / Lookup a previously-run command by its output path.
func (this *BuildLog) LookupByOutput(path string) *LogEntry {
	i, ok := this.entries_[path]
	if ok {
		return i
	}
	return nil
}

// / Serialize an entry into a log file.
func (this *BuildLog) WriteEntry(f *os.File, entry *LogEntry) (bool, error) {
	_, err := fmt.Fprintf(f, "%d\t%d\t%"+PRId64+"\t%s\t%"+PRIx64+"\n",
		entry.start_time, entry.end_time, entry.mtime,
		entry.output, entry.command_hash)
	return err == nil, err
}

// / Rewrite the known log entries, throwing away old data.
func (this *BuildLog) Recompact(path string, user BuildLogUser, err *string) bool {
	METRIC_RECORD(".ninja_log recompact")

	this.Close()
	temp_path := path + ".recompact"
	f, err1 := os.OpenFile(temp_path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err1 != nil {
		*err = err1.Error()
		return false
	}
	_, err1 = fmt.Fprintf(f, kFileSignature, kCurrentVersion)
	if err1 != nil {
		*err = err1.Error()
		f.Close()
		return false
	}

	dead_outputs := []string{}
	for first, second := range this.entries_ {
		if user.IsPathDead(first) {
			dead_outputs = append(dead_outputs, first)
			continue
		}

		_, err1 := this.WriteEntry(f, second)
		if err1 != nil {
			*err = err1.Error()
			f.Close()
			return false
		}
	}

	for i := 0; i < len(dead_outputs); i++ {
		delete(this.entries_, dead_outputs[i])
	}

	f.Close()
	err1 = os.RemoveAll(path)
	if err1 != nil {
		*err = err1.Error()
		return false
	}

	err1 = os.Rename(temp_path, path)
	if err1 != nil {
		*err = err1.Error()
		return false
	}

	return true
}

// / Restat all outputs in the log
func (this *BuildLog) Restat(path string, disk_interface DiskInterface, outputs []string, err *string) bool {
	METRIC_RECORD(".ninja_log restat")
	output_count := len(outputs)

	this.Close()
	temp_path := path + ".restat"
	file, err1 := os.OpenFile(temp_path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err1 != nil {
		*err = err1.Error()
		return false
	}
	defer file.Close()
	_, err1 = fmt.Fprintf(file, kFileSignature, kCurrentVersion)
	if err1 != nil {
		*err = err1.Error()
		return false
	}
	for _, second := range this.entries_ {
		skip := output_count > 0
		for j := 0; j < output_count; j++ {
			if second.output == outputs[j] {
				skip = false
				break
			}
		}
		if !skip {
			var mtime TimeStamp = disk_interface.Stat(second.output, err)
			if mtime == -1 {
				return false
			}
			second.mtime = mtime
		}
		_, err1 = this.WriteEntry(file, second)
		if err1 != nil {
			*err = err1.Error()
			return false
		}
	}

	err1 = os.Remove(path)
	if err1 != nil {
		*err = err1.Error()
		return false
	}
	err1 = os.Rename(temp_path, path)
	if err1 != nil {
		*err = err1.Error()
		return false
	}

	return true
}

type Entries map[string]*LogEntry

func (this *BuildLog) entries() Entries { return this.entries_ }

const kFileSignature = "# ninja log v%d\n"
const kOldestSupportedVersion = 7
const kCurrentVersion = 7

// / Should be called before using log_file_. When false is returned, errno
// / will be set.
func (this *BuildLog) OpenForWriteIfNeeded() bool {
	if this.log_file_ != nil || this.log_file_path_ == "" {
		return true
	}
	file, err := os.OpenFile(this.log_file_path_, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false
	}
	logFile := bufio.NewWriter(file)
	if _, err := file.Seek(0, os.SEEK_END); err != nil {
		return false
	}
	if _, err := file.Readdirnames(1); err != io.EOF {
		_, err := logFile.WriteString(fmt.Sprintf(kFileSignature, kCurrentVersion))
		if err != nil {
			return false
		}
	}

	this.log_file_ = file
	return true
}

func HashCommand(command string) uint64 {
	return rapidhash([]byte(command), len(command))
}

// Used by tests.
func (this *LogEntry) CompareLogEntryEq(o *LogEntry) bool {
	return this.output == o.output && this.command_hash == o.command_hash &&
		this.start_time == o.start_time && this.end_time == o.end_time &&
		this.mtime == o.mtime
}

func NewLogEntry(output string) *LogEntry {
	ret := LogEntry{}
	ret.output = output
	return &ret
}
func NewLogEntry1(output string, command_hash uint64, start_time, end_time int, mtime TimeStamp) *LogEntry {
	ret := LogEntry{}
	ret.output = output
	ret.command_hash = command_hash
	ret.start_time = start_time
	ret.end_time = end_time
	ret.mtime = mtime
	return &ret
}
