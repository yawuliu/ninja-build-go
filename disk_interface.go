package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"
	"unsafe"
)

// DiskInterface ---------------------------------------------------------------

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

func TimeStampFromFileTime(filetime syscall.Filetime) TimeStamp {
	// FILETIME 是自 1601 年以来的 100 纳秒间隔
	// Unix 时间戳是自 1970 年以来的秒数
	// 1601 年到 1970 年的秒数差
	const windowsToUnixEpochDelta = int64((1600*365+89)*24*60*60) * 1000000000

	// FILETIME 值是 100 纳秒间隔，转换为纳秒
	nanoseconds := (int64(filetime.HighDateTime) << 32) + int64(filetime.LowDateTime)

	// 将纳秒转换为 Unix 时间戳
	unixNanoseconds := nanoseconds - windowsToUnixEpochDelta

	// 创建 time.Time 结构
	return TimeStamp(unixNanoseconds)
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
	return TimeStampFromFileTime(fileInfo.Sys().(*syscall.Win32FileAttributeData).LastWriteTime)
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
		info, err := os.Stat(filePath)
		if err != nil {
			*err1 = fmt.Errorf("Stat(%s): %w", filePath, err).Error()
			return false
		}

		// 转换文件名为小写
		lowerName := strings.ToLower(file.Name())

		// 将文件的最后写入时间添加到映射中
		stamps[lowerName] = TimeStampFromFileTime(info.Sys().(*syscall.Win32FileAttributeData).LastWriteTime)
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

var (
	modkernel32              = syscall.NewLazyDLL("kernel32.dll")
	procGetVolumeInformation = modkernel32.NewProc("GetVolumeInformationW")
)

// GetVolumeInformation 结构体用于存储 GetVolumeInformationW 函数的信息
type win32VolumeInfo struct {
	VolumeNameBuffer       [260]uint16 // 保留足够的空间
	FileSystemNameBuffer   [260]uint16
	FileSystemNameMax      uint32
	VolumeSerialNumber     uint32
	MaximumComponentLength uint32
	FileSystemFlags        uint32
}

// AreLongPathsEnabled 检查长路径是否启用
func AreLongPathsEnabled() bool {
	var info win32VolumeInfo
	var rootPath string = `C:\`
	var ptrRootPath *uint16 = syscall.StringToUTF16Ptr(rootPath)

	ret, _, err := procGetVolumeInformation.Call(
		uintptr(unsafe.Pointer(ptrRootPath)),
		uintptr(unsafe.Pointer(&info.VolumeNameBuffer[0])),
		uintptr(len(info.VolumeNameBuffer)),
		uintptr(unsafe.Pointer(&info.VolumeSerialNumber)),
		uintptr(unsafe.Pointer(&info.MaximumComponentLength)),
		uintptr(unsafe.Pointer(&info.FileSystemFlags)),
		uintptr(unsafe.Pointer(&info.FileSystemNameBuffer[0])),
		uintptr(len(info.FileSystemNameBuffer)),
	)

	if ret == 0 {
		if err != nil {
			return false
		}
		return false
	}

	return (info.FileSystemFlags & 0x02) != 0 // FILE_SUPPORTS_LONG_NAMES
}

// / stat() a file, returning the mtime, or 0 if missing and -1 on
// / other errors.
func (this *RealDiskInterface) Stat(path string, err *string) TimeStamp {
	METRIC_RECORD("node stat")
	// MSDN: "Naming Files, Paths, and Namespaces"
	// http://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
	if path != "" && !AreLongPathsEnabled() && path[0] != '\\' && len(path) > syscall.MAX_PATH {
		tmp := fmt.Sprintf("Stat(%s): Filename longer than %d characters", path, syscall.MAX_PATH)
		*err = tmp
		return -1
	}
	if !this.use_cache_ {
		return StatSingleFile(path, err)
	}

	dir := DirName(path)
	base := ""
	if len(dir) > 0 {
		base = path[len(dir)+1:]
	} else {
		base = path[0:]
	}

	if base == ".." {
		// StatAllFilesInDir does not report any information for base = "..".
		base = "."
		dir = path
	}

	dir_lowercase := dir
	dir = transformToLower(dir)
	base = transformToLower(base)

	ci_second, ok := this.cache_[dir_lowercase]
	if !ok {
		this.cache_[dir_lowercase] = DirCache{}
		if dir == "" {
			if !StatAllFilesInDir(".", ci_second, err) {
				delete(this.cache_, dir_lowercase)
				return -1
			}
		} else {
			if !StatAllFilesInDir(dir, ci_second, err) {
				delete(this.cache_, dir_lowercase)
				return -1
			}
		}

	}
	di_second, ok := ci_second[base]
	if ok {
		return di_second
	} else {
		return 0
	}
}

// / Create a file, with the specified name and contents
// / Returns true on success, false on failure
func (this *RealDiskInterface) WriteFile(path string, contents string) bool {
	fp, err := os.Open(path)
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
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return NotFound
	}
	status := Okay
	buf, err1 := os.ReadFile(path)
	if err1 != nil {
		*err = err1.Error()
		status = OtherError
	} else {
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
	this.use_cache_ = allow
	if !this.use_cache_ {
		this.cache_ = map[string]DirCache{}
	}
}

// / Whether long paths are enabled.  Only has an effect on Windows.
func (this *RealDiskInterface) AreLongPathsEnabled() bool {
	return this.long_paths_enabled_
}
