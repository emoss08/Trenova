package emailprofileresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/pkg/pagination"
)

func emailProfileColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.EmailProfileSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func emailProfileConnectionToModel(
	result *pagination.CursorListResult[*email.Profile],
) (*gqlmodel.EmailProfileConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *email.Profile, cursor string) *gqlmodel.EmailProfileEdge {
			return &gqlmodel.EmailProfileEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.EmailProfileEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EmailProfileConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func emailLogConnectionToModel(
	result *pagination.CursorListResult[*email.Message],
) (*gqlmodel.EmailLogConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *email.Message, cursor string) *gqlmodel.EmailLogEdge {
			return &gqlmodel.EmailLogEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EmailLogEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EmailLogConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func applyEmailProfileInput(profile *email.Profile, input *gqlmodel.EmailProfileInput) {
	profile.Name = input.Name
	profile.Description = gqlscalar.StringValue(input.Description)
	profile.SenderName = input.SenderName
	profile.SenderEmail = input.SenderEmail
	profile.ReplyToEmail = gqlscalar.StringValue(input.ReplyToEmail)
	profile.Provider = email.ProviderResend
	if input.Provider != nil {
		profile.Provider = *input.Provider
	}
	profile.Status = email.ProfileStatusActive
	if input.Status != nil {
		profile.Status = *input.Status
	}
}

func assignmentsFromInput(
	input []*gqlmodel.EmailProfileAssignmentInput,
) ([]*email.ProfileAssignment, error) {
	assignments := make([]*email.ProfileAssignment, 0, len(input))
	for _, item := range input {
		if item == nil {
			continue
		}
		profileID, err := gqlid.ParseIDField("profileId", item.ProfileID)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, &email.ProfileAssignment{
			Purpose:   item.Purpose,
			ProfileID: profileID,
		})
	}

	return assignments, nil
}
