package hashutils

import (
	"crypto/sha256"
	"encoding/binary"
)

func PercentBucket(key string) int {
	sum := sha256.Sum256([]byte(key))
	return int(binary.BigEndian.Uint64(sum[:8]) % 100)
}

func InPercentSample(key string, percent int) bool {
	switch {
	case percent <= 0:
		return false
	case percent >= 100:
		return true
	default:
		return PercentBucket(key) < percent
	}
}
