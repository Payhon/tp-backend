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
