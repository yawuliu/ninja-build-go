//go:build linux

package main

import (
	"fmt"
	"os"
)

func (this *RealDiskInterface) Stat(path, prefix string, err1 *string) TimeStamp {
	METRIC_RECORD("node stat")
	var st syscall.Stat_t

	err := syscall.Stat(path, &st)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			// 文件不存在或没有权限时返回0
			return 0
		}
		*err1 = fmt.Errorf("stat(%s): %s", path, err).Error()
		return -1
	}

	if info.IsDir() {
		return HashDirectory(path, prefix)
	} else {
		return HashSingleFile(path, prefix)
	}

	return TimeStamp(mtime)
}
