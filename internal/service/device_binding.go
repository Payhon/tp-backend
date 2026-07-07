package service

import (
	"context"
	"database/sql"
	"strings"
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

func releaseBleMacIfNoAppAssociationsTx(ctx context.Context, db *gorm.DB, deviceID string, now time.Time) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}

	var bindingCount int64
	if err := db.WithContext(ctx).
		Table("device_user_bindings").
		Where("device_id = ?", deviceID).
		Count(&bindingCount).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if bindingCount > 0 {
		return nil
	}

	var addedCount int64
	if err := db.WithContext(ctx).
		Table("app_device_added_records").
		Where("device_id = ?", deviceID).
		Count(&addedCount).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if addedCount > 0 {
		return nil
	}

	if err := db.WithContext(ctx).
		Table("device_batteries").
		Where("device_id = ?", deviceID).
		Updates(map[string]interface{}{
			"ble_mac":    nil,
			"updated_at": now,
		}).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

// getUserOrgID 根据当前登录用户获取归属组织ID（可能为空）
func getUserOrgID(userID string) (string, error) {
	if userID == "" {
		return "", nil
	}

	var orgID sql.NullString
	if err := global.DB.Table("users").
		Select("org_id").
		Where("id = ?", userID).
		Scan(&orgID).Error; err != nil {
		return "", err
	}
	if !orgID.Valid {
		return "", nil
	}
	return orgID.String, nil
}

