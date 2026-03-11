package protocol

import "testing"

func TestNormalizeRWParamKey(t *testing.T) {
	tests := map[string]string{
		"BATTERY_TYPE":        RWParamBatteryType,
		"batteryType":         RWParamBatteryType,
		"battery_type":        RWParamBatteryType,
		"series-count-config": RWParamSeriesCountConfig,
	}

	for input, expected := range tests {
		got, ok := NormalizeRWParamKey(input)
		if !ok {
			t.Fatalf("expected key %q to normalize", input)
		}
		if got != expected {
			t.Fatalf("normalize %q got %q want %q", input, got, expected)
		}
	}
}

func TestLookupRWParamDefBatteryType(t *testing.T) {
	def, ok := LookupRWParamDef("batteryType")
	if !ok {
		t.Fatal("expected batteryType to be found")
	}
	if def.Addr != 0x0004 {
		t.Fatalf("addr got 0x%04X want 0x0004", def.Addr)
	}
	if def.Type != "u8" {
		t.Fatalf("type got %q want u8", def.Type)
	}
	if def.Byte != "H" {
		t.Fatalf("byte got %q want H", def.Byte)
	}
}
