package main

import (
	"errors"
	"fmt"
	"os"
)

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
func (this *DepsLog) RecordDeps(node *Node, mtime TimeStamp, nodes []*Node) bool {
  // Track whether there's any new data to be recorded.
 made_change := false
 node_count := len(nodes)

  // Assign ids to all nodes that are missing one.
  if (node.id() < 0) {
    if !this.RecordId(node) {
		return false
	}
    made_change = true;
  }
  for  i := 0; i < node_count; i++ {
    if (nodes[i].id() < 0) {
      if !this.RecordId(nodes[i]) {
		  return false
	  }
      made_change = true;
    }
  }

  // See if the new data is different than the existing data, if any.
  if (!made_change) {
    deps := this.GetDeps(node);
    if deps==nil || deps.mtime != mtime || deps.node_count != node_count {
      made_change = true;
    } else {
      for i := 0; i < node_count; i++ {
        if deps.nodes[i] != nodes[i] {
          made_change = true;
          break;
        }
      }
    }
  }

  // Don't write anything if there's no new info.
  if (!made_change) {
	  return true
  }

  // Update on-disk representation.
  size := 4 * (1 + 2 + node_count);
  if (size > kMaxRecordSize) {
    errno = ERANGE;
    return false;
  }

  if !this.OpenForWriteIfNeeded() {
    return false;
  }
  size |= 0x80000000;  // Deps record: set high bit.
	_,err1 := fmt.Fprintf(this.file_,"%d", size)
  if err1!=nil {
	  return false
  }
  id := node.id();
   _,err1 = fmt.Fprintf(this.file_,"%d", id)
  if err1!=nil {
	  return false
  }
  mtime_part := uint32(mtime & 0xffffffff)
  _,err1 = fmt.Fprintf(this.file_,"%d", mtime_part)
  if err1!=nil {
	  return false
  }
  mtime_part = uint32((mtime >> 32) & 0xffffffff);
  _,err1 = fmt.Fprintf(this.file_,"%d", mtime_part)
  if err1!=nil {
	  return false
  }
  for i := 0; i < node_count; i++ {
    id = nodes[i].id();
	_,err1 := fmt.Fprintf(this.file_,"%d", id)
    if err1!=nil {
		return false
	}
  }
  err1 = this.file_.Sync()
  if err1!=nil {
	  return false
  }

  // Update in-memory representation.
  deps := NewDeps(mtime, node_count);
  for i := 0; i < node_count; i++ {
		deps.nodes[i] = nodes[i]
  }
  this.UpdateDeps(node.id(), deps);

  return true;
}

func (this *DepsLog) Close() {
	this.OpenForWriteIfNeeded();  // create the file even if nothing has been recorded
	if this.file_!=nil {
		this.file_.Close()
	}
	this.file_ = nil
}

