package main

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ninja-build-go/model"
)

var DB *gorm.DB = nil

func migrate() error {
	err := DB.AutoMigrate(&model.RbeLogEntry{})
	if err != nil {
		return err
	}
	err = DB.AutoMigrate(&model.DepsEntry{})
	if err != nil {
		return err
	}
	return nil
}

func OpenDb(dbPath string) (err error) {
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}
	err = migrate()
	return
}

func CloseDb() (err error) {
	return
}
