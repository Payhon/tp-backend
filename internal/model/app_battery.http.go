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
	ItemUUID           *string  `json:"item_uuid"`
	BatchNumber        *string  `json:"batch_number"`
	ProductSpec        *string  `json:"product_spec"`
	OrderNumber        *string  `json:"order_number"`
	BleMac             *string  `json:"ble_mac"`
	IdentityBleMac     *string  `json:"identity_ble_mac"`
	CommChipID         *string  `json:"comm_chip_id"`
	ProductionDate     *string  `json:"production_date"`
	WarrantyExpireDate *string  `json:"warranty_expire_date"`
	Soc                *float64 `json:"soc"`
	Soh                *float64 `json:"soh"`
	UpdatedAt          *string  `json:"updated_at"`
	IsOnline           int16    `json:"is_online"`
	FwVersion          *string  `json:"fw_version"`
	Remark             *string  `json:"remark"`
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

type AppBatteryCurrentTelemetryValue struct {
	Value interface{} `json:"value"`
	Ts    int64       `json:"ts"`
}

type AppBatteryCurrentTelemetryResp struct {
	DeviceID              string                                     `json:"device_id"`
	IsOnline              int16                                      `json:"is_online"`
	LastReportTs          int64                                      `json:"last_report_ts"`
	SnapshotTs            int64                                      `json:"snapshot_ts,omitempty"`
	Current               map[string]AppBatteryCurrentTelemetryValue `json:"current"`
	Snapshot              map[string]interface{}                     `json:"snapshot,omitempty"`
	InteractiveSnapshotTs int64                                      `json:"interactive_snapshot_ts,omitempty"`
	InteractiveSnapshot   map[string]interface{}                     `json:"interactive_snapshot,omitempty"`
}

// AppBatteryInteractiveSnapshotReq APP 端 MQTT 主动读取快照上报请求。
type AppBatteryInteractiveSnapshotReq struct {
	DeviceID  string                 `json:"device_id" binding:"required"`
	SessionID string                 `json:"session_id" binding:"required"`
	Platform  string                 `json:"platform,omitempty"`
	Snapshot  map[string]interface{} `json:"snapshot" binding:"required"`
}

// AppBatteryInteractiveSnapshotResp APP 端 MQTT 主动读取快照上报响应。
type AppBatteryInteractiveSnapshotResp struct {
	DeviceID string `json:"device_id"`
	Ts       int64  `json:"ts"`
	Accepted bool   `json:"accepted"`
}

// AppBatteryOtaCheckResp APP端OTA检查结果
type AppBatteryOtaCheckResp struct {
	DeviceID       string  `json:"device_id"`
	NeedUpgrade    bool    `json:"need_upgrade"`
	CurrentVersion *string `json:"current_version,omitempty"`
	Version        *string `json:"version,omitempty"`
	TargetVersion  *string `json:"target_version,omitempty"`
	FirmwareURL    *string `json:"firmware_url,omitempty"`
	PackageID      *string `json:"package_id,omitempty"`
	PackageType    *int16  `json:"package_type,omitempty"`
	SignatureType  *string `json:"signature_type,omitempty"`
	Signature      *string `json:"signature,omitempty"`
	Module         *string `json:"module,omitempty"`
	AdditionalInfo *string `json:"additional_info,omitempty"`
	Remark         *string `json:"remark,omitempty"`
}

// AppBatteryOtaCheckReq APP端OTA检查请求
type AppBatteryOtaCheckReq struct {
	DeviceID       string  `json:"device_id"`
	Model          *string `json:"model,omitempty"`
	Version        *string `json:"version,omitempty"`
	BatteryModelID *string `json:"battery_model_id,omitempty"`
	BatchNumber    *string `json:"batch_number,omitempty"`
	ItemUUID       *string `json:"item_uuid,omitempty"`
}

type AppBatteryMeterOtaPackageResp struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	PackageURL  *string `json:"package_url,omitempty"`
}

// AppBatteryReportReq APP端BMS上报请求
type AppBatteryReportReq struct {
	DeviceID string                 `json:"device_id" binding:"required"`
	Ts       int64                  `json:"ts" binding:"required"` // 毫秒时间戳
	ConnType string                 `json:"conn_type" binding:"required"`
	Platform string                 `json:"platform" binding:"required"`
	Core     map[string]interface{} `json:"core" binding:"required"`
	Snapshot map[string]interface{} `json:"snapshot,omitempty"`
}

// AppBatteryReportResp APP端BMS上报响应
type AppBatteryReportResp struct {
	DeviceID      string `json:"device_id"`
	Ts            int64  `json:"ts"`
	Accepted      bool   `json:"accepted"`
	IgnoredReason string `json:"ignored_reason,omitempty"`
}

// AppBatteryConnectionStatusReq APP 端连接状态同步请求
type AppBatteryConnectionStatusReq struct {
	DeviceID     string `json:"device_id" binding:"required"`
	ConnType     string `json:"conn_type" binding:"required"`
	Platform     string `json:"platform,omitempty"`
	BleConnected bool   `json:"ble_connected"`
	Ts           int64  `json:"ts,omitempty"`
}

// AppBatteryConnectionStatusResp APP 端连接状态同步响应
type AppBatteryConnectionStatusResp struct {
	DeviceID      string `json:"device_id"`
	Ts            int64  `json:"ts"`
	BleConnected  bool   `json:"ble_connected"`
	Accepted      bool   `json:"accepted"`
	StatusChanged bool   `json:"status_changed"`
	IgnoredReason string `json:"ignored_reason,omitempty"`
}

// AppBatteryRelayStatusResp BLE Relay 在线状态（WEB 查询）
type AppBatteryRelayStatusResp struct {
	DeviceID      string  `json:"device_id"`
	OwnerOnline   bool    `json:"owner_online"`
	SessionID     *string `json:"session_id,omitempty"`
	Platform      *string `json:"platform,omitempty"`
	ConnType      *string `json:"conn_type,omitempty"`
	LastSeenTs    *int64  `json:"last_seen_ts,omitempty"`
	ExpiresAtTs   *int64  `json:"expires_at_ts,omitempty"`
	OwnerUserID   *string `json:"owner_user_id,omitempty"`
	OwnerTenantID *string `json:"owner_tenant_id,omitempty"`
}

// AppBatteryRelayCommandReq WEB->APP(BLE) Relay 指令请求
type AppBatteryRelayCommandReq struct {
	DeviceID       string      `json:"device_id" validate:"required,max=36"`
	CommandType    string      `json:"command_type" validate:"required,oneof=read_param write_param write_registers"`
	ParamKey       *string     `json:"param_key,omitempty"`
	Value          interface{} `json:"value,omitempty"`
	StartAddress   *int        `json:"start_address,omitempty"`
	RegisterValues []int       `json:"register_values,omitempty"`
	WaitMs         *int64      `json:"wait_ms,omitempty"`
}

// AppBatteryRelayCommandResp Relay 指令执行结果
type AppBatteryRelayCommandResp struct {
	CommandID    string      `json:"command_id"`
	DeviceID     string      `json:"device_id"`
	CommandType  string      `json:"command_type"`
	Status       string      `json:"status"`
	ErrorMessage *string     `json:"error_message,omitempty"`
	Result       interface{} `json:"result,omitempty"`
	CreatedAtTs  int64       `json:"created_at_ts"`
	UpdatedAtTs  int64       `json:"updated_at_ts"`
	FinishedAtTs *int64      `json:"finished_at_ts,omitempty"`
}
