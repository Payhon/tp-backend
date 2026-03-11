package global

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var BMSHistoryExportWSManager *UserWSManager

const bmsHistoryExportWSPattern = "ws:bms_history_export:user:*"

// InitBMSHistoryExportWSManager 初始化 BMS 历史导出 WS 管理器。
func InitBMSHistoryExportWSManager() {
	BMSHistoryExportWSManager = NewUserWSManager(REDIS)
	go BMSHistoryExportWSManager.ListenForEvents()
}

// UserWSManager 用户级 WS 管理器。
type UserWSManager struct {
	redisClient *redis.Client
	userClients map[string]map[string]*UserWSClient
	mutex       sync.RWMutex
}

// UserWSClient 用户级 WS 客户端。
type UserWSClient struct {
	UserID  string
	Conn    *websocket.Conn
	ConnID  string
	MsgType int
	Mu      *sync.Mutex
}

// BMSHistoryExportWSEvent Redis 事件内容。
type BMSHistoryExportWSEvent struct {
	UserID  string          `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

func NewUserWSManager(redisClient *redis.Client) *UserWSManager {
	return &UserWSManager{
		redisClient: redisClient,
		userClients: make(map[string]map[string]*UserWSClient),
	}
}

func (m *UserWSManager) Subscribe(userID, connID string, client *UserWSClient) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.userClients[userID]; !ok {
		m.userClients[userID] = make(map[string]*UserWSClient)
	}
	m.userClients[userID][connID] = client
}

func (m *UserWSManager) Unsubscribe(userID, connID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if clients, ok := m.userClients[userID]; ok {
		delete(clients, connID)
		if len(clients) == 0 {
			delete(m.userClients, userID)
		}
	}
}

func (m *UserWSManager) PublishToUser(userID string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if m.redisClient == nil {
		m.pushToUser(userID, payloadBytes)
		return nil
	}

	event := BMSHistoryExportWSEvent{UserID: userID, Payload: payloadBytes}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	channel := "ws:bms_history_export:user:" + userID
	return m.redisClient.Publish(context.Background(), channel, eventBytes).Err()
}

func (m *UserWSManager) ListenForEvents() {
	if m.redisClient == nil {
		return
	}

	ctx := context.Background()
	pubsub := m.redisClient.PSubscribe(ctx, bmsHistoryExportWSPattern)
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			logrus.WithError(err).Warn("bms history export ws receive redis message failed")
			time.Sleep(time.Second)
			continue
		}

		var event BMSHistoryExportWSEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			logrus.WithError(err).Warn("bms history export ws decode redis payload failed")
			continue
		}

		if event.UserID == "" || len(event.Payload) == 0 {
			continue
		}
		m.pushToUser(event.UserID, event.Payload)
	}
}

func (m *UserWSManager) pushToUser(userID string, payload []byte) {
	m.mutex.RLock()
	clients, ok := m.userClients[userID]
	m.mutex.RUnlock()
	if !ok || len(clients) == 0 {
		return
	}

	for connID, client := range clients {
		client.Mu.Lock()
		err := client.Conn.WriteMessage(client.MsgType, payload)
		client.Mu.Unlock()
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"user_id": userID,
				"conn_id": connID,
			}).Warn("bms history export ws write failed")
		}
	}
}
