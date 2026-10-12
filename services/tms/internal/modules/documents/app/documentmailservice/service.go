package documentmailservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultField       = "documentIds"
	defaultContentType = "application/octet-stream"
	mailedComment      = "Document emailed"
	mailedAccess       = "email"
	mailedChannel      = "attachment"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Repo        documentports.DocumentRepository
	Permissions permissionports.PermissionEngine
	Audit       auditports.AuditService
}

type Service struct {
	l           *zap.Logger
	repo        documentports.DocumentRepository
	permissions permissionports.PermissionEngine
	audit       auditports.AuditService
}

var _ documentports.DocumentMailAttachments = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.document-mail"),
		repo:        p.Repo,
		permissions: p.Permissions,
		audit:       p.Audit,
	}
}

type requested struct {
	id    pulid.ID
	index int
}

func (s *Service) Resolve(
	ctx context.Context,
	req *documentports.ResolveMailAttachmentsRequest,
) ([]*documentports.MailAttachment, error) {
	field := req.Field
	if field == "" {
		field = defaultField
	}

	wanted := distinct(req.DocumentIDs)
	if len(wanted) == 0 {
		return []*documentports.MailAttachment{}, nil
	}
	if len(wanted) > documentports.MaxMailAttachments {
		return nil, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"An email can carry at most {0} files",
			documentports.MaxMailAttachments,
		)
	}

	docs, err := s.documents(ctx, req.TenantInfo, field, wanted)
	if err != nil {
		return nil, err
	}

	if !req.System {
		if err = s.requireReadable(ctx, req.TenantInfo, docs); err != nil {
			return nil, err
		}
	}

	if err = withinBudget(field, docs); err != nil {
		return nil, err
	}

	attachments := make([]*documentports.MailAttachment, 0, len(docs))
	for _, doc := range docs {
		attachments = append(attachments, &documentports.MailAttachment{
			Document:   doc,
			Attachment: attachmentOf(doc),
		})
	}

	if len(req.Recipients) > 0 {
		s.recordMailed(ctx, req, docs)
	}

	return attachments, nil
}

func (s *Service) Mailable(
	ctx context.Context,
	req *documentports.MailableDocumentsRequest,
) ([]*document.Document, error) {
	if len(req.Sources) == 0 {
		return []*document.Document{}, nil
	}

	readable, err := s.readableResources(ctx, req.TenantInfo, sourceResources(req.Sources))
	if err != nil {
		return nil, err
	}
	if !readable[permission.ResourceDocument] {
		return []*document.Document{}, nil
	}

	seen := make(map[pulid.ID]struct{})
	out := make([]*document.Document, 0)
	for _, source := range req.Sources {
		if source.ResourceID == "" || !readable[document.OwnerResource(source.ResourceType)] {
			continue
		}

		docs, listErr := s.repo.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
			TenantInfo:          req.TenantInfo,
			ResourceID:          source.ResourceID,
			ResourceType:        source.ResourceType,
			IncludeDocumentType: true,
		})
		if listErr != nil {
			return nil, listErr
		}

		for _, doc := range docs {
			if !doc.Mailable() {
				continue
			}
			if _, dup := seen[doc.ID]; dup {
				continue
			}
			seen[doc.ID] = struct{}{}
			out = append(out, doc)
		}
	}

	return out, nil
}

func distinct(ids []pulid.ID) []requested {
	out := make([]requested, 0, len(ids))
	seen := make(map[pulid.ID]struct{}, len(ids))
	for index, id := range ids {
		if id.IsNil() {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, requested{id: id, index: index})
	}

	return out
}

