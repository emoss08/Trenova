package accountingsyncresolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

func errAccountingCompanyChoiceUnsupported() error {
	return errortypes.NewBusinessError(
		"The sign-in authorized several companies. Reload the page and connect again to choose one.",
	)
}

func (r *MutationResolver) finishAccountingAuthorization(
	ctx context.Context,
	input gqlmodel.CompleteAccountingAuthorizationInput,
) (*services.AccountingAuthorizationCompletion, error) {
	authCtx, err := r.RequirePermission(
		ctx,
		permission.ResourceAccountingIntegration,
		permission.OpManage,
	)
	if err != nil {
		return nil, err
	}
	if authCtx.UserID.IsNil() {
		return nil, errAccountingNeedsAPerson()
	}

	return r.AccountingConnections.CompleteAuthorization(
		ctx,
		&services.CompleteAccountingAuthorizationRequest{
			TenantInfo:      base.TenantInfo(authCtx),
			UserID:          authCtx.UserID,
			IntegrationType: input.IntegrationType,
			State:           input.State,
			Code:            input.Code,
			RealmID:         typeutils.ValueOrZero(input.RealmID),
		},
	)
}

func errAccountingNeedsAPerson() error {
	return errortypes.NewAuthorizationError(
		"Connecting or disconnecting an accounting system needs a signed-in person, not an API key",
	)
}

type accountingMappingActionFunc func(
	context.Context,
	*services.AccountingMappingActionRequest,
) (*accountingsync.AccountingMapping, error)

func (r *MutationResolver) accountingMappingAction(
	ctx context.Context,
	id string,
	action accountingMappingActionFunc,
) (*accountingsync.AccountingMapping, error) {
	authCtx, err := r.RequirePermission(
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
		TenantInfo: base.TenantInfo(authCtx),
		UserID:     authCtx.UserID,
		ID:         mappingID,
		Source:     accountingsync.MappingSourceManual,
	})
}

func accountingMappingConnectionToModel(
	result *pagination.CursorListResult[*accountingsync.AccountingMapping],
) (*gqlmodel.AccountingMappingConnection, error) {
	page, err := base.EntityCursorConnection(
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

func accountingMappingConfirmations(
	input []*gqlmodel.ConfirmAccountingMappingInput,
) ([]services.AccountingMappingConfirmation, error) {
	items := make([]services.AccountingMappingConfirmation, 0, len(input))
	for _, item := range input {
		if item == nil {
			continue
		}
		id, err := pulid.MustParse(item.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, services.AccountingMappingConfirmation{
			ID:         id,
			ExternalID: strings.TrimSpace(item.ExternalID),
		})
	}
	return items, nil
}
