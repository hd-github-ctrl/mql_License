package handler

import (
	"bytes"
	"encoding/json"
	"license-management-system/internal/database"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestHandleUserRegister(t *testing.T) {
	// 初始化测试环境
	app := fiber.New()
	database.InitTestDB() // 使用测试数据库
	defer database.CleanTestDB()

	tests := []struct {
		name       string
		input      RegisterInput
		wantStatus int
		wantError  bool
	}{
		{
			name: "valid_registration",
			input: RegisterInput{
				Username: "testuser",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantStatus: fiber.StatusCreated,
			wantError:  false,
		},
		{
			name: "duplicate_username",
			input: RegisterInput{
				Username: "testuser",
				Password: "password123",
				Email:    "another@example.com",
			},
			wantStatus: fiber.StatusInternalServerError,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.input)
			req, _ := http.NewRequest("POST", "/api/v1/users/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
