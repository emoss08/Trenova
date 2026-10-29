package securityactivityresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func connectionRequest(
	ctx context.Context,
	input *gqlmodel.DataTableConnectionInput,
	authCtx *authctx.AuthContext,
) (*iamports.ListIAMConnectionRequest, error) {
	tableInput, err := gqlconnection.DataTableConnectionFromGraphQL(
		ctx,
		input,
		resolvercore.TenantInfo(authCtx),
	)
	if err != nil {
		return nil, err
	}

	return &iamports.ListIAMConnectionRequest{
		Filter: tableInput.Filter,
		Cursor: tableInput.Cursor,
	}, nil
}

func authEventConnectionToModel(
	result *pagination.CursorListResult[*iam.AuthEvent],
) (*gqlmodel.AuthEventConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *iam.AuthEvent, cursor string) *gqlmodel.AuthEventEdge {
			return &gqlmodel.AuthEventEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AuthEventEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AuthEventConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func riskDecisionConnectionToModel(
	result *pagination.CursorListResult[*iam.RiskDecision],
) (*gqlmodel.RiskDecisionConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *iam.RiskDecision, cursor string) *gqlmodel.RiskDecisionEdge {
			return &gqlmodel.RiskDecisionEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.RiskDecisionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RiskDecisionConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func externalIdentityConnectionToModel(
	result *pagination.CursorListResult[*iam.ExternalIdentity],
) (*gqlmodel.ExternalIdentityConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *iam.ExternalIdentity, cursor string) *gqlmodel.ExternalIdentityEdge {
			return &gqlmodel.ExternalIdentityEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.ExternalIdentityEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ExternalIdentityConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func mfaAuthenticatorConnectionToModel(
	result *pagination.CursorListResult[*iam.MFAAuthenticator],
) (*gqlmodel.MFAAuthenticatorConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *iam.MFAAuthenticator, cursor string) *gqlmodel.MFAAuthenticatorEdge {
			return &gqlmodel.MFAAuthenticatorEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.MFAAuthenticatorEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.MFAAuthenticatorConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func scimDirectoryConnectionToModel(
	result *pagination.CursorListResult[*iam.SCIMDirectory],
) (*gqlmodel.SCIMDirectoryConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *iam.SCIMDirectory, cursor string) *gqlmodel.SCIMDirectoryEdge {
			return &gqlmodel.SCIMDirectoryEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.SCIMDirectoryEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.SCIMDirectoryConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}
