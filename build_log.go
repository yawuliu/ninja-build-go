package main

import (
	"bufio"
  "errors"
  "fmt"
	"io"
	"os"
  "strconv"
)

type LineReader struct {
  file_ *os.File
  buf_ string
  buf_end_ int64  // Points one past the last valid byte in |buf_|.

  line_start_ int64
  // Points at the next \n in buf_ after line_start, or NULL.
  line_end_ int64
}
func NewLineReader(file *os.File) * LineReader{
 ret := LineReader{}
  ret.file_ = file
  ret.buf_end_=0
  ret.line_start_=0
  ret.line_end_ = 0
 return &ret
}
// Reads a \n-terminated line from the file passed to the constructor.
  // On return, *line_start points to the beginning of the next line, and
  // *line_end points to the \n at the end of the line. If no newline is seen
  // in a fixed buffer size, *line_end is set to NULL. Returns false on EOF.
 func  (this* LineReader)ReadLine(line_start, line_end *int64)bool {
    if this.line_start_ >= this.buf_end_ || this.line_end_==0 {
      // Buffer empty, refill.
      size_read := fread(this.buf_, 1, sizeof(this.buf_), this.file_);
      if (!size_read) {
        return false
      }
      this.line_start_ = this.buf_;
      this.buf_end_ = this.buf_ + size_read;
    } else {
      // Advance to next line in buffer.
      this.line_start_ = this.line_end_ + 1;
    }

   this.line_end_ = static_cast<char*>(memchr(line_start_, '\n', buf_end_ - line_start_));
    if this.line_end_==0 {
      // No newline. Move rest of data to start of buffer, fill rest.
      already_consumed := this.line_start_ - this.buf_;
      size_rest := (this.buf_end_ - this.buf_) - already_consumed;
      memmove(this.buf_, this.line_start_, size_rest);

       read := fread(this.buf_ + size_rest, 1, sizeof(this.buf_) - size_rest, this.file_);
      this.buf_end_ = this.buf_ + size_rest + read;
      this.line_start_ = this.buf_;
      this.line_end_ = static_cast<char*>(memchr(this.line_start_, '\n', this.buf_end_ - this.line_start_));
    }

    *line_start = this.line_start_;
    *line_end = this.line_end_;
    return true;
  }

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
    second,ok := this.entries_[path]
    var log_entry *LogEntry =nil
    if ok {
      log_entry = second;
    } else {
      log_entry = NewLogEntry(path);
      this.entries_[log_entry.output] = log_entry
    }
    log_entry.command_hash = command_hash;
    log_entry.start_time = start_time;
    log_entry.end_time = end_time;
    log_entry.mtime = mtime;

    if (!this.OpenForWriteIfNeeded()) {
      return false;
    }
    if  this.log_file_!=nil {
      _,err1 := this.WriteEntry(this.log_file_, log_entry)
      if err1!=nil {
		  return false
	  }
      err1 = this.log_file_.Sync()
      if err1!=nil {
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
  file,err1 := os.Open(path);
  if err1!=nil {
    if errors.Is(err1, os.ErrNotExist) {
		return LOAD_NOT_FOUND
	}
    *err = err1.Error()
    return LOAD_ERROR;
  }

  log_version := 0
  unique_entry_count := 0
  total_entry_count := 0

   reader := NewLineReader(file);
   line_start := 0;
  line_end := 0;
  for reader.ReadLine(&line_start, &line_end) {
    if log_version==0 {
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
        file.Close()
        os.RemoveAll(path)
        // Don't report this as a failure. A missing build log will cause
        // us to rebuild the outputs anyway.
        return LOAD_NOT_FOUND;
      }
    }

    // If no newline was found in this chunk, read the next.
    if line_end==0 {
      continue
    }

     kFieldSeparator := '\t';

    start := line_start;
    end := static_cast<char*>(memchr(start, kFieldSeparator, line_end - start));
    if  end == 0 {
      continue
    }
    *end = 0;

    start_time := 0
    end_time := 0;
     var mtime TimeStamp = 0;

    start_time,_ = strconv.Atoi(start)
    start = end + 1;

    end = static_cast<char*>(memchr(start, kFieldSeparator, line_end - start));
    if (!end) {
      continue
    }
    *end = 0;
    end_time,_ = strconv.Atoi(start);
    start = end + 1;

    end = static_cast<char*>(memchr(start, kFieldSeparator, line_end - start));
    if end==0 {
      continue
    }
    *end = 0;
    mtime = strtoll(start, nil, 10);
    start = end + 1;

    end = static_cast<char*>(memchr(start, kFieldSeparator, line_end - start));
    if end==0 {
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
      entry = NewLogEntry(output);
      this.entries_[entry.output] = entry
      unique_entry_count++
    }
    total_entry_count++

    entry.start_time = start_time;
    entry.end_time = end_time;
    entry.mtime = mtime;
    c := *end
    *end = '\000';
    entry.command_hash = strtoull(start, nil, 16);
    *end = c;
  }
  file.Close()

  if line_start==0 {
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
  i,ok := this.entries_[path]
  if ok {
    return i
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
  f,err1 := os.OpenFile(temp_path,  os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644);
  if err1!=nil {
    *err =err1.Error()
    return false;
  }
  _,err1 = fmt.Fprintf(f, kFileSignature, kCurrentVersion)
  if err1!=nil {
    *err = err1.Error()
    f.Close()
    return false;
  }

  dead_outputs :=[]string{}
  for first,second := range this.entries_ {
    if user.IsPathDead(first) {
      dead_outputs = append(dead_outputs, first)
      continue;
    }

    _,err1 := this.WriteEntry(f, second)
    if err1!=nil {
      *err = err1.Error()
      f.Close()
      return false;
    }
  }

  for i := 0; i < len(dead_outputs); i++ {
    delete(this.entries_, dead_outputs[i])
  }

  f.Close()
  err1 = os.RemoveAll(path)
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
      var mtime TimeStamp = disk_interface.Stat(second.output, err);
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
  return rapidhash(command);
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
