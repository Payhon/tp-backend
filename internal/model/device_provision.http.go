package model

// DeviceProvisionConfigResp 移动端开通配置
type DeviceProvisionConfigResp struct {
	DTUDomainPort string `json:"dtu_domain_port"`
}

// DeviceProvisionInfoReq 移动端扫码 UUID 查询
type DeviceProvisionInfoReq struct {
	ItemUUID string `form:"item_uuid" validate:"required,max=64"`
}

// DeviceProvisionInfoResp 设备开通信息（按 item_uuid 查询）
type DeviceProvisionInfoResp struct {
	DeviceID     string  `json:"device_id"`
	DeviceNumber string  `json:"device_number"`
	DeviceName   *string `json:"device_name,omitempty"`
	BleMac       *string `json:"ble_mac,omitempty"`
	CommChipID   *string `json:"comm_chip_id,omitempty"`
	IsBound      bool    `json:"is_bound"`
}

// DeviceProvisionBindReq 设备开通绑定请求（按 item_uuid 绑定到当前账号）
type DeviceProvisionBindReq struct {
	ItemUUID string  `json:"item_uuid" validate:"required,max=64"`
	BleMac   *string `json:"ble_mac" validate:"omitempty,max=32"`
}

// DeviceProvisionBindResp 设备开通绑定响应
type DeviceProvisionBindResp struct {
	DeviceID     string `json:"device_id"`
	DeviceNumber string `json:"device_number"`
}
