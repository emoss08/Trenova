package ediinboundservice

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
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

	raw := strings.Replace(partnerTender, "N1*SH*ACME DC", "N1**ACME DC", 1)
	fixture := newInboundFixture(t, raw)
	fixture.expectFileLoadAndUpdates(t)
	fixture.expectPartnerLoad()
	fixture.expectInboundDocumentType(edi.TransactionSet204)

	var (
		mu       sync.Mutex
		messages []ediports.CreateEDIMessageWithDiagnosticsRequest
	)
	fixture.messageRepo.EXPECT().
		CreateMessageWithDiagnostics(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req ediports.CreateEDIMessageWithDiagnosticsRequest) (*edi.EDIMessage, error) {
			mu.Lock()
			defer mu.Unlock()
			messages = append(messages, req)
			message := *req.Message
			message.ID = pulid.MustNew("edimsg_")
			return &message, nil
		}).
		Twice()

	fixture.documentProfileRepo.EXPECT().
		GetActivePartnerDocumentProfile(mock.Anything, ediports.GetActiveEDIPartnerDocumentProfileRequest{
			PartnerID:      fixture.partner.ID,
			TenantInfo:     fixture.tenantInfo(),
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionInbound,
		}).
		Return(&edi.EDIPartnerDocumentProfile{
			Acknowledgment: edi.AcknowledgmentConfig{Expected: true, Type: edi.AcknowledgmentType997},
		}, nil).
		Once()
	ackProfile := &edi.EDIPartnerDocumentProfile{
		ID:             pulid.MustNew("edipdp_"),
		EDIPartnerID:   fixture.partner.ID,
		DocumentTypeID: pulid.MustNew("edidt_"),
		TransactionSet: edi.TransactionSet997,
		Direction:      edi.DocumentDirectionOutbound,
		Envelope:       edi.DefaultX12EnvelopeSettings(),
	}
	ackProfile.Envelope.InterchangeSenderID = "TRENOVA"
	ackProfile.Envelope.InterchangeReceiverID = "PARTNER"
	fixture.documentProfileRepo.EXPECT().
		GetActivePartnerDocumentProfile(mock.Anything, ediports.GetActiveEDIPartnerDocumentProfileRequest{
			PartnerID:      fixture.partner.ID,
			TenantInfo:     fixture.tenantInfo(),
			TransactionSet: edi.TransactionSet997,
			Direction:      edi.DocumentDirectionOutbound,
		}).
		Return(ackProfile, nil).
		Once()
	fixture.messageRepo.EXPECT().
		GetMessageByIdempotencyKey(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("EDIMessage")).
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
		Once()

	file, err := fixture.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     fixture.file.ID,
		TenantInfo: fixture.tenantInfo(),
	})
	require.NoError(t, err)
	assert.Equal(t, edi.InboundFileStatusQuarantined, file.Status, "the only tender was rejected")
	assert.Contains(t, file.FailureReason, "rejected, so it was not processed")
	assert.Contains(t, file.FailureReason, "N101 (Entity Identifier Code) is mandatory but is empty")

	require.Len(t, messages, 2)
	tender := messages[0]
	assert.Equal(t, edi.MessageStatusFailed, tender.Message.Status)
	require.Len(t, tender.Diagnostics, 1)
	assert.Equal(t, "x12.mandatoryElementMissing", tender.Diagnostics[0].Code)
	assert.Equal(t, "N1", tender.Diagnostics[0].SegmentID)
	assert.Equal(t, 1, tender.Diagnostics[0].ElementPosition)
	assert.Equal(t, "loop 0310 · segment 7 · N101", tender.Diagnostics[0].Path)

	ack := messages[1].Message
	assert.Equal(t, edi.TransactionSet997, ack.TransactionSet)
	assert.Equal(t, "000000007", ack.InterchangeControlNumber)
	assert.Contains(t, ack.RawX12,
		"AK1*SM*201~AK2*204*0001~AK3*N1*7*0310*8~AK4*1*98*1~AK5*R*5~AK9*R*1*1*0~")
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
