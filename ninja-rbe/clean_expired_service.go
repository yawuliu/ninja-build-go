package main

import (
	"fmt"
	"github.com/tevino/abool/v2"
	"ninja-build-go/model"
	"os"
	"path/filepath"
	"time"
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
		storeName := fmt.Sprintf("%s", expiredRecord.OutputHash)
		needDeletdPath := filepath.Join(fsRootDir, storeName)
		err := os.RemoveAll(needDeletdPath)
		if err != nil {
			fmt.Println(err)
		} else {
			successCleans = append(successCleans, expiredRecord.ID)
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

func FindExpiredLogWithLimit(limit int) ([]*model.RbeLogEntry, error) {
	var expiredLogs []*model.RbeLogEntry
	now := time.Now().Unix()
	if err := DB.Model(&model.RbeLogEntry{}).Where("`last_access`+`expired_duration` < ?", now).
		Limit(limit).Find(&expiredLogs).Error; err != nil {
		return nil, err
	}
	return expiredLogs, nil
}

func UpdateExpiredCleanResult(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if err := DB.Model(&model.RbeLogEntry{}).Delete(&model.RbeLogEntry{}, ids).Error; err != nil {
		return err
	}
	return nil
}
