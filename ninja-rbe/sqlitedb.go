package main

import (
	"errors"
	"os"
	"time"
	"zombiezen.com/go/sqlite"
)

var (
	conn                  *sqlite.Conn = nil
	stmtLog               *sqlite.Stmt = nil
	stmtFindExistByKey    *sqlite.Stmt = nil
	stmtFindRowByKey      *sqlite.Stmt = nil
	stmtUpdateAccessByKey *sqlite.Stmt = nil
)

func OpenDb(dbPath string) (err error) {
	needCreateTable := false
	if _, err1 := os.Stat(dbPath); errors.Is(err1, os.ErrNotExist) {
		needCreateTable = true
	} else if err1 != nil {
		err = err1
		return
	}
	flag := sqlite.OpenReadWrite
	if needCreateTable {
		flag |= sqlite.OpenCreate
	}
	conn, err = sqlite.OpenConn(dbPath, flag)
	if err != nil {
		return err
	}
	if needCreateTable {
		stmt, err := conn.Prepare("CREATE TABLE IF NOT EXISTS log_entry (`output` TEXT, `command_hash` TEXT, `mtime` TEXT, `start_time` TEXT, `end_time` TEXT," +
			"`instance_id` TEXT, `created_at` INTEGER, `last_access` INTEGER, `expired_duration` INTEGER);")
		if err != nil {
			return err
		}
		if _, err := stmt.Step(); err != nil {
			return err
		}
	}
	stmtLog, err = conn.Prepare("INSERT INTO log_entry (`output`, `command_hash`, `mtime`, `start_time`, `end_time`, " +
		"`instance_id`, `created_at`, `last_access`, `expired_duration`) VALUES" +
		" ($output, $command_hash, $mtime , $start_time, $end_time, $instance_id, $created_at, $last_access, $expired_duration);")
	if err != nil {
		return err
	}
	stmtFindExistByKey, err = conn.Prepare("SELECT count(*) FROM log_entry WHERE `command_hash` = $command_hash AND `mtime` = $mtime;")
	if err != nil {
		return err
	}
	stmtFindRowByKey, err = conn.Prepare("SELECT * FROM log_entry WHERE `command_hash` = $command_hash AND `mtime` = $mtime;")
	if err != nil {
		return err
	}
	stmtUpdateAccessByKey, err = conn.Prepare("UPDATE log_entry SET `last_access` = $last_access WHERE `command_hash` = $command_hash AND `mtime` = $mtime;")
	if err != nil {
		return err
	}
	return
}

func CloseDb() (err error) {
	//err = stmtLog.Finalize()
	//if err != nil {
	//	return
	//}
	//err = stmtFindExistByKey.Finalize()
	//if err != nil {
	//	return
	//}
	//err = stmtFindRowByKey.Finalize()
	//if err != nil {
	//	return
	//}
	err = conn.Close()
	return
}

func InsertLogEntry(output, commandHash, startTime, endTime, mtime, instance_id, expired_duration_str string) error {
	defer stmtLog.Reset()
	now := time.Now()
	created_at := now.Unix()
	last_access := created_at
	stmtLog.SetText("$output", output)
	stmtLog.SetText("$command_hash", commandHash)
	stmtLog.SetText("$start_time", startTime)
	stmtLog.SetText("$end_time", endTime)
	stmtLog.SetText("$mtime", mtime)
	stmtLog.SetText("$instance_id", instance_id)
	expired_duration := 5 * time.Minute
	if expired_duration_str != "" {
		expired_duration, _ = time.ParseDuration(expired_duration_str)
	}
	stmtLog.SetInt64("$expired_duration", int64(expired_duration))
	stmtLog.SetInt64("$created_at", created_at)
	stmtLog.SetInt64("$last_access", last_access)
	_, err := stmtLog.Step()
	if err != nil {
		return err
	}
	return nil
}

func CheckCommandHashAndMtimeExist(command_hash string, mtime string) (bool, error) {
	defer stmtFindExistByKey.Reset()
	stmtFindExistByKey.SetText("$command_hash", command_hash)
	stmtFindExistByKey.SetText("$mtime", mtime)
	if hasRow, err := stmtFindExistByKey.Step(); err != nil {
		return false, err
	} else if !hasRow {
		return false, nil
	}
	cnt := stmtFindExistByKey.ColumnInt(0)
	return cnt > 0, nil
}

// talend api Tester
func UpdateFileAccess(command_hash string, mtime string) error {
	defer stmtUpdateAccessByKey.Reset()
	stmtUpdateAccessByKey.SetText("$command_hash", command_hash)
	stmtUpdateAccessByKey.SetText("$mtime", mtime)
	now := time.Now()
	stmtUpdateAccessByKey.SetInt64("$last_access", now.Unix())
	_, err := stmtUpdateAccessByKey.Step()
	if err != nil {
		return err
	}
	return nil
}

func FindCommandHashAndMtime(command_hash string, mtime string) (*LogEntry, error) {
	defer stmtFindRowByKey.Reset()
	stmtFindRowByKey.SetText("$command_hash", command_hash)
	stmtFindRowByKey.SetText("$mtime", mtime)
	for {
		hasRow, err := stmtFindRowByKey.Step()
		if err != nil {
			return nil, err
		}
		if !hasRow {
			break
		}
		output := stmtFindRowByKey.GetText("output")
		command_hash := stmtFindRowByKey.GetText("command_hash")
		mtime := stmtFindRowByKey.GetText("mtime")
		start_time := stmtFindRowByKey.GetText("start_time")
		end_time := stmtFindRowByKey.GetText("end_time")
		instance_id := stmtFindRowByKey.GetText("instance_id")
		created_at := stmtFindRowByKey.GetInt64("created_at")
		last_access := stmtFindRowByKey.GetInt64("last_access")
		expired_duration := stmtFindRowByKey.GetInt64("expired_duration")
		err = UpdateFileAccess(command_hash, mtime)
		if err != nil {
			return nil, err
		}
		return &LogEntry{Output: output, CommandHash: command_hash, Mtime: mtime,
			StartTime: start_time, EndTime: end_time, InstanceId: instance_id,
			CreatedAt: created_at, LastAccess: last_access, ExpiredDuration: expired_duration,
		}, nil
	}
	return nil, os.ErrNotExist
}