func (s *Service) documents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	field string,
	wanted []requested,
) ([]*document.Document, error) {
	ids := make([]pulid.ID, 0, len(wanted))
	for _, want := range wanted {
		ids = append(ids, want.id)
	}

	found, err := s.repo.GetByIDs(ctx, documentports.BulkDeleteDocumentRequest{
		IDs:        ids,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	byID := make(map[pulid.ID]*document.Document, len(found))
	for _, doc := range found {
		byID[doc.ID] = doc
	}

	multiErr := errortypes.NewMultiError()
	docs := make([]*document.Document, 0, len(wanted))
	for _, want := range wanted {
		doc, ok := byID[want.id]
		switch {
		case !ok:
			multiErr.WithIndex(field, want.index).
				Add("", errortypes.ErrInvalid, "This file no longer exists")
		case !doc.Mailable():
			refuseUnmailable(multiErr.WithIndex(field, want.index), doc)
		default:
			docs = append(docs, doc)
		}
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return docs, nil
}

func refuseUnmailable(multiErr *errortypes.MultiError, doc *document.Document) {
	name := doc.MailFileName()
	switch doc.Status {
	case document.StatusArchived:
		multiErr.Add("", errortypes.ErrInvalid, "{0} is archived and cannot be sent", name)
	case document.StatusExpired:
		multiErr.Add("", errortypes.ErrInvalid, "{0} has expired and cannot be sent", name)
	case document.StatusRejected:
		multiErr.Add("", errortypes.ErrInvalid, "{0} was rejected and cannot be sent", name)
	case document.StatusDraft, document.StatusActive, document.StatusPending,
		document.StatusPendingApproval:
		multiErr.Add("", errortypes.ErrInvalid, "{0} cannot be sent from here", name)
	default:
		multiErr.Add("", errortypes.ErrInvalid, "{0} cannot be sent from here", name)
	}
}

func (s *Service) requireReadable(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	docs []*document.Document,
) error {
	if tenantInfo.UserID.IsNil() {
		return errortypes.NewAuthorizationError("Sign in to attach files to an email")
	}

	resources := make([]permission.Resource, 0, len(docs))
	for _, doc := range docs {
		resources = append(resources, doc.OwnerResource())
	}

	readable, err := s.readableResources(ctx, tenantInfo, resources)
	if err != nil {
		return err
	}
	if !readable[permission.ResourceDocument] {
		return errortypes.NewAuthorizationError("You do not have permission to read documents")
	}
	for _, doc := range docs {
		if !readable[doc.OwnerResource()] {
			return errortypes.NewAuthorizationError(
				"You do not have permission to send {0}",
				doc.MailFileName(),
			)
		}
	}

	return nil
}

func sourceResources(sources []documentports.MailDocumentSource) []permission.Resource {
	resources := make([]permission.Resource, 0, len(sources))
	for _, source := range sources {
		resources = append(resources, document.OwnerResource(source.ResourceType))
	}

	return resources
}

func (s *Service) readableResources(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	owners []permission.Resource,
) (map[permission.Resource]bool, error) {
	resources := make([]permission.Resource, 0, len(owners)+1)
	resources = append(resources, permission.ResourceDocument)
	seen := make(map[permission.Resource]struct{}, len(owners)+1)
	seen[permission.ResourceDocument] = struct{}{}
	for _, owner := range owners {
		if _, dup := seen[owner]; dup {
			continue
		}
		seen[owner] = struct{}{}
		resources = append(resources, owner)
	}

	actor := permissionports.UserActor(tenantInfo)
	checks := make([]permissionports.ResourceOperationCheck, 0, len(resources))
	for _, resource := range resources {
		checks = append(checks, permissionports.ResourceOperationCheck{
			Resource:  resource.String(),
			Operation: permission.OpRead,
		})
	}

	result, err := s.permissions.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Checks:         checks,
	})
	if err != nil {
		return nil, err
	}

	readable := make(map[permission.Resource]bool, len(resources))
	for i, resource := range resources {
		readable[resource] = i < len(result.Results) && result.Results[i].Allowed
	}

	return readable, nil
}

func withinBudget(field string, docs []*document.Document) error {
	var total int64
	for _, doc := range docs {
		total += doc.FileSize
	}
	if total <= emailports.MaxAttachmentBytes {
		return nil
	}

	return errortypes.NewValidationError(
		field,
		errortypes.ErrInvalid,
		"These files add up to {0}; an email can carry at most {1}",
		fileutils.HumanizeBytes(total),
		fileutils.HumanizeBytes(emailports.MaxAttachmentBytes),
	)
}

func attachmentOf(doc *document.Document) emailports.EmailAttachment {
	contentType := strings.TrimSpace(doc.FileType)
	if contentType == "" {
		contentType = defaultContentType
	}

	return emailports.EmailAttachment{
		FileName:    doc.MailFileName(),
		ContentType: contentType,
		ObjectKey:   doc.StoragePath,
		SizeBytes:   doc.FileSize,
		Seal:        email.AttachmentSealDocument,
	}
}

func (s *Service) recordMailed(
	ctx context.Context,
	req *documentports.ResolveMailAttachmentsRequest,
	docs []*document.Document,
) {
	actor := permissionports.SystemAuditActor()
	if req.TenantInfo.UserID.IsNotNil() {
		actor = permissionports.UserActor(req.TenantInfo).AuditActor()
	}

	for _, doc := range docs {
		if err := s.audit.LogAction(&auditports.LogActionParams{
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
			auditservice.WithComment(mailedComment),
			auditservice.WithCategory(audit.CategoryUser),
			auditservice.WithRequest(ctx),
			auditservice.WithMetadata(map[string]any{
				"access":       mailedAccess,
				"channel":      mailedChannel,
				"fileName":     doc.MailFileName(),
				"recipients":   req.Recipients,
				"resourceType": doc.ResourceType,
				"resourceId":   doc.ResourceID,
			}),
		); err != nil {
			s.l.Error("failed to audit an emailed document",
				zap.String("documentId", doc.ID.String()),
				zap.Error(err),
			)
		}
	}
}
