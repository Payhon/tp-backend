package model

import "encoding/json"

// -----------------------------------------------------------------------------
// Apps
// -----------------------------------------------------------------------------

type AppListReq struct {
	PageReq
	Keyword *string `json:"keyword" form:"keyword" validate:"omitempty,max=200"` // AppID/名称模糊搜索
}

type AppCreateReq struct {
	AppID        string            `json:"appid" validate:"required,max=100"`
	AppType      *int16            `json:"app_type" validate:"omitempty,oneof=0 1"`
	Name         string            `json:"name" validate:"required,max=255"`
	Description  *string           `json:"description" validate:"omitempty,max=5000"`
	IconURL      *string           `json:"icon_url" validate:"omitempty,max=500"`
	Introduction *string           `json:"introduction" validate:"omitempty,max=5000"`
	Screenshot   []string          `json:"screenshot" validate:"omitempty"`
	AppAndroid   *json.RawMessage  `json:"app_android" validate:"omitempty,max=10000"`
	AppIOS       *json.RawMessage  `json:"app_ios" validate:"omitempty,max=10000"`
	AppHarmony   *json.RawMessage  `json:"app_harmony" validate:"omitempty,max=10000"`
	H5           *json.RawMessage  `json:"h5" validate:"omitempty,max=10000"`
	QuickApp     *json.RawMessage  `json:"quickapp" validate:"omitempty,max=10000"`
	StoreList    *json.RawMessage  `json:"store_list" validate:"omitempty,max=20000"`
	Remark       *string           `json:"remark" validate:"omitempty,max=5000"`
	Extra        *json.RawMessage  `json:"extra" validate:"omitempty,max=50000"` // 预留：小程序等扩展信息
	Managers     []string          `json:"managers" validate:"omitempty"`
	Members      []string          `json:"members" validate:"omitempty"`
	OwnerType    *int16            `json:"owner_type" validate:"omitempty,oneof=1 2"`
	OwnerID      *string           `json:"owner_id" validate:"omitempty,max=36"`
	CreatorUID   *string           `json:"creator_uid" validate:"omitempty,max=36"`
	MPWeixin     *json.RawMessage  `json:"mp_weixin" validate:"omitempty,max=10000"`
	MPAlipay     *json.RawMessage  `json:"mp_alipay" validate:"omitempty,max=10000"`
	MPBaidu      *json.RawMessage  `json:"mp_baidu" validate:"omitempty,max=10000"`
	MPToutiao    *json.RawMessage  `json:"mp_toutiao" validate:"omitempty,max=10000"`
	MPQQ         *json.RawMessage  `json:"mp_qq" validate:"omitempty,max=10000"`
	MPKuaishou   *json.RawMessage  `json:"mp_kuaishou" validate:"omitempty,max=10000"`
	MPLark       *json.RawMessage  `json:"mp_lark" validate:"omitempty,max=10000"`
	MPJD         *json.RawMessage  `json:"mp_jd" validate:"omitempty,max=10000"`
	MPDingtalk   *json.RawMessage  `json:"mp_dingtalk" validate:"omitempty,max=10000"`
	AppExtraInfo *json.RawMessage  `json:"app_extra" validate:"omitempty,max=50000"` // 预留：与 apps 表 JSON 字段兼容
}

type AppUpdateReq struct {
	AppType      *int16           `json:"app_type" validate:"omitempty,oneof=0 1"`
	Name         *string          `json:"name" validate:"omitempty,max=255"`
	Description  *string          `json:"description" validate:"omitempty,max=5000"`
	IconURL      *string          `json:"icon_url" validate:"omitempty,max=500"`
	Introduction *string          `json:"introduction" validate:"omitempty,max=5000"`
	Screenshot   []string         `json:"screenshot" validate:"omitempty"`
	AppAndroid   *json.RawMessage `json:"app_android" validate:"omitempty,max=10000"`
	AppIOS       *json.RawMessage `json:"app_ios" validate:"omitempty,max=10000"`
	AppHarmony   *json.RawMessage `json:"app_harmony" validate:"omitempty,max=10000"`
	H5           *json.RawMessage `json:"h5" validate:"omitempty,max=10000"`
	QuickApp     *json.RawMessage `json:"quickapp" validate:"omitempty,max=10000"`
	StoreList    *json.RawMessage `json:"store_list" validate:"omitempty,max=20000"`
	Remark       *string          `json:"remark" validate:"omitempty,max=5000"`
	Managers     []string         `json:"managers" validate:"omitempty"`
	Members      []string         `json:"members" validate:"omitempty"`
	OwnerType    *int16           `json:"owner_type" validate:"omitempty,oneof=1 2"`
	OwnerID      *string          `json:"owner_id" validate:"omitempty,max=36"`
}

