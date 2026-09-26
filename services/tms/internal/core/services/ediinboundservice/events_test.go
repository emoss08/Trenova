package ediinboundservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingPublisher struct {
	events []services.AgentEvent
}

func (r *recordingPublisher) Publish(_ context.Context, event services.AgentEvent) {
	r.events = append(r.events, event)
}

func TestProcessInboundFile_WakesIntakeOnlyOnQuarantine(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		rawContent string
		wantStatus edi.InboundFileStatus
		wantEvents int
	}{
		{
			name:       "a file that cannot be parsed wakes intake",
			rawContent: "this is not x12",
			wantStatus: edi.InboundFileStatusQuarantined,
			wantEvents: 1,
		},
		{
			name:       "an empty file is quarantined and wakes intake",
			rawContent: "",
			wantStatus: edi.InboundFileStatusQuarantined,
			wantEvents: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newInboundFixture(t, tc.rawContent)
			publisher := &recordingPublisher{}
			fixture.service.publisher = publisher
			fixture.expectFileLoadAndUpdates(t)

			file, err := fixture.service.ProcessInboundFile(
				t.Context(),
				&ProcessInboundFileRequest{
					FileID:     fixture.file.ID,
					TenantInfo: fixture.tenantInfo(),
				},
			)

			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, file.Status)
			require.Len(t, publisher.events, tc.wantEvents)
			for _, event := range publisher.events {
				require.Equal(t, agent.EventEDIFileQuarantined, event.Kind)
				require.Equal(t, fixture.file.ID, event.SubjectID)
				require.Equal(t, fixture.tenantInfo(), event.TenantInfo)
			}
		})
	}
}

// A file that reaches transfers is doing its job, however much of it needs a
// person's attention later: the quarantine desk is woken by a held-back file,
// not by an unmapped one. The tender it carried waits on an answer, so the
// file is raised once as a tender received, for whoever answers tenders.
func TestProcessInboundFile_RaisesTheTenderItCarriedAndNotAQuarantine(t *testing.T) {
	t.Parallel()

	raw := renderBase204(t, sampleTenderPayload())
	fixture := newInboundFixture(t, raw)
	publisher := &recordingPublisher{}
	fixture.service.publisher = publisher

	fixture.expectFileLoadAndUpdates(t)
	fixture.expectPartnerLoad()
	fixture.expectInboundDocumentType(edi.TransactionSet204)
	fixture.expectMessageCreate()
	fixture.expectNoInboundAckProfile(edi.TransactionSet204)

	fixture.mappingProfileRepo.EXPECT().
		GetMappingItems(mock.Anything, mock.Anything).
		Return([]*edi.EDIMappingProfileItem{}, nil).
		Once()
	fixture.transferRepo.EXPECT().
		CreateTransfer(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, transfer *edi.EDITransfer) (*edi.EDITransfer, error) {
			transfer.ID = pulid.MustNew("edilt_")
			return transfer, nil
		}).
		Once()

	file, err := fixture.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     fixture.file.ID,
		TenantInfo: fixture.tenantInfo(),
	})

	require.NoError(t, err)
	require.Equal(t, edi.InboundFileStatusPartiallyProcessed, file.Status)
	require.Len(t, publisher.events, 1)
	require.Equal(t, agent.EventEDITenderReceived, publisher.events[0].Kind)
	require.Equal(t, fixture.file.ID, publisher.events[0].SubjectID)
	require.Equal(t, fixture.tenantInfo(), publisher.events[0].TenantInfo)
}
