package service

import "project/internal/model"

func leafNode(label, value, registerAddress string, paramKeys ...string) model.DeviceParamTreeNode {
	return model.DeviceParamTreeNode{
		Label:           label,
		Value:           value,
		RegisterAddress: registerAddress,
		ParamKeys:       paramKeys,
	}
}

func buildDeviceParamPermissionTree() []model.DeviceParamTreeNode {
	return []model.DeviceParamTreeNode{
		{
			Label: "单体设置",
			Value: "group:single",
			Children: []model.DeviceParamTreeNode{
				leafNode("单体过压告警电压", "400", "0x0400", "CELL_OV_ALARM_V"),
				leafNode("单体过充保护电压", "401", "0x0401", "CELL_OC_PROTECT_V"),
				leafNode("单体过充告警延时 / 单体过充保护延时", "402", "0x0402", "CELL_OC_ALARM_DELAY_S", "CELL_OC_PROTECT_DELAY_S"),
				leafNode("单体过压保护解除电压", "404", "0x0404", "CELL_OV_PROTECT_RELEASE_V"),
				leafNode("单体过充告警解除电压差", "405", "0x0405", "CELL_OC_ALARM_RELEASE_DELTA_V"),
				leafNode("单体过压告警解除延时 / 单体过压保护解除延时", "408", "0x0408", "CELL_OV_ALARM_RELEASE_DELAY_S", "CELL_OV_PROTECT_RELEASE_DELAY_S"),
				leafNode("常温单体过放告警电压", "409", "0x0409", "NORMAL_CELL_UV_ALARM_V"),
				leafNode("常温单体过放保护电压", "40a", "0x040A", "NORMAL_CELL_UV_PROTECT_V"),
				leafNode("单体过放告警延时", "40d", "0x040D", "CELL_UV_ALARM_DELAY_S"),
				leafNode("单体过放保护延时", "40e", "0x040E", "CELL_UV_PROTECT_DELAY_S"),
				leafNode("单体过放保护解除电压", "40f:protect", "0x040F", "CELL_UV_PROTECT_RELEASE_V"),
				leafNode("单体过放告警解除延时", "410:alarm_release", "0x0410(L)", "CELL_UV_ALARM_RELEASE_DELAY_S"),
				leafNode("单体过放保护解除延时", "410:protect_release", "0x0410(H)", "CELL_UV_PROTECT_RELEASE_DELAY_S"),
			},
		},
		{
			Label: "总压设置",
			Value: "group:voltage",
			Children: []model.DeviceParamTreeNode{
				leafNode("总压过压告警电压", "411", "0x0411", "PACK_OV_ALARM_V"),
				leafNode("总压过压保护电压", "412", "0x0412", "PACK_OV_PROTECT_V"),
				leafNode("总压过压保护延时 / 总压过压告警延时", "413", "0x0413", "PACK_OV_PROTECT_DELAY_S", "PACK_OV_ALARM_DELAY_S"),
				leafNode("总压过压告警解除电压", "414", "0x0414", "PACK_OV_ALARM_RELEASE_V"),
				leafNode("总压过压保护解除电压", "415", "0x0415", "PACK_OV_PROTECT_RELEASE_V"),
				leafNode("总压过放告警延时 / 总压过放保护延时", "41b", "0x041B", "PACK_UV_ALARM_DELAY_S", "PACK_UV_PROTECT_DELAY_S"),
				leafNode("总压过压解除延时 / 总压过压告警解除延时", "416", "0x0416", "PACK_OV_PROTECT_RELEASE_DELAY_S", "PACK_OV_ALARM_RELEASE_DELAY_S"),
				leafNode("常温总压欠压告警电压", "417", "0x0417", "NORMAL_PACK_UV_ALARM_V"),
				leafNode("常温总压欠压保护电压", "418", "0x0418", "NORMAL_PACK_UV_PROTECT_V"),
				leafNode("低温总压欠压告警电压", "419", "0x0419", "LOW_TEMP_PACK_UV_ALARM_V"),
				leafNode("低温总压欠压保护电压", "41a", "0x041A", "LOW_TEMP_PACK_UV_PROTECT_V"),
				leafNode("总压欠压告警解除电压", "41c", "0x041C", "PACK_UV_ALARM_RELEASE_V"),
				leafNode("总压欠压保护解除电压", "41d", "0x041D", "PACK_UV_PROTECT_RELEASE_V"),
				leafNode("总压过放告警解除延时 / 总压过放保护解除延时", "41e", "0x041E", "PACK_UV_ALARM_RELEASE_DELAY_S", "PACK_UV_PROTECT_RELEASE_DELAY_S"),
			},
		},
		{
			Label: "电流设置",
			Value: "group:current",
			Children: []model.DeviceParamTreeNode{
				leafNode("充电过流保护小电流", "420", "0x0420", "CHARGE_OC_PROTECT_SMALL_A"),
				leafNode("充电过流保护大电流", "421", "0x0421", "CHARGE_OC_PROTECT_LARGE_A"),
				leafNode("充电过流告警延时", "422", "0x0422", "CHARGE_OC_ALARM_DELAY_S"),
				leafNode("充电过流保护大电流延时", "422:large_delay", "0x0422(H)", "CHARGE_OC_PROTECT_LARGE_DELAY_S"),
				leafNode("充电过流保护小电流延时", "423:small_delay", "0x0423(L)", "CHARGE_OC_PROTECT_SMALL_DELAY_S"),
				leafNode("放电过流告警电流", "427", "0x0427", "DISCHARGE_OC_ALARM_A"),
				leafNode("放电过流小电流保护电流", "428", "0x0428", "DISCHARGE_OC_PROTECT_SMALL_A"),
				leafNode("放电过流大电流保护电流", "429", "0x0429", "DISCHARGE_OC_PROTECT_LARGE_A"),
				leafNode("放电过流告警延时", "42a:alarm_delay", "0x042A(L)", "DISCHARGE_OC_ALARM_DELAY_S"),
				leafNode("放电过流大电流保护延时", "42a:large_delay", "0x042A(H)", "DISCHARGE_OC_PROTECT_LARGE_DELAY_S"),
				leafNode("放电过流小电流保护延时", "42b:small_delay", "0x042B(L)", "DISCHARGE_OC_PROTECT_SMALL_DELAY_S"),
			},
		},
		{
			Label: "温度设置",
			Value: "group:temperature",
			Children: []model.DeviceParamTreeNode{
				leafNode("MOS高温保护温度", "438", "0x0438(H)", "MOS_OT_PROTECT_C"),
				leafNode("MOS高温保护解除温度", "439", "0x0439(H)", "MOS_OT_PROTECT_RELEASE_C"),
				leafNode("MOS高温保护延时", "43a:protect_delay", "0x043A(H)", "MOS_OT_PROTECT_DELAY_S"),
				leafNode("MOS高温保护解除延时", "43b:protect_release_delay", "0x043B(H)", "MOS_OT_PROTECT_RELEASE_DELAY_S"),
				leafNode("充电低温保护温度", "442", "0x0442(H)", "CHARGE_UT_PROTECT_C"),
				leafNode("充电低温保护解除温度", "443", "0x0443(H)", "CHARGE_UT_PROTECT_RELEASE_C"),
				leafNode("充电高温保护温度", "444", "0x0444(H)", "CHARGE_OT_PROTECT_C"),
				leafNode("充电高温保护解除温度", "445", "0x0445(H)", "CHARGE_OT_PROTECT_RELEASE_C"),
				leafNode("充电高温保护延时", "446:protect_delay", "0x0446(H)", "CHARGE_OT_PROTECT_DELAY_S"),
				leafNode("充电高温保护解除延时", "447:protect_release_delay", "0x0447(H)", "CHARGE_OT_PROTECT_RELEASE_DELAY_S"),
				leafNode("放电低温保护温度", "448", "0x0448(H)", "DISCHARGE_UT_PROTECT_C"),
				leafNode("放电低温保护解除温度", "449", "0x0449(H)", "DISCHARGE_UT_PROTECT_RELEASE_C"),
				leafNode("放电高温保护温度", "44a", "0x044A(H)", "DISCHARGE_OT_PROTECT_C"),
				leafNode("放电高温保护解除温度", "44c", "0x044C", "DISCHARGE_OT_PROTECT_RELEASE_C"),
				leafNode("放电高温保护延时", "44e", "0x044E", "DISCHARGE_OT_PROTECT_DELAY_S"),
				leafNode("放电高温保护解除延时", "450", "0x0450", "DISCHARGE_OT_PROTECT_RELEASE_DELAY_S"),
			},
		},
		{
			Label: "高级参数",
			Value: "group:advanced",
			Children: []model.DeviceParamTreeNode{
				{
					Label: "高级配置",
					Value: "group:advanced:other",
					Children: []model.DeviceParamTreeNode{
						leafNode("均衡开启电压", "458", "0x0458", "BALANCE_START_V"),
						leafNode("开启压差", "459", "0x0459", "BALANCE_START_DELTA_V"),
						leafNode("停止压差 / 均衡功能高温禁止温度", "45a", "0x045A", "BALANCE_STOP_DELTA_V", "BALANCE_DISABLE_HIGH_TEMP_C"),
						leafNode("均衡功能低温禁止温度 / 压差报警压差", "45b", "0x045B", "BALANCE_DISABLE_LOW_TEMP_C", "DELTA_V_ALARM_THRESHOLD_V"),
						leafNode("压差报警恢复压差 / 压差保护压差", "45c", "0x045C", "DELTA_V_ALARM_RELEASE_V", "DELTA_V_PROTECT_THRESHOLD_V"),
						leafNode("压差保护恢复压差 / 压差保护延时", "45d", "0x045D", "DELTA_V_PROTECT_RELEASE_V", "DELTA_V_PROTECT_DELAY_S"),
						leafNode("压差解除延时 / 温差报警阈值", "45e", "0x045E", "DELTA_V_RELEASE_DELAY_S", "TEMP_DIFF_ALARM_THRESHOLD_C"),
						leafNode("温差报警恢复阈值 / 温差保护阈值", "45f", "0x045F", "TEMP_DIFF_ALARM_RELEASE_C", "TEMP_DIFF_PROTECT_THRESHOLD_C"),
						leafNode("温差保护恢复阈值 / 温差保护延时", "460", "0x0460", "TEMP_DIFF_PROTECT_RELEASE_C", "TEMP_DIFF_PROTECT_DELAY_S"),
					},
				},
				{
					Label: "编号配置",
					Value: "group:advanced:numbering",
					Children: []model.DeviceParamTreeNode{
						leafNode("电池组编号", "500", "0x0500", "BATTERY_GROUP_ID"),
						leafNode("DTU域名和端口地址", "53a", "0x053A", "DTU_DOMAIN_PORT"),
					},
				},
				{
					Label: "系统配置",
					Value: "group:advanced:system",
					Children: []model.DeviceParamTreeNode{
						leafNode("高低串数配置", "1", "0x0001", "SERIES_COUNT_CONFIG"),
						leafNode("电池类型", "4", "0x0004", "BATTERY_TYPE"),
						leafNode("设计容量", "30", "0x0030", "DESIGN_CAPACITY_AH"),
						leafNode("满充容量", "32", "0x0032", "FULL_CAPACITY_AH"),
						leafNode("剩余容量", "34", "0x0034", "REMAIN_CAPACITY_AH"),
						{
							Label: "功能控制",
							Value: "group:advanced:system:function",
							Children: []model.DeviceParamTreeNode{
								leafNode("保温加热功能", "function:insulationHeatingEnabled", "0x003E", "FUNCTION_CONFIG", "insulationHeatingEnabled"),
								leafNode("放电加热功能", "function:dischargeHeatingEnabled", "0x003E", "FUNCTION_CONFIG", "dischargeHeatingEnabled"),
								leafNode("低温加热功能", "function:lowTempHeatingEnabled", "0x003E", "FUNCTION_CONFIG", "lowTempHeatingEnabled"),
								leafNode("充电允许", "function:chargeAllowed", "0x003E", "FUNCTION_CONFIG", "chargeAllowed"),
								leafNode("放电允许", "function:dischargeAllowed", "0x003E", "FUNCTION_CONFIG", "dischargeAllowed"),
							},
						},
					},
				},
				{
					Label: "出厂配置",
					Value: "group:advanced:factory",
					Children: []model.DeviceParamTreeNode{
						leafNode("进入测试模式", "factory:enterTestMode", "0x057A~0x057B", "enterTestMode"),
						leafNode("退出测试模式", "factory:exitTestMode", "0x057A~0x057B", "exitTestMode"),
						leafNode("开启全均衡", "factory:balanceAllOn", "0x057A~0x057B", "balanceAllOn"),
						leafNode("关闭全均衡", "factory:balanceAllOff", "0x057A~0x057B", "balanceAllOff"),
						leafNode("进入休眠", "factory:sleep", "0x057A~0x057B", "sleep"),
						leafNode("关机", "factory:powerOff", "0x057A~0x057B", "powerOff"),
						leafNode("恢复出厂参数", "factory:restoreDefaults", "0x057A~0x057B", "restoreDefaults"),
						leafNode("清除历史数据", "factory:clearHistory", "0x057A~0x057B", "clearHistory"),
						leafNode("清除循环计数", "factory:clearCycles", "0x057A~0x057B", "clearCycles"),
						leafNode("清除保护状态", "factory:clearProtection", "0x057A~0x057B", "clearProtection"),
					},
				},
			},
		},
	}
}
