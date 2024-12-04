package ninja_rbe

type TimeStamp int64
type LogEntry struct {
	output       string    /* index_output */
	command_hash uint64    /* index_hash,UNIQUE */
	mtime        TimeStamp /* index_hash,UNIQUE */
	start_time   int
	end_time     int
	//
	InstanceId      string /* index_inst */
	CreatedAt       uint64
	LastLastAccess  uint64 /* index_last_access */
	ExpiredDuration uint64 /* index_expired */
}
