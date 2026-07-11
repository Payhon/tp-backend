package mqttadapter

import (
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

// MQTTConfig MQTT 客户端配置
type MQTTConfig struct {
	Broker            string
	Username          string
	Password          string
	ClientID          string                   // 可选，不提供则自动生成
	OnConnectCallback func(client mqtt.Client) // 连接成功回调（用于重新订阅）
	DeliveryOptions   *MQTTDeliveryOptions     // 可选；nil 时保持适配器原有会话与派发行为
}

// MQTTDeliveryOptions 控制 MQTT 会话恢复和回调派发行为。
// 默认调用者无需设置；请求-响应型桥接可使用干净会话避免离线积压，并开启有序回调。
type MQTTDeliveryOptions struct {
	CleanSession bool
	ResumeSubs   bool
	OrderMatters bool
}

func applyDeliveryOptions(opts *mqtt.ClientOptions, delivery *MQTTDeliveryOptions) {
	// 保持历史默认值，避免改变平台 MQTT Adapter 和模拟器等既有调用者。
	cleanSession := false
	resumeSubs := true
	orderMatters := false
	if delivery != nil {
		cleanSession = delivery.CleanSession
		resumeSubs = delivery.ResumeSubs
		orderMatters = delivery.OrderMatters
	}

	opts.SetCleanSession(cleanSession)
	opts.SetResumeSubs(resumeSubs)
	opts.SetOrderMatters(orderMatters)
}

// CreateMQTTClient 创建 MQTT 客户端（Adapter 专用）
func CreateMQTTClient(config MQTTConfig, logger *logrus.Logger) (mqtt.Client, error) {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// 初始化配置
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)

	// 客户端 ID
	clientID := config.ClientID
	if clientID == "" {
		clientID = "thingspanel-adapter-default"
	}
	opts.SetClientID(clientID)

	applyDeliveryOptions(opts, config.DeliveryOptions)
	// 自动重连
	opts.SetAutoReconnect(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(200 * time.Second)
	// 连接成功回调（首次连接 + 重连成功都会触发）
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.WithField("client_id", clientID).Info("MQTT Adapter client connected")

		// ✨ 重连后自动执行订阅回调（确保订阅不丢失）
		if config.OnConnectCallback != nil {
			logger.Info("Executing OnConnectCallback to re-subscribe topics...")
			config.OnConnectCallback(client)
		}
	})

	// 断线回调（仅记录日志，自动重连由 SetAutoReconnect 处理）
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		logger.WithError(err).Warn("MQTT Adapter connection lost, auto-reconnect will handle it...")
	})

	// 重连中回调（可选，帮助追踪重连状态）
	opts.SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		logger.Info("MQTT Adapter reconnecting...")
	})

	// 创建客户端
	client := mqtt.NewClient(opts)

	// 等待连接成功，失败重新连接
	for {
		token := client.Connect()
		if token.Wait() && token.Error() != nil {
			logger.WithError(token.Error()).Error("MQTT Adapter connection failed, retrying...")
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	logger.WithField("client_id", clientID).Info("MQTT Adapter client created and connected")
	return client, nil
}

// DisconnectMQTTClient 断开 MQTT 客户端连接
func DisconnectMQTTClient(client mqtt.Client, logger *logrus.Logger) {
	if client != nil && client.IsConnected() {
		client.Disconnect(250)
		if logger != nil {
			logger.Info("MQTT Adapter client disconnected")
		}
	}
}
