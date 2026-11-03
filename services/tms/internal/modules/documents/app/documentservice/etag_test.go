package documentservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestETagMatchesAnyTagTheBrowserHolds(t *testing.T) {
	t.Parallel()

	tag := `"abc-240x320-preview"`
	for header, want := range map[string]bool{
		tag:                       true,
		"W/" + tag:                true,
		`"other", ` + tag:         true,
		"*":                       true,
		"":                        false,
		`"other"`:                 false,
		`"abc-240x320-preview-x"`: false,
	} {
		assert.Equal(t, want, etagMatches(header, tag), "If-None-Match: %q", header)
	}
}
