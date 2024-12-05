package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

var (
	conn                  *sqlite.Conn = nil
	stmtLog               *sqlite.Stmt = nil
	stmtFindExistByKey    *sqlite.Stmt = nil
	stmtFindRowByKey      *sqlite.Stmt = nil
	stmtUpdateAccessByKey *sqlite.Stmt = nil
	stmtFindExpired       *sqlite.Stmt = nil
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
		stmt, err := conn.Prepare("CREATE TABLE IF NOT EXISTS log_entry (`id` INTEGER PRIMARY KEY, " +
			"`output` TEXT, `command_hash` TEXT, `mtime` TEXT, `start_time` TEXT, `end_time` TEXT," +
			"`instance` TEXT, `created_at` INTEGER, `last_access` INTEGER, `expired_duration` INTEGER," +
			"`deleted` INTEGER," +
			" UNIQUE (`command_hash`, `output`,  `instance`, `deleted`) ON CONFLICT REPLACE " +
			");")
		if err != nil {
			return err
		}
		if _, err := stmt.Step(); err != nil {
			return err
		}
	}
	stmtLog, err = conn.Prepare("INSERT INTO log_entry (`output`, `command_hash`, `mtime`, `start_time`, `end_time`, " +
		"`instance`, `created_at`, `last_access`, `expired_duration`, `deleted`) VALUES" +
		" ($output, $command_hash, $mtime , $start_time, $end_time, $instance, $created_at, $last_access, $expired_duration, $deleted);")
	if err != nil {
		return err
	}
	stmtFindExistByKey, err = conn.Prepare("SELECT count(*) FROM log_entry WHERE `command_hash` = $command_hash AND `mtime` = $mtime AND `deleted` = 0;")
	if err != nil {
		return err
	}
	stmtFindRowByKey, err = conn.Prepare("SELECT * FROM log_entry " +
		"WHERE `command_hash` = $command_hash AND `instance` = $instance " +
		"AND `output` =$output AND `deleted` = 0 ORDER BY `mtime` DESC LIMIT 1;")
	if err != nil {
		return err
	}
	stmtUpdateAccessByKey, err = conn.Prepare("UPDATE log_entry SET `last_access` = $last_access WHERE `id` = $id;")
	if err != nil {
		return err
	}
	stmtFindExpired, err = conn.Prepare("SELECT `id`, `command_hash`, `mtime`, `last_access` + `expired_duration` as `expired` FROM log_entry " +
		"WHERE `deleted` = 0 AND  expired  < $now ORDER BY id DESC LIMIT $limit;")
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
	err = conn.Close()
	return
}

func InsertLogEntry(output, commandHash, startTime, endTime, mtime, instance, expired_duration_str string) error {
	defer stmtLog.Reset()
	now := time.Now()
	created_at := now.Unix()
	last_access := created_at
	stmtLog.SetText("$output", output)
	stmtLog.SetText("$command_hash", commandHash)
	stmtLog.SetText("$start_time", startTime)
	stmtLog.SetText("$end_time", endTime)
	stmtLog.SetText("$mtime", mtime)
	stmtLog.SetText("$instance", instance)
	expired_duration := 5 * time.Minute
	if expired_duration_str != "" {
		expired_duration, _ = time.ParseDuration(expired_duration_str)
	}
	stmtLog.SetInt64("$expired_duration", int64(expired_duration))
	stmtLog.SetInt64("$created_at", created_at)
	stmtLog.SetInt64("$last_access", last_access)
	stmtLog.SetInt64("$deleted", 0)
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
func UpdateFileAccess(id int64) error {
	defer stmtUpdateAccessByKey.Reset()
	stmtUpdateAccessByKey.SetInt64("$id", id)
	now := time.Now()
	stmtUpdateAccessByKey.SetInt64("$last_access", now.Unix())
	_, err := stmtUpdateAccessByKey.Step()
	if err != nil {
		return err
	}
	return nil
}

func FindCommandHashLastMtime(output, instance, command_hash string) (*RbeLogEntry, error) {
	defer stmtFindRowByKey.Reset()
	stmtFindRowByKey.SetText("$command_hash", command_hash)
	stmtFindRowByKey.SetText("$output", output)
	stmtFindRowByKey.SetText("$instance", instance)
	for {
		hasRow, err := stmtFindRowByKey.Step()
		if err != nil {
			return nil, err
		}
		if !hasRow {
			break
		}
		id := stmtFindRowByKey.GetInt64("id")
		// output := stmtFindRowByKey.GetText("output")
		// command_hash := stmtFindRowByKey.GetText("command_hash")
		mtime := stmtFindRowByKey.GetText("mtime")
		start_time := stmtFindRowByKey.GetText("start_time")
		end_time := stmtFindRowByKey.GetText("end_time")
		// instance := stmtFindRowByKey.GetText("instance")
		created_at := stmtFindRowByKey.GetInt64("created_at")
		last_access := stmtFindRowByKey.GetInt64("last_access")
		expired_duration := stmtFindRowByKey.GetInt64("expired_duration")
		err = UpdateFileAccess(id)
		if err != nil {
			return nil, err
		}
		return &RbeLogEntry{Output: output, CommandHash: command_hash, Mtime: mtime,
			StartTime: start_time, EndTime: end_time, Instance: instance,
			CreatedAt: created_at, LastAccess: last_access, ExpiredDuration: expired_duration,
		}, nil
	}
	return nil, os.ErrNotExist
}

func FindExpiredLogWithLimit(limit int64) ([]*RbeLogEntry, error) {
	defer stmtFindExpired.Reset()
	now := time.Now().Unix()
	stmtFindExpired.SetInt64("$now", now)
	stmtFindExpired.SetInt64("$limit", limit)
	var ret []*RbeLogEntry
	for {
		hasRow, err := stmtFindExpired.Step()
		if err != nil {
			return nil, err
		}
		if !hasRow {
			break
		}
		id := stmtFindExpired.GetInt64("id")
		commandHash := stmtFindExpired.GetText("command_hash")
		mTime := stmtFindExpired.GetText("mtime")
		ret = append(ret, &RbeLogEntry{Id: id, CommandHash: commandHash, Mtime: mTime})
	}
	return ret, nil
}

func UpdateExpiredCleanResult(successCleans []int64) error {
	if len(successCleans) == 0 {
		return nil
	}
	var ids []string
	for _, successClean := range successCleans {
		if successClean > 0 {
			ids = append(ids, strconv.FormatInt(successClean, 10))
		}
	}
	query := fmt.Sprintf("UPDATE log_entry SET `deleted` = 1 WHERE `id` in (%s);", strings.Join(ids, ","))
	err := sqlitex.ExecuteTransient(conn, query, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			return nil
		},
	})
	if err != nil {
		return err
	}
	return nil
}
