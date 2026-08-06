package service

import (
	"context"
	"fmt"
	"testing"

	"project/internal/model"
	"project/internal/query"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/glebarez/sqlite"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

func TestBatchTransferRequestValidation(t *testing.T) {
	validate := validator.New()
	validate.SetTagName("binding")
	tests := []struct {
		name    string
		req     model.BatteryBatchTransferReq
		wantErr bool
	}{
		{
			name:    "empty device ids",
			req:     model.BatteryBatchTransferReq{ToOrgID: "target-org"},
			wantErr: true,
		},
		{
			name:    "missing target org",
			req:     model.BatteryBatchTransferReq{DeviceIDs: []string{"device-1"}},
			wantErr: true,
		},
		{
			name:    "blank device id",
			req:     model.BatteryBatchTransferReq{DeviceIDs: []string{""}, ToOrgID: "target-org"},
			wantErr: true,
		},
		{
			name:    "valid request",
			req:     model.BatteryBatchTransferReq{DeviceIDs: []string{"device-1"}, ToOrgID: "target-org"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected valid request, got %v", err)
			}
		})
	}
}

func setupBatchTransferTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:batch-transfer-%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	schema := []string{
		`CREATE TABLE orgs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			org_type TEXT NOT NULL,
			parent_id TEXT,
			tenant_id TEXT NOT NULL
		)`,
		`CREATE TABLE org_closure (
			tenant_id TEXT NOT NULL,
			ancestor_id TEXT NOT NULL,
			descendant_id TEXT NOT NULL,
			depth INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (tenant_id, ancestor_id, descendant_id)
		)`,
		`CREATE TABLE devices (
			id TEXT PRIMARY KEY,
			voucher TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			is_enabled TEXT NOT NULL,
			activate_flag TEXT NOT NULL,
			device_number TEXT NOT NULL
		)`,
		`CREATE TABLE device_batteries (
			device_id TEXT PRIMARY KEY,
			owner_org_id TEXT,
			updated_at DATETIME
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
			description TEXT,
			extra TEXT,
			occurred_at DATETIME NOT NULL
		)`,
	}
	for _, statement := range schema {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create batch transfer test schema failed: %v", err)
		}
	}

	oldDB := global.DB
	global.DB = db
	query.SetDefault(db)
	t.Cleanup(func() {
		global.DB = oldDB
		if oldDB != nil {
			query.SetDefault(oldDB)
		}
	})
	return db
}

func seedBatchTransferOrg(t *testing.T, db *gorm.DB, id, name, orgType, tenantID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO orgs (id, name, org_type, tenant_id) VALUES (?, ?, ?, ?)",
		id,
		name,
		orgType,
		tenantID,
	).Error; err != nil {
		t.Fatalf("seed org %s failed: %v", id, err)
	}
}

func seedBatchTransferClosure(t *testing.T, db *gorm.DB, tenantID, ancestorID, descendantID string, depth int) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO org_closure (tenant_id, ancestor_id, descendant_id, depth) VALUES (?, ?, ?, ?)",
		tenantID,
		ancestorID,
		descendantID,
		depth,
	).Error; err != nil {
		t.Fatalf("seed org closure failed: %v", err)
	}
}

func seedBatchTransferDevice(t *testing.T, db *gorm.DB, id, number, tenantID, ownerOrgID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO devices (id, voucher, tenant_id, is_enabled, activate_flag, device_number) VALUES (?, ?, ?, ?, ?, ?)",
		id,
		"voucher-"+id,
		tenantID,
		"enabled",
		"inactive",
		number,
	).Error; err != nil {
		t.Fatalf("seed device %s failed: %v", id, err)
	}
	if err := db.Exec(
		"INSERT INTO device_batteries (device_id, owner_org_id) VALUES (?, ?)",
		id,
		ownerOrgID,
	).Error; err != nil {
		t.Fatalf("seed battery %s failed: %v", id, err)
	}
}

