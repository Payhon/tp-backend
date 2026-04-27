package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"
)

const (
	BmsCommDebugSourceBmsBridge = "bms_bridge"
	BmsCommDebugAccessMode4G    = "4G"
)

type BmsCommDebug struct{}

type bmsCommDebugLogRow struct {
	ID            int64           `gorm:"column:id"`
	OccurredAt    time.Time       `gorm:"column:occurred_at"`
	DeviceID      string          `gorm:"column:device_id"`
	DeviceNumber  string          `gorm:"column:device_number"`
	Source        string          `gorm:"column:source"`
	AccessMode    string          `gorm:"column:access_mode"`
	EventType     string          `gorm:"column:event_type"`
	Direction     string          `gorm:"column:direction"`
	MQTTTopic     *string         `gorm:"column:mqtt_topic"`
	QoS           *int            `gorm:"column:qos"`
	MessageID     *string         `gorm:"column:message_id"`
	PayloadRaw    *string         `gorm:"column:payload_raw"`
	PayloadFormat *string         `gorm:"column:payload_format"`
	ParsedSummary json.RawMessage `gorm:"column:parsed_summary"`
	Status        string          `gorm:"column:status"`
	ErrorMessage  *string         `gorm:"column:error_message"`
}

func getBmsCommDebugLogList(ctx context.Context, tenantID, orgID string, page, pageSize int, deviceID, deviceNumber, eventType, status *string, startTime, endTime *time.Time) (int64, []bmsCommDebugLogRow, error) {
	db := global.DB.WithContext(ctx)

	q := db.Table("bms_bridge_comm_logs AS l").
		Select(`
			l.id,
			l.occurred_at,
			l.device_id,
			l.device_number,
			l.source,
			l.access_mode,
			l.event_type,
			l.direction,
			l.mqtt_topic,
			l.qos,
			l.message_id,
			l.payload_raw,
			l.payload_format,
			l.parsed_summary,
			l.status,
			l.error_message
		`).
		Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = l.device_id`).
		Where("l.tenant_id = ?", tenantID)

	if orgID != "" {
		q = q.Where(`dbat.owner_org_id IN (
			SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
		)`, tenantID, orgID)
	}
	if deviceID != nil && strings.TrimSpace(*deviceID) != "" {
		q = q.Where("l.device_id = ?", strings.TrimSpace(*deviceID))
	}
	if deviceNumber != nil && strings.TrimSpace(*deviceNumber) != "" {
		q = q.Where("l.device_number ILIKE ?", "%"+strings.TrimSpace(*deviceNumber)+"%")
	}
	if eventType != nil && strings.TrimSpace(*eventType) != "" {
		q = q.Where("l.event_type = ?", strings.TrimSpace(*eventType))
	}
	if status != nil && strings.TrimSpace(*status) != "" {
		q = q.Where("l.status = ?", strings.TrimSpace(*status))
	}
	if startTime != nil {
		q = q.Where("l.occurred_at >= ?", *startTime)
	}
	if endTime != nil {
		q = q.Where("l.occurred_at <= ?", *endTime)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (page - 1) * pageSize
	rows := make([]bmsCommDebugLogRow, 0, pageSize)
	if err := q.Order("l.occurred_at DESC, l.id DESC").Offset(offset).Limit(pageSize).Scan(&rows).Error; err != nil {
		return 0, nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return total, rows, nil
}

func getBmsCommDebugLogsAfterID(ctx context.Context, tenantID string, afterID int64, limit int, deviceID, eventType *string) ([]bmsCommDebugLogRow, error) {
	db := global.DB.WithContext(ctx)
	q := db.Table("bms_bridge_comm_logs AS l").
		Select(`
			l.id,
			l.occurred_at,
			l.device_id,
			l.device_number,
			l.source,
			l.access_mode,
			l.event_type,
			l.direction,
			l.mqtt_topic,
			l.qos,
			l.message_id,
			l.payload_raw,
			l.payload_format,
			l.parsed_summary,
			l.status,
			l.error_message
		`).
		Where("l.tenant_id = ? AND l.id > ?", tenantID, afterID)

	if deviceID != nil && strings.TrimSpace(*deviceID) != "" {
		q = q.Where("l.device_id = ?", strings.TrimSpace(*deviceID))
	}
	if eventType != nil && strings.TrimSpace(*eventType) != "" {
		q = q.Where("l.event_type = ?", strings.TrimSpace(*eventType))
	}

	rows := make([]bmsCommDebugLogRow, 0, limit)
	if err := q.Order("l.id ASC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return rows, nil
}

func parseParsedSummary(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}

func (BmsCommDebug) GetLogList(ctx context.Context, req model.BmsCommDebugLogListReq, claims *utils.UserClaims, orgID string) (*model.BmsCommDebugLogListResp, error) {
	start, err := parseTimeFlexible(ptrToStr(req.StartTime))
	if err != nil {
		return nil, err
	}
	end, err := parseTimeFlexible(ptrToStr(req.EndTime))
	if err != nil {
		return nil, err
	}

	total, rows, err := getBmsCommDebugLogList(ctx, claims.TenantID, orgID, req.Page, req.PageSize, req.DeviceID, req.DeviceNumber, req.EventType, req.Status, start, end)
	if err != nil {
		return nil, err
	}

	list := make([]model.BmsCommDebugLogItemResp, 0, len(rows))
	for _, r := range rows {
		list = append(list, model.BmsCommDebugLogItemResp{
			ID:            r.ID,
			OccurredAt:    r.OccurredAt.Format("2006-01-02 15:04:05"),
			DeviceID:      r.DeviceID,
			DeviceNumber:  r.DeviceNumber,
			Source:        r.Source,
			AccessMode:    r.AccessMode,
			EventType:     r.EventType,
			Direction:     r.Direction,
			MQTTTopic:     r.MQTTTopic,
			QoS:           r.QoS,
			MessageID:     r.MessageID,
			PayloadRaw:    r.PayloadRaw,
			PayloadFormat: r.PayloadFormat,
			ParsedSummary: parseParsedSummary(r.ParsedSummary),
			Status:        r.Status,
			ErrorMessage:  r.ErrorMessage,
		})
	}

	return &model.BmsCommDebugLogListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (BmsCommDebug) GetLogsAfterID(ctx context.Context, tenantID string, afterID int64, limit int, deviceID, eventType *string) ([]model.BmsCommDebugLogItemResp, error) {
	rows, err := getBmsCommDebugLogsAfterID(ctx, tenantID, afterID, limit, deviceID, eventType)
	if err != nil {
		return nil, err
	}

	list := make([]model.BmsCommDebugLogItemResp, 0, len(rows))
	for _, r := range rows {
		list = append(list, model.BmsCommDebugLogItemResp{
			ID:            r.ID,
			OccurredAt:    r.OccurredAt.Format("2006-01-02 15:04:05"),
			DeviceID:      r.DeviceID,
			DeviceNumber:  r.DeviceNumber,
			Source:        r.Source,
			AccessMode:    r.AccessMode,
			EventType:     r.EventType,
			Direction:     r.Direction,
			MQTTTopic:     r.MQTTTopic,
			QoS:           r.QoS,
			MessageID:     r.MessageID,
			PayloadRaw:    r.PayloadRaw,
			PayloadFormat: r.PayloadFormat,
			ParsedSummary: parseParsedSummary(r.ParsedSummary),
			Status:        r.Status,
			ErrorMessage:  r.ErrorMessage,
		})
	}

	return list, nil
}