func (this *DepsLog) Load(path string, state *State, err *string) LoadStatus {
	METRIC_RECORD(".ninja_deps load")
	var buf string
	f,err1 := os.Open(path)
	if  err1!=nil {
		if errors.Is(err1, os.ErrNotExist) {
			return LOAD_NOT_FOUND
		}
		*err = err1.Error()
		return LOAD_ERROR;
	}
  valid_header := fread(buf, kFileSignatureSize, 1, f) == 1 &&
                      !memcmp(buf, kFileSignature, kFileSignatureSize);

   version := int32(0)
  valid_version := fread(&version, 4, 1, f) == 1 && version == kCurrentVersion;

  // Note: For version differences, this should migrate to the new format.
  // But the v1 format could sometimes (rarely) end up with invalid data, so
  // don't migrate v1 to v3 to force a rebuild. (v2 only existed for a few days,
  // and there was no release with it, so pretend that it never happened.)
  if (!valid_header || !valid_version) {
    if (version == 1) {
		*err = "deps log version change; rebuilding"
	}else {
		*err = "bad deps log signature or version; starting over"
	}
	f.Close()
    os.RemoveAll(path)
    // Don't report this as a failure.  An empty deps log will cause
    // us to rebuild the outputs anyway.
    return LOAD_SUCCESS;
  }

  var offset int64 = ftell(f);
  read_failed := false;
   unique_dep_record_count := 0;
  total_dep_record_count := 0;
  for  {
     size := uint32(0)
    if (fread(&size, sizeof(size), 1, f) < 1) {
      if (!feof(f)) {
		  read_failed = true
	  }
      break;
    }
    is_deps := (size >> 31) != 0;
    size = size & 0x7FFFFFFF;

    if (size > kMaxRecordSize || fread(buf, size, 1, f) < 1) {
      read_failed = true;
      break;
    }
    offset += size + sizeof(size);

    if (is_deps) {
      if ((size % 4) != 0) {
        read_failed = true;
        break;
      }
      deps_data := reinterpret_cast<int*>(buf);
      out_id := deps_data[0];
       var mtime TimeStamp
      mtime = (TimeStamp)(((uint64_t)(unsigned int)deps_data[2] << 32) |
                          (uint64_t)(unsigned int)deps_data[1]);
      deps_data += 3;
      deps_count := (size / 4) - 3;

      for i := 0; i < deps_count; i++ {
        int node_id = deps_data[i];
        if (node_id >= (int)nodes_.size() || !nodes_[node_id]) {
          read_failed = true;
          break;
        }
      }
      if (read_failed){
break;
	  }

      Deps* deps = new Deps(mtime, deps_count);
      for (int i = 0; i < deps_count; ++i) {
        deps.nodes[i] = nodes_[deps_data[i]];
      }

      total_dep_record_count++;
      if (!UpdateDeps(out_id, deps))
        ++unique_dep_record_count;
    } else {
      int path_size = size - 4;
      if (path_size <= 0) {
        read_failed = true;
        break;
      }
      // There can be up to 3 bytes of padding.
      if (buf[path_size - 1] == '\0') {--path_size;}
      if (buf[path_size - 1] == '\0') {--path_size;}
      if (buf[path_size - 1] == '\0') {--path_size;}
      StringPiece subpath(buf, path_size);
      // It is not necessary to pass in a correct slash_bits here. It will
      // either be a Node that's in the manifest (in which case it will already
      // have a correct slash_bits that GetNode will look up), or it is an
      // implicit dependency from a .d which does not affect the build command
      // (and so need not have its slashes maintained).
      node := state.GetNode(subpath, 0);

      // Check that the expected index matches the actual index. This can only
      // happen if two ninja processes write to the same deps log concurrently.
      // (This uses unary complement to make the checksum look less like a
      // dependency record entry.)
      unsigned checksum = *reinterpret_cast<unsigned*>(buf + size - 4);
      int expected_id = ~checksum;
      int id = nodes_.size();
      if (id != expected_id || node.id() >= 0) {
        read_failed = true;
        break;
      }
      node.set_id(id);
      nodes_.push_back(node);
    }
  }

  if (read_failed) {
    // An error occurred while loading; try to recover by truncating the
    // file to the last fully-read record.
    if (ferror(f)) {
      *err = strerror(ferror(f));
    } else {
      *err = "premature end of file";
    }
    fclose(f);

    if (!Truncate(path, offset, err))
      return LOAD_ERROR;

    // The truncate succeeded; we'll just report the load error as a
    // warning because the build can proceed.
    *err += "; recovering";
    return LOAD_SUCCESS;
  }

  fclose(f);

  // Rebuild the log if there are too many dead records.
  int kMinCompactionEntryCount = 1000;
  int kCompactionRatio = 3;
  if (total_dep_record_count > kMinCompactionEntryCount &&
      total_dep_record_count > unique_dep_record_count * kCompactionRatio) {
    needs_recompaction_ = true;
  }

  return LOAD_SUCCESS
}
func (this *DepsLog) GetDeps(node *Node) *Deps                               {
  // Abort if the node has no id (never referenced in the deps) or if
  // there's no deps recorded for the node.
  if (node.id() < 0 || node.id() >= len(this.deps_)){
		return nil
	}
  return  this.deps_[node.id()];
}
func (this *DepsLog) GetFirstReverseDepsNode(node *Node) *Node               {
  for id := 0; id <  len(this.deps_); id++ {
    deps := this.deps_[id];
    if deps==nil {
		continue
	}
    for i := 0; i < deps.node_count; i++ {
      if (deps.nodes[i] == node) {
		  return this.nodes_[id]
	  }
    }
  }
  return nil
}

