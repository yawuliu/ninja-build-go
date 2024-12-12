package main

import "ninja-build-go/model"

func CreateDeps(deps []*model.DepsEntry) error {
	if err := DB.Model(&model.DepsEntry{}).Create(deps).Error; err != nil {
		return err
	}
	return nil
}

func FindDepsByPid(pid int64) ([]*model.DepsEntry, error) {
	deps := make([]*model.DepsEntry, 0)
	if err := DB.Where("pid = ?", pid).Find(&deps).Error; err != nil {
		return nil, err
	}
	return deps, nil
}
