package bmsbridge

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

const (
	commDebugEventUplinkRaw     = "uplink_raw"
	commDebugEventUplinkIgnored = "uplink_ignored"
	commDebugEventUplinkDecoded = "uplink_decoded"
	commDebugEventUplinkParsed  = "uplink_parsed"
	commDebugEventUplinkError   = "uplink_error"
	commDebugEventDownlinkPub   = "downlink_publish"
	commDebugEventDownlinkError = "downlink_error"

	commDebugDirectionInbound  = "inbound"
	commDebugDirectionOutbound = "outbound"

	commDebugStatusSuccess = "success"
	commDebugStatusError   = "error"
)

type tracedDeviceMeta struct {
	TenantID     string  `gorm:"column:tenant_id"`
	DeviceID     string  `gorm:"column:device_id"`
	DeviceNumber string  `gorm:"column:device_number"`
	BmsCommType  *int    `gorm:"column:bms_comm_type"`
	Imei         *string `gorm:"column:imei"`
	Iccid        *string `gorm:"column:iccid"`
	ModuleSW     *string `gorm:"column:module_sw_version"`
	CommChipID   *string `gorm:"column:comm_chip_id"`
}

type commDebugLogCreateRow struct {
	TenantID      string          `gorm:"column:tenant_id"`
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
	OccurredAt    time.Time       `gorm:"column:occurred_at"`
	CreatedAt     time.Time       `gorm:"column:created_at"`
}

type commDebugTraceEntry struct {
	DeviceID      string
	EventType     string
	Direction     string
	MQTTTopic     *string
	QoS           *int
	MessageID     *string
	PayloadRaw    *string
	PayloadFormat *string
	ParsedSummary any
	Status        string
	ErrorMessage  *string
	OccurredAt    time.Time
}

func (b *Bridge) getTraceDeviceMeta(ctx context.Context, deviceID string) (*tracedDeviceMeta, bool) {
	if b.db == nil || strings.TrimSpace(deviceID) == "" {
		return nil, false
	}

	if cached, ok := b.traceMeta.Load(deviceID); ok {
		if meta, ok := cached.(*tracedDeviceMeta); ok && meta != nil {
			return meta, true
		}
		return nil, false
	}

	var meta tracedDeviceMeta
	err := b.db.WithContext(ctx).
		Table("devices AS d").
		Select(`
			d.tenant_id,
			d.id AS device_id,
			d.device_number,
			dbat.bms_comm_type,
			dbat.imei,
			dbat.iccid,
			dbat.module_sw_version,
			dbat.comm_chip_id
		`).
		Joins("INNER JOIN device_batteries AS dbat ON dbat.device_id = d.id").
		Where("d.id = ?", deviceID).
		Limit(1).
		Scan(&meta).Error
	if err != nil || strings.TrimSpace(meta.DeviceID) == "" {
		b.traceMeta.Store(deviceID, (*tracedDeviceMeta)(nil))
		return nil, false
	}

	if !has4GMarker(meta.BmsCommType, meta.Imei, meta.Iccid, meta.ModuleSW, meta.CommChipID) {
		b.traceMeta.Store(deviceID, (*tracedDeviceMeta)(nil))
		return nil, false
	}

	copyMeta := meta
	b.traceMeta.Store(deviceID, &copyMeta)
	return &copyMeta, true
}

func has4GMarker(commType *int, values ...*string) bool {
	if commType != nil && (*commType == 2 || *commType == 3) {
		return true
	}
	for _, v := range values {
		if v != nil && strings.TrimSpace(*v) != "" {
			return true
		}
	}
	return false
}

func marshalParsedSummary(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	bs, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return bs
}

func (b *Bridge) traceCommDebug(ctx context.Context, entry commDebugTraceEntry) {
	meta, ok := b.getTraceDeviceMeta(ctx, entry.DeviceID)
	if !ok || meta == nil {
		return
	}

	row := commDebugLogCreateRow{
		TenantID:      meta.TenantID,
		DeviceID:      meta.DeviceID,
		DeviceNumber:  meta.DeviceNumber,
		Source:        "bms_bridge",
		AccessMode:    "4G",
		EventType:     entry.EventType,
		Direction:     entry.Direction,
		MQTTTopic:     entry.MQTTTopic,
		QoS:           entry.QoS,
		MessageID:     entry.MessageID,
		PayloadRaw:    entry.PayloadRaw,
		PayloadFormat: entry.PayloadFormat,
		ParsedSummary: marshalParsedSummary(entry.ParsedSummary),
		Status:        entry.Status,
		ErrorMessage:  entry.ErrorMessage,
		OccurredAt:    entry.OccurredAt,
		CreatedAt:     time.Now(),
	}

	if err := b.db.WithContext(ctx).Table("bms_bridge_comm_logs").Create(&row).Error; err != nil {
		b.log.WithError(err).WithField("device_id", entry.DeviceID).Warn("failed to write bms bridge comm debug log")
	}
}
