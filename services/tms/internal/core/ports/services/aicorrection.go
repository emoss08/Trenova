package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CaptureShipmentDraftCorrectionRequest struct {
	Draft        *documentshipmentdraft.DocumentShipmentDraft
	ShipmentID   pulid.ID
	CapturedByID pulid.ID
	TenantInfo   pagination.TenantInfo
}

type PurgeExpiredAICorrectionsRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
}

type AICorrectionService interface {
	CaptureShipmentDraft(
		ctx context.Context,
		req *CaptureShipmentDraftCorrectionRequest,
	) (*aicorrection.Correction, error)
	PurgeExpired(ctx context.Context, req PurgeExpiredAICorrectionsRequest) (int64, error)
}
