package assistantservice

import (
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

const (
	payloadTruncated      = "truncated"
	payloadHasMore        = "hasMore"
	payloadNextCursor     = "nextCursor"
	payloadNextOffset     = "nextOffset"
	payloadTotal          = "total"
	payloadTotalEstimated = "totalEstimated"
	payloadPages          = "pages"

	resultHasMore        = "hasMore"
	resultNextCursor     = "nextCursor"
	resultNextOffset     = "nextOffset"
	resultTotal          = "total"
	resultTotalEstimated = "totalEstimated"
	resultItems          = "items"

	argAfter  = "after"
	argOffset = "offset"
)

func notePaging(payload, result map[string]any) {
	payload[payloadPages] = 1
	noteTotal(payload, result)
	if typeutils.BoolOf(payload[payloadTruncated]) {
		payload[payloadHasMore] = true
		return
	}
	notePosition(payload, result)
}

func noteTotal(payload, result map[string]any) {
	total, counted := typeutils.IntOf(result[resultTotal])
	if !counted {
		return
	}
	payload[payloadTotal] = total
	if typeutils.BoolOf(result[resultTotalEstimated]) {
		payload[payloadTotalEstimated] = true
	} else {
		delete(payload, payloadTotalEstimated)
	}
}

func notePosition(payload, result map[string]any) {
	delete(payload, payloadNextCursor)
	delete(payload, payloadNextOffset)
	more := typeutils.BoolOf(result[resultHasMore])
	payload[payloadHasMore] = more
	if !more {
		return
	}
	if cursor := typeutils.StringOfTrimmed(result[resultNextCursor]); cursor != "" {
		payload[payloadNextCursor] = cursor
	}
	if next, ok := typeutils.IntOf(result[resultNextOffset]); ok {
		payload[payloadNextOffset] = next
	}
}

func pagedTable(artifact *assistantartifact.Artifact) bool {
	if artifact.Kind != assistantartifact.KindTableView {
		return false
	}
	if _, opens := artifact.Payload["path"]; opens {
		return false
	}
	pages, _ := typeutils.IntOf(artifact.Payload[payloadPages])

	return typeutils.BoolOf(artifact.Payload[payloadHasMore]) || pages > 1
}

type pagePosition struct {
	after  string
	offset int
}

func positionOf(arguments map[string]any) (pagePosition, bool) {
	at := pagePosition{after: typeutils.StringOfTrimmed(arguments[argAfter])}
	at.offset, _ = typeutils.IntOf(arguments[argOffset])

	return at, at.after != "" || at.offset > 0
}

func (at pagePosition) continues(artifact *assistantartifact.Artifact, toolName string) bool {
	if artifact == nil || artifact.Kind != assistantartifact.KindTableView ||
		typeutils.StringOfTrimmed(artifact.Payload[payloadTool]) != toolName {
		return false
	}
	if at.after != "" {
		return typeutils.StringOfTrimmed(artifact.Payload[payloadNextCursor]) == at.after
	}
	next, ok := typeutils.IntOf(artifact.Payload[payloadNextOffset])

	return ok && next == at.offset
}

func (r *artifactRecorder) continueTable(
	observation *agentports.ToolObservation,
) (*agentports.ShownArtifact, bool) {
	if observation.Failed || observation.Data == nil || observation.Call.ID == "" {
		return nil, false
	}
	at, ok := positionOf(observation.Call.Arguments)
	if !ok {
		return nil, false
	}
	document, ok := toJSONDocument(observation.Data)
	if !ok {
		return nil, false
	}
	if _, rows := document.fields[resultItems].([]any); !rows {
		return nil, false
	}
	table := r.tableContinuedBy(&observation.Call, at)
	if table == nil {
		return nil, false
	}

	next := withPage(table, document.fields)
	if next == nil {
		return shownArtifact(table), true
	}
	saved, err := r.store(next)
	if err != nil {
		return shownArtifact(table), true
	}

	return shownArtifact(saved), true
}

func (r *artifactRecorder) tableContinuedBy(
	call *agentports.ToolCall,
	at pagePosition,
) *assistantartifact.Artifact {
	key := assistantartifact.LineageKeyFor(
		assistantartifact.KindTableView, call.Name, call.Arguments,
	)
	for _, recorded := range slices.Backward(r.recorded) {
		if recorded.LineageKey == key && at.continues(recorded, call.Name) {
			return recorded
		}
	}

	latest, err := r.repo.LatestInLineage(r.ctx, assistantartifactports.LatestInLineageRequest{
		ThreadID:       r.thread.ID,
		TenantInfo:     r.tenant,
		LineageKey:     key,
		ExceptToolCall: call.ID,
	})
	if err != nil {
		r.logger.Warn("the table a page continues could not be read", zap.Error(err))
		return nil
	}
	if !at.continues(latest, call.Name) {
		return nil
	}

	return latest
}

func withPage(
	table *assistantartifact.Artifact,
	result map[string]any,
) *assistantartifact.Artifact {
	payload := maps.Clone(table.Payload)
	records := recordsOf(rowsFrom(result[resultItems]))
	held := rowsFrom(payload[payloadRows])
	rows := make([]any, 0, len(held)+len(records))
	rows = append(rows, held...)
	rows = append(rows, projectRows(
		columnsFrom(payload[payloadColumns]),
		typeutils.StringOfTrimmed(payload["recordEntity"]),
		records,
	)...)

	read, _ := typeutils.IntOf(payload[payloadRowCount])
	pages, _ := typeutils.IntOf(payload[payloadPages])
	payload[payloadRows] = rows
	payload[payloadRowCount] = read + len(records)
	payload[payloadPages] = max(pages, 1) + 1
	noteTotal(payload, result)
	notePosition(payload, result)
	if payloadSize(payload) > assistantartifact.MaxPayloadBytes {
		return nil
	}

	next := *table
	next.Payload = payload

	return &next
}
