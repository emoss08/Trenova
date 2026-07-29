package shipmentcommentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *service) validateAttachmentDocuments(
	ctx context.Context,
	entity *shipment.ShipmentComment,
	commentID pulid.ID,
) ([]*document.Document, error) {
	entity.AttachmentDocumentIDs = uniqueMentionIDs(entity.AttachmentDocumentIDs)
	if len(entity.AttachmentDocumentIDs) == 0 {
		return nil, nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	docs, err := s.documentRepo.GetByIDs(ctx, repositories.BulkDeleteDocumentRequest{
		IDs:        entity.AttachmentDocumentIDs,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if len(docs) != len(entity.AttachmentDocumentIDs) {
		return nil, errortypes.NewValidationError(
			"attachmentDocumentIds",
			errortypes.ErrInvalid,
			"All attachment documents must belong to the current tenant",
		)
	}

	shipmentResourceID := entity.ShipmentID.String()
	commentResourceID := ""
	if !commentID.IsNil() {
		commentResourceID = commentID.String()
	}

	for _, doc := range docs {
		attachedToShipment := doc.ResourceType == "shipment" &&
			doc.ResourceID == shipmentResourceID
		attachedToComment := commentResourceID != "" &&
			doc.ResourceType == shipment.CommentAttachmentResourceType &&
			doc.ResourceID == commentResourceID
		if !attachedToShipment && !attachedToComment {
			return nil, errortypes.NewValidationError(
				"attachmentDocumentIds",
				errortypes.ErrInvalid,
				"Attachments must be uploaded against this shipment before they can be attached to a comment",
			)
		}
	}

	return docs, nil
}

func (s *service) attachDocuments(
	ctx context.Context,
	comment *shipment.ShipmentComment,
	docs []*document.Document,
) error {
	if len(docs) == 0 {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: comment.OrganizationID,
		BuID:  comment.BusinessUnitID,
	}
	commentResourceID := comment.ID.String()

	for _, doc := range docs {
		if doc.ResourceType == shipment.CommentAttachmentResourceType &&
			doc.ResourceID == commentResourceID {
			continue
		}
		if err := s.documentRepo.MoveLineageToResource(
			ctx,
			&repositories.MoveDocumentLineageRequest{
				DocumentID:   doc.ID,
				ResourceID:   commentResourceID,
				ResourceType: shipment.CommentAttachmentResourceType,
				TenantInfo:   tenantInfo,
			},
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) reconcileAttachments(
	ctx context.Context,
	comment *shipment.ShipmentComment,
	desired []*document.Document,
) error {
	tenantInfo := pagination.TenantInfo{
		OrgID: comment.OrganizationID,
		BuID:  comment.BusinessUnitID,
	}

	current, err := s.documentRepo.GetByResourceIDs(
		ctx,
		&repositories.GetDocumentsByResourceIDsRequest{
			TenantInfo:   tenantInfo,
			ResourceType: shipment.CommentAttachmentResourceType,
			ResourceIDs:  []string{comment.ID.String()},
		},
	)
	if err != nil {
		return err
	}

	desiredIDs := make(map[pulid.ID]bool, len(desired))
	for _, doc := range desired {
		desiredIDs[doc.ID] = true
	}

	removedIDs := make([]pulid.ID, 0, len(current))
	for _, doc := range current {
		if !desiredIDs[doc.ID] {
			removedIDs = append(removedIDs, doc.ID)
		}
	}

	if len(removedIDs) > 0 {
		if _, err = s.documentService.BulkDelete(ctx, &documentservice.BulkDeleteRequest{
			IDs:        removedIDs,
			TenantInfo: tenantInfo,
			UserID:     comment.UserID,
		}); err != nil {
			return err
		}
	}

	return s.attachDocuments(ctx, comment, desired)
}

func (s *service) deleteAllAttachments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	comment *shipment.ShipmentComment,
	actorID pulid.ID,
) error {
	docs, err := s.documentRepo.GetByResourceIDs(
		ctx,
		&repositories.GetDocumentsByResourceIDsRequest{
			TenantInfo:   tenantInfo,
			ResourceType: shipment.CommentAttachmentResourceType,
			ResourceIDs:  []string{comment.ID.String()},
		},
	)
	if err != nil {
		return err
	}

	if len(docs) == 0 {
		return nil
	}

	ids := make([]pulid.ID, 0, len(docs))
	for _, doc := range docs {
		ids = append(ids, doc.ID)
	}

	_, err = s.documentService.BulkDelete(ctx, &documentservice.BulkDeleteRequest{
		IDs:        ids,
		TenantInfo: tenantInfo,
		UserID:     actorID,
	})

	return err
}

func (s *service) loadAttachments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	comments ...*shipment.ShipmentComment,
) error {
	targets := make(map[string]*shipment.ShipmentComment, len(comments))
	resourceIDs := make([]string, 0, len(comments))
	for _, comment := range comments {
		if comment == nil {
			continue
		}
		comment.Attachments = make([]*shipment.CommentAttachment, 0, 2)
		if comment.IsDeleted() {
			continue
		}
		id := comment.ID.String()
		targets[id] = comment
		resourceIDs = append(resourceIDs, id)
	}

	if len(resourceIDs) == 0 {
		return nil
	}

	docs, err := s.documentRepo.GetByResourceIDs(
		ctx,
		&repositories.GetDocumentsByResourceIDsRequest{
			TenantInfo:   tenantInfo,
			ResourceType: shipment.CommentAttachmentResourceType,
			ResourceIDs:  resourceIDs,
		},
	)
	if err != nil {
		return err
	}

	if len(docs) == 0 {
		return nil
	}

	urls, err := s.documentService.PresignDocumentURLs(ctx, docs)
	if err != nil {
		return err
	}

	for _, doc := range docs {
		comment, ok := targets[doc.ResourceID]
		if !ok {
			continue
		}
		urlSet := urls[doc.ID]
		comment.Attachments = append(comment.Attachments, &shipment.CommentAttachment{
			DocumentID:   doc.ID,
			FileName:     doc.FileName,
			OriginalName: doc.OriginalName,
			FileSize:     doc.FileSize,
			MimeType:     doc.FileType,
			PreviewURL:   urlSet.PreviewURL,
			DownloadURL:  urlSet.DownloadURL,
			CreatedAt:    doc.CreatedAt,
		})
	}

	return nil
}
