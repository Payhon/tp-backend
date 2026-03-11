package model

import "time"

// BmsDashboardKpiResp BMS Dashboard 指标卡
type BmsDashboardKpiResp struct {
	DeviceTotal     int64 `json:"device_total"`
	DeviceOnline    int64 `json:"device_online"`
	DeviceActivated int64 `json:"device_activated"`

	AlarmActive int64 `json:"alarm_active"` // latest_device_alarms 中非 N 的数量
}

// BmsDashboardAlarmStatusCount 告警状态数量
type BmsDashboardAlarmStatusCount struct {
	Status string `json:"status"` // H/M/L/N
	Count  int64  `json:"count"`
}

// BmsDashboardAlarmTopItem 告警Top
type BmsDashboardAlarmTopItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// BmsDashboardAlarmTrendPoint 告警趋势点（按天）
type BmsDashboardAlarmTrendPoint struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int64  `json:"count"`
}

// BmsDashboardAlarmOverviewResp 告警概览
type BmsDashboardAlarmOverviewResp struct {
	StatusCounts []BmsDashboardAlarmStatusCount `json:"status_counts"`
	Top          []BmsDashboardAlarmTopItem     `json:"top"`
	Trend        []BmsDashboardAlarmTrendPoint  `json:"trend"`
}

// BmsDashboardOnlineTrendPoint 在线趋势点（按小时/按采样）
type BmsDashboardOnlineTrendPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	DeviceTotal   int64     `json:"device_total"`
	DeviceOnline  int64     `json:"device_online"`
	DeviceOffline int64     `json:"device_offline"`
}

// BmsDashboardOnlineTrendResp 在线趋势
type BmsDashboardOnlineTrendResp struct {
	Points []BmsDashboardOnlineTrendPoint `json:"points"`
}

// BmsDashboardActivationTrendPoint 激活趋势点（按天）
type BmsDashboardActivationTrendPoint struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int64  `json:"count"`
}

// BmsInstitutionHomeSummaryResp 机构首页汇总
type BmsInstitutionHomeSummaryResp struct {
	BatteryTotal         int64                              `json:"battery_total"`
	OnlineCount          int64                              `json:"online_count"`
	OfflineCount         int64                              `json:"offline_count"`
	ActivatedCount       int64                              `json:"activated_count"`
	InactiveCount        int64                              `json:"inactive_count"`
	ActivationTrendWeek  []BmsDashboardActivationTrendPoint `json:"activation_trend_week"`
	ActivationTrendMonth []BmsDashboardActivationTrendPoint `json:"activation_trend_month"`
}

// BmsEndUserHomeSummaryResp 终端用户首页汇总
type BmsEndUserHomeSummaryResp struct {
	BatteryTotal int64 `json:"battery_total"`
}

// BmsHomeSummaryResp 首页汇总（按用户类型返回）
type BmsHomeSummaryResp struct {
	UserKind    string                         `json:"user_kind"`             // ORG_USER | END_USER
	OrgType     string                         `json:"org_type"`              // PACK_FACTORY/DEALER/STORE/APP_USER
	Institution *BmsInstitutionHomeSummaryResp `json:"institution,omitempty"` // 机构首页
	EndUser     *BmsEndUserHomeSummaryResp     `json:"end_user,omitempty"`    // 终端用户首页
}
