package toolartifacts

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	PayloadBunched  = "bunched"
	PayloadCalls    = "calls"
	PayloadRecordID = "recordId"
	payloadFields   = "fields"
	payloadDisplay  = "display"
	PayloadReadBy   = "readBy"
	maxBunchColumns = 8
)

type bunchedRecord struct {
	callID string
	id     string
	fields []assistantartifact.DisplayField
}

type bunchParts struct {
	Table *assistantartifact.Artifact
	Cards []*assistantartifact.Artifact
}

func PartsOf(earlier []*assistantartifact.Artifact, tool string) bunchParts {
	var parts bunchParts
	for _, artifact := range earlier {
		if artifact == nil ||
			typeutils.StringOfTrimmed(artifact.Payload[artifactpayload.PayloadTool]) != tool {
			continue
		}
		switch {
		case artifact.Kind == assistantartifact.KindTableView &&
			typeutils.BoolOf(artifact.Payload[PayloadBunched]):
			if parts.Table == nil {
				parts.Table = artifact
			}
		case artifact.Kind == assistantartifact.KindEntityCard:
			parts.Cards = append(parts.Cards, artifact)
		}
	}

	return parts
}

func BunchTable(
	tool string,
	parts bunchParts,
	card *assistantartifact.Artifact,
) *assistantartifact.Artifact {
	entity := typeutils.StringOfTrimmed(card.Payload[artifactpayload.PayloadEntity])
	recordEntity := artifactpayload.RecordEntityOf(entity)

	columns := make([]assistantartifact.DisplayColumn, 0, maxBunchColumns)
	rows := make([]any, 0, len(parts.Cards)+2)
	calls := make([]string, 0, len(parts.Cards)+2)
	source := card.SourceToolCallID
	var id pulid.ID
	if parts.Table != nil {
		columns = append(
			columns,
			artifactpayload.ColumnsFrom(parts.Table.Payload[artifactpayload.PayloadColumns])...)
		rows = append(
			rows,
			artifactpayload.RowsFrom(parts.Table.Payload[artifactpayload.PayloadRows])...)
		calls = append(calls, artifactpayload.StringsOf(parts.Table.Payload[PayloadCalls])...)
		source = parts.Table.SourceToolCallID
		id = parts.Table.ID
	} else if len(parts.Cards) > 0 {
		source = parts.Cards[0].SourceToolCallID
	}

	records := make([]bunchedRecord, 0, len(parts.Cards)+1)
	for _, earlier := range parts.Cards {
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
		payloadDisplay:                     assistantartifact.DisplayVersion,
		artifactpayload.PayloadTool:        tool,
		artifactpayload.PayloadEntity:      entity,
		artifactpayload.PayloadColumns:     columns,
		artifactpayload.PayloadRows:        rows,
		artifactpayload.PayloadRowCount:    len(rows),
		artifactpayload.PayloadSearchedFor: []string{},
		PayloadBunched:                     true,
		PayloadCalls:                       calls,
	}
	if recordEntity != "" {
		payload["recordEntity"] = recordEntity
	}
	fitRows(payload, artifactpayload.PayloadRows)

	return &assistantartifact.Artifact{
		ID:     id,
		Kind:   assistantartifact.KindTableView,
		Status: assistantartifact.StatusReady,
		Title: assistantartifact.NormalizeTitle(fmt.Sprintf("%s (%d)",
			assistantartifact.RecordsTitle(entity), len(rows))),
		Payload:          payload,
		SourceToolCallID: source,
	}
}

func recordOf(card *assistantartifact.Artifact) bunchedRecord {
	return bunchedRecord{
		callID: card.SourceToolCallID,
		id:     typeutils.StringOfTrimmed(card.Payload[PayloadRecordID]),
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
		row[artifactpayload.RecordIDKey] = record.id
	}

	return row
}

func fieldsFrom(value any) []assistantartifact.DisplayField {
	if fields, ok := value.([]assistantartifact.DisplayField); ok {
		return fields
	}

	var fields []assistantartifact.DisplayField
	if !artifactpayload.DecodeDisplay(value, &fields) {
		return nil
	}

	return fields
}

func BunchedCalls(artifacts []*assistantartifact.Artifact) map[string]struct{} {
	calls := make(map[string]struct{})
	for _, artifact := range artifacts {
		if artifact == nil || artifact.Kind != assistantartifact.KindTableView ||
			!typeutils.BoolOf(artifact.Payload[PayloadBunched]) {
			continue
		}
		for _, call := range artifactpayload.StringsOf(artifact.Payload[PayloadCalls]) {
			if strings.TrimSpace(call) != "" {
				calls[call] = struct{}{}
			}
		}
	}

	return calls
}
