package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dal "project/internal/dal"
	"project/internal/model"
	query "project/internal/query"
	"project/pkg/constant"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// Battery BMS: 电池管理（电池列表/导入导出等）
type Battery struct{}

type batteryListRow struct {
	DeviceID     string  `gorm:"column:device_id"`
	DeviceNumber string  `gorm:"column:device_number"`
	DeviceName   *string `gorm:"column:device_name"`

	BatteryModelID   *string `gorm:"column:battery_model_id"`
	BatteryModelName *string `gorm:"column:battery_model_name"`

	ProductionDate     *time.Time `gorm:"column:production_date"`
	WarrantyExpireDate *time.Time `gorm:"column:warranty_expire_date"`

	DealerID   *string `gorm:"column:dealer_id"`
	DealerName *string `gorm:"column:dealer_name"`

	UserID    *string `gorm:"column:user_id"`
	UserName  *string `gorm:"column:user_name"`
	UserPhone *string `gorm:"column:user_phone"`

	ActivationDate   *time.Time `gorm:"column:activation_date"`
	ActivationStatus *string    `gorm:"column:activation_status"`

	IsOnline       int16    `gorm:"column:is_online"`
	Soc            *float64 `gorm:"column:soc"`
	Soh            *float64 `gorm:"column:soh"`
	CurrentVersion *string  `gorm:"column:current_version"`
	TransferStatus *string  `gorm:"column:transfer_status"`
}

