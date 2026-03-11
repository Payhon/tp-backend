# 第三方系统 API 接入指南

> 适用对象：需要与 FJBMS 对接、自动新增电池并查询电池信息的 MES/ERP/WMS 系统。

## 1. 你将获得什么

完成本文档后，你可以：

1. 使用 `AppId + SecretKey` 完成第三方身份认证。
2. 调用接口自动新增（或补全）电池档案。
3. 按电池序列号（SN）查询电池详情。

---

## 2. 接入前准备

请先让平台管理员在后台创建 API 密钥：

- 菜单：`系统管理 -> API 密钥管理`
- 关键字段：
  - `AppId`
  - `SecretKey`
  - `备注`
  - `有效期`
  - `租户 ID`

说明：

- 第三方接口会自动按密钥绑定的租户写入/查询数据。
- 密钥禁用或过期后，请求会鉴权失败。

### 2.1 环境与鉴权信息

| 环境 | 域名 | AppId | AppSecret |
|---|---|---|---|
| 生产环境 | `https://cloud.fjiaenergy.com` | `app_98f35a52466ffdbfa7694080` | `sk_f59a15163615dec34966babdc9ebc003abf42545d160a6c72ff7757a1ebcc891` |
| 测试环境 | `http://fjbms.yz6688.cn` | `app_c346f4be2623ad9b4e493714` | `sk_8f24eb6acec313fba7e40c4e77ccab897e87e1bd7c0fbaff21b54d9be3e62041` |

---

## 3. 认证方式

所有 MES 开放接口都需要带以下请求头：

```http
x-app-id: app_c346f4be2623ad9b4e493714
x-secret-key: sk_8f24eb6acec313fba7e40c4e77ccab897e87e1bd7c0fbaff21b54d9be3e62041
Content-Type: application/json
```

鉴权失败时常见返回：

- HTTP 状态码：`401`
- 响应示例：

```json
{
  "code": 40103,
  "message": "invalid app_id or secret_key"
}
```

---

## 4. 基础约定

- 基础前缀：`/api/v1`
- MES API 前缀：`/api/v1/openapi/mes`
- 字符编码：`UTF-8`
- 数据格式：`application/json`
- 测试环境 Base URL：`http://fjbms.yz6688.cn`
- 生产环境 Base URL：`https://cloud.fjiaenergy.com`

业务成功响应结构：

```json
{
  "code": 200,
  "message": "Success",
  "data": {}
}
```

> 说明：业务错误通常通过 `code` 字段体现（例如 `100002` 参数错误、`100404` 资源不存在）。

---

## 5. 接口一：新增电池

- Method：`POST`
- URL：`/api/v1/openapi/mes/battery`

### 5.1 请求字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| item_uuid | string | 是 | 电池序列号（SN），系统内映射 `devices.device_number` |
| batch_number | string | 是 | 批次号（最大 100） |
| product_spec | string | 是 | 规格型号文本（最大 32） |
| order_number | string | 是 | 订单号（最大 32） |
| bms_comm_type | int | 是 | 通讯类型：`1=蓝牙`、`2=4G`、`3=蓝牙+4G` |
| ble_mac | string | 否 | 蓝牙 MAC（最大 32） |
| comm_chip_id | string | 否 | 通讯芯片 ID（最大 64） |
| battery_model_id | string | 否 | 电池型号 ID（与 `battery_model_name` 二选一） |
| battery_model_name | string | 否 | 电池型号名称（与 `battery_model_id` 二选一） |
| production_date | string | 否 | 出厂日期，格式：`YYYY-MM-DD` |
| warranty_expire_date | string | 否 | 质保到期日期，格式：`YYYY-MM-DD` |
| remark | string | 否 | 备注（最大 255） |

### 5.2 请求示例

```bash
curl -X POST "http://fjbms.yz6688.cn/api/v1/openapi/mes/battery" \
  -H "Content-Type: application/json" \
  -H "x-app-id: app_c346f4be2623ad9b4e493714" \
  -H "x-secret-key: sk_8f24eb6acec313fba7e40c4e77ccab897e87e1bd7c0fbaff21b54d9be3e62041" \
  -d '{
    "item_uuid": "SN202603050001",
    "batch_number": "BATCH-20260305-A",
    "product_spec": "51.2V100Ah",
    "order_number": "MES-PO-20260305",
    "bms_comm_type": 1,
    "battery_model_name": "FJBMS-100A",
    "ble_mac": "AC:11:22:33:44:55",
    "production_date": "2026-03-05",
    "warranty_expire_date": "2027-03-05",
    "remark": "MES自动建档"
  }'
```

