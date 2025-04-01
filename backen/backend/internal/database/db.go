package database

import (
	"license-management-system/internal/model"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	// 创建数据目录
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal("创建数据目录失败:", err)
	}

	// 使用 data 目录下的数据库文件
	dbPath := filepath.Join(dataDir, "license.db")
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 自动迁移模型
	err = DB.AutoMigrate(&model.User{}, &model.License{}, &model.OperationLog{}, &model.LoginLog{})
	if err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	// 检查是否已存在管理员账户
	var adminCount int64
	DB.Model(&model.User{}).Where("username = ?", "admin").Count(&adminCount)

	if adminCount == 0 {
		// 生成密码哈希
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("生成密码哈希失败:", err)
		}

		// 创建管理员账户
		admin := &model.User{
			Username:  "admin",
			Password:  string(hashedPassword),
			Email:     "admin@example.com",
			Role:      "admin",
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := DB.Create(admin).Error; err != nil {
			log.Fatal("创建管理员账户失败:", err)
		}

		log.Println("已创建默认管理员账户")
	}
}
