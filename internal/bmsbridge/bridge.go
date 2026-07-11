package bmsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"project/internal/adapter/mqttadapter"
	"project/internal/bms/protocol"
	"project/internal/bms/status"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Bridge struct {
	cfg Config
	db  *gorm.DB
	log *logrus.Logger

	rules *FileRulesProvider

	client mqtt.Client

	queues []chan incoming
	wg     sync.WaitGroup

	boolState  *BoolStateStore
	traceMeta  sync.Map
	statusMeta sync.Map

	receivedAtMu     sync.Mutex
	lastReceivedAtMs map[string]int64
}

type incoming struct {
	topic       string
	rawDeviceID string
	deviceID    string
	payload     []byte
	qos         byte
	messageID   string
	receivedAt  time.Time
}

type socketPayload struct {
	Hex string `json:"hex"`
}

func New(cfg Config, db *gorm.DB, log *logrus.Logger) *Bridge {
	if log == nil {
		log = logrus.StandardLogger()
	}
	return &Bridge{
		cfg:              cfg,
		db:               db,
		log:              log,
		rules:            NewFileRulesProvider(cfg.Rules.Path, cfg.Rules.ReloadIntervalSec),
		queues:           newIncomingShards(cfg.Workers.Concurrency, cfg.Workers.QueueSize),
		boolState:        NewBoolStateStore(time.Duration(cfg.Events.StateTTLMinutes) * time.Minute),
		lastReceivedAtMs: make(map[string]int64),
	}
}

func (b *Bridge) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("bms bridge is disabled by config")
		return nil
	}

	mqttCfg := mqttadapter.MQTTConfig{
		Broker:   b.cfg.MQTT.Broker,
		Username: b.cfg.MQTT.User,
		Password: b.cfg.MQTT.Pass,
		ClientID: b.cfg.MQTT.ClientID,
		DeliveryOptions: &mqttadapter.MQTTDeliveryOptions{
			CleanSession: true,
			ResumeSubs:   false,
			OrderMatters: true,
		},
		OnConnectCallback: func(client mqtt.Client) {
			if err := b.subscribe(client); err != nil {
				b.log.WithError(err).Error("bms bridge re-subscribe failed")
			}
		},
	}
	client, err := mqttadapter.CreateMQTTClient(mqttCfg, b.log)
	if err != nil {
		return err
	}
	b.client = client

	if err := b.subscribe(client); err != nil {
		return err
	}

	if len(b.queues) == 0 {
		b.queues = newIncomingShards(b.cfg.Workers.Concurrency, b.cfg.Workers.QueueSize)
	}
	for i, queue := range b.queues {
		b.wg.Add(1)
		go func(workerID int, incomingQueue <-chan incoming) {
			defer b.wg.Done()
			b.workerLoop(ctx, workerID, incomingQueue)
		}(i+1, queue)
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.cleanupLoop(ctx)
	}()

	b.log.WithFields(logrus.Fields{
		"subscribe_topic":      b.cfg.MQTT.SubscribeTopic,
		"workers":              len(b.queues),
		"queue_size_per_shard": b.cfg.Workers.QueueSize,
	}).Info("bms mqtt bridge started")
	return nil
}

func (b *Bridge) Stop() {
	if b.client != nil {
		_ = b.client.Unsubscribe(b.cfg.MQTT.SubscribeTopic)
		mqttadapter.DisconnectMQTTClient(b.client, b.log)
	}
}

func (b *Bridge) Wait() {
	b.wg.Wait()
}

func shouldIgnoreRetainedUplink(retained bool) bool {
	return retained
}

func newIncomingShards(concurrency, queueSize int) []chan incoming {
	if concurrency < 1 {
		concurrency = 1
	}
	if queueSize < 1 {
		queueSize = 1
	}
	shards := make([]chan incoming, concurrency)
	for i := range shards {
		// 每个分片保留原配置的缓冲深度，避免热点设备比改造前更早丢消息。
		shards[i] = make(chan incoming, queueSize)
	}
	return shards
}

func incomingShardIndex(deviceID string, shardCount int) int {
	if shardCount <= 1 {
		return 0
	}
	// FNV-1a keeps routing stable without allocating a hash object per uplink.
	hash := uint32(2166136261)
	for i := 0; i < len(deviceID); i++ {
		hash ^= uint32(deviceID[i])
		hash *= 16777619
	}
	return int(hash % uint32(shardCount))
}

