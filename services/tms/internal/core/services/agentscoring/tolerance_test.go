package agentscoring

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/stretchr/testify/assert"
)

func ptr(value float64) *float64 { return &value }

func TestMatch_EveryToleranceRule(t *testing.T) {
	t.Parallel()

	exact := agentquality.Tolerance{Kind: agentquality.ToleranceExact}
	ci := agentquality.Tolerance{Kind: agentquality.ToleranceCI}
	numeric := agentquality.Tolerance{Kind: agentquality.ToleranceNumeric}
	absolute := agentquality.Tolerance{Kind: agentquality.ToleranceNumeric, Abs: ptr(5)}
	relative := agentquality.Tolerance{Kind: agentquality.ToleranceNumeric, Rel: ptr(0.02)}
	day := agentquality.Tolerance{Kind: agentquality.ToleranceDateWindow, WindowSeconds: 86400}
	oneOf := agentquality.Tolerance{
		Kind:   agentquality.ToleranceOneOf,
		Values: []any{"Available", "OutOfService"},
	}
	present := agentquality.Tolerance{Kind: agentquality.TolerancePresent}
	ignore := agentquality.Tolerance{Kind: agentquality.ToleranceIgnore}
	setEq := agentquality.Tolerance{Kind: agentquality.ToleranceSetEq}

	tests := []struct {
		name     string
		rule     agentquality.Tolerance
		expected any
		actual   any
		present  bool
		want     bool
	}{
		{
			name:     "exact string",
			rule:     exact,
			expected: "shp_1",
			actual:   " shp_1 ",
			present:  true,
			want:     true,
		},
		{
			name:     "exact int against float",
			rule:     exact,
			expected: 2,
			actual:   float64(2),
			present:  true,
			want:     true,
		},
		{name: "exact differs", rule: exact, expected: "shp_1", actual: "shp_2", present: true},
		{name: "exact absent", rule: exact, expected: "shp_1"},
		{
			name:     "ci",
			rule:     ci,
			expected: "Customer Hold",
			actual:   "customer hold ",
			present:  true,
			want:     true,
		},
		{
			name:     "ci differs",
			rule:     ci,
			expected: "Customer Hold",
			actual:   "credit hold",
			present:  true,
		},
		{
			name:     "numeric exact",
			rule:     numeric,
			expected: 1350,
			actual:   "1350.00",
			present:  true,
			want:     true,
		},
		{
			name:     "numeric without tolerance",
			rule:     numeric,
			expected: 1350,
			actual:   1351,
			present:  true,
		},
		{
			name:     "numeric within abs",
			rule:     absolute,
			expected: 1350,
			actual:   1354.5,
			present:  true,
			want:     true,
		},
		{name: "numeric past abs", rule: absolute, expected: 1350, actual: 1356, present: true},
		{
			name:     "numeric within rel",
			rule:     relative,
			expected: 1000,
			actual:   "$1,019",
			present:  true,
			want:     true,
		},
		{name: "numeric past rel", rule: relative, expected: 1000, actual: 1030, present: true},
		{
			name:     "numeric not a number",
			rule:     absolute,
			expected: 1350,
			actual:   "soon",
			present:  true,
		},
		{
			name:     "date within window",
			rule:     day,
			expected: "2026-09-01",
			actual:   "2026-09-01T18:00:00Z",
			present:  true,
			want:     true,
		},
		{
			name:     "date epoch within window",
			rule:     day,
			expected: "2026-09-01",
			actual:   int64(1788264000),
			present:  true,
			want:     true,
		},
		{
			name:     "date outside window",
			rule:     day,
			expected: "2026-09-01",
			actual:   "2026-09-03",
			present:  true,
		},
		{
			name:     "date unreadable",
			rule:     day,
			expected: "2026-09-01",
			actual:   "tomorrow",
			present:  true,
		},
		{name: "one of accepted", rule: oneOf, actual: "OutOfService", present: true, want: true},
		{name: "one of refused", rule: oneOf, actual: "Sold", present: true},
		{
			name:    "present",
			rule:    present,
			actual:  "because the customer asked",
			present: true,
			want:    true,
		},
		{name: "present but blank", rule: present, actual: "  ", present: true},
		{name: "absent", rule: present},
		{name: "ignore absent", rule: ignore, want: true},
		{
			name:     "ignore anything",
			rule:     ignore,
			expected: 1,
			actual:   "x",
			present:  true,
			want:     true,
		},
		{
			name:     "set order ignored",
			rule:     setEq,
			expected: []any{"trc_1", "trc_2"},
			actual:   []any{"trc_2", "trc_1", "trc_1"},
			present:  true,
			want:     true,
		},
		{
			name:     "set string slice",
			rule:     setEq,
			expected: []any{"trc_1"},
			actual:   []string{"trc_1"},
			present:  true,
			want:     true,
		},
		{
			name:     "set missing member",
			rule:     setEq,
			expected: []any{"trc_1", "trc_2"},
			actual:   []any{"trc_1"},
			present:  true,
		},
		{
			name:     "set extra member",
			rule:     setEq,
			expected: []any{"trc_1"},
			actual:   []any{"trc_1", "trc_9"},
			present:  true,
		},
		{
			name:     "set against scalar",
			rule:     setEq,
			expected: []any{"trc_1"},
			actual:   "trc_1",
			present:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, Match(tt.rule, tt.expected, tt.actual, tt.present))
		})
	}
}

func TestArgumentsMatch_NamesWhatFailed(t *testing.T) {
	t.Parallel()

	ok, failed := argumentsMatch(
		map[string]any{"rate": 1350, "shipmentId": "shp_1"},
		map[string]agentquality.Tolerance{
			"rate": {Kind: agentquality.ToleranceNumeric, Abs: ptr(10)},
			"note": {Kind: agentquality.TolerancePresent},
		},
		map[string]any{"rate": 1500, "shipmentId": "shp_1"},
	)

	assert.False(t, ok)
	assert.Equal(t, []string{"note", "rate"}, failed)
}
