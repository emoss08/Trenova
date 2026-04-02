package documentservice

import (
	"context"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func documentIDs(docs []*document.Document) []pulid.ID {
	ids := make([]pulid.ID, 0, len(docs))
	for _, doc := range docs {
		ids = append(ids, doc.ID)
	}
	return ids
}

func currentDocumentVersion(docs []*document.Document) *document.Document {
	for _, doc := range docs {
		if doc.IsCurrentVersion {
			return doc
		}
	}
	if len(docs) == 0 {
		return nil
	}
	return docs[0]
}

func (s *Service) resolveLineageForUpload(
	ctx context.Context,
	lineageID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*document.Document, error) {
	versions, err := s.repo.ListVersions(ctx, repositories.ListDocumentVersionsRequest{
		LineageID:  lineageID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, errortypes.NewNotFoundError(
			"Document lineage not found within your organization",
		)
	}

	current := currentDocumentVersion(versions)
	if current == nil {
		return nil, errortypes.NewConflictError("Document lineage does not have a current version")
	}

	return current, nil
}

func (s *Service) nextVersionNumber(lineageInfo *document.Document) int64 {
	if lineageInfo == nil {
		return 1
	}
	return lineageInfo.VersionNumber + 1
}

func (s *Service) makeCurrentDocumentVersion(
	ctx context.Context,
	doc *document.Document,
	lineageInfo *document.Document,
) error {
	return s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if lineageInfo != nil {
			doc.IsCurrentVersion = false
		}
		if _, err := s.repo.Create(txCtx, doc); err != nil {
			return err
		}

		if lineageInfo != nil {
			if err := s.repo.PromoteVersion(txCtx, &repositories.PromoteDocumentVersionRequest{
				LineageID:         doc.LineageID,
				CurrentDocumentID: doc.ID,
				TenantInfo: pagination.TenantInfo{
					OrgID: doc.OrganizationID,
					BuID:  doc.BusinessUnitID,
				},
			}); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Service) deleteSearchProjection(
	ctx context.Context,
	doc *document.Document,
) {
	if doc == nil {
		return
	}

	_ = s.searchProjection.Delete(ctx, doc.ID, pagination.TenantInfo{
		OrgID: doc.OrganizationID,
		BuID:  doc.BusinessUnitID,
	})
}

func (s *Service) ListVersions(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]*document.Document, error) {
	doc, err := s.repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return s.repo.ListVersions(ctx, repositories.ListDocumentVersionsRequest{
		LineageID:  doc.LineageID,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) RestoreVersion(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*document.Document, error) {
	target, err := s.repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	versions, err := s.repo.ListVersions(ctx, repositories.ListDocumentVersionsRequest{
		LineageID:  target.LineageID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	current := currentDocumentVersion(versions)
	if current != nil && current.ID == target.ID {
		return target, nil
	}

	if err := s.repo.PromoteVersion(ctx, &repositories.PromoteDocumentVersionRequest{
		LineageID:         target.LineageID,
		CurrentDocumentID: target.ID,
		TenantInfo:        tenantInfo,
	}); err != nil {
		return nil, err
	}

	if current != nil {
		s.deleteSearchProjection(ctx, current)
	}
	contentText := ""
	if content, contentErr := s.documentIntelligence.GetContent(ctx, target.ID, tenantInfo); contentErr == nil &&
		content != nil {
		contentText = content.ContentText
	}
	s.syncSearchProjection(ctx, s.l, target, contentText)

	if err := s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     target.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(current),
		CurrentState:   jsonutils.MustToJSON(target),
		OrganizationID: target.OrganizationID,
		BusinessUnitID: target.BusinessUnitID,
	}, auditservice.WithComment("Document version restored as current")); err != nil {
		s.l.Warn("failed to log document version restore", zap.Error(err))
	}

	return s.repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         target.ID,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) GetPacketSummary(
	ctx context.Context,
	resourceType, resourceID string,
	tenantInfo pagination.TenantInfo,
) (*documentpacketrule.PacketSummary, error) {
	rules, err := s.packetRuleRepo.ListByResourceType(
		ctx,
		&repositories.ListDocumentPacketRulesByResourceRequest{
			TenantInfo:   tenantInfo,
			ResourceType: resourceType,
		},
	)
	if err != nil {
		return nil, err
	}

	docs, err := s.repo.GetByResourceID(ctx, &repositories.GetDocumentsByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceID:   resourceID,
		ResourceType: resourceType,
	})
	if err != nil {
		return nil, err
	}

	typeByID := make(map[string]*documenttype.DocumentType, len(rules))
	for _, rule := range rules {
		docType, typeErr := s.documentTypeRepo.GetByID(ctx, repositories.GetDocumentTypeByIDRequest{
			ID:         rule.DocumentTypeID,
			TenantInfo: tenantInfo,
		})
		if typeErr != nil {
			return nil, typeErr
		}
		typeByID[rule.DocumentTypeID.String()] = docType
	}

	items := make([]documentpacketrule.PacketItemSummary, 0, len(rules))
	summary := &documentpacketrule.PacketSummary{
		ResourceID:   resourceID,
		ResourceType: documentpacketrule.ResourceType(resourceType),
		Status:       documentpacketrule.PacketStatusComplete,
		TotalRules:   len(rules),
	}

	for _, rule := range rules {
		matchedDocs := make([]*document.Document, 0)
		for _, doc := range docs {
			if doc.DocumentTypeID != nil && *doc.DocumentTypeID == rule.DocumentTypeID {
				matchedDocs = append(matchedDocs, doc)
			}
		}

		item := documentpacketrule.PacketItemSummary{
			DocumentTypeID:        rule.DocumentTypeID,
			Required:              rule.Required,
			AllowMultiple:         rule.AllowMultiple,
			DisplayOrder:          rule.DisplayOrder,
			ExpirationRequired:    rule.ExpirationRequired,
			ExpirationWarningDays: rule.ExpirationWarningDays,
			DocumentCount:         len(matchedDocs),
			CurrentDocumentIDs:    documentIDs(matchedDocs),
			Status:                documentpacketrule.ItemStatusComplete,
		}
		if docType, ok := typeByID[rule.DocumentTypeID.String()]; ok {
			item.DocumentTypeCode = docType.Code
			item.DocumentTypeName = docType.Name
		}

		switch {
		case len(matchedDocs) == 0:
			item.Status = documentpacketrule.ItemStatusMissing
			if rule.Required {
				summary.MissingRequired++
			}
		case rule.ExpirationRequired && slices.ContainsFunc(matchedDocs, func(doc *document.Document) bool {
			return doc.ExpirationDate == nil
		}):
			item.Status = documentpacketrule.ItemStatusNeedsReview
			summary.NeedsReview++
		default:
			now := time.Now().Unix()
			expired := slices.ContainsFunc(matchedDocs, func(doc *document.Document) bool {
				return doc.ExpirationDate != nil && *doc.ExpirationDate <= now
			})
			expiringSoon := slices.ContainsFunc(matchedDocs, func(doc *document.Document) bool {
				if doc.ExpirationDate == nil {
					return false
				}
				windowEnd := now + int64(rule.ExpirationWarningDays*24*60*60)
				return *doc.ExpirationDate > now && *doc.ExpirationDate <= windowEnd
			})
			needsReview := slices.ContainsFunc(matchedDocs, func(doc *document.Document) bool {
				return doc.Status != document.StatusActive ||
					doc.PreviewStatus == document.PreviewStatusFailed ||
					doc.ContentStatus == document.ContentStatusFailed
			})

			switch {
			case needsReview:
				item.Status = documentpacketrule.ItemStatusNeedsReview
				summary.NeedsReview++
			case expired:
				item.Status = documentpacketrule.ItemStatusExpired
				summary.Expired++
			case expiringSoon:
				item.Status = documentpacketrule.ItemStatusExpiringSoon
				summary.ExpiringSoon++
			}
		}

		if item.Status == documentpacketrule.ItemStatusComplete {
			summary.SatisfiedRules++
		}

		items = append(items, item)
	}

	slices.SortStableFunc(items, func(a, b documentpacketrule.PacketItemSummary) int {
		if a.DisplayOrder == b.DisplayOrder {
			return cmpStrings(a.DocumentTypeName, b.DocumentTypeName)
		}
		if a.DisplayOrder < b.DisplayOrder {
			return -1
		}
		return 1
	})

	summary.Items = items
	switch {
	case summary.Expired > 0:
		summary.Status = documentpacketrule.PacketStatusExpired
	case summary.NeedsReview > 0:
		summary.Status = documentpacketrule.PacketStatusNeedsReview
	case summary.MissingRequired > 0:
		summary.Status = documentpacketrule.PacketStatusIncomplete
	case summary.ExpiringSoon > 0:
		summary.Status = documentpacketrule.PacketStatusExpiringSoon
	default:
		summary.Status = documentpacketrule.PacketStatusComplete
	}

	return summary, nil
}

func (s *Service) AttachLineageToResource(
	ctx context.Context,
	documentID pulid.ID,
	resourceType string,
	resourceID string,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*document.Document, error) {
	current, err := s.repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if current.ResourceType == resourceType && current.ResourceID == resourceID {
		return current, nil
	}

	var shipmentID pulid.ID
	if resourceType == "shipment" {
		shipmentID, err = pulid.MustParse(resourceID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"shipmentId",
				errortypes.ErrInvalid,
				"Invalid shipment ID",
			)
		}
	}

	previous := *current
	if err = s.repo.MoveLineageToResource(ctx, &repositories.MoveDocumentLineageRequest{
		DocumentID:   documentID,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		TenantInfo:   tenantInfo,
	}); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if resourceType == "shipment" && s.draftRepo != nil {
		draft, draftErr := s.draftRepo.GetByDocumentID(ctx, updated.ID, tenantInfo)
		switch {
		case draftErr == nil:
			now := time.Now().Unix()
			draft.AttachedShipmentID = &shipmentID
			draft.AttachedAt = &now
			draft.AttachedByID = &userID
			if _, draftErr = s.draftRepo.Upsert(ctx, draft); draftErr != nil {
				return nil, draftErr
			}
		case errortypes.IsNotFoundError(draftErr):
		default:
			return nil, draftErr
		}
	}

	contentText := ""
	if content, contentErr := s.documentIntelligence.GetContent(ctx, updated.ID, tenantInfo); contentErr == nil && content != nil {
		contentText = content.ContentText
	}
	s.syncSearchProjection(ctx, s.l, updated, contentText)

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(previous),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("Document lineage attached to shipment")); err != nil {
		s.l.Warn("failed to log document lineage reassignment", zap.Error(err))
	}

	return updated, nil
}

func cmpStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
