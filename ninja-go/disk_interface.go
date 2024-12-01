package ninja_go

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
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
}

type FileReader interface {
	Stat(path string, err *string) TimeStamp
	WriteFile(path string, contents string) bool
	MakeDir(path string) bool
	MakeDirs(path string, err *string) bool
	ReadFile(path string, contents *string, err *string) StatusEnum
	RemoveFile(path string) int
	AllowStatCache(allow bool)
}

type DiskInterface interface {
	FileReader
	Stat(path string, err *string) TimeStamp
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
	mtime := this.Stat(dir, err1)
	if mtime < 0 {
		Error("%s", err1)
		return false
	}
	if mtime > 0 {
		return true // Exists already; we're done.
	}
	// Directory doesn't exist.  Try creating its parent first.
	success := this.MakeDirs(dir, err1)
	if !success {
		return false
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

// TimeStampFromFileTime 将 FILETIME 结构转换为 Unix 时间戳
func TimeStampFromFileTime(filetime time.Time) TimeStamp {
	// FILETIME 是自 1601 年 1 月 1 日以来的 100 纳秒间隔
	// Unix 时间戳是自 1970 年 1 月 1 日以来的秒数
	// 将 FILETIME 转换为 Unix 时间戳，首先需要将纳秒转换为秒，并调整纪元差异

	// 1601 年到 1970 年之间的秒数差
	const epochDifference = int64((400 * 365 * 24 * 60 * 60) + (97 * 24 * 60 * 60)) // 400 年加上 97 个闰年
	const nanosecondsPerSecond = int64(1e9)

	// FILETIME 表示的纳秒数
	filetimeNanoseconds := uint64(filetime.UnixNano())

	// 转换为 Unix 时间戳
	timestamp := uint64((filetimeNanoseconds - uint64(epochDifference)*uint64(nanosecondsPerSecond)) / uint64(nanosecondsPerSecond))

	return TimeStamp(timestamp)
}

func StatSingleFile(path string, err *string) TimeStamp {
	fileInfo, err1 := os.Stat(path)
	if err1 != nil {
		if os.IsNotExist(err1) || os.IsPermission(err1) {
			return 0 // 对应于 C++ 中的 ERROR_FILE_NOT_FOUND 或 ERROR_PATH_NOT_FOUND
		}
		*err = fmt.Errorf("GetFileAttributesEx(%s): %v", path, err1).Error()
		return -1
	}
	return TimeStampFromFileTime(fileInfo.ModTime())
}

// StatAllFilesInDir 遍历目录中的所有文件，并填充时间戳映射
func StatAllFilesInDir(dir string, stamps map[string]TimeStamp, err1 *string) bool {
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return true // 对应于 C++ 中的 ERROR_FILE_NOT_FOUND 或 ERROR_PATH_NOT_FOUND
		}
		*err1 = fmt.Errorf("ReadDir(%s): %w", dir, err).Error()
		return false
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if file.Name() == ".." {
			// Skip ".." as it is not a file
			continue
		}

		filePath := filepath.Join(dir, file.Name())
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			*err1 = fmt.Errorf("Stat(%s): %w", filePath, err).Error()
			return false
		}

		// 转换文件名为小写
		lowerName := strings.ToLower(file.Name())

		// 将文件的最后写入时间添加到映射中
		stamps[lowerName] = TimeStampFromFileTime(fileInfo.ModTime())
	}

	return true
}

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
	fp, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0664)
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
	this.use_cache_ = allow
	if !this.use_cache_ {
		this.cache_ = map[string]DirCache{}
	}
}

// / Whether long paths are enabled.  Only has an effect on Windows.
func (this *RealDiskInterface) AreLongPathsEnabled() bool {
	return this.long_paths_enabled_
}
