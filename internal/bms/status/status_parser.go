package status

import (
	"fmt"
)

func decodeTempCFromOffset(raw byte, offset int) *float64 {
	if raw == 0xFF {
		return nil
	}
	v := float64(int(raw) + offset)
	return &v
}

func decodeBmsCellTempCFromKelvin10(raw uint16) *float64 {
	// Doc: 25°C => 250 + 2731 (i.e. (C*10) + 2731)
	if raw == 0xFFFF {
		return nil
	}
	v := (float64(raw) - 2731) / 10
	return &v
}

func allSame(b []byte, v byte) bool {
	for i := 0; i < len(b); i++ {
		if b[i] != v {
			return false
		}
	}
	return true
}

func bytesToHex(b []byte, sep byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 0, len(b)*3)
	for i := 0; i < len(b); i++ {
		x := b[i]
		out = append(out, hex[(x>>4)&0xF], hex[x&0xF])
		if sep != 0 && i != len(b)-1 {
			out = append(out, sep)
		}
	}
	return string(out)
}

func decodeBitField32(u32 uint32, mapping map[uint]func() string) map[string]bool {
	out := make(map[string]bool, len(mapping))
	for bit, nameFn := range mapping {
		out[nameFn()] = (u32 & (1 << bit)) != 0
	}
	return out
}

var protectionBits = map[uint]func() string{
	0:  func() string { return "chargeMosFault" },
	1:  func() string { return "dischargeMosFault" },
	2:  func() string { return "poleTempOverTempProtection" },
	3:  func() string { return "antiReverseMosFault" },
	4:  func() string { return "chargeOverCurrentProtectionLv1" },
	5:  func() string { return "dischargeOverCurrentProtectionLv1" },
	6:  func() string { return "shortCircuitProtection" },
	7:  func() string { return "insulationProtection" },
	8:  func() string { return "cellOverVoltageProtectionLv2" },
	9:  func() string { return "cellUnderVoltageProtectionLv2" },
	14: func() string { return "ambientNtcInvalid" },
	18: func() string { return "chargeLowTempProtectionCell" },
	19: func() string { return "dischargeLowTempProtectionCell" },
	20: func() string { return "cellUnderTempProtection" },
	21: func() string { return "cellOverTempProtection" },
	23: func() string { return "dischargeMosOverTempProtection" },
	24: func() string { return "chargeMosOverTempProtection" },
	25: func() string { return "fullChargeProtection" },
	26: func() string { return "deltaVProtection" },
	27: func() string { return "tempDiffProtection" },
	28: func() string { return "heatingFilmTempProtection" },
	29: func() string { return "packUnderVoltageProtection" },
	30: func() string { return "packOverVoltageProtection" },
}

var indicatorBits = map[uint]func() string{
	0:  func() string { return "discharging" },
	1:  func() string { return "charging" },
	3:  func() string { return "feedbackCharging" },
	4:  func() string { return "chargeCurrentLimited" },
	5:  func() string { return "dischargeCurrentLimited" },
	6:  func() string { return "chargeFetOn" },
	7:  func() string { return "dischargeFetOn" },
	8:  func() string { return "prechargeFetOn" },
	9:  func() string { return "antiReverseFetOn" },
	12: func() string { return "commDcdcOutputOn" },
	13: func() string { return "vibrationSensorOn" },
	16: func() string { return "chargerDeltaVDetectOn" },
	17: func() string { return "gpsPowerOn" },
	18: func() string { return "heatingFilmOn" },
	19: func() string { return "chargeHandshakeOk" },
	20: func() string { return "dischargeHandshakeOk" },
	23: func() string { return "cinPlusConnected" },
	26: func() string { return "dtedConnected" },
	28: func() string { return "boardRegistered" },
	29: func() string { return "voltageCalibrating" },
	30: func() string { return "zeroCurrentCalibrating" },
	31: func() string { return "multiCurrentCalibrating" },
}

var alarmBits = map[uint]func() string{
	0:  func() string { return "chargeHighTempAlarmCell" },
	1:  func() string { return "dischargeOrIdleHighTempAlarmCell" },
	2:  func() string { return "chargeLowTempAlarmCell" },
	3:  func() string { return "dischargeOrIdleLowTempAlarmCell" },
	4:  func() string { return "thermalRunawayAlarm" },
	5:  func() string { return "ambientHighTempAlarm" },
	6:  func() string { return "ambientLowTempAlarm" },
	7:  func() string { return "dischargeMosHighTempAlarm" },
	8:  func() string { return "chargeMosHighTempAlarm" },
	9:  func() string { return "lowSocAlarm" },
	10: func() string { return "cellOverVoltageAlarm" },
	11: func() string { return "cellUnderVoltageAlarm" },
	12: func() string { return "packOverVoltageAlarm" },
	13: func() string { return "packUnderVoltageAlarm" },
	14: func() string { return "chargeOverCurrentAlarm" },
	15: func() string { return "dischargeOverCurrentAlarm" },
	16: func() string { return "deltaVAlarm" },
	17: func() string { return "tempDiffAlarm" },
	18: func() string { return "insulationAlarm" },
}

