package manualjournalresolver

import (
	"context"
	"slices"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal/manualjournaldb"
	"github.com/emoss08/trenova/pkg/pagination"
)

func manualJournalColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.ManualJournalSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)
	if len(selection.Columns) == 0 || !selection.HasSpecial(balanceSpecial) {
		return selection.Columns
	}

	columns := selection.Columns
	r := manualjournaldb.Requests
	for _, column := range projection.ColumnNames(r.TotalDebit.Col, r.TotalCredit.Col) {
		if !slices.Contains(columns, column) {
			columns = append(columns, column)
		}
	}

	return columns
}

const balanceSpecial = "balance"

func manualJournalConnectionToModel(
	result *pagination.CursorListResult[*manualjournal.Request],
) (*gqlmodel.ManualJournalConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *manualjournal.Request, cursor string) *gqlmodel.ManualJournalEdge {
			return &gqlmodel.ManualJournalEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.ManualJournalEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ManualJournalConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}
