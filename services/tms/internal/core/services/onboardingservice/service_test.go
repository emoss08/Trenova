package onboardingservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type fakeDB struct{}

func (fakeDB) DB() *bun.DB { return nil }

func (fakeDB) DBForContext(context.Context) bun.IDB { return nil }

func (fakeDB) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

func (fakeDB) HealthCheck(context.Context) error { return nil }

func (fakeDB) IsHealthy(context.Context) bool { return true }

func (fakeDB) Close() error { return nil }

type fakeOrganizations struct {
	org       *tenant.Organization
	updated   *tenant.Organization
	updateErr error
}

func (f *fakeOrganizations) GetByID(
	context.Context,
	repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	copied := *f.org
	return &copied, nil
}

func (f *fakeOrganizations) Update(
	_ context.Context,
	entity *tenant.Organization,
) (*tenant.Organization, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	f.updated = entity
	return entity, nil
}

type fakeSampleData struct {
	calls []*SampleDataRequest
	err   error
}

func (f *fakeSampleData) Load(_ context.Context, req *SampleDataRequest) error {
	f.calls = append(f.calls, req)
	return f.err
}

type deps struct {
	repo   *mocks.MockOnboardingRepository
	orgs   *fakeOrganizations
	sample *fakeSampleData
	svc    *Service
	tenant pagination.TenantInfo
}

func setup(t *testing.T) *deps {
	t.Helper()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	d := &deps{
		repo: mocks.NewMockOnboardingRepository(t),
		orgs: &fakeOrganizations{org: &tenant.Organization{
			ID:                     tenantInfo.OrgID,
			BusinessUnitID:         tenantInfo.BuID,
			Name:                   "Acme Freight, LLC",
			ScacCode:               onboarding.PlaceholderSCAC,
			DOTNumber:              onboarding.PlaceholderDOTNumber,
			City:                   onboarding.PlaceholderCity,
			PostalCode:             onboarding.PlaceholderPostalCode,
			StateID:                pulid.MustNew("us_"),
			Timezone:               "America/New_York",
			BrokerageEnabled:       true,
			AssetOperationsEnabled: true,
			Version:                3,
		}},
		sample: &fakeSampleData{},
		tenant: tenantInfo,
	}
	d.svc = &Service{
		db:            fakeDB{},
		repo:          d.repo,
		organizations: d.orgs,
		sampleData:    d.sample,
		l:             zap.NewNop(),
	}

	return d
}

func pending(info pagination.TenantInfo) *onboarding.Onboarding {
	entity := onboarding.NewPending(info.OrgID, info.BuID)
	entity.ID = pulid.MustNew("oonb_")
	entity.Version = 1
	return entity
}

func validRequest(info pagination.TenantInfo) *services.CompleteOnboardingRequest {
	return &services.CompleteOnboardingRequest{
		TenantInfo: info,
		Actor:      &services.RequestActor{UserID: info.UserID},
		Organization: services.OnboardingOrganization{
			Name:         "  Acme   Freight ",
			Timezone:     "America/Chicago",
			AddressLine1: "1 Main St",
			City:         "Dallas",
			StateID:      pulid.MustNew("us_"),
			PostalCode:   "75201",
			ScacCode:     "acme",
		},
		OperationType:  tenant.OperationTypeBrokerage,
		LoadSampleData: true,
	}
}

func TestGetWithoutARowIsNotRequired(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.repo.EXPECT().Get(mock.Anything, repositories.GetOnboardingRequest{TenantInfo: d.tenant}).
		Return(nil, errortypes.NewNotFoundError("Onboarding not found"))

	state, err := d.svc.Get(t.Context(), d.tenant)
	require.NoError(t, err)
	assert.False(t, state.Required)
}

func TestGetPendingHidesPlaceholders(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.repo.EXPECT().Get(mock.Anything, mock.Anything).Return(pending(d.tenant), nil)

	state, err := d.svc.Get(t.Context(), d.tenant)
	require.NoError(t, err)
	assert.True(t, state.Required)
	assert.Equal(t, onboarding.StatusPending, state.Status)
	require.NotNil(t, state.Organization)
	assert.Equal(t, "Acme Freight, LLC", state.Organization.Name)
	assert.Equal(t, "America/New_York", state.Organization.Timezone)
	assert.Empty(t, state.Organization.City)
	assert.Empty(t, state.Organization.PostalCode)
	assert.Empty(t, state.Organization.ScacCode)
	assert.Empty(t, state.Organization.DOTNumber)
	assert.True(t, state.Organization.StateID.IsNil())
}

func TestGetCompletedShowsTheProfile(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.orgs.org.ScacCode = "ACME"
	d.orgs.org.City = "Dallas"
	entity := pending(d.tenant)
	entity.Complete(onboarding.CompleteParams{
		UserID:        d.tenant.UserID,
		OperationType: tenant.OperationTypeAsset,
	})
	d.repo.EXPECT().Get(mock.Anything, mock.Anything).Return(entity, nil)

	state, err := d.svc.Get(t.Context(), d.tenant)
	require.NoError(t, err)
	assert.False(t, state.Required)
	assert.Equal(t, tenant.OperationTypeAsset, state.OperationType)
	assert.Equal(t, "ACME", state.Organization.ScacCode)
	assert.Empty(t, state.Organization.DOTNumber)
	assert.Equal(t, "Dallas", state.Organization.City)
	assert.NotNil(t, state.CompletedAt)
}

