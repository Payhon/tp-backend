package protocol

import (
	"encoding/hex"
)

func BytesToHexUpper(b []byte) string {
	dst := make([]byte, hex.EncodedLen(len(b)))
	hex.Encode(dst, b)
	// hex.Encode uses lowercase; convert to uppercase for readability/compat.
	for i := 0; i < len(dst); i++ {
		if dst[i] >= 'a' && dst[i] <= 'f' {
			dst[i] = dst[i] - 'a' + 'A'
		}
	}
	return string(dst)
}

func BuildReadRequestFrame(sourceAddress, targetAddress, functionCode byte, startAddress, quantity uint16) []byte {
	frame := make([]byte, 0, 12)
	frame = append(frame, FrameHead0, FrameHead1)
	frame = append(frame, sourceAddress, targetAddress, functionCode)
	frame = append(frame, byte((startAddress>>8)&0xFF), byte(startAddress&0xFF))
	frame = append(frame, byte((quantity>>8)&0xFF), byte(quantity&0xFF))

	// CRC covers bytes from source address (excludes head bytes)
	crc := CRC16Modbus(frame[2:])
	frame = append(frame, byte(crc&0xFF), byte((crc>>8)&0xFF), FrameTail)
	return frame
}

func BuildReadResponseFrame(sourceAddress, targetAddress, functionCode byte, data []byte) []byte {
	if len(data) > 250 {
		// Spec says max 250 bytes.
		data = data[:250]
	}
	frame := make([]byte, 0, 2+3+1+len(data)+2+1)
	frame = append(frame, FrameHead0, FrameHead1)
	frame = append(frame, sourceAddress, targetAddress, functionCode)
	frame = append(frame, byte(len(data)))
	frame = append(frame, data...)

	crc := CRC16Modbus(frame[2:])
	frame = append(frame, byte(crc&0xFF), byte((crc>>8)&0xFF), FrameTail)
	return frame
}

func BuildWriteRequestFrame(sourceAddress, targetAddress, functionCode byte, startAddress uint16, registerValues []uint16) []byte {
	quantity := uint16(len(registerValues))
	byteCount := byte(quantity * 2)

	frame := make([]byte, 0, 13+int(byteCount))
	frame = append(frame, FrameHead0, FrameHead1)
	frame = append(frame, sourceAddress, targetAddress, functionCode)
	frame = append(frame, byte((startAddress>>8)&0xFF), byte(startAddress&0xFF))
	frame = append(frame, byte((quantity>>8)&0xFF), byte(quantity&0xFF))
	frame = append(frame, byteCount)
	for i := 0; i < len(registerValues); i++ {
		v := registerValues[i]
		frame = append(frame, byte((v>>8)&0xFF), byte(v&0xFF))
	}

	crc := CRC16Modbus(frame[2:])
	frame = append(frame, byte(crc&0xFF), byte((crc>>8)&0xFF), FrameTail)
	return frame
}

func BuildWriteResponseFrame(sourceAddress, targetAddress, functionCode byte, startAddress, quantity uint16) []byte {
	frame := make([]byte, 0, 12)
	frame = append(frame, FrameHead0, FrameHead1)
	frame = append(frame, sourceAddress, targetAddress, functionCode)
	frame = append(frame, byte((startAddress>>8)&0xFF), byte(startAddress&0xFF))
	frame = append(frame, byte((quantity>>8)&0xFF), byte(quantity&0xFF))

	crc := CRC16Modbus(frame[2:])
	frame = append(frame, byte(crc&0xFF), byte((crc>>8)&0xFF), FrameTail)
	return frame
}
