package formulatemplateresolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func (d *Deps) requireAnyTemplatePermission(
	ctx context.Context,
	operations ...permission.Operation,
) (*authctx.AuthContext, error) {
	var err error
	for _, operation := range operations {
		var authCtx *authctx.AuthContext
		authCtx, err = d.RequirePermission(ctx, permission.ResourceFormulaTemplate, operation)
		if err == nil {
			return authCtx, nil
		}
		if !errortypes.IsAuthorizationError(err) {
			return nil, err
		}
	}
	return nil, err
}

func (d *Deps) requireAuthoring(ctx context.Context) (*authctx.AuthContext, error) {
	return d.requireAnyTemplatePermission(ctx, permission.OpCreate, permission.OpUpdate)
}

func (d *Deps) templateAllows(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	operation permission.Operation,
) (bool, error) {
	return resolvercore.CheckPermission(
		ctx,
		d.PermissionEngine,
		authCtx,
		permission.ResourceFormulaTemplate,
		operation,
	)
}

func (d *Deps) requireShipmentRead(ctx context.Context, authCtx *authctx.AuthContext) error {
	allowed, err := resolvercore.CheckPermission(
		ctx,
		d.PermissionEngine,
		authCtx,
		permission.ResourceShipment,
		permission.OpRead,
	)
	if err != nil {
		return err
	}
	if !allowed {
		return errortypes.NewAuthorizationError("You don't have permission to read shipments")
	}
	return nil
}

func (d *Deps) optionalShipment(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	raw *string,
) (*pulid.ID, error) {
	value := strings.TrimSpace(gqlscalar.StringValue(raw))
	if value == "" {
		return nil, nil //nolint:nilnil // no shipment rates the sample record
	}
	shipmentID, err := pulid.MustParse(value)
	if err != nil {
		return nil, err
	}
	if err = d.requireShipmentRead(ctx, authCtx); err != nil {
		return nil, err
	}
	return &shipmentID, nil
}

func (d *Deps) requireStudioAssist() error {
	if d.FormulaStudioAssist != nil {
		return nil
	}
	return errortypes.NewBusinessError("The formula assistant is not available on this server")
}

func (d *Deps) requireTemplateAssistant(ctx context.Context) (*authctx.AuthContext, error) {
	authCtx, err := d.RequirePermission(ctx, permission.ResourceFormulaTemplate, permission.OpRead)
	if err != nil {
		return nil, err
	}
	if _, err = d.RequirePermission(ctx, permission.ResourceAssistant, permission.OpCreate); err != nil {
		return nil, err
	}
	if err = d.requireStudioAssist(); err != nil {
		return nil, err
	}
	return authCtx, nil
}
