package supportaccessresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

// requirePerson refuses an API key, an agent and a Trenova support session: support access is
// granted and revoked by a person of the organization, whatever their roles say.
func requirePerson(authCtx *authctx.AuthContext) error {
	if authCtx == nil || authCtx.PrincipalType != authctx.PrincipalTypeUser ||
		authCtx.IsSupport() || supportaccess.IsPrincipalID(authCtx.UserID) || authCtx.UserID.IsNil() {
		return errortypes.NewAuthorizationError(
			"Only a person in this organization may change Trenova support access",
		)
	}
	return nil
}

func grantToModel(g *supportaccess.Grant, now int64) *gqlmodel.SupportAccessGrant {
	out := &gqlmodel.SupportAccessGrant{
		ID:            g.ID.String(),
		StaffEmail:    g.StaffEmail,
		Reason:        g.Reason,
		Scope:         gqlmodel.SupportAccessScope(g.Scope),
		DurationHours: g.DurationHours,
		Status:        gqlmodel.SupportAccessStatus(g.StatusAt(now)),
		GrantedAt:     int(g.CreatedAt),
		ExpiresAt:     int(g.ExpiresAt),
		RevokedAt:     intPtr(g.RevokedAt),
		LastEnteredAt: intPtr(g.LastEnteredAt),
		RevokeReason:  stringPtr(g.RevokeReason),
		LastStaffName: stringPtr(g.LastStaffName),
	}
	if g.GrantedBy != nil {
		out.GrantedByName = stringPtr(g.GrantedBy.Name)
	}
	if g.RevokedBy != nil {
		out.RevokedByName = stringPtr(g.RevokedBy.Name)
	}
	return out
}

func grantConnectionToModel(
	result *pagination.CursorListResult[*supportaccess.Grant],
	now int64,
) (*gqlmodel.SupportAccessGrantConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *supportaccess.Grant, cursor string) *gqlmodel.SupportAccessGrantEdge {
			return &gqlmodel.SupportAccessGrantEdge{Node: grantToModel(node, now), Cursor: cursor}
		},
		func(edge *gqlmodel.SupportAccessGrantEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.SupportAccessGrantConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func intPtr(v *int64) *int {
	if v == nil {
		return nil
	}
	out := int(*v)
	return &out
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
