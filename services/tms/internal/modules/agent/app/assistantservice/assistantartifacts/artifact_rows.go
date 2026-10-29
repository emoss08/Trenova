package assistantartifacts

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	argLimit         = "limit"
	argQuery         = "query"
	argSortBy        = "sortBy"
	argSortDirection = "sortDirection"

	defaultRowsPage = 25
	maxRowsPage     = 100
)

var errNotPageable = errortypes.NewValidationError(
	"artifact", errortypes.ErrInvalid, "Only a table read from a list can be paged",
)

var errStalePage = errortypes.NewValidationError(
	artifactpayload.ArgAfter,
	errortypes.ErrInvalid,
	"This page can no longer be read. Read the list from its start",
)

type listReader struct {
	tool      agenttoolports.AgentQueryTool
	params    agenttoolports.QueryToolParams
	arguments map[string]any
	entity    string
}

func (l *listReader) read(
	ctx context.Context,
	overrides map[string]any,
	includeTotal bool,
) (map[string]any, error) {
	arguments := make(map[string]any, len(l.arguments)+len(overrides))
	maps.Copy(arguments, l.arguments)
	maps.Copy(arguments, overrides)

	params := l.params
	params.Params = arguments
	params.IncludeTotal = includeTotal
	data, err := l.tool.Query(ctx, &params)
	if err != nil {
		return nil, err
	}
	document, ok := artifactpayload.ToJSONDocument(data)
	if !ok {
		return nil, errors.New("the list returned something that is not a table")
	}

	return document.Fields, nil
}

func (l *listReader) properties() map[string]any {
	properties, _ := l.tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)

	return properties
}

func (l *listReader) searchable() bool {
	_, takesQuery := l.properties()[argQuery]

	return takesQuery && typeutils.StringOfTrimmed(l.arguments[argQuery]) == ""
}

func (l *listReader) sortFields(columns []assistantartifact.DisplayColumn) map[string]string {
	property, _ := l.properties()[argSortBy].(map[string]any)
	fields := artifactpayload.StringsOf(property[toolschema.KeyEnum])
	out := make(map[string]string, len(columns))
	for _, column := range columns {
		if field, ok := sortFieldFor(column.Key, fields); ok {
			out[column.Key] = field
		}
	}

	return out
}

func sortFieldFor(key string, fields []string) (string, bool) {
	if slices.Contains(fields, key) {
		return key, true
	}
	for _, field := range fields {
		if strings.HasPrefix(field, key+".") {
			return field, true
		}
	}

	return "", false
}

func rowsArguments(
	query *assistantartifactports.ArtifactRowsQuery,
	sortable map[string]string,
	searchable bool,
) (map[string]any, error) {
	multiErr := errortypes.NewMultiError()
	limit := query.Limit
	if limit <= 0 {
		limit = defaultRowsPage
	}
	overrides := map[string]any{argLimit: min(limit, maxRowsPage)}

	switch after := strings.TrimSpace(query.After); {
	case after != "" && query.Offset > 0:
		multiErr.Add(
			artifactpayload.ArgOffset,
			errortypes.ErrInvalid,
			"Page by a cursor or an offset, not both",
		)
	case after != "":
		overrides[artifactpayload.ArgAfter] = after
	case query.Offset < 0:
		multiErr.Add(
			artifactpayload.ArgOffset,
			errortypes.ErrInvalid,
			"An offset cannot be negative",
		)
	case query.Offset > 0:
		overrides[artifactpayload.ArgOffset] = query.Offset
	}

	if text := strings.TrimSpace(query.Query); text != "" {
		if !searchable {
			multiErr.Add(argQuery, errortypes.ErrInvalid, "This table cannot be searched")
		}
		overrides[argQuery] = text
	}

	if key := strings.TrimSpace(query.SortBy); key != "" {
		field, ok := sortable[key]
		if !ok {
			multiErr.Add(
				argSortBy, errortypes.ErrInvalid, "This table cannot be sorted by that column",
			)
		}
		direction := dbtype.SortDirection(strings.ToLower(strings.TrimSpace(query.SortDirection)))
		switch direction {
		case "":
			direction = dbtype.SortDirectionAsc
		case dbtype.SortDirectionAsc, dbtype.SortDirectionDesc:
		default:
			multiErr.Add(argSortDirection, errortypes.ErrInvalid, "Sort ascending or descending")
		}
		overrides[argSortBy] = field
		overrides[argSortDirection] = string(direction)
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return overrides, nil
}

func withheldColumns(columns []assistantartifact.DisplayColumn, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, column := range columns {
		if slices.ContainsFunc(paths, func(path string) bool {
			return path == column.Key || strings.HasPrefix(path, column.Key+".")
		}) {
			out = append(out, column.Key)
		}
	}

	return out
}

func optionalInt(value any) *int {
	number, ok := typeutils.IntOf(value)
	if !ok {
		return nil
	}

	return &number
}
