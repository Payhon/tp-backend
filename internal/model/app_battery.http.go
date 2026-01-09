package model

// AppBatteryDetailResp APP端电池设备详情（用于设备详情页）
type AppBatteryDetailResp struct {
	DeviceID     string  `json:"device_id"`
	DeviceNumber string  `json:"device_number"`
	DeviceName   *string `json:"device_name"`

	BatteryModelID   *string `json:"battery_model_id"`
	BatteryModelName *string `json:"battery_model_name"`

	// 设备电池扩展信息（device_batteries）
	ItemUUID    *string  `json:"item_uuid"`
	BleMac      *string  `json:"ble_mac"`
	CommChipID  *string  `json:"comm_chip_id"`
	Soc         *float64 `json:"soc"`
	Soh         *float64 `json:"soh"`
	UpdatedAt   *string  `json:"updated_at"`
	IsOnline    int16    `json:"is_online"`
	FwVersion   *string  `json:"fw_version"`
	Remark      *string  `json:"remark"`
}

