package uplink

import "testing"

func TestTelemetryStorageTimestampPreservesUplinkTimestamp(t *testing.T) {
	const fallback = int64(1_720_000_000_999)
	const received = int64(1_720_000_000_123)

	bridgeMessage := &DeviceMessage{
		Timestamp: received,
		Metadata:  map[string]interface{}{telemetryTimestampSourceMetadata: "bms_bridge"},
	}
	if got := telemetryStorageTimestamp(bridgeMessage, fallback); got != received {
		t.Fatalf("timestamp = %d, want original uplink timestamp %d", got, received)
	}
	if got := telemetryStorageTimestamp(&DeviceMessage{}, fallback); got != fallback {
		t.Fatalf("missing timestamp = %d, want fallback %d", got, fallback)
	}
	regularMessage := &DeviceMessage{Timestamp: received}
	if got := telemetryStorageTimestamp(regularMessage, fallback); got != fallback {
		t.Fatalf("regular device timestamp = %d, want storage fallback %d", got, fallback)
	}
}
