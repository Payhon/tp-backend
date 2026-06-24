package api

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestSocketBootFrameForLogParsesPacketAndAck(t *testing.T) {
	packetFrame := []byte{0x55, 0xFE, 0x01, 0x53, 0x00, 0x42, 0x00, 0x44}
	for i := 0; i < 64; i++ {
		packetFrame = append(packetFrame, byte(i))
	}
	packetFrame = append(packetFrame, 0x00, 0x00)

	packet, ok := socketBootFrameForLog([]byte(`{"hex":"` + hex.EncodeToString(packetFrame) + `"}`))
	if !ok {
		t.Fatal("expected packet boot frame")
	}
	if !packet.IsPacket || packet.IsAck {
		t.Fatalf("expected data packet, got packet=%v ack=%v", packet.IsPacket, packet.IsAck)
	}
	if packet.PacketIndex != 0x44 {
		t.Fatalf("expected packet index 0x44, got %#x", packet.PacketIndex)
	}
	if packet.PacketDataBytes != 64 {
		t.Fatalf("expected 64 packet data bytes, got %d", packet.PacketDataBytes)
	}
	packetFields := packet.logFields()
	if got := packetFields["boot_packet_seq_hex"]; got != "0x0044" {
		t.Fatalf("expected boot_packet_seq_hex=0x0044, got %#v", got)
	}
	if got := packetFields["boot_expected_ack_hex"]; got != "0x0045" {
		t.Fatalf("expected boot_expected_ack_hex=0x0045, got %#v", got)
	}

	ackFrame := []byte{0x55, 0x01, 0xFE, 0x53, 0x00, 0x03, 0x00, 0x00, 0x45, 0x00, 0x00}
	ack, ok := socketBootFrameForLog([]byte(hex.EncodeToString(ackFrame)))
	if !ok {
		t.Fatal("expected ack boot frame")
	}
	if !ack.IsAck || ack.IsPacket {
		t.Fatalf("expected ack, got packet=%v ack=%v", ack.IsPacket, ack.IsAck)
	}
	if ack.AckStatus != 0 {
		t.Fatalf("expected ack status 0, got %d", ack.AckStatus)
	}
	if ack.AckRequested != 0x45 {
		t.Fatalf("expected ack requested 0x45, got %#x", ack.AckRequested)
	}
	ackFields := ack.logFields()
	if got := ackFields["boot_ack_requested_hex"]; got != "0x0045" {
		t.Fatalf("expected boot_ack_requested_hex=0x0045, got %#v", got)
	}
	if got := ackFields["boot_ack_for_packet_hex"]; got != "0x0044" {
		t.Fatalf("expected boot_ack_for_packet_hex=0x0044, got %#v", got)
	}
}

func TestSocketBootSessionTraceReportsSlowAck(t *testing.T) {
	trace := &socketBootSessionTrace{}
	base := time.Unix(100, 0)

	downlinkFields := logrus.Fields{}
	slowDownlink := trace.observeDownlink(downlinkFields, socketBootFrameLogInfo{
		IsPacket:    true,
		PacketIndex: 0x44,
	}, base)
	if slowDownlink {
		t.Fatal("first downlink should not be marked slow")
	}
	if got := downlinkFields["boot_packet_attempt"]; got != 1 {
		t.Fatalf("expected first boot_packet_attempt=1, got %#v", got)
	}

	ackFields := logrus.Fields{}
	slowAck := trace.observeAck(ackFields, socketBootFrameLogInfo{
		IsAck:        true,
		AckStatus:    0,
		AckRequested: 0x45,
	}, base.Add(2500*time.Millisecond))
	if !slowAck {
		t.Fatal("expected slow ack when ack arrives 2500ms after matching downlink")
	}
	if got := ackFields["boot_ack_after_downlink_ms"]; got != int64(2500) {
		t.Fatalf("expected boot_ack_after_downlink_ms=2500, got %#v", got)
	}
	if got := ackFields["boot_last_downlink_packet_hex"]; got != "0x0044" {
		t.Fatalf("expected boot_last_downlink_packet_hex=0x0044, got %#v", got)
	}

	nextDownlinkFields := logrus.Fields{}
	slowAfterAck := trace.observeDownlink(nextDownlinkFields, socketBootFrameLogInfo{
		IsPacket:    true,
		PacketIndex: 0x45,
	}, base.Add(2600*time.Millisecond))
	if slowAfterAck {
		t.Fatal("next downlink 100ms after ack should not be marked slow")
	}
	if got := nextDownlinkFields["boot_packet_after_ack_ms"]; got != int64(100) {
		t.Fatalf("expected boot_packet_after_ack_ms=100, got %#v", got)
	}
	if got := nextDownlinkFields["boot_packet_attempt"]; got != 1 {
		t.Fatalf("expected next boot_packet_attempt=1, got %#v", got)
	}

	retryDownlinkFields := logrus.Fields{}
	slowRetryAfterAck := trace.observeDownlink(retryDownlinkFields, socketBootFrameLogInfo{
		IsPacket:    true,
		PacketIndex: 0x45,
	}, base.Add(6600*time.Millisecond))
	if !slowRetryAfterAck {
		t.Fatal("expected retry downlink 4100ms after ack to be marked slow")
	}
	if got := retryDownlinkFields["boot_packet_retry"]; got != true {
		t.Fatalf("expected retry packet to be marked, got %#v", got)
	}
	if got := retryDownlinkFields["boot_packet_retry_count"]; got != 1 {
		t.Fatalf("expected retry count 1, got %#v", got)
	}
	if got := retryDownlinkFields["boot_packet_attempt"]; got != 2 {
		t.Fatalf("expected retry boot_packet_attempt=2, got %#v", got)
	}
	if got := retryDownlinkFields["boot_last_ack_requested_hex"]; got != "0x0045" {
		t.Fatalf("expected boot_last_ack_requested_hex=0x0045, got %#v", got)
	}
}
