package util

import (
	"fmt"
	"regexp"
)

const (
	FalseValue uint8 = 0
	TrueValue  uint8 = 1
)

func ValueToString(value any) string {
	switch val := value.(type) {
	case string:
		return val
	case bool:
		if val {
			return IntToString(TrueValue)
		}

		return IntToString(FalseValue)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%.4f", val)
	default:
		return fmt.Sprint(value)
	}
}

// true if it could be an int or a float
// both with optional sign
func IsNumeric(s string) bool {
	return regexp.MustCompile(`^[+-]?\d+(\.\d+)?$`).MatchString(s)
}

// true if it could be an int with option sign
func IsInteger(s string) bool {
	return regexp.MustCompile(`^[+-]?\d+$`).MatchString(s)
}

func IsUnsignedInteger(s string) bool {
	return regexp.MustCompile(`^\d+$`).MatchString(s)
}

// true if it could be a float with optional sign
func IsFloat(s string) bool {
	return regexp.MustCompile(`^[+-]?\d+\.\d+$`).MatchString(s)
}
