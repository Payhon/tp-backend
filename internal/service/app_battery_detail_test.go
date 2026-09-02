package service

import (
	"context"
	"testing"

	"project/pkg/utils"
)

func TestGetBatteryDetailForAppIncludesImei(t *testing.T) {
	db := setupDeviceLastConnectedTestDB(t)
	const (
		deviceID = "device-imei"
		tenantID = "tenant-imei"
		imei     = "862415074075885"
	)

	if err := db.Exec(`
		INSERT INTO devices (id, device_number, name, tenant_id, is_online)
		VALUES (?, ?, ?, ?, ?)
	`, deviceID, "Battery_00112233445566778899AABBCCDDEEFF", "4G Battery", tenantID, 1).Error; err != nil {
		t.Fatalf("insert device failed: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO device_batteries (device_id, bms_comm_type, item_uuid, imei)
		VALUES (?, ?, ?, ?)
	`, deviceID, 2, "00112233445566778899AABBCCDDEEFF", imei).Error; err != nil {
		t.Fatalf("insert battery failed: %v", err)
	}

	detail, err := new(AppBattery).GetBatteryDetailForApp(context.Background(), deviceID, &utils.UserClaims{
		ID:        "admin-imei",
		TenantID:  tenantID,
		Authority: "TENANT_ADMIN",
	})
	if err != nil {
		t.Fatalf("get battery detail failed: %v", err)
	}
	if detail.Imei == nil || *detail.Imei != imei {
		t.Fatalf("expected imei %q, got %#v", imei, detail.Imei)
	}
}
