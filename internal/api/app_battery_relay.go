package api

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// ServeBatteryRelayByWS APP端 BLE Relay 通道
// 客户端首包:
// {"device_id":"...","token":"...","platform":"wxmp|android_app|ios_app","conn_type":"bluetooth","ble_connected":true}
// 心跳:
// {"type":"relay_heartbeat","ble_connected":true}
// 命令执行回执:
// {"type":"relay_result","cmd_id":"...","ok":true,"result":{...}}
// @Router /api/v1/app/battery/relay/ws [get]
func (*AppBatteryApi) ServeBatteryRelayByWS(c *gin.Context) {
	conn, err := Wsupgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Error(errcode.NewWithMessage(errcode.CodeSystemError, "WebSocket upgrade failed"))
		return
	}
	defer conn.Close()

	msgType, initMsg, err := conn.ReadMessage()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("failed to read init message"))
		return
	}

	var initMap map[string]interface{}
	if err := json.Unmarshal(initMsg, &initMap); err != nil {
		_ = conn.WriteMessage(msgType, []byte("invalid init message format"))
		return
	}

	claims, err := validateAuth(initMap)
	if err != nil {
		_ = conn.WriteMessage(msgType, []byte(err.Error()))
		return
	}

	deviceID, _ := initMap["device_id"].(string)
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		_ = conn.WriteMessage(msgType, []byte("device_id is required"))
		return
	}

	platform := "unknown"
	if raw, ok := initMap["platform"].(string); ok {
		platform = strings.TrimSpace(strings.ToLower(raw))
	}
	connType := "bluetooth"
	if raw, ok := initMap["conn_type"].(string); ok {
		connType = strings.TrimSpace(strings.ToLower(raw))
	}
	bleConnected := true
	if raw, ok := initMap["ble_connected"].(bool); ok {
		bleConnected = raw
	}

	session, err := service.GroupApp.AppBattery.OpenRelaySessionForApp(context.Background(), deviceID, platform, connType, bleConnected, claims)
	if err != nil {
		_ = conn.WriteMessage(msgType, []byte(err.Error()))
		return
	}
	defer service.GroupApp.AppBattery.CloseRelaySessionForApp(context.Background(), session.SessionID)

	ready := map[string]interface{}{
		"type":       "relay_ready",
		"session_id": session.SessionID,
		"device_id":  session.DeviceID,
		"ts":         time.Now().UnixMilli(),
	}
	readyRaw, _ := json.Marshal(ready)
	if err := conn.WriteMessage(msgType, readyRaw); err != nil {
		return
	}

	if global.REDIS == nil {
		_ = conn.WriteMessage(msgType, []byte("redis unavailable"))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	channel := service.AppBatteryRelayCommandChannel(session.SessionID)
	pubsub := global.REDIS.Subscribe(ctx, channel)
	defer pubsub.Close()

	var writeMu sync.Mutex
	go func() {
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case redisMsg, ok := <-ch:
				if !ok {
					return
				}
				writeMu.Lock()
				err := conn.WriteMessage(websocket.TextMessage, []byte(redisMsg.Payload))
				writeMu.Unlock()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_ = conn.SetReadDeadline(time.Now().Add(35 * time.Second))
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					continue
				}
				return
			}

			txt := strings.TrimSpace(string(msg))
			if txt == "ping" {
				_, _ = service.GroupApp.AppBattery.RefreshRelaySessionForApp(context.Background(), session.SessionID, true)
				writeMu.Lock()
				_ = conn.WriteMessage(websocket.TextMessage, []byte("pong"))
				writeMu.Unlock()
				continue
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(msg, &payload); err != nil {
				continue
			}
			msgTypeStr, _ := payload["type"].(string)
			msgTypeStr = strings.TrimSpace(strings.ToLower(msgTypeStr))

			switch msgTypeStr {
			case "relay_heartbeat", "heartbeat":
				connected := true
				if v, ok := payload["ble_connected"].(bool); ok {
					connected = v
				}
				_, _ = service.GroupApp.AppBattery.RefreshRelaySessionForApp(context.Background(), session.SessionID, connected)
			case "relay_result", "relay_cmd_result":
				cmdID, _ := payload["cmd_id"].(string)
				cmdID = strings.TrimSpace(cmdID)
				if cmdID == "" {
					continue
				}
				ok := false
				if v, exist := payload["ok"].(bool); exist {
					ok = v
				}
				var errMsg *string
				if rawErr, exist := payload["error"]; exist {
					msg := strings.TrimSpace(wsMessageToString(rawErr))
					if msg != "" {
						errMsg = &msg
					}
				}
				resp, err := service.GroupApp.AppBattery.AcceptRelayCommandResultForApp(
					context.Background(),
					session.SessionID,
					cmdID,
					ok,
					payload["result"],
					errMsg,
				)
				if err != nil {
					logrus.WithError(err).WithFields(logrus.Fields{
						"session_id": session.SessionID,
						"cmd_id":     cmdID,
					}).Warn("accept relay result failed")
					continue
				}
				ack := map[string]interface{}{
					"type":      "relay_result_ack",
					"cmd_id":    cmdID,
					"status":    resp.Status,
					"updatedAt": resp.UpdatedAtTs,
				}
				ackRaw, _ := json.Marshal(ack)
				writeMu.Lock()
				_ = conn.WriteMessage(websocket.TextMessage, ackRaw)
				writeMu.Unlock()
			default:
				// 忽略未识别消息，保持向后兼容
			}
		}
	}
}

func wsMessageToString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// GetBatteryRelayStatus WEB端查询当前设备是否有可用 BLE Relay owner
// @Router /api/v1/battery/relay/status/{id} [get]
func (*BatteryApi) GetBatteryRelayStatus(c *gin.Context) {
	deviceID := c.Param("id")
	claims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.AppBattery.GetRelayStatusForWeb(context.Background(), deviceID, claims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// SendBatteryRelayCommand WEB端通过 App Relay 下发 BLE 指令
// @Router /api/v1/battery/relay/command [post]
func (*BatteryApi) SendBatteryRelayCommand(c *gin.Context) {
	var req model.AppBatteryRelayCommandReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.AppBattery.SendRelayCommandForWeb(context.Background(), req, claims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBatteryRelayCommand WEB端查询 Relay 指令执行状态
// @Router /api/v1/battery/relay/command/{id} [get]
func (*BatteryApi) GetBatteryRelayCommand(c *gin.Context) {
	commandID := c.Param("id")
	claims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.AppBattery.GetRelayCommandForWeb(context.Background(), commandID, claims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
