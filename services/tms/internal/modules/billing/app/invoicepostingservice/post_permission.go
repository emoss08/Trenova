package invoicepostingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Service) guardPoster(
	ctx context.Context,
	req *invoiceports.PostInvoiceRequest,
	actor *permissionports.RequestActor,
) error {
	if req.Automatic || s.permissions == nil {
		return nil
	}
	if actor == nil || actor.PrincipalType == permissionports.PrincipalTypeSystem {
		return nil
	}

	result, err := s.permissions.Check(
		ctx,
		actor.PermissionCheck(permission.ResourceInvoice, permission.OpPost),
	)
	if err != nil {
		return err
	}
	if result != nil && result.Allowed {
		return nil
	}

	return errortypes.NewAuthorizationError(
		"Your role cannot post invoices. Ask someone who can post invoices to post this one.",
	)
}
