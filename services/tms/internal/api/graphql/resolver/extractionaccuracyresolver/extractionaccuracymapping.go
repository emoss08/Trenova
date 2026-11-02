package extractionaccuracyresolver

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection/aicorrectiondb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const correctionAccuracySpecial = "accuracy"

func aiCorrectionColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := gqlconnection.ProjectedSelection(ctx, projection.AICorrectionSpec, nodePathPrefix)
	if !selection.HasSpecial(correctionAccuracySpecial) || len(selection.Columns) == 0 {
		return selection.Columns
	}

	c := aicorrectiondb.Corrections
	columns := selection.Columns
	for _, column := range projection.ColumnNames(c.CorrectCount.Col, c.ScoredCount.Col) {
		columns = sliceutils.AppendIfMissing(columns, column)
	}

	return columns
}

func aiCorrectionConnectionToModel(
	result *pagination.CursorListResult[*aicorrection.Correction],
) (*gqlmodel.AICorrectionConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
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
		fields = append(
			fields,
			&gqlmodel.ExtractionSnapshotField{Key: key, Value: snapshot.Fields[key]},
		)
	}

	return fields
}
