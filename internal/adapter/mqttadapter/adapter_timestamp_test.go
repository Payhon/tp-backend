package mqttadapter

import (
	"encoding/json"
	"testing"
)

func TestTelemetryPayloadTimestampOnlyTrustsBMSBridge(t *testing.T) {
	const fallback = int64(1_720_000_000_999)
	const bridgeTimestamp = int64(1_720_000_000_123)

	bridgePayload := &publicPayload{
		Source:    "bms_bridge",
		Timestamp: json.RawMessage(`1720000000123`),
	}
	if got := telemetryPayloadTimestamp(bridgePayload, fallback); got != bridgeTimestamp {
		t.Fatalf("bridge timestamp = %d, want %d", got, bridgeTimestamp)
	}

	regularPayload := &publicPayload{
		Timestamp: json.RawMessage(`1600000000000`),
	}
	if got := telemetryPayloadTimestamp(regularPayload, fallback); got != fallback {
		t.Fatalf("regular device timestamp = %d, want receive fallback %d", got, fallback)
	}
}

func TestTelemetryPayloadTimestampRejectsFarFutureBridgeTime(t *testing.T) {
	const receivedAt = int64(1_720_000_000_000)
	payload := &publicPayload{
		Source:    bmsBridgeTimestampSource,
		Timestamp: json.RawMessage(`2720000000000`),
	}
	if got := telemetryPayloadTimestamp(payload, receivedAt); got != receivedAt {
		t.Fatalf("far future bridge timestamp = %d, want receive fallback %d", got, receivedAt)
	}
}

func TestResolveTelemetryTimestampCompatibility(t *testing.T) {
	const fallback = int64(1_720_000_000_999)
	tests := []struct {
		name string
		raw  json.RawMessage
		want int64
	}{
		{name: "missing", raw: nil, want: fallback},
		{name: "invalid", raw: json.RawMessage(`"not-a-time"`), want: fallback},
		{name: "seconds", raw: json.RawMessage(`1720000000`), want: 1_720_000_000_000},
		{name: "milliseconds", raw: json.RawMessage(`1720000000123`), want: 1_720_000_000_123},
		{name: "rfc3339", raw: json.RawMessage(`"2024-07-03T09:46:40.123Z"`), want: 1_720_000_000_123},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveTelemetryTimestamp(tt.raw, fallback); got != tt.want {
				t.Fatalf("resolveTelemetryTimestamp() = %d, want %d", got, tt.want)
			}
		})
	}
}
