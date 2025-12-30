# BMS 设备接入 MQTT 指南

本文档详细说明如何将BMS设备接入 Fjia Cloud平台的 MQTT Broker，包括认证、数据上报、命令接收、数据透传等完整流程。

## 目录

- [1. 概述](#1-概述)
- [2. MQTT 连接认证](#2-mqtt-连接认证)
- [3. 消息格式规范](#3-消息格式规范)
- [4. 数据上报](#4-数据上报)
- [5. 命令接收](#5-命令接收)
- [6. 数据透传](#6-数据透传)


---

## 1. 概述

### 1.1 基本概念

Fjia Cloud平台支持通过 MQTT 协议接入BMS设备，支持以下四种数据类型：

- **遥测 (Telemetry)**: 设备实时上报的测量数据，如温度、湿度等
- **属性 (Attributes)**: 设备的静态或较少变化的特征，如 IP 地址、MAC 地址、固件版本等
- **事件 (Events)**: 设备中发生的特定事件或状态变化，如告警、故障等
- **命令 (Commands)**: 平台向设备下发的控制指令
- **透传 (Sokect)**: 使用 Socket Topic 进行 16 进制数据透传转发

### 1.2 连接信息获取

设备接入前，需要通过平台 API 获取连接信息：

**API Host**: https://fjiacloud.com/api  (待定)

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

## 6. 数据透传(Socket通讯）

### 6.1 概述

数据透传功能允许设备通过 MQTT 协议进行原始二进制数据的透传通信。设备连接 MQTT Broker 后，使用两个专用 Topic 进行双向通信：

- **发送 Topic**: 设备向平台/APP 端发送数据
- **接收 Topic**: 设备订阅以接收平台/APP 端下发的数据

所有数据以 16 进制字符串的形式通过 JSON 格式进行传输。

### 6.2 Topic 说明

#### 6.2.1 发送 Topic（设备 → 平台/APP）

- **格式**: `device/socket/tx/{device_id}`
- **说明**: 设备向此 Topic 发布数据，平台/APP 端订阅此 Topic 接收数据

#### 6.2.2 接收 Topic（平台/APP → 设备）

- **格式**: `device/socket/rx/{device_id}`
- **说明**: 设备订阅此 Topic 接收平台/APP 端下发的数据

**注意**: `{device_id}` 为设备的唯一标识符（UUID）。

### 6.3 QoS 设置

- **推荐**: QoS 1（至少传递一次），确保数据可靠传输
- **可选**: QoS 0（最多传递一次），适用于对实时性要求高、允许少量丢失的场景

### 6.4 消息格式

#### 6.4.1 Payload 格式

所有透传消息的 payload 必须遵循以下 JSON 格式：

```json
{
  "hex": "00AABBCCDDEEFF"
}
```

**字段说明**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `hex` | String | 是 | 16 进制字符串，表示原始二进制数据。字符串长度必须为偶数（每两个字符代表一个字节） |

#### 6.4.2 数据格式说明

- 所有二进制数据必须转换为 16 进制字符串
- 16 进制字符串使用大写字母（A-F）
- 字符串长度必须为偶数（每两个字符代表一个字节）
- 示例：字节数组 `[0x00, 0xAA, 0xBB, 0xCC]` 转换为字符串 `"00AABBCC"`

### 6.5 示例代码

#### 6.5.1 Python 版本

**完整示例**：

```python
import json
import paho.mqtt.client as mqtt
import binascii

# MQTT 客户端配置
device_id = "abc123-def456-ghi789-jkl012"
tx_topic = f"device/socket/tx/{device_id}"  # 发送 Topic
rx_topic = f"device/socket/rx/{device_id}"  # 接收 Topic

# 连接 MQTT Broker
client = mqtt.Client(client_id="mqtt_abc123def4")
client.username_pw_set(username="设备用户名", password="设备密码")
client.connect("127.0.0.1", 1883, 60)

# ========== 16 进制编码/解码工具函数 ==========

def bytes_to_hex(data):
    """
    将字节数组或字节串转换为 16 进制字符串
    
    参数:
        data: bytes 或 bytearray 对象
    
    返回:
        16 进制字符串（大写）
    
    示例:
        bytes_to_hex(b'\x00\xAA\xBB') -> "00AABB"
    """
    return binascii.hexlify(data).decode('utf-8').upper()

def hex_to_bytes(hex_string):
    """
    将 16 进制字符串转换为字节数组
    
    参数:
        hex_string: 16 进制字符串
    
    返回:
        bytes 对象
    
    示例:
        hex_to_bytes("00AABB") -> b'\x00\xAA\xBB'
    """
    # 移除可能的空格和分隔符
    hex_string = hex_string.replace(' ', '').replace('-', '').replace(':', '')
    return binascii.unhexlify(hex_string)

# ========== 发送数据（设备 → 平台/APP） ==========

def send_socket_data(data):
    """
    发送透传数据到平台/APP
    
    参数:
        data: bytes 或 bytearray 对象，原始二进制数据
    """
    # 将二进制数据转换为 16 进制字符串
    hex_string = bytes_to_hex(data)
    
    # 构造 JSON payload
    payload = {
        "hex": hex_string
    }
    
    # 发布消息
    result = client.publish(
        topic=tx_topic,
        payload=json.dumps(payload),
        qos=1
    )
    
    if result.rc == mqtt.MQTT_ERR_SUCCESS:
        print(f"数据已发送: {hex_string}")
    else:
        print(f"数据发送失败: {result.rc}")

# ========== 接收数据（平台/APP → 设备） ==========

def on_message(client, userdata, msg):
    """MQTT 消息接收回调"""
    topic = msg.topic
    
    # 检查是否为透传接收 Topic
    if topic == rx_topic:
        try:
            # 解析 JSON payload
            payload = json.loads(msg.payload.decode('utf-8'))
            hex_string = payload.get('hex', '')
            
            if not hex_string:
                print("错误: payload 中缺少 'hex' 字段")
                return
            
            # 将 16 进制字符串转换为字节数组
            data = hex_to_bytes(hex_string)
            
            print(f"收到透传数据: {hex_string}")
            print(f"原始字节数据: {data}")
            print(f"数据长度: {len(data)} 字节")
            
            # 处理接收到的数据
            process_received_data(data)
            
        except json.JSONDecodeError:
            print("错误: 无法解析 JSON payload")
        except binascii.Error:
            print("错误: 无效的 16 进制字符串")
        except Exception as e:
            print(f"错误: {e}")

def process_received_data(data):
    """
    处理接收到的透传数据
    
    参数:
        data: bytes 对象，原始二进制数据
    """
    # 在这里实现您的数据处理逻辑
    # 例如：解析协议、执行命令等
    print(f"处理数据: {data.hex().upper()}")

# 设置消息回调
client.on_message = on_message

# 订阅接收 Topic
client.subscribe(rx_topic, qos=1)
print(f"已订阅接收 Topic: {rx_topic}")

# 开始监听消息
client.loop_start()

# ========== 使用示例 ==========

# 示例 1: 发送简单的字节数据
if __name__ == "__main__":
    # 等待连接建立
    import time
    time.sleep(1)
    
    # 发送数据示例
    # 示例数据: [0x01, 0x02, 0x03, 0x04]
    example_data = bytes([0x01, 0x02, 0x03, 0x04])
    send_socket_data(example_data)
    
    # 示例 2: 发送字符串转换的字节数据
    text_data = "Hello".encode('utf-8')
    send_socket_data(text_data)
    
    # 示例 3: 发送自定义协议数据
    # 假设协议格式: [命令码(1字节)] [数据长度(1字节)] [数据(N字节)]
    command = 0x10
    data_length = 0x03
    data_bytes = bytes([0xAA, 0xBB, 0xCC])
    protocol_data = bytes([command, data_length]) + data_bytes
    send_socket_data(protocol_data)
    
    # 保持运行以接收消息
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("程序退出")
        client.loop_stop()
        client.disconnect()
```

#### 6.5.2 ESP32 C 语言版本 (ESP-IDF)

**完整示例**：

```c
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include "esp_log.h"
#include "mqtt_client.h"
#include "cJSON.h"

static const char *TAG = "MQTT_SOCKET";

// MQTT 客户端句柄（全局变量）
esp_mqtt_client_handle_t mqtt_client = NULL;

// 设备 ID
static const char *device_id = "abc123-def456-ghi789-jkl012";

// ========== 16 进制编码/解码工具函数 ==========

/**
 * 将字节数组转换为 16 进制字符串
 * 
 * @param data 字节数组
 * @param len 数据长度
 * @param hex_str 输出的 16 进制字符串缓冲区（必须至少为 len*2+1 字节）
 * 
 * @return 成功返回 0，失败返回 -1
 */
int bytes_to_hex(const uint8_t *data, size_t len, char *hex_str)
{
    if (data == NULL || hex_str == NULL || len == 0) {
        return -1;
    }
    
    for (size_t i = 0; i < len; i++) {
        sprintf(hex_str + i * 2, "%02X", data[i]);
    }
    hex_str[len * 2] = '\0';
    
    return 0;
}

/**
 * 将 16 进制字符串转换为字节数组
 * 
 * @param hex_str 16 进制字符串
 * @param data 输出的字节数组缓冲区
 * @param max_len 缓冲区最大长度
 * 
 * @return 成功返回转换的字节数，失败返回 -1
 */
int hex_to_bytes(const char *hex_str, uint8_t *data, size_t max_len)
{
    if (hex_str == NULL || data == NULL) {
        return -1;
    }
    
    size_t hex_len = strlen(hex_str);
    if (hex_len % 2 != 0) {
        ESP_LOGE(TAG, "16 进制字符串长度必须为偶数");
        return -1;
    }
    
    size_t byte_len = hex_len / 2;
    if (byte_len > max_len) {
        ESP_LOGE(TAG, "数据长度超出缓冲区大小");
        return -1;
    }
    
    for (size_t i = 0; i < byte_len; i++) {
        char hex_byte[3] = {hex_str[i * 2], hex_str[i * 2 + 1], '\0'};
        data[i] = (uint8_t)strtol(hex_byte, NULL, 16);
    }
    
    return byte_len;
}

// ========== 发送数据（设备 → 平台/APP） ==========

/**
 * 发送透传数据到平台/APP
 * 
 * @param data 原始二进制数据
 * @param len 数据长度
 * 
 * @return 成功返回消息 ID，失败返回 -1
 */
int send_socket_data(const uint8_t *data, size_t len)
{
    if (mqtt_client == NULL || data == NULL || len == 0) {
        ESP_LOGE(TAG, "参数错误");
        return -1;
    }
    
    // 将二进制数据转换为 16 进制字符串
    char *hex_str = malloc(len * 2 + 1);
    if (hex_str == NULL) {
        ESP_LOGE(TAG, "内存分配失败");
        return -1;
    }
    
    if (bytes_to_hex(data, len, hex_str) != 0) {
        free(hex_str);
        return -1;
    }
    
    // 构造 JSON payload
    cJSON *root = cJSON_CreateObject();
    cJSON_AddStringToObject(root, "hex", hex_str);
    
    // 转换为字符串
    char *json_string = cJSON_Print(root);
    if (json_string == NULL) {
        ESP_LOGE(TAG, "JSON 序列化失败");
        cJSON_Delete(root);
        free(hex_str);
        return -1;
    }
    
    // 构造发送 Topic
    char tx_topic[128];
    snprintf(tx_topic, sizeof(tx_topic), "device/socket/tx/%s", device_id);
    
    // 发布消息 (QoS 1)
    int msg_id = esp_mqtt_client_publish(mqtt_client, 
                                         tx_topic, 
                                         json_string, 
                                         0, 1, 0);
    
    if (msg_id >= 0) {
        ESP_LOGI(TAG, "透传数据已发送, topic=%s, hex=%s, msg_id=%d", 
                 tx_topic, hex_str, msg_id);
    } else {
        ESP_LOGE(TAG, "透传数据发送失败");
    }
    
    // 释放内存
    free(json_string);
    free(hex_str);
    cJSON_Delete(root);
    
    return msg_id;
}

// ========== 接收数据（平台/APP → 设备） ==========

/**
 * 处理接收到的透传数据
 * 
 * @param data 原始二进制数据
 * @param len 数据长度
 */
void process_received_data(const uint8_t *data, size_t len)
{
    // 在这里实现您的数据处理逻辑
    // 例如：解析协议、执行命令等
    
    ESP_LOGI(TAG, "处理接收到的数据, 长度=%d 字节", len);
    
    // 打印 16 进制格式
    char *hex_str = malloc(len * 2 + 1);
    if (hex_str != NULL) {
        bytes_to_hex(data, len, hex_str);
        ESP_LOGI(TAG, "数据内容 (16进制): %s", hex_str);
        free(hex_str);
    }
    
    // 示例: 解析简单的协议
    if (len >= 2) {
        uint8_t command = data[0];
        uint8_t data_len = data[1];
        ESP_LOGI(TAG, "命令码: 0x%02X, 数据长度: %d", command, data_len);
    }
}

/**
 * MQTT 事件处理函数
 */
static void mqtt_event_handler(void *handler_args, esp_event_base_t base, 
                                int32_t event_id, void *event_data)
{
    esp_mqtt_event_handle_t event = event_data;
    
    switch (event->event_id) {
        case MQTT_EVENT_CONNECTED:
            ESP_LOGI(TAG, "MQTT 连接成功");
            
            // 订阅接收 Topic
            char rx_topic[128];
            snprintf(rx_topic, sizeof(rx_topic), "device/socket/rx/%s", device_id);
            esp_mqtt_client_subscribe(mqtt_client, rx_topic, 1);
            ESP_LOGI(TAG, "已订阅接收 Topic: %s", rx_topic);
            break;
            
        case MQTT_EVENT_DATA:
            {
                // 构造接收 Topic 字符串用于比较
                char rx_topic[128];
                snprintf(rx_topic, sizeof(rx_topic), "device/socket/rx/%s", device_id);
                
                // 检查是否为透传接收 Topic
                if (strncmp(event->topic, rx_topic, strlen(rx_topic)) == 0) {
                    // 解析 JSON payload
                    char *payload = malloc(event->data_len + 1);
                    if (payload == NULL) {
                        ESP_LOGE(TAG, "内存分配失败");
                        break;
                    }
                    
                    memcpy(payload, event->data, event->data_len);
                    payload[event->data_len] = '\0';
                    
                    cJSON *root = cJSON_Parse(payload);
                    if (root == NULL) {
                        ESP_LOGE(TAG, "JSON 解析失败");
                        free(payload);
                        break;
                    }
                    
                    cJSON *hex_item = cJSON_GetObjectItem(root, "hex");
                    if (hex_item == NULL || !cJSON_IsString(hex_item)) {
                        ESP_LOGE(TAG, "payload 中缺少 'hex' 字段或类型错误");
                        cJSON_Delete(root);
                        free(payload);
                        break;
                    }
                    
                    const char *hex_string = hex_item->valuestring;
                    size_t hex_len = strlen(hex_string);
                    
                    // 将 16 进制字符串转换为字节数组
                    uint8_t *data = malloc(hex_len / 2);
                    if (data == NULL) {
                        ESP_LOGE(TAG, "内存分配失败");
                        cJSON_Delete(root);
                        free(payload);
                        break;
                    }
                    
                    int byte_len = hex_to_bytes(hex_string, data, hex_len / 2);
                    if (byte_len > 0) {
                        ESP_LOGI(TAG, "收到透传数据, 16进制: %s", hex_string);
                        process_received_data(data, byte_len);
                    } else {
                        ESP_LOGE(TAG, "16 进制字符串转换失败");
                    }
                    
                    free(data);
                    cJSON_Delete(root);
                    free(payload);
                }
            }
            break;
            
        case MQTT_EVENT_ERROR:
            ESP_LOGE(TAG, "MQTT 错误");
            break;
            
        default:
            break;
    }
}

// ========== 初始化函数 ==========

/**
 * 初始化 MQTT Socket 透传功能
 */
void init_socket_transmission(void)
{
    // MQTT 客户端配置
    esp_mqtt_client_config_t mqtt_cfg = {
        .broker.address.uri = "mqtt://127.0.0.1:1883",
        .credentials.username = "设备用户名",
        .credentials.authentication.password = "设备密码",
        .session.keepalive = 60,
        .session.disable_clean_session = false,
    };
    
    // 创建并启动 MQTT 客户端
    mqtt_client = esp_mqtt_client_init(&mqtt_cfg);
    esp_mqtt_client_register_event(mqtt_client, ESP_EVENT_ANY_ID, 
                                   mqtt_event_handler, NULL);
    esp_mqtt_client_start(mqtt_client);
}

// ========== 使用示例 ==========

void app_main(void)
{
    // 初始化 MQTT Socket 透传
    init_socket_transmission();
    
    // 等待连接建立
    vTaskDelay(pdMS_TO_TICKS(2000));
    
    // 示例 1: 发送简单的字节数据
    uint8_t example_data[] = {0x01, 0x02, 0x03, 0x04};
    send_socket_data(example_data, sizeof(example_data));
    
    // 示例 2: 发送字符串转换的字节数据
    const char *text = "Hello";
    send_socket_data((const uint8_t *)text, strlen(text));
    
    // 示例 3: 发送自定义协议数据
    // 假设协议格式: [命令码(1字节)] [数据长度(1字节)] [数据(N字节)]
    uint8_t protocol_data[] = {
        0x10,  // 命令码
        0x03,  // 数据长度
        0xAA, 0xBB, 0xCC  // 数据
    };
    send_socket_data(protocol_data, sizeof(protocol_data));
    
    // 主循环
    while (1) {
        vTaskDelay(pdMS_TO_TICKS(1000));
    }
}
```

### 6.6 使用场景

数据透传功能适用于以下场景：

1. **自定义协议通信**: 设备使用自定义二进制协议与平台/APP 端即时通信
2. **固件升级**: 通过透传方式传输固件升级包
3. **文件传输**: 传输配置文件、日志文件等
4. **加密通信**: 传输已加密的二进制数据
5. **第三方协议适配**: 适配不支持 JSON 格式的第三方设备协议

### 6.7 注意事项

1. **数据大小限制**: 注意 MQTT Broker 的消息大小限制（通常为 256KB），超过限制的数据需要分片传输
2. **16 进制字符串验证**: 确保 16 进制字符串格式正确（仅包含 0-9、A-F 字符，长度为偶数）
3. **QoS 选择**: 根据业务需求选择合适的 QoS 级别，重要数据建议使用 QoS 1
4. **错误处理**: 实现完善的错误处理机制，包括 JSON 解析错误、16 进制转换错误等
5. **内存管理**: 在嵌入式设备上注意内存管理，及时释放动态分配的内存

---
