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

	queue chan incoming
	wg    sync.WaitGroup

	boolState *BoolStateStore
}

type incoming struct {
	topic      string
	deviceID   string
	payload    []byte
	receivedAt time.Time
}

type socketPayload struct {
	Hex string `json:"hex"`
}

func New(cfg Config, db *gorm.DB, log *logrus.Logger) *Bridge {
	if log == nil {
		log = logrus.StandardLogger()
	}
	return &Bridge{
		cfg:       cfg,
		db:        db,
		log:       log,
		rules:     NewFileRulesProvider(cfg.Rules.Path, cfg.Rules.ReloadIntervalSec),
		queue:     make(chan incoming, cfg.Workers.QueueSize),
		boolState: NewBoolStateStore(time.Duration(cfg.Events.StateTTLMinutes) * time.Minute),
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

	for i := 0; i < b.cfg.Workers.Concurrency; i++ {
		b.wg.Add(1)
		go func(workerID int) {
			defer b.wg.Done()
			b.workerLoop(ctx, workerID)
		}(i + 1)
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.cleanupLoop(ctx)
	}()

	b.log.WithFields(logrus.Fields{
		"subscribe_topic": b.cfg.MQTT.SubscribeTopic,
		"workers":         b.cfg.Workers.Concurrency,
		"queue_size":      b.cfg.Workers.QueueSize,
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

func (b *Bridge) subscribe(client mqtt.Client) error {
	token := client.Subscribe(b.cfg.MQTT.SubscribeTopic, b.cfg.MQTT.SubscribeQoS, func(_ mqtt.Client, m mqtt.Message) {
		deviceID := extractDeviceIDFromTxTopic(m.Topic())
		if deviceID == "" {
			return
		}
		msg := incoming{
			topic:      m.Topic(),
			deviceID:   deviceID,
			payload:    append([]byte(nil), m.Payload()...),
			receivedAt: time.Now(),
		}
		select {
		case b.queue <- msg:
		default:
			b.log.WithField("device_id", deviceID).Warn("bms bridge queue full, dropping message")
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

func (b *Bridge) workerLoop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-b.queue:
			if err := b.handleIncoming(ctx, msg); err != nil {
				b.log.WithFields(logrus.Fields{
					"worker":    workerID,
					"device_id": msg.deviceID,
					"topic":     msg.topic,
				}).WithError(err).Warn("bms bridge handle message failed")
			}
		}
	}
}

func (b *Bridge) cleanupLoop(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			b.boolState.Cleanup(now)
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

func (b *Bridge) handleIncoming(ctx context.Context, msg incoming) error {
	b.log.WithFields(logrus.Fields{
		"topic":      msg.topic,
		"device_id":  msg.deviceID,
		"payloadLen": len(msg.payload),
	}).Debug("bms bridge received message")

	hexStr, err := b.decodeSocketHex(msg.payload)
	if err != nil {
		return err
	}
	b.log.WithFields(logrus.Fields{
		"device_id": msg.deviceID,
		"hexLen":    len(hexStr),
	}).Debug("bms bridge decoded hex payload")

	frameBytes, err := protocol.DecodeHexString(hexStr)
	if err != nil {
		return err
	}
	b.log.WithFields(logrus.Fields{
		"device_id": msg.deviceID,
		"frameLen":  len(frameBytes),
	}).Debug("bms bridge decoded frame bytes")

	parsed, err := protocol.ParseFrame(frameBytes)
	if err != nil {
		return err
	}

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

		if f.FunctionCode != b.cfg.Report.FunctionCode {
			// Ignore normal request/response traffic; focus on report frames.
			return nil
		}

		registers, err := protocol.SplitIntoRegistersBE(f.Data)
		if err != nil {
			return err
		}

		reportStart := b.cfg.Report.StatusStartAddress
		flat := make(map[string]any, 256)
		flat["report.startAddress"] = int(reportStart)
		flat["report.quantity"] = len(registers)
		flat = merge(flat, flattenRegisters(reportStart, registers))

		// If it's a status block report, decode semantic status and merge.
		if reportStart == 0x100 {
			if err := status.EnsureStatusRangeLooksValid(reportStart, registers); err != nil {
				b.log.WithField("device_id", msg.deviceID).WithError(err).Debug("status range sanity check failed")
			} else {
				st, err := status.ParseStatusRegisters(reportStart, registers)
				if err != nil {
					return err
				}
				flat = merge(flat, FlattenStatus(st))
			}
		}

		return b.applyRules(ctx, msg.deviceID, flat)

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

		if reportStart == 0x100 {
			if err := status.EnsureStatusRangeLooksValid(reportStart, registers); err != nil {
				b.log.WithField("device_id", msg.deviceID).WithError(err).Debug("status range sanity check failed")
			} else {
				st, err := status.ParseStatusRegisters(reportStart, registers)
				if err != nil {
					return err
				}
				flat = merge(flat, FlattenStatus(st))
			}
		}

		return b.applyRules(ctx, msg.deviceID, flat)

	default:
		// Other frames are not handled yet (passthrough rules are TODO).
		return nil
	}
}

func (b *Bridge) applyRules(ctx context.Context, deviceID string, flat map[string]any) error {
	ruleSet, err := b.rules.Get(deviceID)
	if err != nil {
		return err
	}

	if values := selectValues(flat, ruleSet.Telemetry); values != nil {
		b.log.WithFields(logrus.Fields{"device_id": deviceID, "keys": len(values)}).Debug("publishing telemetry")
		if err := b.publishTelemetry(deviceID, values); err != nil {
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

func (b *Bridge) publishTelemetry(deviceID string, values map[string]any) error {
	payload := map[string]any{
		"device_id": deviceID,
		"values":    values,
	}
	return b.publishJSON(b.cfg.MQTT.TelemetryTopic, b.cfg.MQTT.TelemetryQoS, payload)
}

func (b *Bridge) publishAttributes(deviceID string, values map[string]any) error {
	messageID := makeMessageID()
	topic := b.cfg.MQTT.AttributesTopicPrefix + messageID
	payload := map[string]any{
		"device_id": deviceID,
		"values":    values,
	}
	return b.publishJSON(topic, b.cfg.MQTT.AttributesQoS, payload)
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
		if err := b.publishJSON(topic, b.cfg.MQTT.EventsQoS, payload); err != nil {
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

func (b *Bridge) publishJSON(topic string, qos byte, payload any) error {
	bs, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	token := b.client.Publish(topic, qos, false, bs)
	timeout := time.Duration(b.cfg.MQTT.PublishTimeoutMs) * time.Millisecond
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("mqtt publish timeout: topic=%s", topic)
	}
	return token.Error()
}

var allowedDeviceBatteryColumns = map[string]struct{}{
	"soc":          {},
	"soh":          {},
	"ble_mac":      {},
	"item_uuid":    {},
	"comm_chip_id": {},
}

func (b *Bridge) syncDeviceBatteries(ctx context.Context, deviceID string, flat map[string]any, mapping map[string]string) error {
	if b.db == nil {
		return nil
	}
	values := make(map[string]any, len(mapping))
	for col, flatKey := range mapping {
		if _, ok := allowedDeviceBatteryColumns[col]; !ok {
			continue
		}
		v, ok := flat[flatKey]
		if !ok || v == nil {
			continue
		}
		values[col] = v
	}
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
