package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
	OutputHash  string
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
		//
		{ // output md5 != remote md5 or output not exist download this
			needDownload := false
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
				needDownload = true
				color.Red("%s not exist.\n", path)
			} else if err == nil {
				hash, err2 := hashFileBase64(path, this.PrefixDir)
				if err2 != nil {
					needDownload = true
					color.Red("%s hashFileBase64 fail %s.\n", path, err2.Error())
				} else if string(hash) != ret.OutputHash {
					needDownload = true
					color.Red("%s  string(hash) != ret.OutputHash.\n", path)
				} else {
					color.Yellow("%s  string(hash) == ret.OutputHash.\n", path)
				}
			} else {
				color.Red("%s err: %s.\n", path, err.Error())
			}
			if needDownload {
				color.Green("RbeDownload %s\n", path)
				err2 := this.RbeDownload(path, ret.OutputHash, rbeService)
				if err2 != nil {
					return nil
				}
			} else {

			}
		}
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

func (this *BuildLog) WriteEntryRbe(entry *LogEntry) {
	if entry.mtime == 0 || entry.command_hash == 0 {
		return
	}
	this.UpdateRbeCache(this.config_.RbeService,
		entry.output, entry.output_hash,
		entry.command_hash,
		entry.start_time,
		entry.end_time,
		entry.mtime,
		this.config_.RbeInstance, "12h")
}

// https://github.com/PzaThief/benchmark-go-multipart
func (this *BuildLog) UpdateRbeCache(rbeService,
	output, output_hash string, command_hash uint64, start_time, end_time int,
	mtime TimeStamp, instance, expired_duration string) error {
	command_hash_str := strconv.FormatUint(command_hash, 16)
	start_time_str := strconv.FormatInt(int64(start_time), 10)
	end_time_str := strconv.FormatInt(int64(end_time), 10)
	mtime_str := strconv.FormatInt(int64(mtime), 10)
	file, err := os.Open(output)
	if err != nil {
		return err
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filepath.Base(output))
	io.Copy(part, file)
	writer.WriteField("output", output)
	writer.WriteField("command_hash", command_hash_str)
	writer.WriteField("start_time", start_time_str)
	writer.WriteField("end_time", end_time_str)
	writer.WriteField("mtime", mtime_str)
	writer.WriteField("instance", instance)
	writer.WriteField("expired_duration", expired_duration)
	writer.WriteField("output_hash", output_hash)
	writer.Close()
	url := fmt.Sprintf("%s/upload", rbeService)
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{Transport: tr, Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return nil
		}
		log.Printf("StatusCode: %v, Body: %s\n", resp.StatusCode, string(data))
	}
	return nil
}

func (this *BuildLog) RbeDownload(path, outputHash, rbeService string) error {
	// Create the file with .tmp extension, so that we won't overwrite a
	// file until it's downloaded fully
	out, err := os.Create(path + ".tmp")
	if err != nil {
		return err
	}

	// Get the data
	url := fmt.Sprintf("%s/%s", rbeService, outputHash)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create our bytes counter and pass it to be used alongside our writer
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Println()
	out.Close()
	// Rename the tmp file back to the original file
	err = os.Rename(path+".tmp", path)
	if err != nil {
		return err
	}

	return nil
}
