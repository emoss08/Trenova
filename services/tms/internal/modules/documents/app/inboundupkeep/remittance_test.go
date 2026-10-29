package inboundupkeep

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubRemittanceIntake struct {
	req    *inboundmessageports.RemittanceIntakeRequest
	result *inboundmessageports.RemittanceIntakeResult
}

func (s *stubRemittanceIntake) IngestRemittances(
	_ context.Context,
	req *inboundmessageports.RemittanceIntakeRequest,
) (*inboundmessageports.RemittanceIntakeResult, error) {
	s.req = req
	return s.result, nil
}

func remittanceMessage(class inboundmessage.Classification) *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:             pulid.MustNew("imsg_"),
		Classification: class,
		FromAddress:    "remit@rts.test",
		ReceivedAt:     1_800_000_000,
		Facts: []inboundmessage.Fact{
			{Key: "total", Label: "Total", Value: "$2,823.45"},
			{Key: remittanceFactKey, Label: "Remittance", Value: "stale"},
		},
		Attachments: []*inboundmessage.InboundAttachment{
			{FileName: "remit.csv", ContentType: "text/csv", DocumentID: pulid.MustNew("doc_")},
			{FileName: "unstored.pdf"},
		},
	}
}

func TestIngestRemittances_HandsTheFilesToBillingAndNotesTheOutcome(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.ContentRepo{
		Message: remittanceMessage(inboundmessage.ClassificationFactorRemittance),
	}
	intake := &stubRemittanceIntake{result: &inboundmessageports.RemittanceIntakeResult{
		FactorName:    "RTS Financial",
		RemittanceIDs: []pulid.ID{pulid.MustNew("facrem_")},
	}}
	svc := &testService{
		l:           zap.NewNop(),
		messageRepo: repo,
		remittances: intake,
		notify:      &inboundqueue.Notifier{},
	}

	result, err := svc.upkeep().
		IngestRemittances(t.Context(), repo.Message.ID, pagination.TenantInfo{})
	require.NoError(t, err)
	require.Len(t, result.RemittanceIDs, 1)

	require.NotNil(t, intake.req)
	assert.Equal(t, "remit@rts.test", intake.req.FromAddress)
	require.Len(t, intake.req.Attachments, 1, "a file that was never stored has nothing to read")
	assert.Equal(t, "remit.csv", intake.req.Attachments[0].FileName)

	_, saved := repo.Snapshot()
	require.NotNil(t, saved)
	require.Len(t, saved.Facts, 2, "the earlier note is replaced, not repeated")
	assert.Equal(t, "Opened for RTS Financial", saved.Facts[1].Value)
	assert.Equal(t, inboundmessage.FactToneNeutral, saved.Facts[1].Tone)
}

func TestIngestRemittances_WarnsWhenNoFactorUsesTheAddress(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.ContentRepo{
		Message: remittanceMessage(inboundmessage.ClassificationFactorRemittance),
	}
	intake := &stubRemittanceIntake{result: &inboundmessageports.RemittanceIntakeResult{
		Problem: "No factor uses this address",
	}}
	svc := &testService{
		l:           zap.NewNop(),
		messageRepo: repo,
		remittances: intake,
		notify:      &inboundqueue.Notifier{},
	}

	_, err := svc.upkeep().IngestRemittances(t.Context(), repo.Message.ID, pagination.TenantInfo{})
	require.NoError(t, err)
	_, saved := repo.Snapshot()
	require.NotNil(t, saved)
	last := saved.Facts[len(saved.Facts)-1]
	assert.Equal(t, "No factor uses this address", last.Value)
	assert.Equal(t, inboundmessage.FactToneWarning, last.Tone)
}

func TestIngestRemittances_LeavesOtherMailAlone(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.ContentRepo{
		Message: remittanceMessage(inboundmessage.ClassificationInvoice),
	}
	intake := &stubRemittanceIntake{}
	svc := &testService{
		l:           zap.NewNop(),
		messageRepo: repo,
		remittances: intake,
		notify:      &inboundqueue.Notifier{},
	}

	result, err := svc.upkeep().
		IngestRemittances(t.Context(), repo.Message.ID, pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Empty(t, result.RemittanceIDs)
	assert.Nil(t, intake.req)
	writes, _ := repo.Snapshot()
	assert.Zero(t, writes)
}
