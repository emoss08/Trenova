package sensitivetext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskField_MasksSensitiveNamesAndValuesButNotRecordIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, field, value, want string
		masked                   bool
	}{
		{"secret field name", "apiKey", "abcdefgh", "a******h", true},
		{"ssn value", "note", "123-45-6789", "1*********9", true},
		{"opaque token", "value", "Zx9aB7cD3eF5gH1iJ2kL", "Z******************L", true},
		{"record id stays readable", "shipmentId", "shp_01JBXQ4ZP7M3N8K2R5T6V9W0YA", "shp_01JBXQ4ZP7M3N8K2R5T6V9W0YA", false},
		{"plain text", "status", "InTransit", "InTransit", false},
		{"empty", "password", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, masked := MaskField(tt.field, tt.value)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.masked, masked)
		})
	}
}

func TestMaskText_MasksOnlyTheSensitiveWords(t *testing.T) {
	t.Parallel()

	got, masked := MaskText("Call 555-123-4567 about load 42")
	assert.True(t, masked)
	assert.Equal(t, "Call 5**********7 about load 42", got)

	got, masked = MaskText("Nothing to hide here")
	assert.False(t, masked)
	assert.Equal(t, "Nothing to hide here", got)
}