func TestBatchTransferBatteryDeduplicatesAndKeepsPartialFailures(t *testing.T) {
	db := setupBatchTransferTestDB(t)
	const tenantID = "tenant-a"
	seedBatchTransferOrg(t, db, "pack-a", "PACK A", model.OrgTypePACKFactory, tenantID)
	seedBatchTransferOrg(t, db, "dealer-a", "Dealer A", model.OrgTypeDealer, tenantID)
	seedBatchTransferDevice(t, db, "device-1", "BMS-001", tenantID, "pack-a")
	seedBatchTransferDevice(t, db, "device-2", "BMS-002", tenantID, "pack-a")
	seedBatchTransferDevice(t, db, "device-3", "BMS-003", tenantID, "dealer-a")

	resp, err := GroupApp.Battery.BatchTransferBattery(context.Background(), model.BatteryBatchTransferReq{
		DeviceIDs: []string{"device-1", "device-1", "device-2", "device-3", "missing"},
		ToOrgID:   "dealer-a",
	}, &utils.UserClaims{ID: "operator-a", TenantID: tenantID}, "")
	if err != nil {
		t.Fatalf("batch transfer returned request error: %v", err)
	}
	if resp.Total != 4 || resp.Success != 2 || resp.Failed != 2 {
		t.Fatalf("unexpected summary: %+v", resp)
	}
	if len(resp.Failures) != 2 {
		t.Fatalf("unexpected failures: %+v", resp.Failures)
	}
	if resp.Failures[0].DeviceID != "device-3" || resp.Failures[0].DeviceNumber != "BMS-003" {
		t.Fatalf("failure should keep device identity: %+v", resp.Failures[0])
	}
	if resp.Failures[0].Message != "厂家仅支持从 PACK 厂调拨" {
		t.Fatalf("failure should expose readable business reason: %+v", resp.Failures[0])
	}

	var owner string
	if err := db.Model(&model.DeviceBattery{}).Select("owner_org_id").Where("device_id = ?", "device-1").Scan(&owner).Error; err != nil {
		t.Fatalf("query transferred owner failed: %v", err)
	}
	if owner != "dealer-a" {
		t.Fatalf("device-1 owner = %q, want dealer-a", owner)
	}

	var transferCount int64
	if err := db.Model(&model.DeviceOrgTransfer{}).Where("device_id = ?", "device-1").Count(&transferCount).Error; err != nil {
		t.Fatalf("count transfer logs failed: %v", err)
	}
	if transferCount != 1 {
		t.Fatalf("duplicate device should transfer once, logs = %d", transferCount)
	}
}

func TestBatchTransferBatteryRoleMatrix(t *testing.T) {
	tests := []struct {
		name            string
		operatorOrgType string
		sourceOrgType   string
		targetOrgType   string
		wantSuccess     int
	}{
		{name: "pack to dealer", operatorOrgType: model.OrgTypePACKFactory, sourceOrgType: model.OrgTypePACKFactory, targetOrgType: model.OrgTypeDealer, wantSuccess: 1},
		{name: "pack cannot target pack", operatorOrgType: model.OrgTypePACKFactory, sourceOrgType: model.OrgTypePACKFactory, targetOrgType: model.OrgTypePACKFactory},
		{name: "dealer to store", operatorOrgType: model.OrgTypeDealer, sourceOrgType: model.OrgTypeDealer, targetOrgType: model.OrgTypeStore, wantSuccess: 1},
		{name: "dealer cannot target dealer", operatorOrgType: model.OrgTypeDealer, sourceOrgType: model.OrgTypeDealer, targetOrgType: model.OrgTypeDealer},
		{name: "store rejected", operatorOrgType: model.OrgTypeStore, sourceOrgType: model.OrgTypeStore, targetOrgType: model.OrgTypeDealer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupBatchTransferTestDB(t)
			const tenantID = "tenant-role"
			seedBatchTransferOrg(t, db, "operator-org", "Operator", tt.operatorOrgType, tenantID)
			targetID := "target-org"
			if tt.targetOrgType == tt.operatorOrgType {
				targetID = "operator-org"
			} else {
				seedBatchTransferOrg(t, db, targetID, "Target", tt.targetOrgType, tenantID)
			}
			seedBatchTransferClosure(t, db, tenantID, "operator-org", "operator-org", 0)
			if targetID != "operator-org" {
				seedBatchTransferClosure(t, db, tenantID, "operator-org", targetID, 1)
			}
			seedBatchTransferDevice(t, db, "role-device", "ROLE-001", tenantID, "operator-org")

			resp, err := GroupApp.Battery.BatchTransferBattery(context.Background(), model.BatteryBatchTransferReq{
				DeviceIDs: []string{"role-device"},
				ToOrgID:   targetID,
			}, &utils.UserClaims{ID: "operator-user", TenantID: tenantID}, "operator-org")
			if err != nil {
				t.Fatalf("batch transfer returned request error: %v", err)
			}
			if resp.Success != tt.wantSuccess || resp.Failed != 1-tt.wantSuccess {
				t.Fatalf("unexpected summary: %+v", resp)
			}
		})
	}
}

