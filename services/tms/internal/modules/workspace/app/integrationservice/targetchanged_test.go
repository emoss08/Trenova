package integrationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/stretchr/testify/assert"
)

func TestTargetChanged_ReadsABlankFieldAsItsDefaultOnBothSides(t *testing.T) {
	t.Parallel()

	spec := integration.IntegrationSpec{
		Fields: []integration.ConfigFieldSpec{
			{Key: integration.ConfigKeyFuelPort, Target: true, Default: "22"},
		},
	}
	port := func(value string) map[string]any {
		return map[string]any{integration.ConfigKeyFuelPort: value}
	}

	tests := []struct {
		name    string
		before  map[string]any
		after   map[string]any
		changed bool
	}{
		{name: "the default cleared", before: port("22"), after: port("")},
		{name: "the default written out", before: port(""), after: port("22")},
		{name: "the default absent", before: port("22"), after: map[string]any{}},
		{name: "a new port", before: port("22"), after: port("2222"), changed: true},
		{name: "a custom port cleared", before: port("2222"), after: port(""), changed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.changed, targetChanged(spec, tt.before, tt.after))
		})
	}
}
