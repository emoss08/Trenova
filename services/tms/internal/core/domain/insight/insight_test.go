package insight_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validInsight() *insight.Insight {
	return &insight.Insight{
		ID:             pulid.MustNew("inst_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		DetectorKey:    "ontime-decline",
		Category:       insight.CategoryServiceQuality,
		Severity:       insight.SeverityWarning,
		Status:         insight.StatusActive,
		DedupeKey:      "ontime-decline:cus_1",
		Subject:        "Acme Foods",
		Headline:       "On-time delivery for Acme Foods fell 11 points",
		Metrics: []insight.Metric{{
			Key:       "onTimePercent",
			Label:     "On-time delivery",
			Value:     decimal.NewFromFloat(82.4),
			Unit:      insight.UnitPercent,
			Direction: insight.DirectionLowerIsWorse,
		}},
		Links:       []insight.Link{{Label: "Late stops", Path: "/shipment?customerId=cus_1", Count: 14}},
		WindowStart: 1700000000,
		WindowEnd:   1702592000,
		DetectedAt:  1702592000,
		StaleAt:     1702678400,
	}
}

func validate(t *testing.T, entity *insight.Insight) *errortypes.MultiError {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	return multiErr
}

func TestValidate_AcceptsACompleteInsight(t *testing.T) {
	t.Parallel()

	assert.False(t, validate(t, validInsight()).HasErrors())
}

func TestValidate_RequiresTheFactsThatIdentifyAFinding(t *testing.T) {
	t.Parallel()

	entity := validInsight()
	entity.DetectorKey = ""
	entity.DedupeKey = ""

	multiErr := validate(t, entity)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "Detector key is required")
	assert.Contains(t, multiErr.Error(), "Dedupe key is required")
}

// A card with no wording at all is a blank space on someone's home screen. The
// detector always supplies a fallback, so an empty headline means a bug upstream.
func TestValidate_RequiresAHeadline(t *testing.T) {
	t.Parallel()

	entity := validInsight()
	entity.Headline = ""

	assert.Contains(t, validate(t, entity).Error(), "Headline is required")
}

func TestValidate_BoundsGeneratedProse(t *testing.T) {
	t.Parallel()

	entity := validInsight()
	entity.Narrative = longString(insight.MaxNarrativeLength + 1)

	assert.Contains(t, validate(t, entity).Error(), "Narrative cannot be longer")
}

func TestValidate_RejectsAWindowThatEndsBeforeItStarts(t *testing.T) {
	t.Parallel()

	entity := validInsight()
	entity.WindowStart = entity.WindowEnd + 1

	assert.Contains(t, validate(t, entity).Error(), "Window start must not be after window end")
}

func TestValidate_RejectsAnUnreadableMetric(t *testing.T) {
	t.Parallel()

	entity := validInsight()
	entity.Metrics = []insight.Metric{{
		Key:       "",
		Label:     "",
		Unit:      insight.Unit("Furlongs"),
		Direction: insight.Direction("Sideways"),
	}}

	multiErr := validate(t, entity)
	message := multiErr.Error()

	assert.Contains(t, message, "Metric key is required")
	assert.Contains(t, message, "Metric label is required")
	assert.Contains(t, message, "Invalid metric unit")
	assert.Contains(t, message, "Invalid metric direction")
}

// The index has to reach the client so a form or a debug view can point at the
// offending entry rather than at the whole list.
func TestValidate_NamesTheOffendingMetricByIndex(t *testing.T) {
	t.Parallel()

	entity := validInsight()
	entity.Metrics = append(entity.Metrics, insight.Metric{
		Key:       "dwellHours",
		Label:     "",
		Unit:      insight.UnitHours,
		Direction: insight.DirectionHigherIsWorse,
	})

	assert.Contains(t, fieldsOf(validate(t, entity)), "metrics[1].label")
}

func TestValidate_CapsHowManyNumbersOneCardCarries(t *testing.T) {
	t.Parallel()

	entity := validInsight()
	entity.Metrics = make([]insight.Metric, insight.MaxMetrics+1)

	assert.Contains(t, validate(t, entity).Error(), "cannot carry more than 8 metrics")
}

// A link is something a person is invited to click. Every one of these escapes
// the application, and a stored insight must not be able to carry one.
func TestValidate_RejectsALinkThatLeavesTheApplication(t *testing.T) {
	t.Parallel()

	for name, path := range map[string]string{
		"absolute http":     "http://evil.example/steal",
		"absolute https":    "https://evil.example/steal",
		"protocol relative": "//evil.example/steal",
		"javascript":        "javascript:alert(1)",
		"unrooted":          "shipment?customerId=cus_1",
		"backslash":         "/shipment\\..\\admin",
		"newline injected":  "/shipment\nLocation: http://evil.example",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			entity := validInsight()
			entity.Links = []insight.Link{{Label: "Look", Path: path}}

			assert.Contains(
				t,
				validate(t, entity).Error(),
				"Link must be a path inside this application",
			)
		})
	}
}

func TestValidate_AcceptsAnInApplicationPath(t *testing.T) {
	t.Parallel()

	entity := validInsight()
	entity.Links = []insight.Link{
		{Label: "Late stops", Path: "/shipment?customerId=cus_1&status=Delivered", Count: 14},
	}

	assert.False(t, validate(t, entity).HasErrors())
}

func TestIsStale_ComparesAgainstTheRefreshHorizon(t *testing.T) {
	t.Parallel()

	entity := validInsight()

	assert.False(t, entity.IsStale(entity.StaleAt))
	assert.True(t, entity.IsStale(entity.StaleAt+1))
}

// An insight with no horizon is not stale rather than always stale: the absence
// of a refresh promise is not evidence the numbers are old.
func TestIsStale_TreatsAnUnsetHorizonAsFresh(t *testing.T) {
	t.Parallel()

	entity := validInsight()
	entity.StaleAt = 0

	assert.False(t, entity.IsStale(1<<62))
}

func TestMetricByKey_ReadsNumbersByNameRatherThanPosition(t *testing.T) {
	t.Parallel()

	entity := validInsight()
	entity.Metrics = append(entity.Metrics, insight.Metric{
		Key:       "lateStops",
		Label:     "Late stops",
		Value:     decimal.NewFromInt(14),
		Unit:      insight.UnitCount,
		Direction: insight.DirectionHigherIsWorse,
	})

	metric, ok := entity.MetricByKey("lateStops")

	require.True(t, ok)
	assert.True(t, metric.Value.Equal(decimal.NewFromInt(14)))

	_, missing := entity.MetricByKey("nothingLikeThis")
	assert.False(t, missing)
}

// Severity drives what a reader sees first, so the order must be the business
// order and not the alphabet.
func TestSeverityRank_OrdersByUrgencyNotAlphabetically(t *testing.T) {
	t.Parallel()

	assert.Greater(t, insight.SeverityCritical.Rank(), insight.SeverityWarning.Rank())
	assert.Greater(t, insight.SeverityWarning.Rank(), insight.SeverityInfo.Rank())
	assert.Equal(t, 0, insight.Severity("Nonsense").Rank())
}

// fieldsOf collects the field paths a validation produced. The rendered message
// carries only the wording, and the path is what tells a caller which entry of a
// list was wrong.
func fieldsOf(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func longString(length int) string {
	out := make([]byte, length)
	for index := range out {
		out[index] = 'a'
	}

	return string(out)
}
