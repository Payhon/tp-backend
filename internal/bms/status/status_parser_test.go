package status

import "testing"

func testStatusRegisters() []uint16 {
	return make([]uint16, 0x180-0x100+1)
}

func setU16(regs []uint16, address uint16, value uint16) {
	regs[int(address-0x100)] = value
}

func setU32(regs []uint16, address uint16, value uint32) {
	setU16(regs, address, uint16((value>>16)&0xffff))
	setU16(regs, address+1, uint16(value&0xffff))
}

func TestParseStatusRegistersSeparatesProtectionAndFailureBits(t *testing.T) {
	regs := testStatusRegisters()
	setU32(regs, 0x12F, (1<<0)|(1<<8)|(1<<30))
	setU16(regs, 0x131, (1<<0)|(1<<12))

	got, err := ParseStatusRegisters(0x100, regs)
	if err != nil {
		t.Fatalf("ParseStatusRegisters returned error: %v", err)
	}

	if !got.Status.ProtectionStatus["cellOverVoltageProtectionLv1"] {
		t.Fatalf("expected 0x12F bit0 to decode as cellOverVoltageProtectionLv1")
	}
	if !got.Status.ProtectionStatus["cellOverVoltageProtectionLv2"] {
		t.Fatalf("expected 0x12F bit8 to decode as cellOverVoltageProtectionLv2")
	}
	if !got.Status.ProtectionStatus["packOverVoltageProtection"] {
		t.Fatalf("expected 0x12F bit30 to decode as packOverVoltageProtection")
	}
	if _, ok := got.Status.ProtectionStatus["chargeMosFault"]; ok {
		t.Fatalf("0x12F protection status must not decode bit0 as chargeMosFault")
	}
	if !got.Status.FailureStatus["chargeMosFault"] {
		t.Fatalf("expected 0x131 bit0 to decode as chargeMosFault")
	}
	if !got.Status.FailureStatus["afeCommunicationFault"] {
		t.Fatalf("expected 0x131 bit12 to decode as afeCommunicationFault")
	}
}

func TestParseStatusRegistersDecodesAlarmStatus32Bit(t *testing.T) {
	regs := testStatusRegisters()
	setU32(regs, 0x134, (1<<10)|(1<<16)|(1<<18))

	got, err := ParseStatusRegisters(0x100, regs)
	if err != nil {
		t.Fatalf("ParseStatusRegisters returned error: %v", err)
	}

	if !got.Status.AlarmStatus["cellOverVoltageAlarm"] {
		t.Fatalf("expected 0x134 bit10 to decode as cellOverVoltageAlarm")
	}
	if !got.Status.AlarmStatus["deltaVAlarm"] {
		t.Fatalf("expected 0x135 bit0 / alarm bit16 to decode as deltaVAlarm")
	}
	if !got.Status.AlarmStatus["insulationAlarm"] {
		t.Fatalf("expected 0x135 bit2 / alarm bit18 to decode as insulationAlarm")
	}
}
