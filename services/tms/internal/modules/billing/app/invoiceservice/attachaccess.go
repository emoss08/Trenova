package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func (s *Service) requireAttachable(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	actor *permissionports.RequestActor,
	documentIDs []pulid.ID,
) error {
	documentIDs = sliceutils.Dedupe(documentIDs)
	if len(documentIDs) == 0 {
		return nil
	}

	readable := make(map[permission.Resource]bool, len(documentIDs)+1)
	mayRead := func(resource permission.Resource) (bool, error) {
		if allowed, seen := readable[resource]; seen {
			return allowed, nil
		}
		result, err := s.permissions.Check(ctx, actor.PermissionCheck(resource, permission.OpRead))
		if err != nil {
			return false, err
		}
		readable[resource] = result != nil && result.Allowed

		return readable[resource], nil
	}

	allowed, err := mayRead(permission.ResourceDocument)
	if err != nil {
		return err
	}
	if !allowed {
		return errortypes.NewAuthorizationError("You do not have permission to read documents")
	}
	for _, id := range documentIDs {
		doc, docErr := s.documentForID(ctx, id, tenantInfo)
		if docErr != nil {
			return docErr
		}
		allowed, err = mayRead(doc.OwnerResource())
		if err != nil {
			return err
		}
		if !allowed {
			return errortypes.NewAuthorizationError(
				"You do not have permission to send {0}",
				doc.MailFileName(),
			)
		}
	}

	return nil
}
