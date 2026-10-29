package artifactpayload

import (
	"maps"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	PayloadTruncated      = "truncated"
	PayloadHasMore        = "hasMore"
	PayloadNextCursor     = "nextCursor"
	PayloadNextOffset     = "nextOffset"
	PayloadTotal          = "total"
	PayloadTotalEstimated = "totalEstimated"
	PayloadPages          = "pages"

	ResultHasMore        = "hasMore"
	ResultNextCursor     = "nextCursor"
	ResultNextOffset     = "nextOffset"
	ResultTotal          = "total"
	ResultTotalEstimated = "totalEstimated"
	ResultItems          = "items"

	ArgAfter  = "after"
	ArgOffset = "offset"
)

func NotePaging(payload, result map[string]any) {
	payload[PayloadPages] = 1
	noteTotal(payload, result)
	if typeutils.BoolOf(payload[PayloadTruncated]) {
		payload[PayloadHasMore] = true
		return
	}
	notePosition(payload, result)
}

func noteTotal(payload, result map[string]any) {
	total, counted := typeutils.IntOf(result[ResultTotal])
	if !counted {
		return
	}
	payload[PayloadTotal] = total
	if typeutils.BoolOf(result[ResultTotalEstimated]) {
		payload[PayloadTotalEstimated] = true
	} else {
		delete(payload, PayloadTotalEstimated)
	}
}

func notePosition(payload, result map[string]any) {
	delete(payload, PayloadNextCursor)
	delete(payload, PayloadNextOffset)
	more := typeutils.BoolOf(result[ResultHasMore])
	payload[PayloadHasMore] = more
	if !more {
		return
	}
	if cursor := typeutils.StringOfTrimmed(result[ResultNextCursor]); cursor != "" {
		payload[PayloadNextCursor] = cursor
	}
	if next, ok := typeutils.IntOf(result[ResultNextOffset]); ok {
		payload[PayloadNextOffset] = next
	}
}

func PagedTable(artifact *assistantartifact.Artifact) bool {
	if artifact.Kind != assistantartifact.KindTableView {
		return false
	}
	if _, opens := artifact.Payload["path"]; opens {
		return false
	}
	pages, _ := typeutils.IntOf(artifact.Payload[PayloadPages])

	return typeutils.BoolOf(artifact.Payload[PayloadHasMore]) || pages > 1
}

type PagePosition struct {
	after  string
	offset int
}

func PositionOf(arguments map[string]any) (PagePosition, bool) {
	at := PagePosition{after: typeutils.StringOfTrimmed(arguments[ArgAfter])}
	at.offset, _ = typeutils.IntOf(arguments[ArgOffset])

	return at, at.after != "" || at.offset > 0
}

func (at PagePosition) Continues(artifact *assistantartifact.Artifact, toolName string) bool {
	if artifact == nil || artifact.Kind != assistantartifact.KindTableView ||
		typeutils.StringOfTrimmed(artifact.Payload[PayloadTool]) != toolName {
		return false
	}
	if at.after != "" {
		return typeutils.StringOfTrimmed(artifact.Payload[PayloadNextCursor]) == at.after
	}
	next, ok := typeutils.IntOf(artifact.Payload[PayloadNextOffset])

	return ok && next == at.offset
}

func WithPage(
	table *assistantartifact.Artifact,
	result map[string]any,
) *assistantartifact.Artifact {
	payload := maps.Clone(table.Payload)
	records := RecordsOf(RowsFrom(result[ResultItems]))
	held := RowsFrom(payload[PayloadRows])
	rows := make([]any, 0, len(held)+len(records))
	rows = append(rows, held...)
	rows = append(rows, ProjectRows(
		ColumnsFrom(payload[PayloadColumns]),
		typeutils.StringOfTrimmed(payload["recordEntity"]),
		records,
	)...)

	read, _ := typeutils.IntOf(payload[PayloadRowCount])
	pages, _ := typeutils.IntOf(payload[PayloadPages])
	payload[PayloadRows] = rows
	payload[PayloadRowCount] = read + len(records)
	payload[PayloadPages] = max(pages, 1) + 1
	noteTotal(payload, result)
	notePosition(payload, result)
	if PayloadSize(payload) > assistantartifact.MaxPayloadBytes {
		return nil
	}

	next := *table
	next.Payload = payload

	return &next
}
