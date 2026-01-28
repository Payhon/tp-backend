package service

import (
	"strings"

	"project/internal/dal"
)

// DictKeyFromConfigKey 将配置KEY转换为字典KEY（如 mqtt.ws_address -> MQTT_WS_ADDRESS）
func DictKeyFromConfigKey(configKey string) string {
	key := strings.TrimSpace(configKey)
	if key == "" {
		return ""
	}
	return strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
}

// GetDictValueByConfigKey 优先读取租户字典KEY，未命中则读取全局字典KEY
func GetDictValueByConfigKey(configKey, tenantID string) (string, error) {
	dictKey := DictKeyFromConfigKey(configKey)
	if dictKey == "" {
		return "", nil
	}

	tenantIDs := []string{"0"}
	preferTenant := "0"
	if strings.TrimSpace(tenantID) != "" && tenantID != "0" {
		tenantIDs = []string{"0", tenantID}
		preferTenant = tenantID
	}

	list, err := dal.GetDictListByCode(dictKey, tenantIDs, preferTenant)
	if err != nil {
		return "", err
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		val := strings.TrimSpace(item.DictValue)
		if val != "" {
			return val, nil
		}
	}
	return "", nil
}

// GetDictValueByKey 直接按字典KEY读取（优先租户，其次全局）
func GetDictValueByKey(dictKey, tenantID string) (string, error) {
	key := strings.TrimSpace(dictKey)
	if key == "" {
		return "", nil
	}

	tenantIDs := []string{"0"}
	preferTenant := "0"
	if strings.TrimSpace(tenantID) != "" && tenantID != "0" {
		tenantIDs = []string{"0", tenantID}
		preferTenant = tenantID
	}

	list, err := dal.GetDictListByCode(key, tenantIDs, preferTenant)
	if err != nil {
		return "", err
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		val := strings.TrimSpace(item.DictValue)
		if val != "" {
			return val, nil
		}
	}
	return "", nil
}
