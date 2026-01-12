package status

import (
	"fmt"
)

type RegisterView struct {
	StartAddress uint16
	Registers    []uint16
}

func NewRegisterView(startAddress uint16, registers []uint16) RegisterView {
	return RegisterView{
		StartAddress: startAddress,
		Registers:    registers,
	}
}

func (v RegisterView) indexOf(address uint16) (int, error) {
	if address < v.StartAddress {
		return 0, fmt.Errorf("register address out of range: 0x%X", address)
	}
	idx := int(address - v.StartAddress)
	if idx < 0 || idx >= len(v.Registers) {
		return 0, fmt.Errorf("register address out of range: 0x%X", address)
	}
	return idx, nil
}

func (v RegisterView) U16(address uint16) (uint16, error) {
	idx, err := v.indexOf(address)
	if err != nil {
		return 0, err
	}
	return v.Registers[idx] & 0xFFFF, nil
}

func (v RegisterView) U32(address uint16) (uint32, error) {
	hi, err := v.U16(address)
	if err != nil {
		return 0, err
	}
	lo, err := v.U16(address + 1)
	if err != nil {
		return 0, err
	}
	return (uint32(hi) << 16) | uint32(lo), nil
}

func (v RegisterView) I32(address uint16) (int32, error) {
	u, err := v.U32(address)
	if err != nil {
		return 0, err
	}
	return int32(u), nil
}

func (v RegisterView) ByteH(address uint16) (byte, error) {
	u16, err := v.U16(address)
	if err != nil {
		return 0, err
	}
	return byte((u16 >> 8) & 0xFF), nil
}

func (v RegisterView) ByteL(address uint16) (byte, error) {
	u16, err := v.U16(address)
	if err != nil {
		return 0, err
	}
	return byte(u16 & 0xFF), nil
}

func (v RegisterView) Bytes(address uint16, byteLength int) ([]byte, error) {
	startIdx, err := v.indexOf(address)
	if err != nil {
		return nil, err
	}
	regsNeeded := (byteLength + 1) / 2
	if startIdx+regsNeeded > len(v.Registers) {
		return nil, fmt.Errorf("bytes out of range: 0x%X len=%d", address, byteLength)
	}
	out := make([]byte, regsNeeded*2)
	for i := 0; i < regsNeeded; i++ {
		reg := v.Registers[startIdx+i] & 0xFFFF
		out[i*2] = byte((reg >> 8) & 0xFF)
		out[i*2+1] = byte(reg & 0xFF)
	}
	return out[:byteLength], nil
}
