package main

type TimeStamp int64
type LogEntry struct {
	Output      string /* index_output */
	CommandHash string /* index_hash,UNIQUE */
	Mtime       string /* index_hash,UNIQUE */
	StartTime   string
	EndTime     string
	//
	InstanceId      string /* index_inst */
	CreatedAt       int64
	LastAccess      int64 /* index_last_access */
	ExpiredDuration int64 /* index_expired */
}