func (b *Bridge) enqueueIncoming(msg incoming) bool {
	if len(b.queues) == 0 {
		return false
	}
	// raw topic identifier is stable even while DB identifier resolution changes.
	shardKey := strings.TrimSpace(msg.rawDeviceID)
	if shardKey == "" {
		shardKey = strings.TrimSpace(msg.deviceID)
	}
	queue := b.queues[incomingShardIndex(shardKey, len(b.queues))]
	select {
	case queue <- msg:
		return true
	default:
		return false
	}
}

func (b *Bridge) nextReceivedAt(deviceKey string, observed time.Time) time.Time {
	deviceKey = strings.TrimSpace(deviceKey)
	if deviceKey == "" {
		return observed
	}
	b.receivedAtMu.Lock()
	defer b.receivedAtMu.Unlock()
	if b.lastReceivedAtMs == nil {
		b.lastReceivedAtMs = make(map[string]int64)
	}
	nextMs := observed.UnixMilli()
	if lastMs := b.lastReceivedAtMs[deviceKey]; nextMs <= lastMs {
		nextMs = lastMs + 1
	}
	b.lastReceivedAtMs[deviceKey] = nextMs
	return time.UnixMilli(nextMs)
}

func (b *Bridge) subscribe(client mqtt.Client) error {
	token := client.Subscribe(b.cfg.MQTT.SubscribeTopic, b.cfg.MQTT.SubscribeQoS, func(_ mqtt.Client, m mqtt.Message) {
		rawDeviceID := extractDeviceIDFromTxTopic(m.Topic())
		if rawDeviceID == "" {
			return
		}
		deviceID := b.resolvePlatformDeviceID(context.Background(), rawDeviceID)
		msg := incoming{
			topic:       m.Topic(),
			rawDeviceID: rawDeviceID,
			deviceID:    deviceID,
			payload:     append([]byte(nil), m.Payload()...),
			qos:         m.Qos(),
			messageID:   fmt.Sprintf("%d", m.MessageID()),
			receivedAt:  time.Now(),
		}
		payloadRaw := string(msg.payload)
		payloadFormat := "json"
		if shouldIgnoreRetainedUplink(m.Retained()) {
			reason := "retained MQTT uplink ignored"
			b.traceCommDebug(context.Background(), commDebugTraceEntry{
				DeviceID:      deviceID,
				EventType:     commDebugEventUplinkIgnored,
				Direction:     commDebugDirectionInbound,
				MQTTTopic:     stringPtr(m.Topic()),
				QoS:           intPtr(int(msg.qos)),
				MessageID:     &msg.messageID,
				PayloadRaw:    &payloadRaw,
				PayloadFormat: &payloadFormat,
				ParsedSummary: map[string]any{"reason": "mqtt_retained"},
				Status:        commDebugStatusSuccess,
				ErrorMessage:  &reason,
				OccurredAt:    msg.receivedAt,
			})
			b.log.WithFields(logrus.Fields{
				"device_id":     deviceID,
				"raw_device":    rawDeviceID,
				"topic":         m.Topic(),
				"qos":           m.Qos(),
				"message_id":    msg.messageID,
				"mqtt_retained": true,
			}).Warn("bms bridge retained uplink ignored")
			return
		}
		b.traceCommDebug(context.Background(), commDebugTraceEntry{
			DeviceID:      deviceID,
			EventType:     commDebugEventUplinkRaw,
			Direction:     commDebugDirectionInbound,
			MQTTTopic:     stringPtr(m.Topic()),
			QoS:           intPtr(int(msg.qos)),
			MessageID:     &msg.messageID,
			PayloadRaw:    &payloadRaw,
			PayloadFormat: &payloadFormat,
			Status:        commDebugStatusSuccess,
			OccurredAt:    msg.receivedAt,
		})
		msg.receivedAt = b.nextReceivedAt(rawDeviceID, msg.receivedAt)
		if !b.enqueueIncoming(msg) {
			b.log.WithFields(logrus.Fields{
				"device_id":  deviceID,
				"raw_device": rawDeviceID,
			}).Warn("bms bridge queue full, dropping message")
		}
	})
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func extractDeviceIDFromTxTopic(topic string) string {
	// device/socket/tx/{device_id}
	parts := strings.Split(topic, "/")
	if len(parts) != 4 {
		return ""
	}
	if parts[0] != "device" || parts[1] != "socket" || parts[2] != "tx" {
		return ""
	}
	return parts[3]
}

func (b *Bridge) resolvePlatformDeviceID(ctx context.Context, identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || b.db == nil {
		return identifier
	}

	var row struct {
		DeviceID string `gorm:"column:device_id"`
	}
	err := b.db.WithContext(ctx).
		Table("devices AS d").
		Select("d.id AS device_id").
		Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
		Where(
			`d.id = ? OR d.device_number = ? OR dbat.item_uuid = ? OR dbat.comm_chip_id = ? OR dbat.imei = ? OR dbat.iccid = ?`,
			identifier, identifier, identifier, identifier, identifier, identifier,
		).
		Limit(1).
		Scan(&row).Error
	if err == nil && strings.TrimSpace(row.DeviceID) != "" {
		return row.DeviceID
	}

	return identifier
}

func (b *Bridge) workerLoop(ctx context.Context, workerID int, incomingQueue <-chan incoming) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-incomingQueue:
			if err := b.handleIncoming(ctx, msg); err != nil {
				b.log.WithFields(logrus.Fields{
					"worker":     workerID,
					"device_id":  msg.deviceID,
					"raw_device": msg.rawDeviceID,
					"topic":      msg.topic,
				}).WithError(err).Warn("bms bridge handle message failed")
			}
		}
	}
}

