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
	input.Password = "123456"
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
			"createdat": user.CreatedAt,
			"updatedat": user.UpdatedAt,
			"lastlogin": user.LastLogin,
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
func HandleDeleteUser(c *fiber.Ctx) error {
	// 检查权限 - 只有管理员可以删除用户
	currentUserID, ok := c.Locals("userID").(uint)

	var role string
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "无法获取用户ID",
		})
	} else {
		var user1 model.User
		result1 := database.DB.First(&user1, currentUserID)
		if result1.Error != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "无权修改此用户信息",
			})
		}
		role = user1.Role
	}

	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "无权执行此操作",
		})
	}

	// 获取要删除的用户ID
	userID, err := strconv.Atoi(c.Params("id"))
	if userID == 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "系统管理不准删除",
		})
	}
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的用户ID",
		})
	}

	// 检查用户是否存在
	var user model.User
	result := database.DB.First(&user, userID)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "用户不存在",
		})
	}

	// 不能删除自己

	if uint(userID) == currentUserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "不能删除自己的账户",
		})
	}

	// 开始事务
	tx := database.DB.Begin()

	// 删除关联的登录日志
	if err := tx.Where("user_id = ?", userID).Delete(&model.LoginLog{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "删除用户登录日志失败",
		})
	}

	// 删除用户
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "删除用户失败",
		})
	}

	// 提交事务
	tx.Commit()

	return c.JSON(fiber.Map{
		"message": "用户删除成功",
	})
}

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

// HandleUpdateUser 更新用户信息
func HandleUpdateUser(c *fiber.Ctx) error {
	type UpdateUserInput struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		Status   string `json:"status"`
		Company  string `json:"company"`
	}

	input := new(UpdateUserInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的输入数据",
		})
	}

	// 获取要修改的用户ID
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的用户ID",
		})
	}

	// 检查用户是否存在
	var user model.User
	result := database.DB.First(&user, userID)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "用户不存在",
		})
	}

	// 检查权限 - 管理员或用户自己可以修改
	currentUserID, ok := c.Locals("userID").(uint)
	var role string
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "无法获取用户ID",
		})
	} else {
		var user1 model.User
		result1 := database.DB.First(&user1, currentUserID)
		if result1.Error != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "无权修改此用户信息",
			})
		}
		role = user1.Role
	}

	if uint(userID) != currentUserID && role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "无权修改此用户信息",
		})
	}

	// 更新用户信息
	if input.Username != "" {
		user.Username = input.Username
	}
	if input.Email != "" {
		user.Email = input.Email
	}
	// 只有管理员可以修改角色和状态
	if role == "admin" {
		if input.Role != "" {
			user.Role = input.Role
		}
		if input.Status != "" {
			user.Status = input.Status
		}
	}
	if input.Company != "" {
		user.Company = input.Company
	}
	user.UpdatedAt = time.Now()

	// 保存更新
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "更新用户信息失败",
		})
	}

	// 不返回密码
	user.Password = ""
	return c.JSON(user)
}
