package resolver

import (
	"context"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/pkg/errortypes"
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

// optionalNullDecimal preserves the null/valued distinction: an omitted or empty
// input stays NULL instead of coercing to zero, so "no quote" and "quoted $0" remain
// different states.
func optionalNullDecimal(field string, value *string) (decimal.NullDecimal, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return decimal.NullDecimal{}, nil
	}

	parsed, err := decimal.NewFromString(strings.TrimSpace(*value))
	if err != nil {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Amount must be a valid decimal",
		)
	}

	return decimal.NewNullDecimal(parsed), nil
}

// applyOrderInput copies the GraphQL input onto an order entity. Tenant identity, the
// ID, and the status lifecycle are owned by the caller/service — status is derived
// and never client-writable.
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

	entity.PONumber = stringValue(input.PoNumber)
	entity.BOL = stringValue(input.Bol)

	entity.CurrencyCode = stringValue(input.CurrencyCode)
	if entity.CurrencyCode == "" {
		entity.CurrencyCode = "USD"
	}

	quoted, err := optionalNullDecimal("quotedAmount", input.QuotedAmount)
	if err != nil {
		return err
	}
	entity.QuotedAmount = quoted

	base, err := optionalNullDecimal("baseAmount", input.BaseAmount)
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