// / Rewrite the known log entries, throwing away old data.
func (this *DepsLog) Recompact(path string, err *string) bool {
  METRIC_RECORD(".ninja_deps recompact");

  this.Close();
   temp_path := path + ".recompact";

  // OpenForWrite() opens for append.  Make sure it's not appending to a
  // left-over file from a previous recompaction attempt that crashed somehow.
  os.RemoveAll(temp_path)

  var  new_log DepsLog
  if (!new_log.OpenForWrite(temp_path, err)) {
	  return false
  }

  // Clear all known ids so that new ones can be reassigned.  The new indices
  // will refer to the ordering in new_log, not in the current log.
  for _,i := range this.nodes_ {
		i.set_id(-1)
	}

  // Write out all deps again.
  for old_id := 0; old_id < len(this.deps_); old_id++ {
   deps := this.deps_[old_id];
    if deps==nil {
		continue
	}  // If nodes_[old_id] is a leaf, it has no deps.

    if (!IsDepsEntryLiveFor(this.nodes_[old_id])) {
		continue
	}

    if (!new_log.RecordDeps(this.nodes_[old_id], deps.mtime,  deps.nodes)) {
      new_log.Close();
      return false;
    }
  }

  new_log.Close();

  // All nodes now have ids that refer to new_log, so steal its data.
  this.deps_.swap(new_log.deps_);
  this.nodes_.swap(new_log.nodes_);

  err1:= os.RemoveAll(path)
  if err1!=nil {
    *err = err1.Error()
    return false;
  }

  err1 = os.Rename(temp_path, path)
  if err1!=nil {
    *err = err1.Error()
    return false;
  }

  return true;
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
  return node.in_edge()!=nil && node.in_edge().GetBinding("deps") !=""
}

// / Used for tests.
func (this *DepsLog) nodes() []*Node { return this.nodes_ }
func (this *DepsLog) deps() []*Deps  { return this.deps_ }

// Updates the in-memory representation.  Takes ownership of |deps|.
// Returns true if a prior deps record was deleted.
func (this *DepsLog) UpdateDeps(out_id int, deps *Deps) bool {
  if out_id >= len(this.deps_) {
	  this.deps_.resize(out_id + 1)
  }

  delete_old := this.deps_[out_id] != nil;
  if (delete_old) {
	  delete(this.deps_, out_id)
  }
	this.deps_[out_id] = deps;
  return delete_old;
}

// Write a node name record, assigning it an id.
func (this *DepsLog) RecordId(node *Node) bool {
  path_size := len(node.path())
  if (path_size > 0) {
	  panic("Trying to record empty path Node!")
  }
  padding := (4 - path_size % 4) % 4;  // Pad path to 4 byte boundary.

  size := path_size + padding + 4;
  if (size > kMaxRecordSize) {
    errno = ERANGE;
    return false;
  }

  if (!this.OpenForWriteIfNeeded()) {
    return false;
  }
  if (fwrite(&size, 4, 1, this.file_) < 1) {
	  return false
  }
  if (fwrite(node.path().data(), path_size, 1, this.file_) < 1) {
    return false;
  }
  if (padding && fwrite("\0\0", padding, 1, this.file_) < 1) {
	  return false
  }
  id := this.nodes_.size();
  checksum := ~(unsigned)id;
  if (fwrite(&checksum, 4, 1, this.file_) < 1) {
	  return false
  }
  if (fflush(this.file_) != 0) {
	  return false
  }

  node.set_id(id);
	this.nodes_ =append(this.nodes_, node)

  return true;
}

// / Should be called before using file_. When false is returned, errno will
// / be set.
func (this *DepsLog) OpenForWriteIfNeeded() bool {
  if (this.file_path_.empty()) {
    return true;
  }
	this.file_ = fopen(this.file_path_, "ab");
  if this.file_==nil {
    return false;
  }
  // Set the buffer size to this and flush the file buffer after every record
  // to make sure records aren't written partially.
  if (setvbuf(this.file_, NULL, _IOFBF, kMaxRecordSize + 1) != 0) {
    return false;
  }
	this.SetCloseOnExec(fileno(file_));

  // Opening a file in append mode doesn't set the file pointer to the file's
  // end on Windows. Do that explicitly.
  fseek(this.file_, 0, SEEK_END);

  if (ftell(this.file_) == 0) {
    if (fwrite(kFileSignature, sizeof(kFileSignature) - 1, 1, this.file_) < 1) {
      return false;
    }
    if (fwrite(&kCurrentVersion, 4, 1, this.file_) < 1) {
      return false;
    }
  }
  if (fflush(this.file_) != 0) {
    return false;
  }
	this.file_path_ = ""
  return true;
}
