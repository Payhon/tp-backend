package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AppManage APP管理（应用管理/升级中心）
type AppManage struct{}

type appRow struct {
	ID          string     `gorm:"column:id"`
	AppID       string     `gorm:"column:appid"`
	AppType     int16      `gorm:"column:app_type"`
	Name        string     `gorm:"column:name"`
	Description *string    `gorm:"column:description"`
	Remark      *string    `gorm:"column:remark"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
	IconURL     *string    `gorm:"column:icon_url"`
	Introduction *string   `gorm:"column:introduction"`
	Screenshot  datatypes.JSON `gorm:"column:screenshot"`
	AppAndroid  datatypes.JSON `gorm:"column:app_android"`
	AppIOS      datatypes.JSON `gorm:"column:app_ios"`
	AppHarmony  datatypes.JSON `gorm:"column:app_harmony"`
	H5          datatypes.JSON `gorm:"column:h5"`
	QuickApp    datatypes.JSON `gorm:"column:quickapp"`
	StoreList   datatypes.JSON `gorm:"column:store_list"`
}

func formatLocalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(time.Local).Format("2006-01-02 15:04:05")
}

func rawJSONPtr(j datatypes.JSON) *json.RawMessage {
	if len(j) == 0 {
		return nil
	}
	raw := json.RawMessage(j)
	return &raw
}

func jsonOrNil(raw *json.RawMessage) interface{} {
	if raw == nil {
		return nil
	}
	if len(*raw) == 0 {
		return datatypes.JSON([]byte("null"))
	}
	return datatypes.JSON(*raw)
}

func jsonArrayOfStrings(in []string) datatypes.JSON {
	if len(in) == 0 {
		return datatypes.JSON([]byte("[]"))
	}
	b, _ := json.Marshal(in)
	return datatypes.JSON(b)
}

// ListApps 应用管理列表
func (*AppManage) ListApps(ctx context.Context, req model.AppListReq, claims *utils.UserClaims) (*model.AppListResp, error) {
	db := global.DB.WithContext(ctx).Table("apps").Where("tenant_id = ?", claims.TenantID)

	if req.Keyword != nil && strings.TrimSpace(*req.Keyword) != "" {
		kw := "%" + strings.TrimSpace(*req.Keyword) + "%"
		db = db.Where("(appid ILIKE ? OR name ILIKE ?)", kw, kw)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (req.Page - 1) * req.PageSize
	rows := make([]appRow, 0, req.PageSize)
	if err := db.Select("id, appid, app_type, name, description, remark, created_at").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.AppListItemResp, 0, len(rows))
	for _, r := range rows {
		list = append(list, model.AppListItemResp{
			ID:          r.ID,
			AppID:       r.AppID,
			AppType:     r.AppType,
			Name:        r.Name,
			Description: r.Description,
			Remark:      r.Remark,
			CreatedAt:   formatLocalTime(r.CreatedAt),
		})
	}

	return &model.AppListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetApp 应用详情
func (*AppManage) GetApp(ctx context.Context, id string, claims *utils.UserClaims) (*model.AppDetailResp, error) {
	db := global.DB.WithContext(ctx)
	var r appRow
	if err := db.Table("apps").
		Where("tenant_id = ? AND id = ?", claims.TenantID, id).
		Select(`id, appid, app_type, name, description, icon_url, introduction,
			screenshot, app_android, app_ios, app_harmony, h5, quickapp, store_list,
			remark, created_at, updated_at`).
		Scan(&r).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if r.ID == "" {
		return nil, errcode.New(errcode.CodeNotFound)
	}

	var screenshots []string
	if len(r.Screenshot) != 0 {
		_ = json.Unmarshal(r.Screenshot, &screenshots)
	}

	return &model.AppDetailResp{
		ID:           r.ID,
		AppID:        r.AppID,
		AppType:      r.AppType,
		Name:         r.Name,
		Description:  r.Description,
		IconURL:      r.IconURL,
		Introduction: r.Introduction,
		Screenshot:   screenshots,
		AppAndroid:   rawJSONPtr(r.AppAndroid),
		AppIOS:       rawJSONPtr(r.AppIOS),
		AppHarmony:   rawJSONPtr(r.AppHarmony),
		H5:           rawJSONPtr(r.H5),
		QuickApp:     rawJSONPtr(r.QuickApp),
		StoreList:    rawJSONPtr(r.StoreList),
		Remark:       r.Remark,
		CreatedAt:    formatLocalTime(r.CreatedAt),
		UpdatedAt:    formatLocalTime(r.UpdatedAt),
	}, nil
}

// CreateApp 创建应用
func (*AppManage) CreateApp(ctx context.Context, req model.AppCreateReq, claims *utils.UserClaims) (string, error) {
	now := time.Now().UTC()
	id := uuid.NewString()
	appType := int16(0)
	if req.AppType != nil {
		appType = *req.AppType
	}

	storeList := datatypes.JSON([]byte("[]"))
	if req.StoreList != nil && len(*req.StoreList) != 0 {
		storeList = datatypes.JSON(*req.StoreList)
	}

	if err := global.DB.WithContext(ctx).Table("apps").Create(map[string]interface{}{
		"id":           id,
		"tenant_id":     claims.TenantID,
		"appid":         strings.TrimSpace(req.AppID),
		"app_type":      appType,
		"name":          strings.TrimSpace(req.Name),
		"description":   req.Description,
		"creator_uid":   req.CreatorUID,
		"owner_type":    req.OwnerType,
		"owner_id":      req.OwnerID,
		"managers":      jsonArrayOfStrings(req.Managers),
		"members":       jsonArrayOfStrings(req.Members),
		"icon_url":      req.IconURL,
		"introduction":  req.Introduction,
		"screenshot":    jsonArrayOfStrings(req.Screenshot),
		"app_android":   jsonOrNil(req.AppAndroid),
		"app_ios":       jsonOrNil(req.AppIOS),
		"app_harmony":   jsonOrNil(req.AppHarmony),
		"h5":            jsonOrNil(req.H5),
		"quickapp":      jsonOrNil(req.QuickApp),
		"store_list":    storeList,
		"remark":        req.Remark,
		"created_at":    now,
		"updated_at":    now,
		"mp_weixin":     jsonOrNil(req.MPWeixin),
		"mp_alipay":     jsonOrNil(req.MPAlipay),
		"mp_baidu":      jsonOrNil(req.MPBaidu),
		"mp_toutiao":    jsonOrNil(req.MPToutiao),
		"mp_qq":         jsonOrNil(req.MPQQ),
		"mp_kuaishou":   jsonOrNil(req.MPKuaishou),
		"mp_lark":       jsonOrNil(req.MPLark),
		"mp_jd":         jsonOrNil(req.MPJD),
		"mp_dingtalk":   jsonOrNil(req.MPDingtalk),
	}).Error; err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return id, nil
}

// UpdateApp 更新应用
func (*AppManage) UpdateApp(ctx context.Context, id string, req model.AppUpdateReq, claims *utils.UserClaims) error {
	now := time.Now().UTC()

	updates := map[string]interface{}{
		"updated_at": now,
	}
	if req.AppType != nil {
		updates["app_type"] = *req.AppType
	}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.IconURL != nil {
		updates["icon_url"] = req.IconURL
	}
	if req.Introduction != nil {
		updates["introduction"] = req.Introduction
	}
	if req.Screenshot != nil {
		updates["screenshot"] = jsonArrayOfStrings(req.Screenshot)
	}
	if req.AppAndroid != nil {
		updates["app_android"] = jsonOrNil(req.AppAndroid)
	}
	if req.AppIOS != nil {
		updates["app_ios"] = jsonOrNil(req.AppIOS)
	}
	if req.AppHarmony != nil {
		updates["app_harmony"] = jsonOrNil(req.AppHarmony)
	}
	if req.H5 != nil {
		updates["h5"] = jsonOrNil(req.H5)
	}
	if req.QuickApp != nil {
		updates["quickapp"] = jsonOrNil(req.QuickApp)
	}
	if req.StoreList != nil {
		updates["store_list"] = datatypes.JSON(*req.StoreList)
	}
	if req.Remark != nil {
		updates["remark"] = req.Remark
	}
	if req.Managers != nil {
		updates["managers"] = jsonArrayOfStrings(req.Managers)
	}
	if req.Members != nil {
		updates["members"] = jsonArrayOfStrings(req.Members)
	}
	if req.OwnerType != nil {
		updates["owner_type"] = *req.OwnerType
	}
	if req.OwnerID != nil {
		updates["owner_id"] = req.OwnerID
	}

	res := global.DB.WithContext(ctx).Table("apps").
		Where("tenant_id = ? AND id = ?", claims.TenantID, id).
		Updates(updates)
	if res.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return errcode.New(errcode.CodeNotFound)
	}
	return nil
}

// DeleteApp 删除应用
func (*AppManage) DeleteApp(ctx context.Context, id string, claims *utils.UserClaims) error {
	res := global.DB.WithContext(ctx).Table("apps").
		Where("tenant_id = ? AND id = ?", claims.TenantID, id).
		Delete(map[string]interface{}{})
	if res.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return errcode.New(errcode.CodeNotFound)
	}
	return nil
}

// BatchDeleteApps 批量删除应用
func (*AppManage) BatchDeleteApps(ctx context.Context, ids []string, claims *utils.UserClaims) error {
	if len(ids) == 0 {
		return nil
	}
	res := global.DB.WithContext(ctx).Table("apps").
		Where("tenant_id = ? AND id IN ?", claims.TenantID, ids).
		Delete(map[string]interface{}{})
	if res.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": res.Error.Error()})
	}
	return nil
}

// -----------------------------------------------------------------------------
// App Versions
// -----------------------------------------------------------------------------

type appVersionRow struct {
	ID            string        `gorm:"column:id"`
	AppID         string        `gorm:"column:appid"`
	AppName       string        `gorm:"column:name"`
	Title         *string       `gorm:"column:title"`
	Contents      *string       `gorm:"column:contents"`
	Platform      datatypes.JSON `gorm:"column:platform"`
	Type          string        `gorm:"column:type"`
	Version       string        `gorm:"column:version"`
	MinUniVersion *string       `gorm:"column:min_uni_version"`
	URL           *string       `gorm:"column:url"`
	StablePublish bool          `gorm:"column:stable_publish"`
	IsSilently    bool          `gorm:"column:is_silently"`
	IsMandatory   bool          `gorm:"column:is_mandatory"`
	UniPlatform   string        `gorm:"column:uni_platform"`
	CreateEnv     string        `gorm:"column:create_env"`
	CreateDate    *time.Time    `gorm:"column:create_date"`
}

func parsePlatform(j datatypes.JSON) []string {
	if len(j) == 0 {
		return []string{}
	}
	var out []string
	_ = json.Unmarshal(j, &out)
	return out
}

// ListAppVersions 升级中心列表
func (*AppManage) ListAppVersions(ctx context.Context, req model.AppVersionListReq, claims *utils.UserClaims) (*model.AppVersionListResp, error) {
	db := global.DB.WithContext(ctx).Table("app_versions").Where("tenant_id = ?", claims.TenantID)

	if req.AppID != nil && strings.TrimSpace(*req.AppID) != "" {
		db = db.Where("app_id = ?", strings.TrimSpace(*req.AppID))
	}
	if req.Type != nil && strings.TrimSpace(*req.Type) != "" {
		db = db.Where("type = ?", strings.TrimSpace(*req.Type))
	}
	if req.Keyword != nil && strings.TrimSpace(*req.Keyword) != "" {
		kw := "%" + strings.TrimSpace(*req.Keyword) + "%"
		db = db.Where("(title ILIKE ? OR version ILIKE ?)", kw, kw)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (req.Page - 1) * req.PageSize
	rows := make([]appVersionRow, 0, req.PageSize)
	if err := db.Select("id, appid, name, title, type, platform, version, stable_publish, create_date").
		Order("create_date DESC").
		Offset(offset).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.AppVersionListItemResp, 0, len(rows))
	for _, r := range rows {
		list = append(list, model.AppVersionListItemResp{
			ID:            r.ID,
			AppID:         r.AppID,
			AppName:       r.AppName,
			Title:         r.Title,
			Type:          r.Type,
			Platform:      parsePlatform(r.Platform),
			Version:       r.Version,
			StablePublish: r.StablePublish,
			CreateDate:    formatLocalTime(r.CreateDate),
		})
	}

	return &model.AppVersionListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetAppVersion 版本详情
func (*AppManage) GetAppVersion(ctx context.Context, id string, claims *utils.UserClaims) (*model.AppVersionDetailResp, error) {
	var r appVersionRow
	if err := global.DB.WithContext(ctx).Table("app_versions").
		Where("tenant_id = ? AND id = ?", claims.TenantID, id).
		Select(`id, appid, name, title, contents, platform, type, version, min_uni_version, url,
			stable_publish, is_silently, is_mandatory, uni_platform, create_env, create_date`).
		Scan(&r).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if r.ID == "" {
		return nil, errcode.New(errcode.CodeNotFound)
	}
	return &model.AppVersionDetailResp{
		ID:            r.ID,
		AppID:         r.AppID,
		AppName:       r.AppName,
		Title:         r.Title,
		Contents:      r.Contents,
		Platform:      parsePlatform(r.Platform),
		Type:          r.Type,
		Version:       r.Version,
		MinUniVersion: r.MinUniVersion,
		URL:           r.URL,
		StablePublish: r.StablePublish,
		IsSilently:    r.IsSilently,
		IsMandatory:   r.IsMandatory,
		UniPlatform:   r.UniPlatform,
		CreateEnv:     r.CreateEnv,
		CreateDate:    formatLocalTime(r.CreateDate),
	}, nil
}

// CreateAppVersion 发布新版
func (*AppManage) CreateAppVersion(ctx context.Context, req model.AppVersionCreateReq, claims *utils.UserClaims) (string, error) {
	db := global.DB.WithContext(ctx)

	// 读取 app 信息
	var app struct {
		ID    string `gorm:"column:id"`
		AppID string `gorm:"column:appid"`
		Name  string `gorm:"column:name"`
	}
	if err := db.Table("apps").
		Where("tenant_id = ? AND id = ?", claims.TenantID, req.AppID).
		Select("id, appid, name").
		Scan(&app).Error; err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if app.ID == "" {
		return "", errcode.New(errcode.CodeNotFound)
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	createEnv := "upgrade-center"
	if req.CreateEnv != nil && strings.TrimSpace(*req.CreateEnv) != "" {
		createEnv = strings.TrimSpace(*req.CreateEnv)
	}

	stable := false
	if req.StablePublish != nil {
		stable = *req.StablePublish
	}
	isSilently := false
	if req.IsSilently != nil {
		isSilently = *req.IsSilently
	}
	isMandatory := false
	if req.IsMandatory != nil {
		isMandatory = *req.IsMandatory
	}

	var storeList datatypes.JSON = datatypes.JSON([]byte("[]"))
	if req.StoreList != nil && len(*req.StoreList) != 0 {
		storeList = datatypes.JSON(*req.StoreList)
	}

	if err := db.Table("app_versions").Create(map[string]interface{}{
		"id":            id,
		"tenant_id":      claims.TenantID,
		"app_id":         app.ID,
		"appid":          app.AppID,
		"name":           app.Name,
		"title":          req.Title,
		"contents":       req.Contents,
		"platform":       jsonArrayOfStrings(req.Platform),
		"type":           req.Type,
		"version":        req.Version,
		"min_uni_version": req.MinUniVersion,
		"url":            req.URL,
		"stable_publish": stable,
		"is_silently":    isSilently,
		"is_mandatory":   isMandatory,
		"create_date":    now,
		"uni_platform":   req.UniPlatform,
		"create_env":     createEnv,
		"store_list":     storeList,
		"created_at":     now,
		"updated_at":     now,
	}).Error; err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return id, nil
}

// UpdateAppVersion 更新版本信息（上线/下线等）
func (*AppManage) UpdateAppVersion(ctx context.Context, id string, req model.AppVersionUpdateReq, claims *utils.UserClaims) error {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"updated_at": now,
	}
	if req.Title != nil {
		updates["title"] = req.Title
	}
	if req.Contents != nil {
		updates["contents"] = req.Contents
	}
	if req.Platform != nil {
		updates["platform"] = jsonArrayOfStrings(req.Platform)
	}
	if req.Version != nil {
		updates["version"] = *req.Version
	}
	if req.MinUniVersion != nil {
		updates["min_uni_version"] = req.MinUniVersion
	}
	if req.URL != nil {
		updates["url"] = req.URL
	}
	if req.StablePublish != nil {
		updates["stable_publish"] = *req.StablePublish
	}
	if req.IsSilently != nil {
		updates["is_silently"] = *req.IsSilently
	}
	if req.IsMandatory != nil {
		updates["is_mandatory"] = *req.IsMandatory
	}
	if req.UniPlatform != nil {
		updates["uni_platform"] = *req.UniPlatform
	}
	if req.StoreList != nil {
		updates["store_list"] = datatypes.JSON(*req.StoreList)
	}

	res := global.DB.WithContext(ctx).Table("app_versions").
		Where("tenant_id = ? AND id = ?", claims.TenantID, id).
		Updates(updates)
	if res.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return errcode.New(errcode.CodeNotFound)
	}
	return nil
}

// DeleteAppVersion 删除版本
func (*AppManage) DeleteAppVersion(ctx context.Context, id string, claims *utils.UserClaims) error {
	res := global.DB.WithContext(ctx).Table("app_versions").
		Where("tenant_id = ? AND id = ?", claims.TenantID, id).
		Delete(map[string]interface{}{})
	if res.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return errcode.New(errcode.CodeNotFound)
	}
	return nil
}

// BatchDeleteAppVersions 批量删除版本
func (*AppManage) BatchDeleteAppVersions(ctx context.Context, ids []string, claims *utils.UserClaims) error {
	if len(ids) == 0 {
		return nil
	}
	res := global.DB.WithContext(ctx).Table("app_versions").
		Where("tenant_id = ? AND id IN ?", claims.TenantID, ids).
		Delete(map[string]interface{}{})
	if res.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": res.Error.Error()})
	}
	return nil
}
