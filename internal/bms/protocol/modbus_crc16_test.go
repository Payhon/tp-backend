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

func TestBuildReadRequestFrame_UsesTargetAddressCRCRegion(t *testing.T) {
	frame := BuildReadRequestFrame(0xFE, 0x01, FuncReadHoldingRegisters, 0x0100, 0x0001)

	if len(frame) != 12 {
		t.Fatalf("expected frame length 12, got %d", len(frame))
	}

	declaredCRC := uint16(frame[len(frame)-2])<<8 | uint16(frame[len(frame)-3])
	targetRegionCRC := CRC16Modbus(frame[3 : len(frame)-3])
	sourceRegionCRC := CRC16Modbus(frame[2 : len(frame)-3])

	if declaredCRC != targetRegionCRC {
		t.Fatalf("expected declared CRC 0x%04X to match target-region CRC 0x%04X", declaredCRC, targetRegionCRC)
	}
	if declaredCRC == sourceRegionCRC {
		t.Fatalf("declared CRC unexpectedly matches source-region CRC 0x%04X", sourceRegionCRC)
	}
}

func TestParseFrame_AcceptsTargetAddressCRCFrame(t *testing.T) {
	frame := BuildReadRequestFrame(0xFE, 0x01, FuncReadHoldingRegisters, 0x0100, 0x0001)

	parsed, err := ParseFrame(frame)
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}

	readReq, ok := parsed.(ReadRequestFrame)
	if !ok {
		t.Fatalf("expected ReadRequestFrame, got %T", parsed)
	}
	if readReq.SourceAddress != 0xFE || readReq.TargetAddress != 0x01 {
		t.Fatalf("unexpected addresses: source=0x%02X target=0x%02X", readReq.SourceAddress, readReq.TargetAddress)
	}
	if readReq.StartAddress != 0x0100 || readReq.Quantity != 0x0001 {
		t.Fatalf("unexpected read request payload: start=0x%04X quantity=0x%04X", readReq.StartAddress, readReq.Quantity)
	}
}
