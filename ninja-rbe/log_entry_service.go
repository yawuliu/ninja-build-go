package main

import (
	"gorm.io/gorm"
	"ninja-build-go/model"
	"os"
	"time"
)

func SaveLogEntry(entry *model.RbeLogEntry) error {
	err := DB.Transaction(func(tx *gorm.DB) error {
		deps := entry.Deps
		entry.Deps = nil
		if err := DB.Create(&entry).Error; err != nil {
			return err
		}
		pid := entry.ID
		for i, _ := range deps {
			deps[i].PID = pid
		}
		if err := DB.Create(&deps).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func CheckEntryExist(params_hash string) (bool, error) {
	var cnt int64 = 0
	if err := DB.Model(&model.RbeLogEntry{}).Select("count(*)").
		Where("`params_hash`=?", params_hash).
		Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func UpdateFileAccess(paramsHash string) error {
	now := time.Now()
	if err := DB.Unscoped().Model(&model.RbeLogEntry{}).Where("`params_hash`=?", paramsHash).
		Update("last_access", now.Unix()).Error; err != nil {
		return err
	}
	return nil
}

func FindPotentialCacheRecords(instance, output, commandHash, inputHash string) ([]*model.RbeLogEntry, error) {
	var items []*model.RbeLogEntry
	if err := DB.Model(&model.RbeLogEntry{}).
		Where("`command_hash`=? and `input_hash`=? and `output`=? and `instance`=?",
			commandHash, inputHash, output, instance).Order("created_at desc").
		Limit(5).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, os.ErrNotExist
	}
	return items, nil
}
