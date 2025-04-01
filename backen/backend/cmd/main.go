package main

import (
	"license-management-system/internal/database"
	"license-management-system/internal/handler"
	"license-management-system/internal/middleware"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// 初始化数据库
	database.InitDB()

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// 中间件
	app.Use(logger.New())
	app.Use(cors.New())

	// 路由组
	api := app.Group("/api/v1")
	// 认证路由
	auth := api.Group("/auth")
	auth.Post("/validate-token", handler.HandleValidateToken) // 添加验证token的路由

	// 需要认证的路由
	authProtected := auth.Group("/")
	authProtected.Use(middleware.Auth())
	authProtected.Post("/change-password", handler.HandleChangePassword)
	// 用户路由
	users := api.Group("/users")
	users.Post("/register", handler.HandleUserRegister)
	users.Post("/login", handler.HandleUserLogin)
	users.Get("/info", middleware.Auth(), handler.HandleUserInfo)
	users.Get("/search", middleware.Auth(), middleware.AdminOnly(), handler.HandleSearchUsers)
	users.Get("/login-logs", middleware.Auth(), handler.HandleGetLoginLogs)

	// 许可证路由
	licenses := api.Group("/licenses")
	licenses.Use(middleware.Auth())

	// 管理员专用路由
	licenses.Get("/licenses", middleware.AdminOnly(), handler.HandleGetAllLicenses)
	licenses.Post("/generate", middleware.AdminOnly(), handler.HandleLicenseGenerate)
	licenses.Post("/issue", middleware.AdminOnly(), handler.HandleLicenseIssue)
	licenses.Put("/:key", middleware.AdminOnly(), handler.HandleLicenseUpdate) // 添加更新许可证的路由
	licenses.Get("/statistics", middleware.AdminOnly(), handler.HandleLicenseStatistics)
	licenses.Delete("/:key", middleware.AdminOnly(), handler.HandleLicenseDelete)

	// 普通用户可访问的路由
	licenses.Get("/verify", handler.HandleLicenseVerify)
	licenses.Get("/:key", handler.HandleGetLicense) // 添加更新许可证的路由
	licenses.Post("/activate", handler.HandleLicenseActivate)
	licenses.Get("/usage", handler.HandleLicenseUsage) // 新增license使用记录查询路由

	log.Fatal(app.Listen(":80"))
}
