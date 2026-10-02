package assistantservice

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

const (
	payloadBunched  = "bunched"
	payloadCalls    = "calls"
	payloadRecordID = "recordId"
	payloadTool     = "tool"
	payloadEntity   = "entity"
	payloadColumns  = "columns"
	payloadRows     = "rows"
	payloadFields   = "fields"
	payloadDisplay  = "display"
	payloadRowCount = "rowCount"
	maxBunchColumns = 8
)

type bunchedRecord struct {
	callID string
	id     string
	fields []assistantartifact.DisplayField
}

type bunchParts struct {
	table  *assistantartifact.Artifact
	cards  []*assistantartifact.Artifact
	tables []*assistantartifact.Artifact
}

func (r *artifactRecorder) bunch(
	observation *services.ToolObservation,
	card *assistantartifact.Artifact,
) (*services.ShownArtifact, error) {
	earlier, err := r.repo.ListByToolCalls(r.ctx, &repositories.ListArtifactsByToolCallsRequest{
		ThreadID:   r.thread.ID,
		TenantInfo: r.tenant,
		CallIDs:    observation.Earlier,
	})
	if err != nil {
		r.logger.Warn("the turn's earlier cards could not be read to bunch them",
			zap.String("thread", r.thread.ID.String()),
			zap.Error(err),
		)

		return r.keep(card)
	}

	parts := partsOf(earlier, observation.Call.Name)
	if parts.table == nil && len(parts.cards) == 0 {
		return r.keep(card)
	}

	table := bunchTable(observation.Call.Name, parts, card)
	saved, err := r.save(table)
	if err != nil {
		return nil, err
	}

	r.retire(parts.cards)

	return shownArtifact(saved), nil
}

func (r *artifactRecorder) keep(
	artifact *assistantartifact.Artifact,
) (*services.ShownArtifact, error) {
	saved, err := r.save(artifact)
	if err != nil {
		return nil, err
	}

	return shownArtifact(saved), nil
}

func (r *artifactRecorder) retire(cards []*assistantartifact.Artifact) {
	if len(cards) == 0 {
		return
	}

	ids := make([]pulid.ID, 0, len(cards))
	for _, card := range cards {
		ids = append(ids, card.ID)
	}
	if err := r.repo.Delete(r.ctx, &repositories.DeleteArtifactsRequest{
		ThreadID:   r.thread.ID,
		TenantInfo: r.tenant,
		IDs:        ids,
	}); err != nil {
		r.logger.Warn("the cards a table replaced could not be removed",
			zap.String("thread", r.thread.ID.String()),
			zap.Error(err),
		)

		return
	}

	for _, id := range ids {
		r.forget(id)
		r.emit(services.StreamEvent{
			Event: services.AssistantEventArtifactRemoved,
			Data:  services.AssistantArtifactRemovedEvent{ID: id},
		})
	}
}

func partsOf(earlier []*assistantartifact.Artifact, tool string) bunchParts {
	var parts bunchParts
	for _, artifact := range earlier {
		if artifact == nil || typeutils.StringOfTrimmed(artifact.Payload[payloadTool]) != tool {
			continue
		}
		switch {
		case artifact.Kind == assistantartifact.KindTableView &&
			typeutils.BoolOf(artifact.Payload[payloadBunched]):
			if parts.table == nil {
				parts.table = artifact
			}
		case artifact.Kind == assistantartifact.KindEntityCard:
			parts.cards = append(parts.cards, artifact)
		case artifact.Kind == assistantartifact.KindTableView:
			parts.tables = append(parts.tables, artifact)
		}
	}

	return parts
}

// bunchTables folds a list or search result into the turn's table for the
// same tool. Eleven searches for eleven shipments used to be eleven tables
// in the pane, each a row long; they are one table, with every row once,
// keyed by the turn's first call of the tool, and the tables it replaced
// are withdrawn from the reader.
func (r *artifactRecorder) bunchTables(
	observation *services.ToolObservation,
	table *assistantartifact.Artifact,
) (*services.ShownArtifact, error) {
	earlier, err := r.repo.ListByToolCalls(r.ctx, &repositories.ListArtifactsByToolCallsRequest{
		ThreadID:   r.thread.ID,
		TenantInfo: r.tenant,
		CallIDs:    observation.Earlier,
	})
	if err != nil {
		r.logger.Warn("the turn's earlier tables could not be read to bunch them",
			zap.String("thread", r.thread.ID.String()),
			zap.Error(err),
		)

		return r.keep(table)
	}

	parts := partsOf(earlier, observation.Call.Name)
	if parts.table == nil && len(parts.tables) == 0 {
		return r.keep(table)
	}

	merged := mergeTables(observation.Call.Name, parts, table)
	saved, err := r.save(merged)
	if err != nil {
		return nil, err
	}

	r.retire(parts.tables)

	return shownArtifact(saved), nil
}