type AppListItemResp struct {
	ID          string  `json:"id"`
	AppID       string  `json:"appid"`
	AppType     int16   `json:"app_type"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Remark      *string `json:"remark"`
	CreatedAt   string  `json:"created_at"`
}

type AppDetailResp struct {
	ID           string           `json:"id"`
	AppID        string           `json:"appid"`
	AppType      int16            `json:"app_type"`
	Name         string           `json:"name"`
	Description  *string          `json:"description"`
	IconURL      *string          `json:"icon_url"`
	Introduction *string          `json:"introduction"`
	Screenshot   []string         `json:"screenshot"`
	AppAndroid   *json.RawMessage `json:"app_android"`
	AppIOS       *json.RawMessage `json:"app_ios"`
	AppHarmony   *json.RawMessage `json:"app_harmony"`
	H5           *json.RawMessage `json:"h5"`
	QuickApp     *json.RawMessage `json:"quickapp"`
	StoreList    *json.RawMessage `json:"store_list"`
	Remark       *string          `json:"remark"`
	CreatedAt    string           `json:"created_at"`
	UpdatedAt    string           `json:"updated_at"`
}

type AppListResp struct {
	List     []AppListItemResp `json:"list"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// -----------------------------------------------------------------------------
// App Versions
// -----------------------------------------------------------------------------

type AppVersionListReq struct {
	PageReq
	AppID   *string `json:"app_id" form:"app_id" validate:"omitempty,max=36"`          // apps.id
	Keyword *string `json:"keyword" form:"keyword" validate:"omitempty,max=200"`      // title/version 模糊搜索
	Type    *string `json:"type" form:"type" validate:"omitempty,oneof=native_app wgt"` // 安装包类型
}

type AppVersionCreateReq struct {
	AppID         string   `json:"app_id" validate:"required,max=36"`
	Title         *string  `json:"title" validate:"omitempty,max=255"`
	Contents      string   `json:"contents" validate:"required,max=10000"`
	Platform      []string `json:"platform" validate:"required,min=1"` // Android/iOS/Harmony
	Type          string   `json:"type" validate:"required,oneof=native_app wgt"`
	Version       string   `json:"version" validate:"required,max=50"`
	MinUniVersion *string  `json:"min_uni_version" validate:"omitempty,max=50"`
	URL           string   `json:"url" validate:"required,max=500"`
	StablePublish *bool    `json:"stable_publish" validate:"omitempty"`
	IsSilently    *bool    `json:"is_silently" validate:"omitempty"`
	IsMandatory   *bool    `json:"is_mandatory" validate:"omitempty"`
	UniPlatform   string   `json:"uni_platform" validate:"required,max=50"`
	CreateEnv     *string  `json:"create_env" validate:"omitempty,max=50"` // uni-stat/upgrade-center
	StoreList     *json.RawMessage `json:"store_list" validate:"omitempty,max=20000"`
}

type AppVersionUpdateReq struct {
	Title         *string  `json:"title" validate:"omitempty,max=255"`
	Contents      *string  `json:"contents" validate:"omitempty,max=10000"`
	Platform      []string `json:"platform" validate:"omitempty"`
	Version       *string  `json:"version" validate:"omitempty,max=50"`
	MinUniVersion *string  `json:"min_uni_version" validate:"omitempty,max=50"`
	URL           *string  `json:"url" validate:"omitempty,max=500"`
	StablePublish *bool    `json:"stable_publish" validate:"omitempty"`
	IsSilently    *bool    `json:"is_silently" validate:"omitempty"`
	IsMandatory   *bool    `json:"is_mandatory" validate:"omitempty"`
	UniPlatform   *string  `json:"uni_platform" validate:"omitempty,max=50"`
	StoreList     *json.RawMessage `json:"store_list" validate:"omitempty,max=20000"`
}

type AppVersionListItemResp struct {
	ID           string   `json:"id"`
	AppID        string   `json:"appid"`
	AppName      string   `json:"name"`
	Title        *string  `json:"title"`
	Type         string   `json:"type"`
	Platform     []string `json:"platform"`
	Version      string   `json:"version"`
	StablePublish bool    `json:"stable_publish"`
	CreateDate   string   `json:"create_date"`
}

type AppVersionDetailResp struct {
	ID            string   `json:"id"`
	AppID         string   `json:"appid"`
	AppName       string   `json:"name"`
	Title         *string  `json:"title"`
	Contents      *string  `json:"contents"`
	Platform      []string `json:"platform"`
	Type          string   `json:"type"`
	Version       string   `json:"version"`
	MinUniVersion *string  `json:"min_uni_version"`
	URL           *string  `json:"url"`
	StablePublish bool     `json:"stable_publish"`
	IsSilently    bool     `json:"is_silently"`
	IsMandatory   bool     `json:"is_mandatory"`
	UniPlatform   string   `json:"uni_platform"`
	CreateEnv     string   `json:"create_env"`
	CreateDate    string   `json:"create_date"`
}

type AppVersionListResp struct {
	List     []AppVersionListItemResp `json:"list"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

type BatchDeleteReq struct {
	IDs []string `json:"ids" validate:"required,min=1,dive,max=36"`
}
