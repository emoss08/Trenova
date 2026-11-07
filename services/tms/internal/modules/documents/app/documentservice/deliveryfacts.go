package documentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type SetDeliveryFactsRequest struct {
	DocumentID pulid.ID
	TenantInfo pagination.TenantInfo
	Facts      document.DeliveryFacts
	Version    int64
	Actor      permissionports.RequestActor
}

func (s *Service) SetDeliveryFacts(
	ctx context.Context,
	req *SetDeliveryFactsRequest,
) (*document.Document, error) {
	if req == nil || req.DocumentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrRequired,
			"Document is required",
		)
	}

	personID := req.Actor.PersonUserID()
	if personID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Only a person can record what a document says",
		)
	}
	if err := s.authorizeSignature(ctx, &SetSignatureRequest{
		DocumentID: req.DocumentID,
		TenantInfo: req.TenantInfo,
		Actor:      req.Actor,
	}); err != nil {
		return nil, err
	}

	current, err := s.visibleDocument(ctx, documentports.GetDocumentByIDRequest{
		ID:         req.DocumentID,
		TenantInfo: req.TenantInfo,
	}, &req.Actor)
	if err != nil {
		return nil, err
	}
	if req.Version != current.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"This document changed since you opened it; refresh and try again",
		)
	}

	facts := req.Facts
	facts.Normalize()
	facts.Source = document.FactsSourcePerson
	facts.ReadFields = nil
	facts.UpdatedByID = personID
	facts.UpdatedAt = timeutils.NowUnix()

	multiErr := errortypes.NewMultiError()
	facts.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	version := req.Version
	updated, err := s.repo.RecordDeliveryFacts(ctx, &documentports.RecordDeliveryFactsRequest{
		ID:         current.ID,
		TenantInfo: req.TenantInfo,
		Facts:      &facts,
		Version:    &version,
	})
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"This document changed since you opened it; refresh and try again",
		)
	}

	if err = s.auditService.LogAction(ctx, &auditports.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         req.Actor.AuditActor().UserID,
		PreviousState:  jsonutils.MustToJSON(current),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	},
		auditservice.WithComment("Delivery details recorded"),
		auditservice.WithDiff(current, updated),
		auditservice.WithMetadata(map[string]any{
			metadataResourceID:   updated.ResourceID,
			metadataResourceType: updated.ResourceType,
		}),
	); err != nil {
		s.l.Warn("failed to log document delivery facts change", zap.Error(err))
	}

	s.publishDocumentInvalidation(ctx, updated, &req.Actor, permission.OpUpdate)

	return updated, nil
}
