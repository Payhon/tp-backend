package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
)

type appDeviceListRow struct {
	ID               string     `gorm:"column:id"`
	UserID           string     `gorm:"column:user_id"`
	UserName         *string    `gorm:"column:user_name"`
	UserPhone        string     `gorm:"column:user_phone"`
	DeviceID         string     `gorm:"column:device_id"`
	DeviceNumber     string     `gorm:"column:device_number"`
	DeviceName       *string    `gorm:"column:device_name"`
	BleMac           *string    `gorm:"column:ble_mac"`
	Iccid            *string    `gorm:"column:iccid"`
	BmsCommType      *int       `gorm:"column:bms_comm_type"`
	IsOnline         int16      `gorm:"column:is_online"`
	Soc              *float64   `gorm:"column:soc"`
	IsOwner          *bool      `gorm:"column:is_owner"`
	ActivationStatus *string    `gorm:"column:activation_status"`
	BindingTime      *time.Time `gorm:"column:binding_time"`
	AddedAt          *time.Time `gorm:"column:added_at"`
	RelationTime     *time.Time `gorm:"column:relation_time"`
	RelationType     string     `gorm:"column:relation_type"`
}

type appDeviceViewContext struct {
	isAdmin   bool
	userKind  string
	orgID     string
	orgType   string
	isFactory bool
}

func isAdminClaims(claims *utils.UserClaims) bool {
	if claims == nil {
		return false
	}
	return claims.Authority == "SYS_ADMIN" || claims.Authority == "TENANT_ADMIN"
}

func resolveAppDeviceViewContext(ctx context.Context, claims *utils.UserClaims) (*appDeviceViewContext, error) {
	if claims == nil {
		return &appDeviceViewContext{userKind: model.UserKindEndUser}, nil
	}

	tenantID := strings.TrimSpace(claims.TenantID)
	userID := strings.TrimSpace(claims.ID)
	isAdmin := isAdminClaims(claims)
	if tenantID == "" || userID == "" {
		return &appDeviceViewContext{
			isAdmin:   isAdmin,
			userKind:  model.UserKindEndUser,
			orgID:     "",
			orgType:   "",
			isFactory: isAdmin,
		}, nil
	}

	userKind := model.UserKindEndUser
	if isAdmin {
		userKind = model.UserKindOrgUser
	} else {
		var err error
		userKind, err = GroupApp.OrgTypePermission.GetUserKind(ctx, tenantID, userID)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_user_kind",
				"user_id":   userID,
				"error":     err.Error(),
			})
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

	return &appDeviceViewContext{
		isAdmin:   isAdmin,
		userKind:  userKind,
		orgID:     strings.TrimSpace(orgID),
		orgType:   orgType,
		isFactory: isAdmin || orgType == model.OrgTypeBMSFactory || strings.TrimSpace(orgID) == "",
	}, nil
}

func resolveRequestedViewMode(req *model.DeviceUserBindingListReq, viewCtx *appDeviceViewContext) (string, error) {
	mode := model.AppDeviceViewModeSelfBound
	if req != nil && req.ViewMode != nil && strings.TrimSpace(*req.ViewMode) != "" {
		mode = strings.TrimSpace(*req.ViewMode)
	}

	if viewCtx != nil && viewCtx.userKind == model.UserKindOrgUser {
		if req == nil || req.ViewMode == nil || strings.TrimSpace(*req.ViewMode) == "" {
			return model.AppDeviceViewModeOrgAdded, nil
		}
		switch mode {
		case model.AppDeviceViewModeOrgAdded, model.AppDeviceViewModeEndUserBind:
			return mode, nil
		default:
			return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "invalid view_mode for org user",
			})
		}
	}

	if mode == model.AppDeviceViewModeSelfBound || mode == "" {
		return model.AppDeviceViewModeSelfBound, nil
	}
	return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{
		"message": "view_mode is not allowed for current user",
	})
}

func trimPtr(input *string) string {
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

func parseDateStart(input *string) (*time.Time, error) {
	text := trimPtr(input)
	if text == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", text, time.Local)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "invalid added_start_at, expected YYYY-MM-DD",
		})
	}
	utc := t.UTC()
	return &utc, nil
}

