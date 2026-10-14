package userresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/mappers"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/usermapping"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func userConnectionToModel(
	result *pagination.CursorListResult[*tenant.User],
) (*gqlmodel.UserConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *tenant.User, cursor string) *gqlmodel.UserEdge {
			return &gqlmodel.UserEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.UserEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.UserConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func createUserRequestFromInput(
	tenantInfo pagination.TenantInfo,
	input *gqlmodel.CreateUserInput,
) (*userservice.CreateUserRequest, error) {
	roleIDs, err := base.PulidsFromStrings(input.RoleIds, "roleIds")
	if err != nil {
		return nil, err
	}

	user := &tenant.User{
		Name:         input.Name,
		Username:     input.Username,
		EmailAddress: input.EmailAddress,
		Timezone:     input.Timezone,
		Status:       input.Status,
	}
	if input.Locale != nil {
		user.Locale = *input.Locale
	}
	if input.MustChangePassword != nil {
		user.MustChangePassword = *input.MustChangePassword
	}
	if input.TimeFormat != nil {
		user.TimeFormat, err = usermapping.TimeFormatToDomain(*input.TimeFormat)
		if err != nil {
			return nil, err
		}
	}

	return &userservice.CreateUserRequest{
		TenantInfo: tenantInfo,
		User:       user,
		RoleIDs:    roleIDs,
	}, nil
}

func applyUserPatch(user *tenant.User, input *gqlmodel.UserPatchInput) error {
	if err := mappers.ApplyUserPatch(user, *input); err != nil {
		return err
	}

	value, ok := input.TimeFormat.ValueOK()
	if !ok {
		return nil
	}
	if value == nil {
		return errortypes.NewValidationError(
			"timeFormat",
			errortypes.ErrRequired,
			"Time Format cannot be cleared",
		)
	}

	timeFormat, err := usermapping.TimeFormatToDomain(*value)
	if err != nil {
		return err
	}
	user.TimeFormat = timeFormat

	return nil
}
