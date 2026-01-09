package apps

type apps struct {
	User                       // 用户模块
	Role                       // 角色管理
	Casbin                     // 权限
	Dict                       // 字典模块
	OTA                        // OTA
	UpLoad                     // 文件上传
	AppManage                  // APP管理：应用管理/升级中心
	AppContent                 // APP内容管理：单页/FAQ/用户反馈
	ProtocolPlugin             // 协议插件
	Device                     // 设备
	UiElements                 // ui元素控制
	Board                      // 首页
	EventData                  // 属性数据
	TelemetryData              // 遥测数据
	AttributeData              // 属性数据
	CommandData                // 命令数据
	OperationLog               // 操作日志
	Logo                       // 站标
	DataPolicy                 // 数据清理
	DeviceConfig               // 设备配置
	DataScript                 // 数据处理脚本
	NotificationGroup          // 通知组
	NotificationHistoryGroup   // 通知历史组
	NotificationServicesConfig // 通知服务配置
	Alarm
	SceneAutomations
	Scene
	SysFunction
	ServicePlugin // 插件管理
	ExpectedData  // 预期数据
	OpenAPIKey    // openAPI
	MessagePush
	SystemMonitor      // 系统监控
	DeviceAuth         // 设备动态认证
	Dealer             // BMS: 经销商管理
	BmsDashboard       // BMS: Dashboard
	Battery            // BMS: 电池管理
	BatteryModel       // BMS: 电池型号管理
	DeviceTransfer     // BMS: 设备转移
	DeviceBinding      // BMS: APP设备绑定
	AppBattery         // BMS: APP电池设备（详情/透传）
	Warranty           // BMS: 维保管理
	EndUser            // BMS: 终端用户
	ActivationLog      // BMS: 激活日志
	BatteryMaintenance // BMS: 电池维保记录
	Org                // BMS: 组织管理
	OrgTypePermission  // WEB: 机构类型权限配置（菜单权限/设备参数权限）
}

var Model = new(apps)
