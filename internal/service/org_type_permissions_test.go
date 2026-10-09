package service

import (
	"context"
	"strings"
	"testing"

	"project/internal/model"
	"project/pkg/utils"
)

func TestNormalizeDeviceParamPermissionKeys(t *testing.T) {
	input := []string{
		"403:feedback_delay",
		"403",
		"40E:alarm_release_v",
		"410:alarm_release",
		"410:protect_release",
		"42A:large_delay",
		"43B:protect_release_delay",
		"44b:protect_release",
		"factory:entertestmode",
		"factory:erasecurrentparams",
		"factory:disablecharge",
		"factory:disabledischarge",
		"factory:allowchargedischarge",
		"function:chargeallowed",
		"function:dischargeallowed",
	}

	got := normalizeDeviceParamPermissionKeys(input)
	want := []string{
		"403",
		"40e",
		"410",
		"42a",
		"43b",
		"44b",
		"factory:enterTestMode",
		"factory:eraseCurrentParams",
		"factory:disableCharge",
		"factory:disableDischarge",
		"factory:allowChargeDischarge",
		"function:chargeAllowed",
		"function:dischargeAllowed",
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected length: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected item at %d: got=%q want=%q (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestMobileProtectionStatusPermissionIsolation(t *testing.T) {
	db := setupDeviceLastConnectedTestDB(t)
	for _, statement := range []string{
		`CREATE TABLE users (id TEXT PRIMARY KEY, tenant_id TEXT, user_kind TEXT, org_id TEXT)`,
		`CREATE TABLE orgs (id TEXT PRIMARY KEY, tenant_id TEXT, org_type TEXT)`,
		`CREATE TABLE org_type_permissions (tenant_id TEXT, org_type TEXT, ui_codes TEXT, device_param_permissions TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE sys_ui_elements (id TEXT PRIMARY KEY, parent_id TEXT, element_code TEXT, element_type INTEGER)`,
		`INSERT INTO users VALUES ('end', 'tenant', 'END_USER', 'pack'), ('org', 'tenant', 'ORG_USER', 'pack')`,
		`INSERT INTO orgs VALUES ('pack', 'tenant', 'PACK_FACTORY')`,
		`INSERT INTO sys_ui_elements VALUES ('mobile', '0', 'app_mobile_permissions', 2), ('history', 'mobile', 'app_device_detail_history', 4), ('protection', 'mobile', 'app_device_detail_protection_status', 4), ('web', '0', 'unrelated_web_permission', 4)`,
		`INSERT INTO org_type_permissions (tenant_id, org_type, ui_codes) VALUES ('tenant', 'APP_USER', '["app_device_detail_history","unrelated_web_permission"]'), ('tenant', 'PACK_FACTORY', '["app_device_detail_protection_status"]')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare permission test: %v", err)
		}
	}
	check := func(userID, orgType, want string) {
		t.Helper()
		resp, err := new(OrgTypePermission).GetCurrentMobileUIPermissions(context.Background(), &utils.UserClaims{
			ID: userID, TenantID: "tenant",
		})
		if err != nil {
			t.Fatalf("read %s permissions: %v", userID, err)
		}
		if resp.AllowAll || resp.OrgType != orgType || strings.Join(resp.UICodes, ",") != want {
			t.Fatalf("permissions for %s = %+v, want %s / %q", userID, resp, orgType, want)
		}
	}
	check("end", "APP_USER", "app_device_detail_history")
	check("org", "PACK_FACTORY", "app_device_detail_protection_status")
	if err := db.Exec(`UPDATE org_type_permissions SET ui_codes = CASE org_type WHEN 'APP_USER' THEN '["app_device_detail_history","app_device_detail_protection_status"]' ELSE '[]' END`).Error; err != nil {
		t.Fatal(err)
	}
	check("end", "APP_USER", "app_device_detail_history,app_device_detail_protection_status")
	check("org", "PACK_FACTORY", "")
	if err := db.Exec(`DELETE FROM org_type_permissions`).Error; err != nil {
		t.Fatal(err)
	}
	check("end", "APP_USER", "")
	check("org", "PACK_FACTORY", "")
}

func TestBuildDeviceParamPermissionTreeUsesCanonicalKeys(t *testing.T) {
	tree := buildDeviceParamPermissionTree()
	seen := map[string]string{}

	var walk func(nodes []model.DeviceParamTreeNode)
	walk = func(nodes []model.DeviceParamTreeNode) {
		for _, node := range nodes {
			if len(node.Children) > 0 {
				walk(node.Children)
				continue
			}
			key := node.Value
			if key == "" {
				continue
			}
			if strings.Contains(key, ":") && !strings.HasPrefix(key, "factory:") && !strings.HasPrefix(key, "function:") {
				t.Fatalf("unexpected non-canonical device param key %q on node %q", key, node.Label)
			}
			if prev, ok := seen[key]; ok {
				t.Fatalf("duplicate key %q on nodes %q and %q", key, prev, node.Label)
			}
			seen[key] = node.Label
		}
	}

	walk(tree)

	expectedParamLabels := map[string]string{
		"40b": "低温单体过放告警电压",
		"40c": "低温单体过放保护电压",
	}
	for key, wantLabel := range expectedParamLabels {
		if label, ok := seen[key]; !ok {
			t.Fatalf("missing device param permission key %s", key)
		} else if label != wantLabel {
			t.Fatalf("unexpected device param label for %s: got=%q want=%q", key, label, wantLabel)
		}
	}

	if label, ok := seen["10d"]; !ok {
		t.Fatalf("missing readonly SOH permission key 10d")
	} else if label != "只读 SOH" {
		t.Fatalf("unexpected readonly SOH label: got=%q want=%q", label, "只读 SOH")
	}

	expectedFactoryLabels := map[string]string{
		"factory:disableCharge":        "禁止充电",
		"factory:disableDischarge":     "禁止放电",
		"factory:allowChargeDischarge": "允许充放电",
	}
	for key, wantLabel := range expectedFactoryLabels {
		if label, ok := seen[key]; !ok {
			t.Fatalf("missing factory permission key %s", key)
		} else if label != wantLabel {
			t.Fatalf("unexpected factory label for %s: got=%q want=%q", key, label, wantLabel)
		}
	}
}
