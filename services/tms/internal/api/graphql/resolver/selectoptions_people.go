package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Resolver) resolveWorkerSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.workerService.Get(
				ctx,
				repositories.GetWorkerByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, workerSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.workerService.SelectOptions(
		ctx,
		&repositories.WorkerSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			OwnerOperatorsOnly: selectOptionBoolFilter(req.filters, "ownerOperatorsOnly"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		workerSelectOptionItem,
	)
}

func (r *Resolver) resolveOrganizationSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.organizationService.GetByIDs(
			ctx,
			portservices.GetOrganizationsByIDsRequest{
				TenantInfo:      req.tenantInfo,
				OrganizationIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(
			req.ids,
			entities,
			organizationID,
			organizationSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.organizationService.SelectOptions(
		ctx,
		&repositories.SelectOrganizationOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			Scope:              selectOptionStringFilter(req.filters, "scope"),
			ExcludeCurrent:     selectOptionBoolFilter(req.filters, "excludeCurrent"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		organizationSelectOptionItem,
	)
}

func (r *Resolver) resolveUserSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.userService.GetByID(
				ctx,
				repositories.GetUserByIDRequest{
					TenantInfo:   req.tenantInfo,
					LookupUserID: id,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, userSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.userService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		userSelectOptionItem,
	)
}

func (r *Resolver) resolveRoleSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.roleService.GetRoleByID(
				ctx,
				repositories.GetRoleByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, roleSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.roleService.SelectRoleOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		roleSelectOptionItem,
	)
}

func workerSelectOptionItem(entity *worker.Worker) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		workerSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func workerSelectOption(entity *worker.Worker) *gqlmodel.SelectOption {
	label := entity.WholeName
	if label == "" {
		label = entity.FullName()
	}

	meta := map[string]any{
		"firstName": entity.FirstName,
		"lastName":  entity.LastName,
		"wholeName": label,
	}
	if entity.FleetCode != nil {
		meta["fleetCode"] = entity.FleetCode.Code
	}

	return &gqlmodel.SelectOption{
		ID:    entity.ID.String(),
		Label: label,
		Meta:  meta,
	}
}

func organizationSelectOptionItem(entity *tenant.Organization) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
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

func userSelectOptionItem(entity *tenant.User) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
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

func roleSelectOptionItem(entity *permission.Role) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
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
