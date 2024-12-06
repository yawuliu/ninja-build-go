package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"unicode"
)

// DiskInterface ---------------------------------------------------------------
type DirCache map[string]TimeStamp
type Cache map[string]DirCache
type RealDiskInterface struct {
	DiskInterface
	/// Whether stat information can be cached.
	use_cache_ bool

	/// Whether long paths are enabled.
	long_paths_enabled_ bool

	// TODO: Neither a map nor a hashmap seems ideal here.  If the statcache
	// works out, come up with a better data structure.
	cache_ Cache

	BuildDir string
}

type FileReader interface {
	StatNode(node *Node) (mtime TimeStamp, notExist bool, err error)
	WriteFile(path string, contents string) bool
	MakeDir(path string) bool
	MakeDirs(path string, err *string) bool
	ReadFile(path string, contents *string, err *string) StatusEnum
	RemoveFile(path string) int
	AllowStatCache(allow bool)
}

type DiskInterface interface {
	FileReader
	StatNode(node *Node) (mtime TimeStamp, notExist bool, err error)
	WriteFile(path string, contents string) bool
	MakeDir(path string) bool
	MakeDirs(path string, err *string) bool
	ReadFile(path string, contents, err *string) StatusEnum
	RemoveFile(path string) int
	AllowStatCache(allow bool)
}

// / Create a directory, returning false on failure.
func (this *RealDiskInterface) MakeDir(path string) bool {
	err := os.Mkdir(path, os.ModePerm)
	succ := err == nil
	return succ
}

func DirName(path string) string {
	return filepath.Dir(path)
}

// / Create a directory, returning false on failure.
func (this *RealDiskInterface) MakeDirs(path string, err1 *string) bool {
	dir := DirName(path)
	if dir == "" {
		return true // Reached root; assume it's there.
	}
	_, notExist, err := this.Stat(dir)
	if err != nil {
		*err1 = err.Error()
		Error("%s", err1)
		return false
	}
	if !notExist {
		return true // Exists already; we're done.
	}
	// Directory doesn't exist.  Try creating its parent first.
	success := this.MakeDirs(dir, err1)
	if !success {
		return false
	}
	return this.MakeDir(dir)
}

func NewRealDiskInterface(buildDir string) *RealDiskInterface {
	ret := RealDiskInterface{}
	ret.use_cache_ = false
	ret.long_paths_enabled_ = false
	ret.cache_ = make(map[string]DirCache)
	ret.BuildDir = buildDir
	return &ret
}
func (this *RealDiskInterface) ReleaseRealDiskInterface() {}

// toLowerRune 将 rune 转换为小写
func toLowerRune(r rune) rune {
	if unicode.IsUpper(r) {
		return unicode.ToLower(r)
	}
	return r
}

// transformToLower 将字符串中的所有字符转换为小写
func transformToLower(s string) string {
	var buffer bytes.Buffer
	for _, ch := range s {
		buffer.WriteRune(toLowerRune(ch))
	}
	return buffer.String()
}

// GetVolumeInformation 结构体用于存储 GetVolumeInformationW 函数的信息
type win32VolumeInfo struct {
	VolumeNameBuffer       [260]uint16 // 保留足够的空间
	FileSystemNameBuffer   [260]uint16
	FileSystemNameMax      uint32
	VolumeSerialNumber     uint32
	MaximumComponentLength uint32
	FileSystemFlags        uint32
}

// / Create a file, with the specified name and contents
// / Returns true on success, false on failure
func (this *RealDiskInterface) WriteFile(path string, contents string) bool {
	fp, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0664)
	if err != nil {
		Error("WriteFile(%s): Unable to create file. %v", path, err)
		return false
	}

	_, err = io.WriteString(fp, contents)
	if err != nil {
		Error("WriteFile(%s): Unable to write to the file. %v", path, err)
		fp.Close()
		return false
	}

	err = fp.Close()
	if err != nil {
		Error("WriteFile(%s): Unable to close the file. %v", path, err)
		return false
	}

	return true
}

type StatusEnum int8

const (
	Okay       StatusEnum = 0
	NotFound   StatusEnum = 1
	OtherError StatusEnum = 2
)

func (this *RealDiskInterface) ReadFile(path string, contents, err *string) StatusEnum {
	if _, err1 := os.Stat(path); errors.Is(err1, os.ErrNotExist) {
		*err = err1.Error()
		return NotFound
	}
	status := Okay
	buf, err1 := os.ReadFile(path)
	if err1 != nil {
		*err = err1.Error()
		status = OtherError
	} else {
		*contents = string(buf) + "\x00"
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
	if runtime.GOOS == "windows" {
		this.use_cache_ = allow
		if !this.use_cache_ {
			this.cache_ = map[string]DirCache{}
		}
	}
}

// / Whether long paths are enabled.  Only has an effect on Windows.
func (this *RealDiskInterface) AreLongPathsEnabled() bool {
	return this.long_paths_enabled_
}
