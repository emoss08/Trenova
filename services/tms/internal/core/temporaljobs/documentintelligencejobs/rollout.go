package documentintelligencejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (a *Activities) rolloutPreference(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	payload *ProcessDocumentAIExtractionPayload,
) pulid.ID {
	if a.rollout == nil {
		return pulid.Nil
	}

	preferred, err := a.rollout.AssignExtraction(ctx, &services.AssignExtractionRolloutRequest{
		TenantInfo:  tenantInfo,
		DocumentID:  payload.DocumentID,
		ExtractedAt: payload.ExtractedAt,
	})
	if err != nil {
		a.logger.Warn(
			"could not assign the extraction to a rollout side; using the usual providers",
			zap.String("documentId", payload.DocumentID.String()),
			zap.Error(err),
		)
		return pulid.Nil
	}

	return preferred
}

func (a *Activities) settleRollout(
	ctx context.Context,
	payload *ApplyDocumentAIExtractionPayload,
	tenantInfo pagination.TenantInfo,
	outcome extractionrollout.Outcome,
	row *documentaiextraction.Extraction,
) {
	if a.rollout == nil {
		return
	}

	req := &services.SettleExtractionRolloutRequest{
		TenantInfo:  tenantInfo,
		DocumentID:  payload.DocumentID,
		ExtractedAt: payload.ExtractedAt,
		Outcome:     outcome,
	}
	if row != nil {
		req.ServedProviderID = row.ProviderID
		req.ServedModel = row.Model
	}
	if err := a.rollout.SettleExtraction(ctx, req); err != nil {
		a.logger.Warn("could not record the extraction's rollout outcome",
			zap.String("documentId", payload.DocumentID.String()),
			zap.Error(err),
		)
	}
}

func rolloutOutcome(
	completion *AsyncAIExtractionCompletion,
	acceptance aiAcceptanceStatus,
) extractionrollout.Outcome {
	switch {
	case acceptance == aiAcceptanceStatusAccepted:
		return extractionrollout.OutcomeAccepted
	case completion.Status == services.AIBackgroundExtractionStatusFailed:
		return extractionrollout.OutcomeFailed
	default:
		return extractionrollout.OutcomeRejected
	}
}
