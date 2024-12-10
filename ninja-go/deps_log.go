package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"zombiezen.com/go/sqlite"
)

// The version is stored as 4 bytes after the signature and also serves as a
// byte order mark. Signature and version combined are 16 bytes long.
const kFileSignature_DepsLog = "# ninjadeps %d\n"
const kCurrentVersion_DepsLog = 4

const kMaxRecordSize = (1 << 19) - 1

type DepsLog struct {
	needs_recompaction_ bool
	file_               *sqlite.Conn
	stmtInsert          *sqlite.Stmt
	stmtFileTell        *sqlite.Stmt
	stmtLoad            *sqlite.Stmt
	file_path_          string

	/// Maps id . Node.
	nodes_ []*Node
	/// Maps id . deps of that id.
	deps_ []*Deps
}

type Deps struct {
	mtime      TimeStamp
	node_count int
	nodes      []*Node
}

func NewDeps(mtime int64, node_count int) *Deps {
	ret := Deps{}
	ret.mtime = TimeStamp(mtime)
	ret.node_count = node_count
	ret.nodes = make([]*Node, node_count)
	return &ret
}

func (this *Deps) ReleaseDeps() {
	this.nodes = []*Node{}
}

func NewDepsLog() *DepsLog {
	ret := DepsLog{}
	ret.needs_recompaction_ = false
	ret.file_ = nil
	return &ret
}
func (this *DepsLog) ReleaseDepsLog() {
	this.Close()
}

func (this *DepsLog) RecordDeps(node *Node, mtime TimeStamp, nodes []*Node, err *string) bool {
	// Track whether there's any new data to be recorded.
	made_change := false
	node_count := len(nodes)

	// Assign ids to all nodes that are missing one.
	if node.id() < 0 {
		if !this.RecordId(node, mtime, -1) {
			return false
		}
		made_change = true
	}
	for i := 0; i < node_count; i++ {
		if nodes[i].id() < 0 {
			if !this.RecordId(nodes[i], 0, int64(node.id())) {
				return false
			}
			made_change = true
		}
	}

	// See if the new data is different than the existing data, if any.
	if !made_change {
		deps := this.GetDeps(node)
		if deps == nil || deps.mtime != mtime || deps.node_count != node_count {
			made_change = true
		} else {
			for i := 0; i < node_count; i++ {
				if deps.nodes[i] != nodes[i] {
					made_change = true
					break
				}
			}
		}
	}

	// Don't write anything if there's no new info.
	if !made_change {
		return true
	}
	if !this.OpenForWriteIfNeeded() {
		return false
	}
	// Update in-memory representation.
	deps := NewDeps(int64(mtime), node_count)
	for i := 0; i < node_count; i++ {
		deps.nodes[i] = nodes[i]
	}
	this.UpdateDeps(node.id(), deps)

	return true
}

func (this *DepsLog) Close() {
	this.OpenForWriteIfNeeded() // create the file even if nothing has been recorded
	if this.file_ != nil {
		this.file_.Close()
	}
	this.file_ = nil
}
func binaryLittleEndianToIntSlice(buf []byte) []int {
	slice := make([]int, len(buf)/4)
	for i, j := 0, 0; i < len(buf); i, j = i+4, j+1 {
		slice[j] = int(binary.LittleEndian.Uint32(buf[i : i+4]))
	}
	return slice
}

