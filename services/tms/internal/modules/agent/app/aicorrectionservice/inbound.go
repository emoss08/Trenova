package aicorrectionservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	InboundClassificationField = "classification"
	inboundFactFieldPrefix     = "fact."
)

type InboundRecorderParams struct {
	fx.In

	Logger *zap.Logger
	Repo   aicorrectionports.AICorrectionRepository
}

type InboundRecorder struct {
	l    *zap.Logger
	repo correctionStore
}

func NewInboundRecorder(p InboundRecorderParams) triageports.InboundCorrectionRecorder {
	return &InboundRecorder{
		l:    p.Logger.Named("service.aicorrection.inbound"),
		repo: p.Repo,
	}
}

func (r *InboundRecorder) RecordInboundCorrection(
	ctx context.Context,
	correction *triageports.InboundCorrection,
) error {
	entity := InboundCorrectionEntity(correction)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	if _, err := r.repo.Upsert(ctx, entity); err != nil {
		return fmt.Errorf("record the inbound correction: %w", err)
	}

	r.l.Debug("recorded an inbound classification correction",
		zap.String("messageId", correction.MessageID.String()),
		zap.Int("corrected", entity.CorrectedCount))

	return nil
}

func InboundCorrectionEntity(correction *triageports.InboundCorrection) *aicorrection.Correction {
	predicted := &aicorrection.Snapshot{
		Fields: make(map[string]string, len(correction.Facts)+1),
		Stops:  []aicorrection.StopSnapshot{},
	}
	confirmed := &aicorrection.Snapshot{
		Fields: make(map[string]string, len(correction.Facts)+1),
		Stops:  []aicorrection.StopSnapshot{},
	}
	results := make([]aicorrection.FieldResult, 0, len(correction.Facts)+1)

	predicted.Fields[InboundClassificationField] = string(correction.PredictedClassification)
	confirmed.Fields[InboundClassificationField] = string(correction.ConfirmedClassification)
	results = append(results, aicorrection.FieldResult{
		Key:        InboundClassificationField,
		Predicted:  string(correction.PredictedClassification),
		Confirmed:  string(correction.ConfirmedClassification),
		Outcome:    classificationOutcome(correction),
		Source:     "ai",
		Confidence: correction.PredictedConfidence,
	})

	for _, fact := range correction.Facts {
		key := inboundFactFieldPrefix + fact.Key
		predicted.Fields[key] = fact.Value
		results = append(results, factResult(key, fact))
	}

	entity := &aicorrection.Correction{
		OrganizationID: correction.TenantInfo.OrgID,
		BusinessUnitID: correction.TenantInfo.BuID,
		Task:           aicorrection.TaskInboundClassification,
		SourceType:     aicorrection.SourceInboundMessage,
		SourceID:       correction.MessageID,
		SubjectType:    aicorrection.SubjectInboundMessage,
		SubjectID:      correction.MessageID,
		CapturedByID:   correction.CapturedByID,
		DocumentKind: stringutils.TruncateRunes(
			string(correction.ConfirmedClassification),
			aicorrection.MaxDocumentKindLength,
		),
		DocumentFingerprint: stringutils.TruncateRunes(
			correction.SenderDomain,
			aicorrection.MaxFingerprintLength,
		),
		PredictedConfidence: min(max(correction.PredictedConfidence, 0), 1),
		Predicted:           predicted,
		Confirmed:           confirmed,
		FieldResults:        results,
		CapturedAt:          correction.CapturedAt,
	}
	entity.ApplyTally()

	return entity
}

func classificationOutcome(correction *triageports.InboundCorrection) aicorrection.Outcome {
	if correction.PredictedClassification == correction.ConfirmedClassification {
		return aicorrection.OutcomeCorrect
	}

	return aicorrection.OutcomeCorrected
}

func factResult(key string, fact inboundmessage.Fact) aicorrection.FieldResult {
	result := aicorrection.FieldResult{
		Key:       key,
		Predicted: fact.Value,
		Source:    "ai",
		Outcome:   aicorrection.OutcomeUnconfirmed,
	}
	if fact.Flagged {
		result.Outcome = aicorrection.OutcomeCorrected
	}

	return result
}
