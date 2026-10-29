package deskbench

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type unshadowed struct {
	tenantports.AgentControlRepository
}

func (u unshadowed) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*agentcontrol.AgentControl, error) {
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
	entity *agentcontrol.AgentControl,
) (*agentcontrol.AgentControl, error) {
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
