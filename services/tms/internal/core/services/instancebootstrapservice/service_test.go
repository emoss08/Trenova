package instancebootstrapservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	testPassword           = "Tr4ck-the-l0ads-north!"
	testSystemUserPassword = "system-user-password"
	testNow                = int64(1_790_000_000)
)

type fakeRepo struct {
	state        *repositories.InstanceBootstrapState
	stateErr     error
	finalizeReq  *repositories.FinalizeInstanceBootstrapRequest
	finalizeErr  error
	systemCreate bool
	getCalls     int
}

func (f *fakeRepo) GetState(context.Context) (*repositories.InstanceBootstrapState, error) {
	f.getCalls++
	if f.stateErr != nil {
		return nil, f.stateErr
	}
	return f.state, nil
}

func (f *fakeRepo) Finalize(
	_ context.Context,
	req *repositories.FinalizeInstanceBootstrapRequest,
) (*repositories.FinalizeInstanceBootstrapResult, error) {
	f.finalizeReq = req
	if f.finalizeErr != nil {
		return nil, f.finalizeErr
	}

	return &repositories.FinalizeInstanceBootstrapResult{
		Record: &instancebootstrap.InstanceBootstrap{
			ID:             pulid.MustNew(instancebootstrap.IDPrefix),
			OrganizationID: req.OrganizationID,
			BusinessUnitID: req.BusinessUnitID,
			AdminUserID:    req.AdminUserID,
			Inputs:         req.Inputs,
			CompletedAt:    req.Now,
		},
		SystemUserID:      pulid.MustNew("usr_"),
		SystemUserCreated: f.systemCreate,
	}, nil
}

type deps struct {
	repo    *fakeRepo
	tenants *mocks.MockTenantBootstrapRepository
	auditor *securityaudittest.Recorder
	svc     *Service
}

func newDeps(t *testing.T, state *repositories.InstanceBootstrapState) *deps {
	t.Helper()

	d := &deps{
		repo:    &fakeRepo{state: state, systemCreate: true},
		tenants: mocks.NewMockTenantBootstrapRepository(t),
		auditor: &securityaudittest.Recorder{},
	}
	d.svc = New(Params{
		DB:      dbtest.NopConnection{},
		Tenants: d.tenants,
		Repo:    d.repo,
		Auditor: d.auditor,
		Config: &config.Config{
			System: config.SystemConfig{SystemUserPassword: testSystemUserPassword},
		},
		Logger: zap.NewNop(),
	})
	d.svc.now = func() int64 { return testNow }

	return d
}

func emptyState() *repositories.InstanceBootstrapState {
	return &repositories.InstanceBootstrapState{}
}

func validInputs() *instancebootstrap.Inputs {
	return &instancebootstrap.Inputs{
		OrganizationName: " Acme  Freight ",
		AdminName:        "Dana Whitfield",
		AdminEmail:       "Dana.Whitfield@Acme.example",
		State:            "tx",
		City:             "Dallas",
		PostalCode:       "75201",
		SCAC:             "acmf",
		DOTNumber:        "1234567",
	}
}

func completedState() *repositories.InstanceBootstrapState {
	return &repositories.InstanceBootstrapState{
		UserCount: 4,
		Record: &instancebootstrap.InstanceBootstrap{
			ID:             pulid.MustNew(instancebootstrap.IDPrefix),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			AdminUserID:    pulid.MustNew("usr_"),
			Inputs:         validInputs().Normalize(),
			CompletedAt:    testNow - 100,
		},
	}
}

func expectBootstrap(d *deps) *repositories.BootstrapTenantRequest {
	var captured repositories.BootstrapTenantRequest
	d.tenants.EXPECT().LockProvisioning(mock.Anything).Return(nil)
	d.tenants.EXPECT().Bootstrap(mock.Anything, mock.Anything).RunAndReturn(
		func(
			_ context.Context,
			req *repositories.BootstrapTenantRequest,
		) (*repositories.BootstrapTenantResult, error) {
			captured = *req
			bu := &tenant.BusinessUnit{ID: pulid.MustNew("bu_"), Name: req.BusinessUnitName}
			org := req.Organization
			org.ID = pulid.MustNew("org_")
			org.BusinessUnitID = bu.ID
			org.LoginSlug = req.LoginSlugBase
			owner := req.Owner
			owner.ID = pulid.MustNew("usr_")
			owner.Username = req.UsernameBase
			owner.BusinessUnitID = bu.ID
			owner.CurrentOrganizationID = org.ID
			return &repositories.BootstrapTenantResult{
				BusinessUnit: bu,
				Organization: org,
				Owner:        owner,
				AdminRoleID:  pulid.MustNew("rol_"),
			}, nil
		},
	)

	return &captured
}

