package handler

import (
	"license-management-system/internal/database"
	"license-management-system/internal/model"
	"license-management-system/internal/util"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type RegisterInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 添加用户搜索查询参数
type UserSearchQuery struct {
	Page     int    `query:"page"`
	PageSize int    `query:"page_size"`
	Keyword  string `query:"keyword"`
	Role     string `query:"role"`
	Status   string `query:"status"`
}

func HandleUserRegister(c *fiber.Ctx) error {
	input := new(RegisterInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的输入数据",
		})
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "密码加密失败",
		})
	}

	user := &model.User{
		Username: input.Username,
		Password: string(hashedPassword),
		Email:    input.Email,
		Role:     "user",
	}

	result := database.DB.Create(user)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "用户创建失败",
		})
	}

	// 不返回密码
	user.Password = ""
	return c.Status(fiber.StatusCreated).JSON(user)
}

func HandleUserLogin(c *fiber.Ctx) error {
	input := new(LoginInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的输入数据",
		})
	}

	var user model.User
	result := database.DB.Where("username = ?", input.Username).First(&user)
	if result.Error != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "用户名或密码错误",
		})
	}

	// 验证密码
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "用户名或密码错误",
		})
	}

	// 记录登录日志
	loginLog := &model.LoginLog{
		UserID:    user.ID,
		IP:        c.IP(),
		UserAgent: c.Get("User-Agent"),
		Status:    "success",
		CreatedAt: time.Now(),
	}
	database.DB.Create(loginLog)
	// 更新用户最后登录时间
	user.LastLogin = time.Now()
	database.DB.Save(&user)

	// 生成JWT令牌
	token, err := util.GenerateToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "令牌生成失败",
		})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user": fiber.Map{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"company":   user.Company,
			"createdAt": user.CreatedAt,
			"updatedAt": user.UpdatedAt,
			"lastLogin": user.LastLogin,
		},
	})
}

func HandleUserInfo(c *fiber.Ctx) error {
	// 从上下文中获取用户ID（需要认证中间件支持）
	userID := c.Locals("userID").(uint)

	var user model.User
	result := database.DB.First(&user, userID)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "用户不存在",
		})
	}

	// 不返回密码
	user.Password = ""
	return c.JSON(user)
}

func HandleSearchUsers(c *fiber.Ctx) error {
	query := new(UserSearchQuery)
	if err := c.QueryParser(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的查询参数",
		})
	}

	// 设置默认值
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 10
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	db := database.DB.Model(&model.User{})

	// 关键词搜索
	if query.Keyword != "" {
		db = db.Where("username LIKE ? OR email LIKE ?",
			"%"+query.Keyword+"%", "%"+query.Keyword+"%")
	}

	// 角色筛选
	if query.Role != "" {
		db = db.Where("role = ?", query.Role)
	}

	// 状态筛选
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	// 获取总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取用户总数失败",
		})
	}

	// 获取分页数据
	var users []model.User
	offset := (query.Page - 1) * query.PageSize
	if err := db.Offset(offset).Limit(query.PageSize).Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取用户列表失败",
		})
	}

	// 清除密码
	for i := range users {
		users[i].Password = ""
	}

	return c.JSON(fiber.Map{
		"users": users,
		"total": total,
		"page":  query.Page,
		"size":  query.PageSize,
	})
}

func HandleGetLoginLogs(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "10"))

	// 限制页面大小
	if pageSize > 100 {
		pageSize = 100
	}

	var logs []model.LoginLog
	var total int64

	db := database.DB.Model(&model.LoginLog{}).Where("user_id = ?", userID)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取登录日志总数失败",
		})
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取登录日志失败",
		})
	}

	return c.JSON(fiber.Map{
		"logs":  logs,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// ResetPasswordHandler handles user password reset requests
func HandleChangePassword(c *fiber.Ctx) error {
	type ResetPasswordInput struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}

	input := new(ResetPasswordInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的输入数据",
		})
	}

	// 从上下文中获取用户ID（需要认证中间件支持）
	userID := c.Locals("userID").(uint)

	var user model.User
	result := database.DB.First(&user, userID)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "用户不存在",
		})
	}

	// 验证当前密码
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.CurrentPassword))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "当前密码错误",
		})
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "密码加密失败",
		})
	}

	// 更新密码
	user.Password = string(hashedPassword)
	result = database.DB.Save(&user)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "密码更新失败",
		})
	}

	return c.JSON(fiber.Map{
		"message": "密码更新成功",
	})
}

// HandleValidateToken 验证token的有效性
func HandleValidateToken(c *fiber.Ctx) error {
	type TokenInput struct {
		Token string `json:"token"`
	}

	input := new(TokenInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的输入数据",
		})
	}

	if input.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "未提供token",
			"valid": false,
		})
	}

	// 验证token
	userID, err := util.ValidateToken(input.Token)
	if err != nil {
		return c.JSON(fiber.Map{
			"valid": false,
			"error": "无效的token",
		})
	}

	// 检查用户是否存在
	var user model.User
	result := database.DB.First(&user, userID)
	if result.Error != nil {
		return c.JSON(fiber.Map{
			"valid": false,
			"error": "用户不存在",
		})
	}

	return c.JSON(fiber.Map{
		"valid": true,
		"user": fiber.Map{
			"id":       userID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}
