package service

import (
	"context"
	"strings"
	"time"

	"project/internal/model"
	query "project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"gorm.io/gorm"
)

// BmsDashboard BMS: 首页看板（指标/告警概览/趋势）
type BmsDashboard struct{}

func buildBatteryScopeBase(db *gorm.DB, tenantID, orgID string) func() *gorm.DB {
	return func() *gorm.DB {
		// 设备范围：按 tenant + 可选 org 子树（device_batteries.owner_org_id）
		base := db.Table("devices AS d").
			Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
			Where("d.tenant_id = ?", tenantID)
		if orgID != "" {
			base = base.Where(`dbat.owner_org_id IN (
				SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
			)`, tenantID, orgID)
		}
		return base
	}
}

func buildBatteryScopeByOrgType(db *gorm.DB, tenantID, orgType string) func() *gorm.DB {
	return func() *gorm.DB {
		return db.Table("devices AS d").
			Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
			Where("d.tenant_id = ?", tenantID).
			Where(`dbat.owner_org_id IN (
				SELECT id FROM orgs WHERE tenant_id = ? AND org_type = ?
			)`, tenantID, orgType)
	}
}

func buildLatestDeviceAlarmsBase(db *gorm.DB, tenantID string, orgID string, start time.Time) *gorm.DB {
	q := db.Table("latest_device_alarms AS lda").
		Where("lda.tenant_id = ?", tenantID).
		Where("lda.create_at >= ?", start)

	// 组织隔离：使用 distinct device_id 子查询，避免 device_batteries 多行导致统计被放大
	if orgID != "" {
		orgDeviceSubQ := db.Table("device_batteries AS dbat").
			Select("DISTINCT dbat.device_id").
			Where(`dbat.owner_org_id IN (
				SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
			)`, tenantID, orgID)
		q = q.Joins("JOIN (?) AS org_dev ON org_dev.device_id = lda.device_id", orgDeviceSubQ)
	}

	return q
}

// GetKpi 指标卡（按厂家/组织隔离）
func (*BmsDashboard) GetKpi(ctx context.Context, claims *utils.UserClaims, orgID string) (*model.BmsDashboardKpiResp, error) {
	db := global.DB.WithContext(ctx)

	base := buildBatteryScopeBase(db, claims.TenantID, orgID)

	var deviceTotal int64
	if err := base().Distinct("d.id").Count(&deviceTotal).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	var deviceOnline int64
	if err := base().Where("d.is_online = ?", 1).Distinct("d.id").Count(&deviceOnline).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	// 激活口径：device_batteries.activation_status = ACTIVE
	var deviceActivated int64
	if err := base().Where("dbat.activation_status = ?", "ACTIVE").Distinct("d.id").Count(&deviceActivated).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	// 活跃告警：latest_device_alarms 中非 N（按设备范围过滤）
	alarmQ := db.Table("latest_device_alarms AS lda").
		Where("lda.tenant_id = ?", claims.TenantID).
		Where("lda.alarm_status IS NOT NULL AND lda.alarm_status <> ?", "N")
	if orgID != "" {
		orgDeviceSubQ := db.Table("device_batteries AS dbat").
			Select("DISTINCT dbat.device_id").
			Where(`dbat.owner_org_id IN (
				SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
			)`, claims.TenantID, orgID)
		alarmQ = alarmQ.Joins("JOIN (?) AS org_dev ON org_dev.device_id = lda.device_id", orgDeviceSubQ)
	}
	var alarmActive int64
	if err := alarmQ.Count(&alarmActive).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return &model.BmsDashboardKpiResp{
		DeviceTotal:     deviceTotal,
		DeviceOnline:    deviceOnline,
		DeviceActivated: deviceActivated,
		AlarmActive:     alarmActive,
	}, nil
}

