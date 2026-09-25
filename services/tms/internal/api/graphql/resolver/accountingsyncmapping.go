package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func errAccountingNeedsAPerson() error {
	return errortypes.NewAuthorizationError(
		"Connecting or disconnecting an accounting system needs a signed-in person, not an API key",
	)
}

type accountingMappingActionFunc func(
	context.Context,
	*services.AccountingMappingActionRequest,
) (*accountingsync.AccountingMapping, error)

func (r *mutationResolver) accountingMappingAction(
	ctx context.Context,
	id string,
	action accountingMappingActionFunc,
) (*accountingsync.AccountingMapping, error) {
	authCtx, err := r.requirePermission(
		ctx,
		permission.ResourceAccountingIntegration,
		permission.OpUpdate,
	)
	if err != nil {
		return nil, err
	}
	mappingID, err := pulid.MustParse(id)
	if err != nil {
		return nil, err
	}

	return action(ctx, &services.AccountingMappingActionRequest{
		TenantInfo: tenantInfo(authCtx),
		UserID:     authCtx.UserID,
		ID:         mappingID,
		Source:     accountingsync.MappingSourceManual,
	})
}

func accountingMappingConnectionToModel(
	result *pagination.CursorListResult[*accountingsync.AccountingMapping],
) (*gqlmodel.AccountingMappingConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *accountingsync.AccountingMapping, cursor string) *gqlmodel.AccountingMappingEdge {
			return &gqlmodel.AccountingMappingEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AccountingMappingEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AccountingMappingConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
