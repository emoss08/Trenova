package documentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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

	actor := permissionports.SystemAuditActor()
	if tenantInfo.UserID.IsNotNil() {
		actor = permissionports.UserActor(tenantInfo).AuditActor()
	}

	err := s.auditService.LogAction(&auditports.LogActionParams{
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
