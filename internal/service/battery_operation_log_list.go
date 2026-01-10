package service

import (
	"context"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	"project/pkg/utils"
)

func parseTimeFlexible(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	// 优先 RFC3339（前端常用）
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t, nil
	}
	// 兼容 YYYY-MM-DD
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return &t, nil
	}
	// 兼容 YYYY-MM-DD HH:mm:ss
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return &t, nil
	}
	return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "时间格式错误"})
}

func (*Battery) GetBatteryOperationLogList(ctx context.Context, req model.BatteryOperationLogListReq, claims *utils.UserClaims, orgID string) (*model.BatteryOperationLogListResp, error) {
	start, err := parseTimeFlexible(ptrToStr(req.StartTime))
	if err != nil {
		return nil, err
	}
	end, err := parseTimeFlexible(ptrToStr(req.EndTime))
	if err != nil {
		return nil, err
	}

	total, rows, err := GetBatteryOperationLogList(ctx, claims.TenantID, orgID, req.Page, req.PageSize, req.DeviceNumber, req.OperationType, start, end)
	if err != nil {
		return nil, err
	}

	list := make([]model.BatteryOperationLogItemResp, 0, len(rows))
	for _, r := range rows {
		list = append(list, model.BatteryOperationLogItemResp{
			ID:            r.ID,
			OccurredAt:    r.OccurredAt.Format("2006-01-02 15:04:05"),
			DeviceID:      r.DeviceID,
			DeviceNumber:  r.DeviceNumber,
			OperationType: r.OperationType,
			OperatorID:    r.OperatorID,
			OperatorName:  r.OperatorName,
			Description:   r.Description,
		})
	}

	return &model.BatteryOperationLogListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