func TestBatchTransferBatteryFactoryRoleMatrix(t *testing.T) {
	tests := []struct {
		name        string
		targetType  string
		wantSuccess int
	}{
		{name: "factory to pack", targetType: model.OrgTypePACKFactory, wantSuccess: 1},
		{name: "factory to dealer", targetType: model.OrgTypeDealer, wantSuccess: 1},
		{name: "factory cannot target store", targetType: model.OrgTypeStore},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupBatchTransferTestDB(t)
			const tenantID = "tenant-factory"
			seedBatchTransferOrg(t, db, "source-pack", "Source PACK", model.OrgTypePACKFactory, tenantID)
			seedBatchTransferOrg(t, db, "target-org", "Target", tt.targetType, tenantID)
			seedBatchTransferDevice(t, db, "factory-device", "FACTORY-001", tenantID, "source-pack")

			resp, err := GroupApp.Battery.BatchTransferBattery(context.Background(), model.BatteryBatchTransferReq{
				DeviceIDs: []string{"factory-device"},
				ToOrgID:   "target-org",
			}, &utils.UserClaims{ID: "factory-user", TenantID: tenantID}, "")
			if err != nil {
				t.Fatalf("batch transfer returned request error: %v", err)
			}
			if resp.Success != tt.wantSuccess || resp.Failed != 1-tt.wantSuccess {
				t.Fatalf("unexpected summary: %+v", resp)
			}
		})
	}
}

func TestBatchTransferBatteryRejectsOutOfScopeAndUnmanufactured(t *testing.T) {
	db := setupBatchTransferTestDB(t)
	const tenantID = "tenant-scope"
	seedBatchTransferOrg(t, db, "operator-pack", "Operator PACK", model.OrgTypePACKFactory, tenantID)
	seedBatchTransferOrg(t, db, "outside-pack", "Outside PACK", model.OrgTypePACKFactory, tenantID)
	seedBatchTransferOrg(t, db, "target-dealer", "Target Dealer", model.OrgTypeDealer, tenantID)
	seedBatchTransferClosure(t, db, tenantID, "operator-pack", "operator-pack", 0)
	seedBatchTransferClosure(t, db, tenantID, "operator-pack", "target-dealer", 1)
	seedBatchTransferDevice(t, db, "outside-device", "OUTSIDE-001", tenantID, "outside-pack")
	if err := db.Exec(
		"INSERT INTO devices (id, voucher, tenant_id, is_enabled, activate_flag, device_number) VALUES (?, ?, ?, ?, ?, ?)",
		"unmanufactured-device",
		"voucher-unmanufactured",
		tenantID,
		"enabled",
		"inactive",
		"UNMANUFACTURED-001",
	).Error; err != nil {
		t.Fatalf("seed unmanufactured device failed: %v", err)
	}

	resp, err := GroupApp.Battery.BatchTransferBattery(context.Background(), model.BatteryBatchTransferReq{
		DeviceIDs: []string{"outside-device", "unmanufactured-device"},
		ToOrgID:   "target-dealer",
	}, &utils.UserClaims{ID: "operator-user", TenantID: tenantID}, "operator-pack")
	if err != nil {
		t.Fatalf("batch transfer returned request error: %v", err)
	}
	if resp.Success != 0 || resp.Failed != 2 || len(resp.Failures) != 2 {
		t.Fatalf("out-of-scope and unmanufactured devices should fail: %+v", resp)
	}
	if resp.Failures[0].DeviceNumber != "OUTSIDE-001" || resp.Failures[1].DeviceNumber != "UNMANUFACTURED-001" {
		t.Fatalf("failure details should include device numbers: %+v", resp.Failures)
	}
	if resp.Failures[0].Message != "无权操作该电池" || resp.Failures[1].Message != "设备未出厂，无法调拨" {
		t.Fatalf("failure details should include readable reasons: %+v", resp.Failures)
	}
}

func TestBatchTransferBatteryRejectsCrossTenantTarget(t *testing.T) {
	db := setupBatchTransferTestDB(t)
	seedBatchTransferOrg(t, db, "pack-a", "PACK A", model.OrgTypePACKFactory, "tenant-a")
	seedBatchTransferOrg(t, db, "dealer-b", "Dealer B", model.OrgTypeDealer, "tenant-b")
	seedBatchTransferDevice(t, db, "device-a", "TENANT-001", "tenant-a", "pack-a")

	resp, err := GroupApp.Battery.BatchTransferBattery(context.Background(), model.BatteryBatchTransferReq{
		DeviceIDs: []string{"device-a"},
		ToOrgID:   "dealer-b",
	}, &utils.UserClaims{ID: "operator-a", TenantID: "tenant-a"}, "")
	if err != nil {
		t.Fatalf("batch transfer returned request error: %v", err)
	}
	if resp.Success != 0 || resp.Failed != 1 || len(resp.Failures) != 1 {
		t.Fatalf("cross-tenant target should fail: %+v", resp)
	}
	if resp.Failures[0].Message != "组织不存在" {
		t.Fatalf("cross-tenant failure should not expose only an error code: %+v", resp.Failures[0])
	}
}
