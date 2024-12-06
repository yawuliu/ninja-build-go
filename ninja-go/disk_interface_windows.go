//go:build windows

package main

import (
	"fmt"
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

// // Node in_edge所有文件的Hash, path_也可能存在于远程，in_edge中的文件也可能存在于远程
func (this *RealDiskInterface) StatNode(node *Node) (mtime TimeStamp, notExist bool, err error) {
	if node.in_edge() == nil {
		return this.Stat(node.path())
	} else {
		METRIC_RECORD("node stat")
		// MSDN: "Naming Files, Paths, and Namespaces"
		// http://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
		path := node.path()
		if path != "" && !AreLongPathsEnabled() && path[0] != '\\' && len(path) > syscall.MAX_PATH {
			return -1, true, fmt.Errorf("Stat(%s): Filename longer than %d characters", path, syscall.MAX_PATH)
		}
		path, err = filepath.Abs(path)
		if err != nil {
			return -1, true, err
		}
		_, err = os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) || os.IsPermission(err) {
				return 0, true, nil // 对应于 C++ 中的 ERROR_FILE_NOT_FOUND 或 ERROR_PATH_NOT_FOUND
			}
			return -1, true, err
		}
		return NodesHash(node.in_edge().inputs_, this.BuildDir)
	}

}

// / stat() a file, returning the mtime, or 0 if missing and -1 on
// / other errors.
func (this *RealDiskInterface) Stat(path string) (mtime TimeStamp, notExist bool, err error) {
	METRIC_RECORD("node stat")
	// MSDN: "Naming Files, Paths, and Namespaces"
	// http://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
	if path != "" && !AreLongPathsEnabled() && path[0] != '\\' && len(path) > syscall.MAX_PATH {
		return -1, true, fmt.Errorf("Stat(%s): Filename longer than %d characters", path, syscall.MAX_PATH)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return -1, true, err
	}
	if !this.use_cache_ {
		return StatSingleFile(path, this.BuildDir)
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
			if ok, err := StatAllFilesInDir(".", this.BuildDir, &ci_second); !ok {
				delete(this.cache_, dir_lowercase)
				return -1, true, err
			}
		} else {
			if ok, err := StatAllFilesInDir(dir, this.BuildDir, &ci_second); !ok {
				delete(this.cache_, dir_lowercase)
				return -1, true, err
			}
		}

	}
	di_second, ok := ci_second[base]
	if ok {
		return di_second, false, nil
	} else {
		return 0, true, nil
	}
}

func StatSingleFile(path, prefix string) (mtime TimeStamp, notExist bool, err error) {
	mtime, notExist, err = DirHash(path, prefix)
	if err != nil {
		return TimeStamp(-1), notExist, err
	}
	return mtime, notExist, nil
}

// StatAllFilesInDir 遍历目录中的所有文件，并填充时间戳映射
func StatAllFilesInDir(dir, prefix string, stamps *DirCache) (bool, error) {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // 对应于 C++ 中的 ERROR_FILE_NOT_FOUND 或 ERROR_PATH_NOT_FOUND
		} // || os.IsPermission(err)
		return false, err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return false, err
		}
		lowerName := strings.ToLower(info.Name())
		mtime, _, err := DirHash(filepath.Join(dir, info.Name()), prefix)
		if err != nil {
			return false, err
		}
		(*stamps)[lowerName] = TimeStamp(mtime)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
