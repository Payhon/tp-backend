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

// DeviceBinding 设备绑定服务
type DeviceBinding struct{}

// getUserOrgID 根据当前登录用户获取归属组织ID（可能为空）
func getUserOrgID(userID string) (string, error) {
	if userID == "" {
		return "", nil
	}

	var orgID string
	if err := global.DB.Table("users").
		Select("org_id").
		Where("id = ?", userID).
		Scan(&orgID).Error; err != nil {
		return "", err
	}
	return orgID, nil
}

// BindDevice APP端设备绑定
// 1. 校验设备合法性（存在且属于当前租户）及可选密钥
// 2. 校验设备是否已绑定当前用户
// 3. 创建 device_user_bindings 记录
// 4. 更新 device_batteries 激活状态/流转状态
func (*DeviceBinding) BindDevice(req model.DeviceBindReq, claims *utils.UserClaims) error {
	ctx := context.Background()
	q := query.Use(global.DB)

	// 查询设备信息并校验租户
	device, err := q.Device.WithContext(ctx).
		Where(
			q.Device.DeviceNumber.Eq(req.DeviceNumber),
			q.Device.TenantID.Eq(claims.TenantID),
		).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "device not found",
			})
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 校验设备密钥（如果传入）
	if req.DeviceSecret != nil && *req.DeviceSecret != "" {
		if *req.DeviceSecret != device.Voucher {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "invalid device secret",
			})
		}
	}

	// 获取用户归属组织（用于组织级别的数据校验）
	userOrgID, err := getUserOrgID(claims.ID)
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
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

	// 检查是否已绑定当前用户
	if _, err := tx.DeviceUserBinding.WithContext(ctx).
		Where(
			tx.DeviceUserBinding.DeviceID.Eq(device.ID),
			tx.DeviceUserBinding.UserID.Eq(claims.ID),
		).
		First(); err != nil {
		if err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}
	} else {
		tx.Rollback()
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "device already bound to current user",
		})
	}

	// 查询该设备是否已有其它绑定关系
	existBindings, err := tx.DeviceUserBinding.WithContext(ctx).
		Where(tx.DeviceUserBinding.DeviceID.Eq(device.ID)).
		Find()
	if err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	isFirstBinding := len(existBindings) == 0

	// 处理 device_batteries 信息：组织校验 + 激活状态更新
	deviceBattery, err := tx.DeviceBattery.WithContext(ctx).
		Where(tx.DeviceBattery.DeviceID.Eq(device.ID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 未找到记录时，如果用户有组织归属，则自动写入 owner_org_id
			newBattery := &model.DeviceBattery{
				DeviceID:         device.ID,
				ActivationStatus: StringPtr("ACTIVE"),
				TransferStatus:   StringPtr("USER"),
				ActivationDate:   &t,
				UpdatedAt:        &t,
			}
			if userOrgID != "" {
				newBattery.OwnerOrgID = &userOrgID
			}

			if err := tx.DeviceBattery.Create(newBattery); err != nil {
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
		// 校验设备归属组织与当前用户是否匹配（如果用户有组织信息）
		// 基于组织子树校验：用户的组织必须是设备所属组织的祖先
		if userOrgID != "" && deviceBattery.OwnerOrgID != nil && *deviceBattery.OwnerOrgID != "" {
			var count int64
			global.DB.Table("org_closure").
				Where("tenant_id = ? AND ancestor_id = ? AND descendant_id = ?",
					claims.TenantID, userOrgID, *deviceBattery.OwnerOrgID).
				Count(&count)
			if count == 0 {
				tx.Rollback()
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "device does not belong to current organization",
				})
			}
		}

		updates := map[string]interface{}{
			"activation_status": "ACTIVE",
			"transfer_status":   "USER",
			"activation_date":   t,
			"updated_at":        t,
		}
		// 如果设备当前没有组织归属而用户有归属，则补充 owner_org_id
		if deviceBattery.OwnerOrgID == nil && userOrgID != "" {
			updates["owner_org_id"] = userOrgID
		}

		if _, err := tx.DeviceBattery.WithContext(ctx).
			Where(tx.DeviceBattery.DeviceID.Eq(device.ID)).
			Updates(updates); err != nil {
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}
	}

	// 创建绑定关系
	isOwner := isFirstBinding
	binding := &model.DeviceUserBinding{
		ID:          uuid.New(),
		UserID:      claims.ID,
		DeviceID:    device.ID,
		BindingTime: &t,
		IsOwner:     &isOwner,
	}

	if err := tx.DeviceUserBinding.WithContext(ctx).Create(binding); err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return nil
}

