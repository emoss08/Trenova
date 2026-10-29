package detentionsync

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPublishOccurrenceOpened_FiresOnlyWhenTheClockStarts(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	occurrenceID := pulid.MustNew("do_")

	cases := []struct {
		name     string
		saved    *detention.DetentionOccurrence
		existing *detention.DetentionOccurrence
		want     []agenttrigger.EventKind
	}{
		{
			name:  "a clock that has just started wakes the desk",
			saved: &detention.DetentionOccurrence{ID: occurrenceID, IsOpen: true},
			want:  []agenttrigger.EventKind{agenttrigger.EventDetentionOccurrenceOpened},
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
			want:     []agenttrigger.EventKind{agenttrigger.EventDetentionOccurrenceOpened},
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

type recordingPublisher struct {
	events []agentports.AgentEvent
}

func (r *recordingPublisher) Publish(_ context.Context, event agentports.AgentEvent) {
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

func eventService(publisher agentports.AgentEventPublisher) *Service {
	return &Service{l: zap.NewNop(), watchtower: silentProjector{}, publisher: publisher}
}
