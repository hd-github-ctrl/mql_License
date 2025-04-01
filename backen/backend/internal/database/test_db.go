package database

import (
	"license-management-system/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func InitTestDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect test database")
	}

	// 自动迁移测试数据库
	err = DB.AutoMigrate(&model.User{}, &model.License{}, &model.OperationLog{})
	if err != nil {
		panic("failed to migrate test database")
	}
}

func CleanTestDB() {
	sqlDB, err := DB.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}
