# 设备接入 MQTT 指南

本文档详细说明如何将物联网设备接入 ThingsPanel 平台的 MQTT Broker，包括认证、数据上报、命令接收等完整流程。

## 目录

- [1. 概述](#1-概述)
- [2. MQTT 连接认证](#2-mqtt-连接认证)
- [3. 消息格式规范](#3-消息格式规范)
- [4. 数据上报](#4-数据上报)
- [5. 命令接收](#5-命令接收)
- [6. 完整示例](#6-完整示例)
- [7. 常见问题](#7-常见问题)

---

## 1. 概述

### 1.1 基本概念

ThingsPanel 平台支持通过 MQTT 协议接入物联网设备，支持以下四种数据类型：

- **遥测 (Telemetry)**: 设备实时上报的测量数据，如温度、湿度等
- **属性 (Attributes)**: 设备的静态或较少变化的特征，如 IP 地址、MAC 地址、固件版本等
- **事件 (Events)**: 设备中发生的特定事件或状态变化，如告警、故障等
- **命令 (Commands)**: 平台向设备下发的控制指令

### 1.2 连接信息获取

设备接入前，需要通过平台 API 获取连接信息：

**API 端点**: `/api/device/connect`

**请求参数**:
```json
{
  "device_id": "设备ID"
}
```

**响应示例**:
```json
{
  "500001": "127.0.0.1:1883",           // MQTT Broker 地址
  "500002": "mqtt_abc123def4",          // ClientID（建议格式：mqtt_ + 设备ID前12位）
  "500003": "devices/telemetry",        // 遥测数据上报 Topic
  "500004": "devices/telemetry/control/{device_number}",  // 遥测控制订阅 Topic
  "500005": "{\"switch\":1}"            // 示例数据
}
```

### 1.3 MQTT Broker 配置

MQTT Broker 地址配置在 `backend/configs/conf.yml` 中：

```yaml
mqtt:
  access_address: 127.0.0.1:1883  # 设备连接地址
  broker: 127.0.0.1:1883           # 平台连接地址
  user: root                       # MQTT 用户名
  pass: root                       # MQTT 密码
```

---

## 2. MQTT 连接认证

### 2.1 认证方式

设备使用 **MQTT Username/Password 认证**方式连接 Broker。

### 2.2 凭证获取

设备凭证存储在平台的设备记录中，格式为 JSON：

```json
{
  "username": "设备用户名（UUID）",
  "password": "设备密码（UUID前7位）"
}
```

**凭证类型**：

1. **BASIC 类型**：包含 username 和 password
   ```json
   {
     "username": "abc123-def456-ghi789-jkl012",
     "password": "xyz1234"
   }
   ```

2. **ACCESSTOKEN 类型**：仅包含 username（无 password）
   ```json
   {
     "username": "abc123-def456-ghi789-jkl012"
   }
   ```

### 2.3 认证要求

| 项目 | 要求 |
|------|------|
| **唯一性** | Username + Password 组合必须唯一<br>ClientID 必须唯一 |
| **一致性** | 设备每次连接必须使用相同的 ClientID、Username 和 Password |

### 2.4 连接参数

**Python 示例**:

```python
# Python 示例
import paho.mqtt.client as mqtt

client = mqtt.Client(client_id="mqtt_abc123def4")
client.username_pw_set(
    username="abc123-def456-ghi789-jkl012",
    password="xyz1234"  # ACCESSTOKEN 类型时可为空
)
client.connect("127.0.0.1", 1883, 60)
```

**ESP32 C 语言示例** (ESP-IDF):

```c
#include "mqtt_client.h"

// MQTT 客户端配置
esp_mqtt_client_config_t mqtt_cfg = {
    .broker.address.uri = "mqtt://127.0.0.1:1883",
    .credentials.username = "abc123-def456-ghi789-jkl012",
    .credentials.authentication.password = "xyz1234",  // ACCESSTOKEN 类型时可为 NULL
    .session.keepalive = 60,
    .session.disable_clean_session = false,
};

// 创建并启动 MQTT 客户端
esp_mqtt_client_handle_t mqtt_client = esp_mqtt_client_init(&mqtt_cfg);
esp_mqtt_client_register_event(mqtt_client, ESP_EVENT_ANY_ID, 
                               mqtt_event_handler, NULL);
esp_mqtt_client_start(mqtt_client);
```

---

## 3. 消息格式规范

### 3.1 统一消息格式

所有设备上报的消息必须遵循以下格式：

```json
{
  "device_id": "设备ID（必填）",
  "values": {
    // 实际数据内容
  }
}
```

**字段说明**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `device_id` | String | 是 | 设备的唯一标识符（UUID） |
| `values` | Object/Array | 是 | 实际的数据内容，可以是对象或数组 |

### 3.2 设备身份识别

**重要**：所有设备使用相同的 Topic 发送消息，服务端通过消息 payload 中的 `device_id` 字段识别设备身份。

例如，所有直连设备都向 `devices/telemetry` 发送遥测数据，服务端解析 payload 中的 `device_id` 来确定消息来源。

---

## 4. 数据上报

### 4.1 遥测数据上报

#### Topic

- **直连设备**: `devices/telemetry`
- **网关设备**: `gateway/telemetry`

#### QoS

- **推荐**: QoS 0（最多传递一次）
- **可选**: QoS 1（至少传递一次）

#### 消息格式

```json
{
  "device_id": "abc123-def456-ghi789-jkl012",
  "values": {
    "temperature": 25.5,
    "humidity": 60,
    "pressure": 1013.25
  }
}
```

#### 示例代码

**Python 版本**:

```python
import json
import paho.mqtt.client as mqtt

# 连接 MQTT Broker
client = mqtt.Client(client_id="mqtt_abc123def4")
client.username_pw_set(username="设备用户名", password="设备密码")
client.connect("127.0.0.1", 1883, 60)

# 构造消息
payload = {
    "device_id": "abc123-def456-ghi789-jkl012",
    "values": {
        "temperature": 25.5,
        "humidity": 60
    }
}

# 发布消息
client.publish(
    topic="devices/telemetry",
    payload=json.dumps(payload),
    qos=0
)
```

**ESP32 C 语言版本** (ESP-IDF):

```c
#include <stdio.h>
#include <string.h>
#include "esp_log.h"
#include "mqtt_client.h"
#include "cJSON.h"

static const char *TAG = "MQTT_TELEMETRY";

// MQTT 客户端句柄（全局变量）
esp_mqtt_client_handle_t mqtt_client = NULL;

// 上报遥测数据
void report_telemetry(float temperature, float humidity)
{
    // 构造 JSON 消息
    cJSON *root = cJSON_CreateObject();
    cJSON *values = cJSON_CreateObject();
    
    cJSON_AddStringToObject(root, "device_id", "abc123-def456-ghi789-jkl012");
    cJSON_AddNumberToObject(values, "temperature", temperature);
    cJSON_AddNumberToObject(values, "humidity", humidity);
    cJSON_AddItemToObject(root, "values", values);
    
    // 转换为字符串
    char *json_string = cJSON_Print(root);
    
    // 发布消息
    int msg_id = esp_mqtt_client_publish(mqtt_client, 
                                         "devices/telemetry", 
                                         json_string, 
                                         0, 0, 0);
    
    if (msg_id >= 0) {
        ESP_LOGI(TAG, "遥测数据已发送, msg_id=%d", msg_id);
    } else {
        ESP_LOGE(TAG, "遥测数据发送失败");
    }
    
    // 释放内存
    free(json_string);
    cJSON_Delete(root);
}
```

### 4.2 属性上报

#### Topic

- **直连设备**: `devices/attributes/{message_id}`
- **网关设备**: `gateway/attributes/{message_id}`

**注意**: `{message_id}` 是消息唯一标识符，建议使用毫秒时间戳的后7位。

#### QoS

- **推荐**: QoS 1（至少传递一次）

#### 消息格式

```json
{
  "device_id": "abc123-def456-ghi789-jkl012",
  "values": {
    "ip": "192.168.1.100",
    "mac": "00:11:22:33:44:55",
    "firmware_version": "v1.0.0"
  }
}
```

#### 示例代码

**Python 版本**:

```python
import time

# 生成 message_id（毫秒时间戳后7位）
message_id = str(int(time.time() * 1000))[-7:]

# 构造消息
payload = {
    "device_id": "abc123-def456-ghi789-jkl012",
    "values": {
        "ip": "192.168.1.100",
        "mac": "00:11:22:33:44:55"
    }
}

# 发布消息
topic = f"devices/attributes/{message_id}"
client.publish(
    topic=topic,
    payload=json.dumps(payload),
    qos=1
)
```

**ESP32 C 语言版本** (ESP-IDF):

```c
#include <time.h>
#include "esp_log.h"
#include "mqtt_client.h"
#include "cJSON.h"

static const char *TAG = "MQTT_ATTRIBUTE";

// 生成 message_id（毫秒时间戳后7位）
void generate_message_id(char *msg_id, size_t len)
{
    struct timeval tv;
    gettimeofday(&tv, NULL);
    int64_t ms = (int64_t)tv.tv_sec * 1000 + tv.tv_usec / 1000;
    snprintf(msg_id, len, "%07lld", ms % 10000000);
}

// 上报属性
void report_attribute(const char *ip, const char *mac)
{
    char message_id[8];
    generate_message_id(message_id, sizeof(message_id));
    
    // 构造 JSON 消息
    cJSON *root = cJSON_CreateObject();
    cJSON *values = cJSON_CreateObject();
    
    cJSON_AddStringToObject(root, "device_id", "abc123-def456-ghi789-jkl012");
    cJSON_AddStringToObject(values, "ip", ip);
    cJSON_AddStringToObject(values, "mac", mac);
    cJSON_AddItemToObject(root, "values", values);
    
    // 转换为字符串
    char *json_string = cJSON_Print(root);
    
    // 构造 Topic
    char topic[64];
    snprintf(topic, sizeof(topic), "devices/attributes/%s", message_id);
    
    // 发布消息 (QoS 1)
    int msg_id = esp_mqtt_client_publish(mqtt_client, 
                                         topic, 
                                         json_string, 
                                         0, 1, 0);
    
    if (msg_id >= 0) {
        ESP_LOGI(TAG, "属性已上报, topic=%s, msg_id=%d", topic, msg_id);
    } else {
        ESP_LOGE(TAG, "属性上报失败");
    }
    
    // 释放内存
    free(json_string);
    cJSON_Delete(root);
}
```

#### 平台响应

设备上报属性后，平台会在以下 Topic 发送响应：

- **直连设备**: `devices/attributes/response/{device_number}/{message_id}`
- **网关设备**: `gateway/attributes/response/{device_number}/{message_id}`

**响应格式**:
```json
{
  "result": 0,        // 0-成功, 1-失败
  "message": "success",
  "ts": 1609143039    // 时间戳（秒）
}
```

### 4.3 事件上报

#### Topic

- **直连设备**: `devices/event/{message_id}`
- **网关设备**: `gateway/event/{message_id}`

#### QoS

- **推荐**: QoS 1（至少传递一次）

#### 消息格式

```json
{
  "device_id": "abc123-def456-ghi789-jkl012",
  "values": {
    "method": "TemperatureExceeded",
    "params": {
      "temperature": 35.5,
      "threshold": 30.0,
      "timestamp": 1609143039
    }
  }
}
```

**字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `method` | String | 是 | 事件类型标识，如 `TemperatureExceeded`、`MotionDetected` 等 |
| `params` | Object | 是 | 事件相关的参数数据 |

#### 示例代码

**Python 版本**:

```python
import time

message_id = str(int(time.time() * 1000))[-7:]

payload = {
    "device_id": "abc123-def456-ghi789-jkl012",
    "values": {
        "method": "TemperatureExceeded",
        "params": {
            "temperature": 35.5,
            "threshold": 30.0
        }
    }
}

topic = f"devices/event/{message_id}"
client.publish(
    topic=topic,
    payload=json.dumps(payload),
    qos=1
)
```

**ESP32 C 语言版本** (ESP-IDF):

```c
#include "esp_log.h"
#include "mqtt_client.h"
#include "cJSON.h"

static const char *TAG = "MQTT_EVENT";

// 上报事件
void report_event(const char *method, float temperature, float threshold)
{
    char message_id[8];
    generate_message_id(message_id, sizeof(message_id));
    
    // 构造 JSON 消息
    cJSON *root = cJSON_CreateObject();
    cJSON *values = cJSON_CreateObject();
    cJSON *params = cJSON_CreateObject();
    
    cJSON_AddStringToObject(root, "device_id", "abc123-def456-ghi789-jkl012");
    cJSON_AddStringToObject(values, "method", method);
    cJSON_AddNumberToObject(params, "temperature", temperature);
    cJSON_AddNumberToObject(params, "threshold", threshold);
    cJSON_AddItemToObject(values, "params", params);
    cJSON_AddItemToObject(root, "values", values);
    
    // 转换为字符串
    char *json_string = cJSON_Print(root);
    
    // 构造 Topic
    char topic[64];
    snprintf(topic, sizeof(topic), "devices/event/%s", message_id);
    
    // 发布消息 (QoS 1)
    int msg_id = esp_mqtt_client_publish(mqtt_client, 
                                         topic, 
                                         json_string, 
                                         0, 1, 0);
    
    if (msg_id >= 0) {
        ESP_LOGI(TAG, "事件已上报, topic=%s, method=%s", topic, method);
    } else {
        ESP_LOGE(TAG, "事件上报失败");
    }
    
    // 释放内存
    free(json_string);
    cJSON_Delete(root);
}
```

#### 平台响应

设备上报事件后，平台会在以下 Topic 发送响应：

- **直连设备**: `devices/event/response/{device_number}/{message_id}`
- **网关设备**: `gateway/event/response/{device_number}/{message_id}`

**响应格式**:
```json
{
  "result": 0,
  "message": "success",
  "ts": 1609143039,
  "method": "TemperatureExceeded"
}
```

### 4.4 状态上报

#### Topic

- **格式**: `devices/status/{device_id}`

#### QoS

- **推荐**: QoS 1

#### 消息格式

状态消息的 payload 为字符串：

- `"1"`: 设备在线
- `"0"`: 设备离线

#### 示例代码

**Python 版本**:

```python
device_id = "abc123-def456-ghi789-jkl012"

# 设备上线
client.publish(
    topic=f"devices/status/{device_id}",
    payload="1",
    qos=1
)

# 设备下线
client.publish(
    topic=f"devices/status/{device_id}",
    payload="0",
    qos=1
)
```

**ESP32 C 语言版本** (ESP-IDF):

```c
#include "esp_log.h"
#include "mqtt_client.h"

static const char *TAG = "MQTT_STATUS";

// 上报设备状态
void report_status(const char *device_id, bool online)
{
    char topic[64];
    snprintf(topic, sizeof(topic), "devices/status/%s", device_id);
    
    const char *payload = online ? "1" : "0";
    
    // 发布消息 (QoS 1)
    int msg_id = esp_mqtt_client_publish(mqtt_client, 
                                         topic, 
                                         payload, 
                                         0, 1, 0);
    
    if (msg_id >= 0) {
        ESP_LOGI(TAG, "状态已上报, topic=%s, status=%s", topic, payload);
    } else {
        ESP_LOGE(TAG, "状态上报失败");
    }
}
```

---

## 5. 命令接收

### 5.1 订阅主题

设备需要订阅以下 Topic 以接收平台下发的命令：

#### 5.1.1 遥测控制

- **直连设备**: `devices/telemetry/control/{device_number}`
- **网关设备**: `gateway/telemetry/control/{device_number}`

**消息格式**:
```json
{
  "temperature": 28.5,
  "light": 2000,
  "switch": true
}
```

#### 5.1.2 属性设置

- **直连设备**: `devices/attributes/set/{device_number}/{message_id}`
- **网关设备**: `gateway/attributes/set/{device_number}/{message_id}`

**消息格式**:
```json
{
  "ip": "192.168.1.100",
  "mac": "00:11:22:33:44:55",
  "port": 1883
}
```

**响应 Topic**:
- **直连设备**: `devices/attributes/set/response/{message_id}`
- **网关设备**: `gateway/attributes/set/response/{message_id}`

**响应格式**:
```json
{
  "result": 0,        // 0-成功, 1-失败
  "message": "success",
  "ts": 1609143039
}
```

#### 5.1.3 属性查询

- **直连设备**: `devices/attributes/get/{device_number}`
- **网关设备**: `gateway/attributes/get/{device_number}`

**请求格式**:
```json
{
  "keys": []  // 空数组表示查询所有属性，或指定属性键名数组
}
```

**示例**:
```json
{
  "keys": ["ip", "mac"]  // 仅查询 ip 和 mac 属性
}
```

#### 5.1.4 命令下发

- **直连设备**: `devices/command/{device_number}/{message_id}`
- **网关设备**: `gateway/command/{device_number}/{message_id}`

**消息格式**:
```json
{
  "method": "SetTemperature",
  "params": {
    "temperature": 25.0,
    "mode": "auto"
  }
}
```

**响应 Topic**:
- **直连设备**: `devices/command/response/{message_id}`
- **网关设备**: `gateway/command/response/{message_id}`

**响应格式**:
```json
{
  "device_id": "abc123-def456-ghi789-jkl012",
  "values": {
    "result": 0,
    "message": "success",
    "ts": 1609143039,
    "method": "SetTemperature"
  }
}
```

### 5.2 订阅示例代码

```python
import json

# 消息处理回调
def on_message(client, userdata, msg):
    topic = msg.topic
    payload = json.loads(msg.payload.decode())
    
    if "telemetry/control" in topic:
        # 处理遥测控制
        print(f"收到遥测控制: {payload}")
        # 执行控制逻辑
        
    elif "attributes/set" in topic:
        # 处理属性设置
        print(f"收到属性设置: {payload}")
        # 更新设备属性
        
        # 发送响应
        message_id = topic.split("/")[-1]
        response_topic = f"devices/attributes/set/response/{message_id}"
        response = {
            "result": 0,
            "message": "success",
            "ts": int(time.time())
        }
        client.publish(response_topic, json.dumps(response), qos=1)
        
    elif "attributes/get" in topic:
        # 处理属性查询
        print(f"收到属性查询: {payload}")
        # 返回当前属性值
        
    elif "command" in topic:
        # 处理命令
        print(f"收到命令: {payload}")
        method = payload.get("method")
        params = payload.get("params")
        
        # 执行命令逻辑
        # ...
        
        # 发送响应
        message_id = topic.split("/")[-1]
        response_topic = f"devices/command/response/{message_id}"
        response = {
            "device_id": "abc123-def456-ghi789-jkl012",
            "values": {
                "result": 0,
                "message": "success",
                "ts": int(time.time()),
                "method": method
            }
        }
        client.publish(response_topic, json.dumps(response), qos=1)

# 设置消息回调
client.on_message = on_message

# 订阅主题
device_number = "设备编号"
topics = [
    f"devices/telemetry/control/{device_number}",
    f"devices/attributes/set/{device_number}/+",
    f"devices/attributes/get/{device_number}",
    f"devices/command/{device_number}/+"
]

for topic in topics:
    client.subscribe(topic, qos=1)
    print(f"已订阅: {topic}")

# 开始监听
client.loop_start()
```

---

## 6. 完整示例

### 6.1 Python 完整示例

```python
#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import json
import time
import paho.mqtt.client as mqtt

# 设备配置
DEVICE_ID = "abc123-def456-ghi789-jkl012"
DEVICE_NUMBER = "设备编号"
MQTT_BROKER = "127.0.0.1"
MQTT_PORT = 1883
MQTT_USERNAME = "设备用户名"
MQTT_PASSWORD = "设备密码"
CLIENT_ID = f"mqtt_{DEVICE_ID[:12]}"

# MQTT 客户端
client = mqtt.Client(client_id=CLIENT_ID)
client.username_pw_set(MQTT_USERNAME, MQTT_PASSWORD)

def on_connect(client, userdata, flags, rc):
    """连接成功回调"""
    if rc == 0:
        print("✅ MQTT 连接成功")
        
        # 订阅命令主题
        topics = [
            f"devices/telemetry/control/{DEVICE_NUMBER}",
            f"devices/attributes/set/{DEVICE_NUMBER}/+",
            f"devices/attributes/get/{DEVICE_NUMBER}",
            f"devices/command/{DEVICE_NUMBER}/+"
        ]
        
        for topic in topics:
            client.subscribe(topic, qos=1)
            print(f"📥 已订阅: {topic}")
    else:
        print(f"❌ 连接失败，错误码: {rc}")

def on_message(client, userdata, msg):
    """消息接收回调"""
    topic = msg.topic
    try:
        payload = json.loads(msg.payload.decode())
        print(f"📨 收到消息 - Topic: {topic}")
        print(f"   内容: {json.dumps(payload, indent=2, ensure_ascii=False)}")
        
        # 处理不同类型的消息
        if "telemetry/control" in topic:
            handle_telemetry_control(payload)
        elif "attributes/set" in topic:
            handle_attribute_set(topic, payload)
        elif "attributes/get" in topic:
            handle_attribute_get(payload)
        elif "command" in topic:
            handle_command(topic, payload)
            
    except Exception as e:
        print(f"❌ 处理消息失败: {e}")

def handle_telemetry_control(payload):
    """处理遥测控制"""
    print("🔧 执行遥测控制...")
    # 实现控制逻辑

def handle_attribute_set(topic, payload):
    """处理属性设置"""
    print("🔧 更新设备属性...")
    # 更新属性
    
    # 发送响应
    message_id = topic.split("/")[-1]
    response_topic = f"devices/attributes/set/response/{message_id}"
    response = {
        "result": 0,
        "message": "success",
        "ts": int(time.time())
    }
    client.publish(response_topic, json.dumps(response), qos=1)
    print(f"✅ 已发送属性设置响应: {response_topic}")

def handle_attribute_get(payload):
    """处理属性查询"""
    print("📤 返回设备属性...")
    # 返回当前属性值

def handle_command(topic, payload):
    """处理命令"""
    method = payload.get("method")
    params = payload.get("params")
    print(f"⚡ 执行命令: {method}, 参数: {params}")
    
    # 执行命令逻辑
    # ...
    
    # 发送响应
    message_id = topic.split("/")[-1]
    response_topic = f"devices/command/response/{message_id}"
    response = {
        "device_id": DEVICE_ID,
        "values": {
            "result": 0,
            "message": "success",
            "ts": int(time.time()),
            "method": method
        }
    }
    client.publish(response_topic, json.dumps(response), qos=1)
    print(f"✅ 已发送命令响应: {response_topic}")

# 设置回调
client.on_connect = on_connect
client.on_message = on_message

# 连接 Broker
print(f"🔌 正在连接 MQTT Broker: {MQTT_BROKER}:{MQTT_PORT}")
client.connect(MQTT_BROKER, MQTT_PORT, 60)

# 启动循环
client.loop_start()

# 上报遥测数据
def report_telemetry():
    """上报遥测数据"""
    payload = {
        "device_id": DEVICE_ID,
        "values": {
            "temperature": 25.5,
            "humidity": 60,
            "pressure": 1013.25
        }
    }
    client.publish("devices/telemetry", json.dumps(payload), qos=0)
    print(f"📤 已上报遥测数据: {payload}")

# 上报属性
def report_attribute():
    """上报属性"""
    message_id = str(int(time.time() * 1000))[-7:]
    payload = {
        "device_id": DEVICE_ID,
        "values": {
            "ip": "192.168.1.100",
            "mac": "00:11:22:33:44:55",
            "firmware_version": "v1.0.0"
        }
    }
    topic = f"devices/attributes/{message_id}"
    client.publish(topic, json.dumps(payload), qos=1)
    print(f"📤 已上报属性: {topic}")

# 上报事件
def report_event():
    """上报事件"""
    message_id = str(int(time.time() * 1000))[-7:]
    payload = {
        "device_id": DEVICE_ID,
        "values": {
            "method": "TemperatureExceeded",
            "params": {
                "temperature": 35.5,
                "threshold": 30.0
            }
        }
    }
    topic = f"devices/event/{message_id}"
    client.publish(topic, json.dumps(payload), qos=1)
    print(f"📤 已上报事件: {topic}")

# 上报状态
def report_status(online=True):
    """上报设备状态"""
    status = "1" if online else "0"
    topic = f"devices/status/{DEVICE_ID}"
    client.publish(topic, status, qos=1)
    print(f"📤 已上报状态: {topic} = {status}")

# 主循环
try:
    # 等待连接建立
    time.sleep(2)
    
    # 上报设备上线
    report_status(online=True)
    
    # 模拟数据上报
    while True:
        report_telemetry()
        time.sleep(10)  # 每10秒上报一次
        
        # 每30秒上报一次属性
        if int(time.time()) % 30 == 0:
            report_attribute()
            
except KeyboardInterrupt:
    print("\n👋 正在断开连接...")
    report_status(online=False)
    client.loop_stop()
    client.disconnect()
    print("✅ 已断开连接")
```

### 6.2 ESP32 C 语言完整示例 (ESP-IDF)

#### 6.2.1 项目配置

**CMakeLists.txt**:

```cmake
cmake_minimum_required(VERSION 3.5)

include($ENV{IDF_PATH}/tools/cmake/project.cmake)
project(thingspanel_mqtt_device)
```

**main/CMakeLists.txt**:

```cmake
idf_component_register(
    SRCS "main.c"
    INCLUDE_DIRS "."
    PRIV_REQUIRES mqtt esp_http_client nvs_flash esp_wifi json cJSON
)
```

#### 6.2.2 完整代码示例

**main/main.c**:

```c
#include <stdio.h>
#include <string.h>
#include <time.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/event_groups.h"
#include "esp_system.h"
#include "esp_wifi.h"
#include "esp_event.h"
#include "esp_log.h"
#include "nvs_flash.h"
#include "mqtt_client.h"
#include "cJSON.h"

static const char *TAG = "THINGSPANEL_DEVICE";

// 设备配置
#define DEVICE_ID           "abc123-def456-ghi789-jkl012"
#define DEVICE_NUMBER       "device_001"
#define MQTT_BROKER         "mqtt://127.0.0.1:1883"
#define MQTT_USERNAME       "设备用户名"
#define MQTT_PASSWORD       "设备密码"
#define CLIENT_ID           "mqtt_abc123def4"

// MQTT 客户端句柄
static esp_mqtt_client_handle_t mqtt_client = NULL;

// WiFi 配置（根据实际情况修改）
#define WIFI_SSID           "YourWiFiSSID"
#define WIFI_PASSWORD       "YourWiFiPassword"

// 事件组位
static EventGroupHandle_t s_wifi_event_group;
#define WIFI_CONNECTED_BIT  BIT0
#define WIFI_FAIL_BIT       BIT1

// WiFi 事件处理
static void wifi_event_handler(void* arg, esp_event_base_t event_base,
                               int32_t event_id, void* event_data)
{
    if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_START) {
        esp_wifi_connect();
    } else if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_DISCONNECTED) {
        esp_wifi_connect();
        xEventGroupClearBits(s_wifi_event_group, WIFI_CONNECTED_BIT);
    } else if (event_base == IP_EVENT && event_id == IP_EVENT_STA_GOT_IP) {
        ip_event_got_ip_t* event = (ip_event_got_ip_t*) event_data;
        ESP_LOGI(TAG, "获得 IP 地址:" IPSTR, IP2STR(&event->ip_info.ip));
        xEventGroupSetBits(s_wifi_event_group, WIFI_CONNECTED_BIT);
    }
}

// WiFi 初始化
static void wifi_init_sta(void)
{
    s_wifi_event_group = xEventGroupCreate();

    ESP_ERROR_CHECK(esp_netif_init());
    ESP_ERROR_CHECK(esp_event_loop_create_default());
    esp_netif_create_default_wifi_sta();

    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
    ESP_ERROR_CHECK(esp_wifi_init(&cfg));

    esp_event_handler_instance_t instance_any_id;
    esp_event_handler_instance_t instance_got_ip;
    ESP_ERROR_CHECK(esp_event_handler_instance_register(WIFI_EVENT,
                                                        ESP_EVENT_ANY_ID,
                                                        &wifi_event_handler,
                                                        NULL,
                                                        &instance_any_id));
    ESP_ERROR_CHECK(esp_event_handler_instance_register(IP_EVENT,
                                                        IP_EVENT_STA_GOT_IP,
                                                        &wifi_event_handler,
                                                        NULL,
                                                        &instance_got_ip));

    wifi_config_t wifi_config = {
        .sta = {
            .ssid = WIFI_SSID,
            .password = WIFI_PASSWORD,
            .threshold.authmode = WIFI_AUTH_WPA2_PSK,
        },
    };
    ESP_ERROR_CHECK(esp_wifi_set_mode(WIFI_MODE_STA));
    ESP_ERROR_CHECK(esp_wifi_set_config(WIFI_IF_STA, &wifi_config));
    ESP_ERROR_CHECK(esp_wifi_start());

    ESP_LOGI(TAG, "WiFi 初始化完成，正在连接...");
}

// 生成 message_id
static void generate_message_id(char *msg_id, size_t len)
{
    struct timeval tv;
    gettimeofday(&tv, NULL);
    int64_t ms = (int64_t)tv.tv_sec * 1000 + tv.tv_usec / 1000;
    snprintf(msg_id, len, "%07lld", ms % 10000000);
}

// MQTT 事件处理
static void mqtt_event_handler(void *handler_args, esp_event_base_t base,
                               int32_t event_id, void *event_data)
{
    esp_mqtt_event_handle_t event = event_data;
    esp_mqtt_client_handle_t client = event->client;
    int msg_id;

    switch ((esp_mqtt_event_id_t)event_id) {
    case MQTT_EVENT_CONNECTED:
        ESP_LOGI(TAG, "MQTT 连接成功");
        
        // 订阅命令主题
        char topic[128];
        
        // 遥测控制
        snprintf(topic, sizeof(topic), "devices/telemetry/control/%s", DEVICE_NUMBER);
        msg_id = esp_mqtt_client_subscribe(client, topic, 1);
        ESP_LOGI(TAG, "已订阅: %s, msg_id=%d", topic, msg_id);
        
        // 属性设置
        snprintf(topic, sizeof(topic), "devices/attributes/set/%s/+", DEVICE_NUMBER);
        msg_id = esp_mqtt_client_subscribe(client, topic, 1);
        ESP_LOGI(TAG, "已订阅: %s, msg_id=%d", topic, msg_id);
        
        // 属性查询
        snprintf(topic, sizeof(topic), "devices/attributes/get/%s", DEVICE_NUMBER);
        msg_id = esp_mqtt_client_subscribe(client, topic, 1);
        ESP_LOGI(TAG, "已订阅: %s, msg_id=%d", topic, msg_id);
        
        // 命令下发
        snprintf(topic, sizeof(topic), "devices/command/%s/+", DEVICE_NUMBER);
        msg_id = esp_mqtt_client_subscribe(client, topic, 1);
        ESP_LOGI(TAG, "已订阅: %s, msg_id=%d", topic, msg_id);
        
        // 上报设备上线
        report_status(DEVICE_ID, true);
        break;
        
    case MQTT_EVENT_DISCONNECTED:
        ESP_LOGI(TAG, "MQTT 连接断开");
        break;
        
    case MQTT_EVENT_SUBSCRIBED:
        ESP_LOGI(TAG, "订阅成功, msg_id=%d", event->msg_id);
        break;
        
    case MQTT_EVENT_UNSUBSCRIBED:
        ESP_LOGI(TAG, "取消订阅, msg_id=%d", event->msg_id);
        break;
        
    case MQTT_EVENT_PUBLISHED:
        ESP_LOGI(TAG, "消息已发布, msg_id=%d", event->msg_id);
        break;
        
    case MQTT_EVENT_DATA:
        ESP_LOGI(TAG, "收到消息, topic=%.*s, data=%.*s",
                 event->topic_len, event->topic,
                 event->data_len, event->data);
        
        // 处理消息
        handle_mqtt_message(event->topic, event->topic_len,
                           event->data, event->data_len);
        break;
        
    case MQTT_EVENT_ERROR:
        ESP_LOGI(TAG, "MQTT 错误");
        break;
        
    default:
        ESP_LOGI(TAG, "其他 MQTT 事件, id=%d", event->event_id);
        break;
    }
}

// 处理收到的 MQTT 消息
static void handle_mqtt_message(const char *topic, int topic_len,
                                const char *data, int data_len)
{
    char topic_str[128];
    snprintf(topic_str, sizeof(topic_str), "%.*s", topic_len, topic);
    
    char data_str[512];
    snprintf(data_str, sizeof(data_str), "%.*s", data_len, data);
    
    ESP_LOGI(TAG, "处理消息: topic=%s, data=%s", topic_str, data_str);
    
    // 解析 JSON
    cJSON *json = cJSON_Parse(data_str);
    if (json == NULL) {
        ESP_LOGE(TAG, "JSON 解析失败");
        return;
    }
    
    // 根据 Topic 类型处理
    if (strstr(topic_str, "telemetry/control") != NULL) {
        handle_telemetry_control(json);
    } else if (strstr(topic_str, "attributes/set") != NULL) {
        handle_attribute_set(topic_str, json);
    } else if (strstr(topic_str, "attributes/get") != NULL) {
        handle_attribute_get(json);
    } else if (strstr(topic_str, "command") != NULL) {
        handle_command(topic_str, json);
    }
    
    cJSON_Delete(json);
}

// 处理遥测控制
static void handle_telemetry_control(cJSON *json)
{
    ESP_LOGI(TAG, "执行遥测控制");
    // 实现控制逻辑
}

// 处理属性设置
static void handle_attribute_set(const char *topic, cJSON *json)
{
    ESP_LOGI(TAG, "更新设备属性");
    
    // 从 Topic 中提取 message_id
    char *last_slash = strrchr(topic, '/');
    if (last_slash == NULL) {
        ESP_LOGE(TAG, "无法从 Topic 中提取 message_id");
        return;
    }
    const char *message_id = last_slash + 1;
    
    // 更新属性（根据实际需求实现）
    // ...
    
    // 发送响应
    cJSON *response = cJSON_CreateObject();
    cJSON_AddNumberToObject(response, "result", 0);
    cJSON_AddStringToObject(response, "message", "success");
    cJSON_AddNumberToObject(response, "ts", time(NULL));
    
    char *response_str = cJSON_Print(response);
    char response_topic[128];
    snprintf(response_topic, sizeof(response_topic), 
             "devices/attributes/set/response/%s", message_id);
    
    esp_mqtt_client_publish(mqtt_client, response_topic, 
                           response_str, 0, 1, 0);
    
    free(response_str);
    cJSON_Delete(response);
}

// 处理属性查询
static void handle_attribute_get(cJSON *json)
{
    ESP_LOGI(TAG, "返回设备属性");
    // 返回当前属性值（根据实际需求实现）
}

// 处理命令
static void handle_command(const char *topic, cJSON *json)
{
    // 从 Topic 中提取 message_id
    char *last_slash = strrchr(topic, '/');
    if (last_slash == NULL) {
        ESP_LOGE(TAG, "无法从 Topic 中提取 message_id");
        return;
    }
    const char *message_id = last_slash + 1;
    
    // 解析命令
    cJSON *method_item = cJSON_GetObjectItem(json, "method");
    cJSON *params_item = cJSON_GetObjectItem(json, "params");
    
    if (method_item && cJSON_IsString(method_item)) {
        const char *method = method_item->valuestring;
        ESP_LOGI(TAG, "执行命令: method=%s", method);
        
        // 执行命令逻辑（根据实际需求实现）
        // ...
        
        // 发送响应
        cJSON *response = cJSON_CreateObject();
        cJSON *values = cJSON_CreateObject();
        
        cJSON_AddStringToObject(response, "device_id", DEVICE_ID);
        cJSON_AddNumberToObject(values, "result", 0);
        cJSON_AddStringToObject(values, "message", "success");
        cJSON_AddNumberToObject(values, "ts", time(NULL));
        cJSON_AddStringToObject(values, "method", method);
        cJSON_AddItemToObject(response, "values", values);
        
        char *response_str = cJSON_Print(response);
        char response_topic[128];
        snprintf(response_topic, sizeof(response_topic), 
                 "devices/command/response/%s", message_id);
        
        esp_mqtt_client_publish(mqtt_client, response_topic, 
                               response_str, 0, 1, 0);
        
        free(response_str);
        cJSON_Delete(response);
    }
}

// 上报遥测数据
static void report_telemetry(float temperature, float humidity)
{
    cJSON *root = cJSON_CreateObject();
    cJSON *values = cJSON_CreateObject();
    
    cJSON_AddStringToObject(root, "device_id", DEVICE_ID);
    cJSON_AddNumberToObject(values, "temperature", temperature);
    cJSON_AddNumberToObject(values, "humidity", humidity);
    cJSON_AddItemToObject(root, "values", values);
    
    char *json_string = cJSON_Print(root);
    esp_mqtt_client_publish(mqtt_client, "devices/telemetry", 
                           json_string, 0, 0, 0);
    
    free(json_string);
    cJSON_Delete(root);
}

// 上报属性
static void report_attribute(const char *ip, const char *mac)
{
    char message_id[8];
    generate_message_id(message_id, sizeof(message_id));
    
    cJSON *root = cJSON_CreateObject();
    cJSON *values = cJSON_CreateObject();
    
    cJSON_AddStringToObject(root, "device_id", DEVICE_ID);
    cJSON_AddStringToObject(values, "ip", ip);
    cJSON_AddStringToObject(values, "mac", mac);
    cJSON_AddItemToObject(root, "values", values);
    
    char *json_string = cJSON_Print(root);
    char topic[64];
    snprintf(topic, sizeof(topic), "devices/attributes/%s", message_id);
    
    esp_mqtt_client_publish(mqtt_client, topic, json_string, 0, 1, 0);
    
    free(json_string);
    cJSON_Delete(root);
}

// 上报事件
static void report_event(const char *method, float temperature, float threshold)
{
    char message_id[8];
    generate_message_id(message_id, sizeof(message_id));
    
    cJSON *root = cJSON_CreateObject();
    cJSON *values = cJSON_CreateObject();
    cJSON *params = cJSON_CreateObject();
    
    cJSON_AddStringToObject(root, "device_id", DEVICE_ID);
    cJSON_AddStringToObject(values, "method", method);
    cJSON_AddNumberToObject(params, "temperature", temperature);
    cJSON_AddNumberToObject(params, "threshold", threshold);
    cJSON_AddItemToObject(values, "params", params);
    cJSON_AddItemToObject(root, "values", values);
    
    char *json_string = cJSON_Print(root);
    char topic[64];
    snprintf(topic, sizeof(topic), "devices/event/%s", message_id);
    
    esp_mqtt_client_publish(mqtt_client, topic, json_string, 0, 1, 0);
    
    free(json_string);
    cJSON_Delete(root);
}

// 上报设备状态
static void report_status(const char *device_id, bool online)
{
    char topic[64];
    snprintf(topic, sizeof(topic), "devices/status/%s", device_id);
    const char *payload = online ? "1" : "0";
    esp_mqtt_client_publish(mqtt_client, topic, payload, 0, 1, 0);
}

// 数据上报任务
static void data_report_task(void *pvParameters)
{
    TickType_t last_wake_time = xTaskGetTickCount();
    int report_count = 0;
    
    while (1) {
        // 每10秒上报一次遥测数据
        float temperature = 25.0 + (report_count % 10) * 0.5;
        float humidity = 60.0 + (report_count % 5) * 2.0;
        report_telemetry(temperature, humidity);
        
        // 每30次（5分钟）上报一次属性
        if (report_count % 30 == 0) {
            report_attribute("192.168.1.100", "00:11:22:33:44:55");
        }
        
        report_count++;
        vTaskDelayUntil(&last_wake_time, pdMS_TO_TICKS(10000));
    }
}

// 主函数
void app_main(void)
{
    // 初始化 NVS
    esp_err_t ret = nvs_flash_init();
    if (ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }
    ESP_ERROR_CHECK(ret);
    
    // 初始化 WiFi
    wifi_init_sta();
    
    // 等待 WiFi 连接
    xEventGroupWaitBits(s_wifi_event_group,
                       WIFI_CONNECTED_BIT,
                       false,
                       true,
                       portMAX_DELAY);
    
    // 配置 MQTT
    esp_mqtt_client_config_t mqtt_cfg = {
        .broker.address.uri = MQTT_BROKER,
        .credentials.username = MQTT_USERNAME,
        .credentials.authentication.password = MQTT_PASSWORD,
        .session.keepalive = 60,
        .session.disable_clean_session = false,
    };
    
    mqtt_client = esp_mqtt_client_init(&mqtt_cfg);
    esp_mqtt_client_register_event(mqtt_client, ESP_EVENT_ANY_ID, 
                                   mqtt_event_handler, NULL);
    esp_mqtt_client_start(mqtt_client);
    
    // 创建数据上报任务
    xTaskCreate(&data_report_task, "data_report", 4096, NULL, 5, NULL);
    
    ESP_LOGI(TAG, "ThingsPanel MQTT 设备初始化完成");
}
```

#### 6.2.3 编译和烧录

```bash
# 设置 ESP-IDF 环境
. $HOME/esp/esp-idf/export.sh

# 编译项目
idf.py build

# 烧录到设备
idf.py -p /dev/ttyUSB0 flash

# 监控串口输出
idf.py -p /dev/ttyUSB0 monitor
```

#### 6.2.4 依赖组件

在 `main/CMakeLists.txt` 中需要包含以下组件：

- `mqtt`: MQTT 客户端库
- `esp_http_client`: HTTP 客户端（如需要）
- `nvs_flash`: 非易失性存储
- `esp_wifi`: WiFi 功能
- `json`: JSON 解析（ESP-IDF 内置）
- `cJSON`: cJSON 库（需要添加到 components 目录或使用 idf_component_manager）

---

## 7. EMQX Broker 认证配置

### 7.1 概述

如果使用 EMQX 作为 MQTT Broker，需要配置数据库认证来验证设备凭证。设备凭证存储在 PostgreSQL 数据库的 `devices` 表中。

### 7.1.1 快速开始

**前提条件**：
- EMQX 已安装并运行
- PostgreSQL 数据库已配置，包含 `devices` 表
- ThingsPanel 后端已配置好数据库连接

**快速配置步骤**（EMQX 5.x Dashboard）：

1. 登录 EMQX Dashboard（默认：http://localhost:18083，用户名：admin，密码：public）
2. 进入 **访问控制** → **认证** → **创建**
3. 选择 **PostgreSQL**
4. 填写配置信息：
   - **服务器**：`127.0.0.1:5432`
   - **数据库**：`ThingsPanel`
   - **用户名**：`postgres`
   - **密码**：`postgres`（根据实际配置修改）
   - **认证查询**：见下方 SQL
   - **密码加密方式**：`plain`
5. 点击 **创建** 并 **启用**

**认证查询 SQL**（复制到 EMQX Dashboard）：
```sql
SELECT CASE WHEN voucher::jsonb ? 'password' THEN (voucher::jsonb->>'password')::text ELSE '' END AS password FROM devices WHERE voucher::jsonb->>'username' = ${username} AND voucher != '' AND voucher::jsonb ? 'username' LIMIT 1
```

### 7.2 数据库表结构

设备凭证存储在 `devices` 表的 `voucher` 字段中，格式为 JSON：

```sql
-- devices 表结构（相关字段）
CREATE TABLE public.devices (
    id varchar(36) NOT NULL,
    voucher varchar(500) NOT NULL DEFAULT '',  -- 凭证（JSON格式）
    device_number varchar(36) NOT NULL,
    -- ... 其他字段
    CONSTRAINT devices_unique_1 UNIQUE (voucher)
);
```

**voucher 字段示例**：
- BASIC 类型：`{"username":"abc123-def456-ghi789-jkl012","password":"xyz1234"}`
- ACCESSTOKEN 类型：`{"username":"abc123-def456-ghi789-jkl012"}`

### 7.3 EMQX 配置步骤

#### 7.3.1 安装 PostgreSQL 认证插件

1. **通过 Dashboard 配置**：
   - 登录 EMQX Dashboard（默认地址：http://localhost:18083）
   - 进入 **认证** → **认证器** → **添加**
   - 选择 **PostgreSQL**

2. **通过配置文件**：
   编辑 `emqx/etc/emqx.conf` 或使用环境变量

#### 7.3.2 配置 PostgreSQL 数据源

在 EMQX Dashboard 中配置 PostgreSQL 连接信息：

```yaml
# PostgreSQL 连接配置
auth.postgresql.server: "127.0.0.1:5432"
auth.postgresql.database: "ThingsPanel"
auth.postgresql.username: "postgres"
auth.postgresql.password: "postgres"
auth.postgresql.pool_size: 8
```

#### 7.3.3 配置认证查询 SQL

**认证 SQL 查询**（用于验证用户名和密码）：

```sql
SELECT 
    CASE 
        WHEN voucher::jsonb ? 'password' 
        THEN (voucher::jsonb->>'password')::text 
        ELSE '' 
    END AS password
FROM devices 
WHERE voucher::jsonb->>'username' = ${username}
  AND voucher != ''
  AND voucher::jsonb ? 'username'
LIMIT 1
```

**说明**：
- `${username}` 是 EMQX 的占位符，会被客户端提供的用户名替换
- 查询会从 `voucher` JSON 字段中提取 `password`
- 如果 `voucher` 中没有 `password` 字段（ACCESSTOKEN 类型），返回空字符串 `''`
- `voucher::jsonb ? 'username'` 确保 `voucher` 是有效的 JSON 且包含 `username` 字段

**EMQX 配置**（配置文件格式）：
```hocon
auth.pgsql.auth_query = "SELECT CASE WHEN voucher::jsonb ? 'password' THEN (voucher::jsonb->>'password')::text ELSE '' END AS password FROM devices WHERE voucher::jsonb->>'username' = ${username} AND voucher != '' AND voucher::jsonb ? 'username' LIMIT 1"
auth.pgsql.password_hash = plain
```

**注意**：EMQX 5.x 版本使用 `auth.pgsql.*`，EMQX 4.x 版本使用 `auth.postgresql.*`

#### 7.3.4 处理 ACCESSTOKEN 类型（无密码）

对于 ACCESSTOKEN 类型的设备（无密码），认证 SQL 会返回空字符串 `''`。

**EMQX 配置**：
```hocon
auth.pgsql.password_hash = plain
# 允许空密码（EMQX 默认支持空密码，无需额外配置）
```

**验证逻辑**：
- 如果设备提供空密码，EMQX 会将查询返回的空字符串 `''` 与客户端提供的空密码进行比较
- 如果匹配，认证成功
- 如果设备提供了密码但数据库中为空，认证失败（符合安全要求）

#### 7.3.5 配置 ACL（访问控制列表）

**ACL SQL 查询**（控制设备可以订阅/发布的 Topic）：

```sql
-- 允许设备订阅和发布所有主题（根据实际需求调整）
SELECT 
    'allow' AS access,
    'all' AS topic
FROM devices 
WHERE voucher::jsonb->>'username' = ${username}
LIMIT 1
```

**更严格的 ACL 配置**（推荐）：

```sql
-- 允许设备发布到上报主题
SELECT 
    'allow' AS access,
    'devices/telemetry' AS topic,
    'publish' AS action
FROM devices 
WHERE voucher::jsonb->>'username' = ${username}
  AND voucher != ''
LIMIT 1

UNION ALL

-- 允许设备订阅控制主题
SELECT 
    'allow' AS access,
    CONCAT('devices/telemetry/control/', device_number) AS topic,
    'subscribe' AS action
FROM devices 
WHERE voucher::jsonb->>'username' = ${username}
  AND voucher != ''
LIMIT 1
```

**EMQX ACL 配置**：
```yaml
auth.postgresql.acl_query: "SELECT 'allow' AS access, 'all' AS topic FROM devices WHERE voucher::jsonb->>'username' = ${username} LIMIT 1"
```

### 7.4 完整配置示例

#### 7.4.1 EMQX Dashboard 配置

1. **创建认证器**：
   - 名称：`postgresql_auth`
   - 类型：`PostgreSQL`
   - 数据源：配置 PostgreSQL 连接信息

2. **认证查询**：
```sql
SELECT 
    CASE 
        WHEN voucher::jsonb ? 'password' 
        THEN (voucher::jsonb->>'password')::text 
        ELSE '' 
    END AS password
FROM devices 
WHERE voucher::jsonb->>'username' = ${username}
  AND voucher != ''
LIMIT 1
```

3. **密码加密方式**：选择 `plain`（明文）

4. **ACL 查询**（可选）：
```sql
SELECT 'allow' AS access, 'all' AS topic 
FROM devices 
WHERE voucher::jsonb->>'username' = ${username} 
LIMIT 1
```

#### 7.4.2 配置文件方式

**EMQX 5.x 版本**：

编辑 `emqx/etc/emqx.conf` 或创建 `emqx/data/configs/overrides.conf`：

```hocon
# PostgreSQL 认证配置
authentication = [
  {
    mechanism = "password_based"
    backend = "postgresql"
    enable = true
    
    # 数据库连接配置
    server = "127.0.0.1:5432"
    database = "ThingsPanel"
    username = "postgres"
    password = "postgres"
    pool_size = 8
    
    # 认证查询
    query = "SELECT CASE WHEN voucher::jsonb ? 'password' THEN (voucher::jsonb->>'password')::text ELSE '' END AS password FROM devices WHERE voucher::jsonb->>'username' = ${username} AND voucher != '' AND voucher::jsonb ? 'username' LIMIT 1"
    
    # 密码加密方式（明文）
    password_hash_algorithm {
      name = "plain"
    }
  }
]

# ACL 配置（可选）
authorization = {
  sources = [
    {
      type = "postgresql"
      enable = true
      server = "127.0.0.1:5432"
      database = "ThingsPanel"
      username = "postgres"
      password = "postgres"
      pool_size = 8
      query = "SELECT 'allow' AS action, 'all' AS topic FROM devices WHERE voucher::jsonb->>'username' = ${username} LIMIT 1"
    }
  ]
}
```

**EMQX 4.x 版本**：

编辑 `emqx/etc/plugins/emqx_auth_pgsql.conf`：

```hocon
auth.pgsql.server = "127.0.0.1:5432"
auth.pgsql.database = "ThingsPanel"
auth.pgsql.username = "postgres"
auth.pgsql.password = "postgres"
auth.pgsql.pool = 8

# 认证查询
auth.pgsql.auth_query = "SELECT CASE WHEN voucher::jsonb ? 'password' THEN (voucher::jsonb->>'password')::text ELSE '' END AS password FROM devices WHERE voucher::jsonb->>'username' = ${username} AND voucher != '' AND voucher::jsonb ? 'username' LIMIT 1"

# 密码加密方式
auth.pgsql.password_hash = plain

# ACL 查询
auth.pgsql.acl_query = "SELECT 'allow' AS access, 'all' AS topic FROM devices WHERE voucher::jsonb->>'username' = ${username} LIMIT 1"
```

#### 7.4.3 启用插件

```bash
# 方式一：通过 Dashboard
# 在认证器页面点击"启用"

# 方式二：通过命令行
emqx_ctl plugins load emqx_auth_pgsql

# 方式三：通过配置文件
# 在 emqx/etc/plugins/emqx_plugins.conf 中添加：
# {emqx_auth_pgsql, true}.
```

### 7.5 验证配置

#### 7.5.1 测试连接

使用 MQTT 客户端工具测试：

```bash
# 使用 mosquitto_pub 测试
mosquitto_pub -h 127.0.0.1 -p 1883 \
  -u "abc123-def456-ghi789-jkl012" \
  -P "xyz1234" \
  -t "devices/telemetry" \
  -m '{"device_id":"abc123-def456-ghi789-jkl012","values":{"temperature":25.5}}'
```

#### 7.5.2 查看日志

检查 EMQX 日志确认认证是否成功：

```bash
# 查看认证日志
tail -f emqx/log/emqx.log | grep -i auth
```

### 7.6 常见问题

#### 7.6.1 认证失败

**问题**：设备连接时提示认证失败

**排查步骤**：
1. 检查 PostgreSQL 连接是否正常
2. 验证 SQL 查询是否正确
3. 检查 `voucher` 字段格式是否为有效 JSON
4. 确认用户名和密码是否匹配

**调试 SQL**：
```sql
-- 手动测试认证查询
SELECT 
    voucher::jsonb->>'username' AS username,
    CASE 
        WHEN voucher::jsonb ? 'password' 
        THEN (voucher::jsonb->>'password')::text 
        ELSE NULL 
    END AS password
FROM devices 
WHERE voucher::jsonb->>'username' = 'abc123-def456-ghi789-jkl012';
```

#### 7.6.2 ACCESSTOKEN 类型设备无法连接

**问题**：ACCESSTOKEN 类型设备（无密码）无法连接

**解决方案**：
1. 确保认证 SQL 能正确处理 NULL 或空密码
2. 在 EMQX 中允许空密码认证
3. 或者为 ACCESSTOKEN 类型设备设置一个固定空字符串密码

#### 7.6.3 性能优化

**问题**：大量设备连接时性能下降

**优化建议**：
1. 为 `voucher` 字段创建表达式索引（PostgreSQL）：
   ```sql
   -- 为 username 字段创建索引
   CREATE INDEX idx_devices_voucher_username 
   ON devices ((voucher::jsonb->>'username'))
   WHERE voucher != '' AND voucher::jsonb ? 'username';
   
   -- 或者使用 GIN 索引（适合复杂 JSON 查询）
   CREATE INDEX idx_devices_voucher_gin 
   ON devices USING GIN (voucher::jsonb)
   WHERE voucher != '';
   ```

2. 增加 PostgreSQL 连接池大小：
   ```hocon
   auth.pgsql.pool = 16  # EMQX 4.x
   # 或
   pool_size = 16  # EMQX 5.x
   ```

3. 使用 Redis 缓存认证结果（可选，需要配置 Redis 认证器）

4. 定期清理无效凭证：
   ```sql
   -- 清理格式错误的 voucher
   UPDATE devices 
   SET voucher = '' 
   WHERE voucher != '' 
     AND (voucher::jsonb ? 'username' = false OR voucher::jsonb->>'username' = '');
   ```

### 7.7 安全建议

1. **使用 TLS/SSL**：生产环境建议启用 MQTT over TLS（端口 8883）
2. **限制 ACL**：不要使用 `'all'` 主题，根据实际需求配置具体的 Topic 权限
3. **定期更新密码**：虽然设备凭证是自动生成的，但建议定期轮换
4. **监控异常连接**：监控 EMQX 日志，及时发现异常认证尝试

---

## 8. 常见问题

### 7.1 如何获取设备凭证？

设备凭证可以通过以下方式获取：

1. **设备注册时返回**：通过设备认证接口 (`/api/device/auth`) 注册设备时，会返回设备凭证
2. **平台查询**：通过设备管理 API 查询设备信息，其中包含 `voucher` 字段

### 7.2 为什么所有设备使用相同的 Topic？

为了简化 Topic 管理和支持大规模设备接入，所有设备使用共享 Topic（如 `devices/telemetry`）。服务端通过消息 payload 中的 `device_id` 字段识别设备身份。

### 7.3 message_id 如何生成？

`message_id` 建议使用毫秒时间戳的后7位，确保短期内不重复：

```python
import time
message_id = str(int(time.time() * 1000))[-7:]
```

### 7.4 QoS 级别如何选择？

- **QoS 0**: 适用于遥测数据等可容忍丢失的数据
- **QoS 1**: 适用于属性、事件、命令等需要可靠传输的数据

### 7.5 如何处理连接断开？

MQTT 客户端应实现自动重连机制：

```python
def on_disconnect(client, userdata, rc):
    """断开连接回调"""
    print(f"❌ 连接断开，错误码: {rc}")
    if rc != 0:
        print("🔄 尝试重新连接...")
        client.reconnect()

client.on_disconnect = on_disconnect
```

### 7.6 网关设备与直连设备的区别？

- **Topic 前缀不同**：
  - 直连设备：`devices/`
  - 网关设备：`gateway/`
  
- **消息格式相同**：都使用相同的 payload 格式

### 7.7 如何验证消息格式？

确保消息 payload 包含以下必填字段：

```json
{
  "device_id": "必填，不能为空",
  "values": "必填，不能为空"
}
```

---

## 8. 参考资源

- [MQTT 协议规范](https://mqtt.org/)
- [Paho MQTT Python 客户端](https://www.eclipse.org/paho/clients/python/)
- [ThingsPanel 平台文档](../README.md)

---

## 9. 更新日志

| 日期 | 版本 | 说明 |
|------|------|------|
| 2025-01-XX | 1.0.0 | 初始版本 |

---

**文档维护**: ThingsPanel 开发团队  
**最后更新**: 2025-01-XX

