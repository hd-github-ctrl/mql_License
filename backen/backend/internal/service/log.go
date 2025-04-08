package service

import (
	"encoding/json"
	"license-management-system/internal/database"
	"license-management-system/internal/model"
	"time"
)

func LogOperation(userID uint, action string, target string, targetID string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	log := &model.OperationLog{
		UserID:    userID,
		Action:    action,
		Target:    target,
		TargetID:  targetID,
		Details:   string(detailsJSON),
		CreatedAt: time.Now(),
	}

	return database.DB.Create(log).Error
}

// 获取操作日志列表
func GetOperationLogs(page, pageSize int) ([]model.OperationLog, int64, error) {
	var logs []model.OperationLog
	var total int64

	db := database.DB

	// 获取总数
	if err := db.Model(&model.OperationLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// 获取用户的操��日志
func GetUserOperationLogs(userID uint, page, pageSize int) ([]model.OperationLog, int64, error) {
	var logs []model.OperationLog
	var total int64

	db := database.DB

	// 获取总数
	if err := db.Model(&model.OperationLog{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
