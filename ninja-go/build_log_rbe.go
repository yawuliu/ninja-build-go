package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/segmentio/fasthash/fnv1a"
	"github.com/zeebo/blake3"
	"golang.org/x/mod/sumdb/dirhash"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

func Hash1(files []string, prefix string, open func(string) (io.ReadCloser, error)) ([]byte, error) {
	h := blake3.New()
	files = append([]string(nil), files...)
	sort.Strings(files)
	for _, file := range files {
		if strings.Contains(file, "\n") {
			return nil, errors.New("dirhash: filenames with newlines are not supported")
		}
		r, err := open(file)
		if err != nil {
			return nil, err
		}
		hf := blake3.New()
		_, err = io.Copy(hf, r)
		r.Close()
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(h, "%x  %s\n", hf.Sum(nil), strings.TrimPrefix(file, prefix))
	}
	return h.Sum(nil), nil
}

func hashFile(path, prefix string) ([]byte, error) {
	file := strings.TrimPrefix(path, prefix)
	h := blake3.New()
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	hf := blake3.New()
	_, err = io.Copy(hf, r)
	r.Close()
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(h, "f: %x %s\n", hf.Sum(nil), file)
	return h.Sum(nil), nil
}

type HashFunc func(files []string, prefix string, open func(string) (io.ReadCloser, error)) ([]byte, error)

func hashDir(dir, prefix string, hash HashFunc) ([]byte, error) {
	files, err := dirhash.DirFiles(dir, prefix)
	if err != nil {
		return nil, err
	}
	osOpen := func(name string) (io.ReadCloser, error) {
		return os.Open(name) // filepath.Join(dir, strings.TrimPrefix(name, prefix))
	}
	return hash(files, prefix, osOpen)
}

func hashDirectory(path, prefix string) ([]byte, error) {
	hash, err := hashDir(path, prefix, Hash1)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func DirHash(path, prefix string, err1 *string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return 0 // 对应于 C++ 中的 ERROR_FILE_NOT_FOUND 或 ERROR_PATH_NOT_FOUND
		}
		*err1 = err.Error()
		return -1
	}
	h2 := fnv1a.Init64
	if info.IsDir() {
		hash, err := hashDirectory(path, prefix)
		if err != nil {
			*err1 = err.Error()
			return -1
		}
		h2 = fnv1a.AddBytes64(h2, hash)
	} else {
		hash, err := hashFile(path, prefix)
		if err != nil {
			*err1 = err.Error()
			return -1
		}
		h2 = fnv1a.AddBytes64(h2, hash)
	}
	return int64(h2)
}

type RbeLogEntry struct {
	Id          int64
	Output      string /* index_output */
	CommandHash string /* index_hash,UNIQUE */
	Mtime       string /* index_hash,UNIQUE */
	StartTime   string
	EndTime     string
	//
	Instance        string /* index_inst */
	CreatedAt       int64
	LastAccess      int64 /* index_last_access */
	ExpiredDuration int64 /* index_expired */
	Deleted         int64 /* 0 false 1 true */
}

func (this *BuildLog) LookupByOutputRbe(rbeService, rbeInstance, path, commandHash string) *LogEntry {
	url := fmt.Sprintf("%s/query", rbeService)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return nil
	}
	q := req.URL.Query()
	q.Add("instance", rbeInstance)
	q.Add("output", path)
	q.Add("command_hash", commandHash)
	req.URL.RawQuery = q.Encode()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil
	}
	ret := RbeLogEntry{}
	err = json.Unmarshal(data, &ret)
	if err != nil {
		log.Println(err)
		return nil
	}
	iCommandHash, err := strconv.ParseInt(ret.CommandHash, 16, 64)
	if err != nil {
		log.Println(err)
		return nil
	}
	startTime, err := strconv.ParseInt(ret.StartTime, 10, 64)
	if err != nil {
		log.Println(err)
		return nil
	}
	endTime, err := strconv.ParseInt(ret.EndTime, 10, 64)
	if err != nil {
		log.Println(err)
		return nil
	}
	ucommandHash := uint64(iCommandHash)
	istartTime := int(startTime)
	iendTime := int(endTime)
	mtime, err := strconv.ParseInt(ret.Mtime, 10, 64)
	return &LogEntry{
		output:       ret.Output,
		command_hash: ucommandHash,
		start_time:   istartTime,
		end_time:     iendTime,
		mtime:        TimeStamp(mtime),
	}
}
