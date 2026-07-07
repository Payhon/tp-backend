package service

import (
	"context"
	"errors"
	"strings"
	"time"

	dal "project/internal/dal"
	"project/internal/model"
	"project/pkg/errcode"
	"project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

type UserWarrantyInfo struct{}

type userWarrantyInfoRow struct {
	ContactName  *string `gorm:"column:contact_name"`
	ContactPhone *string `gorm:"column:contact_phone"`
	UserName     *string `gorm:"column:user_name"`
	Username     *string `gorm:"column:username"`
	UserPhone    string  `gorm:"column:user_phone"`
}

type appWarrantyBatteryRow struct {
	DeviceID           string     `gorm:"column:device_id"`
	DeviceNumber       string     `gorm:"column:device_number"`
	BatterySerial      string     `gorm:"column:battery_serial"`
	BatteryModelName   *string    `gorm:"column:battery_model_name"`
	ActivationDate     *time.Time `gorm:"column:activation_date"`
	WarrantyStartDate  *time.Time `gorm:"column:warranty_start_date"`
	WarrantyExpireDate *time.Time `gorm:"column:warranty_expire_date"`
	WarrantyMonths     *int32     `gorm:"column:warranty_months"`
}

type batteryWarrantyRow struct {
	DeviceID               string     `gorm:"column:device_id"`
	DeviceNumber           string     `gorm:"column:device_number"`
	BatterySerial          string     `gorm:"column:battery_serial"`
	BatteryModelID         *string    `gorm:"column:battery_model_id"`
	BatteryModelName       *string    `gorm:"column:battery_model_name"`
	ActivationDate         *time.Time `gorm:"column:activation_date"`
	WarrantyStartDate      *time.Time `gorm:"column:warranty_start_date"`
	WarrantyExpireDate     *time.Time `gorm:"column:warranty_expire_date"`
	WarrantyMonths         *int32     `gorm:"column:warranty_months"`
	WarrantyManualOverride bool       `gorm:"column:warranty_manual_override"`
	WarrantyUpdatedAt      *time.Time `gorm:"column:warranty_updated_at"`
	WarrantyUpdatedBy      *string    `gorm:"column:warranty_updated_by"`
}

type batteryWarrantyUserRow struct {
	UserID       string     `gorm:"column:user_id"`
	UserName     *string    `gorm:"column:user_name"`
	Username     *string    `gorm:"column:username"`
	UserPhone    string     `gorm:"column:user_phone"`
	ContactName  *string    `gorm:"column:contact_name"`
	ContactPhone *string    `gorm:"column:contact_phone"`
	IsOwner      bool       `gorm:"column:is_owner"`
	BindingTime  *time.Time `gorm:"column:binding_time"`
}

type batteryWarrantyActivationRow struct {
	WarrantyMonths         *int32     `gorm:"column:warranty_months"`
	WarrantyStartDate      *time.Time `gorm:"column:warranty_start_date"`
	WarrantyExpireDate     *time.Time `gorm:"column:warranty_expire_date"`
	WarrantyManualOverride bool       `gorm:"column:warranty_manual_override"`
}

func formatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Local().Format("2006-01-02")
	return &s
}

func formatRFC3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func trimmedStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}

func resolveWarrantyCardsEnabled(ctx context.Context, tenantID, appid string) (bool, error) {
	appid = strings.TrimSpace(appid)
	if appid == "" {
		return true, nil
	}
	cfg, err := dal.GetPackWxMpConfigByWxAppID(ctx, tenantID, appid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return true, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	if strings.ToUpper(strings.TrimSpace(cfg.Status)) != "OPEN" {
		return true, nil
	}
	return cfg.WarrantyCardsEnabled, nil
}

func ensureEndUserWarrantyClaims(claims *utils.UserClaims) error {
	if claims == nil || strings.TrimSpace(claims.ID) == "" || strings.TrimSpace(claims.TenantID) == "" {
		return errcode.New(errcode.CodeNoPermission)
	}
	if strings.TrimSpace(claims.UserKind) != "" && strings.TrimSpace(claims.UserKind) != model.UserKindEndUser {
		return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "仅终端用户可访问质保信息"})
	}
	return nil
}

