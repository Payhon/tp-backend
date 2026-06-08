package bmsbridge

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupBridgeResolveTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	stmts := []string{
		`CREATE TABLE devices (
			id varchar(36) PRIMARY KEY,
			device_number varchar(100),
			tenant_id varchar(36)
		)`,
		`CREATE TABLE device_batteries (
			device_id varchar(36) PRIMARY KEY,
			item_uuid varchar(64),
			comm_chip_id varchar(64),
			imei varchar(32),
			iccid varchar(32)
		)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}
	return db
}

func TestResolvePlatformDeviceIDRequeriesAfterSameSNRecreate(t *testing.T) {
	db := setupBridgeResolveTestDB(t)
	const sn = "36011161145053593437373030124A57"

	if err := db.Exec(`INSERT INTO devices (id, device_number, tenant_id) VALUES ('old-dev', ?, 'tenant-a')`, sn).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, item_uuid, comm_chip_id) VALUES ('old-dev', ?, 'comm-old')`, sn).Error; err != nil {
		t.Fatal(err)
	}

	bridge := &Bridge{db: db}
	if got := bridge.resolvePlatformDeviceID(context.Background(), sn); got != "old-dev" {
		t.Fatalf("first resolve = %q, want old-dev", got)
	}

	// Simulate deleting the device row in the admin backend and recreating a new
	// device with the same SN while an old battery extension row is left behind.
	if err := db.Exec(`DELETE FROM devices WHERE id = 'old-dev'`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO devices (id, device_number, tenant_id) VALUES ('new-dev', ?, 'tenant-a')`, sn).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, item_uuid, comm_chip_id) VALUES ('new-dev', ?, 'comm-new')`, sn).Error; err != nil {
		t.Fatal(err)
	}

	if got := bridge.resolvePlatformDeviceID(context.Background(), sn); got != "new-dev" {
		t.Fatalf("resolve after recreate = %q, want new-dev", got)
	}
}
