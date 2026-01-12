package protocol

import "testing"

func TestCRC16Modbus_KnownVector(t *testing.T) {
	// Standard MODBUS RTU request: 01 03 00 00 00 0A => CRC should be C5 CD (lo-hi).
	data := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0A}
	crc := CRC16Modbus(data)
	if crc != 0xCDC5 {
		t.Fatalf("expected 0xCDC5, got 0x%04X", crc)
	}
}
