package ediinboundservice

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// A tender that breaks the partner's guide is recorded as failed with what was wrong,
// is not turned into a transfer, and the group's 997 says so, segment by segment.
func TestProcessInboundFile_RejectsATenderThatBreaksItsGuideAndSaysWhyInThe997(t *testing.T) {
	t.Parallel()

	fixture := newInboundFixture(t, "")
	starter := fixture.expectActiveVersion(t, "997")
	fixture.file.RawContent = strings.Replace(
		starterSample(t, starter, "samples/tender-original.x12"),
		"N1*SH*ORIGIN DC",
		"N1**ORIGIN DC",
		1,
	)
	fixture.expectFileLoadAndUpdates(t)
	fixture.expectPartnerLoad()
	fixture.expectInboundDocumentType(edi.TransactionSet204)
	fixture.expectFlows()
	fixture.mappingProfileRepo.EXPECT().
		GetMappingItems(mock.Anything, mock.Anything).
		Return([]*edi.EDIMappingProfileItem{}, nil).
		Maybe()

	var (
		mu       sync.Mutex
		messages []edimessageports.CreateEDIMessageWithDiagnosticsRequest
	)
	fixture.messageRepo.EXPECT().
		CreateMessageWithDiagnostics(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req edimessageports.CreateEDIMessageWithDiagnosticsRequest) (*edi.EDIMessage, error) {
			mu.Lock()
			defer mu.Unlock()
			messages = append(messages, req)
			message := *req.Message
			message.ID = pulid.MustNew("edimsg_")
			return &message, nil
		}).
		Twice()
	fixture.messageRepo.EXPECT().
		GetMessageByIdempotencyKey(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("EDIMessage")).
		Once()
	fixture.documentTypeRepo.EXPECT().
		ListDocumentTypes(mock.Anything, edipartnerports.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet997,
			Direction:      edi.DocumentDirectionOutbound,
		}).
		Return([]*edi.EDIDocumentType{{
			ID:             pulid.MustNew("edidt_"),
			TransactionSet: edi.TransactionSet997,
			Direction:      edi.DocumentDirectionOutbound,
		}}, nil).
		Once()
	fixture.controlNumberRepo.EXPECT().
		AllocateControlNumbers(mock.Anything, mock.Anything).
		Return(map[edi.ControlNumberKind]int64{
			edi.ControlNumberKindInterchange: 7,
			edi.ControlNumberKindGroup:       7,
			edi.ControlNumberKindTransaction: 7,
		}, nil).
		Once()
	internal := *fixture.partner
	internal.Kind = edi.PartnerKindInternal
	fixture.partnerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&internal, nil).
		Maybe()

	file, err := fixture.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     fixture.file.ID,
		TenantInfo: fixture.tenantInfo(),
	})
	require.NoError(t, err)
	assert.Equal(t, edi.InboundFileStatusQuarantined, file.Status, "the only tender was rejected")
	assert.Contains(t, file.FailureReason, "rejected, so it was not processed")

	require.Len(t, messages, 2)
	tender := messages[0]
	assert.Equal(t, edi.MessageStatusFailed, tender.Message.Status)
	require.NotEmpty(t, tender.Diagnostics)
	assert.Equal(t, "N1", tender.Diagnostics[0].SegmentID)
	assert.Equal(t, 1, tender.Diagnostics[0].ElementPosition)

	ack := messages[1].Message
	assert.Equal(t, edi.TransactionSet997, ack.TransactionSet)
	assert.Equal(t, "000000007", ack.InterchangeControlNumber)
	assert.Contains(t, ack.RawX12, "AK3*N1*")
	assert.Contains(t, ack.RawX12, "AK5*R")
	require.NotNil(t, ack.PayloadSnapshot.FunctionalAcknowledgment)
	assert.Equal(t, "R", ack.PayloadSnapshot.FunctionalAcknowledgment.GroupAcknowledgmentCode)
}

// A group made only of acknowledgments is never acknowledged.
func TestAcknowledgedSet_SkipsAGroupOfAcknowledgments(t *testing.T) {
	t.Parallel()

	interchange := &parsedInterchange{transactions: []parsedTransaction{
		{set: edi.TransactionSet997, group: 0},
		{set: edi.TransactionSet214, group: 1},
		{set: edi.TransactionSet999, group: 1},
	}}
	assert.Empty(t, acknowledgedSet(interchange, 0))
	assert.Equal(t, edi.TransactionSet214, acknowledgedSet(interchange, 1))
}