func (b *Bridge) cleanupLoop(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	lastCommCleanupAt := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			b.boolState.Cleanup(now)
			if b.db != nil && (lastCommCleanupAt.IsZero() || now.Sub(lastCommCleanupAt) >= time.Hour) {
				cutoff := now.Add(-7 * 24 * time.Hour)
				if err := b.db.WithContext(ctx).Table("bms_bridge_comm_logs").Where("occurred_at < ?", cutoff).Delete(nil).Error; err != nil {
					b.log.WithError(err).Warn("bms bridge comm debug cleanup failed")
				} else {
					lastCommCleanupAt = now
				}
			}
		}
	}
}

func (b *Bridge) decodeSocketHex(payload []byte) (string, error) {
	var body socketPayload
	if err := json.Unmarshal(payload, &body); err == nil && body.Hex != "" {
		return body.Hex, nil
	}
	s := strings.TrimSpace(string(payload))
	if s == "" {
		return "", fmt.Errorf("empty payload")
	}
	return s, nil
}

func (b *Bridge) debugEnabled() bool {
	return b.log != nil && b.log.IsLevelEnabled(logrus.DebugLevel)
}

func (b *Bridge) debugLogJSON(label, deviceID string, value any) {
	if !b.debugEnabled() {
		return
	}
	bs, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		b.log.WithField("device_id", deviceID).WithError(err).Debug("bms bridge json debug marshal failed")
		return
	}
	b.log.WithFields(logrus.Fields{
		"device_id": deviceID,
		"json":      string(bs),
	}).Debug(label)
}

func debugFrameValue(parsed protocol.ParsedFrame) any {
	switch f := parsed.(type) {
	case protocol.ReadFrame:
		return map[string]any{
			"type":          f.Type,
			"sourceAddress": f.SourceAddress,
			"targetAddress": f.TargetAddress,
			"functionCode":  f.FunctionCode,
			"byteCount":     f.ByteCount,
			"dataHex":       protocol.BytesToHexUpper(f.Data),
			"rawHex":        protocol.BytesToHexUpper(f.Raw),
			"dataLength":    len(f.Data),
			"rawLength":     len(f.Raw),
		}
	case protocol.WriteRequestFrame:
		return map[string]any{
			"type":          f.Type,
			"sourceAddress": f.SourceAddress,
			"targetAddress": f.TargetAddress,
			"functionCode":  f.FunctionCode,
			"startAddress":  f.StartAddress,
			"quantity":      f.Quantity,
			"byteCount":     f.ByteCount,
			"dataHex":       protocol.BytesToHexUpper(f.Data),
			"rawHex":        protocol.BytesToHexUpper(f.Raw),
			"dataLength":    len(f.Data),
			"rawLength":     len(f.Raw),
		}
	case protocol.ReadRequestFrame:
		return map[string]any{
			"type":          f.Type,
			"sourceAddress": f.SourceAddress,
			"targetAddress": f.TargetAddress,
			"functionCode":  f.FunctionCode,
			"startAddress":  f.StartAddress,
			"quantity":      f.Quantity,
			"rawHex":        protocol.BytesToHexUpper(f.Raw),
			"rawLength":     len(f.Raw),
		}
	case protocol.WriteFrame:
		return map[string]any{
			"type":          f.Type,
			"sourceAddress": f.SourceAddress,
			"targetAddress": f.TargetAddress,
			"functionCode":  f.FunctionCode,
			"startAddress":  f.StartAddress,
			"quantity":      f.Quantity,
			"rawHex":        protocol.BytesToHexUpper(f.Raw),
			"rawLength":     len(f.Raw),
		}
	case protocol.ErrorFrame:
		return map[string]any{
			"type":          f.Type,
			"sourceAddress": f.SourceAddress,
			"targetAddress": f.TargetAddress,
			"functionCode":  f.FunctionCode,
			"errorCode":     f.ErrorCode,
			"rawHex":        protocol.BytesToHexUpper(f.Raw),
			"rawLength":     len(f.Raw),
		}
	default:
		return parsed
	}
}

