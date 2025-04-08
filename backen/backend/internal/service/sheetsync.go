package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"license-management-system/internal/database"
	"license-management-system/internal/model"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"gorm.io/gorm"
)

type SheetSyncService struct {
	service       *sheets.Service
	spreadsheetID string
	sheetName     string
}

func NewSheetSyncService(enableSync bool, credentialPath, spreadsheetID, sheetName string) (*SheetSyncService, error) {
	if !enableSync {
		return nil, nil
	}

	ctx := context.Background()

	// 读取凭证文件
	b, err := os.ReadFile(credentialPath)
	if err != nil {
		return nil, err
	}

	// 使用服务账号授权
	creds, err := google.CredentialsFromJSON(ctx, b, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("无法加载凭证: %v", err)
	}

	srv, err := sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}

	return &SheetSyncService{
		service:       srv,
		spreadsheetID: spreadsheetID,
		sheetName:     sheetName,
	}, nil
}

func (s *SheetSyncService) SyncLicense(license *model.License) error {
	if s == nil {
		return nil
	}

	// 1. 先处理数据库操作
	var existing model.License
	result := database.DB.Where("key = ?", license.Key).First(&existing)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// 记录不存在，创建新记录
			if err := database.DB.Create(license).Error; err != nil {
				log.Printf("创建新许可证记录失败: %v", err)
				return fmt.Errorf("创建新许可证记录失败: %v", err)
			}
		} else {
			log.Printf("查询许可证记录失败: %v", result.Error)
			return fmt.Errorf("查询许可证记录失败: %v", result.Error)
		}
	} else {
		// 更新现有记录
		if err := database.DB.Model(&existing).Updates(map[string]interface{}{
			"Status":          license.Status,
			"ValidUntil":      license.ValidUntil,
			"Version":         license.Version,
			"Permissions":     license.Permissions,
			"UserId":          license.UserId,
			"ProductId":       license.ProductId,
			"LastActivatedAt": license.LastActivatedAt,
			"UpdatedAt":       license.UpdatedAt,
		}).Error; err != nil {
			log.Printf("更新许可证记录失败: %v", err)
			return fmt.Errorf("更新许可证记录失败: %v", err)
		}
	}

	// 2. 处理Google Sheets同步
	// 先检查Sheet中是否已存在该Key
	rangeToSearch := fmt.Sprintf("%s!A2:A", s.sheetName)
	keyResp, err := s.service.Spreadsheets.Values.Get(s.spreadsheetID, rangeToSearch).Do()
	if err != nil {
		log.Printf("查询Sheet数据失败: %v", err)
		return fmt.Errorf("查询Sheet数据失败: %v", err)
	}

	var rowIndex int
	found := false
	for i, row := range keyResp.Values {
		if len(row) > 0 && row[0] == license.Key {
			found = true
			rowIndex = i + 2 // +2因为A2开始且数组从0开始
			break
		}
	}

	// 检查工作表和范围
	log.Printf("正在检查Google Sheet: ID=%s, Sheet=%s", s.spreadsheetID, s.sheetName)

	// 先检查工作表是否存在
	spreadsheet, err := s.service.Spreadsheets.Get(s.spreadsheetID).Do()
	if err != nil {
		log.Printf("获取Spreadsheet信息失败: %+v", err)
		return fmt.Errorf("获取Spreadsheet信息失败: %v", err)
	}

	// 验证工作表是否存在
	sheetExists := false
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == s.sheetName {
			sheetExists = true
			break
		}
	}
	if !sheetExists {
		return fmt.Errorf("工作表'%s'不存在", s.sheetName)
	}

	// 检查范围访问
	_, err = s.service.Spreadsheets.Values.Get(s.spreadsheetID, fmt.Sprintf("'%s'!A1:I1", s.sheetName)).Do()
	if err != nil {
		log.Printf("Google Sheet访问错误详情: %+v", err)
		return fmt.Errorf("检查工作表范围失败: %v", err)
	}

	// 准备数据
	values := [][]interface{}{{
		license.Key,
		license.Status,
		license.ValidUntil.Format(time.RFC3339),
		license.UserId,
		license.ProductId,
		license.Version,
		license.LastActivatedAt.Format(time.RFC3339),
		license.CreatedAt.Format(time.RFC3339),
		license.UpdatedAt.Format(time.RFC3339),
	}}

	// 根据是否找到决定更新还是追加
	if found {
		// 更新现有行
		rangeData := fmt.Sprintf("%s!A%d:I%d", s.sheetName, rowIndex, rowIndex)
		_, err = s.service.Spreadsheets.Values.Update(
			s.spreadsheetID,
			rangeData,
			&sheets.ValueRange{Values: values},
		).ValueInputOption("USER_ENTERED").Do()
	} else {
		// 追加新行
		_, err = s.service.Spreadsheets.Values.Append(
			s.spreadsheetID,
			s.sheetName+"!A2:I",
			&sheets.ValueRange{Values: values},
		).ValueInputOption("USER_ENTERED").Do()
	}

	if err != nil {
		log.Printf("同步到Google Sheet失败: %v", err)
		return fmt.Errorf("同步到Google Sheet失败: %v", err)
	}

	log.Printf("成功同步许可证 %s 到数据库和Google Sheet", license.Key)
	return nil
}

