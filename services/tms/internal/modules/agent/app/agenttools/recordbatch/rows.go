package recordbatch

import (
	"fmt"
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

// rows reads the batch's per-record arguments: one object per record, keyed
// by the single tool's id parameter, holding what that record alone takes. A
// record named twice is refused rather than one of the two being picked.
func (b *RecordBatch) rows(params map[string]any) (map[pulid.ID]map[string]any, []pulid.ID, error) {
	if b.Rows == "" {
		return nil, nil, nil
	}

	entries, err := toolparams.ReadObjects(params, b.Rows, MaxBulkRecords)
	if err != nil {
		return nil, nil, err
	}

	byID := make(map[pulid.ID]map[string]any, len(entries))
	order := make([]pulid.ID, 0, len(entries))
	for idx, entry := range entries {
		id, idErr := toolparams.RequirePulid(entry, b.SingleParam)
		if idErr != nil {
			return nil, nil, fmt.Errorf("%s[%d]: %w", b.Rows, idx, idErr)
		}
		if _, dup := byID[id]; dup {
			return nil, nil, fmt.Errorf(
				"%s names %s %s twice; give each %s one entry",
				b.Rows, b.Noun, id, b.Noun,
			)
		}
		byID[id] = entry
		order = append(order, id)
	}

	return byID, order, nil
}

// recordIDs is the records a call names: its id list when it sends one, or,
// for a batch whose records each carry their own arguments, the records its
// rows name, in the order named. Every id in the list needs its row; a row
// for a record the list leaves out is one a person approving dropped, and is
// ignored.
func (b *RecordBatch) recordIDs(params map[string]any) ([]pulid.ID, error) {
	byID, order, err := b.rows(params)
	if err != nil {
		return nil, err
	}

	if _, sent := params[b.Param]; !sent && b.Rows != "" {
		if len(order) == 0 {
			return nil, fmt.Errorf("missing required parameter %q", b.Rows)
		}

		return order, nil
	}

	ids, err := toolparams.RequirePulidSlice(params, b.Param, MaxBulkRecords)
	if err != nil {
		return nil, err
	}
	if b.Rows == "" {
		return ids, nil
	}

	missing := make([]string, 0)
	for _, id := range ids {
		if _, ok := byID[id]; !ok {
			missing = append(missing, id.String())
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf(
			"%s has no entry for %s; name each %s once in %s with what it takes",
			b.Rows, strings.Join(missing, ", "), b.Noun, b.Rows,
		)
	}

	return ids, nil
}

// row is the record's own arguments, less its id, which the caller sets.
func (b *RecordBatch) row(params map[string]any, id pulid.ID) map[string]any {
	byID, _, err := b.rows(params)
	if err != nil {
		return nil
	}
	entry, ok := byID[id]
	if !ok {
		return nil
	}
	out := make(map[string]any, len(entry))
	maps.Copy(out, entry)
	delete(out, b.SingleParam)

	return out
}

// rowsFor is the rows a call keeps for the records named, in their order: the
// rows of a split call, or the call the selection pins.
func (b *RecordBatch) rowsFor(params map[string]any, ids []pulid.ID) []any {
	byID, _, err := b.rows(params)
	if err != nil {
		return nil
	}
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		if entry, ok := byID[id]; ok {
			out = append(out, entry)
		}
	}

	return out
}

// PinSelection writes the records a rows batch covers into its id list, so
// the person approving sees each as a record they may drop, and drops the
// rows for records the list leaves out. A call that already lists its
// records keeps that list.
func (b *RecordBatch) PinSelection(params map[string]any) (map[string]any, error) {
	ids, err := b.recordIDs(params)
	if err != nil {
		return nil, err
	}

	pinned := make(map[string]any, len(params)+1)
	maps.Copy(pinned, params)
	list := make([]any, 0, len(ids))
	for _, id := range ids {
		list = append(list, id.String())
	}
	pinned[b.Param] = list
	if b.Rows != "" {
		pinned[b.Rows] = b.rowsFor(params, ids)
	}

	return pinned, nil
}

// BulkRowsSchema is the schema of a batch whose records each take their own
// arguments: the rows the model writes, and the list of records they cover,
// which the runtime fills from the rows and a person approving may narrow.
func BulkRowsSchema(
	batch *RecordBatch,
	idsDescription string,
	rowsDescription string,
	row map[string]any,
	extra map[string]any,
) map[string]any {
	properties := map[string]any{
		batch.Param: batch.property(idsDescription),
		batch.Rows: map[string]any{
			toolschema.KeyType:        toolschema.TypeArray,
			toolschema.KeyDescription: rowsDescription,
			toolschema.KeyMinItems:    1,
			toolschema.KeyMaxItems:    MaxBulkRecords,
			toolschema.KeyItems:       row,
		},
	}
	maps.Copy(properties, extra)

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             []string{batch.Rows},
		toolschema.KeyAdditionalProperties: false,
	}
}

// RecordIDs is the records a call covers, each once, in the order named: what
// a batch tool reads back once it has run.
func (b *RecordBatch) RecordIDs(params map[string]any) ([]pulid.ID, error) {
	ids, err := b.recordIDs(params)
	if err != nil {
		return nil, err
	}

	return sliceutils.Dedupe(ids), nil
}
