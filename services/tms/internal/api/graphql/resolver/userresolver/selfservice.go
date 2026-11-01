package userresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxAvatarIDs = 100

func selfUser(authCtx *authctx.AuthContext, user *tenant.User) *tenant.User {
	user.CurrentOrganizationID = authCtx.OrganizationID
	user.BusinessUnitID = authCtx.BusinessUnitID
	return user
}

func sessionScope(authCtx *authctx.AuthContext) userservice.SessionScope {
	return userservice.SessionScope{
		TenantInfo:       resolvercore.TenantInfo(authCtx),
		CurrentSessionID: authCtx.SessionID,
	}
}

func availabilityUpdateFromInput(
	input *gqlmodel.UpdateMyAvailabilityInput,
) (tenant.AvailabilityUpdate, error) {
	update := tenant.AvailabilityUpdate{
		Status:     input.Status,
		FocusUntil: gqlscalar.Int64Ptr(input.FocusUntil),
		BackOn:     gqlscalar.Int64Ptr(input.BackOn),
		AwayNote:   gqlscalar.StringValue(input.AwayNote),
	}

	delegate := strings.TrimSpace(gqlscalar.StringValue(input.DelegateUserID))
	if delegate == "" {
		return update, nil
	}

	delegateID, err := pulid.MustParse(delegate)
	if err != nil {
		return tenant.AvailabilityUpdate{}, errortypes.NewValidationError(
			"delegateUserId",
			errortypes.ErrInvalid,
			"Hand off to a person in your organization",
		)
	}
	update.DelegateUserID = &delegateID

	return update, nil
}

func avatarIDsFromInput(raw []string) ([]pulid.ID, error) {
	if len(raw) > maxAvatarIDs {
		return nil, errortypes.NewValidationError(
			"ids",
			errortypes.ErrInvalid,
			"Ask for at most 100 people at a time",
		)
	}

	ids := make([]pulid.ID, 0, len(raw))
	for _, value := range raw {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		id, err := pulid.MustParse(trimmed)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"ids",
				errortypes.ErrInvalid,
				"Every id must be a user id",
			)
		}
		ids = append(ids, id)
	}

	return ids, nil
}
