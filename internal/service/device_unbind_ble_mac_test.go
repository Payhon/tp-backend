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

func setupUnbindBleMacTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	for _, sql := range []string{
		`CREATE TABLE users (
			id text primary key,
			tenant_id text,
			user_kind text,
			org_id text
		)`,
		`CREATE TABLE device_user_bindings (
			id text primary key,
			user_id text,
			device_id text,
			binding_time datetime,
			is_owner boolean,
			remark text
		)`,
		`CREATE TABLE app_device_added_records (
			id text primary key,
			tenant_id text,
			user_id text,
			device_id text,
			source text,
			added_at datetime,
			last_seen_at datetime,
			remark text
		)`,
		`CREATE TABLE device_batteries (
			device_id text primary key,
			ble_mac text,
			identity_ble_mac text,
			owner_org_id text,
			activation_status text,
			activation_date datetime,
			transfer_status text,
			updated_at datetime
		)`,
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("create table failed: %v", err)
		}
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
	return db
}

func insertUnbindBleMacUser(t *testing.T, db *gorm.DB, id, tenantID, userKind string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO users (id, tenant_id, user_kind) VALUES (?, ?, ?)`, id, tenantID, userKind).Error; err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
}

func insertUnbindBleMacBattery(t *testing.T, db *gorm.DB, deviceID string) {
	t.Helper()
	if err := db.Exec(
		`INSERT INTO device_batteries (device_id, ble_mac, identity_ble_mac, activation_status, transfer_status)
		 VALUES (?, 'EF533171C27F', 'AC461A0F413D', 'ACTIVE', 'USER')`,
		deviceID,
	).Error; err != nil {
		t.Fatalf("insert battery failed: %v", err)
	}
}

func insertUnbindBleMacBinding(t *testing.T, db *gorm.DB, id, userID, deviceID string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO device_user_bindings (id, user_id, device_id, is_owner) VALUES (?, ?, ?, ?)`, id, userID, deviceID, true).Error; err != nil {
		t.Fatalf("insert binding failed: %v", err)
	}
}

