package protocol

import "strings"

type RWParamDef struct {
	Key   string
	Addr  uint16
	Type  string
	Byte  string
	Scale float64
	Unit  string
}

const (
	RWParamSeriesCountConfig = "SERIES_COUNT_CONFIG"
	RWParamBatteryType       = "BATTERY_TYPE"
	RWParamDesignCapacityAh  = "DESIGN_CAPACITY_AH"
	RWParamFullCapacityAh    = "FULL_CAPACITY_AH"
	RWParamRemainCapacityAh  = "REMAIN_CAPACITY_AH"
	RWParamFunctionConfig    = "FUNCTION_CONFIG"
)

var rwParamDefs = map[string]RWParamDef{
	RWParamSeriesCountConfig: {Key: RWParamSeriesCountConfig, Addr: 0x0001, Type: "u16"},
	RWParamBatteryType:       {Key: RWParamBatteryType, Addr: 0x0004, Type: "u8", Byte: "H"},
	RWParamDesignCapacityAh:  {Key: RWParamDesignCapacityAh, Addr: 0x0030, Type: "u32", Scale: 0.001, Unit: "Ah"},
	RWParamFullCapacityAh:    {Key: RWParamFullCapacityAh, Addr: 0x0032, Type: "u32", Scale: 0.001, Unit: "Ah"},
	RWParamRemainCapacityAh:  {Key: RWParamRemainCapacityAh, Addr: 0x0034, Type: "u32", Scale: 0.001, Unit: "Ah"},
	RWParamFunctionConfig:    {Key: RWParamFunctionConfig, Addr: 0x003E, Type: "u16"},
}

func LookupRWParamDef(key string) (RWParamDef, bool) {
	normalized, ok := NormalizeRWParamKey(key)
	if !ok {
		return RWParamDef{}, false
	}
	def, exists := rwParamDefs[normalized]
	return def, exists
}

func NormalizeRWParamKey(key string) (string, bool) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", false
	}
	if _, ok := rwParamDefs[trimmed]; ok {
		return trimmed, true
	}
	canonical := toConstLike(trimmed)
	if _, ok := rwParamDefs[canonical]; ok {
		return canonical, true
	}
	return "", false
}

func toConstLike(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	prevUnderscore := false
	prevLower := false
	prevUpper := false
	prevDigit := false
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteByte(byte(r - 'a' + 'A'))
			prevUnderscore = false
			prevLower = true
			prevUpper = false
			prevDigit = false
		case r >= 'A' && r <= 'Z':
			nextLower := i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z'
			if b.Len() > 0 && !prevUnderscore && (prevLower || prevDigit || (prevUpper && nextLower)) {
				b.WriteByte('_')
			}
			b.WriteByte(byte(r))
			prevUnderscore = false
			prevLower = false
			prevUpper = true
			prevDigit = false
		case r >= '0' && r <= '9':
			if b.Len() > 0 && !prevUnderscore && prevLower {
				b.WriteByte('_')
			}
			b.WriteByte(byte(r))
			prevUnderscore = false
			prevLower = false
			prevUpper = false
			prevDigit = true
		default:
			if !prevUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				prevUnderscore = true
			}
			prevLower = false
			prevUpper = false
			prevDigit = false
		}
	}
	out := strings.Trim(b.String(), "_")
	out = strings.ReplaceAll(out, "__", "_")
	return out
}
