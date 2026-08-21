package service

import "testing"

func TestFilterBMSHistoryWideColumnRows(t *testing.T) {
	columnRows := []bmsHistoryWideColumnRow{
		{DataType: "telemetry", DataIdentifier: "balancingOn", DataName: "balancingOn"},
		{DataType: "telemetry", DataIdentifier: "bms.snapshot", DataName: "bms.snapshot"},
		{DataType: "telemetry", DataIdentifier: "protectCount", DataName: "protectCount"},
		{DataType: "telemetry", DataIdentifier: "protectOn", DataName: "protectOn"},
		{DataType: "telemetry", DataIdentifier: "vPackV", DataName: "Pack(V)"},
		{DataType: "telemetry", DataIdentifier: "cell.voltagesMv", DataName: "cell.voltagesMv"},
		{DataType: "telemetry", DataIdentifier: "dynamicObject", DataName: "dynamicObject"},
		{DataType: "telemetry", DataIdentifier: "dynamicArray", DataName: "dynamicArray"},
		{DataType: "telemetry", DataIdentifier: "packCellSumVoltageV", DataName: "pack电体Sum电压"},
		{DataType: "telemetry", DataIdentifier: "soc", DataName: "SOC(%)"},
		{DataType: "telemetry", DataIdentifier: "soh", DataName: "SOH(%)"},
		{DataType: "telemetry", DataIdentifier: "currentA", DataName: "电流(A)"},
		{DataType: "telemetry", DataIdentifier: "plainText", DataName: "plainText"},
		{DataType: "telemetry", DataIdentifier: "boolText", DataName: "boolText"},
	}
	valueRows := []bmsHistoryWideValueRow{
		{DataType: "telemetry", DataIdentifier: "dynamicObject", Value: `{"cell":{"balancing":[false]}}`},
		{DataType: "telemetry", DataIdentifier: "dynamicArray", Value: `[1,2,3]`},
		{DataType: "telemetry", DataIdentifier: "packCellSumVoltageV", Value: "68.3"},
		{DataType: "telemetry", DataIdentifier: "soc", Value: "77"},
		{DataType: "telemetry", DataIdentifier: "soh", Value: "100"},
		{DataType: "telemetry", DataIdentifier: "currentA", Value: "0"},
		{DataType: "telemetry", DataIdentifier: "plainText", Value: "not-json"},
		{DataType: "telemetry", DataIdentifier: "boolText", Value: "false"},
	}

	filtered := filterBMSHistoryWideColumnRows(columnRows, valueRows)

	got := make(map[string]struct{}, len(filtered))
	for _, row := range filtered {
		got[row.DataIdentifier] = struct{}{}
	}

	for _, identifier := range []string{
		"balancingOn",
		"bms.snapshot",
		"protectCount",
		"protectOn",
		"vPackV",
		"cell.voltagesMv",
		"dynamicObject",
		"dynamicArray",
	} {
		if _, ok := got[identifier]; ok {
			t.Fatalf("expected %s to be excluded, got filtered columns %#v", identifier, filtered)
		}
	}

	for _, identifier := range []string{
		"packCellSumVoltageV",
		"soc",
		"soh",
		"currentA",
		"plainText",
		"boolText",
	} {
		if _, ok := got[identifier]; !ok {
			t.Fatalf("expected %s to be retained, got filtered columns %#v", identifier, filtered)
		}
	}
}

func TestIsBMSHistoryWideJSONValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "object", value: `{"a":1}`, want: true},
		{name: "array", value: `[false,true]`, want: true},
		{name: "invalid object", value: `{not-json}`, want: false},
		{name: "string literal", value: `"text"`, want: false},
		{name: "number", value: "6553.5", want: false},
		{name: "bool", value: "false", want: false},
		{name: "plain", value: "not-json", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBMSHistoryWideJSONValue(tt.value); got != tt.want {
				t.Fatalf("isBMSHistoryWideJSONValue(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestFormatBMSHistoryWideDisplayName(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		dataName   string
		want       string
	}{
		{
			name:       "known identifier",
			identifier: "totalVoltageMv",
			dataName:   "",
			want:       "总电压(mV) / Total Voltage (mV) (totalVoltageMv)",
		},
		{
			name:       "chinese model name",
			identifier: "currentA",
			dataName:   "工作电流",
			want:       "工作电流 / currentA",
		},
		{
			name:       "english model name",
			identifier: "currentA",
			dataName:   "Current",
			want:       "电流(A) / Current (currentA)",
		},
		{
			name:       "missing model name",
			identifier: "packCellSumVoltageV",
			dataName:   "",
			want:       "Pack单体总电压(V) / packCellSumVoltageV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBMSHistoryWideDisplayName(tt.identifier, tt.dataName); got != tt.want {
				t.Fatalf("formatBMSHistoryWideDisplayName(%q, %q) = %q, want %q", tt.identifier, tt.dataName, got, tt.want)
			}
		})
	}
}

func TestFormatBMSHistoryDataType(t *testing.T) {
	tests := map[string]string{
		"attribute": "属性 / Attribute",
		"telemetry": "遥测 / Telemetry",
		"unknown":   "unknown",
		"":          "-",
	}
	for input, want := range tests {
		if got := formatBMSHistoryDataType(input); got != want {
			t.Fatalf("formatBMSHistoryDataType(%q) = %q, want %q", input, got, want)
		}
	}
}