// UnbindDevice APP端设备解绑
// 1. 删除当前用户与设备的绑定关系
// 2. 当设备不存在其它绑定关系时，重置激活状态
func (*DeviceBinding) UnbindDevice(req model.DeviceUnbindReq, claims *utils.UserClaims) error {
	ctx := context.Background()
	tx := query.Use(global.DB).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	t := time.Now().UTC()

	// 校验绑定关系是否存在
	binding, err := tx.DeviceUserBinding.WithContext(ctx).
		Where(
			tx.DeviceUserBinding.DeviceID.Eq(req.DeviceID),
			tx.DeviceUserBinding.UserID.Eq(claims.ID),
		).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			tx.Rollback()
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "binding not found",
			})
		}
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 删除绑定记录
	if _, err := tx.DeviceUserBinding.WithContext(ctx).
		Where(tx.DeviceUserBinding.ID.Eq(binding.ID)).
		Delete(); err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 检查该设备是否还有其它绑定记录
	remainCount, err := tx.DeviceUserBinding.WithContext(ctx).
		Where(tx.DeviceUserBinding.DeviceID.Eq(req.DeviceID)).
		Count()
	if err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 如果没有其它绑定关系，则重置激活状态
	if remainCount == 0 {
		deviceBattery, err := tx.DeviceBattery.WithContext(ctx).
			Where(tx.DeviceBattery.DeviceID.Eq(req.DeviceID)).
			First()
		if err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}

		if err == nil {
			updates := map[string]interface{}{
				"activation_status": "INACTIVE",
				"activation_date":   nil,
				"updated_at":        t,
			}
			// 若存在组织归属，则流转状态回退为 DEALER，否则为 FACTORY
			if deviceBattery.OwnerOrgID != nil && *deviceBattery.OwnerOrgID != "" {
				updates["transfer_status"] = "DEALER"
			} else {
				updates["transfer_status"] = "FACTORY"
			}

			if _, err := tx.DeviceBattery.WithContext(ctx).
				Where(tx.DeviceBattery.DeviceID.Eq(req.DeviceID)).
				Updates(updates); err != nil {
				tx.Rollback()
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"sql_error": err.Error(),
				})
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return nil
}

// GetUserDevices 获取用户绑定的设备列表
func (*DeviceBinding) GetUserDevices(req model.DeviceUserBindingListReq, claims *utils.UserClaims) (*model.DeviceUserBindingListResp, error) {
	ctx := context.Background()

	bindingQuery := query.DeviceUserBinding.WithContext(ctx)

	// 默认查询当前用户的绑定设备
	if req.UserID != nil && *req.UserID != "" {
		bindingQuery = bindingQuery.Where(query.DeviceUserBinding.UserID.Eq(*req.UserID))
	} else {
		bindingQuery = bindingQuery.Where(query.DeviceUserBinding.UserID.Eq(claims.ID))
	}

	// 先根据租户和（可选）设备编号筛选出符合条件的设备ID
	deviceQuery := query.Device.WithContext(ctx).Where(query.Device.TenantID.Eq(claims.TenantID))
	if req.DeviceNumber != nil && *req.DeviceNumber != "" {
		deviceQuery = deviceQuery.Where(query.Device.DeviceNumber.Like("%" + *req.DeviceNumber + "%"))
	}

	devices, err := deviceQuery.Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	if len(devices) == 0 {
		return &model.DeviceUserBindingListResp{
			List:     []model.DeviceUserBindingResp{},
			Total:    0,
			Page:     req.Page,
			PageSize: req.PageSize,
		}, nil
	}

	deviceIDs := make([]string, 0, len(devices))
	for _, d := range devices {
		deviceIDs = append(deviceIDs, d.ID)
	}

	bindingQuery = bindingQuery.Where(query.DeviceUserBinding.DeviceID.In(deviceIDs...))

	// 统计总数
	total, err := bindingQuery.Count()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 分页查询绑定记录
	offset := (req.Page - 1) * req.PageSize
	bindings, err := bindingQuery.
		Offset(offset).
		Limit(req.PageSize).
		Order(query.DeviceUserBinding.BindingTime.Desc()).
		Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	if len(bindings) == 0 {
		return &model.DeviceUserBindingListResp{
			List:     []model.DeviceUserBindingResp{},
			Total:    total,
			Page:     req.Page,
			PageSize: req.PageSize,
		}, nil
	}

	// 收集用户ID和设备ID，避免 N+1 查询
	userIDs := make(map[string]struct{})
	deviceIDSet := make(map[string]struct{})
	for _, b := range bindings {
		userIDs[b.UserID] = struct{}{}
		deviceIDSet[b.DeviceID] = struct{}{}
	}

	userIDList := make([]string, 0, len(userIDs))
	for id := range userIDs {
		userIDList = append(userIDList, id)
	}

	deviceIDList := make([]string, 0, len(deviceIDSet))
	for id := range deviceIDSet {
		deviceIDList = append(deviceIDList, id)
	}

	// 查询用户与设备信息
	users, err := query.User.WithContext(ctx).
		Where(query.User.ID.In(userIDList...)).
		Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	devicesForBindings, err := query.Device.WithContext(ctx).
		Where(
			query.Device.ID.In(deviceIDList...),
			query.Device.TenantID.Eq(claims.TenantID),
		).
		Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	userMap := make(map[string]*model.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	deviceMap := make(map[string]*model.Device, len(devicesForBindings))
	for _, d := range devicesForBindings {
		deviceMap[d.ID] = d
	}

	// 组装响应
	list := make([]model.DeviceUserBindingResp, 0, len(bindings))
	for _, b := range bindings {
		resp := model.DeviceUserBindingResp{
			ID:       b.ID,
			UserID:   b.UserID,
			DeviceID: b.DeviceID,
			IsOwner:  b.IsOwner != nil && *b.IsOwner,
		}

		if b.BindingTime != nil {
			resp.BindingTime = b.BindingTime.Format("2006-01-02 15:04:05")
		}

		if u, ok := userMap[b.UserID]; ok {
			resp.UserName = u.Name
			resp.UserPhone = u.PhoneNumber
		}

		if d, ok := deviceMap[b.DeviceID]; ok {
			resp.DeviceNumber = d.DeviceNumber
			if d.Name != nil {
				resp.DeviceName = *d.Name
			}
		}

		list = append(list, resp)
	}

	return &model.DeviceUserBindingListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
