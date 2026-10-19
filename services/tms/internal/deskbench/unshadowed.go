package deskbench

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type unshadowed struct {
	tenantports.AgentControlRepository
}

func (u unshadowed) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	control, err := u.AgentControlRepository.GetOrCreate(ctx, tenantInfo)
	if err != nil || control == nil {
		return control, err
	}

	live := *control
	live.ShadowMode = false

	return &live, nil
}

func (u unshadowed) Update(
	ctx context.Context,
	entity *tenant.AgentControl,
) (*tenant.AgentControl, error) {
	stored, err := u.AgentControlRepository.GetOrCreate(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	kept := *entity
	kept.ShadowMode = stored.ShadowMode

	return u.AgentControlRepository.Update(ctx, &kept)
}
