package directoryaccounts

import (
	"context"
	"maps"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

func columnName(c keel.Col) string {
	return tenantdb.UsersSchema().Columns[c.Index].Name
}

func statusColumn() keel.Col {
	return tenantdb.Users.Status.Col
}

func changedColumns(before, after *tenant.User) (columns, access []string) {
	u := tenantdb.Users
	candidates := []struct {
		col     keel.Col
		changed bool
		access  bool
	}{
		{u.Name.Col, before.Name != after.Name, false},
		{u.EmailAddress.Col, before.EmailAddress != after.EmailAddress, true},
		{u.JobTitle.Col, before.JobTitle != after.JobTitle, false},
		{u.PhoneNumber.Col, before.PhoneNumber != after.PhoneNumber, false},
		{u.Timezone.Col, before.Timezone != after.Timezone, false},
	}

	for _, candidate := range candidates {
		if !candidate.changed {
			continue
		}
		columns = append(columns, columnName(candidate.col))
		if candidate.access {
			access = append(access, columnName(candidate.col))
		}
	}

	return columns, access
}

func (s *Service) recordChange(
	ctx context.Context,
	actor *Actor,
	change *permissionports.SecurityChange,
) {
	metadata := actor.Metadata()
	maps.Copy(metadata, change.Metadata)

	change.Actor = permissionports.SystemAuditActor()
	change.OrganizationID = actor.OrganizationID
	change.BusinessUnitID = actor.BusinessUnitID
	change.Metadata = metadata
	s.auditor.RecordChange(ctx, change)
}

func (s *Service) recordCreation(ctx context.Context, actor *Actor, user *tenant.User) {
	s.recordChange(ctx, actor, &permissionports.SecurityChange{
		Resource:   permission.ResourceUser,
		ResourceID: user.ID.String(),
		Operation:  permission.OpCreate,
		After: map[string]any{
			"user":            user,
			"organizationIds": []string{actor.OrganizationID.String()},
		},
		Comment:  "User provisioned by directory",
		Metadata: map[string]any{"emailAddress": user.EmailAddress},
	})
}

func (s *Service) recordAccessChange(
	ctx context.Context,
	actor *Actor,
	before, after *tenant.User,
	columns []string,
) {
	s.recordChange(ctx, actor, &permissionports.SecurityChange{
		Resource:   permission.ResourceUser,
		ResourceID: after.ID.String(),
		Operation:  permission.OpUpdate,
		Before:     accessState(before),
		After:      accessState(after),
		Comment:    "User sign-in access changed",
		Metadata:   map[string]any{"changedColumns": columns},
	})
}

func accessState(user *tenant.User) map[string]any {
	return map[string]any{
		"status":       user.Status,
		"emailAddress": user.EmailAddress,
		"isLocked":     user.IsLocked,
	}
}

func (s *Service) publish(ctx context.Context, actor *Actor, user *tenant.User, action string) {
	s.lifecycle.PublishUserChange(ctx, actor.TenantInfo(), user, action)
}