// mergeTables lays the turn's tables of one tool end to end: the columns are
// the union in first-seen order, a row that names a record already in the
// table is not repeated, and what each search looked for is kept so the
// footer can still say so.
func mergeTables(
	tool string,
	parts bunchParts,
	table *assistantartifact.Artifact,
) *assistantartifact.Artifact {
	sources := make([]*assistantartifact.Artifact, 0, len(parts.tables)+2)
	if parts.table != nil {
		sources = append(sources, parts.table)
	}
	sources = append(sources, parts.tables...)
	sources = append(sources, table)

	first := sources[0]
	entity := typeutils.StringOfTrimmed(first.Payload[payloadEntity])
	recordEntity := typeutils.StringOfTrimmed(first.Payload["recordEntity"])
	var id pulid.ID
	if parts.table != nil {
		id = parts.table.ID
	}

	var columns []assistantartifact.DisplayColumn
	rows := make([]any, 0, len(sources))
	calls := make([]string, 0, len(sources))
	searched := make([]string, 0, len(sources))
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		for _, call := range append(
			stringsOf(source.Payload[payloadCalls]), source.SourceToolCallID,
		) {
			if call != "" && !slices.Contains(calls, call) {
				calls = append(calls, call)
			}
		}
		for _, term := range stringsOf(source.Payload["searchedFor"]) {
			if !slices.Contains(searched, term) {
				searched = append(searched, term)
			}
		}
		for _, column := range columnsFrom(source.Payload[payloadColumns]) {
			if !slices.ContainsFunc(columns, func(have assistantartifact.DisplayColumn) bool {
				return have.Key == column.Key
			}) {
				columns = append(columns, column)
			}
		}
		for _, row := range rowsFrom(source.Payload[payloadRows]) {
			record, _ := row.(map[string]any)
			if recordID := typeutils.StringOfTrimmed(record[recordIDKey]); recordID != "" {
				if _, dup := seen[recordID]; dup {
					continue
				}
				seen[recordID] = struct{}{}
			}
			rows = append(rows, row)
		}
	}

	payload := map[string]any{
		payloadDisplay:  assistantartifact.DisplayVersion,
		payloadTool:     tool,
		payloadEntity:   entity,
		payloadColumns:  columns,
		payloadRows:     rows,
		payloadRowCount: len(rows),
		"searchedFor":   searched,
		payloadBunched:  true,
		payloadCalls:    calls,
	}
	if recordEntity != "" {
		payload["recordEntity"] = recordEntity
	}
	fitRows(payload, payloadRows)

	return &assistantartifact.Artifact{
		ID:     id,
		Kind:   assistantartifact.KindTableView,
		Status: assistantartifact.StatusReady,
		Title: artifactTitle(fmt.Sprintf("%s (%d)",
			stringutils.CapitalizeFirst(stringutils.HumanizeSnakeCase(entity)), len(rows))),
		Payload:          payload,
		SourceToolCallID: first.SourceToolCallID,
	}
}

