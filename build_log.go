package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func NewBuildLog() *BuildLog {
	ret := BuildLog{}
	return &ret
}
func (this *BuildLog) ReleaseBuildLog() {
	this.Close();
}

// / Prepares writing to the log file without actually opening it - that will
// / happen when/if it's needed
func (this *BuildLog) OpenForWrite(path string, user BuildLogUser, err *string) bool {
  if this.needs_recompaction_ {
    if (!this.Recompact(path, user, err)) {
		return false
	}
  }

  if this.log_file_==nil {
	  panic("!this.log_file_")
  }
	this.log_file_path_ = path;  // we don't actually open the file right now, but will
                          // do so on the first write attempt
  return true;
}

func (this *BuildLog) RecordCommand(edge *Edge, start_time int, end_time int, mtime TimeStamp) bool {
  command := edge.EvaluateCommand(true);
   command_hash := HashCommand(command);
  for _,out :=range edge.outputs_ {
    path := out.path();
    i := this.entries_.find(path);
    var log_entry *LogEntry =nil
    if (i != this.entries_.end()) {
      log_entry = i.second;
    } else {
      log_entry = NewLogEntry(path);
      this.entries_.insert(Entries::value_type(log_entry.output, log_entry));
    }
    log_entry.command_hash = command_hash;
    log_entry.start_time = start_time;
    log_entry.end_time = end_time;
    log_entry.mtime = mtime;

    if (!this.OpenForWriteIfNeeded()) {
      return false;
    }
    if  this.log_file_!=nil {
      if !this.WriteEntry(this.log_file_, *log_entry) {
		  return false
	  }
      if (fflush(this.log_file_) != 0) {
          return false;
      }
    }
  }
  return true;
}
func (this *BuildLog) Close() {
  this.OpenForWriteIfNeeded();  // create the file even if nothing has been recorded
  if this.log_file_!=nil {
	  this.log_file_.Close()
  }
 this.log_file_ = nil
}

// / Load the on-disk log.
func (this *BuildLog) Load(path string, err *string) LoadStatus {
  METRIC_RECORD(".ninja_log load");
  file := os.OpenFile(path, "r");
  if (!file) {
    if (errno == ENOENT) {
		return LOAD_NOT_FOUND
	}
    *err = strerror(errno);
    return LOAD_ERROR;
  }

  log_version := 0
  unique_entry_count := 0
  total_entry_count := 0

   reader := NewLineReader(file);
  char* line_start = 0;
  char* line_end = 0;
  while (reader.ReadLine(&line_start, &line_end)) {
    if (!log_version) {
      sscanf(line_start, kFileSignature, &log_version);

      invalid_log_version := false;
      if (log_version < kOldestSupportedVersion) {
        invalid_log_version = true;
        *err = "build log version is too old; starting over";

      } else if (log_version > kCurrentVersion) {
        invalid_log_version = true;
        *err = "build log version is too new; starting over";
      }
      if (invalid_log_version) {
        fclose(file);
        unlink(path);
        // Don't report this as a failure. A missing build log will cause
        // us to rebuild the outputs anyway.
        return LOAD_NOT_FOUND;
      }
    }

    // If no newline was found in this chunk, read the next.
    if (!line_end) {
      continue
    }

     kFieldSeparator := '\t';

    start := line_start;
    end := static_cast<char*>(memchr(start, kFieldSeparator, line_end - start));
    if (!end) {
      continue
    }
    *end = 0;

    start_time := 0
    end_time := 0;
     var mtime TimeStamp = 0;

    start_time = atoi(start);
    start = end + 1;

    end = static_cast<char*>(memchr(start, kFieldSeparator, line_end - start));
    if (!end) {
      continue
    }
    *end = 0;
    end_time = atoi(start);
    start = end + 1;

    end = static_cast<char*>(memchr(start, kFieldSeparator, line_end - start));
    if (!end) {
      continue
    }
    *end = 0;
    mtime = strtoll(start, NULL, 10);
    start = end + 1;

    end = static_cast<char*>(memchr(start, kFieldSeparator, line_end - start));
    if (!end) {
      continue
    }
    output := string(start, end - start);

    start = end + 1;
    end = line_end;

    var entry *LogEntry =nil
    i := this.entries_.find(output);
    if (i != this.entries_.end()) {
      entry = i.second;
    } else {
      entry = new LogEntry(output);
      this.entries_.insert(Entries::value_type(entry.output, entry));
      ++unique_entry_count;
    }
    ++total_entry_count;

    entry.start_time = start_time;
    entry.end_time = end_time;
    entry.mtime = mtime;
    char c = *end; *end = '\0';
    entry.command_hash = (uint64_t)strtoull(start, NULL, 16);
    *end = c;
  }
  file.Close()

  if (!line_start) {
    return LOAD_SUCCESS; // file was empty
  }

  // Decide whether it's time to rebuild the log:
  // - if we're upgrading versions
  // - if it's getting large
  kMinCompactionEntryCount := 100;
  kCompactionRatio := 3;
  if (log_version < kCurrentVersion) {
    this.needs_recompaction_ = true;
  } else if (total_entry_count > kMinCompactionEntryCount &&
             total_entry_count > unique_entry_count * kCompactionRatio) {
    this.needs_recompaction_ = true;
  }

  return LOAD_SUCCESS;
}

