package service

import (
	"context"
	"time"

	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

type DeviceTransfer struct{}

// TransferDevicesToOrg 批量转移设备到指定组织（新版，基于 org）
func (*DeviceTransfer) TransferDevicesToOrg(ctx context.Context, req model.DeviceOrgTransferReq, claims *utils.UserClaims, operatorOrgID string) error {
	if len(req.DeviceIDs) == 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "device_ids cannot be empty",
		})
	}

	t := time.Now().UTC()

	// 开启事务
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 验证目标组织是否存在（如果不为空）
		var toOrgID *string
		if req.ToOrgID != nil && *req.ToOrgID != "" {
			var org model.Org
			if err := tx.Where("id = ? AND tenant_id = ?", *req.ToOrgID, claims.TenantID).First(&org).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
						"message": "目标组织不存在",
					})
				}
				return err
			}
			toOrgID = &org.ID
		}

		// 批量处理设备转移
		for _, deviceID := range req.DeviceIDs {
			// 查询设备是否存在
			var device model.Device
			if err := tx.Where("id = ? AND tenant_id = ?", deviceID, claims.TenantID).First(&device).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
						"message": "设备不存在: " + deviceID,
					})
				}
				return err
			}

			// 检查操作权限：操作者的组织必须能访问该设备
			if operatorOrgID != "" {
				var deviceBattery model.DeviceBattery
				if err := tx.Where("device_id = ?", deviceID).First(&deviceBattery).Error; err == nil {
					if deviceBattery.OwnerOrgID != nil && *deviceBattery.OwnerOrgID != "" {
						var count int64
						tx.Table("org_closure").
							Where("tenant_id = ? AND ancestor_id = ? AND descendant_id = ?",
								claims.TenantID, operatorOrgID, *deviceBattery.OwnerOrgID).
							Count(&count)
						if count == 0 {
							return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
								"message": "无权操作该设备: " + deviceID,
							})
						}
					}
				}
			}

			// 查询设备电池信息
			var deviceBattery model.DeviceBattery
			err := tx.Where("device_id = ?", deviceID).First(&deviceBattery).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					// 如果设备电池信息不存在，创建一条
					deviceBattery = model.DeviceBattery{
						DeviceID:   deviceID,
						OwnerOrgID: toOrgID,
						UpdatedAt:  &t,
					}
					if err := tx.Create(&deviceBattery).Error; err != nil {
						return err
					}
					// 记录转移日志
					transferLog := model.DeviceOrgTransfer{
						ID:           uuid.New(),
						DeviceID:     deviceID,
						FromOrgID:    nil,
						ToOrgID:      toOrgID,
						OperatorID:   &claims.ID,
						TransferTime: &t,
						Remark:       req.Remark,
						TenantID:     claims.TenantID,
						CreatedAt:    &t,
					}
					return tx.Create(&transferLog).Error
				}
				return err
			}

			// 记录原组织ID
			fromOrgID := deviceBattery.OwnerOrgID

			// 更新设备归属组织
			updates := map[string]interface{}{
				"owner_org_id": toOrgID,
				"updated_at":   t,
			}

			if err := tx.Model(&model.DeviceBattery{}).Where("device_id = ?", deviceID).Updates(updates).Error; err != nil {
				return err
			}

			// 记录转移日志到新表
			transferLog := model.DeviceOrgTransfer{
				ID:           uuid.New(),
				DeviceID:     deviceID,
				FromOrgID:    fromOrgID,
				ToOrgID:      toOrgID,
				OperatorID:   &claims.ID,
				TransferTime: &t,
				Remark:       req.Remark,
				TenantID:     claims.TenantID,
				CreatedAt:    &t,
			}

			if err := tx.Create(&transferLog).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if e, ok := err.(*errcode.Error); ok {
			return e
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return nil
}

