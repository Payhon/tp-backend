package service

import (
	"context"
	"testing"

	"project/internal/model"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupDeviceProvisionMacTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	for _, sql := range []string{
		`CREATE TABLE devices (
			id text primary key,
			device_number text,
			name text,
			tenant_id text
		)`,
		`CREATE TABLE device_batteries (
			device_id text primary key,
			item_uuid text,
			ble_mac text,
			identity_ble_mac text,
			updated_at datetime
		)`,
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("create test table failed: %v", err)
		}
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
	return db
}

func insertProvisionMacDevice(t *testing.T, db *gorm.DB, deviceID, deviceNumber, tenantID, bleMac, identityBleMac string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO devices (id, device_number, name, tenant_id) VALUES (?, ?, ?, ?)`, deviceID, deviceNumber, "BMS-"+deviceNumber, tenantID).Error; err != nil {
		t.Fatalf("insert device failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, item_uuid, ble_mac, identity_ble_mac) VALUES (?, ?, ?, ?)`, deviceID, deviceNumber, bleMac, identityBleMac).Error; err != nil {
		t.Fatalf("insert battery failed: %v", err)
	}
}

func TestSyncProvisionMacsSupportsExternalBleModuleDualMac(t *testing.T) {
	db := setupDeviceProvisionMacTestDB(t)
	insertProvisionMacDevice(t, db, "dev-1", "item-1", "tenant-a", "EF533171C27F", "")
	row := &deviceProvisionRow{DeviceID: "dev-1", DeviceNumber: "item-1", BleMac: strPtr("EF533171C27F")}

	err := (&DeviceProvision{}).syncProvisionMacs(context.Background(), row, model.DeviceProvisionBindReq{
		ItemUUID:       "item-1",
		BleMac:         strPtr("EF:53:31:71:C2:7F"),
		IdentityBleMac: strPtr("AC:46:1A:0F:41:3D"),
	}, &utils.UserClaims{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("syncProvisionMacs() error = %v", err)
	}

	var got struct {
		BleMac         string `gorm:"column:ble_mac"`
		IdentityBleMac string `gorm:"column:identity_ble_mac"`
	}
	if err := db.Table("device_batteries").Select("ble_mac, identity_ble_mac").Where("device_id = ?", "dev-1").Scan(&got).Error; err != nil {
		t.Fatalf("query device_batteries failed: %v", err)
	}
	if got.BleMac != "EF533171C27F" {
		t.Fatalf("ble_mac = %s, want EF533171C27F", got.BleMac)
	}
	if got.IdentityBleMac != "AC461A0F413D" {
		t.Fatalf("identity_ble_mac = %s, want AC461A0F413D", got.IdentityBleMac)
	}
}

func TestSyncProvisionMacsRejectsConnectionMacOwnedByOtherDevice(t *testing.T) {
	db := setupDeviceProvisionMacTestDB(t)
	insertProvisionMacDevice(t, db, "dev-1", "item-1", "tenant-a", "", "")
	insertProvisionMacDevice(t, db, "dev-2", "item-2", "tenant-a", "EF533171C27F", "")
	row := &deviceProvisionRow{DeviceID: "dev-1", DeviceNumber: "item-1"}

	err := (&DeviceProvision{}).syncProvisionMacs(context.Background(), row, model.DeviceProvisionBindReq{
		ItemUUID:       "item-1",
		BleMac:         strPtr("EF:53:31:71:C2:7F"),
		IdentityBleMac: strPtr("AC:46:1A:0F:41:3D"),
	}, &utils.UserClaims{TenantID: "tenant-a"})
	if err == nil {
		t.Fatalf("expected connection mac conflict error")
	}
}

func TestSyncProvisionMacsRejectsIdentityMacConflict(t *testing.T) {
	db := setupDeviceProvisionMacTestDB(t)
	insertProvisionMacDevice(t, db, "dev-1", "item-1", "tenant-a", "EF533171C27F", "111111111111")
	row := &deviceProvisionRow{
		DeviceID:       "dev-1",
		DeviceNumber:   "item-1",
		BleMac:         strPtr("EF533171C27F"),
		IdentityBleMac: strPtr("111111111111"),
	}

	err := (&DeviceProvision{}).syncProvisionMacs(context.Background(), row, model.DeviceProvisionBindReq{
		ItemUUID:       "item-1",
		BleMac:         strPtr("EF:53:31:71:C2:7F"),
		IdentityBleMac: strPtr("AC:46:1A:0F:41:3D"),
	}, &utils.UserClaims{TenantID: "tenant-a"})
	if err == nil {
		t.Fatalf("expected identity mac conflict error")
	}
}

func TestSyncProvisionMacsLegacySingleMacKeepsStrictMismatch(t *testing.T) {
	db := setupDeviceProvisionMacTestDB(t)
	insertProvisionMacDevice(t, db, "dev-1", "item-1", "tenant-a", "EF533171C27F", "")
	row := &deviceProvisionRow{DeviceID: "dev-1", DeviceNumber: "item-1", BleMac: strPtr("EF533171C27F")}

	err := (&DeviceProvision{}).syncProvisionMacs(context.Background(), row, model.DeviceProvisionBindReq{
		ItemUUID: "item-1",
		BleMac:   strPtr("AC:46:1A:0F:41:3D"),
	}, &utils.UserClaims{TenantID: "tenant-a"})
	if err == nil {
		t.Fatalf("expected legacy mismatch error")
	}
}
