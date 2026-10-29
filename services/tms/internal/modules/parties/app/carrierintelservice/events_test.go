package carrierintelservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingPublisher struct {
	events []agenteventports.AgentEvent
}

func (r *recordingPublisher) Publish(_ context.Context, event *agenteventports.AgentEvent) {
	r.events = append(r.events, *event)
}

func findingOf(
	severity carrierintel.Severity,
	status carrierintelevent.EventStatus,
) *carrierintelevent.CarrierIntelEvent {
	return &carrierintelevent.CarrierIntelEvent{
		ID:       pulid.MustNew("ciev_"),
		Severity: severity,
		Status:   status,
		Category: carrierintel.SectionAuthority,
	}
}

// The filter that decides what a person is shown decides what wakes a run:
// one threshold, not two that can drift apart.
func TestPublishEventsToAgents_MatchesTheTowerFilter(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	cases := []struct {
		name  string
		event *carrierintelevent.CarrierIntelEvent
		want  bool
	}{
		{
			name:  "critical wakes the desk",
			event: findingOf(carrierintel.SeverityCritical, carrierintelevent.EventStatusOpen),
			want:  true,
		},
		{
			name:  "high wakes the desk",
			event: findingOf(carrierintel.SeverityHigh, carrierintelevent.EventStatusOpen),
			want:  true,
		},
		{
			name:  "medium wakes the desk",
			event: findingOf(carrierintel.SeverityMedium, carrierintelevent.EventStatusOpen),
			want:  true,
		},
		{
			name:  "low is part of the record, not news",
			event: findingOf(carrierintel.SeverityLow, carrierintelevent.EventStatusOpen),
			want:  false,
		},
		{
			name:  "info is part of the record, not news",
			event: findingOf(carrierintel.SeverityInfo, carrierintelevent.EventStatusOpen),
			want:  false,
		},
		{
			name: "a finding that arrives already closed wakes nobody",
			event: findingOf(
				carrierintel.SeverityCritical,
				carrierintelevent.EventStatusResolved,
			),
			want: false,
		},
		{
			name: "an override lands acknowledged and wakes nobody",
			event: findingOf(
				carrierintel.SeverityInfo,
				carrierintelevent.EventStatusAcknowledged,
			),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			publisher := &recordingPublisher{}
			svc := &Service{l: zap.NewNop(), publisher: publisher}

			svc.publishEventsToAgents(
				t.Context(),
				tenant,
				[]*carrierintelevent.CarrierIntelEvent{tc.event},
			)

			if !tc.want {
				require.Empty(t, publisher.events)
				return
			}

			require.Len(t, publisher.events, 1)
			require.Equal(t, agenttrigger.EventCarrierIntelEventOpened, publisher.events[0].Kind)
			require.Equal(t, tc.event.ID, publisher.events[0].SubjectID)
			require.Equal(t, tenant, publisher.events[0].TenantInfo)
		})
	}
}
