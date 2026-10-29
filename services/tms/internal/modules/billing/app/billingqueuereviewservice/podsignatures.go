package billingqueuereviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/zap"
)

const (
	podRedetectShipmentLimit = 500
	podRedetectDocumentLimit = 200
	podDocumentResource      = "shipment"
)

func (s *Service) RedetectPODSignatures(
	ctx context.Context,
	req *billingqueueports.RedetectPODSignaturesRequest,
	actor *permissionports.RequestActor,
) (*billingqueueports.RedetectPODSignaturesResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"tenantInfo",
			errortypes.ErrRequired,
			"Tenant is required",
		)
	}
	if actor.PersonUserID().IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Only a person can re-read the signatures on queued PODs",
		)
	}
	if s.documentRepo == nil || s.documentContent == nil {
		return nil, errortypes.NewBusinessError(
			"Signature detection is not available on this server",
		)
	}

	shipmentIDs, err := s.repo.ListOpenShipmentIDs(
		ctx,
		&billingqueueports.ListOpenShipmentIDsRequest{
			TenantInfo: req.TenantInfo,
			Limit:      podRedetectShipmentLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	result := &billingqueueports.RedetectPODSignaturesResult{Shipments: len(shipmentIDs)}
	if len(shipmentIDs) == 0 {
		return result, nil
	}

	resourceIDs := make([]string, 0, len(shipmentIDs))
	for _, id := range shipmentIDs {
		resourceIDs = append(resourceIDs, id.String())
	}
	docs, err := s.documentRepo.GetByResourceIDs(
		ctx,
		&documentports.GetDocumentsByResourceIDsRequest{
			TenantInfo:          req.TenantInfo,
			ResourceType:        podDocumentResource,
			ResourceIDs:         resourceIDs,
			IncludeDocumentType: true,
		},
	)
	if err != nil {
		return nil, err
	}

	var firstErr error
	for _, doc := range docs {
		if !needsSignature(doc) {
			continue
		}
		result.Documents++

		switch {
		case doc.SignatureSetByPerson():
			result.SetByPerson++
		case doc.SignatureStatus != document.SignatureStatusUnknown:
			result.AlreadyRead++
		case !doc.ProcessingProfile.SupportsIntelligence():
			result.NotReadByMachine++
		case result.Queued+result.Failed >= podRedetectDocumentLimit:
			result.Deferred++
		default:
			if err = s.documentContent.Reextract(ctx, doc.ID, req.TenantInfo); err != nil {
				result.Failed++
				if firstErr == nil {
					firstErr = err
				}
				s.l.Warn("could not re-read a queued POD's signature",
					zap.String("documentId", doc.ID.String()),
					zap.Error(err),
				)
				continue
			}
			result.Queued++
		}
	}

	if result.Queued == 0 && firstErr != nil {
		return nil, firstErr
	}

	return result, nil
}

func needsSignature(doc *document.Document) bool {
	return doc != nil &&
		doc.IsCurrentVersion &&
		doc.Status != document.StatusRejected &&
		doc.Status != document.StatusArchived &&
		doc.DocumentType != nil &&
		doc.DocumentType.RequiresSignature
}