// GetAlarmOverview 告警概览（状态分布 + Top + 近N天趋势）
func (*BmsDashboard) GetAlarmOverview(ctx context.Context, claims *utils.UserClaims, orgID string, days int) (*model.BmsDashboardAlarmOverviewResp, error) {
	if days <= 0 {
		days = 7
	}
	db := global.DB.WithContext(ctx)
	start := time.Now().AddDate(0, 0, -days)
	newBase := func() *gorm.DB {
		return buildLatestDeviceAlarmsBase(db, claims.TenantID, orgID, start)
	}

	// 状态分布（H/M/L/N）
	type statusRow struct {
		AlarmStatus string `gorm:"column:alarm_status"`
		Cnt         int64  `gorm:"column:cnt"`
	}
	var statusRows []statusRow
	if err := newBase().
		Select("lda.alarm_status AS alarm_status, COUNT(1) AS cnt").
		Group("lda.alarm_status").
		Scan(&statusRows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	statusCounts := make([]model.BmsDashboardAlarmStatusCount, 0, len(statusRows))
	for _, r := range statusRows {
		statusCounts = append(statusCounts, model.BmsDashboardAlarmStatusCount{Status: r.AlarmStatus, Count: r.Cnt})
	}

	// Top3 告警名称
	type topRow struct {
		Name string `gorm:"column:name"`
		Cnt  int64  `gorm:"column:cnt"`
	}
	var topRows []topRow
	if err := newBase().
		Select("COALESCE(lda.name, '') AS name, COUNT(1) AS cnt").
		Group("lda.name").
		Order("cnt DESC").
		Limit(3).
		Scan(&topRows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	top := make([]model.BmsDashboardAlarmTopItem, 0, len(topRows))
	for _, r := range topRows {
		name := r.Name
		if name == "" {
			name = "未命名"
		}
		top = append(top, model.BmsDashboardAlarmTopItem{Name: name, Count: r.Cnt})
	}

	// 趋势：按天
	type trendRow struct {
		Day string `gorm:"column:day"`
		Cnt int64  `gorm:"column:cnt"`
	}
	var trendRows []trendRow
	if err := newBase().
		Select("to_char(lda.create_at::date, 'YYYY-MM-DD') AS day, COUNT(1) AS cnt").
		Group("lda.create_at::date").
		Order("day ASC").
		Scan(&trendRows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	trend := make([]model.BmsDashboardAlarmTrendPoint, 0, len(trendRows))
	for _, r := range trendRows {
		trend = append(trend, model.BmsDashboardAlarmTrendPoint{Date: r.Day, Count: r.Cnt})
	}

	return &model.BmsDashboardAlarmOverviewResp{
		StatusCounts: statusCounts,
		Top:          top,
		Trend:        trend,
	}, nil
}

// GetOnlineTrend 在线趋势（目前复用 tenant 级 Redis 统计；组织仅返回当前点）
func (*BmsDashboard) GetOnlineTrend(ctx context.Context, claims *utils.UserClaims, orgID string) (*model.BmsDashboardOnlineTrendResp, error) {
	// 组织用户：缺少历史采样（避免返回不准确的 tenant 口径）
	if orgID != "" {
		// 当前点：从 DB 计算
		kpi, err := GroupApp.BmsDashboard.GetKpi(ctx, claims, orgID)
		if err != nil {
			return nil, err
		}
		now := time.Now()
		return &model.BmsDashboardOnlineTrendResp{
			Points: []model.BmsDashboardOnlineTrendPoint{
				{
					Timestamp:     now,
					DeviceTotal:   kpi.DeviceTotal,
					DeviceOnline:  kpi.DeviceOnline,
					DeviceOffline: kpi.DeviceTotal - kpi.DeviceOnline,
				},
			},
		}, nil
	}

	// 厂家：复用现有 device_stats:{tenant}:{date} 数据
	trend, err := GroupApp.Device.GetDeviceTrend(ctx, claims.TenantID)
	if err != nil {
		return nil, err
	}

	points := make([]model.BmsDashboardOnlineTrendPoint, 0, len(trend.Points))
	for _, p := range trend.Points {
		points = append(points, model.BmsDashboardOnlineTrendPoint{
			Timestamp:     p.Timestamp,
			DeviceTotal:   p.DeviceTotal,
			DeviceOnline:  p.DeviceOnline,
			DeviceOffline: p.DeviceOffline,
		})
	}

	return &model.BmsDashboardOnlineTrendResp{Points: points}, nil
}

// 让 gorm/gen 的 query 引用被 go 编译器认为已使用（避免未来 refactor 误删）
var _ = query.Device
var _ = gorm.ErrRecordNotFound

// GetHomeSummary 首页汇总（按用户类型）
func (*BmsDashboard) GetHomeSummary(ctx context.Context, claims *utils.UserClaims, orgID string, viewAs string) (*model.BmsHomeSummaryResp, error) {
	resp := &model.BmsHomeSummaryResp{
		UserKind: model.UserKindEndUser,
		OrgType:  model.OrgTypeAppUser,
	}
	if claims == nil || claims.ID == "" || claims.TenantID == "" {
		return resp, nil
	}

	viewAs = strings.ToUpper(strings.TrimSpace(viewAs))
	isAdmin := claims.Authority == "SYS_ADMIN" || claims.Authority == "TENANT_ADMIN"
	if isAdmin && viewAs != "" && viewAs != "TENANT" {
		switch viewAs {
		case model.OrgTypePACKFactory, model.OrgTypeDealer, model.OrgTypeStore:
			weekTrend, err := GroupApp.BmsDashboard.getActivationTrendByOrgType(ctx, claims.TenantID, viewAs, 7)
			if err != nil {
				return nil, err
			}
			monthTrend, err := GroupApp.BmsDashboard.getActivationTrendByOrgType(ctx, claims.TenantID, viewAs, 30)
			if err != nil {
				return nil, err
			}
			kpi, err := GroupApp.BmsDashboard.getKpiByOrgType(ctx, claims.TenantID, viewAs)
			if err != nil {
				return nil, err
			}

			resp.UserKind = model.UserKindOrgUser
			resp.OrgType = viewAs
			resp.Institution = &model.BmsInstitutionHomeSummaryResp{
				BatteryTotal:         kpi.DeviceTotal,
				OnlineCount:          kpi.DeviceOnline,
				OfflineCount:         kpi.DeviceTotal - kpi.DeviceOnline,
				ActivatedCount:       kpi.DeviceActivated,
				InactiveCount:        kpi.DeviceTotal - kpi.DeviceActivated,
				ActivationTrendWeek:  weekTrend,
				ActivationTrendMonth: monthTrend,
			}
			return resp, nil
		case model.OrgTypeAppUser:
			total, err := GroupApp.BmsDashboard.getEndUserBatteryTotalByTenant(ctx, claims.TenantID)
			if err != nil {
				return nil, err
			}
			resp.UserKind = model.UserKindEndUser
			resp.OrgType = model.OrgTypeAppUser
			resp.EndUser = &model.BmsEndUserHomeSummaryResp{
				BatteryTotal: total,
			}
			return resp, nil
		}
	}

	userKind, err := GroupApp.OrgTypePermission.GetUserKind(ctx, claims.TenantID, claims.ID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_user_kind",
			"user_id":   claims.ID,
			"error":     err.Error(),
		})
	}
	resp.UserKind = userKind

	if userKind == model.UserKindOrgUser {
		orgType := ""
		if t, ok, orgErr := GroupApp.OrgTypePermission.GetUserOrgType(ctx, claims.TenantID, claims.ID); orgErr != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_user_org_type",
				"user_id":   claims.ID,
				"error":     orgErr.Error(),
			})
		} else if ok {
			orgType = t
		}
		resp.OrgType = orgType

		kpi, err := GroupApp.BmsDashboard.GetKpi(ctx, claims, orgID)
		if err != nil {
			return nil, err
		}
		weekTrend, err := GroupApp.BmsDashboard.GetActivationTrend(ctx, claims, orgID, 7)
		if err != nil {
			return nil, err
		}
		monthTrend, err := GroupApp.BmsDashboard.GetActivationTrend(ctx, claims, orgID, 30)
		if err != nil {
			return nil, err
		}

		resp.Institution = &model.BmsInstitutionHomeSummaryResp{
			BatteryTotal:         kpi.DeviceTotal,
			OnlineCount:          kpi.DeviceOnline,
			OfflineCount:         kpi.DeviceTotal - kpi.DeviceOnline,
			ActivatedCount:       kpi.DeviceActivated,
			InactiveCount:        kpi.DeviceTotal - kpi.DeviceActivated,
			ActivationTrendWeek:  weekTrend,
			ActivationTrendMonth: monthTrend,
		}
		return resp, nil
	}

	total, err := GroupApp.BmsDashboard.GetEndUserBatteryTotal(ctx, claims)
	if err != nil {
		return nil, err
	}
	resp.OrgType = model.OrgTypeAppUser
	resp.EndUser = &model.BmsEndUserHomeSummaryResp{
		BatteryTotal: total,
	}
	return resp, nil
}

