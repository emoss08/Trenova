package chainrun

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/testutil/apichain"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
)

const switchOrganizationPath = "/api/v1/users/me/switch-organization/"

type Actor struct {
	Email          string
	UserID         string
	OrganizationID string
	BusinessUnitID string
}

func resolveActor(ctx context.Context, db *keel.DB, email string) (Actor, error) {
	actor := Actor{Email: email}
	var err error
	if actor.UserID, err = keel.QueryValue[string](ctx, db,
		`SELECT id FROM users WHERE email_address = $1`, email); err != nil {
		return Actor{}, fmt.Errorf("find the fixture user %s: %w", email, err)
	}
	if actor.OrganizationID, err = keel.QueryValue[string](ctx, db,
		`SELECT current_organization_id FROM users WHERE id = $1`, actor.UserID); err != nil {
		return Actor{}, fmt.Errorf("read the organization of %s: %w", email, err)
	}
	if actor.BusinessUnitID, err = keel.QueryValue[string](ctx, db,
		`SELECT business_unit_id FROM users WHERE id = $1`, actor.UserID); err != nil {
		return Actor{}, fmt.Errorf("read the business unit of %s: %w", email, err)
	}

	return actor, nil
}

func resolveFixtures(
	ctx context.Context,
	db *keel.DB,
	actor Actor,
	fixtures map[string]string,
) (map[string]any, error) {
	out := make(map[string]any, len(fixtures))
	for name, query := range fixtures {
		value, err := keel.QueryValue[string](
			ctx, db, query, actor.OrganizationID, actor.BusinessUnitID,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"fixture %s: the query must return exactly one value for the actor's "+
					"organization ($1) and business unit ($2): %w", name, err,
			)
		}
		out[name] = value
	}

	return out, nil
}

func otherOrganization(ctx context.Context, db *keel.DB, actor Actor) (string, error) {
	other, err := keel.QueryValue[string](ctx, db, `SELECT organization_id
		FROM user_organization_memberships
		WHERE user_id = $1 AND organization_id <> $2
		ORDER BY organization_id
		LIMIT 1`, actor.UserID, actor.OrganizationID)
	if err != nil {
		return "", fmt.Errorf(
			"the other-tenant probe needs %s to belong to a second organization: %w",
			actor.Email, err,
		)
	}

	return other, nil
}

func (s *Session) SwitchOrganization(ctx context.Context, organizationID string) error {
	result, err := s.REST(ctx, http.MethodPost, switchOrganizationPath,
		map[string]any{"organizationId": organizationID})
	if err != nil {
		return err
	}
	if result.Failed() {
		return fmt.Errorf("switch to %s: %d %s", organizationID, result.Status, describe(result.Body))
	}

	return s.refreshCSRF(ctx)
}

func noPermissionUser(t *testing.T, db *keel.DB, actor Actor) string {
	t.Helper()

	org := pulid.ID(actor.OrganizationID)
	bu := pulid.ID(actor.BusinessUnitID)
	ctx := dbscope.WithTenant(t.Context(), dbscope.Tenant{OrganizationID: org, BusinessUnitID: bu})
	handle := "noperm_" + strings.ToLower(pulid.MustNew("").String()[16:])
	user := seedtest.NewUser(org, bu).
		WithName("Chain No Permission").
		WithUsername(handle).
		WithEmail(handle+"@example.com").
		WithPassword(FixturesPass).
		Build(t, ctx, db)

	err := tenantdb.OrganizationMemberships.Create(ctx, db, &tenant.OrganizationMembership{
		BusinessUnitID: bu,
		UserID:         user.ID,
		OrganizationID: org,
		JoinedAt:       timeutils.NowUnix(),
		IsDefault:      true,
	})
	require.NoError(t, err, "add the no-permission user to the actor's organization")

	return user.EmailAddress
}

func runValues(actor Actor, unique string, now int64) map[string]any {
	return map[string]any{
		apichain.RunUnique:       unique,
		apichain.RunNow:          now,
		apichain.RunOrganization: actor.OrganizationID,
		apichain.RunBusinessUnit: actor.BusinessUnitID,
		apichain.RunUser:         actor.UserID,
	}
}
