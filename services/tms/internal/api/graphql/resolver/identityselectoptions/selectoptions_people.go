package identityselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Sources) Organization(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.OrganizationService.GetByIDs(
			ctx,
			tenantports.GetOrganizationsByIDsRequest{
				TenantInfo:      req.TenantInfo,
				OrganizationIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			organizationID,
			organizationSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.OrganizationService.SelectOptions(
		ctx,
		&tenantports.SelectOrganizationOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			Scope:              selectoption.StringFilter(req.Filters, "scope"),
			ExcludeCurrent:     selectoption.BoolFilter(req.Filters, "excludeCurrent"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		organizationSelectOptionItem,
	)
}

func (r *Sources) User(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.UserService.GetByID(
				ctx,
				tenantports.GetUserByIDRequest{
					TenantInfo:   req.TenantInfo,
					LookupUserID: id,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, userSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.UserService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		userSelectOptionItem,
	)
}

func (r *Sources) Role(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.RoleService.GetRoleByID(
				ctx,
				permissionports.GetRoleByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, roleSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.RoleService.SelectRoleOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		roleSelectOptionItem,
	)
}

func organizationSelectOptionItem(entity *tenant.Organization) selectoption.Item {
	return selectoption.ItemFor(
		organizationSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func organizationSelectOption(entity *tenant.Organization) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.ScacCode),
		Meta: map[string]any{
			"scacCode": entity.ScacCode,
			"city":     entity.City,
		},
	}
}

func organizationID(entity *tenant.Organization) pulid.ID {
	return entity.ID
}

func userSelectOptionItem(entity *tenant.User) selectoption.Item {
	return selectoption.ItemFor(
		userSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func userSelectOption(entity *tenant.User) *gqlmodel.SelectOption {
	label := entity.Name
	if label == "" {
		label = entity.EmailAddress
	}
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       label,
		Description: stringutils.Ptr(entity.EmailAddress),
		Meta: map[string]any{
			"name":         entity.Name,
			"email":        entity.EmailAddress,
			"emailAddress": entity.EmailAddress,
		},
	}
}

func userID(entity *tenant.User) pulid.ID {
	return entity.ID
}

func roleSelectOptionItem(entity *permission.Role) selectoption.Item {
	return selectoption.ItemFor(
		roleSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func roleSelectOption(entity *permission.Role) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Description),
	}
}

func roleID(entity *permission.Role) pulid.ID {
	return entity.ID
}

type Sources struct {
	OrganizationService tenantports.OrganizationService
	UserService         *userservice.Service
	RoleService         *roleservice.Service
}