// GetBatteryList 获取电池列表（厂家/经销商视角）
func (*Battery) GetBatteryList(ctx context.Context, req model.BatteryListReq, claims *utils.UserClaims, dealerID string) (*model.BatteryListResp, error) {
	db := global.DB.WithContext(ctx)

	// 以 devices 作为 tenant 过滤主表
	queryBuilder := db.Table("devices AS d").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			d.name AS device_name,
			dbat.battery_model_id AS battery_model_id,
			bm.name AS battery_model_name,
			dbat.production_date AS production_date,
			dbat.warranty_expire_date AS warranty_expire_date,
			dbat.dealer_id AS dealer_id,
			de.name AS dealer_name,
			u.id AS user_id,
			u.name AS user_name,
			u.phone_number AS user_phone,
			dbat.activation_date AS activation_date,
			dbat.activation_status AS activation_status,
			d.is_online AS is_online,
			dbat.soc AS soc,
			dbat.soh AS soh,
			d.current_version AS current_version,
			dbat.transfer_status AS transfer_status
		`).
		Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id`).
		Joins(`LEFT JOIN battery_models AS bm ON bm.id = dbat.battery_model_id`).
		Joins(`LEFT JOIN dealers AS de ON de.id = dbat.dealer_id`).
		// 仅取主用户（is_owner=true），若无则为空
		Joins(`LEFT JOIN device_user_bindings AS dub ON dub.device_id = d.id AND dub.is_owner = true`).
		Joins(`LEFT JOIN users AS u ON u.id = dub.user_id`).
		Where("d.tenant_id = ?", claims.TenantID)

	// 经销商数据隔离：dealerID 不为空时只看名下设备
	if dealerID != "" {
		queryBuilder = queryBuilder.Where("dbat.dealer_id = ?", dealerID)
	}

	// 条件筛选
	if req.DeviceNumber != nil && *req.DeviceNumber != "" {
		queryBuilder = queryBuilder.Where("d.device_number ILIKE ?", "%"+*req.DeviceNumber+"%")
	}
	if req.BatteryModelID != nil && *req.BatteryModelID != "" {
		queryBuilder = queryBuilder.Where("dbat.battery_model_id = ?", *req.BatteryModelID)
	}
	if req.IsOnline != nil {
		queryBuilder = queryBuilder.Where("d.is_online = ?", *req.IsOnline)
	}
	if req.ActivationStatus != nil && *req.ActivationStatus != "" {
		queryBuilder = queryBuilder.Where("dbat.activation_status = ?", *req.ActivationStatus)
	}
	if req.DealerID != nil && *req.DealerID != "" {
		// 厂家侧可按 dealer_id 过滤；经销商侧该条件与 dealerID 一致/更严
		queryBuilder = queryBuilder.Where("dbat.dealer_id = ?", *req.DealerID)
	}

	// 出厂日期范围（YYYY-MM-DD）
	if req.ProductionDateStart != nil && *req.ProductionDateStart != "" {
		if t, err := time.ParseInLocation("2006-01-02", *req.ProductionDateStart, time.Local); err == nil {
			queryBuilder = queryBuilder.Where("dbat.production_date >= ?", t)
		}
	}
	if req.ProductionDateEnd != nil && *req.ProductionDateEnd != "" {
		if t, err := time.ParseInLocation("2006-01-02", *req.ProductionDateEnd, time.Local); err == nil {
			// end-of-day
			queryBuilder = queryBuilder.Where("dbat.production_date < ?", t.Add(24*time.Hour))
		}
	}

	// 质保状态（IN/OVER）
	if req.WarrantyStatus != nil && *req.WarrantyStatus != "" {
		now := time.Now()
		switch *req.WarrantyStatus {
		case "IN":
			queryBuilder = queryBuilder.Where("dbat.warranty_expire_date IS NOT NULL AND dbat.warranty_expire_date >= ?", now)
		case "OVER":
			queryBuilder = queryBuilder.Where("dbat.warranty_expire_date IS NOT NULL AND dbat.warranty_expire_date < ?", now)
		}
	}

	// 统计总数
	var total int64
	if err := queryBuilder.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 分页
	offset := (req.Page - 1) * req.PageSize
	rows := make([]batteryListRow, 0, req.PageSize)
	if err := queryBuilder.
		Order("d.created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 构建响应
	list := make([]model.BatteryListItemResp, 0, len(rows))
	for _, r := range rows {
		item := model.BatteryListItemResp{
			DeviceID:         r.DeviceID,
			DeviceNumber:     r.DeviceNumber,
			DeviceName:       r.DeviceName,
			BatteryModelID:   r.BatteryModelID,
			BatteryModelName: r.BatteryModelName,
			DealerID:         r.DealerID,
			DealerName:       r.DealerName,
			UserID:           r.UserID,
			UserName:         r.UserName,
			UserPhone:        r.UserPhone,
			ActivationStatus: r.ActivationStatus,
			IsOnline:         r.IsOnline,
			Soc:              r.Soc,
			Soh:              r.Soh,
			CurrentVersion:   r.CurrentVersion,
			TransferStatus:   r.TransferStatus,
		}

		if r.ProductionDate != nil {
			s := r.ProductionDate.Format("2006-01-02")
			item.ProductionDate = &s
		}
		if r.WarrantyExpireDate != nil {
			s := r.WarrantyExpireDate.Format("2006-01-02")
			item.WarrantyExpireDate = &s
		}
		if r.ActivationDate != nil {
			s := r.ActivationDate.Format("2006-01-02 15:04:05")
			item.ActivationDate = &s
		}

		list = append(list, item)
	}

	return &model.BatteryListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// buildBatteryQuery 构建电池查询（复用逻辑）
func buildBatteryQuery(ctx context.Context, req model.BatteryExportReq, claims *utils.UserClaims, dealerID string) *gorm.DB {
	db := global.DB.WithContext(ctx)

	queryBuilder := db.Table("devices AS d").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			d.name AS device_name,
			dbat.battery_model_id AS battery_model_id,
			bm.name AS battery_model_name,
			dbat.production_date AS production_date,
			dbat.warranty_expire_date AS warranty_expire_date,
			dbat.dealer_id AS dealer_id,
			de.name AS dealer_name,
			u.id AS user_id,
			u.name AS user_name,
			u.phone_number AS user_phone,
			dbat.activation_date AS activation_date,
			dbat.activation_status AS activation_status,
			d.is_online AS is_online,
			dbat.soc AS soc,
			dbat.soh AS soh,
			d.current_version AS current_version,
			dbat.transfer_status AS transfer_status
		`).
		Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id`).
		Joins(`LEFT JOIN battery_models AS bm ON bm.id = dbat.battery_model_id`).
		Joins(`LEFT JOIN dealers AS de ON de.id = dbat.dealer_id`).
		Joins(`LEFT JOIN device_user_bindings AS dub ON dub.device_id = d.id AND dub.is_owner = true`).
		Joins(`LEFT JOIN users AS u ON u.id = dub.user_id`).
		Where("d.tenant_id = ?", claims.TenantID)

	// 经销商数据隔离
	if dealerID != "" {
		queryBuilder = queryBuilder.Where("dbat.dealer_id = ?", dealerID)
	}

	// 条件筛选
	if req.DeviceNumber != nil && *req.DeviceNumber != "" {
		queryBuilder = queryBuilder.Where("d.device_number ILIKE ?", "%"+*req.DeviceNumber+"%")
	}
	if req.BatteryModelID != nil && *req.BatteryModelID != "" {
		queryBuilder = queryBuilder.Where("dbat.battery_model_id = ?", *req.BatteryModelID)
	}
	if req.IsOnline != nil {
		queryBuilder = queryBuilder.Where("d.is_online = ?", *req.IsOnline)
	}
	if req.ActivationStatus != nil && *req.ActivationStatus != "" {
		queryBuilder = queryBuilder.Where("dbat.activation_status = ?", *req.ActivationStatus)
	}
	if req.DealerID != nil && *req.DealerID != "" {
		queryBuilder = queryBuilder.Where("dbat.dealer_id = ?", *req.DealerID)
	}

	// 出厂日期范围
	if req.ProductionDateStart != nil && *req.ProductionDateStart != "" {
		if t, err := time.ParseInLocation("2006-01-02", *req.ProductionDateStart, time.Local); err == nil {
			queryBuilder = queryBuilder.Where("dbat.production_date >= ?", t)
		}
	}
	if req.ProductionDateEnd != nil && *req.ProductionDateEnd != "" {
		if t, err := time.ParseInLocation("2006-01-02", *req.ProductionDateEnd, time.Local); err == nil {
			queryBuilder = queryBuilder.Where("dbat.production_date < ?", t.Add(24*time.Hour))
		}
	}

	// 质保状态
	if req.WarrantyStatus != nil && *req.WarrantyStatus != "" {
		now := time.Now()
		switch *req.WarrantyStatus {
		case "IN":
			queryBuilder = queryBuilder.Where("dbat.warranty_expire_date IS NOT NULL AND dbat.warranty_expire_date >= ?", now)
		case "OVER":
			queryBuilder = queryBuilder.Where("dbat.warranty_expire_date IS NOT NULL AND dbat.warranty_expire_date < ?", now)
		}
	}

	return queryBuilder
}

