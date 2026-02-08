package protocol

import "fmt"

const (
	FrameHead0 byte = 0x7F
	FrameHead1 byte = 0x55
	FrameTail  byte = 0xFD
)

const (
	FuncReadHoldingRegisters byte = 0x03
	FuncSocketRead           byte = 0x0F
	FuncWriteMultipleRegs    byte = 0x10
	FuncAssignSlaveAddr      byte = 0x11
	FuncReadUUID             byte = 0xFF
)

type FrameType string

const (
	FrameTypeError        FrameType = "error"
	FrameTypeRead         FrameType = "read"  // response
	FrameTypeWrite        FrameType = "write" // response
	FrameTypeReadRequest  FrameType = "read_request"
	FrameTypeWriteRequest FrameType = "write_request"
)

type ErrorFrame struct {
	Type          FrameType
	SourceAddress byte
	TargetAddress byte
	FunctionCode  byte
	ErrorCode     byte
	Raw           []byte
}

type ReadFrame struct {
	Type          FrameType
	SourceAddress byte
	TargetAddress byte
	FunctionCode  byte
	ByteCount     byte
	Data          []byte
	Raw           []byte
}

type WriteFrame struct {
	Type          FrameType
	SourceAddress byte
	TargetAddress byte
	FunctionCode  byte
	StartAddress  uint16
	Quantity      uint16
	Raw           []byte
}

// ReadRequestFrame matches the request format for reading registers (e.g. function 0x03).
// It is 12 bytes long (including CRC and tail).
type ReadRequestFrame struct {
	Type          FrameType
	SourceAddress byte
	TargetAddress byte
	FunctionCode  byte
	StartAddress  uint16
	Quantity      uint16
	Raw           []byte
}

// WriteRequestFrame matches the request format for writing multiple registers (0x10)
// but is also used by custom report frames because it carries start address + data.
type WriteRequestFrame struct {
	Type          FrameType
	SourceAddress byte
	TargetAddress byte
	FunctionCode  byte
	StartAddress  uint16
	Quantity      uint16
	ByteCount     byte
	Data          []byte
	Raw           []byte
}

type ParsedFrame interface {
	isParsedFrame()
}

func (ErrorFrame) isParsedFrame()        {}
func (ReadFrame) isParsedFrame()         {}
func (WriteFrame) isParsedFrame()        {}
func (ReadRequestFrame) isParsedFrame()  {}
func (WriteRequestFrame) isParsedFrame() {}