func TestPlanReportsReadyOnAnEmptyInstance(t *testing.T) {
	t.Parallel()

	d := newDeps(t, emptyState())
	plan, err := d.svc.Plan(t.Context(), validInputs())
	require.NoError(t, err)

	assert.Equal(t, services.InstanceBootstrapReady, plan.Status)
	assert.Equal(t, "Acme Freight", plan.Inputs.OrganizationName)
	assert.Equal(t, "dana.whitfield@acme.example", plan.Inputs.AdminEmail)
}

func TestPlanReportsCompletedForIdenticalInputs(t *testing.T) {
	t.Parallel()

	d := newDeps(t, completedState())
	inputs := validInputs()
	inputs.AdminEmail = "DANA.WHITFIELD@acme.example"
	inputs.Timezone = "America/Chicago"

	plan, err := d.svc.Plan(t.Context(), inputs)
	require.NoError(t, err)
	assert.Equal(t, services.InstanceBootstrapCompleted, plan.Status)
}

func TestPlanRefusesDifferentInputsAfterCompletion(t *testing.T) {
	t.Parallel()

	d := newDeps(t, completedState())
	inputs := validInputs()
	inputs.AdminEmail = "someone@acme.example"

	_, err := d.svc.Plan(t.Context(), inputs)
	require.ErrorIs(t, err, ErrInputsDiffer)
	assert.Contains(t, err.Error(), instancebootstrap.FieldAdminEmail)
}

func TestPlanRefusesWhenUsersAlreadyExist(t *testing.T) {
	t.Parallel()

	d := newDeps(t, &repositories.InstanceBootstrapState{UserCount: 3})
	_, err := d.svc.Plan(t.Context(), validInputs())
	require.ErrorIs(t, err, ErrAlreadyInitialized)
	assert.Contains(t, err.Error(), "3 user(s)")
}

func TestPlanValidatesInputsBeforeReadingTheDatabase(t *testing.T) {
	t.Parallel()

	d := newDeps(t, emptyState())
	_, err := d.svc.Plan(t.Context(), &instancebootstrap.Inputs{})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Zero(t, d.repo.getCalls)
}

func TestPlanPropagatesRepositoryErrors(t *testing.T) {
	t.Parallel()

	d := newDeps(t, emptyState())
	boom := errors.New("boom")
	d.repo.stateErr = boom

	_, err := d.svc.Plan(t.Context(), validInputs())
	require.ErrorIs(t, err, boom)
}

func TestBootstrapRejectsAPasswordOutsideThePolicy(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"too short":          "short-pass",
		"common":             "password1234",
		"contains the email": "dana.whitfield-2026!",
	}

	for name, password := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			d := newDeps(t, emptyState())
			_, err := d.svc.Bootstrap(t.Context(), &services.InstanceBootstrapRequest{
				Inputs:   *validInputs(),
				Password: password,
			})

			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			require.Len(t, multiErr.Errors, 1)
			assert.Equal(t, fieldPassword, multiErr.Errors[0].Field)
			assert.Zero(t, d.repo.getCalls)
		})
	}
}

func TestBootstrapRejectsANilRequest(t *testing.T) {
	t.Parallel()

	d := newDeps(t, emptyState())
	_, err := d.svc.Bootstrap(t.Context(), nil)
	require.Error(t, err)
}

