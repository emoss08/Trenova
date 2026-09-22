package detentionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingPublisher struct {
	events []services.AgentEvent
}

func (r *recordingPublisher) Publish(_ context.Context, event services.AgentEvent) {
	r.events = append(r.events, event)
}

func (r *recordingPublisher) kinds() []agent.EventKind {
	if len(r.events) == 0 {
		return nil
	}
	out := make([]agent.EventKind, 0, len(r.events))
	for _, event := range r.events {
		out = append(out, event.Kind)
	}

	return out
}

type silentProjector struct{}

func (silentProjector) Upsert(context.Context, services.WatchtowerItemInput) {}

func (silentProjector) Resolve(
	context.Context,
	pagination.TenantInfo,
	watchtower.SourceKind,
	string,
) {
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func eventService(publisher services.AgentEventPublisher) *Service {
	return &Service{l: zap.NewNop(), watchtower: silentProjector{}, publisher: publisher}
}

func TestPublishOccurrenceOpened_FiresOnlyWhenTheClockStarts(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	occurrenceID := pulid.MustNew("do_")

	cases := []struct {
		name     string
		saved    *detention.DetentionOccurrence
		existing *detention.DetentionOccurrence
		want     []agent.EventKind
	}{
		{
			name:  "a clock that has just started wakes the desk",
			saved: &detention.DetentionOccurrence{ID: occurrenceID, IsOpen: true},
			want:  []agent.EventKind{agent.EventDetentionOccurrenceOpened},
		},
		{
			name:     "a clock that was already running is not news",
			saved:    &detention.DetentionOccurrence{ID: occurrenceID, IsOpen: true},
			existing: &detention.DetentionOccurrence{ID: occurrenceID, IsOpen: true},
			want:     nil,
		},
		{
			name:     "a clock that has stopped wakes nobody",
			saved:    &detention.DetentionOccurrence{ID: occurrenceID, IsOpen: false},
			existing: &detention.DetentionOccurrence{ID: occurrenceID, IsOpen: true},
			want:     nil,
		},
		{
			name:     "an occurrence reopened after closing is news again",
			saved:    &detention.DetentionOccurrence{ID: occurrenceID, IsOpen: true},
			existing: &detention.DetentionOccurrence{ID: occurrenceID, IsOpen: false},
			want:     []agent.EventKind{agent.EventDetentionOccurrenceOpened},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			publisher := &recordingPublisher{}
			svc := eventService(publisher)

			svc.publishOccurrenceOpened(t.Context(), tc.saved, computeStopParams{
				existing:   tc.existing,
				tenantInfo: tenant,
			})

			require.Equal(t, tc.want, publisher.kinds())
			for _, event := range publisher.events {
				require.Equal(t, occurrenceID, event.SubjectID)
				require.Equal(t, tenant, event.TenantInfo)
			}
		})
	}
}

type stubOccurrenceRepo struct {
	repositories.DetentionOccurrenceRepository
	due []*detention.DetentionOccurrence
}

func (s stubOccurrenceRepo) ListNoticesDue(
	context.Context,
	*repositories.ListNoticesDueRequest,
) ([]*detention.DetentionOccurrence, error) {
	return s.due, nil
}

type stubPolicyRepo struct {
	repositories.DetentionPolicyRepository
	policy *detention.DetentionPolicy
}

func (s stubPolicyRepo) GetByID(
	context.Context,
	*repositories.GetDetentionPolicyByIDRequest,
) (*detention.DetentionPolicy, error) {
	return s.policy, nil
}

func TestSweepNoticesDue_RaisesTheNoticesItLeavesOnTheDesk(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	policyID := pulid.MustNew("dp_")
	occurrenceID := pulid.MustNew("do_")

	cases := []struct {
		name   string
		policy *detention.DetentionPolicy
		want   []agent.EventKind
	}{
		{
			name: "a policy that leaves sending to a person wakes the desk",
			policy: &detention.DetentionPolicy{
				ID:             policyID,
				AutoSendNotice: false,
			},
			want: []agent.EventKind{agent.EventDetentionNoticeDue},
		},
		{
			name:   "a policy that could not be read wakes nobody",
			policy: nil,
			want:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			publisher := &recordingPublisher{}
			svc := eventService(publisher)
			svc.occurrenceRepo = stubOccurrenceRepo{
				due: []*detention.DetentionOccurrence{
					{ID: occurrenceID, DetentionPolicyID: &policyID},
				},
			}
			svc.policyRepo = stubPolicyRepo{policy: tc.policy}
			svc.now = func() int64 { return 1_767_225_600 }

			result, err := svc.SweepNoticesDue(t.Context(), tenant)
			require.NoError(t, err)
			require.Equal(t, 1, result.Skipped)
			require.Equal(t, tc.want, publisher.kinds())
			for _, event := range publisher.events {
				require.Equal(t, occurrenceID, event.SubjectID)
			}
		})
	}
}
