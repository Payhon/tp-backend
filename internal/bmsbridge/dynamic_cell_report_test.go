package bmsbridge

import (
	"context"
	"math"
	"testing"

	"project/internal/bms/protocol"

	"github.com/sirupsen/logrus"
)

func TestCollectDeviceBatterySyncValuesDoesNotSyncItemUUID(t *testing.T) {
	values := collectDeviceBatterySyncValues(
		map[string]any{
			"identity.boardCode":    "USER-BOARD-CODE",
			"identity.bluetoothMac": "AA:BB:CC:DD:EE:FF",
			"socket.imei":           "",
			"energy.socPct":         float64(88),
		},
		map[string]string{
			"item_uuid":        "identity.boardCode",
			"identity_ble_mac": "identity.bluetoothMac",
			"imei":             "socket.imei",
			"soc":              "energy.socPct",
		},
	)

	if _, ok := values["item_uuid"]; ok {
		t.Fatalf("item_uuid should not be synced from identity.boardCode")
	}
	if _, ok := values["imei"]; ok {
		t.Fatalf("empty string should not be synced")
	}
	if got := values["identity_ble_mac"]; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("identity_ble_mac = %v, want AA:BB:CC:DD:EE:FF", got)
	}
	if got := values["soc"]; got != float64(88) {
		t.Fatalf("soc = %v, want 88", got)
	}
}

func TestDecodeDynamicCellReportFromSocketFrame(t *testing.T) {
	raw := "7F55FAFEFF3C0141001C0BAE0BAA0BA90BA70BB20BAC0BA90BB10BBA0BA50BAD0BA90BA60B9D0BB90BC30BB90BB94A4632422D355644324331343030302D3630412DED41FD"
	frameBytes, err := protocol.DecodeHexString(raw)
	if err != nil {
		t.Fatalf("DecodeHexString() error = %v", err)
	}
	parsed, err := protocol.ParseFrame(frameBytes)
	if err != nil {
		t.Fatalf("ParseFrame() error = %v", err)
	}
	readFrame, ok := parsed.(protocol.ReadFrame)
	if !ok {
		t.Fatalf("parsed frame type = %T, want protocol.ReadFrame", parsed)
	}
	start, registers, ok, err := extractReadFramePayload(readFrame, protocol.FuncSocketRead, 0x100)
	if err != nil {
		t.Fatalf("extractReadFramePayload() error = %v", err)
	}
	if !ok {
		t.Fatal("extractReadFramePayload() ok = false")
	}
	if start != 0x141 {
		t.Fatalf("start = 0x%04X, want 0x0141", start)
	}
	if len(registers) != 28 {
		t.Fatalf("register count = %d, want 28", len(registers))
	}

	bridge := &Bridge{log: logrus.New()}
	bridge.statusMeta.Store("dev-1", dynamicStatusMeta{SeriesCount: 14, CellTempCount: 4})

	flat := bridge.decodeDynamicCellReport(context.Background(), "dev-1", start, registers)
	if flat == nil {
		t.Fatal("decodeDynamicCellReport() returned nil")
	}

	voltages, ok := flat["cell.voltagesMv"].([]uint16)
	if !ok {
		t.Fatalf("cell.voltagesMv type = %T", flat["cell.voltagesMv"])
	}
	wantVoltages := []uint16{2990, 2986, 2985, 2983, 2994, 2988, 2985, 2993, 3002, 2981, 2989, 2985, 2982, 2973}
	if len(voltages) != len(wantVoltages) {
		t.Fatalf("voltages len = %d, want %d", len(voltages), len(wantVoltages))
	}
	for i := range wantVoltages {
		if voltages[i] != wantVoltages[i] {
			t.Fatalf("voltages[%d] = %d, want %d", i, voltages[i], wantVoltages[i])
		}
	}

	if got := flat["electrical.highestCellVoltageMv"]; got != uint16(3002) {
		t.Fatalf("highest = %v, want 3002", got)
	}
	if got := flat["electrical.lowestCellVoltageMv"]; got != uint16(2973) {
		t.Fatalf("lowest = %v, want 2973", got)
	}
	if got := flat["electrical.maxCellVoltageDiffMv"]; got != uint16(29) {
		t.Fatalf("diff = %v, want 29", got)
	}
	if got := flat["electrical.cellVoltageIndex.highest"]; got != 9 {
		t.Fatalf("highest index = %v, want 9", got)
	}
	if got := flat["electrical.cellVoltageIndex.lowest"]; got != 14 {
		t.Fatalf("lowest index = %v, want 14", got)
	}
	if got := flat["electrical.packCellSumVoltageV"].(float64); math.Abs(got-41.816) > 0.0001 {
		t.Fatalf("pack cell sum = %.6f, want 41.816", got)
	}

	temps, ok := flat["temperature.cellTempsC"].([]*float64)
	if !ok {
		t.Fatalf("temperature.cellTempsC type = %T", flat["temperature.cellTempsC"])
	}
	if len(temps) != 4 {
		t.Fatalf("temps len = %d, want 4", len(temps))
	}
	wantTemps := []float64{27.0, 28.0, 27.0, 27.0}
	for i := range wantTemps {
		if temps[i] == nil || math.Abs(*temps[i]-wantTemps[i]) > 0.0001 {
			t.Fatalf("temps[%d] = %v, want %.1f", i, temps[i], wantTemps[i])
		}
	}
}
