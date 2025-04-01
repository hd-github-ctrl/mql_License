package model

import "time"

// DailyUsage 每日使用统计
type DailyUsage struct {
	Date           time.Time `json:"date"`
	ActiveUsers    int       `json:"active_users"`
	NewActivations int       `json:"new_activations"`
	TotalChecks    int       `json:"total_checks"`
}

// LicenseStatistics 许可证统计信息
type LicenseStatistics struct {
	TotalLicenses        int64          `json:"total_licenses"`
	ActiveLicenses       int64          `json:"active_licenses"`
	ExpiredLicenses      int64          `json:"expired_licenses"`
	ExpiringLicenses     int64          `json:"expiring_licenses"`
	SuspendedLicenses    int64          `json:"suspended_licenses"`
	RevokedLicenses      int64          `json:"revoked_licenses"`
	LicensesByProduct    map[string]int `json:"licenses_by_product"`
	DailyUsage           []DailyUsage   `json:"daily_usage"`
	UsageByCountry       map[string]int `json:"usage_by_country"`
	UsageByDevice        map[string]int `json:"usage_by_device"`
	AverageUsageDuration float64        `json:"average_usage_duration"`
	TotalActivations     int64          `json:"total_activations"`
	FailedActivations    int64          `json:"failed_activations"`
}

// GetSuccessRate 计算激活成功率
func (ls *LicenseStatistics) GetSuccessRate() float64 {
	if ls.TotalActivations == 0 {
		return 0
	}
	return float64(ls.TotalActivations-ls.FailedActivations) / float64(ls.TotalActivations)
}

// GetExpiringLicensesCount 获取即将过期的许可证数量（30天内）
func (ls *LicenseStatistics) GetExpiringLicensesCount() int64 {
	return ls.ExpiringLicenses
}

// GetTotalActiveUsers 获取总活跃用户数
func (ls *LicenseStatistics) GetTotalActiveUsers() int64 {
	return ls.ActiveLicenses
}

// GetUsageByProduct 获取指定产品的使用量
func (ls *LicenseStatistics) GetUsageByProduct(product string) int {
	if count, ok := ls.LicensesByProduct[product]; ok {
		return count
	}
	return 0
}

// GetUsageByCountry 获取指定国家的使用量
func (ls *LicenseStatistics) GetUsageByCountry(country string) int {
	if count, ok := ls.UsageByCountry[country]; ok {
		return count
	}
	return 0
}

// GetUsageByDevice 获取指定设备类型的使用量
func (ls *LicenseStatistics) GetUsageByDevice(device string) int {
	if count, ok := ls.UsageByDevice[device]; ok {
		return count
	}
	return 0
}

// GetDailyUsageByDate 获取指定日期的使用统计
func (ls *LicenseStatistics) GetDailyUsageByDate(date time.Time) *DailyUsage {
	for _, usage := range ls.DailyUsage {
		if usage.Date.Year() == date.Year() &&
			usage.Date.Month() == date.Month() &&
			usage.Date.Day() == date.Day() {
			return &usage
		}
	}
	return nil
}