func (s *SheetSyncService) BatchSyncLicenses(licenses []*model.License) error {
	if s == nil {
		return nil
	}

	var values [][]interface{}
	for _, license := range licenses {
		values = append(values, []interface{}{
			license.Key,
			license.Status,
			license.ValidUntil.Format(time.RFC3339),
			license.UserId,
			license.ProductId,
			license.Version,
			license.LastActivatedAt.Format(time.RFC3339),
			license.CreatedAt.Format(time.RFC3339),
			license.UpdatedAt.Format(time.RFC3339),
		})
	}

	rangeData := s.sheetName + "!A2:I"
	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err := s.service.Spreadsheets.Values.Append(
		s.spreadsheetID,
		rangeData,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		log.Printf("Failed to batch sync licenses: %v", err)
		return err
	}

	return nil
}

// SyncFromSheet 从Google Sheet读取数据并完全覆盖数据库
func (s *SheetSyncService) SyncFromSheet(db *gorm.DB) error {
	if s == nil {
		return nil
	}

	// 读取整个工作表数据
	resp, err := s.service.Spreadsheets.Values.Get(s.spreadsheetID, s.sheetName+"!A2:I").Do()
	if err != nil {
		return fmt.Errorf("读取工作表失败: %v", err)
	}

	// 使用事务确保数据一致性
	err = db.Transaction(func(tx *gorm.DB) error {
		// 1. 清空现有数据
		if err := tx.Where("1 = 1").Delete(&model.License{}).Error; err != nil {
			return fmt.Errorf("清空数据库失败: %v", err)
		}

		// 2. 批量插入Sheet数据
		var licenses []*model.License
		for i, row := range resp.Values {
			if len(row) < 7 {
				log.Printf("第%d行数据不完整，跳过", i+2)
				continue
			}

			// 解析数据
			license := &model.License{
				Key:       row[0].(string),
				Status:    row[1].(string),
				UserId:    row[3].(string),
				ProductId: row[4].(string),
				Version:   row[5].(string),
			}

			// 解析时间
			validUntil, err := time.Parse(time.RFC3339, row[2].(string))
			if err != nil {
				log.Printf("解析有效期时间失败(行%d): %v", i+2, err)
				continue
			}
			license.ValidUntil = validUntil

			lastActivated, err := time.Parse(time.RFC3339, row[6].(string))
			if err != nil {
				log.Printf("解析最后激活时间失败(行%d): %v", i+2, err)
				continue
			}
			license.LastActivatedAt = lastActivated
			createAt, err := time.Parse(time.RFC3339, row[7].(string))
			if err != nil {
				log.Printf("解析创建时间失败(行%d): %v", i+2, err)
				continue
			}
			license.CreatedAt = createAt

			updateAt, err := time.Parse(time.RFC3339, row[8].(string))
			if err != nil {
				log.Printf("解析更新时间失败(行%d): %v", i+2, err)
				continue
			}
			license.UpdatedAt = updateAt
			licenses = append(licenses, license)
		}

		// 批量创建记录
		if err := tx.CreateInBatches(licenses, 100).Error; err != nil {
			return fmt.Errorf("批量插入数据失败: %v", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("成功从Google Sheet同步%d条数据到数据库(完全覆盖)", len(resp.Values))
	return nil
}
