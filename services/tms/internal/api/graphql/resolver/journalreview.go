package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func journalReviewRequest(
	authCtx *authctx.AuthContext,
	input gqlmodel.JournalReviewInput,
) (*services.JournalReviewRequest, error) {
	entryIDs := make([]pulid.ID, 0, len(input.EntryIds))
	for _, raw := range input.EntryIds {
		id, err := pulid.MustParse(raw)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"entryIds",
				errortypes.ErrInvalid,
				"Invalid journal entry",
			)
		}
		entryIDs = append(entryIDs, id)
	}

	return &services.JournalReviewRequest{
		TenantInfo: tenantInfo(authCtx),
		EntryIDs:   entryIDs,
	}, nil
}

func journalEntryConnectionToModel(
	result *pagination.CursorListResult[*journalentry.JournalEntry],
) (*gqlmodel.JournalEntryConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *journalentry.JournalEntry, cursor string) *gqlmodel.JournalEntryEdge {
			return &gqlmodel.JournalEntryEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.JournalEntryEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.JournalEntryConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
