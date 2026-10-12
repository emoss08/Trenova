package secretutils

import (
	"strings"
	"unicode/utf8"
)

const (
	maxPrefixSegments    = 2
	maxFirstSegmentLen   = 6
	maxLaterSegmentLen   = 8
	maxPrefixLen         = 16
	minHiddenAfterPrefix = 12
	lastFourLen          = 4
	minLenForLastFour    = 8
)

var fixedPrefixes = [...]string{"AIza"}

type Fingerprint struct {
	Prefix   string
	LastFour string
}

func FingerprintOf(secret string) Fingerprint {
	trimmed := strings.TrimSpace(secret)

	return Fingerprint{Prefix: KeyPrefix(trimmed), LastFour: LastFour(trimmed)}
}

func KeyPrefix(secret string) string {
	trimmed := strings.TrimSpace(secret)
	for _, fixed := range fixedPrefixes {
		if strings.HasPrefix(trimmed, fixed) && len(trimmed)-len(fixed) >= minHiddenAfterPrefix {
			return fixed
		}
	}

	end := 0
	for segment := range maxPrefixSegments {
		limit := maxLaterSegmentLen
		if segment == 0 {
			limit = maxFirstSegmentLen
		}
		length := lowerLetters(trimmed[end:], limit)
		if length == 0 {
			break
		}
		next := end + length
		if next >= len(trimmed) || !isSeparator(trimmed[next]) {
			break
		}
		end = next + 1
	}

	if end == 0 || end > maxPrefixLen || len(trimmed)-end < minHiddenAfterPrefix {
		return ""
	}

	return trimmed[:end]
}

func LastFour(secret string) string {
	trimmed := strings.TrimSpace(secret)
	if utf8.RuneCountInString(trimmed) < minLenForLastFour {
		return ""
	}

	runes := []rune(trimmed)

	return string(runes[len(runes)-lastFourLen:])
}

func lowerLetters(value string, limit int) int {
	count := 0
	for count < len(value) && count <= limit {
		ch := value[count]
		if ch < 'a' || ch > 'z' {
			break
		}
		count++
	}
	if count > limit {
		return 0
	}

	return count
}

func isSeparator(ch byte) bool {
	return ch == '-' || ch == '_'
}