// BindDevice APP端设备绑定
// 1. 校验设备合法性（存在且属于当前租户）及可选密钥
// 2. 校验设备是否已绑定当前用户
// 3. 创建 device_user_bindings 记录
// 4. 更新 device_batteries 激活状态/流转状态
func (*DeviceBinding) BindDevice(req model.DeviceBindReq, claims *utils.UserClaims) error {
	ctx := context.Background()
	viewCtx, err := resolveAppDeviceViewContext(ctx, claims)
	if err != nil {
		return err
	}
	if viewCtx.userKind != model.UserKindEndUser {
		return errcode.New(errcode.CodeNoPermission)
	}

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
			if err := applyBatteryWarrantyActivationTx(ctx, tx.DeviceBattery.UnderlyingDB(), device.ID, claims.ID, t); err != nil {
				tx.Rollback()
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
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
		if err := applyBatteryWarrantyActivationTx(ctx, tx.DeviceBattery.UnderlyingDB(), device.ID, claims.ID, t); err != nil {
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
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
	viewCtx, err := resolveAppDeviceViewContext(ctx, claims)
	if err != nil {
		return err
	}
	if viewCtx.userKind != model.UserKindEndUser {
		return errcode.New(errcode.CodeNoPermission)
	}

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

	if err := releaseBleMacIfNoAppAssociationsTx(ctx, tx.DeviceBattery.UnderlyingDB(), req.DeviceID, t); err != nil {
		tx.Rollback()
		return err
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
	viewCtx, err := resolveAppDeviceViewContext(ctx, claims)
	if err != nil {
		return nil, err
	}

	viewMode, err := resolveRequestedViewMode(&req, viewCtx)
	if err != nil {
		return nil, err
	}

	switch viewMode {
	case model.AppDeviceViewModeSelfBound:
		return GroupApp.DeviceBinding.listSelfBoundDevices(ctx, req, claims)
	case model.AppDeviceViewModeOrgAdded:
		return GroupApp.DeviceBinding.listOrgAddedDevices(ctx, req, claims, viewCtx)
	case model.AppDeviceViewModeEndUserBind:
		return GroupApp.DeviceBinding.listOrgEndUserBoundDevices(ctx, req, claims, viewCtx)
	default:
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "unsupported view_mode",
		})
	}
}

type appOrgDeviceRow struct {
	DeviceID         string  `gorm:"column:device_id"`
	DeviceNumber     string  `gorm:"column:device_number"`
	DeviceName       *string `gorm:"column:device_name"`
	IsOnline         int16   `gorm:"column:is_online"`
	ActivationStatus *string `gorm:"column:activation_status"`
	OwnerOrgID       *string `gorm:"column:owner_org_id"`
	OwnerOrgName     *string `gorm:"column:owner_org_name"`
	OwnerOrgType     *string `gorm:"column:owner_org_type"`
}

func allowOrgFilter(orgType, targetType string) bool {
	switch strings.TrimSpace(orgType) {
	case model.OrgTypeBMSFactory:
		return targetType == model.OrgTypePACKFactory || targetType == model.OrgTypeDealer || targetType == model.OrgTypeStore
	case model.OrgTypePACKFactory:
		return targetType == model.OrgTypeDealer || targetType == model.OrgTypeStore
	case model.OrgTypeDealer:
		return targetType == model.OrgTypeStore
	default:
		return false
	}
}

func canAccessOrg(tenantID, accessorOrgID, targetOrgID string) bool {
	if accessorOrgID == "" {
		return true
	}
	if accessorOrgID == targetOrgID {
		return true
	}
	var count int64
	global.DB.Table("org_closure").
		Where("tenant_id = ? AND ancestor_id = ? AND descendant_id = ?", tenantID, accessorOrgID, targetOrgID).
		Count(&count)
	return count > 0
}

func trimOptionalFilterValue(input *string) string {
	if input == nil {
		return ""
	}
	text := strings.TrimSpace(*input)
	switch strings.ToLower(text) {
	case "", "undefined", "null":
		return ""
	default:
		return text
	}
}

// GetOrgDevices 获取组织范围设备列表（APP端）
func (*DeviceBinding) GetOrgDevices(req model.AppOrgDeviceListReq, claims *utils.UserClaims) (*model.AppOrgDeviceListResp, error) {
	ctx := context.Background()
	tenantID := strings.TrimSpace(claims.TenantID)
	userID := strings.TrimSpace(claims.ID)
	if tenantID == "" || userID == "" {
		return &model.AppOrgDeviceListResp{
			List:     []model.AppOrgDeviceListItem{},
			Total:    0,
			Page:     req.Page,
			PageSize: req.PageSize,
		}, nil
	}

	isAdmin := claims.Authority == "SYS_ADMIN" || claims.Authority == "TENANT_ADMIN"
	userKind := model.UserKindEndUser
	if !isAdmin {
		var err error
		userKind, err = GroupApp.OrgTypePermission.GetUserKind(ctx, tenantID, userID)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_user_kind",
				"user_id":   userID,
				"error":     err.Error(),
			})
		}
		if userKind != model.UserKindOrgUser {
			return nil, errcode.New(errcode.CodeNoPermission)
		}
	}

	orgID, _ := getUserOrgID(userID)
	orgType := ""
	if !isAdmin {
		var ok bool
		var err error
		orgType, ok, err = GroupApp.OrgTypePermission.GetUserOrgType(ctx, tenantID, userID)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_user_org_type",
				"user_id":   userID,
				"error":     err.Error(),
			})
		}
		if !ok {
			orgType = ""
		}
	}
	orgType = strings.TrimSpace(orgType)
	isFactory := isAdmin || orgType == model.OrgTypeBMSFactory || orgID == ""

	db := global.DB.WithContext(ctx)
	queryBuilder := db.Table("devices AS d").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			d.name AS device_name,
			d.is_online AS is_online,
			dbat.activation_status AS activation_status,
			dbat.owner_org_id AS owner_org_id,
			org.name AS owner_org_name,
			org.org_type AS owner_org_type
		`).
		Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id`).
		Joins(`LEFT JOIN orgs AS org ON org.id = dbat.owner_org_id`).
		Where("d.tenant_id = ?", tenantID)

	if !isFactory && orgID != "" {
		queryBuilder = queryBuilder.Where(`dbat.owner_org_id IN (
			SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
		)`, tenantID, orgID)
	}

	if deviceNumber := trimOptionalFilterValue(req.DeviceNumber); deviceNumber != "" {
		queryBuilder = queryBuilder.Where("d.device_number ILIKE ?", "%"+deviceNumber+"%")
	}
	if ownerID := trimOptionalFilterValue(req.OwnerOrgID); ownerID != "" {
		if !isFactory && orgID != "" {
			if !canAccessOrg(tenantID, orgID, ownerID) {
				return nil, errcode.New(errcode.CodeNoPermission)
			}
		}
		queryBuilder = queryBuilder.Where("dbat.owner_org_id = ?", ownerID)
	} else if ownerType := trimOptionalFilterValue(req.OwnerOrgType); ownerType != "" {
		if !isFactory && !allowOrgFilter(orgType, ownerType) {
			return nil, errcode.New(errcode.CodeNoPermission)
		}
		queryBuilder = queryBuilder.Where("org.org_type = ?", ownerType)
	}

	var total int64
	if err := queryBuilder.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	offset := (req.Page - 1) * req.PageSize
	rows := make([]appOrgDeviceRow, 0, req.PageSize)
	if err := queryBuilder.
		Order("d.created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	list := make([]model.AppOrgDeviceListItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, model.AppOrgDeviceListItem{
			DeviceID:         r.DeviceID,
			DeviceNumber:     r.DeviceNumber,
			DeviceName:       r.DeviceName,
			IsOnline:         r.IsOnline,
			ActivationStatus: r.ActivationStatus,
			OwnerOrgID:       r.OwnerOrgID,
			OwnerOrgName:     r.OwnerOrgName,
			OwnerOrgType:     r.OwnerOrgType,
		})
	}

	return &model.AppOrgDeviceListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetOrgOptions 获取组织选项（APP端）
