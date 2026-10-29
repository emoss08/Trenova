package detentionflows

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingPublisher struct {
	events []agenteventports.AgentEvent
}

func (r *recordingPublisher) Publish(_ context.Context, event agenteventports.AgentEvent) {
	r.events = append(r.events, event)
}

func (r *recordingPublisher) kinds() []agenttrigger.EventKind {
	if len(r.events) == 0 {
		return nil
	}
	out := make([]agenttrigger.EventKind, 0, len(r.events))
	for _, event := range r.events {
		out = append(out, event.Kind)
	}

	return out
}

type silentProjector struct{}

func (silentProjector) Upsert(context.Context, watchtowerports.WatchtowerItemInput) {}

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

func eventWiring(publisher agenteventports.AgentEventPublisher) *wiring {
	return &wiring{Logger: zap.NewNop(), Watchtower: silentProjector{}, Publisher: publisher}
}

type stubOccurrenceRepo struct {
	detentionports.DetentionOccurrenceRepository
	due []*detention.DetentionOccurrence
}

func (s stubOccurrenceRepo) ListNoticesDue(
	context.Context,
	*detentionports.ListNoticesDueRequest,
) ([]*detention.DetentionOccurrence, error) {
	return s.due, nil
}

type stubPolicyRepo struct {
	detentionports.DetentionPolicyRepository
	policy *detentionpolicy.DetentionPolicy
}

func (s stubPolicyRepo) GetByID(
	context.Context,
	*detentionports.GetDetentionPolicyByIDRequest,
) (*detentionpolicy.DetentionPolicy, error) {
	return s.policy, nil
}

func TestSweepNoticesDue_RaisesTheNoticesItLeavesOnTheDesk(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	policyID := pulid.MustNew("dp_")
	occurrenceID := pulid.MustNew("do_")

	cases := []struct {
		name   string
		policy *detentionpolicy.DetentionPolicy
		want   []agenttrigger.EventKind
	}{
		{
			name: "a policy that leaves sending to a person wakes the desk",
			policy: &detentionpolicy.DetentionPolicy{
				ID:             policyID,
				AutoSendNotice: false,
			},
			want: []agenttrigger.EventKind{agenttrigger.EventDetentionNoticeDue},
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
			w := eventWiring(publisher)
			w.OccurrenceRepo = stubOccurrenceRepo{
				due: []*detention.DetentionOccurrence{
					{ID: occurrenceID, DetentionPolicyID: &policyID},
				},
			}
			w.PolicyRepo = stubPolicyRepo{policy: tc.policy}
			w.Now = func() int64 { return 1_767_225_600 }
			svc := wire(w)

			result, err := svc.notices.SweepNoticesDue(t.Context(), tenant)
			require.NoError(t, err)
			require.Equal(t, 1, result.Skipped)
			require.Equal(t, tc.want, publisher.kinds())
			for _, event := range publisher.events {
				require.Equal(t, occurrenceID, event.SubjectID)
			}
		})
	}
}
