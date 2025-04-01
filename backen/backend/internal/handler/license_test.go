package handler

import (
	"bytes"
	"encoding/json"
	"license-management-system/internal/database"
	"license-management-system/internal/model"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestHandleLicenseGenerate(t *testing.T) {
	app := fiber.New()
	database.InitTestDB()
	defer database.CleanTestDB()

	// 创建测试管理员用户
	adminUser := &model.User{
		Username: "admin",
		Role:     "admin",
	}
	database.DB.Create(adminUser)

	tests := []struct {
		name       string
		input      LicenseInput
		wantStatus int
		wantError  bool
	}{
		{
			name: "valid_license",
			input: LicenseInput{
				Version:     "1.0",
				ValidUntil:  time.Now().AddDate(0, 0, 30),
				Permissions: "full",
			},
			wantStatus: fiber.StatusCreated,
			wantError:  false,
		},
		{
			name: "invalid_days",
			input: LicenseInput{
				Version:     "1.0",
				ValidUntil:  time.Now().AddDate(0, 0, -1), // 无效的日期用于测试错误情况
				Permissions: "full",
			},
			wantStatus: fiber.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input)
			req, _ := http.NewRequest("POST", "/api/v1/licenses/generate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