func (this *DepsLog) Load(path string, state *State, err1 *string) LoadStatus {
	this.file_path_ = path
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LOAD_NOT_FOUND
		}
		*err1 = err.Error()
		return LOAD_ERROR
	}
	if !this.OpenForWriteIfNeeded() {
		*err1 = "OpenForWriteIfNeeded failed"
		return LOAD_ERROR
	}
	this.stmtLoad.Reset()
	type DepLogEntry struct {
		Id    int64
		Mtime int64
		Path  string
		Pid   int64
	}
	var entries []*DepLogEntry
	for {
		hasRow, err := this.stmtLoad.Step()
		if err != nil {
			*err1 = err.Error()
			return LOAD_ERROR
		}
		if !hasRow {
			break
		}
		id := this.stmtLoad.GetInt64("id")
		mtime := this.stmtLoad.GetInt64("mtime")
		path := this.stmtLoad.GetText("path")
		pid := this.stmtLoad.GetInt64("pid")
		entries = append(entries, &DepLogEntry{Id: id, Mtime: mtime, Path: path, Pid: pid})
	}
	total := int64(len(entries))
	this.nodes_ = make([]*Node, total)
	for _, entry := range entries {
		if entry.Id >= total {
			*err1 = "OUT RANGE"
			return LOAD_ERROR
		}
		node := state.GetNode(entry.Path, 0)
		if node == nil {
			*err1 = fmt.Sprintf("GetNode for %s return empty.", entry.Path)
			return LOAD_ERROR
		}
		node.id_ = int(entry.Id)
		node.mtime_ = TimeStamp(entry.Mtime)
		this.nodes_[entry.Id] = node
		//NewNodeWithMtimeAndId(entry.Path, TimeStamp(entry.Mtime), entry.Id, 0)
	}
	this.deps_ = make([]*Deps, total)
	for _, entry := range entries {
		if entry.Pid >= total {
			*err1 = "Dep PID OUT RANGE"
			return LOAD_ERROR
		}
		if entry.Pid >= 0 {
			if this.deps_[entry.Pid] == nil {
				this.deps_[entry.Pid] = NewDeps(int64(this.nodes_[entry.Pid].mtime_), 0)
			}
			this.deps_[entry.Pid].node_count += 1
			this.deps_[entry.Pid].nodes = append(this.deps_[entry.Pid].nodes, this.nodes_[entry.Id])
		}
	}
	return LOAD_SUCCESS
}

func (d *DepsLog) Truncate(path string, offset int64, err1 *string) bool {
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		*err1 = err.Error()
		return false
	}
	defer file.Close()

	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		*err1 = err.Error()
		return false
	}

	err = file.Truncate(offset)
	if err != nil {
		*err1 = err.Error()
		return false
	}
	return true
}

func (this *DepsLog) GetDeps(node *Node) *Deps {
	// Abort if the node has no id (never referenced in the deps) or if
	// there's no deps recorded for the node.
	if node.id() < 0 || node.id() >= len(this.deps_) {
		return nil
	}
	return this.deps_[node.id()]
}
func (this *DepsLog) GetFirstReverseDepsNode(node *Node) *Node {
	for id := 0; id < len(this.deps_); id++ {
		deps := this.deps_[id]
		if deps == nil {
			continue
		}
		for i := 0; i < deps.node_count; i++ {
			if deps.nodes[i] == node {
				return this.nodes_[id]
			}
		}
	}
	return nil
}

// / Rewrite the known log entries, throwing away old data.
func (this *DepsLog) Recompact(path string, err *string) bool {
	METRIC_RECORD(".ninja_deps recompact")
	return true
}

// / Returns if the deps entry for a node is still reachable from the manifest.
// /
// / The deps log can contain deps entries for files that were built in the
// / past but are no longer part of the manifest.  This function returns if
// / this is the case for a given node.  This function is slow, don't call
// / it from code that runs on every build.
func IsDepsEntryLiveFor(node *Node) bool {
	// Skip entries that don't have in-edges or whose edges don't have a
	// "deps" attribute. They were in the deps log from previous builds, but
	// the the files they were for were removed from the build and their deps
	// entries are no longer needed.
	// (Without the check for "deps", a chain of two or more nodes that each
	// had deps wouldn't be collected in a single recompaction.)
	return node.in_edge() != nil && node.in_edge().GetBinding("deps") != ""
}

// / Used for tests.
func (this *DepsLog) nodes() []*Node { return this.nodes_ }
func (this *DepsLog) deps() []*Deps  { return this.deps_ }

// Updates the in-memory representation.  Takes ownership of |deps|.
// Returns true if a prior deps record was deleted.
func (this *DepsLog) UpdateDeps(out_id int, deps *Deps) bool {
	// 如果 outID 超出了当前切片的范围，则扩展切片
	if out_id >= len(this.deps_) {
		this.deps_ = append(this.deps_, make([]*Deps, out_id+1-len(this.deps_))...)
	}

	// 检查是否需要删除旧的依赖项
	deleteOld := this.deps_[out_id] != nil
	if deleteOld {
		// 如果需要，删除旧的依赖项
		// 在 Go 中，我们不需要显式删除对象，但可能需要做一些清理工作
		// 例如，如果 Deps 包含指针或需要显式释放资源
		// 这里只是将旧的依赖项设置为 nil
		this.deps_[out_id] = nil
	}
	this.deps_[out_id] = deps
	return deleteOld
}

