package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"project/internal/model"
	"project/pkg/errcode"
	"project/pkg/global"

	"github.com/spf13/viper"
)

type bmsConnInfoRow struct {
	DeviceID string `gorm:"column:device_id"`
	Voucher  string `gorm:"column:voucher"`
}

func getBmsDefaultTopic(configKey, fallback string) string {
	if v := strings.TrimSpace(viper.GetString(configKey)); v != "" {
		return v
	}
	return fallback
}

func (*Device) GetBMSDeviceConnInfo(ctx context.Context, req *model.DeviceConnInfoReq) (*model.DeviceConnInfoResp, error) {
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "tenant_id is required")
	}
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device_id is required")
	}
	sk := strings.TrimSpace(req.SK)
	if sk == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "sk is required")
	}

	// 校验安全 KEY
	dcInfoSK, err := GetDictValueByKey("DC_INFO_SK", tenantID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if dcInfoSK == "" {
		return nil, errcode.NewWithMessage(errcode.CodeNotFound, "DC_INFO_SK not configured")
	}
	if dcInfoSK != sk {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "sk is invalid")
	}

	// 查询设备（device_batteries.item_uuid -> devices.id）
	var row bmsConnInfoRow
	if err := global.DB.WithContext(ctx).
		Table("device_batteries AS dbat").
		Select("d.id AS device_id, d.voucher AS voucher").
		Joins("JOIN devices AS d ON d.id = dbat.device_id").
		Where("dbat.item_uuid = ? AND d.tenant_id = ?", deviceID, tenantID).
		Limit(1).
		Scan(&row).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if row.DeviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeNotFound, "device not found by item_uuid")
	}

	// 解析凭证
	voucherText := strings.TrimSpace(row.Voucher)
	if voucherText == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device voucher is empty")
	}
	var voucher model.DeviceVoucher
	if err := json.Unmarshal([]byte(voucherText), &voucher); err != nil {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device voucher is not valid json")
	}
	username := strings.TrimSpace(voucher.Username)
	if username == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device voucher username is empty")
	}
	password := strings.TrimSpace(voucher.Password)

	// 取 MQTT Broker 地址（字典优先）
	dtuDomainPort, err := GetDictValueByKey("DTU_DOMAIN_PORT", tenantID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if dtuDomainPort == "" {
		if v := strings.TrimSpace(viper.GetString("bms.dtu_domain_port")); v != "" {
			dtuDomainPort = v
		} else {
			dtuDomainPort = strings.TrimSpace(viper.GetString("bms.provision.dtu_domain_port"))
		}
	}
	if dtuDomainPort == "" {
		return nil, errcode.NewWithMessage(errcode.CodeNotFound, "dtu_domain_port not configured")
	}

	// 取 TX/RX Topic（字典优先，再走配置默认）
	txTopic, err := GetDictValueByKey("TX_TOPIC", tenantID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	rxTopic, err := GetDictValueByKey("RX_TOPIC", tenantID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	txDefault := getBmsDefaultTopic("bms.tx_topic", "device/socket/tx/{device_id}")
	rxDefault := getBmsDefaultTopic("bms.rx_topic", "device/socket/rx/{device_id}")
	if strings.TrimSpace(txTopic) == "" {
		txTopic = txDefault
	}
	if strings.TrimSpace(rxTopic) == "" {
		rxTopic = rxDefault
	}

	txTopic = strings.ReplaceAll(txTopic, "{device_id}", deviceID)
	rxTopic = strings.ReplaceAll(rxTopic, "{device_id}", deviceID)

	return &model.DeviceConnInfoResp{
		DTUDomainPort: dtuDomainPort,
		ClientID:      fmt.Sprintf("bms_%s", row.DeviceID),
		Username:      username,
		Password:      password,
		TxTopic:       txTopic,
		RxTopic:       rxTopic,
	}, nil
}