func extractReadFramePayload(f protocol.ReadFrame, reportFunctionCode byte, defaultStart uint16) (reportStart uint16, registers []uint16, ok bool, err error) {
	switch f.FunctionCode {
	case protocol.FuncSocketRead, protocol.FuncReadHoldingRegisters, protocol.FuncReadUUID:
		if len(f.Data) >= 4 {
			startAddress, quantity, rest, parseErr := protocol.ParseSocketReadPayload(f.Data)
			if parseErr == nil && quantity > 0 && int(quantity)*2 == len(rest) {
				regs, splitErr := protocol.SplitIntoRegistersBE(rest)
				if splitErr != nil {
					return 0, nil, false, splitErr
				}
				return startAddress, regs, true, nil
			}
		}
	}

	if f.FunctionCode != reportFunctionCode {
		return 0, nil, false, nil
	}

	regs, splitErr := protocol.SplitIntoRegistersBE(f.Data)
	if splitErr != nil {
		return 0, nil, false, splitErr
	}
	return defaultStart, regs, true, nil
}

func parseStatusRegistersCompatible(startAddress uint16, registers []uint16) (status.BmsStatus, error) {
	st, err := status.ParseStatusRegisters(startAddress, registers)
	if err == nil {
		return st, nil
	}
	if startAddress != 0x100 || len(registers) == 0 {
		return status.BmsStatus{}, err
	}

	seriesCount := int((registers[0] >> 8) & 0xFF)
	cellTempCount := int(registers[0] & 0xFF)
	requiredEnd := 0x175 + seriesCount + cellTempCount
	requiredLen := requiredEnd - int(startAddress) + 1
	if requiredLen <= len(registers) {
		return status.ParseStatusRegisters(startAddress, registers)
	}

	padded := make([]uint16, requiredLen)
	for i := range padded {
		padded[i] = 0xFFFF
	}
	copy(padded, registers)
	return status.ParseStatusRegisters(startAddress, padded)
}

func stripPaddedDynamicStatusFields(flat map[string]any, startAddress uint16, registers []uint16, st status.BmsStatus) {
	if startAddress != 0x100 || st.Meta.SeriesCount <= 0 {
		return
	}
	cellStartOffset := int(0x141 - startAddress)
	requiredLen := cellStartOffset + st.Meta.SeriesCount + st.Meta.CellTempCount
	if len(registers) >= requiredLen {
		return
	}
	for _, key := range []string{
		"cell.voltagesMv",
		"temperature.cellTempsC",
		"electrical.packCellSumVoltageV",
		"electrical.avgCellVoltageMv",
	} {
		delete(flat, key)
	}
}

