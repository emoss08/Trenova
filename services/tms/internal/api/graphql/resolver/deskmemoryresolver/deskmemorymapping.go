package deskmemoryresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

// deskActor is the person on the Desk. Reading memories is part of using the
// assistant, and so is changing one's own; the agent-memory permissions they
// hold decide whether they may also change what their team or organization
// shares, which the service enforces memory by memory.
func (r *Deps) deskActor(
	ctx context.Context,
	operation permission.Operation,
) (*services.DeskMemoryActor, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAssistant, operation)
	if err != nil {
		return nil, err
	}

	return &services.DeskMemoryActor{
		Actor:           actorutil.FromAuthContext(authCtx),
		MayCreateShared: r.HasPermission(ctx, authCtx, permission.ResourceAgentMemory, permission.OpCreate),
		MayUpdateShared: r.HasPermission(ctx, authCtx, permission.ResourceAgentMemory, permission.OpUpdate),
	}, nil
}

func optionalID(raw *string) (pulid.ID, error) {
	if raw == nil || *raw == "" {
		return pulid.Nil, nil
	}

	return pulid.Parse(*raw)
}

func idString(id *pulid.ID) *string {
	if id == nil || id.IsNil() {
		return nil
	}
	value := id.String()

	return &value
}

func deskMemoryToModel(memory *services.DeskMemory) *gqlmodel.DeskMemory {
	entity := memory.Memory
	out := &gqlmodel.DeskMemory{
		ID:          entity.ID.String(),
		Content:     entity.Content,
		Kind:        entity.Kind,
		Scope:       entity.Scope,
		RoleID:      idString(entity.RoleID),
		RoleName:    memory.RoleName,
		Status:      entity.Status,
		Source:      entity.Source,
		SourceTitle: memory.SourceTitle,
		UseCount:    entity.UseCount,
		CreatedAt:   int(entity.CreatedAt),
		Version:     int(entity.Version),
		Editable:    memory.Editable,
	}
	if entity.LastUsedAt != nil {
		at := int(*entity.LastUsedAt)
		out.LastUsedAt = &at
	}

	return out
}

func deskPageToModel(page *services.DeskMemoryPage) *gqlmodel.DeskMemoryPage {
	out := &gqlmodel.DeskMemoryPage{
		Items:  make([]*gqlmodel.DeskMemory, 0, len(page.Items)),
		All:    page.All,
		Counts: make([]*gqlmodel.DeskMemoryCount, 0, len(page.Counts)),
	}
	if page.Next != "" {
		next := page.Next
		out.Next = &next
	}
	for _, item := range page.Items {
		out.Items = append(out.Items, deskMemoryToModel(item))
	}
	for _, count := range page.Counts {
		roleID := count.RoleID
		out.Counts = append(out.Counts, &gqlmodel.DeskMemoryCount{
			Scope:  count.Scope,
			RoleID: idString(&roleID),
			Count:  count.Count,
		})
	}

	return out
}

func deskSettingsToModel(settings *services.DeskMemorySettings) *gqlmodel.DeskMemorySettings {
	out := &gqlmodel.DeskMemorySettings{
		SavingMode:               settings.SavingMode,
		Roles:                    make([]*gqlmodel.DeskMemoryRole, 0, len(settings.Roles)),
		CanShareWithOrganization: settings.CanShareWithOrganization,
	}
	for _, role := range settings.Roles {
		out.Roles = append(out.Roles, &gqlmodel.DeskMemoryRole{
			ID:       role.ID.String(),
			Name:     role.Name,
			Writable: role.Writable,
		})
	}

	return out
}
