package permission

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLooksLikeResource(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		value string
		want  bool
	}{
		"plain":               {value: "shipment", want: true},
		"snake case":          {value: "shipment_type", want: true},
		"empty":               {value: "", want: false},
		"capitals":            {value: "Shipment", want: false},
		"scope separator":     {value: "shipment:x", want: false},
		"leading underscore":  {value: "_shipment", want: false},
		"trailing underscore": {value: "shipment_", want: false},
		"too long":            {value: strings.Repeat("a", 65), want: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, LooksLikeResource(tc.value))
		})
	}
}
