package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DiskInterface ---------------------------------------------------------------

// / Create a directory, returning false on failure.
func (this *RealDiskInterface) MakeDir(path string) (bool, error) {
	err := os.Mkdir(path, os.ModePerm)
	succ := err == nil
	return succ, err
}

func DirName(path string) string {
	return filepath.Dir(path)
}

// / Create a directory, returning false on failure.
func (this *RealDiskInterface) MakeDirs(path string) (bool, error) {
	dir := DirName(path)
	if dir == "" {
		return true, nil // Reached root; assume it's there.
	}
	err := ""
	mtime := this.Stat(dir, &err)
	if mtime < 0 {
		Error("%s", err)
		return false, errors.New(err)
	}
	if mtime > 0 {
		return true, nil // Exists already; we're done.
	}
	// Directory doesn't exist.  Try creating its parent first.
	success, er := this.MakeDirs(dir)
	if !success {
		return false, er
	}
	return this.MakeDir(dir)
}

func NewRealDiskInterface() *RealDiskInterface {
	ret := RealDiskInterface{}
	ret.use_cache_ = false
	ret.long_paths_enabled_ = false
	return &ret
}
func (this *RealDiskInterface) ReleaseRealDiskInterface() {}

func  TimeStampFromFileTime(filetime *FILETIME) TimeStamp{
	// FILETIME is in 100-nanosecond increments since the Windows epoch.
	// We don't much care about epoch correctness but we do want the
	// resulting value to fit in a 64-bit integer.
	mtime := ((uint64)filetime.dwHighDateTime << 32) | ((uint64_t)filetime.dwLowDateTime)
	// 1600 epoch -> 2000 epoch (subtract 400 years).
	return TimeStamp(mtime) - 12622770400LL * (1000000000LL / 100)
}

func StatSingleFile(path string,  err *string) TimeStamp {
	WIN32_FILE_ATTRIBUTE_DATA attrs;
	if (!GetFileAttributesExA(path.c_str(), GetFileExInfoStandard, &attrs)) {
		 win_err := GetLastError();
		if (win_err == ERROR_FILE_NOT_FOUND || win_err == ERROR_PATH_NOT_FOUND) {
			return 0
		}
		*err = "GetFileAttributesEx(" + path + "): " + GetLastErrorString();
		return -1;
	}
	return TimeStampFromFileTime(attrs.ftLastWriteTime);
}

// / stat() a file, returning the mtime, or 0 if missing and -1 on
// / other errors.
func (this *RealDiskInterface) Stat(path string, err *string) TimeStamp {
	METRIC_RECORD("node stat");
  // MSDN: "Naming Files, Paths, and Namespaces"
  // http://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
  if !path.empty() && !AreLongPathsEnabled() && path[0] != '\\' && path.size() > MAX_PATH {
    fmt.Printf("Stat(%s): Filename longer than %d characters", path, MAX_PATH)
    *err = err_stream.str();
    return -1;
  }
  if !this.use_cache_ {
	  return StatSingleFile(path, err)
  }

  dir := DirName(path);
  base := (path.substr(dir.size() ? dir.size() + 1 : 0));
  if (base == "..") {
    // StatAllFilesInDir does not report any information for base = "..".
    base = ".";
    dir = path;
  }

  dir_lowercase := dir
  transform(dir.begin(), dir.end(), dir_lowercase.begin(), ::tolower);
  transform(base.begin(), base.end(), base.begin(), ::tolower);

  ci := this.cache_.find(dir_lowercase);
  if (ci == this.cache_.end()) {
    ci = this.cache_.insert(make_pair(dir_lowercase, DirCache())).first;
    if (!StatAllFilesInDir(dir.empty() ? "." : dir, &ci.second, err)) {
      cache_.erase(ci);
      return -1;
    }
  }
  di := ci.second.find(base);
  if  di != ci.second.end() {
	  return di.second
  }  else{
	  return  0
  }
}

// / Create a file, with the specified name and contents
// / Returns true on success, false on failure
func (this *RealDiskInterface) WriteFile(path string, contents string) bool        {
	fp,err := os.Open(path);
	if err!=nil {
		Error("WriteFile(%s): Unable to create file. %v",  path, err);
		return false;
	}

	_,err = io.WriteString(fp, contents)
	if err !=nil  {
		Error("WriteFile(%s): Unable to write to the file. %v", path, err)
		fp.Close()
		return false
	}

	err = fp.Close()
	if err!=nil {
		Error("WriteFile(%s): Unable to close the file. %v", path, err);
		return false;
	}

	return true;
}

type StatusEnum int8
const (
	Okay StatusEnum = 0
	NotFound StatusEnum = 1
	OtherError  StatusEnum = 2
)

func (this *RealDiskInterface) ReadFile(path string, contents, err *string) StatusEnum {
	if _,err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return NotFound
	}
	status := Okay
	buf,err1 := os.ReadFile(path)
	if err1!=nil {
		*err = err1.Error()
		status = OtherError
	}else{
		*contents = string(buf)
	}
	return status
}

// / Remove the file named @a path. It behaves like 'rm -f path' so no errors
// / are reported if it does not exists.
// / @returns 0 if the file has been removed,
// /          1 if the file does not exist, and
// /          -1 if an error occurs.
func (this *RealDiskInterface) RemoveFile(path string) int {
	err := os.RemoveAll(path)
	if err != nil {
		return -1
	}
	return 0
}

// / Whether stat information can be cached.  Only has an effect on Windows.
func (this *RealDiskInterface) AllowStatCache(allow bool) {
	this.use_cache_ = allow;
	if !this.use_cache_ {
		this.cache_.clear()
	}
}

// / Whether long paths are enabled.  Only has an effect on Windows.
func (this *RealDiskInterface) AreLongPathsEnabled() bool {
	return this.long_paths_enabled_
}
