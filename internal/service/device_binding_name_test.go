package service

import (
	"context"
	"testing"

	"project/internal/model"
	"project/pkg/utils"
)

func TestBoundDeviceNameResetsAfterUnbind(t *testing.T) {
	db := setupDeviceLastConnectedTestDB(t)
	for _, statement := range []string{
		`CREATE TABLE users (id TEXT PRIMARY KEY, tenant_id TEXT, user_kind TEXT, org_id TEXT, name TEXT, phone_number TEXT)`,
		`CREATE TABLE device_user_bindings (id TEXT PRIMARY KEY, user_id TEXT, device_id TEXT, binding_time DATETIME, is_owner BOOLEAN, remark TEXT, custom_name TEXT)`,
		`CREATE TABLE app_device_added_records (id TEXT PRIMARY KEY, device_id TEXT)`,
		`ALTER TABLE device_batteries ADD COLUMN iccid TEXT`,
		`ALTER TABLE device_batteries ADD COLUMN owner_org_id TEXT`,
		`ALTER TABLE device_batteries ADD COLUMN activation_status TEXT`,
		`ALTER TABLE device_batteries ADD COLUMN activation_date DATETIME`,
		`ALTER TABLE device_batteries ADD COLUMN transfer_status TEXT`,
		`ALTER TABLE device_batteries ADD COLUMN warranty_months INTEGER`,
		`ALTER TABLE device_batteries ADD COLUMN warranty_start_date DATETIME`,
		`ALTER TABLE device_batteries ADD COLUMN warranty_manual_override BOOLEAN`,
		`ALTER TABLE device_batteries ADD COLUMN warranty_updated_at DATETIME`,
		`ALTER TABLE device_batteries ADD COLUMN warranty_updated_by TEXT`,
		`ALTER TABLE battery_bms_models ADD COLUMN warranty_months INTEGER`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create name test schema: %v", err)
		}
	}
	for _, statement := range []string{
		`INSERT INTO users (id, tenant_id, user_kind, phone_number) VALUES ('user-a', 'tenant-a', 'END_USER', '100'), ('user-b', 'tenant-a', 'END_USER', '200'), ('user-c', 'tenant-a', 'END_USER', '300')`,
		`INSERT INTO devices (id, device_number, name, tenant_id) VALUES ('device-1', 'BMS001', '原始名称', 'tenant-a')`,
		`INSERT INTO device_batteries (device_id, activation_status, transfer_status) VALUES ('device-1', 'ACTIVE', 'USER')`,
		`INSERT INTO device_user_bindings (id, user_id, device_id, is_owner, remark) VALUES ('bind-a', 'user-a', 'device-1', true, '激活业务备注'), ('bind-b', 'user-b', 'device-1', false, NULL)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("seed name test: %v", err)
		}
	}

	claimsA := &utils.UserClaims{ID: "user-a", TenantID: "tenant-a"}
	claimsB := &utils.UserClaims{ID: "user-b", TenantID: "tenant-a"}
	name := "我的电池"
	if _, err := new(Device).UpdateDevice(model.UpdateDeviceReq{Id: "device-1", Name: &name}, claimsA); err != nil {
		t.Fatalf("rename bound device: %v", err)
	}
	if _, err := new(Device).UpdateDevice(model.UpdateDeviceReq{Id: "device-1", Name: &name}, &utils.UserClaims{ID: "user-c", TenantID: "tenant-a"}); err == nil {
		t.Fatal("unbound user renamed device")
	}

	var baseName string
	if err := db.Table("devices").Select("name").Where("id = ?", "device-1").Scan(&baseName).Error; err != nil {
		t.Fatalf("read base name: %v", err)
	}
	if baseName != "原始名称" {
		t.Fatalf("base name = %q, want 原始名称", baseName)
	}
	var remark string
	if err := db.Table("device_user_bindings").Select("remark").Where("id = ?", "bind-a").Scan(&remark).Error; err != nil || remark != "激活业务备注" {
		t.Fatalf("binding remark = %q err=%v, want 激活业务备注", remark, err)
	}
	checkName := func(claims *utils.UserClaims, want string) {
		t.Helper()
		list, err := new(DeviceBinding).GetUserDevices(model.DeviceUserBindingListReq{Page: 1, PageSize: 20}, claims)
		if err != nil || len(list.List) != 1 || list.List[0].DeviceName != want {
			t.Fatalf("list name for %s: response=%+v err=%v, want %q", claims.ID, list, err, want)
		}
		detail, err := new(AppBattery).GetBatteryDetailForApp(context.Background(), "device-1", claims)
		if err != nil || detail.DeviceName == nil || *detail.DeviceName != want {
			t.Fatalf("detail name for %s: response=%+v err=%v, want %q", claims.ID, detail, err, want)
		}
	}
	checkName(claimsA, name)
	checkName(claimsB, "原始名称")

	if err := new(DeviceBinding).UnbindDevice(model.DeviceUnbindReq{DeviceID: "device-1"}, claimsA); err != nil {
		t.Fatalf("unbind device: %v", err)
	}
	if err := new(DeviceBinding).BindDevice(model.DeviceBindReq{DeviceNumber: "BMS001"}, claimsA); err != nil {
		t.Fatalf("rebind device: %v", err)
	}
	var rebound struct{ CustomName *string }
	if err := db.Table("device_user_bindings").Select("custom_name").Where("user_id = ? AND device_id = ?", claimsA.ID, "device-1").Scan(&rebound).Error; err != nil {
		t.Fatalf("read rebound name: %v", err)
	}
	if rebound.CustomName != nil {
		t.Fatalf("rebound custom name = %q, want nil", *rebound.CustomName)
	}
	checkName(claimsA, "原始名称")
	checkName(claimsB, "原始名称")
}
