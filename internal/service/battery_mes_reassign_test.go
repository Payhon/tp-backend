package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"project/internal/model"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupMESPackFactoryReassignTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:mes-pack-reassign-%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	schema := []string{
		`CREATE TABLE devices (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			device_number TEXT NOT NULL,
			UNIQUE(tenant_id, device_number)
		)`,
		`CREATE TABLE orgs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			org_type TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			status TEXT
		)`,
		`CREATE TABLE device_batteries (
			device_id TEXT PRIMARY KEY,
			owner_org_id TEXT,
			pack_factory_org_id TEXT,
			bms_factory_org_id TEXT,
			activation_status TEXT,
			transfer_status TEXT,
			cell_brand_seq_no INTEGER,
			battery_model_seq_no INTEGER,
			updated_at DATETIME
		)`,
		`CREATE TABLE device_user_bindings (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			is_owner BOOLEAN
		)`,
		`CREATE TABLE device_org_transfers (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			from_org_id TEXT,
			to_org_id TEXT,
			operator_id TEXT,
			transfer_time DATETIME,
			remark TEXT,
			tenant_id TEXT NOT NULL,
			created_at DATETIME
		)`,
		`CREATE TABLE battery_operation_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			device_number TEXT NOT NULL,
			operation_type TEXT NOT NULL,
			operator_id TEXT,
			occurred_at DATETIME NOT NULL,
			description TEXT,
			extra JSON
		)`,
	}
	for _, ddl := range schema {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create MES reassign test schema failed: %v", err)
		}
	}

	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() {
		global.DB = oldDB
		_ = sqlDB.Close()
	})
	return db
}

func seedMESReassignOrg(t *testing.T, db *gorm.DB, id, name, orgType, tenantID, status string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO orgs (id, name, org_type, tenant_id, status) VALUES (?, ?, ?, ?, ?)",
		id,
		name,
		orgType,
		tenantID,
		status,
	).Error; err != nil {
		t.Fatalf("seed org %s failed: %v", id, err)
	}
}

