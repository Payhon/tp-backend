package service

import (
	"context"
	"testing"
	"time"

	"project/internal/model"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func strPtr(s string) *string {
	return &s
}

func otaMatchPkg(id, version string, createdAt time.Time, batteryModelID, batchNumber, itemUUID *string) model.OtaUpgradePackage {
	return model.OtaUpgradePackage{
		ID:             id,
		Name:           id,
		Version:        version,
		CreatedAt:      createdAt,
		PackageType:    2,
		BatteryModelID: batteryModelID,
		BatchNumber:    batchNumber,
		ItemUUID:       itemUUID,
	}
}

func TestSelectAppBatteryOtaPackageConstraintPriority(t *testing.T) {
	now := time.Now()
	criteria := appBatteryOtaMatchCriteria{
		BatteryModelID: "model-a",
		BatchNumber:    "batch-a",
		ItemUUID:       "item-a",
	}
	packages := []model.OtaUpgradePackage{
		otaMatchPkg("generic", "99", now.Add(4*time.Minute), nil, nil, nil),
		otaMatchPkg("item", "98", now.Add(3*time.Minute), nil, nil, strPtr("item-a")),
		otaMatchPkg("two", "97", now.Add(2*time.Minute), strPtr("model-a"), strPtr("batch-a"), nil),
		otaMatchPkg("three", "11", now.Add(time.Minute), strPtr("model-a"), strPtr("batch-a"), strPtr("item-a")),
	}

	got := selectAppBatteryOtaPackage(packages, "10", criteria)
	if got == nil || got.ID != "three" {
		t.Fatalf("expected three-field package, got %#v", got)
	}
}

func TestSelectAppBatteryOtaPackageSingleConstraintPriority(t *testing.T) {
	now := time.Now()
	criteria := appBatteryOtaMatchCriteria{
		BatteryModelID: "model-a",
		BatchNumber:    "batch-a",
		ItemUUID:       "item-a",
	}
	packages := []model.OtaUpgradePackage{
		otaMatchPkg("batch", "99", now.Add(3*time.Minute), nil, strPtr("batch-a"), nil),
		otaMatchPkg("model", "98", now.Add(2*time.Minute), strPtr("model-a"), nil, nil),
		otaMatchPkg("item", "11", now.Add(time.Minute), nil, nil, strPtr("item-a")),
	}

	got := selectAppBatteryOtaPackage(packages, "10", criteria)
	if got == nil || got.ID != "item" {
		t.Fatalf("expected item_uuid package, got %#v", got)
	}
}

func TestSelectAppBatteryOtaPackageGenericFallbackAndMismatch(t *testing.T) {
	now := time.Now()
	criteria := appBatteryOtaMatchCriteria{
		BatteryModelID: "model-a",
		BatchNumber:    "batch-a",
		ItemUUID:       "item-a",
	}
	packages := []model.OtaUpgradePackage{
		otaMatchPkg("mismatch", "99", now.Add(2*time.Minute), nil, nil, strPtr("item-b")),
		otaMatchPkg("generic", "11", now.Add(time.Minute), nil, nil, nil),
	}

	got := selectAppBatteryOtaPackage(packages, "10", criteria)
	if got == nil || got.ID != "generic" {
		t.Fatalf("expected generic package fallback, got %#v", got)
	}
}

func TestSelectAppBatteryOtaPackageTargetVersionAndVersionGate(t *testing.T) {
	now := time.Now()
	target9 := "9"
	target10 := "10"
	packages := []model.OtaUpgradePackage{
		func() model.OtaUpgradePackage {
			p := otaMatchPkg("target-mismatch", "99", now.Add(3*time.Minute), nil, nil, nil)
			p.TargetVersion = &target9
			return p
		}(),
		otaMatchPkg("not-newer", "10", now.Add(2*time.Minute), nil, nil, nil),
		func() model.OtaUpgradePackage {
			p := otaMatchPkg("target-match", "11", now.Add(time.Minute), nil, nil, nil)
			p.TargetVersion = &target10
			return p
		}(),
	}

	got := selectAppBatteryOtaPackage(packages, "10", appBatteryOtaMatchCriteria{})
	if got == nil || got.ID != "target-match" {
		t.Fatalf("expected target-version matched newer package, got %#v", got)
	}
}

func setupAppBatteryOtaCheckTestDB(t *testing.T) *gorm.DB {
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
			tenant_id varchar(36) NOT NULL,
			current_version varchar(36)
		)`,
		`CREATE TABLE device_batteries (
			device_id varchar(36) PRIMARY KEY,
			battery_model_id varchar(36),
			batch_number varchar(100),
			item_uuid varchar(64)
		)`,
		`CREATE TABLE battery_bms_models (
			id varchar(36) PRIMARY KEY,
			name varchar(200),
			tenant_id varchar(36) NOT NULL
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

func TestCheckBatteryOtaForAppSkipsDevicePermissionCheck(t *testing.T) {
	db := setupAppBatteryOtaCheckTestDB(t)
	if err := db.Exec(`INSERT INTO devices (id, tenant_id, current_version) VALUES ('dev-1', 'tenant-a', '10')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, batch_number, item_uuid) VALUES ('dev-1', 'model-a', 'batch-a', 'item-a')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(
		`INSERT INTO ota_upgrade_packages
			(id, name, version, package_type, package_url, created_at, tenant_id, device_kind, battery_model_id, batch_number, item_uuid)
		 VALUES ('pkg-1', 'pkg-1', '11', 2, 'https://example.com/pkg-1.bin', ?, 'tenant-a', ?, 'model-a', 'batch-a', 'item-a')`,
		time.Now(), model.OTADeviceKindBMS,
	).Error; err != nil {
		t.Fatal(err)
	}

	resp, err := new(AppBattery).CheckBatteryOtaForApp(context.Background(), model.AppBatteryOtaCheckReq{
		DeviceID: "dev-1",
		Version:  strPtr("10"),
		ItemUUID: strPtr("item-a"),
	}, &utils.UserClaims{
		ID:        "org-user-without-binding",
		TenantID:  "tenant-a",
		Authority: "TENANT_USER",
	})
	if err != nil {
		t.Fatalf("check ota failed: %v", err)
	}
	if resp == nil || !resp.NeedUpgrade {
		t.Fatalf("expected upgrade response, got %#v", resp)
	}
	if resp.PackageID == nil || *resp.PackageID != "pkg-1" {
		t.Fatalf("expected pkg-1, got %#v", resp)
	}
}

func TestCheckBatteryOtaForAppResolvesInstrumentUUIDConstraints(t *testing.T) {
	db := setupAppBatteryOtaCheckTestDB(t)
	if err := db.Exec(`INSERT INTO devices (id, tenant_id, current_version) VALUES ('dev-meter-bms', 'tenant-a', '8')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO battery_bms_models (id, name, tenant_id) VALUES ('model-a', 'FJ-BMS-A', 'tenant-a')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, batch_number, item_uuid) VALUES ('dev-meter-bms', 'model-a', 'batch-a', 'AABBCCDDEEFF00112233445566778899')`).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.Exec(
		`INSERT INTO ota_upgrade_packages
			(id, name, version, package_type, package_url, created_at, tenant_id, device_kind)
		 VALUES ('generic', 'generic', '99', 2, 'generic.bin', ?, 'tenant-a', ?)`,
		now.Add(time.Minute), model.OTADeviceKindBMS,
	).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(
		`INSERT INTO ota_upgrade_packages
			(id, name, version, package_type, package_url, created_at, tenant_id, device_kind, battery_model_id, batch_number, item_uuid)
		 VALUES ('specific', 'specific', '11', 2, 'specific.bin', ?, 'tenant-a', ?, 'model-a', 'batch-a', 'AABBCCDDEEFF00112233445566778899')`,
		now, model.OTADeviceKindBMS,
	).Error; err != nil {
		t.Fatal(err)
	}

	version := "10"
	itemUUID := "aabbccddeeff00112233445566778899"
	resp, err := new(AppBattery).CheckBatteryOtaForApp(context.Background(), model.AppBatteryOtaCheckReq{
		Version:  &version,
		ItemUUID: &itemUUID,
	}, &utils.UserClaims{ID: "user-a", TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("check ota failed: %v", err)
	}
	if resp == nil || !resp.NeedUpgrade || resp.PackageID == nil || *resp.PackageID != "specific" {
		t.Fatalf("expected UUID-resolved specific package, got %#v", resp)
	}
	if resp.DeviceID != "dev-meter-bms" {
		t.Fatalf("expected resolved device id, got %#v", resp)
	}
}

func TestCheckBatteryOtaForAppMatchesUnregisteredInstrumentUUID(t *testing.T) {
	db := setupAppBatteryOtaCheckTestDB(t)
	if err := db.Exec(
		`INSERT INTO ota_upgrade_packages
			(id, name, version, package_type, package_url, created_at, tenant_id, device_kind, item_uuid)
		 VALUES ('uuid-only', 'uuid-only', '11', 2, 'uuid.bin', ?, 'tenant-a', ?, '00112233445566778899AABBCCDDEEFF')`,
		time.Now(), model.OTADeviceKindBMS,
	).Error; err != nil {
		t.Fatal(err)
	}

	version := "10"
	itemUUID := "00112233445566778899aabbccddeeff"
	resp, err := new(AppBattery).CheckBatteryOtaForApp(context.Background(), model.AppBatteryOtaCheckReq{
		Version:  &version,
		ItemUUID: &itemUUID,
	}, &utils.UserClaims{ID: "user-a", TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("check ota failed: %v", err)
	}
	if resp == nil || !resp.NeedUpgrade || resp.PackageID == nil || *resp.PackageID != "uuid-only" {
		t.Fatalf("expected UUID-only package, got %#v", resp)
	}
}

func TestCheckBatteryOtaForAppFallsBackToTenantModelName(t *testing.T) {
	db := setupAppBatteryOtaCheckTestDB(t)
	if err := db.Exec(`INSERT INTO battery_bms_models (id, name, tenant_id) VALUES ('model-a', 'FJ-BMS-A', 'tenant-a'), ('model-b', 'FJ-BMS-A', 'tenant-b')`).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.Exec(
		`INSERT INTO ota_upgrade_packages
			(id, name, version, package_type, package_url, created_at, tenant_id, device_kind, battery_model_id)
		 VALUES ('tenant-model', 'tenant-model', '11', 2, 'model.bin', ?, 'tenant-a', ?, 'model-a')`,
		now, model.OTADeviceKindBMS,
	).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(
		`INSERT INTO ota_upgrade_packages
			(id, name, version, package_type, package_url, created_at, tenant_id, device_kind, battery_model_id)
		 VALUES ('other-tenant', 'other-tenant', '99', 2, 'other.bin', ?, 'tenant-b', ?, 'model-b')`,
		now.Add(time.Minute), model.OTADeviceKindBMS,
	).Error; err != nil {
		t.Fatal(err)
	}

	version := "10"
	modelName := "fj-bms-a"
	resp, err := new(AppBattery).CheckBatteryOtaForApp(context.Background(), model.AppBatteryOtaCheckReq{
		Version: &version,
		Model:   &modelName,
	}, &utils.UserClaims{ID: "user-a", TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("check ota failed: %v", err)
	}
	if resp == nil || !resp.NeedUpgrade || resp.PackageID == nil || *resp.PackageID != "tenant-model" {
		t.Fatalf("expected current-tenant model package, got %#v", resp)
	}
}