// GetActivationTrend 激活趋势（按天）
func (*BmsDashboard) GetActivationTrend(ctx context.Context, claims *utils.UserClaims, orgID string, days int) ([]model.BmsDashboardActivationTrendPoint, error) {
	if days <= 0 {
		days = 7
	}
	if days > 365 {
		days = 365
	}

	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))

	db := global.DB.WithContext(ctx)
	q := db.Table("device_batteries AS dbat").
		Select("to_char(dbat.activation_date::date, 'YYYY-MM-DD') AS day, COUNT(DISTINCT dbat.device_id) AS cnt").
		Joins("JOIN devices AS d ON d.id = dbat.device_id").
		Where("d.tenant_id = ?", claims.TenantID).
		Where("dbat.activation_status = ?", "ACTIVE").
		Where("dbat.activation_date IS NOT NULL").
		Where("dbat.activation_date >= ?", startDate)

	if orgID != "" {
		q = q.Where(`dbat.owner_org_id IN (
			SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
		)`, claims.TenantID, orgID)
	}

	type dayCountRow struct {
		Day string `gorm:"column:day"`
		Cnt int64  `gorm:"column:cnt"`
	}
	var rows []dayCountRow
	if err := q.Group("dbat.activation_date::date").Order("day ASC").Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_activation_trend",
			"error":     err.Error(),
		})
	}

	countMap := make(map[string]int64, len(rows))
	for _, row := range rows {
		countMap[row.Day] = row.Cnt
	}

	result := make([]model.BmsDashboardActivationTrendPoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		result = append(result, model.BmsDashboardActivationTrendPoint{
			Date:  day,
			Count: countMap[day],
		})
	}
	return result, nil
}

