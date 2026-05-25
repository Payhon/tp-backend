package model

const (
	OTADeviceKindBMS      int16 = 1
	OTADeviceKindMeter    int16 = 2
	OTADeviceKind4GModule int16 = 3
)

type CreateOTAUpgradePackageReq struct {
	Name           string  `json:"name" validate:"required,max=200"`                     // 升级包名称
	Version        string  `json:"version"  validate:"omitempty,max=36"`                 // 版本号
	TargetVersion  *string `json:"target_version" validate:"omitempty,max=36"`           // 目标版本号
	DeviceConfigID string  `json:"device_config_id" validate:"omitempty,max=36"`         // 设备配置ID
	BatteryModelID *string `json:"battery_model_id" validate:"omitempty,max=36"`         // BMS型号ID
	BatchNumber    *string `json:"batch_number" validate:"omitempty,max=100"`            // 批号约束
	ItemUUID       *string `json:"item_uuid" validate:"omitempty,max=64"`                // 序列号约束
	Module         *string `json:"module" validate:"omitempty,max=36"`                   // 模块名称
	PackageType    *int16  `json:"package_type" validate:"omitempty,oneof=1 2"`          // 升级包类型升级包类型1-差分 2-整包
	SignatureType  *string `json:"signature_type" validate:"omitempty,oneof=MD5 SHA256"` // 签名算法 MD5 SHA256
	AdditionalInfo *string `json:"additional_info" validate:"omitempty" example:"{}"`    // 附加信息,json格式
	Description    *string `json:"description" validate:"omitempty,max=500"`             // 描述
	PackageUrl     *string `json:"package_url" validate:"omitempty,max=500"`             // 升级包地址
	Remark         *string `json:"remark" validate:"omitempty,max=255"`
	DeviceKind     *int16  `json:"device_kind" validate:"omitempty,oneof=1 2 3"` // 设备类型 1-BMS 2-仪表 3-4G模块
	IsLatest       *bool   `json:"is_latest" validate:"omitempty"`               // 是否最新固件
}

type UpdateOTAUpgradePackageReq struct {
	Id             string  `json:"id" validate:"required,max=36"`                        // 升级包ID
	Name           string  `json:"name" validate:"omitempty,max=200"`                    // 升级包名称
	Version        string  `json:"version"  validate:"omitempty,max=36"`                 // 版本号
	TargetVersion  *string `json:"target_version" validate:"omitempty,max=36"`           // 目标版本号
	DeviceConfigID string  `json:"device_config_id" validate:"omitempty,max=36"`         // 设备配置ID
	BatteryModelID *string `json:"battery_model_id" validate:"omitempty,max=36"`         // BMS型号ID
	BatchNumber    *string `json:"batch_number" validate:"omitempty,max=100"`            // 批号约束
	ItemUUID       *string `json:"item_uuid" validate:"omitempty,max=64"`                // 序列号约束
	Module         *string `json:"module" validate:"omitempty,max=36"`                   // 模块名称
	PackageType    *int16  `json:"package_type" validate:"omitempty,oneof=1 2"`          // 升级包类型
	SignatureType  *string `json:"signature_type" validate:"omitempty,oneof=MD5 SHA256"` // 签名算法 MD5 SHA256
	AdditionalInfo *string `json:"additional_info" validate:"omitempty"`                 // 附加信息,json格式
	Description    *string `json:"description" validate:"omitempty,max=500"`             // 描述
	PackageUrl     *string `json:"package_url" validate:"omitempty,max=500"`             // 升级包地址
	Remark         *string `json:"remark" validate:"omitempty,max=255"`                  // 备注
	DeviceKind     *int16  `json:"device_kind" validate:"omitempty,oneof=1 2 3"`         // 设备类型 1-BMS 2-仪表 3-4G模块
	IsLatest       *bool   `json:"is_latest" validate:"omitempty"`                       // 是否最新固件
}

type GetOTAUpgradePackageLisyByPageReq struct {
	PageReq
	DeviceConfigID string `json:"device_configs_id" form:"device_config_id" validate:"omitempty,max=36" example:"uuid"` // 设备配置ID
	Name           string `json:"name" form:"name" validate:"omitempty,max=200"`                                        //  升级包名称
	DeviceKind     int16  `json:"device_kind" form:"device_kind" validate:"omitempty,oneof=1 2 3"`                      // 设备类型 1-BMS 2-仪表 3-4G模块
}

type GetOTAUpgradeTaskListByPageRsp struct {
	OtaUpgradePackage
	DeviceConfigName string `json:"device_config_name" validate:"omitempty,max=200"` // 设备配置名称
	BatteryModelName string `json:"battery_model_name" validate:"omitempty,max=100"` // BMS型号名称
}

type GetOTA4GModuleUpgradeCheckReq struct {
	Version  string `json:"version" form:"version" validate:"required,max=36"`      // 4G模块当前版本号
	Imei     string `json:"imei" form:"imei" validate:"required,max=64"`            // 4G模块 IMEI 或 4G通讯卡ID
	TenantID string `json:"tenant_id" form:"tenant_id" validate:"omitempty,max=36"` // 租户ID
}

type OTA4GModuleUpgradeCheckResp struct {
	NeedUpgrade    bool    `json:"need_upgrade"`
	CurrentVersion string  `json:"current_version"`
	Version        *string `json:"version,omitempty"`
	FirmwareURL    *string `json:"firmware_url,omitempty"`
	PackageID      *string `json:"package_id,omitempty"`
	Name           *string `json:"name,omitempty"`
	Description    *string `json:"description,omitempty"`
	IsLatest       bool    `json:"is_latest"`
	Imei           string  `json:"imei"`
}
