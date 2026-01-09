package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"project/internal/service"
	"project/pkg/errcode"
	"project/pkg/utils"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"sync"
)

// AppBatteryApi APP端：电池设备详情/透传
type AppBatteryApi struct{}

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

// ServeBatterySocketByWS APP端：MQTT透传(WebSocket桥接)
// 客户端首次消息需发送 JSON：{"device_id":"...","token":"..."}
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
	// 校验：设备必须绑定到当前用户（避免任意设备透传）
	if _, err := service.GroupApp.AppBattery.GetBatteryDetailForApp(context.Background(), deviceID, claims); err != nil {
		conn.WriteMessage(msgType, []byte(err.Error()))
		return
	}

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

	mqttClientID := fmt.Sprintf("app_ws_%s_%d", deviceID[:8], time.Now().UnixNano())
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
		conn.WriteMessage(msgType, []byte("mqtt connect failed"))
		return
	}
	defer func() {
		mc.Disconnect(250)
	}()

	txTopic := fmt.Sprintf("device/socket/tx/%s", deviceID)
	rxTopic := fmt.Sprintf("device/socket/rx/%s", deviceID)

	// websocket 写锁（paho 回调可能并发）
	var writeMu sync.Mutex

	subToken := mc.Subscribe(txTopic, 1, func(_ mqtt.Client, m mqtt.Message) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.WriteMessage(websocket.TextMessage, m.Payload())
	})
	if subToken.Wait() && subToken.Error() != nil {
		conn.WriteMessage(msgType, []byte("mqtt subscribe failed"))
		return
	}
	defer mc.Unsubscribe(txTopic)

	// 主循环：读取客户端消息 -> 发布到 MQTT
	for {
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		_, in, err := conn.ReadMessage()
		if err != nil {
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				continue
			}
			// 关闭
			return
		}

		txt := string(in)
		if txt == "ping" {
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

		// 发布
		pub := mc.Publish(rxTopic, 1, false, payload)
		if pub.Wait() && pub.Error() != nil {
			writeMu.Lock()
			_ = conn.WriteMessage(websocket.TextMessage, []byte("mqtt publish failed"))
			writeMu.Unlock()
		}
	}
}
