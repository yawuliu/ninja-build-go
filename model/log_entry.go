package model

import "gorm.io/plugin/soft_delete"

type RbeLogEntry struct {
	ID         int64  `json:"id" gorm:"default:0"`
	ParamsHash string `json:"params_hash" gorm:"index:idx_params_hash,unique"`
	// 本文件的路径  /* index_output */
	Output string `json:"output" gorm:"index:idx_output"`
	// 本文件命令行的HASH值 /* index_hash,UNIQUE */
	CommandHash string `json:"command_hash" gorm:"index:idx_command_hash,unique"`
	// 本文件输入文件的HASH /* index_hash,UNIQUE */
	InputHash string `json:"inputHash" gorm:"index:idx_input_hash"`
	//	// 依赖文件信息 -- 外键指向
	// 开始时间
	StartTime string
	// 结束时间
	EndTime string
	// 本文件的HASH值
	OutputHash string
	//
	Deps []*DepsEntry `json:"deps" gorm:"ForeignKey:PID;AssociationForeignKey:ID"`
	//
	Instance        string /* index_inst */
	CreatedAt       int64
	LastAccess      int64 /* index_last_access */
	ExpiredDuration int64 /* index_expired */
	/* 0 false 1 true */
	Deleted soft_delete.DeletedAt `gorm:"softDelete:flag;default:0"`
}

func (RbeLogEntry) TableName() string {
	return "log_entry"
}
