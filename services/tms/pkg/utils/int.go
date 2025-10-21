package utils

import (
	"fmt"
	"math"
	"regexp"
)

func CalculateLuhnCheckDigit(s string) int {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(s, "")

	sum := 0
	parity := len(digits) % 2

	for i, r := range digits {
		digit := int(r - '0')

		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	return (10 - (sum % 10)) % 10
}

func SafeInt16(value int) (int16, error) {
	if value > math.MaxInt16 || value < math.MinInt16 {
		return 0, fmt.Errorf("value %d is outside int16 range", value)
	}
	return int16(value), nil
}

func SafeInt32(value int) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}

func SafeIntToUint16(value int) uint16 {
	if value > math.MaxUint16 || value < 0 {
		return 0
	}

	return uint16(value)
}

func ToInt16(i *int16) int16 {
	if i == nil {
		return math.MinInt16
	}
	return *i
}

func SafeUint64ToInt64(value uint64) (int64, error) {
	if value > 1<<63-1 {
		return 0, fmt.Errorf("value %d is outside int64 range", value)
	}
	return int64(value), nil
}

func ConvertToUint32(value int) uint32 {
	if value >= 0 && value <= math.MaxUint32 {
		return uint32(value)
	}

	return 0
}
