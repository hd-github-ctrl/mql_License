package handler

import (
	"license-management-system/internal/service"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func HandleGetLogs(c *fiber.Ctx) error {
	// 获取分页参数
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "10"))

	// 限制页面大小
	if pageSize > 100 {
		pageSize = 100
	}

	logs, total, err := service.GetOperationLogs(page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取日志失败",
		})
	}

	return c.JSON(fiber.Map{
		"logs":  logs,
		"total": total,
		"page":  page,
	})
}

func HandleGetUserLogs(c *fiber.Ctx) error {
	// 获取分页参数
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "10"))

	// 限制页面大小
	if pageSize > 100 {
		pageSize = 100
	}

	// 从上下文获取用户ID
	userID := c.Locals("userID").(uint)

	logs, total, err := service.GetUserOperationLogs(userID, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取日志失败",
		})
	}

	return c.JSON(fiber.Map{
		"logs":  logs,
		"total": total,
		"page":  page,
	})
}
