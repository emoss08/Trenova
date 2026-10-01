package tenantboundary

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportWithoutATrackerIsDropped(t *testing.T) {
	t.Parallel()

	assert.False(t, Report(t.Context(), Violation{Source: SourcePath}))
}

func TestReportCollectsViolations(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	ctx := With(t.Context(), tracker)
	orgID := pulid.MustNew("org_")

	require.True(t, Report(ctx, Violation{Source: SourceRequestBody, Field: "OrganizationID", OrganizationID: orgID}))

	violations := tracker.Violations()
	require.Len(t, violations, 1)
	assert.Equal(t, SourceRequestBody, violations[0].Source)
	assert.Equal(t, orgID, violations[0].OrganizationID)
}

func TestFromFallsBackToTheGinKey(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	ctx := context.WithValue(t.Context(), GinContextKey, tracker) //nolint:staticcheck // mirrors gin.Context.Value

	got, ok := From(ctx)
	require.True(t, ok)
	assert.Same(t, tracker, got)
}

func TestTrackerCapsViolationsPerRequest(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	ctx := With(t.Context(), tracker)
	for range maxViolationsPerRun + 5 {
		Report(ctx, Violation{Source: SourceDatabasePolicy})
	}

	assert.Len(t, tracker.Violations(), maxViolationsPerRun)
}
