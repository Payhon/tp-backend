package bmsbridge

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func bootSeqHex(seq int) string {
	return fmt.Sprintf("0x%04X", seq&0xffff)
}

func cleanBootHex(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'a' && r <= 'f':
			b.WriteRune(r - 'a' + 'A')
		case r >= 'A' && r <= 'F':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func bootHexPrefix(hexText string, maxLen int) string {
	if maxLen <= 0 || len(hexText) <= maxLen {
		return hexText
	}
	return hexText[:maxLen] + "..."
}

func bootDebugSummaryFromHex(raw string) (map[string]any, bool) {
	cleanHex := cleanBootHex(raw)
	if cleanHex == "" || len(cleanHex)%2 != 0 {
		return nil, false
	}
	frame, err := hex.DecodeString(cleanHex)
	if err != nil || len(frame) < 6 || frame[0] != 0x55 {
		return nil, false
	}
	cmd := frame[3]
	dataLen := (int(frame[4]) << 8) | int(frame[5])
	summary := map[string]any{
		"boot_cmd":           fmt.Sprintf("0x%02X", cmd),
		"boot_source":        fmt.Sprintf("0x%02X", frame[1]),
		"boot_target":        fmt.Sprintf("0x%02X", frame[2]),
		"boot_data_len":      dataLen,
		"payload_bytes":      len(frame),
		"payload_hex_prefix": bootHexPrefix(cleanHex, 96),
	}
	if cmd == 0x53 && dataLen >= 2 && len(frame) >= 8 {
		if dataLen == 3 && len(frame) >= 9 {
			ackRequested := (int(frame[7]) << 8) | int(frame[8])
			summary["boot_ack_status"] = int(frame[6])
			summary["boot_ack_requested"] = ackRequested
			summary["boot_ack_requested_seq"] = ackRequested
			summary["boot_ack_requested_hex"] = bootSeqHex(ackRequested)
			if ackRequested > 0 {
				summary["boot_ack_for_packet_seq"] = ackRequested - 1
				summary["boot_ack_for_packet_hex"] = bootSeqHex(ackRequested - 1)
			}
		} else {
			packetIndex := (int(frame[6]) << 8) | int(frame[7])
			summary["boot_packet_index"] = packetIndex
			summary["boot_packet_seq"] = packetIndex
			summary["boot_packet_seq_hex"] = bootSeqHex(packetIndex)
			summary["boot_expected_ack_seq"] = packetIndex + 1
			summary["boot_expected_ack_hex"] = bootSeqHex(packetIndex + 1)
			summary["boot_packet_data_bytes"] = dataLen - 2
		}
	}
	return summary, true
}
