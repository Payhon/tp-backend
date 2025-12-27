package service

import (
	"context"
	"time"

	"project/internal/model"
	query "project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"gorm.io/gorm"
)

// EndUser BMS: 终端用户（穿透/强制解绑）
type EndUser struct{}

func endUserListSelectSQL(orgScopeID string) (string, []interface{}) {
	selectSQL := `
		u.id AS user_id,
		u.name AS user_name,
		u.phone_number AS user_phone,
		COUNT(DISTINCT d.id) AS device_count,
		MAX(dub.binding_time) AS last_bind_at,
		dbat.owner_org_id AS owner_org_id,
		org.name AS owner_org_name,
		CASE WHEN ? <> '' THEN dbat.owner_org_id ELSE NULL END AS dealer_id,
		CASE WHEN ? <> '' THEN org.name ELSE NULL END AS dealer_name
	`
	return selectSQL, []interface{}{orgScopeID, orgScopeID}
}

// GetEndUserList 终端用户列表（从绑定关系聚合）
func (*EndUser) GetEndUserList(ctx context.Context, req model.EndUserListReq, claims *utils.UserClaims, orgScopeID string) (*model.EndUserListResp, error) {
	db := global.DB.WithContext(ctx)

	// org 过滤：组织用户强制限定子树；厂家账号可选 req.OwnerOrgID
	effectiveOrgID := ""
	if orgScopeID != "" {
		effectiveOrgID = orgScopeID
	} else if req.OwnerOrgID != nil && *req.OwnerOrgID != "" {
		effectiveOrgID = *req.OwnerOrgID
	}

	base := db.Table("device_user_bindings AS dub").
		Joins("LEFT JOIN devices AS d ON d.id = dub.device_id").
		Joins("LEFT JOIN users AS u ON u.id = dub.user_id").
		Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
		Joins("LEFT JOIN orgs AS org ON org.id = dbat.owner_org_id").
		Where("d.tenant_id = ?", claims.TenantID)

	if effectiveOrgID != "" {
		base = base.Where(`dbat.owner_org_id IN (
			SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
		)`, claims.TenantID, effectiveOrgID)
	}
	if req.Phone != nil && *req.Phone != "" {
		base = base.Where("u.phone_number ILIKE ?", "%"+*req.Phone+"%")
	}
	if req.DeviceNumber != nil && *req.DeviceNumber != "" {
		base = base.Where("d.device_number ILIKE ?", "%"+*req.DeviceNumber+"%")
	}

	// total（按 user 去重）
	var total int64
	if err := base.Select("u.id").Distinct("u.id").Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	type row struct {
		UserID       string     `gorm:"column:user_id"`
		UserName     *string    `gorm:"column:user_name"`
		UserPhone    string     `gorm:"column:user_phone"`
		DeviceCount  int64      `gorm:"column:device_count"`
		LastBindAt   *time.Time `gorm:"column:last_bind_at"`
		DealerID     *string    `gorm:"column:dealer_id"`
		DealerName   *string    `gorm:"column:dealer_name"`
		OwnerOrgID   *string    `gorm:"column:owner_org_id"`
		OwnerOrgName *string    `gorm:"column:owner_org_name"`
	}

	offset := (req.Page - 1) * req.PageSize
	rows := make([]row, 0, req.PageSize)

	selectSQL, selectArgs := endUserListSelectSQL(effectiveOrgID)

	if err := base.Select(selectSQL, selectArgs...).
		Group("u.id, u.name, u.phone_number, owner_org_id, owner_org_name").
		Order("last_bind_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.EndUserListItemResp, 0, len(rows))
	for _, r := range rows {
		var lastBindAt *string
		if r.LastBindAt != nil {
			s := r.LastBindAt.In(time.Local).Format("2006-01-02 15:04:05")
			lastBindAt = &s
		}
		list = append(list, model.EndUserListItemResp{
			UserID:       r.UserID,
			UserName:     r.UserName,
			UserPhone:    r.UserPhone,
			DeviceCount:  r.DeviceCount,
			LastBindAt:   lastBindAt,
			OwnerOrgID:   r.OwnerOrgID,
			OwnerOrgName: r.OwnerOrgName,
			DealerID:     r.DealerID,
			DealerName:   r.DealerName,
		})
	}

	return &model.EndUserListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetEndUserDevices 获取用户绑定设备列表（用于穿透查看）
func (*EndUser) GetEndUserDevices(ctx context.Context, req model.EndUserDeviceListReq, claims *utils.UserClaims, orgScopeID string) (*model.EndUserDeviceListResp, error) {
	db := global.DB.WithContext(ctx)

	base := db.Table("device_user_bindings AS dub").
		Joins("LEFT JOIN devices AS d ON d.id = dub.device_id").
		Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
		Where("d.tenant_id = ?", claims.TenantID).
		Where("dub.user_id = ?", req.UserID)
	if orgScopeID != "" {
		base = base.Where(`dbat.owner_org_id IN (
			SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
		)`, claims.TenantID, orgScopeID)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	type row struct {
		ID           string     `gorm:"column:id"`
		DeviceID     string     `gorm:"column:device_id"`
		DeviceNumber string     `gorm:"column:device_number"`
		DeviceName   *string    `gorm:"column:device_name"`
		IsOwner      *bool      `gorm:"column:is_owner"`
		BindingTime  *time.Time `gorm:"column:binding_time"`
	}

	offset := (req.Page - 1) * req.PageSize
	rows := make([]row, 0, req.PageSize)
	if err := base.Select(`
			dub.id AS id,
			dub.device_id AS device_id,
			d.device_number AS device_number,
			d.name AS device_name,
			dub.is_owner AS is_owner,
			dub.binding_time AS binding_time
		`).
		Order("dub.binding_time DESC").
		Offset(offset).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.EndUserDeviceItemResp, 0, len(rows))
	for _, r := range rows {
		isOwner := false
		if r.IsOwner != nil {
			isOwner = *r.IsOwner
		}
		bt := ""
		if r.BindingTime != nil {
			bt = r.BindingTime.In(time.Local).Format("2006-01-02 15:04:05")
		}
		list = append(list, model.EndUserDeviceItemResp{
			BindingID:    r.ID,
			DeviceID:     r.DeviceID,
			DeviceNumber: r.DeviceNumber,
			DeviceName:   r.DeviceName,
			IsOwner:      isOwner,
			BindingTime:  bt,
		})
	}

	return &model.EndUserDeviceListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ForceUnbind 强制解绑（管理员/组织用户）
func (*EndUser) ForceUnbind(ctx context.Context, req model.EndUserForceUnbindReq, claims *utils.UserClaims, orgScopeID string) error {
	tx := query.Use(global.DB).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 查找绑定记录
	binding, err := tx.DeviceUserBinding.WithContext(ctx).
		Where(tx.DeviceUserBinding.ID.Eq(req.BindingID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			tx.Rollback()
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "binding not found"})
		}
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	// 数据范围校验：组织用户仅能解绑子树名下设备
	if orgScopeID != "" {
		dbat, err := tx.DeviceBattery.WithContext(ctx).
			Where(tx.DeviceBattery.DeviceID.Eq(binding.DeviceID)).
			First()
		if err == nil && dbat.OwnerOrgID != nil && *dbat.OwnerOrgID != "" {
			var count int64
			global.DB.Table("org_closure").
				Where("tenant_id = ? AND ancestor_id = ? AND descendant_id = ?",
					claims.TenantID, orgScopeID, *dbat.OwnerOrgID).
				Count(&count)
			if count == 0 {
				tx.Rollback()
				return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "no permission"})
			}
		}
	}

	// 删除绑定
	if _, err := tx.DeviceUserBinding.WithContext(ctx).
		Where(tx.DeviceUserBinding.ID.Eq(req.BindingID)).
		Delete(); err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	// 若该设备无其它绑定，重置激活状态（复用 APP 解绑逻辑）
	remain, err := tx.DeviceUserBinding.WithContext(ctx).
		Where(tx.DeviceUserBinding.DeviceID.Eq(binding.DeviceID)).
		Count()
	if err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if remain == 0 {
		t := time.Now().UTC()
		deviceBattery, err := tx.DeviceBattery.WithContext(ctx).
			Where(tx.DeviceBattery.DeviceID.Eq(binding.DeviceID)).
			First()
		if err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}

		if err == nil {
			updates := map[string]interface{}{
				"activation_status": "INACTIVE",
				"activation_date":   nil,
				"updated_at":        t,
			}
			if deviceBattery.OwnerOrgID != nil && *deviceBattery.OwnerOrgID != "" {
				updates["transfer_status"] = "DEALER"
			} else {
				updates["transfer_status"] = "FACTORY"
			}

			if _, err := tx.DeviceBattery.WithContext(ctx).
				Where(tx.DeviceBattery.DeviceID.Eq(binding.DeviceID)).
				Updates(updates); err != nil {
				tx.Rollback()
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}
