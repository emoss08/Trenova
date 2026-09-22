package carrierintelservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
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

func findingOf(severity carrierintel.Severity, status carrierintel.EventStatus) *carrierintel.CarrierIntelEvent {
	return &carrierintel.CarrierIntelEvent{
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
		event *carrierintel.CarrierIntelEvent
		want  bool
	}{
		{
			name:  "critical wakes the desk",
			event: findingOf(carrierintel.SeverityCritical, carrierintel.EventStatusOpen),
			want:  true,
		},
		{
			name:  "high wakes the desk",
			event: findingOf(carrierintel.SeverityHigh, carrierintel.EventStatusOpen),
			want:  true,
		},
		{
			name:  "medium wakes the desk",
			event: findingOf(carrierintel.SeverityMedium, carrierintel.EventStatusOpen),
			want:  true,
		},
		{
			name:  "low is part of the record, not news",
			event: findingOf(carrierintel.SeverityLow, carrierintel.EventStatusOpen),
			want:  false,
		},
		{
			name:  "info is part of the record, not news",
			event: findingOf(carrierintel.SeverityInfo, carrierintel.EventStatusOpen),
			want:  false,
		},
		{
			name: "a finding that arrives already closed wakes nobody",
			event: findingOf(
				carrierintel.SeverityCritical,
				carrierintel.EventStatusResolved,
			),
			want: false,
		},
		{
			name: "an override lands acknowledged and wakes nobody",
			event: findingOf(
				carrierintel.SeverityInfo,
				carrierintel.EventStatusAcknowledged,
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
				[]*carrierintel.CarrierIntelEvent{tc.event},
			)

			if !tc.want {
				require.Empty(t, publisher.events)
				return
			}

			require.Len(t, publisher.events, 1)
			require.Equal(t, agent.EventCarrierIntelEventOpened, publisher.events[0].Kind)
			require.Equal(t, tc.event.ID, publisher.events[0].SubjectID)
			require.Equal(t, tenant, publisher.events[0].TenantInfo)
		})
	}
}
