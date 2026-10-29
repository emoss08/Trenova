package accesssettingsservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/permchangetest"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memoryRepo struct {
	settings map[pagination.TenantInfo]*iam.AccessSettings
	reads    int
}

func (r *memoryRepo) Get(
	_ context.Context,
	tenant pagination.TenantInfo,
) (*iam.AccessSettings, error) {
	r.reads++
	if s, ok := r.settings[tenant]; ok {
		copied := *s
		return &copied, nil
	}
	return iam.DefaultAccessSettings(tenant.OrgID, tenant.BuID), nil
}

func (r *memoryRepo) Save(_ context.Context, s *iam.AccessSettings) (*iam.AccessSettings, error) {
	s.Version++
	copied := *s
	r.settings[pagination.TenantInfo{OrgID: s.OrganizationID, BuID: s.BusinessUnitID}] = &copied
	return s, nil
}

type constraints struct {
	permissionports.RBACRepository
	dynamic []*rbac.RoleConstraint
}

func (c *constraints) ListRoleConstraints(
	context.Context,
	permissionports.ListRoleConstraintsRequest,
) ([]*rbac.RoleConstraint, error) {
	return c.dynamic, nil
}

type fixture struct {
	svc      *Service
	repo     *memoryRepo
	rbac     *constraints
	audit    *securityaudittest.Recorder
	notifier *permchangetest.Recorder
	tenant   pagination.TenantInfo
	actor    *permissionports.RequestActor
}

func newFixture() *fixture {
	f := &fixture{
		repo:     &memoryRepo{settings: map[pagination.TenantInfo]*iam.AccessSettings{}},
		rbac:     &constraints{},
		audit:    &securityaudittest.Recorder{},
		notifier: &permchangetest.Recorder{},
		tenant:   pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
	f.actor = &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
	}
	f.svc = New(Params{
		Logger:   zap.NewNop(),
		Repo:     f.repo,
		RBAC:     f.rbac,
		Auditor:  f.audit,
		Notifier: f.notifier,
	})
	return f
}

func TestRoleActivationRequired_IsOffUntilAnOrganizationTurnsItOn(t *testing.T) {
	t.Parallel()

	f := newFixture()

	required, err := f.svc.RoleActivationRequired(t.Context(), f.tenant)

	require.NoError(t, err)
	assert.False(t, required)
}

func TestRoleActivationRequired_ReadsTheSettingEachTime(t *testing.T) {
	t.Parallel()

	f := newFixture()
	for range 3 {
		_, err := f.svc.RoleActivationRequired(t.Context(), f.tenant)
		require.NoError(t, err)
	}

	assert.Equal(
		t,
		3,
		f.repo.reads,
		"the repository's result cache answers repeats, not the service",
	)
}

func TestUpdate_TurningActivationOnTakesEffectAndIsAudited(t *testing.T) {
	t.Parallel()

	f := newFixture()
	_, err := f.svc.RoleActivationRequired(t.Context(), f.tenant)
	require.NoError(t, err)

	saved, err := f.svc.Update(t.Context(), &iamports.UpdateAccessSettingsRequest{
		TenantInfo:            f.tenant,
		Actor:                 f.actor,
		RequireRoleActivation: true,
	})
	require.NoError(t, err)
	assert.True(t, saved.RequireRoleActivation)

	required, err := f.svc.RoleActivationRequired(t.Context(), f.tenant)
	require.NoError(t, err)
	assert.True(t, required, "the change takes effect at once")
	assert.Len(t, f.audit.Changes(), 1)
	assert.Len(t, f.notifier.Organizations(), 1)
}

func TestUpdate_KeepsActivationWhileDynamicSeparationNeedsIt(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.repo.settings[f.tenant] = &iam.AccessSettings{
		OrganizationID:        f.tenant.OrgID,
		BusinessUnitID:        f.tenant.BuID,
		RequireRoleActivation: true,
	}
	f.rbac.dynamic = []*rbac.RoleConstraint{{Type: rbac.RoleConstraintTypeDSD}}

	_, err := f.svc.Update(t.Context(), &iamports.UpdateAccessSettingsRequest{
		TenantInfo:            f.tenant,
		Actor:                 f.actor,
		RequireRoleActivation: false,
	})

	require.Error(t, err)
	assert.True(t, f.repo.settings[f.tenant].RequireRoleActivation)
	assert.Empty(t, f.audit.Changes())
}

func TestUpdate_SavingTheSameSettingRecordsNothing(t *testing.T) {
	t.Parallel()

	f := newFixture()

	_, err := f.svc.Update(t.Context(), &iamports.UpdateAccessSettingsRequest{
		TenantInfo: f.tenant,
		Actor:      f.actor,
	})

	require.NoError(t, err)
	assert.Empty(t, f.audit.Changes())
	assert.Empty(t, f.notifier.Organizations())
}
