package service

import (
	"context"
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

	baseWhere := `
ol.tenant_id = ? AND ol.path = '/api/v1/app/device/bind' AND ol.name = 'POST'
`
	args := []interface{}{claims.TenantID}

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

	// Method：当前仅实现 APP（WEB 手动绑定暂无对应接口/日志来源）
	if req.Method != nil && *req.Method != "" {
		if *req.Method == "WEB" {
			return &model.ActivationLogListResp{
				List:     []model.ActivationLogResp{},
				Total:    0,
				Page:     req.Page,
				PageSize: req.PageSize,
			}, nil
		}
	}

	// count
	var total int64
	countSQL := `
SELECT COUNT(1)
FROM operation_logs ol
LEFT JOIN users u ON u.id = ol.user_id
LEFT JOIN LATERAL (
  SELECT (ol.request_message::jsonb ->> 'device_number') AS device_number
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
  'APP扫码' AS activation_way
FROM operation_logs ol
LEFT JOIN users u ON u.id = ol.user_id
LEFT JOIN LATERAL (
  SELECT (ol.request_message::jsonb ->> 'device_number') AS device_number
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
		out = append(out, model.ActivationLogResp{
			DeviceNumber:    r.DeviceNumber,
			BatteryModel:    r.BatteryModel,
			UserPhone:       r.UserPhone,
			ActivationTime:  r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
			ActivationWay:   r.ActivationWay,
			BindingTerminal: inferBindingTerminalByUA(ua),
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
