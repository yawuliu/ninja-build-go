//go:build linux

package main

import (
	"fmt"
	"os"
	"syscall"
)

func (this *RealDiskInterface) Stat(path string, err1 *string) TimeStamp {
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

	// 将最后修改时间转换为纳秒
	mtime := st.Mtim.Sec*1e9 + st.Mtim.Nsec

	// 有些系统（如Flatpak）可能将mtime设置为0，这里我们返回1以避免与不存在的返回值冲突
	if mtime == 0 {
		return 1
	}

	return TimeStamp(mtime)
}