// GetOrgTransferHistory 获取组织转移记录（新版）
func (*DeviceTransfer) GetOrgTransferHistory(ctx context.Context, req model.DeviceOrgTransferListReq, claims *utils.UserClaims) (*model.DeviceOrgTransferListResp, error) {
	db := global.DB.WithContext(ctx)

	// 构建查询
	queryBuilder := db.Table("device_org_transfers AS t").
		Select(`
			t.id, t.device_id, t.from_org_id, t.to_org_id, t.operator_id, t.transfer_time, t.remark,
			d.device_number, d.name AS device_name,
			fo.name AS from_org_name, fo.org_type AS from_org_type,
			tor.name AS to_org_name, tor.org_type AS to_org_type,
			u.name AS operator_name
		`).
		Joins("LEFT JOIN devices AS d ON d.id = t.device_id").
		Joins("LEFT JOIN orgs AS fo ON fo.id = t.from_org_id").
		Joins("LEFT JOIN orgs AS tor ON tor.id = t.to_org_id").
		Joins("LEFT JOIN users AS u ON u.id = t.operator_id").
		Where("t.tenant_id = ?", claims.TenantID)

	// 条件筛选
	if req.DeviceNumber != nil && *req.DeviceNumber != "" {
		queryBuilder = queryBuilder.Where("d.device_number ILIKE ?", "%"+*req.DeviceNumber+"%")
	}

	if req.FromOrgID != nil && *req.FromOrgID != "" {
		queryBuilder = queryBuilder.Where("t.from_org_id = ?", *req.FromOrgID)
	}

	if req.ToOrgID != nil && *req.ToOrgID != "" {
		queryBuilder = queryBuilder.Where("t.to_org_id = ?", *req.ToOrgID)
	}

	if req.StartTime != nil && *req.StartTime != "" {
		if startTime, err := time.Parse("2006-01-02 15:04:05", *req.StartTime); err == nil {
			queryBuilder = queryBuilder.Where("t.transfer_time >= ?", startTime)
		}
	}

	if req.EndTime != nil && *req.EndTime != "" {
		if endTime, err := time.Parse("2006-01-02 15:04:05", *req.EndTime); err == nil {
			queryBuilder = queryBuilder.Where("t.transfer_time <= ?", endTime)
		}
	}

	// 统计总数
	var total int64
	if err := queryBuilder.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 分页查询
	type transferRow struct {
		ID           string     `gorm:"column:id"`
		DeviceID     string     `gorm:"column:device_id"`
		FromOrgID    *string    `gorm:"column:from_org_id"`
		ToOrgID      *string    `gorm:"column:to_org_id"`
		OperatorID   *string    `gorm:"column:operator_id"`
		TransferTime *time.Time `gorm:"column:transfer_time"`
		Remark       *string    `gorm:"column:remark"`
		DeviceNumber string     `gorm:"column:device_number"`
		DeviceName   *string    `gorm:"column:device_name"`
		FromOrgName  *string    `gorm:"column:from_org_name"`
		FromOrgType  *string    `gorm:"column:from_org_type"`
		ToOrgName    *string    `gorm:"column:to_org_name"`
		ToOrgType    *string    `gorm:"column:to_org_type"`
		OperatorName *string    `gorm:"column:operator_name"`
	}

	offset := (req.Page - 1) * req.PageSize
	var rows []transferRow
	if err := queryBuilder.Order("t.transfer_time DESC").Offset(offset).Limit(req.PageSize).Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 构建响应
	list := make([]model.DeviceOrgTransferResp, 0, len(rows))
	for _, r := range rows {
		resp := model.DeviceOrgTransferResp{
			ID:           r.ID,
			DeviceID:     r.DeviceID,
			DeviceNumber: r.DeviceNumber,
			DeviceName:   r.DeviceName,
			FromOrgID:    r.FromOrgID,
			FromOrgName:  r.FromOrgName,
			FromOrgType:  r.FromOrgType,
			ToOrgID:      r.ToOrgID,
			ToOrgName:    r.ToOrgName,
			ToOrgType:    r.ToOrgType,
			OperatorID:   r.OperatorID,
			OperatorName: r.OperatorName,
			Remark:       r.Remark,
		}
		if r.TransferTime != nil {
			s := r.TransferTime.Format("2006-01-02 15:04:05")
			resp.TransferTime = &s
		}
		list = append(list, resp)
	}

	return &model.DeviceOrgTransferListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// TransferDevices 批量转移设备
func (*DeviceTransfer) TransferDevices(req model.DeviceTransferReq, claims *utils.UserClaims) error {
	if len(req.DeviceIDs) == 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "device_ids cannot be empty",
		})
	}

	// 开启事务
	tx := query.Use(global.DB).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	t := time.Now().UTC()

	// 验证目标经销商是否存在（如果不为空）
	var toDealerID *string
	if req.ToDealerID != nil && *req.ToDealerID != "" {
		dealer, err := tx.Dealer.Where(
			tx.Dealer.ID.Eq(*req.ToDealerID),
			tx.Dealer.TenantID.Eq(claims.TenantID),
		).First()
		if err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "target dealer not found",
				})
			}
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}
		toDealerID = &dealer.ID
	}

	// 批量处理设备转移
	for _, deviceID := range req.DeviceIDs {
		// 查询设备是否存在
		_, err := tx.Device.Where(
			tx.Device.ID.Eq(deviceID),
			tx.Device.TenantID.Eq(claims.TenantID),
		).First()
		if err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "device not found: " + deviceID,
				})
			}
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}

		// 查询设备电池信息
		deviceBattery, err := tx.DeviceBattery.Where(tx.DeviceBattery.DeviceID.Eq(deviceID)).First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 如果设备电池信息不存在，创建一条
				deviceBattery = &model.DeviceBattery{
					DeviceID:  deviceID,
					DealerID:  toDealerID,
					UpdatedAt: &t,
				}
				if err := tx.DeviceBattery.Create(deviceBattery); err != nil {
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
			// 记录原经销商ID
			fromDealerID := deviceBattery.DealerID

			// 更新设备归属经销商
			updates := map[string]interface{}{
				"dealer_id":  toDealerID,
				"updated_at": t,
			}

			// 如果转移给经销商，更新流转状态
			if toDealerID != nil {
				updates["transfer_status"] = "DEALER"
			} else {
				// 转移回厂家
				updates["transfer_status"] = "FACTORY"
			}

			if _, err := tx.DeviceBattery.Where(tx.DeviceBattery.DeviceID.Eq(deviceID)).Updates(updates); err != nil {
				tx.Rollback()
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"sql_error": err.Error(),
				})
			}

			// 记录转移日志
			transferLog := &model.DeviceTransfer{
				ID:           uuid.New(),
				DeviceID:     deviceID,
				FromDealerID: fromDealerID,
				ToDealerID:   toDealerID,
				OperatorID:   &claims.ID,
				TransferTime: &t,
				Remark:       req.Remark,
				TenantID:     claims.TenantID,
			}

			if err := tx.DeviceTransfer.Create(transferLog); err != nil {
				tx.Rollback()
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"sql_error": err.Error(),
				})
			}
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return nil
}

