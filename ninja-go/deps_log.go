package ninja_go

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// The version is stored as 4 bytes after the signature and also serves as a
// byte order mark. Signature and version combined are 16 bytes long.
const DepsLog_kFileSignature = "# ninjadeps\n"
const kFileSignatureSize = len(DepsLog_kFileSignature) - 1

const DepsLog_kCurrentVersion = 4

const kMaxRecordSize = (1 << 19) - 1

type DepsLog struct {
	needs_recompaction_ bool
	file_               *os.File
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

// Writing (build-time) interface.
func (this *DepsLog) OpenForWrite(path string, err *string) bool {
	if this.needs_recompaction_ {
		if !this.Recompact(path, err) {
			return false
		}
	}

	if this.file_ == nil {
		panic("!this.file_")
	}
	this.file_path_ = path // we don't actually open the file right now, but will do
	// so on the first write attempt
	return true
}

func (this *DepsLog) RecordDeps(node *Node, mtime TimeStamp, nodes []*Node, err *string) bool {
	// Track whether there's any new data to be recorded.
	made_change := false
	node_count := len(nodes)

	// Assign ids to all nodes that are missing one.
	if node.id() < 0 {
		if !this.RecordId(node) {
			return false
		}
		made_change = true
	}
	for i := 0; i < node_count; i++ {
		if nodes[i].id() < 0 {
			if !this.RecordId(nodes[i]) {
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

	// Update on-disk representation.
	size := 4 * (1 + 2 + node_count)
	if size > kMaxRecordSize {
		*err = "ERANGE"
		return false
	}

	if !this.OpenForWriteIfNeeded() {
		return false
	}
	size |= 0x80000000 // Deps record: set high bit.
	_, err1 := fmt.Fprintf(this.file_, "%d", size)
	if err1 != nil {
		*err = err1.Error()
		return false
	}
	id := node.id()
	_, err1 = fmt.Fprintf(this.file_, "%d", id)
	if err1 != nil {
		*err = err1.Error()
		return false
	}
	mtime_part := uint32(mtime & 0xffffffff)
	_, err1 = fmt.Fprintf(this.file_, "%d", mtime_part)
	if err1 != nil {
		*err = err1.Error()
		return false
	}
	mtime_part = uint32((mtime >> 32) & 0xffffffff)
	_, err1 = fmt.Fprintf(this.file_, "%d", mtime_part)
	if err1 != nil {
		*err = err1.Error()
		return false
	}
	for i := 0; i < node_count; i++ {
		id = nodes[i].id()
		_, err1 := fmt.Fprintf(this.file_, "%d", id)
		if err1 != nil {
			*err = err1.Error()
			return false
		}
	}
	err1 = this.file_.Sync()
	if err1 != nil {
		*err = err1.Error()
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
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LOAD_NOT_FOUND
		}
		*err1 = err.Error()
		return LOAD_ERROR
	}
	defer file.Close()

	buf := make([]byte, kMaxRecordSize+1)
	_, err = io.ReadFull(file, buf[:kFileSignatureSize])
	if err != nil {
		return LOAD_ERROR
	}
	if !strings.HasPrefix(string(buf[:kFileSignatureSize]), kFileSignature) {
		*err1 = "bad deps log signature or version; starting over"
		os.Remove(path)
		return LOAD_SUCCESS
	}

	var version int32
	err = binary.Read(file, binary.LittleEndian, &version)
	if err != nil || version != kCurrentVersion {
		*err1 = "bad deps log version; starting over"
		os.Remove(path)
		return LOAD_SUCCESS
	}

	offset, _ := file.Seek(0, io.SeekCurrent)
	readFailed := false
	uniqueDepRecordCount := 0
	totalDepRecordCount := 0

	for {
		var size uint32
		err = binary.Read(file, binary.LittleEndian, &size)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				readFailed = true
			}
			break
		}
		isDeps := size&(1<<31) != 0
		size &= 0x7FFFFFFF

		if size > kMaxRecordSize || int(size)+4 > len(buf) {
			readFailed = true
			break
		}
		_, err = io.ReadFull(file, buf[:size])
		if err != nil {
			readFailed = true
			break
		}
		offset += int64(size) + 4

		if isDeps {
			if size%4 != 0 {
				readFailed = true
				break
			}
			depsData := binaryLittleEndianToIntSlice(buf[:size])
			outID := depsData[0]
			mtime := uint64(uint64(depsData[2])<<32) | uint64(depsData[1])
			depsData = depsData[3:]

			depsCount := len(depsData)
			for _, nodeID := range depsData {
				if nodeID >= len(this.nodes_) || this.nodes_[nodeID] == nil {
					readFailed = true
					break
				}
			}
			if readFailed {
				break
			}

			deps := &Deps{mtime: TimeStamp(mtime), nodes: make([]*Node, depsCount)}
			for i, nodeID := range depsData {
				deps.nodes[i] = this.nodes_[nodeID]
			}

			totalDepRecordCount++
			if !this.UpdateDeps(outID, deps) {
				uniqueDepRecordCount++
			}
		} else {
			pathSize := size - 4
			if pathSize <= 0 {
				readFailed = true
				break
			}
			for buf[pathSize-1] == '\000' {
				pathSize--
			}
			subpath := string(buf[:pathSize])
			node := state.GetNode(subpath, 0)

			checksum := binary.LittleEndian.Uint32(buf[size-4:])
			expectedID := int(^checksum)
			id := len(this.nodes_)
			if id != expectedID || node.id_ >= 0 {
				readFailed = true
				break
			}
			node.id_ = id
			this.nodes_ = append(this.nodes_, node)
		}
	}

	if readFailed {
		*err1 = "premature end of file"
		if !this.Truncate(path, offset, err1) {
			return LOAD_ERROR
		}
		*err1 += "; recovering"
		return LOAD_SUCCESS
	}

	if totalDepRecordCount > 1000 && totalDepRecordCount > uniqueDepRecordCount*3 {
		this.needs_recompaction_ = true
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

	this.Close()
	temp_path := path + ".recompact"

	// OpenForWrite() opens for append.  Make sure it's not appending to a
	// left-over file from a previous recompaction attempt that crashed somehow.
	os.RemoveAll(temp_path)

	var new_log DepsLog
	if !new_log.OpenForWrite(temp_path, err) {
		return false
	}

	// Clear all known ids so that new ones can be reassigned.  The new indices
	// will refer to the ordering in new_log, not in the current log.
	for _, i := range this.nodes_ {
		i.set_id(-1)
	}

	// Write out all deps again.
	for old_id := 0; old_id < len(this.deps_); old_id++ {
		deps := this.deps_[old_id]
		if deps == nil {
			continue
		} // If nodes_[old_id] is a leaf, it has no deps.

		if !IsDepsEntryLiveFor(this.nodes_[old_id]) {
			continue
		}

		if !new_log.RecordDeps(this.nodes_[old_id], deps.mtime, deps.nodes, err) {
			new_log.Close()
			return false
		}
	}

	new_log.Close()

	// All nodes now have ids that refer to new_log, so steal its data.
	this.deps_ = new_log.deps_
	this.nodes_ = new_log.nodes_

	err1 := os.RemoveAll(path)
	if err1 != nil {
		*err = err1.Error()
		return false
	}

	err1 = os.Rename(temp_path, path)
	if err1 != nil {
		*err = err1.Error()
		return false
	}

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
func (this *DepsLog) RecordId(node *Node) bool {
	pathSize := len(node.path_)
	if pathSize == 0 {
		return false // 尝试记录空路径节点
	}

	padding := (4 - pathSize%4) % 4 // 填充到4字节边界

	size := uint32(pathSize + padding + 4)
	if size > kMaxRecordSize {
		return false // 超过最大记录大小
	}

	if !this.OpenForWriteIfNeeded() {
		return false
	}

	writer := bufio.NewWriter(this.file_)
	if err := binary.Write(writer, binary.LittleEndian, size); err != nil {
		return false
	}
	if _, err := writer.WriteString(node.path_); err != nil {
		return false
	}
	if padding > 0 {
		if _, err := writer.Write(make([]byte, padding)); err != nil {
			return false
		}
	}
	id := len(this.nodes_)
	checksum := uint32(^uint32(id))
	if err := binary.Write(writer, binary.LittleEndian, checksum); err != nil {
		return false
	}
	if err := writer.Flush(); err != nil {
		return false
	}

	node.id_ = id
	this.nodes_ = append(this.nodes_, node)

	return true
}

// / Should be called before using file_. When false is returned, errno will
// / be set.
func (this *DepsLog) OpenForWriteIfNeeded() bool {
	if this.file_path_ == "" {
		return true
	}
	var err error
	this.file_, err = os.OpenFile(this.file_path_, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return false
	}
	defer func() {
		if this.file_ != nil {
			this.file_.Close()
		}
	}()

	// 设置缓冲区大小，并在每个记录后刷新文件缓冲区以确保记录不会被部分写入
	// this.file_.SetBufSize(kMaxRecordSize + 1)

	// 设置文件描述符在执行时关闭
	SetCloseOnExec(int(this.file_.Fd()))

	// 在 Windows 上，以追加模式打开文件不会将文件指针设置为文件末尾。明确地执行此操作。
	if _, err := this.file_.Seek(0, os.SEEK_END); err != nil {
		return false
	}

	// 如果文件位置为 0，则写入文件签名和版本号
	if this.fileTell() == 0 {
		if _, err := this.file_.Write([]byte(kFileSignature)); err != nil {
			return false
		}
		if err := binary.Write(this.file_, binary.LittleEndian, kCurrentVersion); err != nil {
			return false
		}
	}
	if err := this.file_.Sync(); err != nil {
		return false
	}
	this.file_path_ = "" // 文件已打开，清除路径
	return true
}

func (d *DepsLog) fileTell() int64 {
	pos, err := d.file_.Seek(0, os.SEEK_CUR)
	if err != nil {
		// 处理错误
		return 0
	}
	return pos
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