func parseDateEnd(input *string) (*time.Time, error) {
	text := trimPtr(input)
	if text == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", text, time.Local)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "invalid added_end_at, expected YYYY-MM-DD",
		})
	}
	end := t.Add(24*time.Hour - time.Nanosecond).UTC()
	return &end, nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(time.Local).Format("2006-01-02 15:04:05")
}

func buildDeviceListResp(rows []appDeviceListRow, total int64, page, pageSize int) *model.DeviceUserBindingListResp {
	list := make([]model.DeviceUserBindingResp, 0, len(rows))
	for _, row := range rows {
		deviceName := row.DeviceNumber
		if row.DeviceName != nil && strings.TrimSpace(*row.DeviceName) != "" {
			deviceName = strings.TrimSpace(*row.DeviceName)
		}

		relationTime := row.RelationTime
		if relationTime == nil {
			if row.BindingTime != nil {
				relationTime = row.BindingTime
			} else {
				relationTime = row.AddedAt
			}
		}

		list = append(list, model.DeviceUserBindingResp{
			ID:               row.ID,
			UserID:           row.UserID,
			UserName:         row.UserName,
			UserPhone:        row.UserPhone,
			DeviceID:         row.DeviceID,
			DeviceNumber:     row.DeviceNumber,
			DeviceName:       deviceName,
			BleMac:           row.BleMac,
			Iccid:            row.Iccid,
			BmsCommType:      row.BmsCommType,
			IsOnline:         row.IsOnline,
			Soc:              row.Soc,
			IsOwner:          row.IsOwner != nil && *row.IsOwner,
			BindingTime:      formatTimePtr(row.BindingTime),
			AddedAt:          formatTimePtr(row.AddedAt),
			RelationTime:     formatTimePtr(relationTime),
			RelationType:     row.RelationType,
			ActivationStatus: row.ActivationStatus,
		})
	}

	return &model.DeviceUserBindingListResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

func addAccessibleOrgWhere(base string, args *[]interface{}, claims *utils.UserClaims, viewCtx *appDeviceViewContext, ownerOrgExpr string) string {
	if claims == nil || viewCtx == nil || viewCtx.isFactory || viewCtx.orgID == "" {
		return base
	}
	base += fmt.Sprintf(" AND %s IN (SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?)", ownerOrgExpr)
	*args = append(*args, claims.TenantID, viewCtx.orgID)
	return base
}

func addDeviceFilters(base string, args *[]interface{}, req model.DeviceUserBindingListReq, allowName bool) string {
	if allowName && trimPtr(req.DeviceName) != "" {
		base += " AND d.name ILIKE ?"
		*args = append(*args, "%"+trimPtr(req.DeviceName)+"%")
	}
	if trimPtr(req.DeviceNumber) != "" {
		base += " AND d.device_number ILIKE ?"
		*args = append(*args, "%"+trimPtr(req.DeviceNumber)+"%")
	}
	if trimPtr(req.BleMac) != "" {
		base += " AND dbat.ble_mac ILIKE ?"
		*args = append(*args, "%"+trimPtr(req.BleMac)+"%")
	}
	return base
}

func (*DeviceBinding) listSelfBoundDevices(ctx context.Context, req model.DeviceUserBindingListReq, claims *utils.UserClaims) (*model.DeviceUserBindingListResp, error) {
	targetUserID := strings.TrimSpace(claims.ID)
	if isAdminClaims(claims) && trimPtr(req.UserID) != "" {
		targetUserID = trimPtr(req.UserID)
	}
	if targetUserID == "" {
		return &model.DeviceUserBindingListResp{List: []model.DeviceUserBindingResp{}, Total: 0, Page: req.Page, PageSize: req.PageSize}, nil
	}

	baseWhere := "dub.user_id = ? AND d.tenant_id = ?"
	args := []interface{}{targetUserID, claims.TenantID}
	baseWhere = addDeviceFilters(baseWhere, &args, req, true)

	countSQL := `
SELECT COUNT(1)
FROM device_user_bindings AS dub
JOIN devices AS d ON d.id = dub.device_id
LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id
WHERE ` + baseWhere
	var total int64
	if err := global.DB.WithContext(ctx).Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if total == 0 {
		return &model.DeviceUserBindingListResp{List: []model.DeviceUserBindingResp{}, Total: 0, Page: req.Page, PageSize: req.PageSize}, nil
	}

	listSQL := `
SELECT
	dub.id AS id,
	dub.user_id AS user_id,
	u.name AS user_name,
	u.phone_number AS user_phone,
	d.id AS device_id,
	d.device_number AS device_number,
	d.name AS device_name,
	dbat.ble_mac AS ble_mac,
	COALESCE(NULLIF(dbat.iccid, ''), NULLIF(dbat.comm_chip_id, '')) AS iccid,
	dbat.bms_comm_type AS bms_comm_type,
	d.is_online AS is_online,
	dbat.soc AS soc,
	dub.is_owner AS is_owner,
	dbat.activation_status AS activation_status,
	dub.binding_time AS binding_time,
	dub.binding_time AS relation_time,
	'BINDING' AS relation_type
FROM device_user_bindings AS dub
JOIN devices AS d ON d.id = dub.device_id
LEFT JOIN users AS u ON u.id = dub.user_id
LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id
WHERE ` + baseWhere + `
ORDER BY dub.binding_time DESC
LIMIT ? OFFSET ?`

	listArgs := append(args, req.PageSize, (req.Page-1)*req.PageSize)
	rows := make([]appDeviceListRow, 0, req.PageSize)
	if err := global.DB.WithContext(ctx).Raw(listSQL, listArgs...).Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return buildDeviceListResp(rows, total, req.Page, req.PageSize), nil
}

func (*DeviceBinding) listOrgAddedDevices(ctx context.Context, req model.DeviceUserBindingListReq, claims *utils.UserClaims, viewCtx *appDeviceViewContext) (*model.DeviceUserBindingListResp, error) {
	baseWhere := "ar.tenant_id = ? AND ar.user_id = ? AND d.tenant_id = ?"
	args := []interface{}{claims.TenantID, claims.ID, claims.TenantID}

	startAt, err := parseDateStart(req.AddedStartAt)
	if err != nil {
		return nil, err
	}
	endAt, err := parseDateEnd(req.AddedEndAt)
	if err != nil {
		return nil, err
	}
	if startAt != nil {
		baseWhere += " AND ar.added_at >= ?"
		args = append(args, *startAt)
	}
	if endAt != nil {
		baseWhere += " AND ar.added_at <= ?"
		args = append(args, *endAt)
	}

	baseWhere = addAccessibleOrgWhere(baseWhere, &args, claims, viewCtx, "dbat.owner_org_id")
	baseWhere = addDeviceFilters(baseWhere, &args, req, false)

	countSQL := `
SELECT COUNT(1)
FROM app_device_added_records AS ar
JOIN devices AS d ON d.id = ar.device_id
LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id
WHERE ` + baseWhere
	var total int64
	if err := global.DB.WithContext(ctx).Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if total == 0 {
		return &model.DeviceUserBindingListResp{List: []model.DeviceUserBindingResp{}, Total: 0, Page: req.Page, PageSize: req.PageSize}, nil
	}

	listSQL := `
SELECT
	ar.id AS id,
	ar.user_id AS user_id,
	u.name AS user_name,
	u.phone_number AS user_phone,
	d.id AS device_id,
	d.device_number AS device_number,
	d.name AS device_name,
	dbat.ble_mac AS ble_mac,
	COALESCE(NULLIF(dbat.iccid, ''), NULLIF(dbat.comm_chip_id, '')) AS iccid,
	dbat.bms_comm_type AS bms_comm_type,
	d.is_online AS is_online,
	dbat.soc AS soc,
	dbat.activation_status AS activation_status,
	ar.added_at AS added_at,
	COALESCE(ar.last_seen_at, ar.added_at) AS relation_time,
	'ORG_ADDED' AS relation_type
FROM app_device_added_records AS ar
JOIN devices AS d ON d.id = ar.device_id
LEFT JOIN users AS u ON u.id = ar.user_id
LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id
WHERE ` + baseWhere + `
ORDER BY ar.added_at DESC
LIMIT ? OFFSET ?`

	listArgs := append(args, req.PageSize, (req.Page-1)*req.PageSize)
	rows := make([]appDeviceListRow, 0, req.PageSize)
	if err := global.DB.WithContext(ctx).Raw(listSQL, listArgs...).Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return buildDeviceListResp(rows, total, req.Page, req.PageSize), nil
}

func (*DeviceBinding) listOrgEndUserBoundDevices(ctx context.Context, req model.DeviceUserBindingListReq, claims *utils.UserClaims, viewCtx *appDeviceViewContext) (*model.DeviceUserBindingListResp, error) {
	baseWhere := "d.tenant_id = ? AND COALESCE(u.user_kind, 'END_USER') = 'END_USER'"
	args := []interface{}{claims.TenantID}
	baseWhere = addAccessibleOrgWhere(baseWhere, &args, claims, viewCtx, "dbat.owner_org_id")
	baseWhere = addDeviceFilters(baseWhere, &args, req, true)

	withClause := `
WITH latest_bind AS (
	SELECT DISTINCT ON (b.device_id)
		b.id,
		b.device_id,
		b.user_id,
		b.binding_time,
		b.is_owner
	FROM device_user_bindings AS b
	ORDER BY b.device_id, COALESCE(b.is_owner, FALSE) DESC, b.binding_time DESC
) `

	countSQL := withClause + `
SELECT COUNT(1)
FROM latest_bind AS lb
JOIN devices AS d ON d.id = lb.device_id
JOIN users AS u ON u.id = lb.user_id
LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id
WHERE ` + baseWhere
	var total int64
	if err := global.DB.WithContext(ctx).Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if total == 0 {
		return &model.DeviceUserBindingListResp{List: []model.DeviceUserBindingResp{}, Total: 0, Page: req.Page, PageSize: req.PageSize}, nil
	}

	listSQL := withClause + `
SELECT
	lb.id AS id,
	lb.user_id AS user_id,
	u.name AS user_name,
	u.phone_number AS user_phone,
	d.id AS device_id,
	d.device_number AS device_number,
	d.name AS device_name,
	dbat.ble_mac AS ble_mac,
	COALESCE(NULLIF(dbat.iccid, ''), NULLIF(dbat.comm_chip_id, '')) AS iccid,
	dbat.bms_comm_type AS bms_comm_type,
	d.is_online AS is_online,
	dbat.soc AS soc,
	lb.is_owner AS is_owner,
	dbat.activation_status AS activation_status,
	lb.binding_time AS binding_time,
	lb.binding_time AS relation_time,
	'END_USER_BOUND' AS relation_type
FROM latest_bind AS lb
JOIN devices AS d ON d.id = lb.device_id
JOIN users AS u ON u.id = lb.user_id
LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id
WHERE ` + baseWhere + `
ORDER BY lb.binding_time DESC
LIMIT ? OFFSET ?`

	listArgs := append(args, req.PageSize, (req.Page-1)*req.PageSize)
	rows := make([]appDeviceListRow, 0, req.PageSize)
	if err := global.DB.WithContext(ctx).Raw(listSQL, listArgs...).Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return buildDeviceListResp(rows, total, req.Page, req.PageSize), nil
}

func (*DeviceBinding) upsertOrgAddedDeviceRecord(ctx context.Context, claims *utils.UserClaims, deviceID, source string) error {
	now := time.Now().UTC()
	if err := global.DB.WithContext(ctx).Exec(
		`INSERT INTO app_device_added_records (id, tenant_id, user_id, device_id, source, added_at, last_seen_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (tenant_id, user_id, device_id)
		 DO UPDATE SET source = EXCLUDED.source, last_seen_at = EXCLUDED.last_seen_at`,
		uuid.New(), claims.TenantID, claims.ID, deviceID, source, now, now,
	).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func (*DeviceBinding) RemoveOrgAddedDevice(req model.AppDeviceRemoveReq, claims *utils.UserClaims) error {
	ctx := context.Background()
	viewCtx, err := resolveAppDeviceViewContext(ctx, claims)
	if err != nil {
		return err
	}
	if viewCtx.userKind != model.UserKindOrgUser {
		return errcode.New(errcode.CodeNoPermission)
	}

	res := global.DB.WithContext(ctx).Exec(
		"DELETE FROM app_device_added_records WHERE tenant_id = ? AND user_id = ? AND device_id = ?",
		claims.TenantID, claims.ID, req.DeviceID,
	)
	if res.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "record not found"})
	}
	return nil
}
