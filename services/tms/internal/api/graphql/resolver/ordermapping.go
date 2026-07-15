package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func nullDecimalToString(d decimal.NullDecimal) *string {
	if !d.Valid {
		return nil
	}
	s := d.Decimal.String()
	return &s
}

// applyOrderInput copies the GraphQL input onto an order entity. Tenant identity and
// the ID/version are set by the caller so the same helper serves create and update.
func applyOrderInput(entity *order.Order, input gqlmodel.OrderInput) error {
	customerID, err := pulid.MustParse(input.CustomerID)
	if err != nil {
		return err
	}
	entity.CustomerID = customerID

	ownerID, err := optionalID(input.OwnerID)
	if err != nil {
		return err
	}
	entity.OwnerID = ownerID

	if input.Status != nil {
		entity.Status = *input.Status
	} else if entity.Status == "" {
		entity.Status = order.StatusDraft
	}

	entity.PONumber = stringValue(input.PoNumber)
	entity.BOL = stringValue(input.Bol)

	entity.CurrencyCode = stringValue(input.CurrencyCode)
	if entity.CurrencyCode == "" {
		entity.CurrencyCode = "USD"
	}

	quoted, err := nullDecimalFromInput(input.QuotedAmount)
	if err != nil {
		return err
	}
	entity.QuotedAmount = quoted

	base, err := nullDecimalFromInput(input.BaseAmount)
	if err != nil {
		return err
	}
	entity.BaseAmount = base

	return nil
}

func orderColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.OrderSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func orderConnectionToModel(
	result *pagination.CursorListResult[*order.Order],
) (*gqlmodel.OrderConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *order.Order, cursor string) *gqlmodel.OrderEdge {
			return &gqlmodel.OrderEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.OrderEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.OrderConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
