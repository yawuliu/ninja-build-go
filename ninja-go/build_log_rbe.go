package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

func HashSingleFile(path, prefix string) (string, error) {
	file := strings.TrimPrefix(path, prefix)
	h := blake3.New()
	r, err := os.Open(file)
	if err != nil {
		return "", err
	}
	hf := blake3.New()
	_, err = io.Copy(hf, r)
	r.Close()
	if err != nil {
		return "", err
	}
	fmt.Fprintf(h, "%x  %s\n", hf.Sum(nil), file)
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func HashBlake3(files []string, open func(string) (io.ReadCloser, error)) (string, error) {
	h := blake3.New()
	files = append([]string(nil), files...)
	sort.Strings(files)
	for _, file := range files {
		if strings.Contains(file, "\n") {
			return "", errors.New("dirhash: filenames with newlines are not supported")
		}
		r, err := open(file)
		if err != nil {
			return "", err
		}
		hf := blake3.New()
		_, err = io.Copy(hf, r)
		r.Close()
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%x  %s\n", hf.Sum(nil), file)
	}
	return "h1:" + base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func HashDirectory(path, prefix string) (string, error) {
	hash, err := dirhash.HashDir(path, prefix, HashBlake3)
	if err != nil {
		return "", err
	}
	return hash, nil
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
	commandHash, err := strconv.ParseInt(ret.CommandHash, 16, 64)
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
	ucommandHash := uint64(commandHash)
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
