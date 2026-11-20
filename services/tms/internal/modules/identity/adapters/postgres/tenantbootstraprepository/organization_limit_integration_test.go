//go:build integration

package tenantbootstraprepository

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/organizationmeter"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedusageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type heldState struct {
	state *managedusage.State
}

func (h *heldState) Get(context.Context, managedscope.Scope) (*managedusage.State, error) {
	if h.state == nil {
		return nil, managedusage.ErrNoState
	}
	return h.state, nil
}

func (h *heldState) Save(_ context.Context, _ managedscope.Scope, state *managedusage.State) error {
	h.state = state
	return nil
}

func organizationsAllowed(allowance int64) *managedusage.State {
	return &managedusage.State{
		WindowStart: 1,
		WindowEnd:   time.Now().Add(time.Hour).Unix(),
		Overage:     true,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterOrganizationsTotal,
			Unit:      managedusage.UnitCount,
			Allowance: allowance,
			HardCap:   true,
		}},
	}
}

type limitRig struct {
	ctx      context.Context
	conn     *postgres.Connection
	repo     tenantports.TenantBootstrapRepository
	stateID  pulid.ID
	existing int64
}

func newLimitRig(t *testing.T, state *managedusage.State) *limitRig {
	t.Helper()
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.WrapDB(db)
	organizations := organizationmeter.New(organizationmeter.Params{DB: conn}).Reader
	gate := managedusageservice.New(managedusageservice.Params{
		Repo:   &heldState{state: state},
		Scopes: managedscope.NewResolver(false),
		Levels: []managedusageports.LevelReader{organizations},
		Clock:  func() int64 { return time.Now().Unix() },
		Logger: zap.NewNop(),
	})
	sysCtx := dbscope.WithSystem(ctx, "count the organizations a test starts with")
	existing, err := tenantdb.Organizations.AllTenants().Count(sysCtx, db)
	require.NoError(t, err)

	return &limitRig{
		ctx:  ctx,
		conn: conn,
		repo: New(Params{
			DB:     conn,
			Logger: zap.NewNop(),
			Quota: quotaservice.NewManaged(
				gate,
				managedscope.NewResolver(false),
				func() int64 { return time.Now().Unix() },
			),
		}),
		stateID:  data.State.ID,
		existing: existing,
	}
}

func (r *limitRig) bootstrap(t *testing.T, slug string) error {
	t.Helper()
	org := newOrganization("Second Freight " + slug)
	org.StateID = r.stateID
	sysCtx := dbscope.WithSystem(r.ctx, "bootstrap a tenant the way instance bootstrap does")
	return r.conn.Transact(sysCtx, kernelports.TxOptions{}, func(ctx context.Context) error {
		_, err := r.repo.Bootstrap(ctx, &tenantports.BootstrapTenantRequest{
			BusinessUnitName: "Second Freight " + slug,
			Organization:     org,
			LoginSlugBase:    "second-freight-" + slug,
			Owner:            newOwner(slug + "@second.example"),
			UsernameBase:     slug,
			Now:              time.Now().Unix(),
		})
		return err
	})
}

func TestCreatingAnOrganizationBelowThePlanAllowanceIsAllowed(t *testing.T) {
	state := organizationsAllowed(0)
	rig := newLimitRig(t, state)
	state.Meters[0].Allowance = rig.existing + 1

	require.NoError(t, rig.bootstrap(t, "below"))
}

func TestCreatingAnOrganizationPastThePlanAllowanceIsRefusedWithThePlanWording(t *testing.T) {
	state := organizationsAllowed(0)
	rig := newLimitRig(t, state)
	state.Meters[0].Allowance = rig.existing

	err := rig.bootstrap(t, "past")

	var quota *errortypes.QuotaExceededError
	require.ErrorAs(t, err, &quota)
	assert.Equal(t, string(platformcatalog.MeterOrganizationsTotal), quota.Meter)
	assert.Equal(t, rig.existing, quota.Limit)
	assert.Contains(t, quota.Error(), "Your plan includes")
}

func TestAnInstanceWithNoUsageStateCreatesOrganizationsFreely(t *testing.T) {
	rig := newLimitRig(t, nil)

	require.NoError(t, rig.bootstrap(t, "nostate"))
}