// ExportBatteryList 导出电池列表（Excel）
func (*Battery) ExportBatteryList(ctx context.Context, req model.BatteryExportReq, claims *utils.UserClaims, dealerID string) (string, error) {
	queryBuilder := buildBatteryQuery(ctx, req, claims, dealerID)

	// 限制导出数量（防止内存溢出）
	const maxExportLimit = 50000
	rows := make([]batteryListRow, 0)
	if err := queryBuilder.
		Order("d.created_at DESC").
		Limit(maxExportLimit).
		Scan(&rows).Error; err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	if len(rows) == 0 {
		return "", errcode.New(errcode.CodeParamError)
	}

	// 创建 Excel 文件
	f := excelize.NewFile()
	sheetName := "Sheet1"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{"序列号", "设备名称", "电池型号", "出厂日期", "质保到期", "经销商", "终端用户", "用户电话", "激活状态", "激活时间", "在线状态", "SOC(%)", "SOH(%)", "固件版本", "流转状态"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, h)
	}

	// 写入数据
	for i, r := range rows {
		rowNum := i + 2
		col := 0

		setCell := func(val interface{}) {
			cell := fmt.Sprintf("%c%d", 'A'+col, rowNum)
			f.SetCellValue(sheetName, cell, val)
			col++
		}

		setCell(r.DeviceNumber)
		if r.DeviceName != nil {
			setCell(*r.DeviceName)
		} else {
			setCell("")
		}
		if r.BatteryModelName != nil {
			setCell(*r.BatteryModelName)
		} else {
			setCell("")
		}
		if r.ProductionDate != nil {
			setCell(r.ProductionDate.Format("2006-01-02"))
		} else {
			setCell("")
		}
		if r.WarrantyExpireDate != nil {
			setCell(r.WarrantyExpireDate.Format("2006-01-02"))
		} else {
			setCell("")
		}
		if r.DealerName != nil {
			setCell(*r.DealerName)
		} else {
			setCell("厂家")
		}
		if r.UserName != nil {
			setCell(*r.UserName)
		} else {
			setCell("")
		}
		if r.UserPhone != nil {
			setCell(*r.UserPhone)
		} else {
			setCell("")
		}
		if r.ActivationStatus != nil {
			if *r.ActivationStatus == "ACTIVE" {
				setCell("已激活")
			} else {
				setCell("未激活")
			}
		} else {
			setCell("未激活")
		}
		if r.ActivationDate != nil {
			setCell(r.ActivationDate.Format("2006-01-02 15:04:05"))
		} else {
			setCell("")
		}
		if r.IsOnline == 1 {
			setCell("在线")
		} else {
			setCell("离线")
		}
		if r.Soc != nil {
			setCell(*r.Soc)
		} else {
			setCell("")
		}
		if r.Soh != nil {
			setCell(*r.Soh)
		} else {
			setCell("")
		}
		if r.CurrentVersion != nil {
			setCell(*r.CurrentVersion)
		} else {
			setCell("")
		}
		if r.TransferStatus != nil {
			setCell(*r.TransferStatus)
		} else {
			setCell("FACTORY")
		}
	}

	// 保存文件
	uploadDir := "./files/excel/"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return "", errcode.WithVars(errcode.CodeFilePathGenError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	filename := fmt.Sprintf("电池列表_%s.xlsx", time.Now().Format("20060102150405"))
	filePath := filepath.Join(uploadDir, filename)

	if err := f.SaveAs(filePath); err != nil {
		return "", errcode.WithVars(errcode.CodeFileSaveError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return filePath, nil
}

// GetBatteryImportTemplate 获取导入模板（Excel）
func (*Battery) GetBatteryImportTemplate() (string, error) {
	f := excelize.NewFile()
	sheetName := "Sheet1"
	index, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{"序列号*", "电池型号ID", "出厂日期(YYYY-MM-DD)", "质保到期(YYYY-MM-DD)", "批次号", "经销商ID"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, h)
	}

	// 添加示例数据（第2行）
	example := []string{"DEVICE001", "", "2024-01-01", "2027-01-01", "BATCH001", ""}
	for i, val := range example {
		cell := fmt.Sprintf("%c2", 'A'+i)
		f.SetCellValue(sheetName, cell, val)
	}

	// 保存文件
	uploadDir := "./files/excel/"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return "", errcode.WithVars(errcode.CodeFilePathGenError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	filename := "电池导入模板.xlsx"
	filePath := filepath.Join(uploadDir, filename)

	if err := f.SaveAs(filePath); err != nil {
		return "", errcode.WithVars(errcode.CodeFileSaveError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return filePath, nil
}

// ImportBatteryList 导入电池列表（Excel）
func (*Battery) ImportBatteryList(ctx context.Context, filePath string, claims *utils.UserClaims, dealerID string) (*model.BatteryImportResp, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, errcode.WithVars(errcode.CodeFileSaveError, map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Error("close excel file error:", err)
		}
	}()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, errcode.WithVars(errcode.CodeFileSaveError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	if len(rows) < 2 {
		return nil, errcode.New(errcode.CodeParamError)
	}

	// 跳过表头，从第2行开始
	resp := &model.BatteryImportResp{
		Failures: make([]model.BatteryImportFailure, 0),
	}

	db := global.DB.WithContext(ctx)
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		resp.Total++

		// 解析行数据：序列号*, 电池型号ID, 出厂日期, 质保到期, 批次号, 经销商ID
		if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
			resp.Failed++
			resp.Failures = append(resp.Failures, model.BatteryImportFailure{
				Row:          i + 1,
				DeviceNumber: nil,
				Message:      "序列号不能为空",
			})
			continue
		}

		deviceNumber := strings.TrimSpace(row[0])
		var batteryModelID *string
		if len(row) > 1 && strings.TrimSpace(row[1]) != "" {
			s := strings.TrimSpace(row[1])
			batteryModelID = &s
		}

		var productionDate *time.Time
		if len(row) > 2 && strings.TrimSpace(row[2]) != "" {
			if t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(row[2]), time.Local); err == nil {
				productionDate = &t
			} else {
				resp.Failed++
				resp.Failures = append(resp.Failures, model.BatteryImportFailure{
					Row:          i + 1,
					DeviceNumber: &deviceNumber,
					Message:      "出厂日期格式错误，应为 YYYY-MM-DD",
				})
				continue
			}
		}

		var warrantyExpireDate *time.Time
		if len(row) > 3 && strings.TrimSpace(row[3]) != "" {
			if t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(row[3]), time.Local); err == nil {
				warrantyExpireDate = &t
			} else {
				resp.Failed++
				resp.Failures = append(resp.Failures, model.BatteryImportFailure{
					Row:          i + 1,
					DeviceNumber: &deviceNumber,
					Message:      "质保到期日期格式错误，应为 YYYY-MM-DD",
				})
				continue
			}
		}

		var batchNumber *string
		if len(row) > 4 && strings.TrimSpace(row[4]) != "" {
			s := strings.TrimSpace(row[4])
			batchNumber = &s
		}

		var importDealerID *string
		if len(row) > 5 && strings.TrimSpace(row[5]) != "" {
			s := strings.TrimSpace(row[5])
			importDealerID = &s
		} else if dealerID != "" {
			// 经销商视角导入时，默认使用当前经销商ID
			importDealerID = &dealerID
		}

		// 查找设备
		device, err := dal.GetDeviceByDeviceNumber(deviceNumber)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				resp.Failed++
				resp.Failures = append(resp.Failures, model.BatteryImportFailure{
					Row:          i + 1,
					DeviceNumber: &deviceNumber,
					Message:      "设备不存在",
				})
				continue
			}
			resp.Failed++
			resp.Failures = append(resp.Failures, model.BatteryImportFailure{
				Row:          i + 1,
				DeviceNumber: &deviceNumber,
				Message:      "查询设备失败: " + err.Error(),
			})
			continue
		}

		// 校验租户
		if device.TenantID != claims.TenantID {
			resp.Failed++
			resp.Failures = append(resp.Failures, model.BatteryImportFailure{
				Row:          i + 1,
				DeviceNumber: &deviceNumber,
				Message:      "设备不属于当前租户",
			})
			continue
		}

		// 查找或创建 device_batteries 记录
		_, err = query.DeviceBattery.WithContext(ctx).
			Where(query.DeviceBattery.DeviceID.Eq(device.ID)).
			First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 创建新记录
				now := time.Now()
				newDeviceBattery := &model.DeviceBattery{
					DeviceID:           device.ID,
					BatteryModelID:     batteryModelID,
					DealerID:           importDealerID,
					ProductionDate:     productionDate,
					WarrantyExpireDate: warrantyExpireDate,
					BatchNumber:        batchNumber,
					ActivationStatus:   StringPtr("INACTIVE"),
					TransferStatus:     StringPtr("FACTORY"),
					UpdatedAt:          &now,
				}
				if err := tx.Create(newDeviceBattery).Error; err != nil {
					tx.Rollback()
					return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
						"sql_error": err.Error(),
					})
				}
			} else {
				resp.Failed++
				resp.Failures = append(resp.Failures, model.BatteryImportFailure{
					Row:          i + 1,
					DeviceNumber: &deviceNumber,
					Message:      "查询电池信息失败: " + err.Error(),
				})
				continue
			}
		} else {
			// 更新现有记录
			updates := make(map[string]interface{})
			if batteryModelID != nil {
				updates["battery_model_id"] = *batteryModelID
			}
			if importDealerID != nil {
				updates["dealer_id"] = *importDealerID
			}
			if productionDate != nil {
				updates["production_date"] = *productionDate
			}
			if warrantyExpireDate != nil {
				updates["warranty_expire_date"] = *warrantyExpireDate
			}
			if batchNumber != nil {
				updates["batch_number"] = *batchNumber
			}
			updates["updated_at"] = time.Now()

			if err := tx.Model(&model.DeviceBattery{}).
				Where("device_id = ?", device.ID).
				Updates(updates).Error; err != nil {
				tx.Rollback()
				return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"sql_error": err.Error(),
				})
			}
		}

		resp.Success++
	}

	if err := tx.Commit().Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return resp, nil
}

