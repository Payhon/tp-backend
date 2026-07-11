package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/errcode"
	"project/pkg/utils"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// AppBatteryApi APP端：电池设备详情/透传
type AppBatteryApi struct{}

const (
	socketBootSlowAckThreshold        = 2 * time.Second
	socketOwnerDataRefreshMinInterval = 5 * time.Second
	socketOnlineRefreshMinInterval    = 30 * time.Second
)

type socketBootFrameLogInfo struct {
	CleanHex string

	Source       byte
	Target       byte
	Cmd          byte
	DataLen      int
	PayloadBytes int

	IsPacket        bool
	PacketIndex     int
	PacketDataBytes int

	IsAck        bool
	AckStatus    int
	AckRequested int

	FirmwareSize int
	PrepareBaud  int
}

type socketBootSessionTrace struct {
	mu sync.Mutex

	hasLastDownlinkPacket bool
	lastDownlinkPacket    int
	lastDownlinkAt        time.Time
	lastDownlinkRetry     int

	hasLastAck       bool
	lastAckRequested int
	lastAckAt        time.Time
}

func shouldIgnoreSocketUplink(retained bool) bool {
	return retained
}

func websocketInitString(msg map[string]interface{}, key string) string {
	val, ok := msg[key]
	if !ok || val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(val))
}

func websocketInitHasFeature(msg map[string]interface{}, feature string) bool {
	feature = strings.TrimSpace(feature)
	if feature == "" {
		return false
	}
	val, ok := msg["features"]
	if !ok || val == nil {
		return false
	}
	switch items := val.(type) {
	case []interface{}:
		for _, item := range items {
			if strings.TrimSpace(fmt.Sprint(item)) == feature {
				return true
			}
		}
	case []string:
		for _, item := range items {
			if strings.TrimSpace(item) == feature {
				return true
			}
		}
	case string:
		for _, item := range strings.Split(items, ",") {
			if strings.TrimSpace(item) == feature {
				return true
			}
		}
	}
	return false
}

func socketPayloadHexForLog(payload []byte) string {
	var body struct {
		Hex string `json:"hex"`
	}
	if err := json.Unmarshal(payload, &body); err == nil && strings.TrimSpace(body.Hex) != "" {
		return body.Hex
	}
	return string(payload)
}

