package status

import "bytes"

func DecodeASCII(b []byte) string {
	// Stop at first '\0'. Also trim trailing 0xFF padding.
	end := len(b)
	if i := bytes.IndexByte(b, 0x00); i >= 0 {
		end = i
	}
	for end > 0 && b[end-1] == 0xFF {
		end--
	}
	return string(b[:end])
}