// BatchAssignDealer 批量分配经销商
func (*Battery) BatchAssignDealer(ctx context.Context, req model.BatteryBatchAssignDealerReq, claims *utils.UserClaims, dealerID string) error {
	db := global.DB.WithContext(ctx)
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	for _, deviceID := range req.DeviceIDs {
		// 校验设备是否存在且属于当前租户
		_, err := query.Device.WithContext(ctx).
			Where(query.Device.ID.Eq(deviceID), query.Device.TenantID.Eq(claims.TenantID)).
			First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				tx.Rollback()
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "设备不存在: " + deviceID,
				})
			}
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}

		// 经销商数据隔离：经销商只能操作自己名下的设备
		if dealerID != "" {
			existingBattery, err := query.DeviceBattery.WithContext(ctx).
				Where(query.DeviceBattery.DeviceID.Eq(deviceID)).
				First()
			if err == nil && existingBattery.DealerID != nil && *existingBattery.DealerID != dealerID {
				tx.Rollback()
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "无权操作该设备: " + deviceID,
				})
			}
		}

		// 查找或创建 device_batteries 记录
		_, err = query.DeviceBattery.WithContext(ctx).
			Where(query.DeviceBattery.DeviceID.Eq(deviceID)).
			First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 创建新记录
				newBattery := &model.DeviceBattery{
					DeviceID:         deviceID,
					DealerID:         &req.DealerID,
					ActivationStatus: StringPtr("INACTIVE"),
					TransferStatus:   StringPtr("DEALER"),
					UpdatedAt:        &now,
				}
				if err := tx.Create(newBattery).Error; err != nil {
					tx.Rollback()
					return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
						"sql_error": err.Error(),
					})
				}
			} else {
				tx.Rollback()
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"sql_error": err.Error(),
				})
			}
		} else {
			// 更新经销商ID
			if err := tx.Model(&model.DeviceBattery{}).
				Where("device_id = ?", deviceID).
				Updates(map[string]interface{}{
					"dealer_id":       req.DealerID,
					"updated_at":      now,
					"transfer_status": "DEALER",
				}).Error; err != nil {
				tx.Rollback()
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"sql_error": err.Error(),
				})
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return nil
}

