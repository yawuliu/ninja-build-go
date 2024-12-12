package main

type RbeLogEntry struct {
	Id int64
	// 本文件的路径
	Output string /* index_output */
	// 本文件命令行的HASH值
	CommandHash string /* index_hash,UNIQUE */
	// 本文件输入文件的HASH
	Mtime string /* index_hash,UNIQUE */
	// 依赖文件信息 -- 外键指向
	// 开始时间
	StartTime string
	// 结束时间
	EndTime string
	// 本文件的HASH值
	OutputHash string
	//
	Instance        string /* index_inst */
	CreatedAt       int64
	LastAccess      int64 /* index_last_access */
	ExpiredDuration int64 /* index_expired */
	Deleted         int64 /* 0 false 1 true */
}