func (*DeviceBinding) GetOrgOptions(ctx context.Context, req *model.AppOrgOptionReq, claims *utils.UserClaims) ([]model.AppOrgOptionResp, error) {
	tenantID := strings.TrimSpace(claims.TenantID)
	userID := strings.TrimSpace(claims.ID)
	if tenantID == "" || userID == "" {
		return []model.AppOrgOptionResp{}, nil
	}

	isAdmin := claims.Authority == "SYS_ADMIN" || claims.Authority == "TENANT_ADMIN"
	userKind := model.UserKindEndUser
	if !isAdmin {
		var err error
		userKind, err = GroupApp.OrgTypePermission.GetUserKind(ctx, tenantID, userID)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_user_kind",
				"user_id":   userID,
				"error":     err.Error(),
			})
		}
		if userKind != model.UserKindOrgUser {
			return nil, errcode.New(errcode.CodeNoPermission)
		}
	}

	orgID, _ := getUserOrgID(userID)
	orgType := ""
	if !isAdmin {
		var ok bool
		var err error
		orgType, ok, err = GroupApp.OrgTypePermission.GetUserOrgType(ctx, tenantID, userID)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_user_org_type",
				"user_id":   userID,
				"error":     err.Error(),
			})
		}
		if !ok {
			orgType = ""
		}
	}
	orgType = strings.TrimSpace(orgType)
	isFactory := isAdmin || orgType == model.OrgTypeBMSFactory || orgID == ""

	if !isFactory && !allowOrgFilter(orgType, req.OrgType) {
		return nil, errcode.New(errcode.CodeNoPermission)
	}

	queryBuilder := global.DB.WithContext(ctx).
		Table("orgs AS o").
		Select("o.id, o.name, o.org_type").
		Where("o.tenant_id = ? AND o.org_type = ?", tenantID, req.OrgType)

	if !isFactory && orgID != "" {
		switch orgType {
		case model.OrgTypePACKFactory:
			if req.OrgType != model.OrgTypeDealer && req.OrgType != model.OrgTypeStore {
				return nil, errcode.New(errcode.CodeNoPermission)
			}
			queryBuilder = queryBuilder.Where(`o.id IN (
				SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
			)`, tenantID, orgID)
		case model.OrgTypeDealer:
			switch req.OrgType {
			case model.OrgTypeStore:
				queryBuilder = queryBuilder.Where(`o.id IN (
					SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
				)`, tenantID, orgID)
			case model.OrgTypePACKFactory:
				queryBuilder = queryBuilder.Where(`o.id IN (
					SELECT ancestor_id FROM org_closure WHERE tenant_id = ? AND descendant_id = ?
				)`, tenantID, orgID)
			default:
				return nil, errcode.New(errcode.CodeNoPermission)
			}
		case model.OrgTypeStore:
			if req.OrgType != model.OrgTypeDealer {
				return nil, errcode.New(errcode.CodeNoPermission)
			}
			queryBuilder = queryBuilder.Where(`o.id IN (
				SELECT ancestor_id FROM org_closure WHERE tenant_id = ? AND descendant_id = ?
			)`, tenantID, orgID)
		default:
			return nil, errcode.New(errcode.CodeNoPermission)
		}
	}

	var rows []model.AppOrgOptionResp
	if err := queryBuilder.Order("o.created_at ASC").Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	return rows, nil
}