// GetEndUserBatteryTotal 获取终端用户持有电池总数
func (*BmsDashboard) GetEndUserBatteryTotal(ctx context.Context, claims *utils.UserClaims) (int64, error) {
	var total int64
	if err := global.DB.WithContext(ctx).
		Table("device_user_bindings AS dub").
		Distinct("dub.device_id").
		Joins("JOIN devices AS d ON d.id = dub.device_id").
		Where("dub.user_id = ? AND d.tenant_id = ?", claims.ID, claims.TenantID).
		Count(&total).Error; err != nil {
		return 0, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_end_user_battery_total",
			"error":     err.Error(),
		})
	}
	return total, nil
}

func (*BmsDashboard) getKpiByOrgType(ctx context.Context, tenantID, orgType string) (*model.BmsDashboardKpiResp, error) {
	db := global.DB.WithContext(ctx)

	base := buildBatteryScopeByOrgType(db, tenantID, orgType)

	var deviceTotal int64
	if err := base().Distinct("d.id").Count(&deviceTotal).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	var deviceOnline int64
	if err := base().Where("d.is_online = ?", 1).Distinct("d.id").Count(&deviceOnline).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	var deviceActivated int64
	if err := base().Where("dbat.activation_status = ?", "ACTIVE").Distinct("d.id").Count(&deviceActivated).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	orgDeviceSubQ := db.Table("device_batteries AS dbat").
		Select("DISTINCT dbat.device_id").
		Where(`dbat.owner_org_id IN (
			SELECT id FROM orgs WHERE tenant_id = ? AND org_type = ?
		)`, tenantID, orgType)
	alarmQ := db.Table("latest_device_alarms AS lda").
		Where("lda.tenant_id = ?", tenantID).
		Where("lda.alarm_status IS NOT NULL AND lda.alarm_status <> ?", "N").
		Joins("JOIN (?) AS org_dev ON org_dev.device_id = lda.device_id", orgDeviceSubQ)

	var alarmActive int64
	if err := alarmQ.Count(&alarmActive).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return &model.BmsDashboardKpiResp{
		DeviceTotal:     deviceTotal,
		DeviceOnline:    deviceOnline,
		DeviceActivated: deviceActivated,
		AlarmActive:     alarmActive,
	}, nil
}

func (*BmsDashboard) getActivationTrendByOrgType(ctx context.Context, tenantID, orgType string, days int) ([]model.BmsDashboardActivationTrendPoint, error) {
	if days <= 0 {
		days = 7
	}
	if days > 365 {
		days = 365
	}

	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))

	db := global.DB.WithContext(ctx)
	q := db.Table("device_batteries AS dbat").
		Select("to_char(dbat.activation_date::date, 'YYYY-MM-DD') AS day, COUNT(DISTINCT dbat.device_id) AS cnt").
		Joins("JOIN devices AS d ON d.id = dbat.device_id").
		Where("d.tenant_id = ?", tenantID).
		Where("dbat.activation_status = ?", "ACTIVE").
		Where("dbat.activation_date IS NOT NULL").
		Where("dbat.activation_date >= ?", startDate).
		Where(`dbat.owner_org_id IN (
			SELECT id FROM orgs WHERE tenant_id = ? AND org_type = ?
		)`, tenantID, orgType)

	type dayCountRow struct {
		Day string `gorm:"column:day"`
		Cnt int64  `gorm:"column:cnt"`
	}
	var rows []dayCountRow
	if err := q.Group("dbat.activation_date::date").Order("day ASC").Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_activation_trend_by_org_type",
			"error":     err.Error(),
		})
	}

	countMap := make(map[string]int64, len(rows))
	for _, row := range rows {
		countMap[row.Day] = row.Cnt
	}

	result := make([]model.BmsDashboardActivationTrendPoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		result = append(result, model.BmsDashboardActivationTrendPoint{
			Date:  day,
			Count: countMap[day],
		})
	}
	return result, nil
}

func (*BmsDashboard) getEndUserBatteryTotalByTenant(ctx context.Context, tenantID string) (int64, error) {
	var total int64
	if err := global.DB.WithContext(ctx).
		Table("device_user_bindings AS dub").
		Distinct("dub.device_id").
		Joins("JOIN devices AS d ON d.id = dub.device_id").
		Where("d.tenant_id = ?", tenantID).
		Count(&total).Error; err != nil {
		return 0, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_end_user_battery_total_by_tenant",
			"error":     err.Error(),
		})
	}
	return total, nil
}
