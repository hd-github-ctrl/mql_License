package middleware

import (
	"license-management-system/internal/database"
	"license-management-system/internal/model"
	"license-management-system/internal/util"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func Auth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "未提供认证令牌",
			})
		}

		// 获取 Bearer token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "无效的认证格式",
			})
		}

		// 验证令牌
		userID, err := util.ValidateToken(tokenParts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "无效的认证令牌",
			})
		}

		// 将用户ID存储在上下文中
		c.Locals("userID", userID)
		return c.Next()
	}
}

func AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {

		userID := c.Locals("userID").(uint)

		// 从数据库获取用户信息并检查角色
		var user model.User
		result := database.DB.First(&user, userID)
		if result.Error != nil || user.Role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "需要管理员权限",
			})
		}

		return c.Next()
	}
}
