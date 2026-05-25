package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	model "project/internal/model"
	global "project/pkg/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupOTA4GTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	oldDB := global.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	global.DB = db
	t.Cleanup(func() {
		global.DB = oldDB
	})

	stmts := []string{
		`CREATE TABLE devices (
			id varchar(36) PRIMARY KEY,
			tenant_id varchar(36) NOT NULL
		)`,
		`CREATE TABLE device_batteries (
				device_id varchar(36) PRIMARY KEY,
				comm_chip_id varchar(64),
				imei varchar(32)
			)`,
		`CREATE TABLE ota_upgrade_packages (
			id varchar(36) PRIMARY KEY,
			name varchar(200) NOT NULL,
			version varchar(36) NOT NULL,
			target_version varchar(36),
			device_config_id varchar(36) NOT NULL DEFAULT '',
			battery_model_id varchar(36),
			batch_number varchar(100),
			item_uuid varchar(64),
			module varchar(36),
			package_type int2 NOT NULL,
			signature_type varchar(36),
			additional_info text DEFAULT '{}',
			description varchar(500),
			package_url varchar(500),
			created_at datetime NOT NULL,
			updated_at datetime,
			remark varchar(255),
			signature varchar(255),
			tenant_id varchar(36),
			device_kind int2 NOT NULL DEFAULT 1,
			is_latest boolean NOT NULL DEFAULT false
		)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}
	return db
}

func insert4GPackage(t *testing.T, db *gorm.DB, id, version string, latest bool, createdAt time.Time) {
	t.Helper()
	if err := db.Exec(
		`INSERT INTO ota_upgrade_packages
			(id, name, version, package_type, package_url, created_at, tenant_id, device_kind, is_latest)
		 VALUES (?, ?, ?, 2, ?, ?, 'tenant-a', ?, ?)`,
		id, "pkg-"+id, version, "https://example.com/"+id+".bin", createdAt, model.OTADeviceKind4GModule, latest,
	).Error; err != nil {
		t.Fatalf("insert package failed: %v", err)
	}
}

func TestCheck4GModuleUpgrade(t *testing.T) {
	db := setupOTA4GTestDB(t)
	ctx := context.Background()
	ota := &OTA{}

	if _, err := ota.Check4GModuleUpgrade(ctx, &model.GetOTA4GModuleUpgradeCheckReq{Version: "1.0.0", Imei: "8600001"}, ""); err == nil {
		t.Fatalf("expected tenant header error")
	}

	resp, err := ota.Check4GModuleUpgrade(ctx, &model.GetOTA4GModuleUpgradeCheckReq{Version: "1.0.0", Imei: "missing"}, "tenant-a")
	if err != nil {
		t.Fatalf("check missing device failed: %v", err)
	}
	if resp.NeedUpgrade {
		t.Fatalf("missing device should not need upgrade")
	}

	if err := db.Exec(`INSERT INTO devices (id, tenant_id) VALUES ('dev-1', 'tenant-a')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, imei) VALUES ('dev-1', '8600001')`).Error; err != nil {
		t.Fatal(err)
	}

	insert4GPackage(t, db, "pkg-1", "1.0.1", false, time.Now().Add(-2*time.Hour))
	resp, err = ota.Check4GModuleUpgrade(ctx, &model.GetOTA4GModuleUpgradeCheckReq{Version: "1.0.0", Imei: "8600001"}, "tenant-a")
	if err != nil {
		t.Fatalf("check single package failed: %v", err)
	}
	if !resp.NeedUpgrade || resp.PackageID == nil || *resp.PackageID != "pkg-1" {
		t.Fatalf("expected single higher package pkg-1, got %#v", resp)
	}

	insert4GPackage(t, db, "pkg-2", "1.0.2", false, time.Now().Add(-1*time.Hour))
	resp, err = ota.Check4GModuleUpgrade(ctx, &model.GetOTA4GModuleUpgradeCheckReq{Version: "1.0.0", Imei: "8600001"}, "tenant-a")
	if err != nil {
		t.Fatalf("check multiple without latest failed: %v", err)
	}
	if resp.NeedUpgrade {
		t.Fatalf("multiple higher packages without latest should not need upgrade")
	}

	if err := db.Exec(`UPDATE ota_upgrade_packages SET is_latest = true WHERE id = 'pkg-2'`).Error; err != nil {
		t.Fatal(err)
	}
	resp, err = ota.Check4GModuleUpgrade(ctx, &model.GetOTA4GModuleUpgradeCheckReq{Version: "1.0.0", Imei: "8600001"}, "tenant-a")
	if err != nil {
		t.Fatalf("check latest package failed: %v", err)
	}
	if !resp.NeedUpgrade || resp.PackageID == nil || *resp.PackageID != "pkg-2" || !resp.IsLatest {
		t.Fatalf("expected latest package pkg-2, got %#v", resp)
	}

	resp, err = ota.Check4GModuleUpgrade(ctx, &model.GetOTA4GModuleUpgradeCheckReq{Version: "1.0.2", Imei: "8600001"}, "tenant-a")
	if err != nil {
		t.Fatalf("check same version failed: %v", err)
	}
	if resp.NeedUpgrade {
		t.Fatalf("same version should not need upgrade")
	}

	if err := db.Exec(`INSERT INTO devices (id, tenant_id) VALUES ('dev-2', 'tenant-a')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, comm_chip_id) VALUES ('dev-2', 'comm-8600002')`).Error; err != nil {
		t.Fatal(err)
	}
	resp, err = ota.Check4GModuleUpgrade(ctx, &model.GetOTA4GModuleUpgradeCheckReq{Version: "1.0.0", Imei: "comm-8600002"}, "tenant-a")
	if err != nil {
		t.Fatalf("check comm chip id fallback failed: %v", err)
	}
	if !resp.NeedUpgrade || resp.PackageID == nil || *resp.PackageID != "pkg-2" {
		t.Fatalf("expected comm_chip_id match to use latest package pkg-2, got %#v", resp)
	}
}

func TestCreate4GModulePackageLatestUniqueness(t *testing.T) {
	db := setupOTA4GTestDB(t)
	content := []byte("firmware")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	ota := &OTA{}
	kind := model.OTADeviceKind4GModule
	latest := true
	for i := 1; i <= 2; i++ {
		url := fmt.Sprintf("%s/%d.bin", server.URL, i)
		req := &model.CreateOTAUpgradePackageReq{
			Name:       fmt.Sprintf("4g-%d", i),
			Version:    fmt.Sprintf("1.0.%d", i),
			PackageUrl: &url,
			DeviceKind: &kind,
			IsLatest:   &latest,
		}
		if err := ota.CreateOTAUpgradePackage(req, "tenant-a"); err != nil {
			t.Fatalf("create package %d failed: %v", i, err)
		}
	}

	var latestCount int64
	if err := db.Table(model.TableNameOtaUpgradePackage).
		Where("tenant_id = ? AND device_kind = ? AND is_latest = ?", "tenant-a", model.OTADeviceKind4GModule, true).
		Count(&latestCount).Error; err != nil {
		t.Fatalf("count latest failed: %v", err)
	}
	if latestCount != 1 {
		t.Fatalf("expected one latest package, got %d", latestCount)
	}
}
