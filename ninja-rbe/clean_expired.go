package main

import (
	"fmt"
	"github.com/tevino/abool/v2"
	"os"
	"path/filepath"
)

var cleanRunning = abool.NewBool(false)

func cleanTask() {
	if cleanRunning.IsSet() {
		return
	}
	cleanRunning.Set()
	defer cleanRunning.UnSet()
	fmt.Println("I am running clean task.")
	expiredRecords, err := FindExpiredLogWithLimit(2000)
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(expiredRecords) == 0 {
		return
	}
	var successCleans []int64
	for _, expiredRecord := range expiredRecords {
		storeName := fmt.Sprintf("%s_%s", expiredRecord.CommandHash, expiredRecord.Mtime)
		needDeletdPath := filepath.Join(fsRootDir, storeName)
		err := os.RemoveAll(needDeletdPath)
		if err != nil {
			fmt.Println(err)
		} else {
			successCleans = append(successCleans, expiredRecord.Id)
		}
	}
	if len(successCleans) == 0 {
		return
	}
	err = UpdateExpiredCleanResult(successCleans)
	if err != nil {
		fmt.Println(err)
	}
}