// GetTransferHistory 获取设备转移记录
func (*DeviceTransfer) GetTransferHistory(req model.DeviceTransferListReq, claims *utils.UserClaims) (*model.DeviceTransferListResp, error) {
	ctx := context.Background()
	q := query.DeviceTransfer
	d := query.Device
	dealer := query.Dealer
	user := query.User

	// 构建查询
	queryBuilder := q.WithContext(ctx).Where(q.TenantID.Eq(claims.TenantID))

	// 条件筛选
	if req.DeviceNumber != nil && *req.DeviceNumber != "" {
		// 通过设备编号查询设备ID
		devices, err := d.WithContext(ctx).Where(
			d.DeviceNumber.Like("%"+*req.DeviceNumber+"%"),
			d.TenantID.Eq(claims.TenantID),
		).Find()
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}
		deviceIDs := make([]string, 0, len(devices))
		for _, device := range devices {
			deviceIDs = append(deviceIDs, device.ID)
		}
		if len(deviceIDs) > 0 {
			queryBuilder = queryBuilder.Where(q.DeviceID.In(deviceIDs...))
		} else {
			// 没有匹配的设备，返回空列表
			return &model.DeviceTransferListResp{
				List:     []model.DeviceTransferResp{},
				Total:    0,
				Page:     req.Page,
				PageSize: req.PageSize,
			}, nil
		}
	}

	if req.FromDealerID != nil && *req.FromDealerID != "" {
		queryBuilder = queryBuilder.Where(q.FromDealerID.Eq(*req.FromDealerID))
	}

	if req.ToDealerID != nil && *req.ToDealerID != "" {
		queryBuilder = queryBuilder.Where(q.ToDealerID.Eq(*req.ToDealerID))
	}

	// 时间范围筛选
	if req.StartTime != nil && *req.StartTime != "" {
		startTime, err := time.Parse("2006-01-02 15:04:05", *req.StartTime)
		if err == nil {
			queryBuilder = queryBuilder.Where(q.TransferTime.Gte(startTime))
		}
	}

	if req.EndTime != nil && *req.EndTime != "" {
		endTime, err := time.Parse("2006-01-02 15:04:05", *req.EndTime)
		if err == nil {
			queryBuilder = queryBuilder.Where(q.TransferTime.Lte(endTime))
		}
	}

	// 统计总数
	total, err := queryBuilder.Count()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	transfers, err := queryBuilder.Offset(offset).Limit(req.PageSize).Order(q.TransferTime.Desc()).Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 构建响应
	list := make([]model.DeviceTransferResp, 0, len(transfers))
	for _, transfer := range transfers {
		resp := model.DeviceTransferResp{
			ID:           transfer.ID,
			DeviceID:     transfer.DeviceID,
			FromDealerID: transfer.FromDealerID,
			ToDealerID:   transfer.ToDealerID,
			OperatorID:   transfer.OperatorID,
			Remark:       transfer.Remark,
		}

		if transfer.TransferTime != nil {
			resp.TransferTime = transfer.TransferTime.Format("2006-01-02 15:04:05")
		}

		// 查询设备信息
		device, err := d.WithContext(ctx).Where(d.ID.Eq(transfer.DeviceID)).First()
		if err == nil {
			resp.DeviceNumber = device.DeviceNumber
			if device.Name != nil {
				resp.DeviceModel = *device.Name
			}
		}

		// 查询原经销商名称
		if transfer.FromDealerID != nil && *transfer.FromDealerID != "" {
			fromDealer, err := dealer.WithContext(ctx).Where(dealer.ID.Eq(*transfer.FromDealerID)).First()
			if err == nil {
				resp.FromDealerName = &fromDealer.Name
			}
		}

		// 查询目标经销商名称
		if transfer.ToDealerID != nil && *transfer.ToDealerID != "" {
			toDealer, err := dealer.WithContext(ctx).Where(dealer.ID.Eq(*transfer.ToDealerID)).First()
			if err == nil {
				resp.ToDealerName = &toDealer.Name
			}
		}

		// 查询操作人名称
		if transfer.OperatorID != nil && *transfer.OperatorID != "" {
			operator, err := user.WithContext(ctx).Where(user.ID.Eq(*transfer.OperatorID)).First()
			if err == nil && operator.Name != nil {
				resp.OperatorName = operator.Name
			}
		}

		list = append(list, resp)
	}

	return &model.DeviceTransferListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
