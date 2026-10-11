package registry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type fakeActivities struct{}

func (fakeActivities) SweepActivity(_ context.Context) error { return nil }

func (fakeActivities) ReportActivity(_ context.Context, _ string) (int, error) { return 0, nil }

// NotAnActivity takes no context, so Temporal would not register it and
// neither should the name list: a helper method sharing a name across two
// packages is not a collision.
func (fakeActivities) NotAnActivity() string { return "" }

// AlsoNotAnActivity returns no error, which Temporal refuses.
func (fakeActivities) AlsoNotAnActivity(_ context.Context) string { return "" }

func (fakeActivities) unexported(_ context.Context) error { return nil }

func TestActivityNamesListsOnlyWhatTemporalRegisters(t *testing.T) {
	t.Parallel()

	registry := NewDomainRegistry(
		&DomainConfig{Name: "fake", TaskQueue: "q"},
		fakeActivities{},
		nil,
		zap.NewNop(),
	)

	assert.Equal(t, []string{"ReportActivity", "SweepActivity"}, registry.ActivityNames())
}

func TestConflictingActivitiesNamesWhoHoldsTheName(t *testing.T) {
	t.Parallel()

	taken := map[string]string{"RetentionActivity": "briefing-worker"}

	assert.Empty(t, conflictingActivities(taken, []string{"SweepActivity"}))
	assert.Equal(t,
		[]activityConflict{{Activity: "RetentionActivity", HeldBy: "briefing-worker"}},
		conflictingActivities(taken, []string{"SweepActivity", "RetentionActivity"}),
	)
}
