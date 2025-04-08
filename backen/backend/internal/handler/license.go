package handler

import (
	"license-management-system/internal/database"
	"license-management-system/internal/model"
	"license-management-system/internal/service"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

var sheetSync *service.SheetSyncService

func InitSheetSync(enableSync bool, credentialPath, spreadsheetID, sheetName string) (*service.SheetSyncService, error) {
	var err error
	sheetSync, err = service.NewSheetSyncService(enableSync, credentialPath, spreadsheetID, sheetName)
	return sheetSync, err
}

// HandleGetAllLicenses 管理员获取所有许可证数据
func HandleGetAllLicenses(c *fiber.Ctx) error {
	// TODO: 添加管理员权限验证
	// 这里需要实现管理员权限验证逻辑

	var licenses []model.License
	result := database.DB.Find(&licenses)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取许可证数据失败",
		})
	}

	return c.JSON(fiber.Map{
		"licenses": licenses,
	})
}

func HandleLicenseGenerate(c *fiber.Ctx) error {
	input := new(model.LicenseInput)

	// 生成唯一的许可证密钥
	key := generateLicenseKey()

	license := &model.License{
		Key:         key,
		Status:      "inactive",
		ValidUntil:  time.Now().AddDate(0, 0, 30),
		Version:     input.Version,
		Permissions: input.Permissions,
		UserId:      input.UserId,
		ProductId:   input.ProductId,

		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LastActivatedAt: time.Now(),
	}

	return c.Status(fiber.StatusCreated).JSON(license)
}

func HandleLicenseIssue(c *fiber.Ctx) error {
	type IssueInput struct {
		LicenseKey string `json:"license_key"`
		UserID     uint   `json:"user_id"`
	}

	input := new(IssueInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的输入数据",
		})
	}

	var license model.License
	result := database.DB.Where("key = ?", input.LicenseKey).First(&license)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "许可证不存在",
		})
	}

	license.IssuedTo = input.UserID
	license.UpdatedAt = time.Now()

	database.DB.Save(&license)

	return c.JSON(license)
}

func HandleLicenseVerify(c *fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "许可证密钥不能为空",
		})
	}

	userid := c.Query("userid")
	if userid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "用户名不能为空",
		})
	}

	productid := c.Query("productid")
	if productid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "策略名称不能为空",
		})
	}

	var license model.License
	result := database.DB.Where("key = ?", key).First(&license)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "许可证不存在",
		})
	}

	if license.UserId != userid || license.ProductId != productid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "用户名或策略名称不匹配",
		})
	}

	isValid := license.Status != "已吊销" && time.Now().Before(license.ValidUntil)
	usage := model.LicenseUsage{
		LicenseKey: key,
		Action:     "unkown",
		IPAddress:  c.IP(),
		UserAgent:  c.Get("User-Agent"),

		Timestamp: time.Now(),
	}

	usage.Action = "verify license " + strconv.FormatBool(isValid)
	usage.Timestamp = time.Now()

	database.DB.Create(&usage)

	return c.JSON(fiber.Map{
		"valid":  isValid,
		"status": license.Status,
	})
}

// HandleLicenseUsage 查询license使用记录
func HandleLicenseUsage(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "许可证密钥不能为空",
		})
	}

	var usages []model.LicenseUsage
	result := database.DB.Where("license_key = ?", key).Order("timestamp desc").Limit(20).Find(&usages)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询使用记录失败",
		})
	}

	return c.JSON(fiber.Map{
		"usages": usages,
	})
}

func HandleLicenseActivate(c *fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "许可证密钥不能为空",
		})
	}

	var license model.License
	result := database.DB.Where("key = ?", key).First(&license)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "许可证不存在",
		})
	}

	if license.Status == "已激活" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "许可证已经激活",
		})
	}

	license.Status = "已激活"
	license.UpdatedAt = time.Now()

	database.DB.Save(&license)

	if sheetSync != nil {
		go sheetSync.SyncLicense(&license)
	}

	return c.JSON(license)
}

// HandleLicenseUpdate 更新许可证信息
func HandleLicenseUpdate(c *fiber.Ctx) error {
	// 获取许可证密钥
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "许可证密钥不能为空",
		})
	}

	// 定义更新许可证的输入结构
	type UpdateInput struct {
		Status      string `json:"status"`
		ValidUntil  string `json:"validuntil"`
		Version     string `json:"version"`
		Permissions string `json:"permissions"`
		UserId      string `json:"userid"`
		ProductId   string `json:"productid"`
	}

	// 解析请求体
	input := new(UpdateInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的输入数据",
		})
	}

	// 查找许可证
	var license model.License
	result := database.DB.Where("key = ?", key).First(&license)
	if result.Error != nil {
		license = model.License{
			Key:             key,
			Status:          "active",
			ValidUntil:      time.Now().AddDate(0, 0, 3650),
			Version:         input.Version,
			Permissions:     input.Permissions,
			UserId:          input.UserId,
			ProductId:       input.ProductId,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			LastActivatedAt: time.Now(),
		}
	}

	// 更新许可证信息
	if input.Status != "" {
		license.Status = input.Status
	}
	if input.ValidUntil != "" {
		parsedTime, err := time.Parse("2006-01-02T15:04:05.000", input.ValidUntil)
		if err == nil {
			license.ValidUntil = parsedTime
		}
	}
	if input.Version != "" {
		license.Version = input.Version
	}
	if input.Permissions != "" {
		license.Permissions = input.Permissions
	}
	if input.UserId != "" {
		license.UserId = input.UserId
	}
	if input.ProductId != "" {
		license.ProductId = input.ProductId
	}

	// 更新时间戳
	license.UpdatedAt = time.Now()

	// 保存更新
	result = database.DB.Save(&license)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "更新许可证失败",
		})
	}

	if sheetSync != nil {
		go sheetSync.SyncLicense(&license)
	}

	return c.JSON(fiber.Map{
		"message": "许可证更新成功",
		"license": license,
	})
}

// HandleLicenseDelete 删除许可证
func HandleLicenseDelete(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "许可证密钥不能为空",
		})
	}

	var license model.License
	result := database.DB.Where("key = ?", key).First(&license)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "许可证不存在",
		})
	}

	result = database.DB.Delete(&license)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "删除许可证失败",
		})
	}

	return c.JSON(fiber.Map{
		"message": "许可证删除成功",
	})
}

// HandleGetLicense 获取单个许可证详情
func HandleGetLicense(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "许可证密钥不能为空",
		})
	}

	var license model.License
	result := database.DB.Where("key = ?", key).First(&license)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "许可证不存在",
		})
	}

	return c.JSON(license)
}

// 生成唯一的许可证密钥
func generateLicenseKey() string {
	// 这里应该实现一个更复杂的密钥生成算法
	return time.Now().Format("20060102150405")
}