// / Lookup a previously-run command by its output path.
func (this *BuildLog) LookupByOutput(path string) *LogEntry {
  i := this.entries_.find(path);
  if (i != this.entries_.end()) {
    return i.second
  }
  return nil
}

// / Serialize an entry into a log file.
func (this *BuildLog) WriteEntry(f *os.File, entry *LogEntry) (bool, error) {
   _,err :=  fmt.Fprintf(f, "%d\t%d\t%" + PRId64 + "\t%s\t%" + PRIx64 + "\n",
          entry.start_time, entry.end_time, entry.mtime,
          entry.output, entry.command_hash)
	return err==nil,err
}

// / Rewrite the known log entries, throwing away old data.
func (this *BuildLog) Recompact(path string, user BuildLogUser, err *string) bool {
  METRIC_RECORD(".ninja_log recompact");

  this.Close();
  temp_path := path + ".recompact";
  f := os.OpenFile(temp_path, "wb");
  if !f{
    *err = strerror(errno);
    return false;
  }

  if (fprintf(f, kFileSignature, kCurrentVersion) < 0) {
    *err = strerror(errno);
    f.Close()
    return false;
  }

  dead_outputs :=[]string{}
  for i := range this.entries_ {
    if (user.IsPathDead(i.first)) {
      dead_outputs.push_back(i.first);
      continue;
    }

    if (!WriteEntry(f, *i.second)) {
      *err = strerror(errno);
      f.Close()
      return false;
    }
  }

  for (size_t i = 0; i < dead_outputs.size(); ++i){
    entries_.erase(dead_outputs[i])
  }

  f.Close()
  if (unlink(path) < 0) {
    *err = strerror(errno);
    return false;
  }

  if (rename(temp_path, path) < 0) {
    *err = strerror(errno);
    return false;
  }

  return true;
}

// / Restat all outputs in the log
func (this *BuildLog) Restat(path string, disk_interface DiskInterface, outputs []string, err *string) bool {
  METRIC_RECORD(".ninja_log restat");
  output_count := len(outputs)

  this.Close();
  temp_path := path + ".restat";
  file, err1 := os.OpenFile(temp_path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644);
  if err1 != nil {
    *err = err1.Error()
    return false;
  }
  defer file.Close()
  _,err1 = fmt.Fprintf(file, kFileSignature, kCurrentVersion)
  if err1 != nil {
	*err = err1.Error()
    return false;
  }
  for _,second := range this.entries_ {
    skip := output_count > 0
    for j := 0; j < output_count; j++ {
      if second.output == outputs[j] {
        skip = false;
        break;
      }
    }
    if (!skip) {
      var mtime TimeStamp = disk_interface.Stat(i.second.output, err);
      if (mtime == -1) {
        return false;
      }
      second.mtime = mtime;
    }
	_,err1 = this.WriteEntry(file, second)
    if err1!=nil {
      *err = err1.Error()
      return false;
    }
  }

  err1 = os.Remove(path)
  if err1!=nil {
    *err = err1.Error()
    return false;
  }
  err1 = os.Rename(temp_path, path)
  if err1!=nil {
    *err = err1.Error()
    return false;
  }

  return true;
}

type Entries map[string]*LogEntry

func (this *BuildLog) entries() Entries { return this.entries_ }

const kFileSignature = "# ninja log v%d\n"
const kOldestSupportedVersion = 7
const kCurrentVersion = 7

// / Should be called before using log_file_. When false is returned, errno
// / will be set.
func (this *BuildLog) OpenForWriteIfNeeded() bool {
	if this.log_file_!=nil ||  this.log_file_path_=="" {
		return true
	}
	file, err := os.OpenFile(this.log_file_path_,os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err!=nil {
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
