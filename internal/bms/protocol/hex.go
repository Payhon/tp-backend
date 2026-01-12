package protocol

import (
	"encoding/hex"
	"strings"
)

// DecodeHexString decodes a hex string into bytes.
//
// It tolerates common separators like spaces, '-', ':' and newlines.
func DecodeHexString(s string) ([]byte, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil, &ProtocolError{Message: "empty hex string"}
	}

	normalized := make([]byte, 0, len(trimmed))
	for i := 0; i < len(trimmed); i++ {
		c := trimmed[i]
		switch c {
		case ' ', '\t', '\r', '\n', '-', ':':
			continue
		default:
			normalized = append(normalized, c)
		}
	}

	if len(normalized)%2 != 0 {
		return nil, &ProtocolError{
			Message: "hex string length must be even",
			Extra:   map[string]any{"length": len(normalized)},
		}
	}

	out := make([]byte, hex.DecodedLen(len(normalized)))
	if _, err := hex.Decode(out, normalized); err != nil {
		return nil, &ProtocolError{Message: "invalid hex string", Extra: map[string]any{"err": err.Error()}}
	}
	return out, nil
}