func ParseStatusRegisters(startAddress uint16, registers []uint16) (BmsStatus, error) {
	view := NewRegisterView(startAddress, registers)

	sByte, err := view.ByteH(0x100)
	if err != nil {
		return BmsStatus{}, err
	}
	nByte, err := view.ByteL(0x100)
	if err != nil {
		return BmsStatus{}, err
	}
	s := int(sByte)
	n := int(nByte)

	hwVersionRaw, err := view.ByteH(0x101)
	if err != nil {
		return BmsStatus{}, err
	}
	swVersionRaw, err := view.ByteL(0x101)
	if err != nil {
		return BmsStatus{}, err
	}

	packCurrentRaw, err := view.I32(0x119)
	if err != nil {
		return BmsStatus{}, err
	}
	packCurrentA := float64(packCurrentRaw) * 0.0001 // 0.1mA/bit

	highestTempIndex, err := view.ByteH(0x126)
	if err != nil {
		return BmsStatus{}, err
	}
	highestTempValueByte, err := view.ByteL(0x126)
	if err != nil {
		return BmsStatus{}, err
	}
	lowestTempIndex, err := view.ByteH(0x127)
	if err != nil {
		return BmsStatus{}, err
	}
	lowestTempValueByte, err := view.ByteL(0x127)
	if err != nil {
		return BmsStatus{}, err
	}

	cellVoltageHighestIndex, err := view.ByteH(0x128)
	if err != nil {
		return BmsStatus{}, err
	}
	cellVoltageLowestIndex, err := view.ByteL(0x128)
	if err != nil {
		return BmsStatus{}, err
	}

	protU32, err := view.U32(0x12F)
	if err != nil {
		return BmsStatus{}, err
	}
	indU32, err := view.U32(0x132)
	if err != nil {
		return BmsStatus{}, err
	}
	alarmU32, err := view.U32(0x134)
	if err != nil {
		return BmsStatus{}, err
	}

	productionDateRaw, err := view.U16(0x138)
	if err != nil {
		return BmsStatus{}, err
	}
	productionDate := ProductionDate{
		Raw:   productionDateRaw,
		Year:  int((productionDateRaw >> 9) & 0x7F),
		Month: int((productionDateRaw >> 5) & 0x0F),
		Day:   int(productionDateRaw & 0x1F),
	}

	customParams := make([]uint16, 0, 8)
	for i := 0; i < 8; i++ {
		v, err := view.U16(uint16(0x139 + i))
		if err != nil {
			return BmsStatus{}, err
		}
		customParams = append(customParams, v)
	}

	cellVoltagesStart := uint16(0x141)
	cellTempsStart := cellVoltagesStart + uint16(s)
	hwModelStart := cellTempsStart + uint16(n)
	batteryGroupIDStart := hwModelStart + 16
	boardCodeStart := batteryGroupIDStart + 16
	macStart := boardCodeStart + 16

	cellVoltagesMv := make([]uint16, 0, s)
	for i := 0; i < s; i++ {
		v, err := view.U16(cellVoltagesStart + uint16(i))
		if err != nil {
			return BmsStatus{}, err
		}
		cellVoltagesMv = append(cellVoltagesMv, v)
	}

	cellTempsC := make([]*float64, 0, n)
	for i := 0; i < n; i++ {
		raw, err := view.U16(cellTempsStart + uint16(i))
		if err != nil {
			return BmsStatus{}, err
		}
		cellTempsC = append(cellTempsC, decodeBmsCellTempCFromKelvin10(raw))
	}

	hwModelBytes, err := view.Bytes(hwModelStart, 32)
	if err != nil {
		return BmsStatus{}, err
	}
	batteryGroupBytes, err := view.Bytes(batteryGroupIDStart, 32)
	if err != nil {
		return BmsStatus{}, err
	}
	boardCodeBytes, err := view.Bytes(boardCodeStart, 32)
	if err != nil {
		return BmsStatus{}, err
	}

	hardwareModel := DecodeASCII(hwModelBytes)
	batteryGroupID := DecodeASCII(batteryGroupBytes)
	boardCode := DecodeASCII(boardCodeBytes)

	macBytes, err := view.Bytes(macStart, 10)
	if err != nil {
		return BmsStatus{}, err
	}
	var bluetoothMac *string
	if !(allSame(macBytes, 0x00) || allSame(macBytes, 0xFF)) {
		s := bytesToHex(macBytes, ':')
		bluetoothMac = &s
	}

	balanceWord1, err := view.U16(0x11B)
	if err != nil {
		return BmsStatus{}, err
	}
	balanceWord2, err := view.U16(0x11C)
	if err != nil {
		return BmsStatus{}, err
	}

	balanceBits := make([]bool, 32)
	for i := 0; i < 16; i++ {
		balanceBits[i] = (balanceWord1 & (1 << uint(i))) != 0
		balanceBits[i+16] = (balanceWord2 & (1 << uint(i))) != 0
	}
	if s < len(balanceBits) {
		balanceBits = balanceBits[:s]
	}

	chargeMosByte, err := view.ByteH(0x11D)
	if err != nil {
		return BmsStatus{}, err
	}
	dischargeMosByte, err := view.ByteL(0x11D)
	if err != nil {
		return BmsStatus{}, err
	}
	prechargeMosByte, err := view.ByteH(0x11E)
	if err != nil {
		return BmsStatus{}, err
	}
	ambientByte, err := view.ByteL(0x11E)
	if err != nil {
		return BmsStatus{}, err
	}
	heatingFilmByte, err := view.ByteH(0x11F)
	if err != nil {
		return BmsStatus{}, err
	}
	poleByte, err := view.ByteL(0x11F)
	if err != nil {
		return BmsStatus{}, err
	}

	customStatus, err := view.U32(0x136)
	if err != nil {
		return BmsStatus{}, err
	}

	specialIDByte, err := view.ByteH(0x102)
	if err != nil {
		return BmsStatus{}, err
	}
	protoVersionByte, err := view.ByteL(0x102)
	if err != nil {
		return BmsStatus{}, err
	}

	designCap, err := view.U32(0x103)
	if err != nil {
		return BmsStatus{}, err
	}
	remCap, err := view.U32(0x105)
	if err != nil {
		return BmsStatus{}, err
	}
	fullCap, err := view.U32(0x107)
	if err != nil {
		return BmsStatus{}, err
	}
	fullWhRaw, err := view.U32(0x109)
	if err != nil {
		return BmsStatus{}, err
	}
	remWhRaw, err := view.U32(0x10B)
	if err != nil {
		return BmsStatus{}, err
	}
	socByte, err := view.ByteH(0x10D)
	if err != nil {
		return BmsStatus{}, err
	}
	sohByte, err := view.ByteL(0x10D)
	if err != nil {
		return BmsStatus{}, err
	}
	cycleCount, err := view.U16(0x10E)
	if err != nil {
		return BmsStatus{}, err
	}
	totalChargeCapRaw, err := view.U32(0x12B)
	if err != nil {
		return BmsStatus{}, err
	}
	totalDischargeCapRaw, err := view.U32(0x12D)
	if err != nil {
		return BmsStatus{}, err
	}

	maxChargeInterval, err := view.U16(0x10F)
	if err != nil {
		return BmsStatus{}, err
	}
	currentChargeInterval, err := view.U16(0x110)
	if err != nil {
		return BmsStatus{}, err
	}
	dischargeRemaining, err := view.U16(0x111)
	if err != nil {
		return BmsStatus{}, err
	}
	chargeRemaining, err := view.U16(0x112)
	if err != nil {
		return BmsStatus{}, err
	}
	chargeCount, err := view.U16(0x113)
	if err != nil {
		return BmsStatus{}, err
	}
	dischargeCount, err := view.U16(0x114)
	if err != nil {
		return BmsStatus{}, err
	}
	bmsTimestamp, err := view.U32(0x120)
	if err != nil {
		return BmsStatus{}, err
	}
	powerOnWorkHours, err := view.U32(0x129)
	if err != nil {
		return BmsStatus{}, err
	}

	packCellSumVoltageRaw, err := view.U16(0x115)
	if err != nil {
		return BmsStatus{}, err
	}
	vBatRaw, err := view.U16(0x116)
	if err != nil {
		return BmsStatus{}, err
	}
	vPackRaw, err := view.U16(0x117)
	if err != nil {
		return BmsStatus{}, err
	}
	vLoadRaw, err := view.U16(0x118)
	if err != nil {
		return BmsStatus{}, err
	}
	highestCellV, err := view.U16(0x122)
	if err != nil {
		return BmsStatus{}, err
	}
	lowestCellV, err := view.U16(0x123)
	if err != nil {
		return BmsStatus{}, err
	}
	avgCellV, err := view.U16(0x124)
	if err != nil {
		return BmsStatus{}, err
	}
	maxDiffV, err := view.U16(0x125)
	if err != nil {
		return BmsStatus{}, err
	}

	return BmsStatus{
		Meta: Meta{
			SeriesCount:     s,
			CellTempCount:   n,
			HardwareVersion: float64(hwVersionRaw) / 10,
			SoftwareVersion: float64(swVersionRaw) / 10,
			SpecialID:       int(specialIDByte),
			ProtocolVersion: int(protoVersionByte),
			ProductionDate:  productionDate,
		},
		Energy: Energy{
			DesignCapacityMah:         designCap,
			RemainingCapacityMah:      remCap,
			FullCapacityMah:           fullCap,
			FullWh:                    float64(fullWhRaw) * 0.1,
			RemainingWh:               float64(remWhRaw) * 0.1,
			SocPct:                    float64(socByte) * 0.5,
			SohPct:                    float64(sohByte),
			CycleCount:                cycleCount,
			TotalChargeCapacityRaw:    totalChargeCapRaw,
			TotalDischargeCapacityRaw: totalDischargeCapRaw,
		},
		Timing: Timing{
			MaxChargeIntervalHours:     maxChargeInterval,
			CurrentChargeIntervalHours: currentChargeInterval,
			DischargeRemainingMin:      dischargeRemaining,
			ChargeRemainingMin:         chargeRemaining,
			ChargeCount:                chargeCount,
			DischargeCount:             dischargeCount,
			BmsTimestamp:               bmsTimestamp,
			PowerOnWorkHours:           powerOnWorkHours,
		},
		Electrical: Electrical{
			PackCellSumVoltageV:  float64(packCellSumVoltageRaw) * 0.1,
			VBatV:                float64(vBatRaw) * 0.1,
			VPackV:               float64(vPackRaw) * 0.1,
			VLoadV:               float64(vLoadRaw) * 0.1,
			CurrentA:             packCurrentA,
			HighestCellVoltageMv: highestCellV,
			LowestCellVoltageMv:  lowestCellV,
			AvgCellVoltageMv:     avgCellV,
			MaxCellVoltageDiffMv: maxDiffV,
			CellVoltageIndex: CellVoltageIndex{
				Highest: int(cellVoltageHighestIndex),
				Lowest:  int(cellVoltageLowestIndex),
			},
		},
		Temperature: Temperature{
			ChargeMosC:    decodeTempCFromOffset(chargeMosByte, -40),
			DischargeMosC: decodeTempCFromOffset(dischargeMosByte, -40),
			PrechargeMosC: decodeTempCFromOffset(prechargeMosByte, -40),
			AmbientC:      decodeTempCFromOffset(ambientByte, -40),
			HeatingFilmC:  decodeTempCFromOffset(heatingFilmByte, -40),
			PoleC:         decodeTempCFromOffset(poleByte, -40),
			HighestTemp: TempIndexValue{
				Index:  int(highestTempIndex),
				ValueC: decodeTempCFromOffset(highestTempValueByte, -40),
			},
			LowestTemp: TempIndexValue{
				Index:  int(lowestTempIndex),
				ValueC: decodeTempCFromOffset(lowestTempValueByte, -40),
			},
			CellTempsC: cellTempsC,
		},
		Cell: Cell{
			VoltagesMv: cellVoltagesMv,
			Balancing:  balanceBits,
		},
		Status: StatusBits{
			ProtectionStatus: decodeBitField32(uint32(protU32), protectionBits),
			IndicatorStatus:  decodeBitField32(uint32(indU32), indicatorBits),
			AlarmStatus:      decodeBitField32(uint32(alarmU32), alarmBits),
			CustomStatus:     customStatus,
		},
		Identity: Identity{
			HardwareModel:  hardwareModel,
			BatteryGroupID: batteryGroupID,
			BoardCode:      boardCode,
			BluetoothMac:   bluetoothMac,
		},
		CustomParams: customParams,
	}, nil
}

func EnsureStatusRangeLooksValid(startAddress uint16, regs []uint16) error {
	// Minimal sanity check to avoid treating random payload as status.
	if startAddress != 0x100 {
		return nil
	}
	if len(regs) < 0x140-0x100+1 {
		return fmt.Errorf("status registers too short: got=%d", len(regs))
	}
	return nil
}
