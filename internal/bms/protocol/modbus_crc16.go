package protocol

// CRC16Modbus calculates CRC-16/MODBUS.
// Polynomial: 0xA001, Init: 0xFFFF, Refin: true, Refout: true, Xorout: 0x0000
func CRC16Modbus(bytes []byte) uint16 {
	var crc uint16 = 0xFFFF
	for i := 0; i < len(bytes); i++ {
		crc ^= uint16(bytes[i])
		for bit := 0; bit < 8; bit++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}
