package model

// AppBatteryDetailResp APP端电池设备详情（用于设备详情页）
type AppBatteryDetailResp struct {
	DeviceID     string  `json:"device_id"`
	DeviceNumber string  `json:"device_number"`
	DeviceName   *string `json:"device_name"`

	BmsCommType *int `json:"bms_comm_type,omitempty"` // 1:蓝牙、2:4G、3:蓝牙+4G

	BatteryModelID   *string `json:"battery_model_id"`
	BatteryModelName *string `json:"battery_model_name"`

	// 设备电池扩展信息（device_batteries）
	ItemUUID   *string  `json:"item_uuid"`
	BleMac     *string  `json:"ble_mac"`
	CommChipID *string  `json:"comm_chip_id"`
	Soc        *float64 `json:"soc"`
	Soh        *float64 `json:"soh"`
	UpdatedAt  *string  `json:"updated_at"`
	IsOnline   int16    `json:"is_online"`
	FwVersion  *string  `json:"fw_version"`
	Remark     *string  `json:"remark"`
}

// AppBatteryMqttCredentialResp APP端直连 MQTT(Broker WebSocket) 所需信息
type AppBatteryMqttCredentialResp struct {
	DeviceID   string `json:"device_id"`
	WsURL      string `json:"ws_url"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	WriteTopic string `json:"write_topic"`
	ReadTopic  string `json:"read_topic"`
}