func seedMESReassignDevice(
	t *testing.T,
	db *gorm.DB,
	id string,
	serialNumber string,
	tenantID string,
	ownerOrgID string,
	packFactoryOrgID *string,
	activationStatus string,
	transferStatus string,
	cellBrandSeqNo *int,
	batteryModelSeqNo *int,
) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO devices (id, tenant_id, device_number) VALUES (?, ?, ?)",
		id,
		tenantID,
		serialNumber,
	).Error; err != nil {
		t.Fatalf("seed device %s failed: %v", id, err)
	}
	if err := db.Exec(`
		INSERT INTO device_batteries (
			device_id, owner_org_id, pack_factory_org_id, activation_status,
			transfer_status, cell_brand_seq_no, battery_model_seq_no
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		id,
		ownerOrgID,
		packFactoryOrgID,
		activationStatus,
		transferStatus,
		cellBrandSeqNo,
		batteryModelSeqNo,
	).Error; err != nil {
		t.Fatalf("seed battery %s failed: %v", id, err)
	}
}

func strPtrForMESReassign(value string) *string {
	return &value
}

func intPtrForMESReassign(value int) *int {
	return &value
}

func TestNormalizeMESPackFactoryReassignSerials(t *testing.T) {
	got, err := normalizeMESPackFactoryReassignSerials([]string{" SN-1 ", "SN-1", "sn-1", "SN-2"})
	if err != nil {
		t.Fatalf("normalize serials failed: %v", err)
	}
	want := []string{"SN-1", "sn-1", "SN-2"}
	if len(got) != len(want) {
		t.Fatalf("normalized length = %d, want %d: %#v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("normalized[%d] = %q, want %q", index, got[index], want[index])
		}
	}

	if _, err := normalizeMESPackFactoryReassignSerials(nil); err == nil {
		t.Fatal("empty serial list should fail")
	}
	if _, err := normalizeMESPackFactoryReassignSerials([]string{"  "}); err == nil {
		t.Fatal("blank serial should fail")
	}
	tooMany := make([]string, 501)
	for index := range tooMany {
		tooMany[index] = fmt.Sprintf("SN-%03d", index)
	}
	if _, err := normalizeMESPackFactoryReassignSerials(tooMany); err == nil {
		t.Fatal("more than 500 serials should fail")
	}
}

func TestResolveMESReassignTargetPackFactory(t *testing.T) {
	const tenantID = "tenant-target"

	t.Run("normal unique pack", func(t *testing.T) {
		db := setupMESPackFactoryReassignTestDB(t)
		seedMESReassignOrg(t, db, "pack-target", "目标PACK", model.OrgTypePACKFactory, tenantID, model.OrgStatusNormal)
		org, err := resolveMESReassignTargetPackFactory(context.Background(), tenantID, " 目标PACK ")
		if err != nil {
			t.Fatalf("resolve target failed: %v", err)
		}
		if org.ID != "pack-target" {
			t.Fatalf("target org = %s, want pack-target", org.ID)
		}
	})

	t.Run("missing wrong type disabled duplicate and cross tenant", func(t *testing.T) {
		db := setupMESPackFactoryReassignTestDB(t)
		seedMESReassignOrg(t, db, "dealer", "经销商", model.OrgTypeDealer, tenantID, model.OrgStatusNormal)
		seedMESReassignOrg(t, db, "disabled", "禁用PACK", model.OrgTypePACKFactory, tenantID, model.OrgStatusDisabled)
		seedMESReassignOrg(t, db, "dup-1", "重名PACK", model.OrgTypePACKFactory, tenantID, model.OrgStatusNormal)
		seedMESReassignOrg(t, db, "dup-2", "重名PACK", model.OrgTypePACKFactory, tenantID, model.OrgStatusNormal)
		seedMESReassignOrg(t, db, "other-tenant", "跨租户PACK", model.OrgTypePACKFactory, "tenant-other", model.OrgStatusNormal)

		for _, name := range []string{"不存在", "经销商", "禁用PACK", "重名PACK", "跨租户PACK"} {
			if _, err := resolveMESReassignTargetPackFactory(context.Background(), tenantID, name); err == nil {
				t.Fatalf("target %q should fail", name)
			}
		}
	})
}

func TestReassignPackFactoryForMESPartialSuccessAndEligibility(t *testing.T) {
	db := setupMESPackFactoryReassignTestDB(t)
	const tenantID = "tenant-partial"
	seedMESReassignOrg(t, db, "pack-old", "原PACK", model.OrgTypePACKFactory, tenantID, model.OrgStatusDisabled)
	seedMESReassignOrg(t, db, "pack-target", "目标PACK", model.OrgTypePACKFactory, tenantID, model.OrgStatusNormal)
	seedMESReassignOrg(t, db, "dealer", "经销商", model.OrgTypeDealer, tenantID, model.OrgStatusNormal)

	seedMESReassignDevice(t, db, "device-ok", "SN-OK", tenantID, "pack-old", strPtrForMESReassign("pack-old"), "INACTIVE", "FACTORY", nil, nil)
	seedMESReassignDevice(t, db, "device-nil-pack", "SN-NIL-PACK", tenantID, "pack-old", nil, "INACTIVE", "FACTORY", nil, nil)
	seedMESReassignDevice(t, db, "device-unchanged", "SN-UNCHANGED", tenantID, "pack-target", strPtrForMESReassign("pack-target"), "ACTIVE", "USER", intPtrForMESReassign(1), intPtrForMESReassign(2))
	seedMESReassignDevice(t, db, "device-active", "SN-ACTIVE", tenantID, "pack-old", strPtrForMESReassign("pack-old"), "ACTIVE", "FACTORY", nil, nil)
	seedMESReassignDevice(t, db, "device-bad-activation", "SN-BAD-ACTIVATION", tenantID, "pack-old", strPtrForMESReassign("pack-old"), "UNKNOWN", "FACTORY", nil, nil)
	seedMESReassignDevice(t, db, "device-bound", "SN-BOUND", tenantID, "pack-old", strPtrForMESReassign("pack-old"), "INACTIVE", "FACTORY", nil, nil)
	seedMESReassignDevice(t, db, "device-complete", "SN-COMPLETE", tenantID, "pack-old", strPtrForMESReassign("pack-old"), "INACTIVE", "FACTORY", intPtrForMESReassign(1), nil)
	seedMESReassignDevice(t, db, "device-flowed", "SN-FLOWED", tenantID, "pack-old", strPtrForMESReassign("pack-old"), "INACTIVE", "DEALER", nil, nil)
	seedMESReassignDevice(t, db, "device-nonpack", "SN-NONPACK", tenantID, "dealer", nil, "INACTIVE", "FACTORY", nil, nil)
	seedMESReassignDevice(t, db, "device-mismatch", "SN-MISMATCH", tenantID, "pack-old", strPtrForMESReassign("pack-target"), "INACTIVE", "FACTORY", nil, nil)
	seedMESReassignDevice(t, db, "device-other-tenant", "SN-OTHER", "tenant-other", "pack-old", strPtrForMESReassign("pack-old"), "INACTIVE", "FACTORY", nil, nil)
	if err := db.Exec(
		"INSERT INTO devices (id, tenant_id, device_number) VALUES (?, ?, ?)",
		"device-not-shipped",
		tenantID,
		"SN-NOT-SHIPPED",
	).Error; err != nil {
		t.Fatalf("seed unshipped device failed: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO device_user_bindings (id, user_id, device_id, is_owner) VALUES (?, ?, ?, ?)",
		"binding-1",
		"user-1",
		"device-bound",
		true,
	).Error; err != nil {
		t.Fatalf("seed owner binding failed: %v", err)
	}

	resp, err := GroupApp.Battery.ReassignPackFactoryForMES(
		context.Background(),
		model.MESPackFactoryReassignReq{
			SerialNumbers: []string{
				"SN-OK",
				"SN-OK",
				"SN-NIL-PACK",
				"SN-UNCHANGED",
				"SN-ACTIVE",
				"SN-BAD-ACTIVATION",
				"SN-BOUND",
				"SN-COMPLETE",
				"SN-FLOWED",
				"SN-NONPACK",
				"SN-MISMATCH",
				"SN-NOT-SHIPPED",
				"SN-OTHER",
				"SN-MISSING",
			},
			TargetPackFactoryName: "目标PACK",
			Remark:                strPtrForMESReassign("  MES批量改派  "),
		},
		&utils.UserClaims{ID: "openapi-user", TenantID: tenantID},
		"open-key-1",
		"request-1",
	)
	if err != nil {
		t.Fatalf("batch reassign returned request error: %v", err)
	}
	if resp.Total != 13 || resp.Success != 2 || resp.Unchanged != 1 || resp.Failed != 10 {
		t.Fatalf("unexpected summary: %+v", resp)
	}
	if resp.RequestID != "request-1" || resp.TargetPackFactoryName != "目标PACK" {
		t.Fatalf("unexpected response metadata: %+v", resp)
	}
	if len(resp.Results) != 13 {
		t.Fatalf("result count = %d, want 13", len(resp.Results))
	}
	if resp.Results[0].Status != model.MESPackFactoryReassignStatusReassigned ||
		resp.Results[0].FromPackFactoryName == nil ||
		*resp.Results[0].FromPackFactoryName != "原PACK" {
		t.Fatalf("first result should expose resolved source pack: %+v", resp.Results[0])
	}
	if resp.Results[1].Status != model.MESPackFactoryReassignStatusReassigned {
		t.Fatalf("nil pack factory field should be initialized from current owner: %+v", resp.Results[1])
	}
	if resp.Results[2].Status != model.MESPackFactoryReassignStatusUnchanged {
		t.Fatalf("third unique result should be unchanged: %+v", resp.Results[2])
	}

	var updated model.DeviceBattery
	if err := db.Where("device_id = ?", "device-ok").Take(&updated).Error; err != nil {
		t.Fatalf("query updated battery failed: %v", err)
	}
	if updated.OwnerOrgID == nil || *updated.OwnerOrgID != "pack-target" ||
		updated.PackFactoryOrgID == nil || *updated.PackFactoryOrgID != "pack-target" {
		t.Fatalf("both pack fields must move together: %+v", updated)
	}

	var transferCount int64
	if err := db.Model(&model.DeviceOrgTransfer{}).Where("device_id = ?", "device-ok").Count(&transferCount).Error; err != nil {
		t.Fatalf("count transfer logs failed: %v", err)
	}
	if transferCount != 1 {
		t.Fatalf("successful unique device should have one transfer log, got %d", transferCount)
	}
	var unchangedTransferCount int64
	if err := db.Model(&model.DeviceOrgTransfer{}).Where("device_id = ?", "device-unchanged").Count(&unchangedTransferCount).Error; err != nil {
		t.Fatalf("count unchanged transfer logs failed: %v", err)
	}
	if unchangedTransferCount != 0 {
		t.Fatalf("unchanged device should not create transfer log, got %d", unchangedTransferCount)
	}

	var operation struct {
		OperationType string `gorm:"column:operation_type"`
		Description   string `gorm:"column:description"`
		Extra         string `gorm:"column:extra"`
	}
	if err := db.Table("battery_operation_logs").
		Where("device_id = ?", "device-ok").
		Take(&operation).Error; err != nil {
		t.Fatalf("query operation log failed: %v", err)
	}
	if operation.OperationType != BatteryOpTypePackFactoryReassign ||
		!strings.Contains(operation.Description, "原PACK -> 目标PACK") ||
		!strings.Contains(operation.Extra, "open-key-1") ||
		!strings.Contains(operation.Extra, "request-1") {
		t.Fatalf("unexpected operation audit: %+v", operation)
	}
}

func TestReassignPackFactoryForMESRollsBackWhenAuditFails(t *testing.T) {
	db := setupMESPackFactoryReassignTestDB(t)
	const tenantID = "tenant-rollback"
	seedMESReassignOrg(t, db, "pack-old", "原PACK", model.OrgTypePACKFactory, tenantID, model.OrgStatusNormal)
	seedMESReassignOrg(t, db, "pack-target", "目标PACK", model.OrgTypePACKFactory, tenantID, model.OrgStatusNormal)
	seedMESReassignDevice(t, db, "device-rollback", "SN-ROLLBACK", tenantID, "pack-old", strPtrForMESReassign("pack-old"), "INACTIVE", "FACTORY", nil, nil)
	if err := db.Exec("DROP TABLE battery_operation_logs").Error; err != nil {
		t.Fatalf("drop operation log table failed: %v", err)
	}

	resp, err := GroupApp.Battery.ReassignPackFactoryForMES(
		context.Background(),
		model.MESPackFactoryReassignReq{
			SerialNumbers:         []string{"SN-ROLLBACK"},
			TargetPackFactoryName: "目标PACK",
		},
		&utils.UserClaims{ID: "openapi-user", TenantID: tenantID},
		"open-key",
		"request-rollback",
	)
	if err != nil {
		t.Fatalf("request should return per-device failure: %v", err)
	}
	if resp.Failed != 1 || resp.Success != 0 {
		t.Fatalf("audit failure should fail the device: %+v", resp)
	}

	var battery model.DeviceBattery
	if err := db.Where("device_id = ?", "device-rollback").Take(&battery).Error; err != nil {
		t.Fatalf("query rolled back battery failed: %v", err)
	}
	if battery.OwnerOrgID == nil || *battery.OwnerOrgID != "pack-old" ||
		battery.PackFactoryOrgID == nil || *battery.PackFactoryOrgID != "pack-old" {
		t.Fatalf("battery assignment must roll back: %+v", battery)
	}
	var transferCount int64
	if err := db.Model(&model.DeviceOrgTransfer{}).Where("device_id = ?", "device-rollback").Count(&transferCount).Error; err != nil {
		t.Fatalf("count rolled back transfers failed: %v", err)
	}
	if transferCount != 0 {
		t.Fatalf("transfer log must roll back, got %d", transferCount)
	}
}

func TestReassignPackFactoryForMESConcurrentRetryCreatesOneTransfer(t *testing.T) {
	db := setupMESPackFactoryReassignTestDB(t)
	const tenantID = "tenant-concurrent"
	seedMESReassignOrg(t, db, "pack-old", "原PACK", model.OrgTypePACKFactory, tenantID, model.OrgStatusNormal)
	seedMESReassignOrg(t, db, "pack-target", "目标PACK", model.OrgTypePACKFactory, tenantID, model.OrgStatusNormal)
	seedMESReassignDevice(t, db, "device-concurrent", "SN-CONCURRENT", tenantID, "pack-old", strPtrForMESReassign("pack-old"), "INACTIVE", "FACTORY", nil, nil)

	req := model.MESPackFactoryReassignReq{
		SerialNumbers:         []string{"SN-CONCURRENT"},
		TargetPackFactoryName: "目标PACK",
	}
	claims := &utils.UserClaims{ID: "openapi-user", TenantID: tenantID}
	responses := make([]*model.MESPackFactoryReassignResp, 2)
	errors := make([]error, 2)
	var wg sync.WaitGroup
	for index := range responses {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			responses[index], errors[index] = GroupApp.Battery.ReassignPackFactoryForMES(
				context.Background(),
				req,
				claims,
				"open-key",
				fmt.Sprintf("request-%d", index),
			)
		}(index)
	}
	wg.Wait()
	for index, err := range errors {
		if err != nil {
			t.Fatalf("concurrent request %d failed: %v", index, err)
		}
	}
	totalSuccess := responses[0].Success + responses[1].Success
	totalUnchanged := responses[0].Unchanged + responses[1].Unchanged
	if totalSuccess != 1 || totalUnchanged != 1 {
		t.Fatalf("concurrent retry should produce one transfer and one unchanged: %+v / %+v", responses[0], responses[1])
	}
	var transferCount int64
	if err := db.Model(&model.DeviceOrgTransfer{}).Where("device_id = ?", "device-concurrent").Count(&transferCount).Error; err != nil {
		t.Fatalf("count concurrent transfers failed: %v", err)
	}
	if transferCount != 1 {
		t.Fatalf("concurrent retry created %d transfer logs, want 1", transferCount)
	}
}
