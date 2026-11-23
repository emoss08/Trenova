package desktranscriptresolver

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/loaderfields"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const removedRequester = "A removed person"

func filterFromInput(
	input *gqlmodel.DeskConversationFilterInput,
) (desktranscript.ConversationFilter, error) {
	filter := desktranscript.ConversationFilter{Timeframe: desktranscript.AllTime()}
	if input == nil {
		return filter, nil
	}

	userID, err := gqlid.ParseOptionalIDField("filter.userId", input.UserID)
	if err != nil {
		return filter, err
	}
	agentID, err := gqlid.ParseOptionalIDField("filter.agentId", input.AgentID)
	if err != nil {
		return filter, err
	}
	filter.UserID = userID
	filter.AgentDefinitionID = agentID
	filter.Origins = input.Origins
	if input.Timeframe != nil {
		filter.Timeframe = desktranscript.Timeframe{
			Preset:   input.Timeframe.Preset,
			FromDate: gqlscalar.StringValue(input.Timeframe.FromDate),
			ToDate:   gqlscalar.StringValue(input.Timeframe.ToDate),
		}
	}

	return filter, nil
}

func refuseTableFilters(input *gqlmodel.DataTableConnectionInput) error {
	if len(input.FieldFilters) > 0 || len(input.FilterGroups) > 0 {
		return errortypes.NewValidationError(
			"fieldFilters",
			errortypes.ErrInvalid,
			"Narrow the conversations with the timeframe, person, agent and origin above the table",
		)
	}

	return nil
}

func conversationConnection(
	result *pagination.CursorListResult[*desktranscript.Conversation],
) (*gqlmodel.DeskConversationConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *desktranscript.Conversation, cursor string) *gqlmodel.DeskConversationEdge {
			return &gqlmodel.DeskConversationEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.DeskConversationEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, fmt.Errorf("build desk conversation connection: %w", err)
	}

	return &gqlmodel.DeskConversationConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func exportConnection(
	result *pagination.CursorListResult[*desktranscript.Export],
) (*gqlmodel.DeskTranscriptExportConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *desktranscript.Export, cursor string) *gqlmodel.DeskTranscriptExportEdge {
			return &gqlmodel.DeskTranscriptExportEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.DeskTranscriptExportEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, fmt.Errorf("build desk transcript export connection: %w", err)
	}

	return &gqlmodel.DeskTranscriptExportConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func requesterName(ctx context.Context, export *desktranscript.Export) (string, error) {
	user, err := loaderfields.LoadUser(ctx, export.RequestedByUserID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return removedRequester, nil
		}

		return "", err
	}
	if user == nil || user.Name == "" {
		return removedRequester, nil
	}

	return user.Name, nil
}
