package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

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

func (this *BuildLog) LookupByOutputRbe(rbeService, rbeInstance, path string, commandHash uint64, currentMtime TimeStamp) *LogEntry {
	url := fmt.Sprintf("%s/query", rbeService)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return nil
	}
	q := req.URL.Query()
	q.Add("instance", rbeInstance)
	q.Add("output", path)
	q.Add("command_hash", strconv.FormatUint(commandHash, 16))
	q.Add("mtime", strconv.FormatInt(int64(currentMtime), 10))
	req.URL.RawQuery = q.Encode()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{Transport: tr, Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
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
		//iCommandHash, err := strconv.ParseInt(ret.CommandHash, 16, 64)
		//if err != nil {
		//	log.Println(err)
		//	return nil
		//}
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
		//ucommandHash := uint64(iCommandHash)
		istartTime := int(startTime)
		iendTime := int(endTime)
		//mtime, err := strconv.ParseInt(ret.Mtime, 10, 64)
		return &LogEntry{
			output:       ret.Output,
			command_hash: commandHash,
			start_time:   istartTime,
			end_time:     iendTime,
			mtime:        currentMtime,
		}
	} else {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return nil
		}
		log.Printf("StatusCode: %v, Body: %s\n", resp.StatusCode, string(data))
	}
	return nil
}

func UpdateRbeCache() {

}
