package service

import (
	"strings"
	"testing"

	"project/internal/model"
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
