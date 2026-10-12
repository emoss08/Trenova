package aicorrectionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inboundCorrection() *triageports.InboundCorrection {
	return &triageports.InboundCorrection{
		TenantInfo:              pagination.TenantInfo{OrgID: testOrg, BuID: testBU},
		MessageID:               "imsg_test",
		CapturedByID:            testUser,
		SenderDomain:            "acme.com",
		PredictedClassification: inboundmessage.ClassificationInvoice,
		ConfirmedClassification: inboundmessage.ClassificationRateConfirmation,
		PredictedConfidence:     1.4,
		Facts: []inboundmessage.Fact{
			{Key: "rate", Value: "$1,800"},
			{Key: "pickup", Value: "Oct 14", Flagged: true},
		},
		CapturedAt: capturedAt,
	}
}

func TestInboundCorrectionEntityScoresTheClassification(t *testing.T) {
	t.Parallel()

	entity := InboundCorrectionEntity(inboundCorrection())

	assert.Equal(t, aicorrection.TaskInboundClassification, entity.Task)
	assert.Equal(t, aicorrection.SourceInboundMessage, entity.SourceType)
	assert.Equal(t, aicorrection.SubjectInboundMessage, entity.SubjectType)
	assert.Equal(t, "RateConfirmation", entity.DocumentKind)
	assert.Equal(t, "acme.com", entity.DocumentFingerprint)
	assert.InDelta(t, 1.0, entity.PredictedConfidence, 0.0001)
	assert.Equal(t, "Invoice", entity.Predicted.Fields[InboundClassificationField])
	assert.Equal(t, "RateConfirmation", entity.Confirmed.Fields[InboundClassificationField])
	assert.Equal(t, "$1,800", entity.Predicted.Fields["fact.rate"])

	require.Len(t, entity.FieldResults, 3)
	assert.Equal(t, aicorrection.OutcomeCorrected, entity.FieldResults[0].Outcome)
	assert.Equal(t, aicorrection.OutcomeUnconfirmed, entity.FieldResults[1].Outcome)
	assert.Equal(t, aicorrection.OutcomeCorrected, entity.FieldResults[2].Outcome)
	assert.Equal(t, 2, entity.CorrectedCount)
	assert.Equal(t, 1, entity.UnconfirmedCount)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestInboundCorrectionEntityKeepsAConfirmedClassificationCorrect(t *testing.T) {
	t.Parallel()

	correction := inboundCorrection()
	correction.ConfirmedClassification = correction.PredictedClassification
	correction.Facts = nil

	entity := InboundCorrectionEntity(correction)

	require.Len(t, entity.FieldResults, 1)
	assert.Equal(t, aicorrection.OutcomeCorrect, entity.FieldResults[0].Outcome)
	assert.Equal(t, 1, entity.CorrectCount)
}
