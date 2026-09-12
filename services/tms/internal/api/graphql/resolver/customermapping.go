package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/pkg/pagination"
)

func customerColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.CustomerSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func customerConnectionToModel(
	result *pagination.CursorListResult[*customer.Customer],
) (*gqlmodel.CustomerConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *customer.Customer, cursor string) *gqlmodel.CustomerEdge {
			return &gqlmodel.CustomerEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.CustomerEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.CustomerConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}


// legacyBillingCycleType renders the new delivery mode and cadence back into the
// single enum the old API exposed, for clients still reading the deprecated
// field. SemiMonthly has no legacy member and reports as BiWeekly, the closest
// cadence such a client can act on.
func legacyBillingCycleType(
	obj *customer.CustomerBillingProfile,
) gqlmodel.CustomerBillingCycleType {
	if obj.InvoiceDelivery != customer.InvoiceDeliveryConsolidated {
		return gqlmodel.CustomerBillingCycleTypeImmediate
	}

	switch obj.BillingCycle {
	case customer.BillingCycleDaily:
		return gqlmodel.CustomerBillingCycleTypeDaily
	case customer.BillingCycleWeekly:
		return gqlmodel.CustomerBillingCycleTypeWeekly
	case customer.BillingCycleBiWeekly, customer.BillingCycleSemiMonthly:
		return gqlmodel.CustomerBillingCycleTypeBiWeekly
	case customer.BillingCycleMonthly:
		return gqlmodel.CustomerBillingCycleTypeMonthly
	case customer.BillingCycleQuarterly:
		return gqlmodel.CustomerBillingCycleTypeQuarterly
	default:
		return gqlmodel.CustomerBillingCycleTypeImmediate
	}
}
