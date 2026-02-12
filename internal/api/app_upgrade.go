package api

import (
	"encoding/json"
	"strings"
	"time"

	"project/internal/middleware"
	"project/internal/model"
	global "project/pkg/global"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// AppUpgradeApi APP 升级（APP端检查更新）
//
// 注意：此接口用于替代 uniCloud 云函数，返回结构对齐 uni-upgrade-center。
// 为避免客户端通用 request 封装把“业务上的无更新”当成接口错误，这里始终返回外层 code=200，
// 业务状态通过内层 data.code（>0/0/<0）表达。
type AppUpgradeApi struct{}

type appRowLite struct {
	AppID string `gorm:"column:appid"`
	Name  string `gorm:"column:name"`
}

type appVersionRow struct {
	ID            string         `gorm:"column:id"`
	AppID         string         `gorm:"column:appid"`
	Name          string         `gorm:"column:name"`
	Title         *string        `gorm:"column:title"`
	Contents      *string        `gorm:"column:contents"`
	Platform      datatypes.JSON `gorm:"column:platform"`
	Type          string         `gorm:"column:type"`
	Version       string         `gorm:"column:version"`
	MinUniVersion *string        `gorm:"column:min_uni_version"`
	URL           *string        `gorm:"column:url"`
	StablePublish bool           `gorm:"column:stable_publish"`
	IsSilently    bool           `gorm:"column:is_silently"`
	IsMandatory   bool           `gorm:"column:is_mandatory"`
	UniPlatform   string         `gorm:"column:uni_platform"`
	CreateEnv     string         `gorm:"column:create_env"`
	CreateDate    time.Time      `gorm:"column:create_date"`
	StoreList     datatypes.JSON `gorm:"column:store_list"`
}

func compareVersion(v1, v2 string) int {
	// 对齐 uni-upgrade-center-app/utils/utils.ts 的 compare 规则：
	// - 按 '.' 分段；数值比较；较长版本剩余段有 >0 则更大
	a1 := strings.Split(strings.TrimSpace(v1), ".")
	a2 := strings.Split(strings.TrimSpace(v2), ".")
	minLen := len(a1)
	if len(a2) < minLen {
		minLen = len(a2)
	}

	for i := 0; i < minLen; i++ {
		n1 := toInt(a1[i])
		n2 := toInt(a2[i])
		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}

	if len(a1) == len(a2) {
		return 0
	}

	var rest []string
	v1Longer := len(a1) > len(a2)
	if v1Longer {
		rest = a1[minLen:]
	} else {
		rest = a2[minLen:]
	}
	for _, s := range rest {
		if toInt(s) > 0 {
			if v1Longer {
				return 1
			}
			return -1
		}
	}
	return 0
}

func toInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// 仅取前缀数字，避免出现 "1-beta" 之类导致解析失败
	n := 0
	sign := 1
	for i, r := range s {
		if i == 0 && r == '-' {
			sign = -1
			continue
		}
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n * sign
}

func normalizeUniPlatform(p string) string {
	p = strings.TrimSpace(strings.ToLower(p))
	switch p {
	case "android":
		return "android"
	case "ios":
		return "ios"
	case "harmony", "harmonyos":
		return "harmony"
	case "app":
		return "app"
	default:
		return p
	}
}

func normalizeClientPlatform(p string) string {
	p = strings.TrimSpace(p)
	switch strings.ToLower(p) {
	case "android":
		return "Android"
	case "ios":
		return "iOS"
	case "harmony", "harmonyos":
		return "Harmony"
	default:
		// 允许直接传 "Android"/"iOS"/"Harmony"
		return p
	}
}

func mapUniPlatformToClientPlatform(uniPlatform string) string {
	switch normalizeUniPlatform(uniPlatform) {
	case "android":
		return "Android"
	case "ios":
		return "iOS"
	case "harmony":
		return "Harmony"
	default:
		return ""
	}
}

func containsString(arr []string, target string) bool {
	t := strings.TrimSpace(target)
	if t == "" {
		return false
	}
	for _, s := range arr {
		if strings.TrimSpace(s) == t {
			return true
		}
	}
	return false
}

// CheckVersion APP端检查更新
// @Summary APP端检查更新（替代 uniCloud）
// @Tags APP-Upgrade
// @Accept json
// @Produce json
// @Param body body model.AppUpgradeCheckReq true "检查更新"
// @Success 200 {object} model.Response
// @Router /api/v1/app/upgrade/check [post]
func (*AppUpgradeApi) CheckVersion(c *gin.Context) {
	var req model.AppUpgradeCheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Set("data", &model.AppUpgradeCheckResp{
			Code:    -1,
			Message: "invalid request body",
		})
		return
	}

	req.AppID = strings.TrimSpace(req.AppID)
	req.AppVersion = strings.TrimSpace(req.AppVersion)
	if req.AppID == "" || req.AppVersion == "" {
		c.Set("data", &model.AppUpgradeCheckResp{
			Code:    -1,
			Message: "appid/appVersion required",
		})
		return
	}

	tenantID := strings.TrimSpace(middleware.GetTenantIDFromHeader(c))

	uniPlatform := ""
	if req.UniPlatform != nil {
		uniPlatform = normalizeUniPlatform(*req.UniPlatform)
	}

	clientPlatform := ""
	if req.ClientPlatform != nil {
		clientPlatform = normalizeClientPlatform(*req.ClientPlatform)
	}
	if clientPlatform == "" {
		// 兜底：尝试用 uni_platform 推导（android/ios/harmony）
		clientPlatform = mapUniPlatformToClientPlatform(uniPlatform)
	}

	wgtVersion := "0"
	if req.WgtVersion != nil && strings.TrimSpace(*req.WgtVersion) != "" {
		wgtVersion = strings.TrimSpace(*req.WgtVersion)
	}

	// 1) 查 apps（用于 name/title 兜底）
	var appLite appRowLite
	if err := global.DB.WithContext(c.Request.Context()).
		Table("apps").
		Select("appid, name").
		Where("tenant_id = ? AND appid = ?", tenantID, req.AppID).
		Limit(1).
		Scan(&appLite).Error; err != nil {
		c.Set("data", &model.AppUpgradeCheckResp{
			Code:    -500,
			Message: "db error",
		})
		return
	}
	if strings.TrimSpace(appLite.AppID) == "" {
		c.Set("data", &model.AppUpgradeCheckResp{
			Code:    -404,
			Message: "app not found",
		})
		return
	}

	// 2) 查最新版本（稳定版，按 create_date desc 拉取一段，再按版本号挑最大）
	var rows []appVersionRow
	if err := global.DB.WithContext(c.Request.Context()).
		Table("app_versions").
		Select(`id, appid, name, title, contents, platform, type, version, min_uni_version, url,
			stable_publish, is_silently, is_mandatory, uni_platform, create_env, create_date, store_list`).
		Where("tenant_id = ? AND appid = ? AND stable_publish = TRUE", tenantID, req.AppID).
		Where("type IN (?)", []string{"wgt", "native_app"}).
		Order("create_date DESC").
		Limit(200).
		Scan(&rows).Error; err != nil {
		c.Set("data", &model.AppUpgradeCheckResp{
			Code:    -500,
			Message: "db error",
		})
		return
	}

	var bestWgt *appVersionRow
	var bestNative *appVersionRow
	for i := range rows {
		r := &rows[i]

		// 升级弹窗依赖 url/version/type；缺失则跳过，避免弹窗无法打开
		if strings.TrimSpace(r.Version) == "" || r.URL == nil || strings.TrimSpace(*r.URL) == "" {
			continue
		}

		// platform 过滤：优先使用 app_versions.platform（Android/iOS/Harmony）
		if clientPlatform != "" && len(r.Platform) != 0 {
			var plats []string
			_ = json.Unmarshal(r.Platform, &plats)
			if len(plats) > 0 && !containsString(plats, clientPlatform) {
				continue
			}
		}

		switch r.Type {
		case "wgt":
			if bestWgt == nil || compareVersion(r.Version, bestWgt.Version) == 1 {
				bestWgt = r
			}
		case "native_app":
			if bestNative == nil || compareVersion(r.Version, bestNative.Version) == 1 {
				bestNative = r
			}
		}
	}

	// 3) 选择升级包：优先 wgt；wgt 不满足则回退 native_app
	chosen := (*appVersionRow)(nil)
	if bestWgt != nil && compareVersion(bestWgt.Version, wgtVersion) == 1 {
		// min_uni_version 校验（客户端未传 uniVersion 时跳过）
		if bestWgt.MinUniVersion != nil && req.UniVersion != nil {
			if compareVersion(strings.TrimSpace(*req.UniVersion), strings.TrimSpace(*bestWgt.MinUniVersion)) >= 0 {
				chosen = bestWgt
			}
		} else {
			chosen = bestWgt
		}
	}
	if chosen == nil && bestNative != nil && compareVersion(bestNative.Version, req.AppVersion) == 1 {
		chosen = bestNative
	}

	if chosen == nil {
		c.Set("data", &model.AppUpgradeCheckResp{
			Code:    0,
			Message: "no update",
		})
		return
	}

	platform := make([]string, 0)
	if len(chosen.Platform) != 0 {
		_ = json.Unmarshal(chosen.Platform, &platform)
	}
	storeList := make([]model.StoreListItem, 0)
	if len(chosen.StoreList) != 0 {
		_ = json.Unmarshal(chosen.StoreList, &storeList)
	}

	title := "更新日志"
	if chosen.Title != nil && strings.TrimSpace(*chosen.Title) != "" {
		title = strings.TrimSpace(*chosen.Title)
	}
	contents := ""
	if chosen.Contents != nil {
		contents = strings.TrimSpace(*chosen.Contents)
	}
	url := ""
	if chosen.URL != nil {
		url = strings.TrimSpace(*chosen.URL)
	}

	c.Set("data", &model.AppUpgradeCheckResp{
		ID:            chosen.ID,
		AppID:         chosen.AppID,
		Name:          chosen.Name,
		Title:         title,
		Contents:      contents,
		URL:           url,
		Platform:      platform,
		Version:       chosen.Version,
		UniPlatform:   chosen.UniPlatform,
		StablePublish: chosen.StablePublish,
		IsMandatory:   chosen.IsMandatory,
		IsSilently:    chosen.IsSilently,
		CreateEnv:     chosen.CreateEnv,
		CreateDate:    chosen.CreateDate.UnixMilli(),
		Message:       "update available",
		Code:          1,
		Type:          chosen.Type,
		StoreList:     storeList,
		MinUniVersion: chosen.MinUniVersion,
	})
}
