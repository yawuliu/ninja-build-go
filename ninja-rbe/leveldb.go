package ninja_rbe

import (
	"zombiezen.com/go/sqlite"
)

var conn *sqlite.Conn = nil

func OpenDb(dbPath string) (err error) {
	// Open an in-memory database.
	conn, err = sqlite.OpenConn(":memory:", sqlite.OpenReadWrite)
	if err != nil {
		return err
	}
	return
}

func CloseDb() (err error) {
	err = conn.Close()
	return
}

func InsertLogEntry(key string, value string) error {
	// Execute a query.
	stmt, err := conn.Prepare("INSERT INTO log_entry (a, b, c) VALUES ($a, $b, $c);")
	if err != nil {
		return err
	}
	stmt.SetText("$a", "col_a")
	stmt.SetText("$b", "col_b")
	stmt.SetText("$badparam", "notaval")
	stmt.SetText("$c", "col_c")
	_, err = stmt.Step()
	if err != nil {
		return err
	}
	err = stmt.Finalize()
	if err != nil {
		return err
	}
	return nil
}