func (*UserWarrantyInfo) GetProfile(ctx context.Context, claims *utils.UserClaims, appid string) (*model.AppWarrantyProfileResp, error) {
	if err := ensureEndUserWarrantyClaims(claims); err != nil {
		return nil, err
	}
	cardsEnabled, err := resolveWarrantyCardsEnabled(ctx, claims.TenantID, appid)
	if err != nil {
		return nil, err
	}

	var info userWarrantyInfoRow
	if err := global.DB.WithContext(ctx).
		Table("users AS u").
		Select(`uwi.contact_name, uwi.contact_phone, u.name AS user_name, u.username, u.phone_number AS user_phone`).
		Joins(`LEFT JOIN user_warranty_infos AS uwi ON uwi.tenant_id = u.tenant_id AND uwi.user_id = u.id`).
		Where("u.tenant_id = ? AND u.id = ?", claims.TenantID, claims.ID).
		Limit(1).
		Scan(&info).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	resp := &model.AppWarrantyProfileResp{
		ContactName:          info.ContactName,
		ContactPhone:         info.ContactPhone,
		WarrantyCardsEnabled: cardsEnabled,
		Batteries:            []model.AppWarrantyBatteryCardResp{},
	}
	if resp.ContactName == nil {
		resp.ContactName = info.UserName
		if resp.ContactName == nil {
			resp.ContactName = info.Username
		}
	}
	if resp.ContactPhone == nil && strings.TrimSpace(info.UserPhone) != "" {
		phone := strings.TrimSpace(info.UserPhone)
		resp.ContactPhone = &phone
	}
	if !cardsEnabled {
		return resp, nil
	}

	var rows []appWarrantyBatteryRow
	if err := global.DB.WithContext(ctx).
		Table("device_user_bindings AS dub").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			COALESCE(NULLIF(dbat.item_uuid, ''), d.device_number) AS battery_serial,
			COALESCE(bm_pack.name, bm_bms.name) AS battery_model_name,
			dbat.activation_date AS activation_date,
			dbat.warranty_start_date AS warranty_start_date,
			dbat.warranty_expire_date AS warranty_expire_date,
			dbat.warranty_months AS warranty_months
		`).
		Joins(`JOIN devices AS d ON d.id = dub.device_id AND d.tenant_id = ?`, claims.TenantID).
		Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id`).
		Joins(`LEFT JOIN battery_models AS bm_pack ON bm_pack.id = dbat.battery_model_id`).
		Joins(`LEFT JOIN battery_bms_models AS bm_bms ON bm_bms.id = dbat.battery_model_id`).
		Where("dub.user_id = ?", claims.ID).
		Order("COALESCE(dbat.activation_date, dub.binding_time) DESC").
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	for _, r := range rows {
		resp.Batteries = append(resp.Batteries, model.AppWarrantyBatteryCardResp{
			DeviceID:           r.DeviceID,
			DeviceNumber:       r.DeviceNumber,
			BatterySerial:      r.BatterySerial,
			BatteryModelName:   r.BatteryModelName,
			ActivationDate:     formatDatePtr(r.ActivationDate),
			WarrantyStartDate:  formatDatePtr(r.WarrantyStartDate),
			WarrantyExpireDate: formatDatePtr(r.WarrantyExpireDate),
			WarrantyMonths:     r.WarrantyMonths,
		})
	}
	return resp, nil
}

func (*UserWarrantyInfo) SaveProfile(ctx context.Context, claims *utils.UserClaims, appid string, req *model.AppWarrantyProfileSaveReq) (*model.AppWarrantyProfileResp, error) {
	if err := ensureEndUserWarrantyClaims(claims); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "request is empty"})
	}
	now := time.Now().UTC()
	if err := global.DB.WithContext(ctx).Exec(`
		INSERT INTO user_warranty_infos (
			id, tenant_id, user_id, contact_name, contact_phone, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (tenant_id, user_id)
		DO UPDATE SET
			contact_name = EXCLUDED.contact_name,
			contact_phone = EXCLUDED.contact_phone,
			updated_at = EXCLUDED.updated_at
	`, uuid.New(), claims.TenantID, claims.ID, trimmedStringPtr(req.ContactName), trimmedStringPtr(req.ContactPhone), now, now).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return GroupApp.UserWarrantyInfo.GetProfile(ctx, claims, appid)
}

