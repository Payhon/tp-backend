package bmsbridge

import "testing"

func TestBootDebugSummaryFromHexParsesAckSequence(t *testing.T) {
	summary, ok := bootDebugSummaryFromHex("55 01 FE 53 00 03 00 04 FB 12 34 FD")
	if !ok {
		t.Fatal("expected boot summary")
	}
	if got := summary["boot_ack_requested_seq"]; got != 0x04FB {
		t.Fatalf("expected boot_ack_requested_seq=0x04FB, got %#v", got)
	}
	if got := summary["boot_ack_requested_hex"]; got != "0x04FB" {
		t.Fatalf("expected boot_ack_requested_hex=0x04FB, got %#v", got)
	}
	if got := summary["boot_ack_for_packet_seq"]; got != 0x04FA {
		t.Fatalf("expected boot_ack_for_packet_seq=0x04FA, got %#v", got)
	}
	if got := summary["boot_ack_for_packet_hex"]; got != "0x04FA" {
		t.Fatalf("expected boot_ack_for_packet_hex=0x04FA, got %#v", got)
	}
}

func TestBootDebugSummaryFromHexParsesPacketSequence(t *testing.T) {
	summary, ok := bootDebugSummaryFromHex("55FE0153004204FA00112233445566778899AABBCCDDEEFF00112233445566778899AABBCCDDEEFF00112233445566778899AABBCCDDEEFF00112233445566778899AABBCCDD0000")
	if !ok {
		t.Fatal("expected boot summary")
	}
	if got := summary["boot_packet_seq"]; got != 0x04FA {
		t.Fatalf("expected boot_packet_seq=0x04FA, got %#v", got)
	}
	if got := summary["boot_packet_seq_hex"]; got != "0x04FA" {
		t.Fatalf("expected boot_packet_seq_hex=0x04FA, got %#v", got)
	}
	if got := summary["boot_expected_ack_seq"]; got != 0x04FB {
		t.Fatalf("expected boot_expected_ack_seq=0x04FB, got %#v", got)
	}
	if got := summary["boot_expected_ack_hex"]; got != "0x04FB" {
		t.Fatalf("expected boot_expected_ack_hex=0x04FB, got %#v", got)
	}
}
