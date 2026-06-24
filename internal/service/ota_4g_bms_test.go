package service

import (
	"context"
	"testing"
	"time"

	"project/internal/model"

	"gorm.io/gorm"
)

func insert4GBMSPackage(
	t *testing.T,
	db *gorm.DB,
	id string,
	version string,
	targetVersion *string,
	deviceKind int16,
	batteryModelID *string,
	batchNumber *string,
	itemUUID *string,
	createdAt time.Time,
) {
	t.Helper()
	if err := db.Exec(
		`INSERT INTO ota_upgrade_packages
			(id, name, version, target_version, package_type, signature_type, signature, additional_info, description, package_url, created_at, tenant_id, device_kind, battery_model_id, batch_number, item_uuid, module, remark)
		 VALUES (?, ?, ?, ?, 2, 'SHA256', 'abc123', '{"min_soc":30}', 'BMS package', ?, ?, 'tenant-a', ?, ?, ?, ?, 'BMS', 'direct ota')`,
		id, "pkg-"+id, version, targetVersion, "https://example.com/"+id+".bin", createdAt, deviceKind, batteryModelID, batchNumber, itemUUID,
	).Error; err != nil {
		t.Fatalf("insert 4g bms package failed: %v", err)
	}
}

func TestCheck4GBMSUpgradeTenantAndDeviceGate(t *testing.T) {
	setupAppBatteryOtaCheckTestDB(t)
	ctx := context.Background()
	ota := &OTA{}

	if _, err := ota.Check4GBMSUpgrade(ctx, &model.GetOTA4GBMSUpgradeCheckReq{Version: "10", ItemUUID: "item-a"}, ""); err == nil {
		t.Fatalf("expected tenant id error")
	}

	resp, err := ota.Check4GBMSUpgrade(ctx, &model.GetOTA4GBMSUpgradeCheckReq{Version: "10", ItemUUID: "missing"}, "tenant-a")
	if err != nil {
		t.Fatalf("check missing device failed: %v", err)
	}
	if resp.NeedUpgrade || resp.ItemUUID != "missing" || resp.CurrentVersion != "10" {
		t.Fatalf("expected no upgrade for missing device, got %#v", resp)
	}
}

func TestCheck4GBMSUpgradeUsesBMSPackageConstraints(t *testing.T) {
	db := setupAppBatteryOtaCheckTestDB(t)
	if err := db.Exec(`INSERT INTO devices (id, tenant_id, current_version) VALUES ('dev-1', 'tenant-a', '10')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, batch_number, item_uuid) VALUES ('dev-1', 'model-a', 'batch-a', 'item-a')`).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	target9 := "9"
	target10 := "10"
	insert4GBMSPackage(t, db, "pkg-4g", "99", nil, model.OTADeviceKind4GModule, nil, nil, nil, now.Add(5*time.Minute))
	insert4GBMSPackage(t, db, "pkg-target-mismatch", "98", &target9, model.OTADeviceKindBMS, nil, nil, nil, now.Add(4*time.Minute))
	insert4GBMSPackage(t, db, "pkg-generic", "97", nil, model.OTADeviceKindBMS, nil, nil, nil, now.Add(3*time.Minute))
	insert4GBMSPackage(t, db, "pkg-item", "96", nil, model.OTADeviceKindBMS, nil, nil, strPtr("item-a"), now.Add(2*time.Minute))
	insert4GBMSPackage(t, db, "pkg-three", "11", &target10, model.OTADeviceKindBMS, strPtr("model-a"), strPtr("batch-a"), strPtr("item-a"), now.Add(time.Minute))

	resp, err := new(OTA).Check4GBMSUpgrade(context.Background(), &model.GetOTA4GBMSUpgradeCheckReq{
		Version:  "10",
		ItemUUID: "item-a",
	}, "tenant-a")
	if err != nil {
		t.Fatalf("check 4g bms ota failed: %v", err)
	}
	if resp == nil || !resp.NeedUpgrade {
		t.Fatalf("expected upgrade response, got %#v", resp)
	}
	if resp.PackageID == nil || *resp.PackageID != "pkg-three" {
		t.Fatalf("expected highest constraint BMS package pkg-three, got %#v", resp)
	}
	if resp.Version == nil || *resp.Version != "11" {
		t.Fatalf("expected version 11, got %#v", resp)
	}
	if resp.FirmwareURL == nil || *resp.FirmwareURL != "https://example.com/pkg-three.bin" {
		t.Fatalf("expected package download url, got %#v", resp)
	}
	if resp.PackageType == nil || *resp.PackageType != 2 {
		t.Fatalf("expected package type 2, got %#v", resp)
	}
	if resp.SignatureType == nil || *resp.SignatureType != "SHA256" || resp.Signature == nil || *resp.Signature != "abc123" {
		t.Fatalf("expected signature fields, got %#v", resp)
	}
	if resp.AdditionalInfo == nil || *resp.AdditionalInfo != `{"min_soc":30}` {
		t.Fatalf("expected additional info, got %#v", resp)
	}
}
