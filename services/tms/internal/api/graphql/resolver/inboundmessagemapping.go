package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// optionalPulid renders a nullable id. A nil id is absent rather than an empty
// string, because "" would read on the client as an id that is known and blank.
func inboundOptionalID(id pulid.ID) *string {
	if id.IsNil() {
		return nil
	}
	value := id.String()

	return &value
}

// inboundMessageCursor is the edge cursor for one row, on the keyset the list
// pages by, so a client can resume from any edge it holds rather than only
// from the end of a page.
func inboundMessageCursor(message *inboundmessage.InboundMessage) string {
	encoded, err := pagination.EncodeCursor(pagination.Cursor{
		CreatedAt: message.CreatedAt,
		ID:        message.ID,
	})
	if err != nil {
		return ""
	}

	return encoded
}

func inboundMessageConnection(
	result *pagination.CursorListResult[*inboundmessage.InboundMessage],
) *gqlmodel.InboundMessageConnection {
	edges := make([]*gqlmodel.InboundMessageEdge, 0, len(result.Items))
	for _, message := range result.Items {
		edges = append(edges, &gqlmodel.InboundMessageEdge{
			Node:   message,
			Cursor: inboundMessageCursor(message),
		})
	}

	pageInfo := &gqlmodel.PageInfo{HasNextPage: result.HasNextPage}
	if len(edges) > 0 {
		endCursor := edges[len(edges)-1].Cursor
		pageInfo.EndCursor = &endCursor
	}

	return &gqlmodel.InboundMessageConnection{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: result.TotalCount,
	}
}
