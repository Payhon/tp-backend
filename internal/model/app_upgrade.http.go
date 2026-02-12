package model

// AppUpgradeCheckReq APP 端检查更新请求
//
// 为了兼容 uni-upgrade-center 的入参命名，沿用 appVersion/wgtVersion 字段名。
type AppUpgradeCheckReq struct {
	// 兼容字段（uni-upgrade-center）：action=checkVersion
	Action *string `json:"action" validate:"omitempty,max=50"`

	AppID      string  `json:"appid" validate:"required,max=100"`
	AppVersion string  `json:"appVersion" validate:"required,max=50"`
	WgtVersion *string `json:"wgtVersion" validate:"omitempty,max=50"`

	// 平台：android/ios/harmony（建议由客户端传入）
	UniPlatform *string `json:"uni_platform" validate:"omitempty,max=20"`

	// 设备平台（用于匹配 app_versions.platform）：Android/iOS/Harmony
	ClientPlatform *string `json:"client_platform" validate:"omitempty,max=20"`

	// uni-app 运行时版本（用于判断 min_uni_version，客户端可选传入）
	UniVersion *string `json:"uniVersion" validate:"omitempty,max=50"`

	// 是否 uni-app x（预留）
	IsUniappX *bool `json:"is_uniapp_x" validate:"omitempty"`
}

// StoreListItem 应用商店信息（兼容 uni-upgrade-center）
type StoreListItem struct {
	Enable   bool   `json:"enable"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Scheme   string `json:"scheme"`
	Priority int    `json:"priority"`
}

// AppUpgradeCheckResp APP 端检查更新响应（兼容 uni-upgrade-center）
type AppUpgradeCheckResp struct {
	ID string `json:"_id"`

	AppID string `json:"appid"`
	Name  string `json:"name"`
	Title string `json:"title"`

	Contents string   `json:"contents"`
	URL      string   `json:"url"`
	Platform []string `json:"platform"`

	Version     string `json:"version"`
	UniPlatform string `json:"uni_platform"`

	StablePublish bool `json:"stable_publish"`
	IsMandatory   bool `json:"is_mandatory"`
	IsSilently    bool `json:"is_silently"`

	CreateEnv  string `json:"create_env"`
	CreateDate int64  `json:"create_date"` // unix ms

	Message string `json:"message"`
	Code    int    `json:"code"` // >0 有更新；0 无更新；<0 异常

	Type          string          `json:"type"` // native_app / wgt
	StoreList     []StoreListItem `json:"store_list"`
	MinUniVersion *string         `json:"min_uni_version"`
}