func TestBootstrapCreatesTheOrganizationAndAdministrator(t *testing.T) {
	t.Parallel()

	d := newDeps(t, emptyState())
	captured := expectBootstrap(d)

	result, err := d.svc.Bootstrap(t.Context(), &services.InstanceBootstrapRequest{
		Inputs:   *validInputs(),
		Password: testPassword,
	})
	require.NoError(t, err)

	assert.Equal(t, services.InstanceBootstrapCreated, result.Status)
	assert.True(t, result.SystemUserCreated)
	assert.Equal(t, "dana.whitfield", result.AdminUsername)
	assert.Equal(t, "dana.whitfield@acme.example", result.AdminEmail)
	assert.Equal(t, "acme-freight", result.LoginSlug)
	assert.Equal(t, testNow, result.CompletedAt)

	assert.Equal(t, "Acme Freight", captured.BusinessUnitName)
	assert.Equal(t, "TX", captured.StateAbbreviation)
	assert.Equal(t, testNow, captured.Now)
	assert.Equal(t, "ACMF", captured.Organization.ScacCode)
	assert.Equal(t, "1234567", captured.Organization.DOTNumber)
	assert.Equal(t, "Dallas", captured.Organization.City)
	assert.Equal(t, "75201", captured.Organization.PostalCode)
	assert.Equal(t, "America/New_York", captured.Organization.Timezone)
	assert.False(t, captured.Owner.MustChangePassword)
	require.NoError(
		t,
		bcrypt.CompareHashAndPassword([]byte(captured.Owner.Password), []byte(testPassword)),
	)

	require.NotNil(t, d.repo.finalizeReq)
	assert.Equal(t, testSystemUserPassword, d.repo.finalizeReq.SystemUserPassword)
	assert.Equal(t, result.AdminUserID, d.repo.finalizeReq.AdminUserID)
	assert.Equal(t, validInputs().Normalize(), d.repo.finalizeReq.Inputs)

	changes := d.auditor.Changes()
	require.Len(t, changes, 4)
	assert.Equal(t, permission.ResourceUser, changes[0].Resource)
	assert.Equal(t, permission.OpCreate, changes[0].Operation)
	assert.Equal(t, result.AdminUserID.String(), changes[0].ResourceID)
	assert.Equal(t, permission.ResourceRole, changes[1].Resource)
	assert.Equal(t, permission.OpCreate, changes[1].Operation)
	assert.Equal(t, permission.ResourceRole, changes[2].Resource)
	assert.Equal(t, permission.OpAssign, changes[2].Operation)
	assert.Equal(t, permission.ResourceUser, changes[3].Resource)
	for _, change := range changes {
		assert.Equal(t, services.SystemAuditActor(), change.Actor)
		assert.Equal(t, result.OrganizationID, change.OrganizationID)
		assert.Equal(t, auditSource, change.Metadata["source"])
	}
}

func TestBootstrapSkipsTheSystemUserAuditWhenItAlreadyExisted(t *testing.T) {
	t.Parallel()

	d := newDeps(t, emptyState())
	d.repo.systemCreate = false
	expectBootstrap(d)

	result, err := d.svc.Bootstrap(t.Context(), &services.InstanceBootstrapRequest{
		Inputs:   *validInputs(),
		Password: testPassword,
	})
	require.NoError(t, err)
	assert.False(t, result.SystemUserCreated)
	assert.Len(t, d.auditor.Changes(), 3)
}

func TestBootstrapIsANoOpForIdenticalInputsAfterCompletion(t *testing.T) {
	t.Parallel()

	state := completedState()
	d := newDeps(t, state)
	d.tenants.EXPECT().LockProvisioning(mock.Anything).Return(nil)

	result, err := d.svc.Bootstrap(t.Context(), &services.InstanceBootstrapRequest{
		Inputs:   *validInputs(),
		Password: testPassword,
	})
	require.NoError(t, err)

	assert.Equal(t, services.InstanceBootstrapCompleted, result.Status)
	assert.Equal(t, state.Record.OrganizationID, result.OrganizationID)
	assert.Equal(t, state.Record.AdminUserID, result.AdminUserID)
	assert.Nil(t, d.repo.finalizeReq)
	assert.Empty(t, d.auditor.Changes())
}

func TestBootstrapRefusesUnderTheLockWhenUsersAppeared(t *testing.T) {
	t.Parallel()

	d := newDeps(t, &repositories.InstanceBootstrapState{UserCount: 1})
	d.tenants.EXPECT().LockProvisioning(mock.Anything).Return(nil)

	_, err := d.svc.Bootstrap(t.Context(), &services.InstanceBootstrapRequest{
		Inputs:   *validInputs(),
		Password: testPassword,
	})
	require.ErrorIs(t, err, ErrAlreadyInitialized)
	assert.Nil(t, d.repo.finalizeReq)
	assert.Empty(t, d.auditor.Changes())
}

func TestBootstrapRecordsNothingWhenFinalizingFails(t *testing.T) {
	t.Parallel()

	d := newDeps(t, emptyState())
	boom := errors.New("finalize failed")
	d.repo.finalizeErr = boom
	expectBootstrap(d)

	_, err := d.svc.Bootstrap(t.Context(), &services.InstanceBootstrapRequest{
		Inputs:   *validInputs(),
		Password: testPassword,
	})
	require.ErrorIs(t, err, boom)
	assert.Empty(t, d.auditor.Changes())
}

func TestUsernameBase(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "dana.whitfield", usernameBase("Dana.Whitfield@acme.example"))
	assert.Equal(t, "ops", usernameBase("+ops+@acme.example"))
	assert.Equal(t, "abcdefghijklmnopqrst", usernameBase("abcdefghijklmnopqrstuvwxyz@acme.example"))
	assert.Empty(t, usernameBase("+++@acme.example"))
}