### 5.3 成功响应示例

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "device_id": "f81fcb70-4732-4d8d-b3ab-8ff950451e0c",
    "device_number": "SN202603050001",
    "battery_model_id": "b1f2366d-91b9-4aa6-bf6b-f13d2861a7f1",
    "battery_model_name": "FJBMS-100A",
    "item_uuid": "SN202603050001",
    "batch_number": "BATCH-20260305-A",
    "product_spec": "51.2V100Ah",
    "order_number": "MES-PO-20260305",
    "bms_comm_type": 1,
    "ble_mac": "AC:11:22:33:44:55",
    "comm_chip_id": null,
    "production_date": "2026-03-05",
    "warranty_expire_date": "2027-03-05"
  }
}
```

### 5.4 行为说明

- 当 `item_uuid` 在当前租户不存在时，系统会自动创建对应 `devices` 记录，再写入电池信息。
- 当 `item_uuid` 已存在时，系统会更新该设备对应的电池档案。
- 如果 `item_uuid` 已存在于其他租户，系统会拒绝写入（租户隔离保护）。

---

## 6. 接口二：按序列号查询电池

- Method：`GET`
- URL：`/api/v1/openapi/mes/battery/{serial_number}`

### 6.1 请求示例

```bash
curl -X GET "http://fjbms.yz6688.cn/api/v1/openapi/mes/battery/SN202603050001" \
  -H "x-app-id: app_c346f4be2623ad9b4e493714" \
  -H "x-secret-key: sk_8f24eb6acec313fba7e40c4e77ccab897e87e1bd7c0fbaff21b54d9be3e62041"
```

### 6.2 成功响应示例（节选）

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "device_id": "f81fcb70-4732-4d8d-b3ab-8ff950451e0c",
    "device_number": "SN202603050001",
    "battery_model_id": "b1f2366d-91b9-4aa6-bf6b-f13d2861a7f1",
    "battery_model_name": "FJBMS-100A",
    "item_uuid": "SN202603050001",
    "batch_number": "BATCH-20260305-A",
    "ble_mac": "AC:11:22:33:44:55",
    "production_date": "2026-03-05",
    "warranty_expire_date": "2027-03-05",
    "activation_status": "INACTIVE",
    "is_online": 0
  }
}
```

返回对象还可能包含：持有组织信息、用户绑定信息、SOC/SOH、当前版本号、调拨状态等字段。

---

## 7. 常见错误码与排查

| 场景 | HTTP | code | 说明 |
|---|---:|---:|---|
| 缺少鉴权头 | 401 | 40100 | 未传 `x-app-id` 或 `x-secret-key` |
| 密钥无效/过期/禁用 | 401 | 40103 | `AppId` 与 `SecretKey` 不匹配，或密钥不可用 |
| 参数错误 | 200 | 100002 | 字段缺失、长度不合法、日期格式错误等 |
| 电池不存在 | 200 | 100404 | 查询的 `serial_number` 不在当前租户下 |
| 跨租户冲突 | 200 | 201002 | 设备编号已在其他租户存在，禁止自动创建 |

建议排查顺序：

1. 先确认请求头是否完整。
2. 再确认密钥是否过期/禁用。
3. 最后检查请求体字段及日期格式（`YYYY-MM-DD`）。

---

## 8. 安全与联调建议

- 仅在服务端保存 `SecretKey`，不要写入前端页面或客户端安装包。
- 建议按系统/产线分配独立密钥，便于审计和失效管理。
- 建议按季度轮换 `SecretKey`，并保留灰度切换窗口。
- 建议记录请求日志（请求时间、SN、响应 code、request_id）用于快速定位问题。

---

## 9. 快速联调清单

- [ ] 已从 `系统管理 -> API 密钥管理` 获取 `AppId/SecretKey`
- [ ] 已使用正确的 URL 前缀：`/api/v1/openapi/mes`
- [ ] 新增接口返回 `code=200`
- [ ] 查询接口可按 SN 查到刚新增的电池数据
- [ ] 已完成异常场景验证（鉴权失败、参数错误、SN不存在）
