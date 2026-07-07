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
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	for _, sql := range []string{
		`CREATE TABLE users (
			id text primary key,
			tenant_id text,
			user_kind text,
			org_id text
		)`,
		`CREATE TABLE devices (
			id text primary key,
			device_number text,
			name text,
			tenant_id text
		)`,
		`CREATE TABLE device_batteries (
			device_id text primary key,
			battery_model_id text,
			item_uuid text,
			ble_mac text,
			identity_ble_mac text,
			activation_status text,
			transfer_status text,
			activation_date datetime,
			warranty_months integer,
			warranty_start_date datetime,
			warranty_expire_date datetime,
			warranty_manual_override boolean default false,
			warranty_updated_at datetime,
			warranty_updated_by text,
			updated_at datetime
		)`,
		`CREATE TABLE battery_bms_models (
			id text primary key,
			warranty_months integer
		)`,
		`CREATE TABLE device_user_bindings (
			id text primary key,
			user_id text,
			device_id text,
			binding_time datetime,
			is_owner boolean,
			remark text
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

func insertProvisionMacUser(t *testing.T, db *gorm.DB, userID, tenantID, userKind string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO users (id, tenant_id, user_kind) VALUES (?, ?, ?)`, userID, tenantID, userKind).Error; err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
}

func insertProvisionMacBinding(t *testing.T, db *gorm.DB, bindingID, userID, deviceID string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO device_user_bindings (id, user_id, device_id, is_owner) VALUES (?, ?, ?, ?)`, bindingID, userID, deviceID, true).Error; err != nil {
		t.Fatalf("insert binding failed: %v", err)
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

func TestSyncProvisionMacsClaimsConnectionMacFromReusableExternalModuleRecord(t *testing.T) {
	db := setupDeviceProvisionMacTestDB(t)
	insertProvisionMacDevice(t, db, "dev-1", "item-1", "tenant-a", "", "")
	insertProvisionMacDevice(t, db, "dev-2", "item-2", "tenant-a", "EF533171C27F", "")
	row := &deviceProvisionRow{DeviceID: "dev-1", DeviceNumber: "item-1"}

	err := (&DeviceProvision{}).syncProvisionMacs(context.Background(), row, model.DeviceProvisionBindReq{
		ItemUUID:       "item-1",
		BleMac:         strPtr("EF:53:31:71:C2:7F"),
		IdentityBleMac: strPtr("AC:46:1A:0F:41:3D"),
	}, &utils.UserClaims{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("syncProvisionMacs() error = %v", err)
	}

	var rows []struct {
		DeviceID string  `gorm:"column:device_id"`
		BleMac   *string `gorm:"column:ble_mac"`
	}
	if err := db.Table("device_batteries").Select("device_id, ble_mac").Order("device_id").Scan(&rows).Error; err != nil {
		t.Fatalf("query device_batteries failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows len = %d, want 2", len(rows))
	}
	if rows[0].DeviceID != "dev-1" || rows[0].BleMac == nil || *rows[0].BleMac != "EF533171C27F" {
		t.Fatalf("dev-1 ble_mac = %#v, want EF533171C27F", rows[0])
	}
	if rows[1].DeviceID != "dev-2" || rows[1].BleMac != nil {
		t.Fatalf("dev-2 ble_mac = %#v, want nil", rows[1])
	}
}

func TestBindEndUserDeviceTxIsIdempotentWhenAlreadyBoundToCurrentUser(t *testing.T) {
	db := setupDeviceProvisionMacTestDB(t)
	insertProvisionMacUser(t, db, "user-1", "tenant-a", model.UserKindEndUser)
	insertProvisionMacDevice(t, db, "dev-1", "item-1", "tenant-a", "EF533171C27F", "")
	insertProvisionMacBinding(t, db, "binding-1", "user-1", "dev-1")
	row := &deviceProvisionRow{DeviceID: "dev-1", DeviceNumber: "item-1", BleMac: strPtr("EF533171C27F")}

	err := (&DeviceProvision{}).bindEndUserDeviceTx(context.Background(), db, row, &utils.UserClaims{
		ID:       "user-1",
		TenantID: "tenant-a",
	})
	if err != nil {
		t.Fatalf("bindEndUserDeviceTx() error = %v", err)
	}

	var cnt int64
	if err := db.Table("device_user_bindings").Where("device_id = ? AND user_id = ?", "dev-1", "user-1").Count(&cnt).Error; err != nil {
		t.Fatalf("count binding failed: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("binding count = %d, want 1", cnt)
	}
}

func TestSyncProvisionMacsRejectsConnectionMacOwnedByIdentityDevice(t *testing.T) {
	db := setupDeviceProvisionMacTestDB(t)
	insertProvisionMacDevice(t, db, "dev-1", "item-1", "tenant-a", "", "")
	insertProvisionMacDevice(t, db, "dev-2", "item-2", "tenant-a", "EF533171C27F", "EF533171C27F")
	row := &deviceProvisionRow{DeviceID: "dev-1", DeviceNumber: "item-1"}

	err := (&DeviceProvision{}).syncProvisionMacs(context.Background(), row, model.DeviceProvisionBindReq{
		ItemUUID:       "item-1",
		BleMac:         strPtr("EF:53:31:71:C2:7F"),
		IdentityBleMac: strPtr("AC:46:1A:0F:41:3D"),
	}, &utils.UserClaims{TenantID: "tenant-a"})
	if err == nil {
		t.Fatalf("expected identity device mac conflict error")
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

func TestSyncProvisionMacsAllowsConnectionMacReplacementWhenIdentityMacMissing(t *testing.T) {
	db := setupDeviceProvisionMacTestDB(t)
	insertProvisionMacDevice(t, db, "dev-1", "item-1", "tenant-a", "AC4F1B10423E", "")
	row := &deviceProvisionRow{DeviceID: "dev-1", DeviceNumber: "item-1", BleMac: strPtr("AC4F1B10423E")}

	err := (&DeviceProvision{}).syncProvisionMacs(context.Background(), row, model.DeviceProvisionBindReq{
		ItemUUID: "item-1",
		BleMac:   strPtr("EF:53:31:71:C2:7F"),
	}, &utils.UserClaims{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("syncProvisionMacs() error = %v", err)
	}

	var got struct {
		BleMac string `gorm:"column:ble_mac"`
	}
	if err := db.Table("device_batteries").Select("ble_mac").Where("device_id = ?", "dev-1").Scan(&got).Error; err != nil {
		t.Fatalf("query device_batteries failed: %v", err)
	}
	if got.BleMac != "EF533171C27F" {
		t.Fatalf("ble_mac = %s, want EF533171C27F", got.BleMac)
	}
}

func TestSyncProvisionMacsClaimsMissingIdentityMacFromReusableExternalModuleRecord(t *testing.T) {
	db := setupDeviceProvisionMacTestDB(t)
	insertProvisionMacDevice(t, db, "dev-1", "item-1", "tenant-a", "AC4F1B10423E", "")
	insertProvisionMacDevice(t, db, "dev-2", "item-2", "tenant-a", "EF533171C27F", "")
	row := &deviceProvisionRow{DeviceID: "dev-1", DeviceNumber: "item-1", BleMac: strPtr("AC4F1B10423E")}

	err := (&DeviceProvision{}).syncProvisionMacs(context.Background(), row, model.DeviceProvisionBindReq{
		ItemUUID: "item-1",
		BleMac:   strPtr("EF:53:31:71:C2:7F"),
	}, &utils.UserClaims{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("syncProvisionMacs() error = %v", err)
	}

	var rows []struct {
		DeviceID string  `gorm:"column:device_id"`
		BleMac   *string `gorm:"column:ble_mac"`
	}
	if err := db.Table("device_batteries").Select("device_id, ble_mac").Order("device_id").Scan(&rows).Error; err != nil {
		t.Fatalf("query device_batteries failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows len = %d, want 2", len(rows))
	}
	if rows[0].DeviceID != "dev-1" || rows[0].BleMac == nil || *rows[0].BleMac != "EF533171C27F" {
		t.Fatalf("dev-1 ble_mac = %#v, want EF533171C27F", rows[0])
	}
	if rows[1].DeviceID != "dev-2" || rows[1].BleMac != nil {
		t.Fatalf("dev-2 ble_mac = %#v, want nil", rows[1])
	}
}