// BatchSendCommand 批量下发指令（在线设备）
func (*Battery) BatchSendCommand(ctx context.Context, req model.BatteryBatchCommandReq, claims *utils.UserClaims, dealerID string) (*model.BatteryBatchCommandResp, error) {
	resp := &model.BatteryBatchCommandResp{
		Total:    len(req.DeviceIDs),
		Success:  0,
		Failed:   0,
		Failures: make([]model.BatteryBatchCommandFailure, 0),
	}

	for _, deviceID := range req.DeviceIDs {
		// 校验设备是否存在且属于当前租户
		device, err := query.Device.WithContext(ctx).
			Where(query.Device.ID.Eq(deviceID), query.Device.TenantID.Eq(claims.TenantID)).
			First()
		if err != nil {
			resp.Failed++
			resp.Failures = append(resp.Failures, model.BatteryBatchCommandFailure{
				DeviceID:     deviceID,
				DeviceNumber: "",
				Message:      "设备不存在或无权限",
			})
			continue
		}

		// 经销商隔离
		if dealerID != "" {
			existingBattery, err := query.DeviceBattery.WithContext(ctx).
				Where(query.DeviceBattery.DeviceID.Eq(deviceID)).
				First()
			if err == nil && existingBattery.DealerID != nil && *existingBattery.DealerID != dealerID {
				resp.Failed++
				resp.Failures = append(resp.Failures, model.BatteryBatchCommandFailure{
					DeviceID:     deviceID,
					DeviceNumber: device.DeviceNumber,
					Message:      "无权操作该设备",
				})
				continue
			}
		}

		// 仅在线设备允许立即下发
		if device.IsOnline != 1 {
			resp.Failed++
			resp.Failures = append(resp.Failures, model.BatteryBatchCommandFailure{
				DeviceID:     deviceID,
				DeviceNumber: device.DeviceNumber,
				Message:      "设备离线（请使用离线指令）",
			})
			continue
		}

		put := &model.PutMessageForCommand{
			DeviceID: deviceID,
			Value:    req.Value,
			Identify: req.Identify,
		}

		_, err = GroupApp.CommandData.CommandPutMessageReturnMessageID(ctx, claims.ID, put, strconv.Itoa(constant.Manual))
		if err != nil {
			resp.Failed++
			resp.Failures = append(resp.Failures, model.BatteryBatchCommandFailure{
				DeviceID:     deviceID,
				DeviceNumber: device.DeviceNumber,
				Message:      err.Error(),
			})
			continue
		}

		resp.Success++
	}

	return resp, nil
}

