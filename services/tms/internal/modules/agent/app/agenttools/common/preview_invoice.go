package common

import (
	"strings"
)

func SlicesNonEmpty(values []string) []string {
	kept := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			kept = append(kept, value)
		}
	}

	return kept
}
