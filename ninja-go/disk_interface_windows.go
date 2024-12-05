//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modkernel32              = syscall.NewLazyDLL("kernel32.dll")
	procGetVolumeInformation = modkernel32.NewProc("GetVolumeInformationW")
)

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
	var err1 error
	path, err1 = filepath.Abs(path)
	if err1 != nil {
		*err = err1.Error()
		return -1
	}
	if !this.use_cache_ {
		return StatSingleFile(path, this.BuildDir, err)
	}

	dir := DirName(path)
	base := filepath.Base(path)

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
		ci_second = make(map[string]TimeStamp)
		this.cache_[dir_lowercase] = ci_second
		if dir == "" {
			if !StatAllFilesInDir(".", this.BuildDir, &ci_second, err) {
				delete(this.cache_, dir_lowercase)
				return -1
			}
		} else {
			if !StatAllFilesInDir(dir, this.BuildDir, &ci_second, err) {
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

func StatSingleFile(path, prefix string, err *string) TimeStamp {
	mtime := DirHash(path, prefix, err)
	if *err != "" {
		return -1
	}
	return TimeStamp(mtime)
}

// StatAllFilesInDir 遍历目录中的所有文件，并填充时间戳映射
func StatAllFilesInDir(dir, prefix string, stamps *DirCache, err1 *string) bool {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true // 对应于 C++ 中的 ERROR_FILE_NOT_FOUND 或 ERROR_PATH_NOT_FOUND
		} // || os.IsPermission(err)
		*err1 = fmt.Errorf("ReadDir(%s): %w", dir, err).Error()
		return false
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		*err1 = fmt.Errorf("ReadDir(%s): %w", dir, err).Error()
		return false
	}
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			*err1 = fmt.Errorf("Stat(%s): %w", dir, err).Error()
			return false
		}
		lowerName := strings.ToLower(info.Name())
		mtime := DirHash(filepath.Join(dir, info.Name()), prefix, err1)
		if *err1 != "" {
			return false
		}
		(*stamps)[lowerName] = TimeStamp(mtime)
	}
	if err != nil {
		log.Printf("walk error [%v]\n", err)
	}
	return true
}
