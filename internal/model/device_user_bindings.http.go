package model

const (
	AppDeviceViewModeSelfBound   = "self_bound"
	AppDeviceViewModeOrgAdded    = "org_added"
	AppDeviceViewModeEndUserBind = "end_user_bound"
)

// DeviceBindReq APP端设备绑定请求
type DeviceBindReq struct {
	DeviceNumber string  `json:"device_number" binding:"required"`
	DeviceSecret *string `json:"device_secret"` // 设备密钥（可选，用于验证）
}

// DeviceUnbindReq APP端设备解绑请求
type DeviceUnbindReq struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// AppDeviceRemoveReq APP端机构用户移除“我添加的设备”
type AppDeviceRemoveReq struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// DeviceUserBindingListReq 设备绑定记录查询请求
type DeviceUserBindingListReq struct {
	Page         int     `form:"page" binding:"required,min=1"`
	PageSize     int     `form:"page_size" binding:"required,min=1,max=100"`
	UserID       *string `form:"user_id"`
	ViewMode     *string `form:"view_mode" binding:"omitempty,oneof=self_bound org_added end_user_bound"`
	DeviceName   *string `form:"device_name"`
	DeviceNumber *string `form:"device_number"`
	BleMac       *string `form:"ble_mac"`
	AddedStartAt *string `form:"added_start_at"`
	AddedEndAt   *string `form:"added_end_at"`
}

// DeviceUserBindingResp 设备绑定记录响应
type DeviceUserBindingResp struct {
	ID               string   `json:"id"`
	UserID           string   `json:"user_id"`
	UserName         *string  `json:"user_name"`
	UserPhone        string   `json:"user_phone"`
	DeviceID         string   `json:"device_id"`
	DeviceNumber     string   `json:"device_number"`
	DeviceName       string   `json:"device_name"`
	BleMac           *string  `json:"ble_mac,omitempty"`
	BmsCommType      *int     `json:"bms_comm_type,omitempty"`
	IsOnline         int16    `json:"is_online"`
	Soc              *float64 `json:"soc,omitempty"`
	IsOwner          bool     `json:"is_owner"`
	BindingTime      string   `json:"binding_time"`
	AddedAt          string   `json:"added_at,omitempty"`
	RelationTime     string   `json:"relation_time,omitempty"`
	RelationType     string   `json:"relation_type,omitempty"`
	ActivationStatus *string  `json:"activation_status,omitempty"`
}

// DeviceUserBindingListResp 设备绑定记录列表响应
type DeviceUserBindingListResp struct {
	List     []DeviceUserBindingResp `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

// AppOrgDeviceListReq APP端组织范围设备列表请求
type AppOrgDeviceListReq struct {
	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=1,max=100"`

	// 设备编号模糊查询
	DeviceNumber *string `form:"device_number"`

	// 按组织筛选
	OwnerOrgID *string `form:"owner_org_id"`

	// 按组织类型筛选（PACK_FACTORY/DEALER/STORE）
	OwnerOrgType *string `form:"owner_org_type"`
}

// AppOrgDeviceListItem APP端组织范围设备列表项
type AppOrgDeviceListItem struct {
	DeviceID         string  `json:"device_id"`
	DeviceNumber     string  `json:"device_number"`
	DeviceName       *string `json:"device_name"`
	IsOnline         int16   `json:"is_online"`
	ActivationStatus *string `json:"activation_status"`

	OwnerOrgID   *string `json:"owner_org_id"`
	OwnerOrgName *string `json:"owner_org_name"`
	OwnerOrgType *string `json:"owner_org_type"`
}

// AppOrgDeviceListResp APP端组织范围设备列表响应
type AppOrgDeviceListResp struct {
	List     []AppOrgDeviceListItem `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}
