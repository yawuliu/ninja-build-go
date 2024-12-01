//go:build windows

package ninja_go

import (
	"fmt"
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
