package resolver

import (
	"context"
	"slices"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const correctionAccuracySpecial = "accuracy"

func projectedSelection(
	ctx context.Context,
	spec projection.TypeSpec,
	nodePathPrefix string,
) projection.Selection {
	return projection.Select(
		spec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)
}

func aiCorrectionColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projectedSelection(ctx, projection.AICorrectionSpec, nodePathPrefix)
	if !selection.HasSpecial(correctionAccuracySpecial) || len(selection.Columns) == 0 {
		return selection.Columns
	}

	cols := buncolgen.CorrectionColumns
	columns := sliceutils.AppendIfMissing(selection.Columns, cols.CorrectCount.Bare())

	return sliceutils.AppendIfMissing(columns, cols.ScoredCount.Bare())
}

func extractionEvalCaseColumns(ctx context.Context, nodePathPrefix string) []string {
	return projectedSelection(ctx, projection.ExtractionEvalCaseSpec, nodePathPrefix).Columns
}

func extractionEvalRunColumns(ctx context.Context, nodePathPrefix string) []string {
	return projectedSelection(ctx, projection.ExtractionEvalRunSpec, nodePathPrefix).Columns
}

func extractionEvalResultColumns(ctx context.Context, nodePathPrefix string) []string {
	return projectedSelection(ctx, projection.ExtractionEvalResultSpec, nodePathPrefix).Columns
}

func aiCorrectionConnectionToModel(
	result *pagination.CursorListResult[*aicorrection.Correction],
) (*gqlmodel.AICorrectionConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *aicorrection.Correction, cursor string) *gqlmodel.AICorrectionEdge {
			return &gqlmodel.AICorrectionEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AICorrectionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AICorrectionConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func extractionEvalCaseConnectionToModel(
	result *pagination.CursorListResult[*extractioneval.ExtractionCase],
) (*gqlmodel.ExtractionEvalCaseConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *extractioneval.ExtractionCase, cursor string) *gqlmodel.ExtractionEvalCaseEdge {
			return &gqlmodel.ExtractionEvalCaseEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.ExtractionEvalCaseEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ExtractionEvalCaseConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func extractionEvalRunConnectionToModel(
	result *pagination.CursorListResult[*extractioneval.ExtractionRun],
) (*gqlmodel.ExtractionEvalRunConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *extractioneval.ExtractionRun, cursor string) *gqlmodel.ExtractionEvalRunEdge {
			return &gqlmodel.ExtractionEvalRunEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.ExtractionEvalRunEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ExtractionEvalRunConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func extractionEvalResultConnectionToModel(
	result *pagination.CursorListResult[*extractioneval.ExtractionResult],
) (*gqlmodel.ExtractionEvalResultConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *extractioneval.ExtractionResult, cursor string) *gqlmodel.ExtractionEvalResultEdge {
			return &gqlmodel.ExtractionEvalResultEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.ExtractionEvalResultEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ExtractionEvalResultConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func snapshotFields(snapshot *aicorrection.Snapshot) []*gqlmodel.ExtractionSnapshotField {
	if snapshot == nil || len(snapshot.Fields) == 0 {
		return []*gqlmodel.ExtractionSnapshotField{}
	}

	keys := make([]string, 0, len(snapshot.Fields))
	for key := range snapshot.Fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	fields := make([]*gqlmodel.ExtractionSnapshotField, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, &gqlmodel.ExtractionSnapshotField{Key: key, Value: snapshot.Fields[key]})
	}

	return fields
}