// BatchPushOTA 批量 OTA 推送（创建 OTA 任务并触发推送）
func (*Battery) BatchPushOTA(ctx context.Context, req model.BatteryBatchOtaPushReq, claims *utils.UserClaims, dealerID string) (*model.BatteryBatchOtaPushResp, error) {
	// 校验升级包归属当前租户
	pkg, err := query.OtaUpgradePackage.WithContext(ctx).
		Where(query.OtaUpgradePackage.ID.Eq(req.OTAUpgradePackageID), query.OtaUpgradePackage.TenantID.Eq(claims.TenantID)).
		First()
	if err != nil || pkg == nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "升级包不存在或无权限"})
	}

	accepted := make([]string, 0, len(req.DeviceIDs))
	failures := make([]model.BatteryBatchOtaPushFailure, 0)

	for _, deviceID := range req.DeviceIDs {
		device, err := query.Device.WithContext(ctx).
			Where(query.Device.ID.Eq(deviceID), query.Device.TenantID.Eq(claims.TenantID)).
			First()
		if err != nil || device == nil {
			failures = append(failures, model.BatteryBatchOtaPushFailure{
				DeviceID:     deviceID,
				DeviceNumber: "",
				Message:      "设备不存在或无权限",
			})
			continue
		}

		// 经销商隔离
		if dealerID != "" {
			existingBattery, err := query.DeviceBattery.WithContext(ctx).
				Where(query.DeviceBattery.DeviceID.Eq(deviceID)).
				First()
			if err == nil && existingBattery.DealerID != nil && *existingBattery.DealerID != dealerID {
				failures = append(failures, model.BatteryBatchOtaPushFailure{
					DeviceID:     deviceID,
					DeviceNumber: device.DeviceNumber,
					Message:      "无权操作该设备",
				})
				continue
			}
		}

		accepted = append(accepted, deviceID)
	}

	if len(accepted) == 0 {
		return &model.BatteryBatchOtaPushResp{
			TaskID:   "",
			Total:    len(req.DeviceIDs),
			Accepted: 0,
			Rejected: len(failures),
			Failures: failures,
		}, nil
	}

	// 生成任务名称
	taskName := ""
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		taskName = strings.TrimSpace(*req.Name)
	} else {
		taskName = fmt.Sprintf("BMS批量OTA_%s_%s", pkg.Version, time.Now().In(time.Local).Format("20060102150405"))
	}

	createReq := &model.CreateOTAUpgradeTaskReq{
		Name:                taskName,
		OTAUpgradePackageId: req.OTAUpgradePackageID,
		Description:         req.Description,
		Remark:              req.Remark,
		DeviceIdList:        accepted,
	}

	// 用 DAL 创建任务以拿到 task_id，然后逐个触发推送
	taskDetails, err := dal.CreateOTAUpgradeTaskWithDetail(createReq)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	taskID := ""
	if len(taskDetails) > 0 {
		taskID = taskDetails[0].OtaUpgradeTaskID
	}
	go func(details []*model.OtaUpgradeTaskDetail) {
		for _, d := range details {
			_ = GroupApp.OTA.PushOTAUpgradePackage(d)
		}
	}(taskDetails)

	return &model.BatteryBatchOtaPushResp{
		TaskID:   taskID,
		Total:    len(req.DeviceIDs),
		Accepted: len(accepted),
		Rejected: len(failures),
		Failures: failures,
	}, nil
}