func (b *Bridge) handleIncoming(ctx context.Context, msg incoming) error {
	b.log.WithFields(logrus.Fields{
		"topic":      msg.topic,
		"device_id":  msg.deviceID,
		"payloadLen": len(msg.payload),
	}).Debug("bms bridge received message")

	hexStr, err := b.decodeSocketHex(msg.payload)
	if err != nil {
		payloadRaw := string(msg.payload)
		payloadFormat := "json"
		errMsg := err.Error()
		b.traceCommDebug(ctx, commDebugTraceEntry{
			DeviceID:      msg.deviceID,
			EventType:     commDebugEventUplinkError,
			Direction:     commDebugDirectionInbound,
			MQTTTopic:     stringPtr(msg.topic),
			QoS:           intPtr(int(msg.qos)),
			MessageID:     &msg.messageID,
			PayloadRaw:    &payloadRaw,
			PayloadFormat: &payloadFormat,
			Status:        commDebugStatusError,
			ErrorMessage:  &errMsg,
			OccurredAt:    msg.receivedAt,
		})
		return err
	}
	b.log.WithFields(logrus.Fields{
		"device_id": msg.deviceID,
		"hexLen":    len(hexStr),
	}).Debug("bms bridge decoded hex payload")
	hexRaw := strings.ToUpper(strings.TrimSpace(hexStr))
	hexFormat := "hex"
	var bootSummary any
	if summary, ok := bootDebugSummaryFromHex(hexRaw); ok {
		bootSummary = summary
	}
	b.traceCommDebug(ctx, commDebugTraceEntry{
		DeviceID:      msg.deviceID,
		EventType:     commDebugEventUplinkDecoded,
		Direction:     commDebugDirectionInbound,
		MQTTTopic:     stringPtr(msg.topic),
		QoS:           intPtr(int(msg.qos)),
		MessageID:     &msg.messageID,
		PayloadRaw:    &hexRaw,
		PayloadFormat: &hexFormat,
		ParsedSummary: bootSummary,
		Status:        commDebugStatusSuccess,
		OccurredAt:    msg.receivedAt,
	})
	if b.debugEnabled() {
		b.log.WithFields(logrus.Fields{
			"device_id": msg.deviceID,
			"raw_hex":   strings.ToUpper(strings.TrimSpace(hexStr)),
		}).Debug("bms bridge raw hex payload")
	}

	frameBytes, err := protocol.DecodeHexString(hexStr)
	if err != nil {
		errMsg := err.Error()
		b.traceCommDebug(ctx, commDebugTraceEntry{
			DeviceID:      msg.deviceID,
			EventType:     commDebugEventUplinkError,
			Direction:     commDebugDirectionInbound,
			MQTTTopic:     stringPtr(msg.topic),
			QoS:           intPtr(int(msg.qos)),
			MessageID:     &msg.messageID,
			PayloadRaw:    &hexRaw,
			PayloadFormat: &hexFormat,
			ParsedSummary: bootSummary,
			Status:        commDebugStatusError,
			ErrorMessage:  &errMsg,
			OccurredAt:    msg.receivedAt,
		})
		return err
	}
	b.log.WithFields(logrus.Fields{
		"device_id": msg.deviceID,
		"frameLen":  len(frameBytes),
	}).Debug("bms bridge decoded frame bytes")

	parsed, err := protocol.ParseFrame(frameBytes)
	if err != nil {
		errMsg := err.Error()
		b.traceCommDebug(ctx, commDebugTraceEntry{
			DeviceID:      msg.deviceID,
			EventType:     commDebugEventUplinkError,
			Direction:     commDebugDirectionInbound,
			MQTTTopic:     stringPtr(msg.topic),
			QoS:           intPtr(int(msg.qos)),
			MessageID:     &msg.messageID,
			PayloadRaw:    &hexRaw,
			PayloadFormat: &hexFormat,
			ParsedSummary: bootSummary,
			Status:        commDebugStatusError,
			ErrorMessage:  &errMsg,
			OccurredAt:    msg.receivedAt,
		})
		return err
	}
	frameSummary := debugFrameValue(parsed)
	b.traceCommDebug(ctx, commDebugTraceEntry{
		DeviceID:      msg.deviceID,
		EventType:     commDebugEventUplinkParsed,
		Direction:     commDebugDirectionInbound,
		MQTTTopic:     stringPtr(msg.topic),
		QoS:           intPtr(int(msg.qos)),
		MessageID:     &msg.messageID,
		PayloadRaw:    &hexRaw,
		PayloadFormat: &hexFormat,
		ParsedSummary: frameSummary,
		Status:        commDebugStatusSuccess,
		OccurredAt:    msg.receivedAt,
	})
	b.debugLogJSON("bms bridge parsed frame json", msg.deviceID, debugFrameValue(parsed))

	switch f := parsed.(type) {
	case protocol.ReadFrame:
		b.log.WithFields(logrus.Fields{
			"device_id":     msg.deviceID,
			"type":          string(f.Type),
			"func":          fmt.Sprintf("0x%02X", f.FunctionCode),
			"sourceAddress": fmt.Sprintf("0x%02X", f.SourceAddress),
			"targetAddress": fmt.Sprintf("0x%02X", f.TargetAddress),
			"byteCount":     f.ByteCount,
		}).Debug("bms bridge parsed read frame")

		reportStart, registers, ok, err := extractReadFramePayload(f, b.cfg.Report.FunctionCode, b.cfg.Report.StatusStartAddress)
		if err != nil {
			return err
		}
		if !ok {
			// Generic 0x03 response does not carry enough addressing info unless the payload
			// embeds [startAddress, quantity, data...]. If it doesn't, we skip semantic decode.
			return nil
		}

		flat := make(map[string]any, 256)
		flat["report.startAddress"] = int(reportStart)
		flat["report.quantity"] = len(registers)
		flat = merge(flat, flattenRegisters(reportStart, registers))
		if extra := decodeSocketRegisters(reportStart, registers); extra != nil {
			flat = merge(flat, extra)
		}

		// If it's a status block report, decode semantic status and merge.
		if reportStart == 0x100 {
			if err := status.EnsureStatusRangeLooksValid(reportStart, registers); err != nil {
				b.log.WithField("device_id", msg.deviceID).WithError(err).Debug("status range sanity check failed, trying compatible parse")
			}
			st, err := parseStatusRegistersCompatible(reportStart, registers)
			if err != nil {
				return err
			}
			b.debugLogJSON("bms bridge parsed status json", msg.deviceID, st)
			b.rememberStatusMeta(msg.deviceID, st)
			flat = merge(flat, FlattenStatus(st))
			stripPaddedDynamicStatusFields(flat, reportStart, registers, st)
		} else if reportStart == 0x141 {
			if extra := b.decodeDynamicCellReport(ctx, msg.deviceID, reportStart, registers); extra != nil {
				flat = merge(flat, extra)
			}
		}
		b.debugLogJSON("bms bridge parsed payload json", msg.deviceID, flat)

		return b.applyRules(ctx, msg.deviceID, flat, msg.receivedAt)

	case protocol.WriteRequestFrame:
		b.log.WithFields(logrus.Fields{
			"device_id":     msg.deviceID,
			"type":          string(f.Type),
			"func":          fmt.Sprintf("0x%02X", f.FunctionCode),
			"sourceAddress": fmt.Sprintf("0x%02X", f.SourceAddress),
			"targetAddress": fmt.Sprintf("0x%02X", f.TargetAddress),
			"startAddress":  fmt.Sprintf("0x%04X", f.StartAddress),
			"quantity":      f.Quantity,
			"byteCount":     f.ByteCount,
		}).Debug("bms bridge parsed write-request frame")

		if f.FunctionCode != b.cfg.Report.FunctionCode {
			return nil
		}
		registers, err := protocol.SplitIntoRegistersBE(f.Data)
		if err != nil {
			return err
		}
		reportStart := f.StartAddress

		flat := make(map[string]any, 256)
		flat["report.startAddress"] = int(reportStart)
		flat["report.quantity"] = len(registers)
		flat = merge(flat, flattenRegisters(reportStart, registers))
		if extra := decodeSocketRegisters(reportStart, registers); extra != nil {
			flat = merge(flat, extra)
		}

		if reportStart == 0x100 {
			if err := status.EnsureStatusRangeLooksValid(reportStart, registers); err != nil {
				b.log.WithField("device_id", msg.deviceID).WithError(err).Debug("status range sanity check failed, trying compatible parse")
			}
			st, err := parseStatusRegistersCompatible(reportStart, registers)
			if err != nil {
				return err
			}
			b.debugLogJSON("bms bridge parsed status json", msg.deviceID, st)
			b.rememberStatusMeta(msg.deviceID, st)
			flat = merge(flat, FlattenStatus(st))
			stripPaddedDynamicStatusFields(flat, reportStart, registers, st)
		} else if reportStart == 0x141 {
			if extra := b.decodeDynamicCellReport(ctx, msg.deviceID, reportStart, registers); extra != nil {
				flat = merge(flat, extra)
			}
		}
		b.debugLogJSON("bms bridge parsed payload json", msg.deviceID, flat)

		return b.applyRules(ctx, msg.deviceID, flat, msg.receivedAt)

	default:
		// Other frames are not handled yet (passthrough rules are TODO).
		return nil
	}
}

