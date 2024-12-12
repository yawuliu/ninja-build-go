package model

import "gorm.io/plugin/soft_delete"

type DepsEntry struct {
	ID int64 `gorm:"primarykey"`
	// 文件路径
	FilePath string
	// 本文件的HASH值
	FileHash string
	// 被谁依赖的节点的ID
	PID int64 `json:"pid" gorm:"index:idx_pid"`
	/* 0 false 1 true */
	Deleted soft_delete.DeletedAt `gorm:"softDelete:flag;default:0"`
}

func (DepsEntry) TableName() string {
	return "deps_entry"
}
