package ediinboundservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingPublisher struct {
	events []agentports.AgentEvent
}

func (r *recordingPublisher) Publish(_ context.Context, event agentports.AgentEvent) {
	r.events = append(r.events, event)
}

func TestProcessInboundFile_WakesIntakeOnlyOnQuarantine(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		rawContent string
		wantStatus ediinboundfile.InboundFileStatus
		wantEvents int
	}{
		{
			name:       "a file that cannot be parsed wakes intake",
			rawContent: "this is not x12",
			wantStatus: ediinboundfile.InboundFileStatusQuarantined,
			wantEvents: 1,
		},
		{
			name:       "an empty file is quarantined and wakes intake",
			rawContent: "",
			wantStatus: ediinboundfile.InboundFileStatusQuarantined,
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
				require.Equal(t, agenttrigger.EventEDIFileQuarantined, event.Kind)
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

	fixture := newInboundFixture(t, "")
	starter := fixture.expectActiveVersion(t)
	fixture.file.RawContent = starterSample(t, starter, "samples/tender-original.x12")
	publisher := &recordingPublisher{}
	fixture.service.publisher = publisher

	fixture.expectFileLoadAndUpdates(t)
	fixture.expectPartnerLoad()
	fixture.expectInboundDocumentType(edidocument.TransactionSet204)
	fixture.expectMessageCreate()
	fixture.expectFlows()

	fixture.mappingProfileRepo.EXPECT().
		GetMappingItems(mock.Anything, mock.Anything).
		Return([]*edi.EDIMappingProfileItem{}, nil)
	fixture.transferRepo.EXPECT().
		CreateTransfer(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, transfer *editransfer.EDITransfer) (*editransfer.EDITransfer, error) {
			transfer.ID = pulid.MustNew("edilt_")
			return transfer, nil
		}).
		Once()

	file, err := fixture.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     fixture.file.ID,
		TenantInfo: fixture.tenantInfo(),
	})

	require.NoError(t, err)
	require.Equal(t, ediinboundfile.InboundFileStatusPartiallyProcessed, file.Status)
	require.Len(t, publisher.events, 1)
	require.Equal(t, agenttrigger.EventEDITenderReceived, publisher.events[0].Kind)
	require.Equal(t, fixture.file.ID, publisher.events[0].SubjectID)
	require.Equal(t, fixture.tenantInfo(), publisher.events[0].TenantInfo)
}