func ParseFrame(frameBytes []byte) (ParsedFrame, error) {
	if len(frameBytes) < 6 {
		return nil, &ProtocolError{Message: "frame too short", Extra: map[string]any{"length": len(frameBytes)}}
	}
	if frameBytes[0] != FrameHead0 || frameBytes[1] != FrameHead1 {
		return nil, &ProtocolError{Message: "bad frame header"}
	}
	if frameBytes[len(frameBytes)-1] != FrameTail {
		return nil, &ProtocolError{Message: "bad frame tail"}
	}

	declaredCrcLo := frameBytes[len(frameBytes)-3]
	declaredCrcHi := frameBytes[len(frameBytes)-2]
	declaredCrc := uint16(declaredCrcHi)<<8 | uint16(declaredCrcLo)

	bodyWithoutCrcAndTail := frameBytes[:len(frameBytes)-3]
	if len(bodyWithoutCrcAndTail) < 3 {
		return nil, &ProtocolError{Message: "frame body too short"}
	}

	// CRC covers bytes from source address to last data byte (excludes 0x7F,0x55)
	crcRegion := bodyWithoutCrcAndTail[2:]
	calcCrc := CRC16Modbus(crcRegion)
	if declaredCrc != calcCrc {
		return nil, &ProtocolError{
			Message: "CRC mismatch",
			Extra: map[string]any{
				"declaredCrc": fmt.Sprintf("0x%04X", declaredCrc),
				"calcCrc":     fmt.Sprintf("0x%04X", calcCrc),
			},
		}
	}

	sourceAddress := frameBytes[2]
	targetAddress := frameBytes[3]
	functionCode := frameBytes[4]

	// Error response: functionCode = req + 0x80, then 1 byte errorCode
	if len(frameBytes) == 9 && (functionCode&0x80) != 0 {
		return ErrorFrame{
			Type:          FrameTypeError,
			SourceAddress: sourceAddress,
			TargetAddress: targetAddress,
			FunctionCode:  functionCode,
			ErrorCode:     frameBytes[5],
			Raw:           append([]byte(nil), frameBytes...),
		}, nil
	}

	// 12-byte frames can be:
	// - Read request:  [.., func, addrHi, addrLo, qtyHi, qtyLo, crcLo, crcHi, tail] (func e.g. 0x03)
	// - Write response:[.., func, addrHi, addrLo, qtyHi, qtyLo, crcLo, crcHi, tail] (func e.g. 0x10/0x11)
	if len(frameBytes) == 12 && (functionCode == FuncWriteMultipleRegs || functionCode == FuncAssignSlaveAddr) {
		startAddress := uint16(frameBytes[5])<<8 | uint16(frameBytes[6])
		quantity := uint16(frameBytes[7])<<8 | uint16(frameBytes[8])
		return WriteFrame{
			Type:          FrameTypeWrite,
			SourceAddress: sourceAddress,
			TargetAddress: targetAddress,
			FunctionCode:  functionCode,
			StartAddress:  startAddress,
			Quantity:      quantity,
			Raw:           append([]byte(nil), frameBytes...),
		}, nil
	}

	if len(frameBytes) == 12 {
		startAddress := uint16(frameBytes[5])<<8 | uint16(frameBytes[6])
		quantity := uint16(frameBytes[7])<<8 | uint16(frameBytes[8])
		return ReadRequestFrame{
			Type:          FrameTypeReadRequest,
			SourceAddress: sourceAddress,
			TargetAddress: targetAddress,
			FunctionCode:  functionCode,
			StartAddress:  startAddress,
			Quantity:      quantity,
			Raw:           append([]byte(nil), frameBytes...),
		}, nil
	}

	// Write request (or custom report with start address + data):
	// [.., func, addrHi, addrLo, qtyHi, qtyLo, byteCount, data..., crcLo, crcHi, tail]
	if len(frameBytes) >= 13 {
		startAddress := uint16(frameBytes[5])<<8 | uint16(frameBytes[6])
		quantity := uint16(frameBytes[7])<<8 | uint16(frameBytes[8])
		byteCount := frameBytes[9]
		expectedLength := 13 + int(byteCount)
		if expectedLength == len(frameBytes) {
			data := append([]byte(nil), frameBytes[10:10+int(byteCount)]...)
			return WriteRequestFrame{
				Type:          FrameTypeWriteRequest,
				SourceAddress: sourceAddress,
				TargetAddress: targetAddress,
				FunctionCode:  functionCode,
				StartAddress:  startAddress,
				Quantity:      quantity,
				ByteCount:     byteCount,
				Data:          data,
				Raw:           append([]byte(nil), frameBytes...),
			}, nil
		}
	}

	// Read-like response: [.., func, byteCount, data..., crcLo, crcHi, tail]
	if len(frameBytes) >= 10 {
		byteCount := frameBytes[5]
		expectedLength := 2 + 3 + 1 + int(byteCount) + 2 + 1
		if expectedLength == len(frameBytes) {
			dataStart := 6
			dataEnd := dataStart + int(byteCount)
			return ReadFrame{
				Type:          FrameTypeRead,
				SourceAddress: sourceAddress,
				TargetAddress: targetAddress,
				FunctionCode:  functionCode,
				ByteCount:     byteCount,
				Data:          append([]byte(nil), frameBytes[dataStart:dataEnd]...),
				Raw:           append([]byte(nil), frameBytes...),
			}, nil
		}
	}

	return nil, &ProtocolError{
		Message: "unknown frame type",
		Extra:   map[string]any{"functionCode": fmt.Sprintf("0x%02X", functionCode), "length": len(frameBytes)},
	}
}

func SplitIntoRegistersBE(dataBytes []byte) ([]uint16, error) {
	if len(dataBytes)%2 != 0 {
		return nil, &ProtocolError{Message: "register data length must be even", Extra: map[string]any{"length": len(dataBytes)}}
	}
	regs := make([]uint16, len(dataBytes)/2)
	for i := 0; i < len(regs); i++ {
		hi := dataBytes[i*2]
		lo := dataBytes[i*2+1]
		regs[i] = uint16(hi)<<8 | uint16(lo)
	}
	return regs, nil
}
