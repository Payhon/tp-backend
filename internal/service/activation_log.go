package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"project/internal/model"
	global "project/pkg/global"
	"project/pkg/utils"
)

// ActivationLog 激活日志（从 operation_logs 派生：APP 绑定接口）
type ActivationLog struct{}

func inferBindingTerminalByUA(ua string) string {
	if ua == "" {
		return "APP"
	}
	u := strings.ToLower(ua)
	if strings.Contains(u, "micromessenger") {
		return "小程序"
	}
	return "APP"
}

// GetActivationLogList 激活日志列表（分页）
func (*ActivationLog) GetActivationLogList(ctx context.Context, req model.ActivationLogListReq, claims *utils.UserClaims, dealerScopeID string) (*model.ActivationLogListResp, error) {
	// 基于 operation_logs 里 APP bind 的请求记录（POST /api/v1/app/device/bind）
	// 通过 request_message::jsonb->>'device_number' 获取序列号
	type row struct {
		DeviceNumber    string    `gorm:"column:device_number"`
		BatteryModel    *string   `gorm:"column:battery_model"`
		UserPhone       string    `gorm:"column:user_phone"`
		CreatedAt       time.Time `gorm:"column:created_at"`
		IP              string    `gorm:"column:ip"`
		UserAgent       *string   `gorm:"column:user_agent"`
		ActivationWay   string    `gorm:"column:activation_way"`
		BindingTerminal string    `gorm:"-"`
	}

	appPaths := []string{"/api/v1/app/device/bind", "/api/v1/app/device/provision/bind"}
	webPaths := []string{"/api/v1/battery/activate"}
	paths := append([]string{}, appPaths...)
	paths = append(paths, webPaths...)
	if req.Method != nil && *req.Method != "" {
		if *req.Method == "APP" {
			paths = appPaths
		} else if *req.Method == "WEB" {
			paths = webPaths
		}
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(paths)), ",")
	baseWhere := fmt.Sprintf("ol.tenant_id = ? AND ol.name = 'POST' AND ol.path IN (%s)", placeholders)
	args := []interface{}{claims.TenantID}
	for _, p := range paths {
		args = append(args, p)
	}

	if req.StartTime != nil && req.EndTime != nil {
		baseWhere += " AND ol.created_at BETWEEN ? AND ?"
		args = append(args, *req.StartTime, *req.EndTime)
	}

	if req.UserPhone != nil && *req.UserPhone != "" {
		baseWhere += " AND u.phone_number LIKE ?"
		args = append(args, "%"+*req.UserPhone+"%")
	}

	if req.DeviceNumber != nil && *req.DeviceNumber != "" {
		baseWhere += " AND d.device_number LIKE ?"
		args = append(args, "%"+*req.DeviceNumber+"%")
	}

	// dealerScope：仅看名下设备的激活（通过 device_batteries 过滤）
	if dealerScopeID != "" {
		baseWhere += " AND dbat.dealer_id = ?"
		args = append(args, dealerScopeID)
	}

	// count
	var total int64
	countSQL := `
SELECT COUNT(1)
FROM operation_logs ol
LEFT JOIN users u ON u.id = CASE
  WHEN ol.path = '/api/v1/battery/activate' THEN (ol.request_message::jsonb ->> 'user_id')
  ELSE ol.user_id
END
LEFT JOIN LATERAL (
  SELECT COALESCE(
    (ol.request_message::jsonb ->> 'device_number'),
    (
      SELECT d2.device_number
      FROM device_batteries db2
      JOIN devices d2 ON d2.id = db2.device_id AND d2.tenant_id = ol.tenant_id
      WHERE db2.item_uuid = (ol.request_message::jsonb ->> 'item_uuid')
      LIMIT 1
    ),
    (
      SELECT d3.device_number
      FROM devices d3
      WHERE d3.id = (ol.request_message::jsonb ->> 'device_id') AND d3.tenant_id = ol.tenant_id
      LIMIT 1
    )
  ) AS device_number
) req ON true
LEFT JOIN devices d ON d.device_number = req.device_number AND d.tenant_id = ol.tenant_id
LEFT JOIN device_batteries dbat ON dbat.device_id = d.id
WHERE ` + baseWhere
	_ = global.DB.WithContext(ctx).Raw(countSQL, args...).Scan(&total).Error

	// list
	offset := (req.Page - 1) * req.PageSize
	listSQL := `
SELECT
  d.device_number AS device_number,
  bm.name AS battery_model,
  u.phone_number AS user_phone,
  ol.created_at AS created_at,
  ol.ip AS ip,
  ol.remark AS user_agent,
  CASE WHEN ol.path = '/api/v1/battery/activate' THEN 'WEB后台' ELSE 'APP扫码' END AS activation_way
FROM operation_logs ol
LEFT JOIN users u ON u.id = CASE
  WHEN ol.path = '/api/v1/battery/activate' THEN (ol.request_message::jsonb ->> 'user_id')
  ELSE ol.user_id
END
LEFT JOIN LATERAL (
  SELECT COALESCE(
    (ol.request_message::jsonb ->> 'device_number'),
    (
      SELECT d2.device_number
      FROM device_batteries db2
      JOIN devices d2 ON d2.id = db2.device_id AND d2.tenant_id = ol.tenant_id
      WHERE db2.item_uuid = (ol.request_message::jsonb ->> 'item_uuid')
      LIMIT 1
    ),
    (
      SELECT d3.device_number
      FROM devices d3
      WHERE d3.id = (ol.request_message::jsonb ->> 'device_id') AND d3.tenant_id = ol.tenant_id
      LIMIT 1
    )
  ) AS device_number
) req ON true
LEFT JOIN devices d ON d.device_number = req.device_number AND d.tenant_id = ol.tenant_id
LEFT JOIN device_batteries dbat ON dbat.device_id = d.id
LEFT JOIN battery_models bm ON bm.id = dbat.battery_model_id
WHERE ` + baseWhere + `
ORDER BY ol.created_at DESC
LIMIT ? OFFSET ?
`
	listArgs := append(args, req.PageSize, offset)
	var rows []row
	if err := global.DB.WithContext(ctx).Raw(listSQL, listArgs...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]model.ActivationLogResp, 0, len(rows))
	for _, r := range rows {
		ua := ""
		if r.UserAgent != nil {
			ua = *r.UserAgent
		}
		bindingTerminal := inferBindingTerminalByUA(ua)
		if r.ActivationWay == "WEB后台" {
			bindingTerminal = "WEB"
		}
		out = append(out, model.ActivationLogResp{
			DeviceNumber:    r.DeviceNumber,
			BatteryModel:    r.BatteryModel,
			UserPhone:       r.UserPhone,
			ActivationTime:  r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
			ActivationWay:   r.ActivationWay,
			BindingTerminal: bindingTerminal,
			IP:              r.IP,
		})
	}

	return &model.ActivationLogListResp{
		List:     out,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
