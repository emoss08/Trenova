package inboundfixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/triagefixtures"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	seedTeammatePassword = "admin123!" // #nosec G101 -- the published development login.
)

type seedTeammateSpec struct {
	name     string
	username string
	email    string
}

var seedTeammateSpecs = []seedTeammateSpec{
	{name: "Dana Whitfield", username: "dana.whitfield", email: "dana.whitfield@trenova.app"},
	{name: "Marcus Lee", username: "marcus.lee", email: "marcus.lee@trenova.app"},
	{name: "Priya Shah", username: "priya.shah", email: "priya.shah@trenova.app"},
}

type seedTeammateParams struct {
	org   *tenant.Organization
	admin *tenant.User
	role  *rbac.Role
	spec  seedTeammateSpec
}

func seedTeammates(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
	admin *tenant.User,
) (triagefixtures.Team, error) {
	role, err := administratorRole(ctx, db, org)
	if err != nil {
		return triagefixtures.Team{}, err
	}

	users := make([]*tenant.User, 0, len(seedTeammateSpecs))
	for _, spec := range seedTeammateSpecs {
		user, userErr := seedTeammate(ctx, db, seedTeammateParams{
			org:   org,
			admin: admin,
			role:  role,
			spec:  spec,
		})
		if userErr != nil {
			return triagefixtures.Team{}, userErr
		}
		users = append(users, user)
	}

	return triagefixtures.Team{Dana: users[0], Marcus: users[1], Priya: users[2]}, nil
}

func administratorRole(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
) (*rbac.Role, error) {
	r := permissiondb.Roles
	role, err := r.AllTenants().Where(
		r.OrganizationID.Eq(org.ID),
		r.BusinessUnitID.Eq(org.BusinessUnitID),
		r.Name.Eq(tenantbootstrap.AdministratorRoleName),
	).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("find the administrator role: %w", err)
	}

	return role, nil
}

func seedTeammate(ctx context.Context, db *keel.DB, p seedTeammateParams) (*tenant.User, error) {
	now := timeutils.NowUnix()
	user := &tenant.User{
		CurrentOrganizationID: p.org.ID,
		BusinessUnitID:        p.org.BusinessUnitID,
		Name:                  p.spec.name,
		Username:              p.spec.username,
		EmailAddress:          p.spec.email,
		Status:                domaintypes.StatusActive,
		Timezone:              p.org.Timezone,
		CreatedAt:             now,
		UpdatedAt:             now,
		TimeFormat:            domaintypes.TimeFormat12Hour,
		Locale:                "en",
		AvatarStyle:           tenant.AvatarStyleAuto,
		AvailabilityStatus:    tenant.AvailabilityAvailable,
	}
	hashed, err := user.GeneratePassword(seedTeammatePassword)
	if err != nil {
		return nil, fmt.Errorf("hash the password of %s: %w", p.spec.username, err)
	}
	user.Password = hashed
	if err = tenantdb.Users.Create(ctx, db, user); err != nil {
		return nil, fmt.Errorf("create %s: %w", p.spec.username, err)
	}
	membership := &tenant.OrganizationMembership{
		BusinessUnitID: p.org.BusinessUnitID,
		OrganizationID: p.org.ID,
		UserID:         user.ID,
		JoinedAt:       now,
		GrantedByID:    p.admin.ID,
		IsDefault:      true,
	}
	if err = tenantdb.OrganizationMemberships.Create(ctx, db, membership); err != nil {
		return nil, fmt.Errorf("add %s to the organization: %w", p.spec.username, err)
	}

	if err = tenantbootstrap.AssignRole(ctx, db, tenantbootstrap.RoleAssignmentParams{
		UserID:         user.ID,
		OrganizationID: p.org.ID,
		RoleID:         p.role.ID,
		AssignedBy:     p.admin.ID,
		AssignedAt:     now,
	}); err != nil {
		return nil, err
	}

	return user, nil
}

func matchedBy(rule *triage.Rule, message *inboundmessage.InboundMessage) bool {
	if rule.Action == inboundmessage.NovaActionAssign {
		return rule.Covers(message) && message.OwnerID == rule.AssigneeID
	}

	return message.HandledByRuleID == rule.ID
}

func markRuleMatches(
	ctx context.Context,
	db *keel.DB,
	rules []*triage.Rule,
	messages []triagefixtures.Message,
) error {
	for _, rule := range rules {
		var latest *inboundmessage.InboundMessage
		for _, seeded := range messages {
			message := seeded.Message
			if !matchedBy(rule, message) {
				continue
			}
			if latest == nil || message.ReceivedAt > latest.ReceivedAt {
				latest = message
			}
		}
		if latest == nil {
			continue
		}

		matchedAt := latest.ReceivedAt + triagefixtures.RuleDelay
		lastMessageID := latest.ID
		rule.LastMatchedAt = matchedAt
		rule.LastMessageID = lastMessageID
		r := triagedb.Rules
		if err := r.Save(ctx, db, rule, r.LastMatchedAt, r.LastMessageID); err != nil {
			return fmt.Errorf("mark rule %s matched: %w", rule.ID, err)
		}
	}

	return nil
}