func (*UserWarrantyInfo) GetBatteryWarranty(ctx context.Context, deviceID string, claims *utils.UserClaims, orgID string) (*model.BatteryWarrantyResp, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "device_id is required"})
	}
	if err := checkDeviceOrgAccess(ctx, deviceID, claims.TenantID, orgID); err != nil {
		return nil, err
	}

	var row batteryWarrantyRow
	tx := global.DB.WithContext(ctx).
		Table("devices AS d").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			COALESCE(NULLIF(dbat.item_uuid, ''), d.device_number) AS battery_serial,
			dbat.battery_model_id AS battery_model_id,
			COALESCE(bm_pack.name, bm_bms.name) AS battery_model_name,
			dbat.activation_date AS activation_date,
			dbat.warranty_start_date AS warranty_start_date,
			dbat.warranty_expire_date AS warranty_expire_date,
			dbat.warranty_months AS warranty_months,
			COALESCE(dbat.warranty_manual_override, false) AS warranty_manual_override,
			dbat.warranty_updated_at AS warranty_updated_at,
			dbat.warranty_updated_by AS warranty_updated_by
		`).
		Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id`).
		Joins(`LEFT JOIN battery_models AS bm_pack ON bm_pack.id = dbat.battery_model_id`).
		Joins(`LEFT JOIN battery_bms_models AS bm_bms ON bm_bms.id = dbat.battery_model_id`).
		Where("d.tenant_id = ? AND d.id = ?", claims.TenantID, deviceID).
		Limit(1).
		Scan(&row)
	if tx.Error != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": tx.Error.Error()})
	}
	if tx.RowsAffected == 0 {
		return nil, errcode.New(errcode.CodeNotFound)
	}

	resp := &model.BatteryWarrantyResp{
		DeviceID:               row.DeviceID,
		DeviceNumber:           row.DeviceNumber,
		BatterySerial:          row.BatterySerial,
		BatteryModelID:         row.BatteryModelID,
		BatteryModelName:       row.BatteryModelName,
		ActivationDate:         formatDatePtr(row.ActivationDate),
		WarrantyStartDate:      formatDatePtr(row.WarrantyStartDate),
		WarrantyExpireDate:     formatDatePtr(row.WarrantyExpireDate),
		WarrantyMonths:         row.WarrantyMonths,
		WarrantyManualOverride: row.WarrantyManualOverride,
		WarrantyUpdatedAt:      formatRFC3339Ptr(row.WarrantyUpdatedAt),
		WarrantyUpdatedBy:      row.WarrantyUpdatedBy,
		Users:                  []model.BatteryWarrantyUserResp{},
	}

	var users []batteryWarrantyUserRow
	if err := global.DB.WithContext(ctx).
		Table("device_user_bindings AS dub").
		Select(`
			u.id AS user_id,
			u.name AS user_name,
			u.username AS username,
			u.phone_number AS user_phone,
			uwi.contact_name AS contact_name,
			uwi.contact_phone AS contact_phone,
			COALESCE(dub.is_owner, false) AS is_owner,
			dub.binding_time AS binding_time
		`).
		Joins(`JOIN users AS u ON u.id = dub.user_id AND u.tenant_id = ?`, claims.TenantID).
		Joins(`LEFT JOIN user_warranty_infos AS uwi ON uwi.tenant_id = u.tenant_id AND uwi.user_id = u.id`).
		Where("dub.device_id = ?", deviceID).
		Order("COALESCE(dub.is_owner, false) DESC, dub.binding_time ASC").
		Scan(&users).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	for _, u := range users {
		resp.Users = append(resp.Users, model.BatteryWarrantyUserResp{
			UserID:       u.UserID,
			UserName:     u.UserName,
			Username:     u.Username,
			UserPhone:    u.UserPhone,
			ContactName:  u.ContactName,
			ContactPhone: u.ContactPhone,
			IsOwner:      u.IsOwner,
			BindingTime:  formatRFC3339Ptr(u.BindingTime),
		})
	}
	return resp, nil
}

