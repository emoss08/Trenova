package documentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type documentAccess string

const (
	documentAccessDownload documentAccess = "download"
	documentAccessView     documentAccess = "view"
)

type documentChannel string

const (
	documentChannelContent      documentChannel = "content"
	documentChannelPresignedURL documentChannel = "presigned_url"
)

var documentAccessComments = map[documentAccess]string{
	documentAccessDownload: "Document downloaded",
	documentAccessView:     "Document viewed",
}

func (s *Service) recordAccess(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	doc *document.Document,
	access documentAccess,
	channel documentChannel,
) {
	if s.auditService == nil {
		return
	}

	actor := services.SystemAuditActor()
	if tenantInfo.UserID.IsNotNil() {
		actor = services.UserActor(tenantInfo).AuditActor()
	}

	err := s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     doc.ID.String(),
		Operation:      permission.OpRead,
		UserID:         actor.UserID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		APIKeyID:       actor.APIKeyID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
	},
		auditservice.WithComment(documentAccessComments[access]),
		auditservice.WithCategory(audit.CategoryUser),
		auditservice.WithRequest(ctx),
		auditservice.WithMetadata(map[string]any{
			"access":             string(access),
			"channel":            string(channel),
			"fileName":           doc.OriginalName,
			metadataResourceType: doc.ResourceType,
			metadataResourceID:   doc.ResourceID,
		}),
	)
	if err != nil {
		s.l.Error("failed to audit document access",
			zap.Error(err),
			zap.String("documentId", doc.ID.String()),
			zap.String("access", string(access)),
		)
	}
}