func (b *Bridge) applyRules(ctx context.Context, deviceID string, flat map[string]any, receivedAt time.Time) error {
	ruleSet, err := b.rules.Get(deviceID)
	if err != nil {
		return err
	}

	if values := selectValues(flat, ruleSet.Telemetry); values != nil {
		b.log.WithFields(logrus.Fields{"device_id": deviceID, "keys": len(values)}).Debug("publishing telemetry")
		if err := b.publishTelemetry(deviceID, values, receivedAt); err != nil {
			return err
		}
	}

	if values := selectValues(flat, ruleSet.Attributes); values != nil {
		b.log.WithFields(logrus.Fields{"device_id": deviceID, "keys": len(values)}).Debug("publishing attributes")
		if err := b.publishAttributes(deviceID, values); err != nil {
			return err
		}
	}

	if ruleSet.Events.Enabled {
		if err := b.emitEventsFromStatusChange(deviceID, flat, ruleSet.Events); err != nil {
			return err
		}
	}

	if ruleSet.DBSync.Enabled {
		if err := b.syncDeviceBatteries(ctx, deviceID, flat, ruleSet.DBSync.DeviceBatteries); err != nil {
			return err
		}
	}

	return nil
}

func flattenRegisters(startAddress uint16, regs []uint16) map[string]any {
	out := make(map[string]any, len(regs))
	for i := 0; i < len(regs); i++ {
		addr := startAddress + uint16(i)
		out[fmt.Sprintf("reg.0x%04X", addr)] = int(regs[i])
	}
	return out
}

