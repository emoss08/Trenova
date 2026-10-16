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

type SetSignatureRequest struct {
	DocumentID pulid.ID
	TenantInfo pagination.TenantInfo
	Status     document.SignatureStatus
	SignedAt   *int64
	Version    int64
	Actor      permissionports.RequestActor
}

var signatureAuditComments = map[document.SignatureStatus]string{
	document.SignatureStatusSigned:   "Document marked signed",
	document.SignatureStatusUnsigned: "Document marked unsigned",
	document.SignatureStatusUnknown:  "Document signature marked unknown",
}

func (s *Service) SetSignature(
	ctx context.Context,
	req *SetSignatureRequest,
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
			"Only a person can record whether a document is signed",
		)
	}

	if err := s.authorizeSignature(ctx, req); err != nil {
		return nil, err
	}

	current, err := s.repo.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         req.DocumentID,
		TenantInfo: req.TenantInfo,
	})
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

	if current.SignatureSetByPerson() && current.SignatureMatches(req.Status, req.SignedAt) {
		return current, nil
	}

	after := *current
	if err = after.MarkSignature(&document.SignatureMark{
		Status:   req.Status,
		SignedAt: req.SignedAt,
		By:       personID,
		At:       timeutils.NowUnix(),
	}); err != nil {
		return nil, err
	}

	multiErr := errortypes.NewMultiError()
	after.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.repo.Update(ctx, &after)
	if err != nil {
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         req.Actor.AuditActor().UserID,
		PreviousState:  jsonutils.MustToJSON(current),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	},
		auditservice.WithComment(signatureAuditComments[updated.SignatureStatus]),
		auditservice.WithDiff(current, updated),
		auditservice.WithRequest(ctx),
		auditservice.WithMetadata(map[string]any{
			metadataResourceID:   updated.ResourceID,
			metadataResourceType: updated.ResourceType,
			"signatureStatus":    updated.SignatureStatus,
			"signedAt":           updated.SignedAt,
			"previousStatus":     current.SignatureStatus,
			"previousSource":     current.SignatureSource,
		}),
	); err != nil {
		s.l.Warn("failed to log document signature change", zap.Error(err))
	}

	s.publishDocumentInvalidation(ctx, updated, &req.Actor, permission.OpUpdate)

	return updated, nil
}

func (s *Service) authorizeSignature(ctx context.Context, req *SetSignatureRequest) error {
	if s.permissions == nil {
		return errortypes.NewAuthorizationError(
			"You don't have permission to change this document",
		)
	}

	documentID := req.DocumentID
	result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  req.Actor.PrincipalType,
		PrincipalID:    req.Actor.PrincipalID,
		UserID:         req.Actor.UserID,
		APIKeyID:       req.Actor.APIKeyID,
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Resource:       permission.ResourceDocument.String(),
		Operation:      permission.OpUpdate,
		ResourceID:     &documentID,
	})
	if err != nil {
		return err
	}
	if result == nil || !result.Allowed {
		return errortypes.NewAuthorizationError(
			"You don't have permission to change this document",
		)
	}

	return nil
}
