package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	"project/pkg/global"
	"project/pkg/utils"
)

type AppUsers struct{}

func formatAppUserTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.In(time.Local).Format("2006-01-02 15:04:05")
	return &s
}

func (*AppUsers) SourceOptions(ctx context.Context, claims *utils.UserClaims) ([]model.AppUserSourceOptionResp, error) {
	type wxmpRow struct {
		WxAppID string  `gorm:"column:wx_appid"`
		AppName *string `gorm:"column:app_name"`
		OrgName *string `gorm:"column:org_name"`
	}
	rows := make([]wxmpRow, 0)
	if err := global.DB.WithContext(ctx).
		Table("pack_wxmp_configs AS p").
		Select("p.wx_appid, a.name AS app_name, o.name AS org_name").
		Joins("LEFT JOIN apps AS a ON a.id = p.app_id").
		Joins("LEFT JOIN orgs AS o ON o.id = p.org_id").
		Where("p.tenant_id = ?", claims.TenantID).
		Order("a.name ASC, o.name ASC, p.wx_appid ASC").
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	children := make([]model.AppUserSourceOptionResp, 0, len(rows))
	for _, row := range rows {
		label := strings.TrimSpace(row.WxAppID)
		if row.AppName != nil && strings.TrimSpace(*row.AppName) != "" {
			label = strings.TrimSpace(*row.AppName)
		} else if row.OrgName != nil && strings.TrimSpace(*row.OrgName) != "" {
			label = strings.TrimSpace(*row.OrgName)
		}
		wxAppID := row.WxAppID
		children = append(children, model.AppUserSourceOptionResp{
			Key:        "WXMP:" + row.WxAppID,
			Label:      label,
			SourceType: string(model.AppUserSourceTypeWXMP),
			WxAppID:    &wxAppID,
		})
	}

	return []model.AppUserSourceOptionResp{
		{Key: "APP", Label: "App 端", SourceType: string(model.AppUserSourceTypeAPP)},
		{Key: "WXMP", Label: "微信小程序端", SourceType: string(model.AppUserSourceTypeWXMP), Children: children},
	}, nil
}

func (*AppUsers) List(ctx context.Context, req model.AppUserListReq, claims *utils.UserClaims) (*model.AppUserListResp, error) {
	sourceType := strings.ToUpper(strings.TrimSpace(req.SourceType))
	if sourceType == string(model.AppUserSourceTypeWXMP) {
		return listWxmpAppUsers(ctx, req, claims)
	}
	return listNativeAppUsers(ctx, req, claims)
}

type appUserRow struct {
	ID            string     `gorm:"column:id"`
	PhoneNumber   string     `gorm:"column:phone_number"`
	Email         string     `gorm:"column:email"`
	Username      *string    `gorm:"column:username"`
	Name          *string    `gorm:"column:name"`
	Status        *string    `gorm:"column:status"`
	UserKind      *string    `gorm:"column:user_kind"`
	SourceType    string     `gorm:"column:source_type"`
	SourceName    string     `gorm:"column:source_name"`
	WxAppID       *string    `gorm:"column:wx_appid"`
	IdentityTypes string     `gorm:"column:identity_types"`
	DeviceCount   int64      `gorm:"column:device_count"`
	LastBindAt    *time.Time `gorm:"column:last_bind_at"`
	LastVisitTime *time.Time `gorm:"column:last_visit_time"`
	CreatedAt     *time.Time `gorm:"column:created_at"`
}

func appendAppUserFilters(where []string, args []interface{}, req model.AppUserListReq) ([]string, []interface{}) {
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		where = append(where, "u.status = ?")
		args = append(args, strings.TrimSpace(*req.Status))
	}
	if req.Keyword != nil && strings.TrimSpace(*req.Keyword) != "" {
		kw := "%" + strings.TrimSpace(*req.Keyword) + "%"
		where = append(where, "(u.phone_number ILIKE ? OR u.email ILIKE ? OR u.username ILIKE ? OR u.name ILIKE ?)")
		args = append(args, kw, kw, kw, kw)
	}
	return where, args
}

