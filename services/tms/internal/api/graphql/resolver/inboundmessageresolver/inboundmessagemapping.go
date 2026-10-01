package inboundmessageresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
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
	page, err := base.EntityCursorConnection(
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

func mailboxSettings(input gqlmodel.InboundMailboxInput) inboundmessageservice.MailboxSettings {
	return inboundmessageservice.MailboxSettings{
		Name:          input.Name,
		Address:       input.Address,
		Provider:      input.Provider,
		Purpose:       stringutils.FromPtr(input.Purpose),
		ReviewPolicy:  input.ReviewPolicy,
		MinConfidence: input.MinConfidence,
		Status:        input.Status,
	}
}
