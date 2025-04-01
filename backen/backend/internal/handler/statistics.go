package handler

import (
	"license-management-system/internal/database"
	"license-management-system/internal/model"
	"time"

	"github.com/gofiber/fiber/v2"
)

// HandleLicenseStatistics 处理许可证统计信息请求
func HandleLicenseStatistics(c *fiber.Ctx) error {
	// 获取查询参数
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	//product := c.Query("product")

	// 解析日期
	var start, end time.Time
	var err error

	if startDate != "" {
		start, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    400,
				"message": "开始日期格式错误",
				"errors": []fiber.Map{
					{"field": "start_date", "message": "日期格式应为 YYYY-MM-DD"},
				},
			})
		}
	} else {
		// 默认为30天前
		start = time.Now().AddDate(0, 0, -30)
	}

	if endDate != "" {
		end, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    400,
				"message": "结束日期格式错误",
				"errors": []fiber.Map{
					{"field": "end_date", "message": "日期格式应为 YYYY-MM-DD"},
				},
			})
		}
	} else {
		// 默认为当前时间
		end = time.Now()
	}

	// 获取数据库连接
	db := database.DB

	// 构建统计信息
	stats := &model.LicenseStatistics{
		LicensesByProduct: make(map[string]int),
		UsageByCountry:    make(map[string]int),
		UsageByDevice:     make(map[string]int),
		DailyUsage:        make([]model.DailyUsage, 0),
	}

	// 统计许可证总数
	if err := db.Model(&model.License{}).Count(&stats.TotalLicenses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取许可证总数失败",
		})
	}

	// 统计活跃许可证数
	if err := db.Model(&model.License{}).Where("status = ?", "active").Count(&stats.ActiveLicenses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取活跃许可证数失败",
		})
	}

	// 统计过期许可证数
	if err := db.Model(&model.License{}).Where("status = ?", "expired").Count(&stats.ExpiredLicenses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取过期许可证数失败",
		})
	}

	// 统计即将过期的许可证数（30天内）
	thirtyDaysLater := time.Now().AddDate(0, 0, 30)
	if err := db.Model(&model.License{}).
		Where("status = ? AND valid_until <= ?", "active", thirtyDaysLater).
		Count(&stats.ExpiringLicenses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取即将过期许可证数失败",
		})
	}

	// 统计已暂停的许可证数
	if err := db.Model(&model.License{}).Where("status = ?", "suspended").Count(&stats.SuspendedLicenses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取已暂停许可证数失败",
		})
	}

	// 统计已撤销的许可证数
	if err := db.Model(&model.License{}).Where("status = ?", "revoked").Count(&stats.RevokedLicenses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取已撤销许可证数失败",
		})
	}

	// 按产品统计许可证数量
	var productStats []struct {
		Product string
		Count   int
	}
	if err := db.Model(&model.License{}).
		Select("product, count(*) as count").
		Group("product").
		Scan(&productStats).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取产品统计失败",
		})
	}
	for _, ps := range productStats {
		stats.LicensesByProduct[ps.Product] = ps.Count
	}

	// 获取每日使用统计
	var dailyStats []model.DailyUsage
	if err := db.Model(&model.LoginLog{}).
		Select("DATE(created_at) as date, COUNT(DISTINCT user_id) as active_users, COUNT(*) as total_checks").
		Where("created_at BETWEEN ? AND ?", start, end).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&dailyStats).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取每日使用统计失败",
		})
	}
	stats.DailyUsage = dailyStats

	// 按国家统计使用量
	var countryStats []struct {
		Country string
		Count   int
	}
	if err := db.Model(&model.LoginLog{}).
		Select("country, count(*) as count").
		Where("created_at BETWEEN ? AND ?", start, end).
		Group("country").
		Scan(&countryStats).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取国家统计失败",
		})
	}
	for _, cs := range countryStats {
		stats.UsageByCountry[cs.Country] = cs.Count
	}

	// 按设备类型统计使用量
	var deviceStats []struct {
		Device string
		Count  int
	}
	if err := db.Model(&model.LoginLog{}).
		Select("device, count(*) as count").
		Where("created_at BETWEEN ? AND ?", start, end).
		Group("device").
		Scan(&deviceStats).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取设备统计失败",
		})
	}
	for _, ds := range deviceStats {
		stats.UsageByDevice[ds.Device] = ds.Count
	}

	// 计算平均使用时长
	if err := db.Model(&model.LoginLog{}).
		Select("AVG(TIMESTAMPDIFF(HOUR, created_at, updated_at)) as avg_duration").
		Where("created_at BETWEEN ? AND ?", start, end).
		Scan(&stats.AverageUsageDuration).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取平均使用时长失败",
		})
	}

	// 统计激活次数
	if err := db.Model(&model.License{}).Where("status != ?", "inactive").Count(&stats.TotalActivations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取激活次数失败",
		})
	}

	// 统计失败的激活次数
	if err := db.Model(&model.License{}).Where("status = ?", "activation_failed").Count(&stats.FailedActivations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "获取失败激活次数失败",
		})
	}

	return c.JSON(fiber.Map{
		"code":    200,
		"message": "success",
		"data":    stats,
	})
}
