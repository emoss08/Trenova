package accountingsyncresolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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
	*accountingsyncports.AccountingMappingActionRequest,
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

	return action(ctx, &accountingsyncports.AccountingMappingActionRequest{
		TenantInfo: base.TenantInfo(authCtx),
		UserID:     authCtx.UserID,
		ID:         mappingID,
		Source:     accountingsync.MappingSourceManual,
	})
}

// accountingMappingOptionsMost is how many records a mapping's options
// offer: enough to choose among, few enough to read.
const accountingMappingOptionsMost = 10

// accountingMappingDetail is one mapping and, when the person searched, the
// usable accounting system records of its kind whose names match.
func (r *QueryResolver) accountingMappingDetail(
	ctx context.Context,
	find *accountingsyncports.SetAccountingMappingRequest,
	optionsQuery string,
) (*gqlmodel.AccountingMappingDetail, error) {
	mapping, err := r.AccountingMappingService.FindMapping(ctx, find)
	if err != nil {
		return nil, err
	}
	detail := &gqlmodel.AccountingMappingDetail{
		Mapping: mapping,
		Options: []*accountingsync.AccountingReferenceObject{},
	}
	if optionsQuery == "" {
		return detail, nil
	}
	detail.Options, err = r.AccountingMappingService.SearchReference(
		ctx,
		&accountingsyncports.SearchAccountingReferenceRequest{
			TenantInfo:      find.TenantInfo,
			IntegrationType: find.IntegrationType,
			Kind:            mapping.ProviderKind,
			Query:           optionsQuery,
			UsableOnly:      true,
			Limit:           accountingMappingOptionsMost,
		},
	)
	if err != nil {
		return nil, err
	}

	return detail, nil
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
) ([]accountingsyncports.AccountingMappingConfirmation, error) {
	items := make([]accountingsyncports.AccountingMappingConfirmation, 0, len(input))
	for _, item := range input {
		if item == nil {
			continue
		}
		id, err := pulid.MustParse(item.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, accountingsyncports.AccountingMappingConfirmation{
			ID:         id,
			ExternalID: strings.TrimSpace(item.ExternalID),
		})
	}
	return items, nil
}
