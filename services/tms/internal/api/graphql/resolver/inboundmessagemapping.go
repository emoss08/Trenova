package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/errortypes"
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

func inboundMessageConnection(
	result *pagination.CursorListResult[*inboundmessage.InboundMessage],
) (*gqlmodel.InboundMessageConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *inboundmessage.InboundMessage, cursor string) *gqlmodel.InboundMessageEdge {
			return &gqlmodel.InboundMessageEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.InboundMessageEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.InboundMessageConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

// inboundPreviewLength is how much of a body a list row shows: about two lines
// at the reading pane's narrowest.
const inboundPreviewLength = 180

// optionalMatch reads a matched record for display. A record that is gone or
// out of the reader's reach is no match, not a failed inbox; anything else is
// a real failure and stays one.
func optionalMatch[T any](value T, err error) (T, error) {
	if err != nil {
		var zero T
		if errortypes.IsNotFoundError(err) {
			return zero, nil
		}

		return zero, err
	}

	return value, nil
}

func requestLoaders(ctx context.Context) (*loaders.Loaders, error) {
	l, ok := loaders.FromContext(ctx)
	if !ok || l == nil {
		return nil, errortypes.NewDatabaseError("Request loaders are not configured")
	}

	return l, nil
}