func merge(dst, src map[string]any) map[string]any {
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func newTelemetryPayload(deviceID string, values map[string]any, receivedAt time.Time) map[string]any {
	return map[string]any{
		"device_id": deviceID,
		"source":    "bms_bridge",
		"timestamp": receivedAt.UnixMilli(),
		"values":    values,
	}
}

func (b *Bridge) publishTelemetry(deviceID string, values map[string]any, receivedAt time.Time) error {
	payload := newTelemetryPayload(deviceID, values, receivedAt)
	return b.publishJSON(deviceID, commDebugEventDownlinkPub, b.cfg.MQTT.TelemetryTopic, b.cfg.MQTT.TelemetryQoS, nil, payload)
}

func (b *Bridge) publishAttributes(deviceID string, values map[string]any) error {
	messageID := makeMessageID()
	topic := b.cfg.MQTT.AttributesTopicPrefix + messageID
	payload := map[string]any{
		"device_id": deviceID,
		"values":    values,
	}
	return b.publishJSON(deviceID, commDebugEventDownlinkPub, topic, b.cfg.MQTT.AttributesQoS, &messageID, payload)
}

func (b *Bridge) emitEventsFromStatusChange(deviceID string, flat map[string]any, rules EventRules) error {
	if !rules.EmitOnChange {
		return nil
	}
	deltas := b.boolState.DiffAndSet(deviceID, flat, rules.TrackKeyPrefixes)
	for _, d := range deltas {
		if d.OldValue == d.NewValue {
			continue
		}
		method := rules.MethodPrefix + d.Key
		messageID := makeMessageID()
		topic := b.cfg.MQTT.EventsTopicPrefix + messageID

		payload := map[string]any{
			"device_id": deviceID,
			"values": map[string]any{
				"method": method,
				"params": map[string]any{
					"key":    d.Key,
					"active": d.NewValue,
					"ts":     time.Now().Unix(),
				},
			},
		}
		if err := b.publishJSON(deviceID, commDebugEventDownlinkPub, topic, b.cfg.MQTT.EventsQoS, &messageID, payload); err != nil {
			return err
		}
	}
	return nil
}

func makeMessageID() string {
	// milliseconds timestamp last 7 digits
	ms := time.Now().UnixMilli()
	return fmt.Sprintf("%07d", ms%10000000)
}

func (b *Bridge) publishJSON(deviceID, eventType, topic string, qos byte, messageID *string, payload any) error {
	bs, err := json.Marshal(payload)
	if err != nil {
		errMsg := err.Error()
		payloadRaw := string(bs)
		payloadFormat := "json"
		b.traceCommDebug(context.Background(), commDebugTraceEntry{
			DeviceID:      deviceID,
			EventType:     commDebugEventDownlinkError,
			Direction:     commDebugDirectionOutbound,
			MQTTTopic:     stringPtr(topic),
			QoS:           intPtr(int(qos)),
			MessageID:     messageID,
			PayloadRaw:    &payloadRaw,
			PayloadFormat: &payloadFormat,
			Status:        commDebugStatusError,
			ErrorMessage:  &errMsg,
			OccurredAt:    time.Now(),
		})
		return err
	}
	payloadRaw := string(bs)
	payloadFormat := "json"
	now := time.Now()
	b.traceCommDebug(context.Background(), commDebugTraceEntry{
		DeviceID:      deviceID,
		EventType:     eventType,
		Direction:     commDebugDirectionOutbound,
		MQTTTopic:     stringPtr(topic),
		QoS:           intPtr(int(qos)),
		MessageID:     messageID,
		PayloadRaw:    &payloadRaw,
		PayloadFormat: &payloadFormat,
		ParsedSummary: payload,
		Status:        commDebugStatusSuccess,
		OccurredAt:    now,
	})
	token := b.client.Publish(topic, qos, false, bs)
	timeout := time.Duration(b.cfg.MQTT.PublishTimeoutMs) * time.Millisecond
	if !token.WaitTimeout(timeout) {
		err := fmt.Errorf("mqtt publish timeout: topic=%s", topic)
		errMsg := err.Error()
		b.traceCommDebug(context.Background(), commDebugTraceEntry{
			DeviceID:      deviceID,
			EventType:     commDebugEventDownlinkError,
			Direction:     commDebugDirectionOutbound,
			MQTTTopic:     stringPtr(topic),
			QoS:           intPtr(int(qos)),
			MessageID:     messageID,
			PayloadRaw:    &payloadRaw,
			PayloadFormat: &payloadFormat,
			Status:        commDebugStatusError,
			ErrorMessage:  &errMsg,
			OccurredAt:    time.Now(),
		})
		return err
	}
	if err := token.Error(); err != nil {
		errMsg := err.Error()
		b.traceCommDebug(context.Background(), commDebugTraceEntry{
			DeviceID:      deviceID,
			EventType:     commDebugEventDownlinkError,
			Direction:     commDebugDirectionOutbound,
			MQTTTopic:     stringPtr(topic),
			QoS:           intPtr(int(qos)),
			MessageID:     messageID,
			PayloadRaw:    &payloadRaw,
			PayloadFormat: &payloadFormat,
			Status:        commDebugStatusError,
			ErrorMessage:  &errMsg,
			OccurredAt:    time.Now(),
		})
		return err
	}
	return nil
}

func stringPtr(v string) *string {
	return &v
}

func intPtr(v int) *int {
	return &v
}

var allowedDeviceBatteryColumns = map[string]struct{}{
	"soc":               {},
	"soh":               {},
	"ble_mac":           {},
	"identity_ble_mac":  {},
	"comm_chip_id":      {},
	"longitude":         {},
	"latitude":          {},
	"speed":             {},
	"altitude":          {},
	"rssi":              {},
	"tac":               {},
	"cell_id":           {},
	"imei":              {},
	"iccid":             {},
	"module_sw_version": {},
}

func collectDeviceBatterySyncValues(flat map[string]any, mapping map[string]string) map[string]any {
	values := make(map[string]any, len(mapping))
	for col, flatKey := range mapping {
		if _, ok := allowedDeviceBatteryColumns[col]; !ok {
			continue
		}
		v, ok := flat[flatKey]
		if !ok || v == nil {
			continue
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			continue
		}
		values[col] = v
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func (b *Bridge) syncDeviceBatteries(ctx context.Context, deviceID string, flat map[string]any, mapping map[string]string) error {
	if b.db == nil {
		return nil
	}
	values := collectDeviceBatterySyncValues(flat, mapping)
	if len(values) == 0 {
		return nil
	}

	cols := make([]string, 0, len(values))
	args := make([]any, 0, len(values)+1)
	args = append(args, deviceID)
	placeholders := make([]string, 0, len(values))

	i := 0
	for col, v := range values {
		cols = append(cols, col)
		args = append(args, v)
		placeholders = append(placeholders, "?")
		i++
	}

	// INSERT ... ON CONFLICT (device_id) DO UPDATE ...
	insertCols := append([]string{"device_id"}, cols...)
	insertPlaceholders := append([]string{"?"}, placeholders...)

	setParts := make([]string, 0, len(cols)+1)
	for _, col := range cols {
		setParts = append(setParts, fmt.Sprintf("%s=EXCLUDED.%s", col, col))
	}
	setParts = append(setParts, "updated_at=NOW()")

	sql := fmt.Sprintf(
		"INSERT INTO device_batteries (%s, updated_at) VALUES (%s, NOW()) "+
			"ON CONFLICT (device_id) DO UPDATE SET %s",
		strings.Join(insertCols, ", "),
		strings.Join(insertPlaceholders, ", "),
		strings.Join(setParts, ", "),
	)

	return b.db.WithContext(ctx).Exec(sql, args...).Error
}
