package documentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var errDocumentNotReadable = errortypes.NewNotFoundError("Document not found")

func (s *Service) RequireReadable(
	ctx context.Context,
	req documentports.GetDocumentByIDRequest,
	actor *permissionports.RequestActor,
) error {
	doc, err := s.Get(ctx, req)
	if err != nil {
		return err
	}

	return s.requireReadable(ctx, []*document.Document{doc}, actor)
}

func (s *Service) RequireAllReadable(
	ctx context.Context,
	ids []pulid.ID,
	tenantInfo pagination.TenantInfo,
	actor *permissionports.RequestActor,
) error {
	docs, err := s.repo.GetByIDs(ctx, documentports.GetDocumentsByIDsRequest{
		IDs:        ids,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	return s.requireReadable(ctx, docs, actor)
}

func (s *Service) RequireResourceReadable(
	ctx context.Context,
	resourceType string,
	actor *permissionports.RequestActor,
) error {
	owner := document.OwnerResource(resourceType)
	if !permissioncatalog.Default().HasResource(owner.String()) {
		return errDocumentNotReadable
	}

	return s.requireOwnerRead(ctx, owner, actor)
}

func (s *Service) requireReadable(
	ctx context.Context,
	docs []*document.Document,
	actor *permissionports.RequestActor,
) error {
	person := actor.PersonUserID()
	checked := make(map[permission.Resource]struct{}, 1)
	for _, doc := range docs {
		owner, ok := doc.ReadableOwner(person, permissioncatalog.Default())
		if !ok {
			return errDocumentNotReadable
		}
		if _, seen := checked[owner]; seen {
			continue
		}
		checked[owner] = struct{}{}
		if err := s.requireOwnerRead(ctx, owner, actor); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) requireOwnerRead(
	ctx context.Context,
	owner permission.Resource,
	actor *permissionports.RequestActor,
) error {
	result, err := s.permissions.Check(ctx, actor.PermissionCheck(owner, permission.OpRead))
	if err != nil {
		return err
	}
	if result == nil || !result.Allowed {
		return errDocumentNotReadable
	}

	return nil
}

func (s *Service) visibleDocument(
	ctx context.Context,
	req documentports.GetDocumentByIDRequest,
	actor *permissionports.RequestActor,
) (*document.Document, error) {
	doc, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	if !doc.VisibleToPerson(actor.PersonUserID()) {
		return nil, errDocumentNotReadable
	}

	return doc, nil
}

func (s *Service) SearchByResource(
	ctx context.Context,
	req *documentports.GetDocumentsByResourceRequest,
	query string,
) ([]*document.Document, error) {
	docs, err := s.documentIntelligence.SearchDocuments(
		ctx,
		req.TenantInfo,
		req.ResourceType,
		req.ResourceID,
		query,
	)
	if err != nil {
		return nil, err
	}

	return document.VisibleDocuments(docs, req.Viewer), nil
}