func scanAppUserRows(ctx context.Context, rowsSQL string, countSQL string, args []interface{}, countArgs []interface{}, req model.AppUserListReq) (*model.AppUserListResp, error) {
	var total int64
	if err := global.DB.WithContext(ctx).Raw(countSQL, countArgs...).Scan(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	rows := make([]appUserRow, 0, req.PageSize)
	if err := global.DB.WithContext(ctx).Raw(rowsSQL, args...).Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	list := make([]model.AppUserListItemResp, 0, len(rows))
	for _, row := range rows {
		list = append(list, model.AppUserListItemResp{
			ID:            row.ID,
			PhoneNumber:   row.PhoneNumber,
			Email:         row.Email,
			Username:      row.Username,
			Name:          row.Name,
			Status:        row.Status,
			UserKind:      row.UserKind,
			SourceType:    row.SourceType,
			SourceName:    row.SourceName,
			WxAppID:       row.WxAppID,
			IdentityTypes: row.IdentityTypes,
			DeviceCount:   row.DeviceCount,
			LastBindAt:    formatAppUserTime(row.LastBindAt),
			LastVisitTime: formatAppUserTime(row.LastVisitTime),
			CreatedAt:     formatAppUserTime(row.CreatedAt),
		})
	}
	return &model.AppUserListResp{List: list, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

func listNativeAppUsers(ctx context.Context, req model.AppUserListReq, claims *utils.UserClaims) (*model.AppUserListResp, error) {
	where := []string{"u.tenant_id = ?", "u.user_kind = ?", "EXISTS (SELECT 1 FROM user_identities ui WHERE ui.tenant_id = u.tenant_id AND ui.user_id = u.id AND ui.identity_type IN ('PHONE', 'EMAIL'))"}
	args := []interface{}{claims.TenantID, model.UserKindEndUser}
	where, args = appendAppUserFilters(where, args, req)
	whereSQL := strings.Join(where, " AND ")
	baseSQL := fmt.Sprintf(`
		FROM users AS u
		LEFT JOIN (
			SELECT tenant_id, user_id, STRING_AGG(DISTINCT identity_type, ',' ORDER BY identity_type) AS identity_types
			FROM user_identities
			GROUP BY tenant_id, user_id
		) AS ids ON ids.tenant_id = u.tenant_id AND ids.user_id = u.id
		LEFT JOIN (
			SELECT dub.user_id, COUNT(DISTINCT dub.device_id) AS device_count, MAX(dub.binding_time) AS last_bind_at
			FROM device_user_bindings AS dub
			JOIN devices AS d ON d.id = dub.device_id
			WHERE d.tenant_id = ?
			GROUP BY dub.user_id
		) AS ds ON ds.user_id = u.id
		WHERE %s`, whereSQL)
	countArgs := append([]interface{}{claims.TenantID}, args...)
	rowsArgs := append([]interface{}{claims.TenantID}, args...)
	rowsArgs = append(rowsArgs, (req.Page-1)*req.PageSize, req.PageSize)
	rowsSQL := `
		SELECT u.id, u.phone_number, u.email, u.username, u.name, u.status, u.user_kind,
			'APP' AS source_type, 'App 端' AS source_name, NULL AS wx_appid,
			COALESCE(ids.identity_types, '') AS identity_types,
			COALESCE(ds.device_count, 0) AS device_count, ds.last_bind_at, u.last_visit_time, u.created_at
	` + baseSQL + ` ORDER BY u.created_at DESC OFFSET ? LIMIT ?`
	countSQL := `SELECT COUNT(*) ` + baseSQL
	return scanAppUserRows(ctx, rowsSQL, countSQL, rowsArgs, countArgs, req)
}

func listWxmpAppUsers(ctx context.Context, req model.AppUserListReq, claims *utils.UserClaims) (*model.AppUserListResp, error) {
	where := []string{"u.tenant_id = ?", "u.user_kind = ?", "ui.identity_type = 'WXMP_OPENID'"}
	args := []interface{}{claims.TenantID, model.UserKindEndUser}
	if req.WxAppID != nil && strings.TrimSpace(*req.WxAppID) != "" {
		where = append(where, "SPLIT_PART(ui.identifier, ':', 1) = ?")
		args = append(args, strings.TrimSpace(*req.WxAppID))
	}
	where, args = appendAppUserFilters(where, args, req)
	whereSQL := strings.Join(where, " AND ")
	baseSQL := fmt.Sprintf(`
		FROM users AS u
		JOIN user_identities AS ui ON ui.tenant_id = u.tenant_id AND ui.user_id = u.id
		LEFT JOIN pack_wxmp_configs AS p ON p.tenant_id = u.tenant_id AND p.wx_appid = SPLIT_PART(ui.identifier, ':', 1)
		LEFT JOIN apps AS a ON a.id = p.app_id
		LEFT JOIN orgs AS o ON o.id = p.org_id
		LEFT JOIN (
			SELECT tenant_id, user_id, STRING_AGG(DISTINCT identity_type, ',' ORDER BY identity_type) AS identity_types
			FROM user_identities
			GROUP BY tenant_id, user_id
		) AS ids ON ids.tenant_id = u.tenant_id AND ids.user_id = u.id
		LEFT JOIN (
			SELECT dub.user_id, COUNT(DISTINCT dub.device_id) AS device_count, MAX(dub.binding_time) AS last_bind_at
			FROM device_user_bindings AS dub
			JOIN devices AS d ON d.id = dub.device_id
			WHERE d.tenant_id = ?
			GROUP BY dub.user_id
		) AS ds ON ds.user_id = u.id
		WHERE %s`, whereSQL)
	countArgs := append([]interface{}{claims.TenantID}, args...)
	rowsArgs := append([]interface{}{claims.TenantID}, args...)
	rowsArgs = append(rowsArgs, (req.Page-1)*req.PageSize, req.PageSize)
	rowsSQL := `
		SELECT u.id, u.phone_number, u.email, u.username, u.name, u.status, u.user_kind,
			'WXMP' AS source_type,
			COALESCE(NULLIF(a.name, ''), NULLIF(o.name, ''), SPLIT_PART(ui.identifier, ':', 1)) AS source_name,
			SPLIT_PART(ui.identifier, ':', 1) AS wx_appid,
			COALESCE(ids.identity_types, '') AS identity_types,
			COALESCE(ds.device_count, 0) AS device_count, ds.last_bind_at, u.last_visit_time, u.created_at
	` + baseSQL + ` ORDER BY u.created_at DESC OFFSET ? LIMIT ?`
	countSQL := `SELECT COUNT(*) FROM (SELECT u.id, SPLIT_PART(ui.identifier, ':', 1) ` + baseSQL + ` GROUP BY u.id, SPLIT_PART(ui.identifier, ':', 1)) AS t`
	return scanAppUserRows(ctx, rowsSQL, countSQL, rowsArgs, countArgs, req)
}