func bunchTable(
	tool string,
	parts bunchParts,
	card *assistantartifact.Artifact,
) *assistantartifact.Artifact {
	entity := typeutils.StringOfTrimmed(card.Payload[payloadEntity])
	recordEntity := recordEntityOf(entity)

	columns := make([]assistantartifact.DisplayColumn, 0, maxBunchColumns)
	rows := make([]any, 0, len(parts.cards)+2)
	calls := make([]string, 0, len(parts.cards)+2)
	source := card.SourceToolCallID
	var id pulid.ID
	if parts.table != nil {
		columns = append(columns, columnsFrom(parts.table.Payload[payloadColumns])...)
		rows = append(rows, rowsFrom(parts.table.Payload[payloadRows])...)
		calls = append(calls, stringsOf(parts.table.Payload[payloadCalls])...)
		source = parts.table.SourceToolCallID
		id = parts.table.ID
	} else if len(parts.cards) > 0 {
		source = parts.cards[0].SourceToolCallID
	}

	records := make([]bunchedRecord, 0, len(parts.cards)+1)
	for _, earlier := range parts.cards {
		records = append(records, recordOf(earlier))
	}
	records = append(records, recordOf(card))

	for _, record := range records {
		if slices.Contains(calls, record.callID) {
			continue
		}
		calls = append(calls, record.callID)
		columns = mergeColumns(columns, record.fields)
		rows = append(rows, rowOf(record, columns, recordEntity))
	}

	payload := map[string]any{
		payloadDisplay:  assistantartifact.DisplayVersion,
		payloadTool:     tool,
		payloadEntity:   entity,
		payloadColumns:  columns,
		payloadRows:     rows,
		payloadRowCount: len(rows),
		"searchedFor":   []string{},
		payloadBunched:  true,
		payloadCalls:    calls,
	}
	if recordEntity != "" {
		payload["recordEntity"] = recordEntity
	}
	fitRows(payload, payloadRows)

	return &assistantartifact.Artifact{
		ID:     id,
		Kind:   assistantartifact.KindTableView,
		Status: assistantartifact.StatusReady,
		Title: artifactTitle(fmt.Sprintf("%s (%d)",
			stringutils.CapitalizeFirst(stringutils.HumanizeSnakeCase(entity)), len(rows))),
		Payload:          payload,
		SourceToolCallID: source,
	}
}

func recordOf(card *assistantartifact.Artifact) bunchedRecord {
	return bunchedRecord{
		callID: card.SourceToolCallID,
		id:     typeutils.StringOfTrimmed(card.Payload[payloadRecordID]),
		fields: fieldsFrom(card.Payload[payloadFields]),
	}
}

func mergeColumns(
	columns []assistantartifact.DisplayColumn,
	fields []assistantartifact.DisplayField,
) []assistantartifact.DisplayColumn {
	for _, field := range fields {
		if len(columns) == maxBunchColumns {
			break
		}
		if slices.ContainsFunc(columns, func(column assistantartifact.DisplayColumn) bool {
			return column.Key == field.Key
		}) {
			continue
		}
		columns = append(columns, field.DisplayColumn)
	}

	return columns
}

func rowOf(
	record bunchedRecord,
	columns []assistantartifact.DisplayColumn,
	recordEntity string,
) map[string]any {
	row := make(map[string]any, len(columns)+1)
	for _, field := range record.fields {
		if slices.ContainsFunc(columns, func(column assistantartifact.DisplayColumn) bool {
			return column.Key == field.Key
		}) {
			row[field.Key] = field.Value
		}
	}
	if recordEntity != "" && record.id != "" {
		row[recordIDKey] = record.id
	}

	return row
}

func fieldsFrom(value any) []assistantartifact.DisplayField {
	if fields, ok := value.([]assistantartifact.DisplayField); ok {
		return fields
	}

	var fields []assistantartifact.DisplayField
	if !decodeDisplay(value, &fields) {
		return nil
	}

	return fields
}

func columnsFrom(value any) []assistantartifact.DisplayColumn {
	if columns, ok := value.([]assistantartifact.DisplayColumn); ok {
		return columns
	}

	var columns []assistantartifact.DisplayColumn
	if !decodeDisplay(value, &columns) {
		return nil
	}

	return columns
}

func rowsFrom(value any) []any {
	if rows, ok := value.([]any); ok {
		return rows
	}

	var rows []any
	if !decodeDisplay(value, &rows) {
		return nil
	}

	return rows
}

func decodeDisplay(value, into any) bool {
	if value == nil {
		return false
	}
	encoded, err := sonic.Marshal(value)
	if err != nil {
		return false
	}

	return sonic.Unmarshal(encoded, into) == nil
}

func bunchedCalls(artifacts []*assistantartifact.Artifact) map[string]struct{} {
	calls := make(map[string]struct{})
	for _, artifact := range artifacts {
		if artifact == nil || artifact.Kind != assistantartifact.KindTableView ||
			!typeutils.BoolOf(artifact.Payload[payloadBunched]) {
			continue
		}
		for _, call := range stringsOf(artifact.Payload[payloadCalls]) {
			if strings.TrimSpace(call) != "" {
				calls[call] = struct{}{}
			}
		}
	}

	return calls
}