func (*UserWarrantyInfo) UpdateBatteryWarranty(ctx context.Context, deviceID string, req *model.BatteryWarrantyUpdateReq, claims *utils.UserClaims, orgID string) (*model.BatteryWarrantyResp, error) {
	if req == nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "request is empty"})
	}
	current, err := GroupApp.UserWarrantyInfo.GetBatteryWarranty(ctx, deviceID, claims, orgID)
	if err != nil {
		return nil, err
	}

	var startDate *time.Time
	if current.WarrantyStartDate != nil {
		startDate, _ = parseDateYYYYMMDD(*current.WarrantyStartDate)
	}
	if req.WarrantyStartDate != nil {
		startDate, err = parseDateYYYYMMDD(*req.WarrantyStartDate)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "质保开始日期格式错误，应为 YYYY-MM-DD"})
		}
	}

	warrantyMonths := current.WarrantyMonths
	if req.WarrantyMonths != nil {
		warrantyMonths = req.WarrantyMonths
	}

	var expireDate *time.Time
	if req.WarrantyExpireDate != nil {
		expireDate, err = parseDateYYYYMMDD(*req.WarrantyExpireDate)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "质保到期日期格式错误，应为 YYYY-MM-DD"})
		}
	} else if req.WarrantyStartDate != nil || req.WarrantyMonths != nil {
		if startDate != nil && warrantyMonths != nil && *warrantyMonths > 0 {
			t := startDate.AddDate(0, int(*warrantyMonths), 0)
			expireDate = &t
		}
	} else if current.WarrantyExpireDate != nil {
		expireDate, _ = parseDateYYYYMMDD(*current.WarrantyExpireDate)
	}

	updates := map[string]interface{}{
		"warranty_manual_override": true,
		"warranty_updated_at":      time.Now().UTC(),
		"warranty_updated_by":      claims.ID,
		"updated_at":               time.Now().UTC(),
	}
	if req.WarrantyMonths != nil {
		updates["warranty_months"] = *req.WarrantyMonths
	}
	if req.WarrantyStartDate != nil {
		updates["warranty_start_date"] = startDate
	}
	if req.WarrantyExpireDate != nil || req.WarrantyStartDate != nil || req.WarrantyMonths != nil {
		updates["warranty_expire_date"] = expireDate
	}

	if err := global.DB.WithContext(ctx).
		Table("device_batteries").
		Where("device_id = ?", deviceID).
		Updates(updates).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return GroupApp.UserWarrantyInfo.GetBatteryWarranty(ctx, deviceID, claims, orgID)
}

func applyBatteryWarrantyActivationTx(ctx context.Context, db *gorm.DB, deviceID, operatorID string, activationAt time.Time) error {
	if db == nil {
		db = global.DB
	}
	var row batteryWarrantyActivationRow
	tx := db.WithContext(ctx).
		Table("device_batteries AS dbat").
		Select(`
			COALESCE(dbat.warranty_months, bm.warranty_months) AS warranty_months,
			dbat.warranty_start_date AS warranty_start_date,
			dbat.warranty_expire_date AS warranty_expire_date,
			COALESCE(dbat.warranty_manual_override, false) AS warranty_manual_override
		`).
		Joins(`LEFT JOIN battery_bms_models AS bm ON bm.id = dbat.battery_model_id`).
		Where("dbat.device_id = ?", strings.TrimSpace(deviceID)).
		Limit(1).
		Scan(&row)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil
	}

	startDate := row.WarrantyStartDate
	if startDate == nil {
		t := activationAt
		startDate = &t
	}
	updates := map[string]interface{}{
		"warranty_start_date": startDate,
		"warranty_updated_at": activationAt,
		"updated_at":          activationAt,
	}
	if row.WarrantyMonths != nil {
		updates["warranty_months"] = *row.WarrantyMonths
	}
	if !row.WarrantyManualOverride && startDate != nil && row.WarrantyMonths != nil && *row.WarrantyMonths > 0 {
		expire := startDate.AddDate(0, int(*row.WarrantyMonths), 0)
		updates["warranty_expire_date"] = expire
	}
	if strings.TrimSpace(operatorID) != "" {
		updates["warranty_updated_by"] = operatorID
	}
	return db.WithContext(ctx).
		Table("device_batteries").
		Where("device_id = ?", strings.TrimSpace(deviceID)).
		Updates(updates).Error
}
