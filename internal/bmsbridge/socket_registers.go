package bmsbridge

import (
	"project/internal/bms/status"
)

func decodeSocketRegisters(startAddress uint16, regs []uint16) map[string]any {
	const base = 0x900
	if startAddress > 0x923 {
		return nil
	}
	last := startAddress + uint16(len(regs)) - 1
	if last < base {
		return nil
	}

	view := status.NewRegisterView(startAddress, regs)
	out := make(map[string]any, 12)

	if v, ok := u32At(view, 0x900); ok {
		out["socket.longitude"] = float64(v) * 0.00001
	}
	if v, ok := u32At(view, 0x902); ok {
		out["socket.latitude"] = float64(v) * 0.00001
	}
	if v, ok := u16At(view, 0x904); ok {
		out["socket.speedKmh"] = float64(v) * 0.001
	}
	if v, ok := u16At(view, 0x905); ok {
		out["socket.altitudeM"] = int(v)
	}
	if v, ok := u16At(view, 0x906); ok {
		out["socket.rssi"] = int(v)
	}
	if v, ok := u16At(view, 0x907); ok {
		out["socket.tac"] = int(v)
	}
	if v, ok := u32At(view, 0x908); ok {
		out["socket.cellId"] = int64(v)
	}
	if s, ok := strAt(view, 0x90A, 18); ok {
		out["socket.imei"] = s
	}
	if s, ok := strAt(view, 0x913, 22); ok {
		out["socket.iccid"] = s
	}
	if s, ok := strAt(view, 0x91E, 12); ok {
		out["socket.moduleVersion"] = s
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func u16At(view status.RegisterView, addr uint16) (uint16, bool) {
	v, err := view.U16(addr)
	if err != nil {
		return 0, false
	}
	return v, true
}

func u32At(view status.RegisterView, addr uint16) (uint32, bool) {
	v, err := view.U32(addr)
	if err != nil {
		return 0, false
	}
	return v, true
}

func strAt(view status.RegisterView, addr uint16, byteLen int) (string, bool) {
	b, err := view.Bytes(addr, byteLen)
	if err != nil {
		return "", false
	}
	return status.DecodeASCII(b), true
}