// Write a node name record, assigning it an id.
func (this *DepsLog) RecordId(node *Node, mtime TimeStamp, pid int64) bool {
	pathSize := len(node.path_)
	if pathSize == 0 {
		return false // 尝试记录空路径节点
	}
	if !this.OpenForWriteIfNeeded() {
		return false
	}
	id := len(this.nodes_)
	this.stmtInsert.Reset()
	this.stmtInsert.SetInt64("$id", int64(id))
	this.stmtInsert.SetText("$path", node.path_)
	this.stmtInsert.SetInt64("$mtime", int64(mtime))
	this.stmtInsert.SetInt64("$pid", pid)
	_, err := this.stmtInsert.Step()
	if err != nil {
		panic(err)
		return false
	}
	node.id_ = id
	this.nodes_ = append(this.nodes_, node)
	return true
}

// / Should be called before using file_. When false is returned, errno will
// / be set.
func (this *DepsLog) OpenForWriteIfNeeded() bool {
	if this.file_ != nil {
		return true
	}
	needCreateTable := false
	if _, err := os.Stat(this.file_path_); errors.Is(err, os.ErrNotExist) {
		needCreateTable = true
	} else if err != nil {
		panic(err)
		return false
	}
	var err error
	flag := sqlite.OpenReadWrite
	if needCreateTable {
		flag |= sqlite.OpenCreate
	}
	this.file_, err = sqlite.OpenConn(this.file_path_, flag)
	if err != nil {
		panic(err)
		return false
	}
	if needCreateTable {
		stmt, err := this.file_.Prepare("CREATE TABLE IF NOT EXISTS ninja_deps " +
			"(`id` INTEGER PRIMARY KEY, `path` TEXT, `mtime` INTEGER, `pid` INTEGER);")
		if err != nil {
			panic(err)
			return false
		}
		if _, err := stmt.Step(); err != nil {
			panic(err)
			return false
		}
	}
	this.stmtInsert, err = this.file_.Prepare("INSERT INTO ninja_deps (`id`, `path`, `mtime`, `pid`) VALUES" +
		" ($id, $path, $mtime, $pid) ON CONFLICT(id) " +
		" DO UPDATE SET `path`=$path, `mtime`= $mtime, `pid`=$pid;")
	if err != nil {
		panic(err)
		return false
	}
	this.stmtFileTell, err = this.file_.Prepare("SELECT count(*) FROM ninja_deps WHERE `id`=-2")
	if err != nil {
		panic(err)
		return false
	}
	this.stmtLoad, err = this.file_.Prepare("SELECT * FROM ninja_deps WHERE `id`<>-2")
	if err != nil {
		panic(err)
		return false
	}

	// 如果文件位置为 0，则写入文件签名和版本号
	if this.fileTell() == 0 {
		this.stmtInsert.Reset()
		this.stmtInsert.SetInt64("$id", -2)
		this.stmtInsert.SetText("$path", fmt.Sprintf(kFileSignature_DepsLog, kCurrentVersion_DepsLog))
		this.stmtInsert.SetInt64("$mtime", kCurrentVersion_DepsLog)
		this.stmtInsert.SetInt64("$pid", -2)
		_, err := this.stmtInsert.Step()
		if err != nil {
			panic(err)
			return false
		}
	}
	return true
}

func (this *DepsLog) fileTell() int64 {
	this.stmtFileTell.Reset()
	if hasRow, err := this.stmtFileTell.Step(); err != nil {
		return 0
	} else if !hasRow {
		return 0
	}
	cnt := this.stmtFileTell.ColumnInt64(0)
	return cnt
}

// SetCloseOnExec 设置文件描述符在执行时关闭
func SetCloseOnExec(fd int) {
	//var flags uint
	//if err := syscall.Fcntl(fd, syscall.F_GETFD, &flags); err != nil {
	//	// 处理错误
	//	return
	//}
	//flags |= syscall.FD_CLOEXEC
	//if err := syscall.Fcntl(fd, syscall.F_SETFD, &flags); err != nil {
	//	// 处理错误
	//}
}
