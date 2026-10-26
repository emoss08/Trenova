package assistantservice

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
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
	argAfter,
	errortypes.ErrInvalid,
	"This page can no longer be read. Read the list from its start",
)

type listReader struct {
	tool      agentports.AgentQueryTool
	params    agentports.QueryToolParams
	arguments map[string]any
	entity    string
}

func (s *Service) listOf(
	ctx context.Context,
	req *agentports.GetThreadRequest,
	actor *permissionports.RequestActor,
	source *exportSource,
) (*listReader, error) {
	toolName := typeutils.StringOfTrimmed(source.artifact.Payload[payloadTool])
	if s.queries == nil || toolName == "" {
		return nil, errNotExportable
	}
	tool, ok := s.queries.Get(toolName)
	if !ok {
		return nil, errortypes.NewBusinessError(
			"The list this table came from is no longer available",
		)
	}
	if err := s.mayRead(ctx, actor, tool.Policy()); err != nil {
		return nil, err
	}

	arguments := maps.Clone(source.arguments)
	delete(arguments, argAfter)
	delete(arguments, argOffset)
	delete(arguments, argLimit)

	return &listReader{
		tool: tool,
		params: agentports.QueryToolParams{
			OrganizationID:    req.TenantInfo.OrgID,
			BusinessUnitID:    req.TenantInfo.BuID,
			Actor:             actor,
			Timezone:          s.timezoneOf(ctx, *req),
			DataAccessCeiling: s.ceilingOf(ctx, *req),
		},
		arguments: arguments,
		entity:    typeutils.StringOfTrimmed(source.artifact.Payload[payloadEntity]),
	}, nil
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
	document, ok := toJSONDocument(data)
	if !ok {
		return nil, errors.New("the list returned something that is not a table")
	}

	return document.fields, nil
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
	fields := stringsOf(property[toolschema.KeyEnum])
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

//nolint:gocritic // hugeParam: GetThreadRequest is passed by value across AssistantService.
func (s *Service) ArtifactRows(
	ctx context.Context,
	req agentports.GetThreadRequest,
	artifactID pulid.ID,
	query *agentports.ArtifactRowsQuery,
) (*agentports.ArtifactRows, error) {
	source, err := s.exportSourceOf(ctx, req, artifactID)
	if err != nil {
		return nil, err
	}
	if source.artifact.Kind != assistantartifact.KindTableView {
		return nil, errNotPageable
	}
	actor := permissionports.UserActor(req.TenantInfo)
	list, err := s.listOf(ctx, &req, actor, source)
	if err != nil {
		return nil, err
	}

	columns := columnsFrom(source.artifact.Payload[payloadColumns])
	sortable := list.sortFields(columns)
	searchable := list.searchable()
	overrides, err := rowsArguments(query, sortable, searchable)
	if err != nil {
		return nil, err
	}

	page, err := list.read(ctx, overrides, query.IncludeTotal)
	if errors.Is(err, agentquerytoolservice.ErrStaleCursor) {
		return nil, errStalePage
	}
	if err != nil {
		return nil, err
	}

	return &agentports.ArtifactRows{
		Rows: projectRows(
			columns,
			typeutils.StringOfTrimmed(source.artifact.Payload["recordEntity"]),
			recordsOf(rowsFrom(page[resultItems])),
		),
		HasMore:        typeutils.BoolOf(page[resultHasMore]),
		NextCursor:     typeutils.StringOfTrimmed(page[resultNextCursor]),
		NextOffset:     optionalInt(page[resultNextOffset]),
		Total:          optionalInt(page[resultTotal]),
		TotalEstimated: typeutils.BoolOf(page[resultTotalEstimated]),
		Withheld:       withheldColumns(columns, stringsOf(page["withheldByAccess"])),
		Searchable:     searchable,
		Sortable:       slices.Sorted(maps.Keys(sortable)),
	}, nil
}

func rowsArguments(
	query *agentports.ArtifactRowsQuery,
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
		multiErr.Add(argOffset, errortypes.ErrInvalid, "Page by a cursor or an offset, not both")
	case after != "":
		overrides[argAfter] = after
	case query.Offset < 0:
		multiErr.Add(argOffset, errortypes.ErrInvalid, "An offset cannot be negative")
	case query.Offset > 0:
		overrides[argOffset] = query.Offset
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