func insertUnbindBleMacAddedRecord(t *testing.T, db *gorm.DB, id, tenantID, userID, deviceID string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO app_device_added_records (id, tenant_id, user_id, device_id, source) VALUES (?, ?, ?, ?, 'BLE_SCAN')`, id, tenantID, userID, deviceID).Error; err != nil {
		t.Fatalf("insert added record failed: %v", err)
	}
}

func queryUnbindBleMacBattery(t *testing.T, db *gorm.DB, deviceID string) (bleMac *string, identityBleMac *string) {
	t.Helper()
	var row struct {
		BleMac         *string `gorm:"column:ble_mac"`
		IdentityBleMac *string `gorm:"column:identity_ble_mac"`
	}
	if err := db.Table("device_batteries").
		Select("ble_mac, identity_ble_mac").
		Where("device_id = ?", deviceID).
		Scan(&row).Error; err != nil {
		t.Fatalf("query battery failed: %v", err)
	}
	return row.BleMac, row.IdentityBleMac
}

func TestAppUnbindClearsBleMacWhenNoAppAssociationsRemain(t *testing.T) {
	db := setupUnbindBleMacTestDB(t)
	insertUnbindBleMacUser(t, db, "user-1", "tenant-a", model.UserKindEndUser)
	insertUnbindBleMacBattery(t, db, "dev-1")
	insertUnbindBleMacBinding(t, db, "bind-1", "user-1", "dev-1")

	err := (&DeviceBinding{}).UnbindDevice(model.DeviceUnbindReq{DeviceID: "dev-1"}, &utils.UserClaims{
		ID:       "user-1",
		TenantID: "tenant-a",
	})
	if err != nil {
		t.Fatalf("UnbindDevice() error = %v", err)
	}
	bleMac, identityBleMac := queryUnbindBleMacBattery(t, db, "dev-1")
	if bleMac != nil {
		t.Fatalf("ble_mac = %v, want nil", *bleMac)
	}
	if identityBleMac == nil || *identityBleMac != "AC461A0F413D" {
		t.Fatalf("identity_ble_mac = %v, want AC461A0F413D", identityBleMac)
	}
}

func TestAppUnbindKeepsBleMacWhenOtherBindingRemains(t *testing.T) {
	db := setupUnbindBleMacTestDB(t)
	insertUnbindBleMacUser(t, db, "user-1", "tenant-a", model.UserKindEndUser)
	insertUnbindBleMacBattery(t, db, "dev-1")
	insertUnbindBleMacBinding(t, db, "bind-1", "user-1", "dev-1")
	insertUnbindBleMacBinding(t, db, "bind-2", "user-2", "dev-1")

	err := (&DeviceBinding{}).UnbindDevice(model.DeviceUnbindReq{DeviceID: "dev-1"}, &utils.UserClaims{
		ID:       "user-1",
		TenantID: "tenant-a",
	})
	if err != nil {
		t.Fatalf("UnbindDevice() error = %v", err)
	}
	bleMac, _ := queryUnbindBleMacBattery(t, db, "dev-1")
	if bleMac == nil || *bleMac != "EF533171C27F" {
		t.Fatalf("ble_mac = %v, want EF533171C27F", bleMac)
	}
}

func TestAppUnbindKeepsBleMacWhenOrgAddedRecordRemains(t *testing.T) {
	db := setupUnbindBleMacTestDB(t)
	insertUnbindBleMacUser(t, db, "user-1", "tenant-a", model.UserKindEndUser)
	insertUnbindBleMacBattery(t, db, "dev-1")
	insertUnbindBleMacBinding(t, db, "bind-1", "user-1", "dev-1")
	insertUnbindBleMacAddedRecord(t, db, "record-1", "tenant-a", "org-user-1", "dev-1")

	err := (&DeviceBinding{}).UnbindDevice(model.DeviceUnbindReq{DeviceID: "dev-1"}, &utils.UserClaims{
		ID:       "user-1",
		TenantID: "tenant-a",
	})
	if err != nil {
		t.Fatalf("UnbindDevice() error = %v", err)
	}
	bleMac, _ := queryUnbindBleMacBattery(t, db, "dev-1")
	if bleMac == nil || *bleMac != "EF533171C27F" {
		t.Fatalf("ble_mac = %v, want EF533171C27F", bleMac)
	}
}

func TestRemoveOrgAddedDeviceClearsBleMacWhenNoAppAssociationsRemain(t *testing.T) {
	db := setupUnbindBleMacTestDB(t)
	insertUnbindBleMacUser(t, db, "org-user-1", "tenant-a", model.UserKindOrgUser)
	insertUnbindBleMacBattery(t, db, "dev-1")
	insertUnbindBleMacAddedRecord(t, db, "record-1", "tenant-a", "org-user-1", "dev-1")

	err := (&DeviceBinding{}).RemoveOrgAddedDevice(model.AppDeviceRemoveReq{DeviceID: "dev-1"}, &utils.UserClaims{
		ID:        "org-user-1",
		TenantID:  "tenant-a",
		Authority: "TENANT_ADMIN",
	})
	if err != nil {
		t.Fatalf("RemoveOrgAddedDevice() error = %v", err)
	}
	bleMac, identityBleMac := queryUnbindBleMacBattery(t, db, "dev-1")
	if bleMac != nil {
		t.Fatalf("ble_mac = %v, want nil", *bleMac)
	}
	if identityBleMac == nil || *identityBleMac != "AC461A0F413D" {
		t.Fatalf("identity_ble_mac = %v, want AC461A0F413D", identityBleMac)
	}
}

func TestForceUnbindClearsBleMacWhenNoAppAssociationsRemain(t *testing.T) {
	db := setupUnbindBleMacTestDB(t)
	insertUnbindBleMacBattery(t, db, "dev-1")
	insertUnbindBleMacBinding(t, db, "bind-1", "user-1", "dev-1")

	err := (&EndUser{}).ForceUnbind(context.Background(), model.EndUserForceUnbindReq{BindingID: "bind-1"}, &utils.UserClaims{
		ID:       "admin-1",
		TenantID: "tenant-a",
	}, "")
	if err != nil {
		t.Fatalf("ForceUnbind() error = %v", err)
	}
	bleMac, identityBleMac := queryUnbindBleMacBattery(t, db, "dev-1")
	if bleMac != nil {
		t.Fatalf("ble_mac = %v, want nil", *bleMac)
	}
	if identityBleMac == nil || *identityBleMac != "AC461A0F413D" {
		t.Fatalf("identity_ble_mac = %v, want AC461A0F413D", identityBleMac)
	}
}
