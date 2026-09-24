package airetrievalstatusservice

import (
	"cmp"
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/memtable"
)

const (
	MaxFailedEntries       = 1000
	failedEntryCursorScope = "ai-retrieval-failed-entries"
)

type failedEntryRow struct {
	entry *serviceports.AIRetrievalFailedEntry
	rank  int
}

var failedEntryTable = memtable.New(memtable.Config[failedEntryRow]{
	CursorScope: failedEntryCursorScope,
	Search: func(row *failedEntryRow) string {
		return row.entry.Error + "\x00" + row.entry.SourceID.String() + "\x00" + row.entry.ModelKey
	},
	Order: func(a, b *failedEntryRow) int { return cmp.Compare(a.rank, b.rank) },
	Fields: []memtable.Field[failedEntryRow]{
		{
			Name: "sourceType",
			Kind: memtable.KindEnum,
			Values: []string{
				airetrieval.SourceTypeMemory.String(),
				airetrieval.SourceTypeDocument.String(),
				airetrieval.SourceTypeInboundMessage.String(),
			},
			Filterable: true,
			Sortable:   true,
			Text:       func(row *failedEntryRow) string { return row.entry.SourceType.String() },
		},
		{
			Name: "status",
			Kind: memtable.KindEnum,
			Values: []string{
				airetrieval.IndexStatusPending.String(),
				airetrieval.IndexStatusFailed.String(),
			},
			Filterable: true,
			Sortable:   true,
			Text:       func(row *failedEntryRow) string { return row.entry.Status.String() },
		},
		{
			Name:       "sourceId",
			Kind:       memtable.KindText,
			Filterable: true,
			Sortable:   true,
			Text:       func(row *failedEntryRow) string { return row.entry.SourceID.String() },
		},
		{
			Name:       "modelKey",
			Kind:       memtable.KindText,
			Filterable: true,
			Sortable:   true,
			Text:       func(row *failedEntryRow) string { return row.entry.ModelKey },
		},
		{
			Name:       "error",
			Kind:       memtable.KindText,
			Filterable: true,
			Sortable:   true,
			Text:       func(row *failedEntryRow) string { return row.entry.Error },
		},
		{
			Name:       "attempts",
			Kind:       memtable.KindNumber,
			Filterable: true,
			Sortable:   true,
			Number: func(row *failedEntryRow) (float64, bool) {
				return float64(row.entry.Attempts), true
			},
		},
		{
			Name:     "lastAttemptAt",
			Kind:     memtable.KindNumber,
			Sortable: true,
			Number: func(row *failedEntryRow) (float64, bool) {
				return memtable.OptionalNumber(row.entry.LastAttemptAt)
			},
		},
		{
			Name:     "nextAttemptAt",
			Kind:     memtable.KindNumber,
			Sortable: true,
			Number: func(row *failedEntryRow) (float64, bool) {
				return memtable.OptionalNumber(row.entry.NextAttemptAt)
			},
		},
	},
})

func FailedEntryID(entry *airetrieval.IndexEntry) string {
	return entry.SourceType.String() + ":" + entry.SourceID.String() + "@" + entry.ModelKey
}

func failedEntryView(entry *airetrieval.IndexEntry) *serviceports.AIRetrievalFailedEntry {
	return &serviceports.AIRetrievalFailedEntry{
		ID:            FailedEntryID(entry),
		SourceType:    entry.SourceType,
		SourceID:      entry.SourceID,
		ModelKey:      entry.ModelKey,
		Status:        entry.Status,
		Attempts:      entry.Attempts,
		Error:         entry.LastError,
		LastAttemptAt: entry.LastAttemptAt,
		NextAttemptAt: entry.NextAttemptAt,
	}
}

func (s *Service) ListFailedEntries(
	ctx context.Context,
	req *serviceports.ListAIRetrievalFailedEntriesRequest,
) (*serviceports.AIRetrievalFailedEntryPage, error) {
	if req == nil {
		req = &serviceports.ListAIRetrievalFailedEntriesRequest{}
	}
	if err := validateTenant(req.TenantInfo); err != nil {
		return nil, err
	}
	if req.SourceType != "" {
		if err := validateSourceType(req.SourceType); err != nil {
			return nil, err
		}
	}

	settings, err := s.repo.GetSettings(ctx, req.TenantInfo)
	if err != nil {
		return nil, fmt.Errorf("read retrieval settings: %w", err)
	}

	rows := make([]failedEntryRow, 0)
	if keys := settings.IndexedModelKeys(); len(keys) > 0 {
		entries, lerr := s.repo.ListErroredIndexEntries(
			ctx,
			repositories.ListErroredIndexEntriesRequest{
				TenantInfo: req.TenantInfo,
				SourceType: req.SourceType,
				ModelKeys:  keys,
				Limit:      MaxFailedEntries,
			},
		)
		if lerr != nil {
			return nil, fmt.Errorf("list failed index entries: %w", lerr)
		}
		rows = make([]failedEntryRow, 0, len(entries))
		for idx, entry := range entries {
			rows = append(rows, failedEntryRow{entry: failedEntryView(entry), rank: idx})
		}
	}

	page, err := failedEntryTable.List(rows, &req.Table)
	if err != nil {
		return nil, err
	}

	out := &serviceports.AIRetrievalFailedEntryPage{
		Edges:       make([]*serviceports.AIRetrievalFailedEntryEdge, 0, len(page.Items)),
		HasNextPage: page.HasNextPage,
		TotalCount:  page.TotalCount,
	}
	for idx, row := range page.Items {
		out.Edges = append(out.Edges, &serviceports.AIRetrievalFailedEntryEdge{
			Node:   row.entry,
			Cursor: page.Cursors[idx],
		})
	}

	return out, nil
}
