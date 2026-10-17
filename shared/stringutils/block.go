package stringutils

import (
	"strings"
	"unicode/utf8"
)

func TruncateAtBlock(value string, maxBytes int) string {
	value = strings.TrimSpace(value)
	if maxBytes <= 0 {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}

	window := value[:maxBytes]
	floor := maxBytes / 2
	if cut := strings.LastIndex(window, "\n\n"); cut > 0 && cut >= floor {
		return strings.TrimSpace(window[:cut])
	}
	if cut := strings.LastIndexByte(window, '\n'); cut > 0 && cut >= floor {
		return strings.TrimSpace(window[:cut])
	}

	end := maxBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}

	return strings.TrimSpace(value[:end])
}