func TestCompleteUpdatesTheOrganizationAndLoadsSamples(t *testing.T) {
	t.Parallel()

	d := setup(t)
	entity := pending(d.tenant)
	d.repo.EXPECT().Get(mock.Anything, mock.Anything).Return(entity, nil)
	d.repo.EXPECT().Complete(mock.Anything, mock.MatchedBy(func(o *onboarding.Onboarding) bool {
		return o.Status == onboarding.StatusCompleted &&
			o.OperationType == tenant.OperationTypeBrokerage &&
			o.SampleDataLoaded && o.CompletedByID == d.tenant.UserID && o.Version == 1
	})).RunAndReturn(func(_ context.Context, o *onboarding.Onboarding) (*onboarding.Onboarding, error) {
		return o, nil
	})

	req := validRequest(d.tenant)
	state, err := d.svc.Complete(t.Context(), req)
	require.NoError(t, err)

	updated := d.orgs.updated
	require.NotNil(t, updated)
	assert.Equal(t, "Acme Freight", updated.Name)
	assert.Equal(t, "America/Chicago", updated.Timezone)
	assert.Equal(t, "Dallas", updated.City)
	assert.Equal(t, "75201", updated.PostalCode)
	assert.Equal(t, req.Organization.StateID, updated.StateID)
	assert.Equal(t, "ACME", updated.ScacCode)
	assert.Equal(t, onboarding.PlaceholderDOTNumber, updated.DOTNumber)
	assert.True(t, updated.BrokerageEnabled)
	assert.False(t, updated.AssetOperationsEnabled)
	assert.Equal(t, int64(3), updated.Version)

	require.Len(t, d.sample.calls, 1)
	assert.Equal(t, updated, d.sample.calls[0].Organization)
	assert.Equal(t, d.tenant, d.sample.calls[0].TenantInfo)

	assert.False(t, state.Required)
	assert.Equal(t, onboarding.StatusCompleted, state.Status)
	assert.True(t, state.SampleDataLoaded)
	assert.Equal(t, "ACME", state.Organization.ScacCode)
}

func TestCompleteWithoutSampleData(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.repo.EXPECT().Get(mock.Anything, mock.Anything).Return(pending(d.tenant), nil)
	d.repo.EXPECT().Complete(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, o *onboarding.Onboarding) (*onboarding.Onboarding, error) {
			return o, nil
		},
	)

	req := validRequest(d.tenant)
	req.LoadSampleData = false
	req.OperationType = tenant.OperationTypeBoth
	req.Organization.DOTNumber = "1234567"

	state, err := d.svc.Complete(t.Context(), req)
	require.NoError(t, err)
	assert.Empty(t, d.sample.calls)
	assert.False(t, state.SampleDataLoaded)
	assert.True(t, d.orgs.updated.BrokerageEnabled)
	assert.True(t, d.orgs.updated.AssetOperationsEnabled)
	assert.Equal(t, "1234567", d.orgs.updated.DOTNumber)
}

func TestCompleteValidation(t *testing.T) {
	t.Parallel()

	d := setup(t)

	req := &services.CompleteOnboardingRequest{
		TenantInfo: d.tenant,
		Organization: services.OnboardingOrganization{
			PostalCode: "7520",
			ScacCode:   "A1",
			DOTNumber:  "12ab",
		},
		OperationType: "fleet",
	}

	_, err := d.svc.Complete(t.Context(), req)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	assert.ElementsMatch(t, []string{
		"operationType",
		"organization.name",
		"organization.timezone",
		"organization.addressLine1",
		"organization.city",
		"organization.stateId",
		"organization.postalCode",
		"organization.scacCode",
		"organization.dotNumber",
	}, fields)
}

func TestCompletePrefixesOrganizationValidation(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.repo.EXPECT().Get(mock.Anything, mock.Anything).Return(pending(d.tenant), nil)
	orgErr := errortypes.NewMultiError()
	orgErr.Add("scacCode", errortypes.ErrInvalidLength, "SCAC code must be 4 characters")
	d.orgs.updateErr = orgErr

	_, err := d.svc.Complete(t.Context(), validRequest(d.tenant))
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "organization.scacCode", multiErr.Errors[0].Field)
	assert.Empty(t, d.sample.calls)
}

func TestCompleteSurfacesSampleDataFailures(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.repo.EXPECT().Get(mock.Anything, mock.Anything).Return(pending(d.tenant), nil)
	d.sample.err = errortypes.NewQuotaExceededError("shipments.total", 12, 12, "free_demo")

	_, err := d.svc.Complete(t.Context(), validRequest(d.tenant))
	require.True(t, errortypes.IsQuotaExceededError(err))
}

func TestCompleteTwiceIsAConflict(t *testing.T) {
	t.Parallel()

	d := setup(t)
	entity := pending(d.tenant)
	entity.Complete(
		onboarding.CompleteParams{
			UserID:        d.tenant.UserID,
			OperationType: tenant.OperationTypeAsset,
		},
	)
	d.repo.EXPECT().Get(mock.Anything, mock.Anything).Return(entity, nil)

	_, err := d.svc.Complete(t.Context(), validRequest(d.tenant))
	require.True(t, errortypes.IsConflictError(err))
}

func TestCompleteMissingRow(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.repo.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, errortypes.NewNotFoundError("x"))

	_, err := d.svc.Complete(t.Context(), validRequest(d.tenant))
	require.True(t, errortypes.IsBusinessError(err))

	d2 := setup(t)
	d2.repo.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, errors.New("db down"))
	_, err = d2.svc.Complete(t.Context(), validRequest(d2.tenant))
	require.EqualError(t, err, "db down")
}
