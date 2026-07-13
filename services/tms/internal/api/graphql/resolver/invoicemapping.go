package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/pkg/pagination"
)

func invoiceConnectionToModel(
	result *pagination.CursorListResult[*invoice.Invoice],
) (*gqlmodel.InvoiceConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *invoice.Invoice, cursor string) *gqlmodel.InvoiceEdge {
			return &gqlmodel.InvoiceEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.InvoiceEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.InvoiceConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