func cleanSocketHexForLog(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'a' && r <= 'f':
			b.WriteRune(r - 'a' + 'A')
		case r >= 'A' && r <= 'F':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func socketHexPrefixForLog(hexText string, maxLen int) string {
	if maxLen <= 0 || len(hexText) <= maxLen {
		return hexText
	}
	return hexText[:maxLen] + "..."
}

func socketFrameLogFields(payload []byte) logrus.Fields {
	rawHex := socketPayloadHexForLog(payload)
	cleanHex := cleanSocketHexForLog(rawHex)
	fields := logrus.Fields{
		"payload_bytes": len(payload),
	}
	if cleanHex == "" {
		return fields
	}
	fields["payload_hex_len"] = len(cleanHex)
	fields["payload_hex_prefix"] = socketHexPrefixForLog(cleanHex, 96)
	frame, err := hex.DecodeString(cleanHex)
	if err != nil {
		return fields
	}
	fields["frame_bytes"] = len(frame)
	if len(frame) < 2 || frame[0] != 0x7F || frame[1] != 0x55 {
		return fields
	}
	if len(frame) >= 5 {
		fields["frame_source"] = fmt.Sprintf("0x%02X", frame[2]&0xff)
		fields["frame_target"] = fmt.Sprintf("0x%02X", frame[3]&0xff)
		fields["frame_function"] = fmt.Sprintf("0x%02X", frame[4]&0xff)
	}
	if len(frame) == 12 && len(frame) >= 9 {
		fields["frame_start_address"] = fmt.Sprintf("0x%04X", (int(frame[5]&0xff)<<8)|int(frame[6]&0xff))
		fields["frame_quantity"] = (int(frame[7]&0xff) << 8) | int(frame[8]&0xff)
	} else if len(frame) >= 6 {
		fields["frame_byte_count"] = int(frame[5] & 0xff)
	}
	return fields
}

func socketBootSeqHex(seq int) string {
	return fmt.Sprintf("0x%04X", seq&0xffff)
}

func socketBootFrameForLog(payload []byte) (socketBootFrameLogInfo, bool) {
	rawHex := socketPayloadHexForLog(payload)
	cleanHex := cleanSocketHexForLog(rawHex)
	if cleanHex == "" || len(cleanHex)%2 != 0 {
		return socketBootFrameLogInfo{}, false
	}
	frame, err := hex.DecodeString(cleanHex)
	if err != nil || len(frame) < 6 || frame[0] != 0x55 {
		return socketBootFrameLogInfo{}, false
	}
	cmd := frame[3] & 0xff
	dataLen := (int(frame[4]&0xff) << 8) | int(frame[5]&0xff)
	info := socketBootFrameLogInfo{
		CleanHex:     cleanHex,
		Source:       frame[1],
		Target:       frame[2],
		Cmd:          cmd,
		DataLen:      dataLen,
		PayloadBytes: len(frame),
	}
	if cmd == 0x53 && dataLen >= 2 && len(frame) >= 8 {
		if dataLen == 3 && len(frame) >= 9 {
			info.IsAck = true
			info.AckStatus = int(frame[6] & 0xff)
			info.AckRequested = (int(frame[7]&0xff) << 8) | int(frame[8]&0xff)
		} else {
			info.IsPacket = true
			info.PacketIndex = (int(frame[6]&0xff) << 8) | int(frame[7]&0xff)
			info.PacketDataBytes = dataLen - 2
		}
	}
	if cmd == 0x52 && dataLen >= 8 && len(frame) >= 14 {
		info.FirmwareSize = (int(frame[6]&0xff) << 24) | (int(frame[7]&0xff) << 16) | (int(frame[8]&0xff) << 8) | int(frame[9]&0xff)
		info.PrepareBaud = (int(frame[10]&0xff) << 24) | (int(frame[11]&0xff) << 16) | (int(frame[12]&0xff) << 8) | int(frame[13]&0xff)
	}
	return info, true
}

func socketBootLogFields(payload []byte) (logrus.Fields, bool) {
	info, ok := socketBootFrameForLog(payload)
	if !ok {
		return nil, false
	}
	return info.logFields(), true
}

func (info socketBootFrameLogInfo) logFields() logrus.Fields {
	fields := logrus.Fields{
		"boot_cmd":           fmt.Sprintf("0x%02X", info.Cmd),
		"boot_source":        fmt.Sprintf("0x%02X", info.Source&0xff),
		"boot_target":        fmt.Sprintf("0x%02X", info.Target&0xff),
		"boot_data_len":      info.DataLen,
		"payload_bytes":      info.PayloadBytes,
		"payload_hex_prefix": socketHexPrefixForLog(info.CleanHex, 96),
	}
	if info.IsAck {
		fields["boot_ack_status"] = info.AckStatus
		fields["boot_ack_requested"] = info.AckRequested
		fields["boot_ack_requested_seq"] = info.AckRequested
		fields["boot_ack_requested_hex"] = socketBootSeqHex(info.AckRequested)
		if info.AckRequested > 0 {
			fields["boot_ack_for_packet_seq"] = info.AckRequested - 1
			fields["boot_ack_for_packet_hex"] = socketBootSeqHex(info.AckRequested - 1)
		}
	}
	if info.IsPacket {
		fields["boot_packet_index"] = info.PacketIndex
		fields["boot_packet_seq"] = info.PacketIndex
		fields["boot_packet_seq_hex"] = socketBootSeqHex(info.PacketIndex)
		fields["boot_expected_ack_seq"] = info.PacketIndex + 1
		fields["boot_expected_ack_hex"] = socketBootSeqHex(info.PacketIndex + 1)
		fields["boot_packet_data_bytes"] = info.PacketDataBytes
	}
	if info.Cmd == 0x52 && info.DataLen >= 8 {
		fields["boot_firmware_size"] = info.FirmwareSize
		fields["boot_prepare_baud"] = info.PrepareBaud
	}
	return fields
}

func (info socketBootFrameLogInfo) isBootOTACommand() bool {
	switch info.Cmd {
	case 0x50, 0x51, 0x52, 0x53, 0x54:
		return true
	default:
		return false
	}
}

func (trace *socketBootSessionTrace) observeDownlink(fields logrus.Fields, info socketBootFrameLogInfo, at time.Time) bool {
	if !info.IsPacket {
		return false
	}

	trace.mu.Lock()
	defer trace.mu.Unlock()

	slowAfterAck := false
	if trace.hasLastDownlinkPacket {
		fields["boot_packet_since_prev_downlink_ms"] = at.Sub(trace.lastDownlinkAt).Milliseconds()
		fields["boot_packet_index_delta"] = info.PacketIndex - trace.lastDownlinkPacket
		if info.PacketIndex == trace.lastDownlinkPacket {
			trace.lastDownlinkRetry++
			fields["boot_packet_retry"] = true
			fields["boot_packet_retry_count"] = trace.lastDownlinkRetry
		} else {
			trace.lastDownlinkRetry = 0
		}
	}
	fields["boot_packet_attempt"] = trace.lastDownlinkRetry + 1
	if trace.hasLastAck && trace.lastAckRequested == info.PacketIndex {
		afterAckMs := at.Sub(trace.lastAckAt).Milliseconds()
		fields["boot_last_ack_requested_seq"] = trace.lastAckRequested
		fields["boot_last_ack_requested_hex"] = socketBootSeqHex(trace.lastAckRequested)
		fields["boot_packet_after_ack_ms"] = afterAckMs
		slowAfterAck = afterAckMs >= socketBootSlowAckThreshold.Milliseconds()
	}

	trace.hasLastDownlinkPacket = true
	trace.lastDownlinkPacket = info.PacketIndex
	trace.lastDownlinkAt = at
	return slowAfterAck
}

func (trace *socketBootSessionTrace) observeAck(fields logrus.Fields, info socketBootFrameLogInfo, at time.Time) bool {
	if !info.IsAck {
		return false
	}

	trace.mu.Lock()
	defer trace.mu.Unlock()

	slowAck := false
	if trace.hasLastAck {
		ackGapMs := at.Sub(trace.lastAckAt).Milliseconds()
		fields["boot_ack_gap_ms"] = ackGapMs
		fields["boot_ack_requested_delta"] = info.AckRequested - trace.lastAckRequested
		if info.AckRequested <= trace.lastAckRequested {
			fields["boot_ack_non_progressing"] = true
		}
		if ackGapMs >= socketBootSlowAckThreshold.Milliseconds() {
			slowAck = true
		}
	}
	if trace.hasLastDownlinkPacket && trace.lastDownlinkPacket+1 == info.AckRequested {
		afterDownlinkMs := at.Sub(trace.lastDownlinkAt).Milliseconds()
		fields["boot_last_downlink_packet_seq"] = trace.lastDownlinkPacket
		fields["boot_last_downlink_packet_hex"] = socketBootSeqHex(trace.lastDownlinkPacket)
		fields["boot_ack_after_downlink_ms"] = afterDownlinkMs
		if afterDownlinkMs >= socketBootSlowAckThreshold.Milliseconds() {
			slowAck = true
		}
	}

	trace.hasLastAck = true
	trace.lastAckRequested = info.AckRequested
	trace.lastAckAt = at
	return slowAck
}

// GetBatteryDetail 获取APP端电池设备详情
// @Summary 获取电池设备详情(APP)
// @Description APP端设备详情页使用：从 devices + device_batteries 查询基础信息（含 ble_mac/item_uuid/comm_chip_id）
// @Tags APP-Battery
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID(UUID)"
// @Success 200 {object} model.AppBatteryDetailResp
// @Router /api/v1/app/battery/detail/{device_id} [get]
func (*AppBatteryApi) GetBatteryDetail(c *gin.Context) {
	deviceID := c.Param("device_id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	data, err := service.GroupApp.AppBattery.GetBatteryDetailForApp(context.Background(), deviceID, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBatteryCurrentTelemetry 获取APP端电池当前遥测
// @Summary 获取电池当前遥测(APP)
// @Description APP端4G设备详情页使用：复用绑定/组织权限校验后读取当前遥测与bms.snapshot
// @Tags APP-Battery
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID(UUID)"
// @Success 200 {object} model.AppBatteryCurrentTelemetryResp
// @Router /api/v1/app/battery/current-telemetry/{device_id} [get]
func (*AppBatteryApi) GetBatteryCurrentTelemetry(c *gin.Context) {
	deviceID := c.Param("device_id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	data, err := service.GroupApp.AppBattery.GetBatteryCurrentTelemetryForApp(context.Background(), deviceID, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBatteryMqttCredential 获取APP端直连MQTT凭证
// @Summary 获取APP端直连MQTT凭证
// @Description APP端直连 EMQX WebSocket 使用
// @Tags APP-Battery
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID(UUID)"
// @Success 200 {object} model.AppBatteryMqttCredentialResp
// @Router /api/v1/app/battery/mqtt-credential/{device_id} [get]
func (*AppBatteryApi) GetBatteryMqttCredential(c *gin.Context) {
	deviceID := c.Param("device_id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	data, err := service.GroupApp.AppBattery.GetBatteryMqttCredentialForApp(context.Background(), deviceID, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// CheckBatteryOta APP端OTA升级检查
// @Summary APP端OTA升级检查
// @Description 返回是否需要升级以及固件下载地址
// @Tags APP-Battery
// @Accept json
// @Produce json
// @Param body body model.AppBatteryOtaCheckReq true "检查请求"
// @Success 200 {object} model.AppBatteryOtaCheckResp
// @Router /api/v1/app/battery/ota/check [post]
func (*AppBatteryApi) CheckBatteryOta(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	var req model.AppBatteryOtaCheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errcode.NewWithMessage(errcode.CodeParamError, "invalid request body"))
		return
	}

	data, err := service.GroupApp.AppBattery.CheckBatteryOtaForApp(context.Background(), req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetMeterOtaPackages 获取APP端仪表升级包列表
// @Summary 获取仪表升级包列表
// @Description 返回当前租户全部仪表升级包
// @Tags APP-Battery
// @Accept json
// @Produce json
// @Success 200 {array} model.AppBatteryMeterOtaPackageResp
// @Router /api/v1/app/battery/ota/meter-packages [get]
func (*AppBatteryApi) GetMeterOtaPackages(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	tenantHeader := middleware.GetTenantIDFromHeader(c)

	data, err := service.GroupApp.AppBattery.GetMeterOtaPackagesForApp(context.Background(), userClaims, tenantHeader)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// ReportBatteryData APP端BMS数据上报（BLE 经 App 上云）
// @Summary APP端BMS数据上报
// @Description BLE连接场景下，APP将读取到的BMS核心数据/快照上报到云端
// @Tags APP-Battery
// @Accept json
// @Produce json
// @Param body body model.AppBatteryReportReq true "上报请求"
// @Success 200 {object} model.AppBatteryReportResp
// @Router /api/v1/app/battery/report [post]
func (*AppBatteryApi) ReportBatteryData(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	var req model.AppBatteryReportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errcode.NewWithMessage(errcode.CodeParamError, "invalid request body"))
		return
	}

	data, err := service.GroupApp.AppBattery.ReportBatteryDataForApp(context.Background(), req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// ReportBatteryConnectionStatus APP端蓝牙连接状态同步
// @Summary APP端蓝牙连接状态同步
// @Description 手机端蓝牙连接/断开时主动同步设备在线状态
// @Tags APP-Battery
// @Accept json
// @Produce json
// @Param body body model.AppBatteryConnectionStatusReq true "连接状态同步请求"
// @Success 200 {object} model.AppBatteryConnectionStatusResp
// @Router /api/v1/app/battery/connection-status [post]
func (*AppBatteryApi) ReportBatteryConnectionStatus(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	var req model.AppBatteryConnectionStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errcode.NewWithMessage(errcode.CodeParamError, "invalid request body"))
		return
	}

	data, err := service.GroupApp.AppBattery.ReportBatteryConnectionStatusForApp(context.Background(), req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// ServeBatterySocketByWS APP端：MQTT透传(WebSocket桥接)
// 客户端首次消息需发送 JSON：{"device_id":"...","token":"...","features":["mqtt_socket_owner_v1"]}
// 随后发送：
// - "ping" -> "pong"
// - {"hex":"00AABB"} 或 纯十六进制字符串 -> 发布到 device/socket/rx/{device_id}
// 服务器订阅 device/socket/tx/{device_id} 并原样转发给客户端
// @Router /api/v1/app/battery/socket/ws [get]
func (*AppBatteryApi) ServeBatterySocketByWS(c *gin.Context) {
	conn, err := Wsupgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Error(errcode.NewWithMessage(errcode.CodeSystemError, "WebSocket upgrade failed"))
		return
	}
	defer conn.Close()
	var writeMu sync.Mutex

	msgType, msg, err := conn.ReadMessage()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Failed to read message"))
		return
	}

	var initMsg map[string]interface{}
	if err := json.Unmarshal(msg, &initMsg); err != nil {
		conn.WriteMessage(msgType, []byte("Invalid message format"))
		return
	}

	deviceIDVal, ok := initMsg["device_id"]
	if !ok {
		conn.WriteMessage(msgType, []byte("device_id is required"))
		return
	}
	deviceID, _ := deviceIDVal.(string)
	if deviceID == "" {
		conn.WriteMessage(msgType, []byte("device_id must be a non-empty string"))
		return
	}

	claims, err := validateAuth(initMsg)
	if err != nil {
		conn.WriteMessage(msgType, []byte(err.Error()))
		return
	}
	ownerFeatureEnabled := websocketInitHasFeature(initMsg, service.AppBatteryMqttSocketOwnerFeatureV1)
	platform := strings.TrimSpace(websocketInitString(initMsg, "platform"))

	writeControl := func(payload map[string]interface{}, fallbackText string) {
		writeMu.Lock()
		defer writeMu.Unlock()
		if ownerFeatureEnabled {
			b, _ := json.Marshal(payload)
			_ = conn.WriteMessage(websocket.TextMessage, b)
			return
		}
		_ = conn.WriteMessage(msgType, []byte(fallbackText))
	}

	// 校验：设备必须绑定到当前用户（避免任意设备透传）
	detail, err := service.GroupApp.AppBattery.GetBatteryDetailForApp(context.Background(), deviceID, claims)
	if err != nil {
		conn.WriteMessage(msgType, []byte(err.Error()))
		return
	}
	socketTopicID := strings.TrimSpace(detail.DeviceNumber)
	if socketTopicID == "" {
		conn.WriteMessage(msgType, []byte("device_number is required for mqtt socket topic"))
		return
	}

	owner, acquired, err := service.GroupApp.AppBattery.OpenMqttSocketOwnerForApp(context.Background(), deviceID, socketTopicID, platform, claims)
	if err != nil {
		writeControl(map[string]interface{}{
			"type":    "socket_error",
			"message": "实时连接暂不可用，请稍后重试",
		}, "mqtt socket unavailable")
		return
	}
	if !acquired {
		writeControl(map[string]interface{}{
			"type":           "socket_occupied",
			"message":        "设备正在被其他账号实时连接",
			"retry_after_ms": 30000,
		}, "device socket occupied")
		return
	}
	defer service.GroupApp.AppBattery.CloseMqttSocketOwnerForApp(context.Background(), owner.DeviceID, owner.SessionID)

	// 使用后台MQTT配置作为透传桥接（APP无需直连broker）
	broker := viper.GetString("mqtt.broker")
	if broker == "" {
		broker = viper.GetString("mqtt.access_address")
	}
	if broker == "" {
		broker = "127.0.0.1:1883"
	}

	user := viper.GetString("mqtt.user")
	pass := viper.GetString("mqtt.pass")

	clientIDPart := deviceID
	if len(clientIDPart) > 8 {
		clientIDPart = clientIDPart[:8]
	}
	mqttClientID := fmt.Sprintf("app_ws_%s_%d", clientIDPart, time.Now().UnixNano())
	brokerURL := broker
	if !strings.Contains(brokerURL, "://") {
		brokerURL = "tcp://" + brokerURL
	}
	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(mqttClientID).
		SetUsername(user).
		SetPassword(pass).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(10 * time.Second)

	mc := mqtt.NewClient(opts)
	if token := mc.Connect(); token.Wait() && token.Error() != nil {
		writeControl(map[string]interface{}{
			"type":    "socket_error",
			"message": "MQTT连接失败",
		}, "mqtt connect failed")
		return
	}
	defer func() {
		mc.Disconnect(250)
	}()

	txTopic := fmt.Sprintf("device/socket/tx/%s", socketTopicID)
	rxTopic := fmt.Sprintf("device/socket/rx/%s", socketTopicID)
	bootTrace := &socketBootSessionTrace{}
	var ownerRefreshMu sync.Mutex
	lastOwnerRefreshAt := time.Now()
	var onlineRefreshMu sync.Mutex
	lastOnlineRefreshAt := time.Time{}

	refreshOwner := func() bool {
		ok, err := service.GroupApp.AppBattery.RefreshMqttSocketOwnerForApp(context.Background(), owner.DeviceID, owner.SessionID)
		if err != nil {
			writeControl(map[string]interface{}{
				"type":    "socket_error",
				"message": "实时连接状态刷新失败",
			}, "mqtt socket owner refresh failed")
			return false
		}
		if !ok {
			writeControl(map[string]interface{}{
				"type":           "socket_occupied",
				"message":        "设备实时连接已被释放，请重新进入详情页",
				"retry_after_ms": 30000,
			}, "device socket owner lost")
			return false
		}
		return true
	}
	refreshOwnerForData := func() bool {
		ownerRefreshMu.Lock()
		if time.Since(lastOwnerRefreshAt) < socketOwnerDataRefreshMinInterval {
			ownerRefreshMu.Unlock()
			return true
		}
		ownerRefreshMu.Unlock()
		if !refreshOwner() {
			return false
		}
		ownerRefreshMu.Lock()
		lastOwnerRefreshAt = time.Now()
		ownerRefreshMu.Unlock()
		return true
	}
	markOnlineByInteraction := func(source string) {
		now := time.Now()
		onlineRefreshMu.Lock()
		if !lastOnlineRefreshAt.IsZero() && now.Sub(lastOnlineRefreshAt) < socketOnlineRefreshMinInterval {
			onlineRefreshMu.Unlock()
			return
		}
		lastOnlineRefreshAt = now
		onlineRefreshMu.Unlock()

		if changed, err := service.GroupApp.AppBattery.MarkFourGBatteryOnlineByInteraction(context.Background(), detail, source); err != nil {
			onlineRefreshMu.Lock()
			lastOnlineRefreshAt = time.Time{}
			onlineRefreshMu.Unlock()
			logrus.WithError(err).WithFields(logrus.Fields{
				"device_id":     deviceID,
				"device_number": socketTopicID,
				"session_id":    owner.SessionID,
				"source":        source,
			}).Warn("mark 4g battery online by mqtt socket interaction failed")
		} else if changed {
			logrus.WithFields(logrus.Fields{
				"device_id":     deviceID,
				"device_number": socketTopicID,
				"session_id":    owner.SessionID,
				"source":        source,
			}).Info("4g battery online by mqtt socket interaction")
		}
	}

	subToken := mc.Subscribe(txTopic, 1, func(_ mqtt.Client, m mqtt.Message) {
		callbackAt := time.Now()
		payload := append([]byte(nil), m.Payload()...)
		traceFields := socketFrameLogFields(payload)
		traceFields["direction"] = "mqtt_tx_to_ws"
		traceFields["device_id"] = deviceID
		traceFields["device_number"] = socketTopicID
		traceFields["session_id"] = owner.SessionID
		traceFields["topic"] = txTopic
		traceFields["mqtt_qos"] = m.Qos()
		traceFields["mqtt_message_id"] = m.MessageID()
		traceFields["mqtt_retained"] = m.Retained()
		traceFields["mqtt_duplicate"] = m.Duplicate()
		bootInfo, isBootFrame := socketBootFrameForLog(payload)
		if isBootFrame {
			for k, v := range bootInfo.logFields() {
				traceFields[k] = v
			}
		}
		if shouldIgnoreSocketUplink(m.Retained()) {
			logrus.WithFields(traceFields).Warn("bms mqtt socket retained uplink ignored")
			return
		}
		slowAck := false
		if isBootFrame {
			slowAck = bootTrace.observeAck(traceFields, bootInfo, callbackAt)
		}
		if !m.Retained() {
			go markOnlineByInteraction("mqtt_socket_uplink")
		}
		writeMu.Lock()
		lockWaitMs := time.Since(callbackAt).Milliseconds()
		writeStartedAt := time.Now()
		writeErr := conn.WriteMessage(websocket.TextMessage, payload)
		writeMs := time.Since(writeStartedAt).Milliseconds()
		writeMu.Unlock()
		traceFields["ws_lock_wait_ms"] = lockWaitMs
		traceFields["ws_write_ms"] = writeMs
		traceFields["bridge_elapsed_ms"] = time.Since(callbackAt).Milliseconds()
		entry := logrus.WithFields(traceFields)
		if writeErr != nil {
			entry.WithError(writeErr).Warn("bms mqtt socket uplink websocket write failed")
			return
		}
		if isBootFrame {
			if slowAck {
				entry.Warn("bms mqtt socket boot uplink slow ack")
			} else {
				entry.Info("bms mqtt socket boot uplink websocket write")
			}
			return
		}
		if lockWaitMs > 200 || writeMs > 200 {
			entry.Warn("bms mqtt socket uplink websocket write slow")
		} else {
			entry.Debug("bms mqtt socket uplink websocket write")
		}
	})
	if subToken.Wait() && subToken.Error() != nil {
		writeControl(map[string]interface{}{
			"type":    "socket_error",
			"message": "MQTT订阅失败",
		}, "mqtt subscribe failed")
		return
	}
	defer mc.Unsubscribe(txTopic)

	if ownerFeatureEnabled {
		writeControl(map[string]interface{}{
			"type":          "socket_ready",
			"session_id":    owner.SessionID,
			"device_id":     owner.DeviceID,
			"device_number": owner.DeviceNumber,
		}, "")
	}

	// 主循环：读取客户端消息 -> 发布到 MQTT
	for {
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		_, in, err := conn.ReadMessage()
		readAt := time.Now()
		if err != nil {
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				continue
			}
			// 关闭
			return
		}

		txt := string(in)
		if txt == "ping" {
			if !refreshOwner() {
				return
			}
			ownerRefreshMu.Lock()
			lastOwnerRefreshAt = time.Now()
			ownerRefreshMu.Unlock()
			writeMu.Lock()
			_ = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
			writeMu.Unlock()
			continue
		}

		// 支持两种格式：
		// 1) {"hex":"..."}
		// 2) "00AABB"
		payload := in
		var body struct {
			Hex string `json:"hex"`
		}
		if err := json.Unmarshal(in, &body); err == nil && body.Hex != "" {
			// 标准 JSON 透传
			payload = in
		} else {
			// 纯 hex：包装成 JSON
			body.Hex = txt
			b, _ := json.Marshal(body)
			payload = b
		}

		traceFields := socketFrameLogFields(payload)
		traceFields["direction"] = "ws_to_mqtt_rx"
		traceFields["device_id"] = deviceID
		traceFields["device_number"] = socketTopicID
		traceFields["session_id"] = owner.SessionID
		traceFields["topic"] = rxTopic
		traceFields["ws_payload_bytes"] = len(in)
		bootInfo, isBootFrame := socketBootFrameForLog(payload)
		if isBootFrame {
			for k, v := range bootInfo.logFields() {
				traceFields[k] = v
			}
		}
		refreshStartedAt := time.Now()
		if !refreshOwnerForData() {
			return
		}
		traceFields["owner_refresh_ms"] = time.Since(refreshStartedAt).Milliseconds()
		// 发布
		publishStartedAt := time.Now()
		pub := mc.Publish(rxTopic, 1, false, payload)
		if tokenWithMessageID, ok := pub.(interface{ MessageID() uint16 }); ok {
			traceFields["mqtt_publish_message_id"] = tokenWithMessageID.MessageID()
		}
		pub.Wait()
		publishMs := time.Since(publishStartedAt).Milliseconds()
		pubErr := pub.Error()
		slowDownlinkAfterAck := false
		if isBootFrame && pubErr == nil {
			slowDownlinkAfterAck = bootTrace.observeDownlink(traceFields, bootInfo, time.Now())
		}
		traceFields["mqtt_publish_wait_ms"] = publishMs
		traceFields["bridge_elapsed_ms"] = time.Since(readAt).Milliseconds()
		entry := logrus.WithFields(traceFields)
		if isBootFrame {
			if pubErr != nil {
				entry.WithError(pubErr).Warn("bms mqtt socket boot downlink publish failed")
			} else if slowDownlinkAfterAck {
				entry.Warn("bms mqtt socket boot downlink slow after ack")
			} else {
				entry.Info("bms mqtt socket boot downlink publish")
			}
		} else if pubErr != nil {
			entry.WithError(pubErr).Warn("bms mqtt socket downlink publish failed")
		} else if publishMs > 500 {
			entry.Warn("bms mqtt socket downlink publish slow")
		} else {
			entry.Debug("bms mqtt socket downlink publish")
		}
		if pubErr != nil {
			writeControl(map[string]interface{}{
				"type":    "socket_error",
				"message": "MQTT发布失败",
			}, "mqtt publish failed")
		}
	}
}
