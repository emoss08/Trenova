//go:build integration

package tenantbootstraprepository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func newOrganization(name string) *tenant.Organization {
	return &tenant.Organization{
		Name:                   name,
		ScacCode:               "TBDX",
		DOTNumber:              "0",
		City:                   "Pending",
		PostalCode:             "00000",
		Timezone:               "America/New_York",
		Locale:                 "en",
		BrokerageEnabled:       true,
		AssetOperationsEnabled: true,
	}
}

func newOwner(email string) *tenant.User {
	return &tenant.User{
		Name:         "Dana Whitfield",
		EmailAddress: email,
		Password:     "$2a$10$abcdefghijklmnopqrstuuR0Z5o4Ue9Jm6dEoXlqkY8kZf3UtcV2q",
		Status:       domaintypes.StatusActive,
		Timezone:     "America/New_York",
		Locale:       "en",
		TimeFormat:   domaintypes.TimeFormat12Hour,
	}
}

func count(t *testing.T, ctx context.Context, db bun.IDB, model any, orgID any) int {
	t.Helper()
	n, err := db.NewSelect().Model(model).Where("organization_id = ?", orgID).Count(ctx)
	require.NoError(t, err)
	return n
}

func TestBootstrapCreatesEverythingATenantNeeds(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{DB: conn, Logger: zap.NewNop()})

	org := newOrganization("Acme Freight, LLC")
	org.StateID = data.State.ID

	var result *repositories.BootstrapTenantResult
	err := conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		require.NoError(t, repo.LockProvisioning(ctx))
		var bootErr error
		result, bootErr = repo.Bootstrap(ctx, &repositories.BootstrapTenantRequest{
			BusinessUnitName: "Acme Freight, LLC",
			Organization:     org,
			LoginSlugBase:    "acme-freight-llc",
			Owner:            newOwner("dana@acme.example"),
			UsernameBase:     "dana",
			Now:              time.Now().Unix(),
		})
		return bootErr
	})
	require.NoError(t, err)

	assert.Equal(t, "acme-freight-llc", result.Organization.LoginSlug)
	assert.Equal(t, "acme-freight-llc", result.Organization.BucketName)
	assert.True(t, strings.HasPrefix(result.BusinessUnit.Code, "CL"))
	assert.Equal(t, "dana", result.Owner.Username)
	assert.Equal(t, result.Organization.ID, result.Owner.CurrentOrganizationID)

	orgID := result.Organization.ID
	assert.Equal(t, 1, count(t, ctx, db, (*tenant.AccountingControl)(nil), orgID))
	assert.Equal(t, 1, count(t, ctx, db, (*tenant.ShipmentControl)(nil), orgID))
	assert.Len(t, tenantbootstrap.DefaultSequenceTypes(), count(t, ctx, db, (*tenant.Sequence)(nil), orgID))
	assert.Equal(t, 1, count(t, ctx, db, (*tenant.OrganizationMembership)(nil), orgID))
	assert.Equal(t, 1, count(t, ctx, db, (*permission.UserRoleAssignment)(nil), orgID))
	assert.Equal(t, 6, count(t, ctx, db, (*accounttype.AccountType)(nil), orgID))
	assert.Positive(t, count(t, ctx, db, (*documenttype.DocumentType)(nil), orgID))
	assert.Positive(t, count(t, ctx, db, (*servicefailure.ReasonCode)(nil), orgID))
	assert.Len(t, documenttemplate.AllKinds(), count(t, ctx, db, (*documenttemplate.DocumentTemplate)(nil), orgID))
	assert.Equal(t, 2, count(t, ctx, db, (*agentdefinition.Definition)(nil), orgID))

	second := newOrganization("Acme Freight, LLC")
	second.StateID = data.State.ID
	again, err := repo.Bootstrap(ctx, &repositories.BootstrapTenantRequest{
		BusinessUnitName: "Acme Freight, LLC",
		Organization:     second,
		LoginSlugBase:    "acme-freight-llc",
		Owner:            newOwner("ops@acme.example"),
		UsernameBase:     "dana",
	})
	require.NoError(t, err)
	assert.Equal(t, "acme-freight-llc-2", again.Organization.LoginSlug)
	assert.Equal(t, "dana-2", again.Owner.Username)
	assert.NotEqual(t, result.BusinessUnit.ID, again.BusinessUnit.ID)
}

func TestBootstrapResolvesThePlaceholderState(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	result, err := repo.Bootstrap(ctx, &repositories.BootstrapTenantRequest{
		BusinessUnitName:  "Beta Haulers",
		Organization:      newOrganization("Beta Haulers"),
		LoginSlugBase:     "beta-haulers",
		StateAbbreviation: data.State.Abbreviation,
		Owner:             newOwner("beta@haulers.example"),
		UsernameBase:      "beta",
	})
	require.NoError(t, err)
	assert.Equal(t, data.State.ID, result.Organization.StateID)

	_, err = repo.Bootstrap(ctx, &repositories.BootstrapTenantRequest{
		BusinessUnitName:  "Gamma",
		Organization:      newOrganization("Gamma"),
		StateAbbreviation: "ZZ",
		Owner:             newOwner("gamma@example.com"),
	})
	require.Error(t, err)
}
