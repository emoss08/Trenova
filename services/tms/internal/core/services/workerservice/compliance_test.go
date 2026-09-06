package workerservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDispatchControlRepo struct {
	mock.Mock
}

func (m *mockDispatchControlRepo) GetByOrgID(
	ctx context.Context,
	req repositories.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dispatchcontrol.DispatchControl), args.Error(1)
}

func (m *mockDispatchControlRepo) Create(
	ctx context.Context,
	entity *dispatchcontrol.DispatchControl,
) (*dispatchcontrol.DispatchControl, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dispatchcontrol.DispatchControl), args.Error(1)
}

func (m *mockDispatchControlRepo) Update(
	ctx context.Context,
	entity *dispatchcontrol.DispatchControl,
) (*dispatchcontrol.DispatchControl, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dispatchcontrol.DispatchControl), args.Error(1)
}

func (m *mockDispatchControlRepo) GetOrCreate(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*dispatchcontrol.DispatchControl, error) {
	args := m.Called(ctx, orgID, buID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dispatchcontrol.DispatchControl), args.Error(1)
}

func (m *mockDispatchControlRepo) ListHorizonPlanningTenants(
	ctx context.Context,
) ([]pagination.TenantInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]pagination.TenantInfo), args.Error(1)
}

func newComplianceWorker() *worker.Worker {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	return &worker.Worker{
		ID:             pulid.MustNew("wrk_"),
		BusinessUnitID: buID,
		OrganizationID: orgID,
		Profile: &worker.WorkerProfile{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			DOB:            time.Now().AddDate(-25, 0, 0).Unix(),
			LicenseExpiry:  time.Now().AddDate(2, 0, 0).Unix(),
			HireDate:       time.Now().AddDate(-1, 0, 0).Unix(),
			Endorsement:    worker.EndorsementTypeNone,
			// A worker loaded from the database always carries a testing
			// standing; Clear is the ordinary case and keeps the unrelated
			// rules under test from tripping over a missing drug test.
			DrugAlcoholStatus: worker.DrugAlcoholClear,
		},
	}
}

func newDispatchControl() *dispatchcontrol.DispatchControl {
	return &dispatchcontrol.DispatchControl{
		ID:                                   pulid.MustNew("dc_"),
		EnforceDriverQualificationCompliance: true,
		EnforceMedicalCertCompliance:         true,
		EnforceHOSCompliance:                 true,
		EnforceDrugAndAlcoholCompliance:      true,
		EnforceHazmatCompliance:              true,
		ComplianceEnforcementLevel:           dispatchcontrol.ComplianceEnforcementLevelWarning,
	}
}

func runComplianceRule(
	t *testing.T,
	rule validationframework.TenantedRule[*worker.Worker],
	w *worker.Worker,
) *errortypes.MultiError {
	t.Helper()
	ctx := t.Context()
	multiErr := errortypes.NewMultiError()
	valCtx := &validationframework.TenantedValidationContext{
		Mode:           validationframework.ModeCreate,
		OrganizationID: w.OrganizationID,
		BusinessUnitID: w.BusinessUnitID,
		EntityID:       w.ID,
	}
	err := rule.Validate(ctx, w, valCtx, multiErr)
	require.NoError(t, err)
	return multiErr
}

func TestWorkerComplianceRuleFetchesDispatchControlOnce(t *testing.T) {
	t.Parallel()

	dcRepo := new(mockDispatchControlRepo)
	w := newComplianceWorker()
	w.Profile.DOB = time.Now().AddDate(-19, 0, 0).Unix()
	w.Profile.LicenseExpiry = time.Now().AddDate(-1, 0, 0).Unix()
	dc := newDispatchControl()
	dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil).Once()

	rule := createWorkerComplianceRule(dcRepo)
	multiErr := runComplianceRule(t, rule, w)

	require.True(t, multiErr.HasErrors())
	assert.GreaterOrEqual(t, len(multiErr.Errors), 2)
	dcRepo.AssertExpectations(t)
}

func TestAgeComplianceRule(t *testing.T) {
	t.Parallel()

	t.Run("passes for driver over 21", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.DOB = time.Now().AddDate(-25, 0, 0).Unix()
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createAgeComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails for driver under 21", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.DOB = time.Now().AddDate(-20, 0, 0).Unix()
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createAgeComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "profile.dob", multiErr.Errors[0].Field)
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes for driver exactly 21", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.DOB = time.Now().AddDate(-21, 0, -1).Unix()
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createAgeComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("skips when enforcement disabled", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.DOB = time.Now().AddDate(-18, 0, 0).Unix()
		dc := newDispatchControl()
		dc.EnforceDriverQualificationCompliance = false
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createAgeComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("skips when profile is nil", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createAgeComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("returns error when dispatch control fetch fails", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).
			Return(nil, errors.New("db error"))

		rule := createAgeComplianceRule(dcRepo)
		ctx := t.Context()
		multiErr := errortypes.NewMultiError()
		valCtx := &validationframework.TenantedValidationContext{
			Mode:           validationframework.ModeCreate,
			OrganizationID: w.OrganizationID,
			BusinessUnitID: w.BusinessUnitID,
		}
		err := rule.Validate(ctx, w, valCtx, multiErr)

		require.Error(t, err)
		dcRepo.AssertExpectations(t)
	})

	t.Run("uses compliance violation code when level is Block", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.DOB = time.Now().AddDate(-19, 0, 0).Unix()
		dc := newDispatchControl()
		dc.ComplianceEnforcementLevel = dispatchcontrol.ComplianceEnforcementLevelBlock
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createAgeComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, errortypes.ErrComplianceViolation, multiErr.Errors[0].Code)
		dcRepo.AssertExpectations(t)
	})

	t.Run("uses invalid code when level is Warning", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.DOB = time.Now().AddDate(-19, 0, 0).Unix()
		dc := newDispatchControl()
		dc.ComplianceEnforcementLevel = dispatchcontrol.ComplianceEnforcementLevelWarning
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createAgeComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
		dcRepo.AssertExpectations(t)
	})
}

func TestCDLComplianceRule(t *testing.T) {
	t.Parallel()

	t.Run("passes for valid license", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.LicenseExpiry = time.Now().AddDate(1, 0, 0).Unix()
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createCDLComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails for expired license", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.LicenseExpiry = time.Now().AddDate(-1, 0, 0).Unix()
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createCDLComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "profile.licenseExpiry", multiErr.Errors[0].Field)
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails for zero license expiry", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.LicenseExpiry = 0
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createCDLComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("skips when enforcement disabled", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.LicenseExpiry = time.Now().AddDate(-1, 0, 0).Unix()
		dc := newDispatchControl()
		dc.EnforceDriverQualificationCompliance = false
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createCDLComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("skips when profile is nil", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createCDLComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("uses compliance violation code when level is Block", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.LicenseExpiry = time.Now().AddDate(-1, 0, 0).Unix()
		dc := newDispatchControl()
		dc.ComplianceEnforcementLevel = dispatchcontrol.ComplianceEnforcementLevelBlock
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createCDLComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, errortypes.ErrComplianceViolation, multiErr.Errors[0].Code)
		dcRepo.AssertExpectations(t)
	})

	t.Run("returns error when dispatch control fetch fails", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).
			Return(nil, errors.New("db error"))

		rule := createCDLComplianceRule(dcRepo)
		ctx := t.Context()
		multiErr := errortypes.NewMultiError()
		valCtx := &validationframework.TenantedValidationContext{
			Mode:           validationframework.ModeCreate,
			OrganizationID: w.OrganizationID,
			BusinessUnitID: w.BusinessUnitID,
		}
		err := rule.Validate(ctx, w, valCtx, multiErr)

		require.Error(t, err)
		dcRepo.AssertExpectations(t)
	})
}

func TestMedicalCertComplianceRule(t *testing.T) {
	t.Parallel()

	t.Run("passes with valid medical cert", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		futureExpiry := time.Now().AddDate(1, 0, 0).Unix()
		w.Profile.MedicalCardExpiry = &futureExpiry
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMedicalCertComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails with expired medical cert", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		pastExpiry := time.Now().AddDate(-1, 0, 0).Unix()
		w.Profile.MedicalCardExpiry = &pastExpiry
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMedicalCertComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, "profile.medicalCardExpiry", multiErr.Errors[0].Field)
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails with overdue physical due date", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		pastDue := time.Now().AddDate(0, -6, 0).Unix()
		w.Profile.PhysicalDueDate = &pastDue
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMedicalCertComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, "profile.physicalDueDate", multiErr.Errors[0].Field)
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails with both expired cert and overdue physical", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		pastExpiry := time.Now().AddDate(-1, 0, 0).Unix()
		pastDue := time.Now().AddDate(0, -6, 0).Unix()
		w.Profile.MedicalCardExpiry = &pastExpiry
		w.Profile.PhysicalDueDate = &pastDue
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMedicalCertComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 2)
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes with nil medical card expiry", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.MedicalCardExpiry = nil
		w.Profile.PhysicalDueDate = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMedicalCertComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("skips when enforcement disabled", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		pastExpiry := time.Now().AddDate(-1, 0, 0).Unix()
		w.Profile.MedicalCardExpiry = &pastExpiry
		dc := newDispatchControl()
		dc.EnforceMedicalCertCompliance = false
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMedicalCertComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("skips when profile is nil", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMedicalCertComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes with future physical due date", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		futureDue := time.Now().AddDate(0, 6, 0).Unix()
		w.Profile.PhysicalDueDate = &futureDue
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMedicalCertComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("returns error when dispatch control fetch fails", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).
			Return(nil, errors.New("db error"))

		rule := createMedicalCertComplianceRule(dcRepo)
		ctx := t.Context()
		multiErr := errortypes.NewMultiError()
		valCtx := &validationframework.TenantedValidationContext{
			Mode:           validationframework.ModeCreate,
			OrganizationID: w.OrganizationID,
			BusinessUnitID: w.BusinessUnitID,
		}
		err := rule.Validate(ctx, w, valCtx, multiErr)

		require.Error(t, err)
		dcRepo.AssertExpectations(t)
	})
}

func TestMVRComplianceRule(t *testing.T) {
	t.Parallel()

	t.Run("passes with recent MVR check", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.LastMVRCheck = time.Now().AddDate(0, -6, 0).Unix()
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMVRComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails with overdue MVR check", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.LastMVRCheck = time.Now().AddDate(-2, 0, 0).Unix()
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMVRComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, "profile.lastMvrCheck", multiErr.Errors[0].Field)
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails with overdue MVR due date", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		pastDue := time.Now().AddDate(0, -1, 0).Unix()
		w.Profile.MVRDueDate = &pastDue
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMVRComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, "profile.mvrDueDate", multiErr.Errors[0].Field)
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes with zero LastMVRCheck", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.LastMVRCheck = 0
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMVRComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes with nil MVR due date", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.MVRDueDate = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMVRComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails with both overdue MVR check and due date", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.LastMVRCheck = time.Now().AddDate(-2, 0, 0).Unix()
		pastDue := time.Now().AddDate(0, -1, 0).Unix()
		w.Profile.MVRDueDate = &pastDue
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMVRComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 2)
		dcRepo.AssertExpectations(t)
	})

	t.Run("skips when HOS enforcement disabled", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.LastMVRCheck = time.Now().AddDate(-2, 0, 0).Unix()
		dc := newDispatchControl()
		dc.EnforceHOSCompliance = false
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMVRComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("skips when profile is nil", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMVRComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes with future MVR due date", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		futureDue := time.Now().AddDate(0, 6, 0).Unix()
		w.Profile.MVRDueDate = &futureDue
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createMVRComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("returns error when dispatch control fetch fails", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).
			Return(nil, errors.New("db error"))

		rule := createMVRComplianceRule(dcRepo)
		ctx := t.Context()
		multiErr := errortypes.NewMultiError()
		valCtx := &validationframework.TenantedValidationContext{
			Mode:           validationframework.ModeCreate,
			OrganizationID: w.OrganizationID,
			BusinessUnitID: w.BusinessUnitID,
		}
		err := rule.Validate(ctx, w, valCtx, multiErr)

		require.Error(t, err)
		dcRepo.AssertExpectations(t)
	})
}

func TestDrugTestComplianceRule(t *testing.T) {
	t.Parallel()

	runDrugRule := func(t *testing.T, mutate func(w *worker.Worker, dc *dispatchcontrol.DispatchControl)) *errortypes.MultiError {
		t.Helper()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		dc := newDispatchControl()
		mutate(w, dc)
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createDrugTestComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)
		dcRepo.AssertExpectations(t)
		return multiErr
	}

	t.Run("passes when the record is clear", func(t *testing.T) {
		t.Parallel()
		multiErr := runDrugRule(t, func(w *worker.Worker, _ *dispatchcontrol.DispatchControl) {
			w.Profile.DrugAlcoholStatus = worker.DrugAlcoholClear
		})

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("fails when nothing is on file", func(t *testing.T) {
		t.Parallel()
		multiErr := runDrugRule(t, func(w *worker.Worker, _ *dispatchcontrol.DispatchControl) {
			w.Profile.DrugAlcoholStatus = worker.DrugAlcoholUnknown
		})

		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "profile.drugAlcoholStatus", multiErr.Errors[0].Field)
		assert.Contains(t, multiErr.Errors[0].Message, "pre-employment drug test")
	})

	// A profile assembled in memory has no status at all. Reading that as
	// "clear" would let an untested driver through the one check meant to catch
	// them.
	t.Run("treats an unset status as nothing on file", func(t *testing.T) {
		t.Parallel()
		multiErr := runDrugRule(t, func(w *worker.Worker, _ *dispatchcontrol.DispatchControl) {
			w.Profile.DrugAlcoholStatus = ""
		})

		assert.True(t, multiErr.HasErrors())
	})

	// Using a prohibited driver is not something an organisation can configure
	// its way out of, so the enforcement level does not soften this one.
	t.Run("a prohibition is an error at every enforcement level", func(t *testing.T) {
		t.Parallel()
		multiErr := runDrugRule(t, func(w *worker.Worker, dc *dispatchcontrol.DispatchControl) {
			w.Profile.DrugAlcoholStatus = worker.DrugAlcoholProhibited
			dc.ComplianceEnforcementLevel = dispatchcontrol.ComplianceEnforcementLevelWarning
		})

		require.True(t, multiErr.HasErrors())
		assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
		assert.Contains(t, multiErr.Errors[0].Message, "return-to-duty")
	})

	t.Run("flags an overdue clearinghouse query", func(t *testing.T) {
		t.Parallel()
		overdue := time.Now().AddDate(0, -1, 0).Unix()
		multiErr := runDrugRule(t, func(w *worker.Worker, _ *dispatchcontrol.DispatchControl) {
			w.Profile.DrugAlcoholStatus = worker.DrugAlcoholClear
			w.Profile.NextClearinghouseQueryDue = &overdue
		})

		require.True(t, multiErr.HasErrors())
		assert.Equal(t, "profile.nextClearinghouseQueryDue", multiErr.Errors[0].Field)
	})

	t.Run("a query still in date is not flagged", func(t *testing.T) {
		t.Parallel()
		future := time.Now().AddDate(0, 6, 0).Unix()
		multiErr := runDrugRule(t, func(w *worker.Worker, _ *dispatchcontrol.DispatchControl) {
			w.Profile.DrugAlcoholStatus = worker.DrugAlcoholClear
			w.Profile.NextClearinghouseQueryDue = &future
		})

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("skips when drug and alcohol enforcement disabled", func(t *testing.T) {
		t.Parallel()
		multiErr := runDrugRule(t, func(w *worker.Worker, dc *dispatchcontrol.DispatchControl) {
			w.Profile.DrugAlcoholStatus = worker.DrugAlcoholProhibited
			dc.EnforceDrugAndAlcoholCompliance = false
		})

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("skips when profile is nil", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createDrugTestComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})
}

func TestHazmatComplianceRule(t *testing.T) {
	t.Parallel()

	t.Run("passes for non-hazmat endorsement", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeNone
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes for tanker endorsement without hazmat", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeTanker
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes with valid hazmat expiry for H endorsement", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeHazmat
		futureExpiry := time.Now().AddDate(2, 0, 0).Unix()
		w.Profile.HazmatExpiry = &futureExpiry
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes with valid hazmat expiry for X endorsement", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeTankerHazmat
		futureExpiry := time.Now().AddDate(2, 0, 0).Unix()
		w.Profile.HazmatExpiry = &futureExpiry
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails with nil hazmat expiry for H endorsement", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeHazmat
		w.Profile.HazmatExpiry = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, "profile.hazmatExpiry", multiErr.Errors[0].Field)
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails with zero hazmat expiry for X endorsement", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeTankerHazmat
		zeroExpiry := int64(0)
		w.Profile.HazmatExpiry = &zeroExpiry
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, "profile.hazmatExpiry", multiErr.Errors[0].Field)
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails with expired hazmat", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeHazmat
		pastExpiry := time.Now().AddDate(-1, 0, 0).Unix()
		w.Profile.HazmatExpiry = &pastExpiry
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, "profile.hazmatExpiry", multiErr.Errors[0].Field)
		dcRepo.AssertExpectations(t)
	})

	t.Run("fails with hazmat expiry exceeding 5 year max", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeHazmat
		tooFarExpiry := time.Now().AddDate(6, 0, 0).Unix()
		w.Profile.HazmatExpiry = &tooFarExpiry
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, "profile.hazmatExpiry", multiErr.Errors[0].Field)
		dcRepo.AssertExpectations(t)
	})

	t.Run("skips when hazmat enforcement disabled", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeHazmat
		w.Profile.HazmatExpiry = nil
		dc := newDispatchControl()
		dc.EnforceHazmatCompliance = false
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("skips when profile is nil", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("uses compliance violation code when level is Block", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeHazmat
		pastExpiry := time.Now().AddDate(-1, 0, 0).Unix()
		w.Profile.HazmatExpiry = &pastExpiry
		dc := newDispatchControl()
		dc.ComplianceEnforcementLevel = dispatchcontrol.ComplianceEnforcementLevelBlock
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, errortypes.ErrComplianceViolation, multiErr.Errors[0].Code)
		dcRepo.AssertExpectations(t)
	})

	t.Run("returns error when dispatch control fetch fails", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).
			Return(nil, errors.New("db error"))

		rule := createHazmatComplianceRule(dcRepo)
		ctx := t.Context()
		multiErr := errortypes.NewMultiError()
		valCtx := &validationframework.TenantedValidationContext{
			Mode:           validationframework.ModeCreate,
			OrganizationID: w.OrganizationID,
			BusinessUnitID: w.BusinessUnitID,
		}
		err := rule.Validate(ctx, w, valCtx, multiErr)

		require.Error(t, err)
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes for passenger endorsement without hazmat expiry", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypePassenger
		w.Profile.HazmatExpiry = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})

	t.Run("passes for double-triple endorsement without hazmat expiry", func(t *testing.T) {
		t.Parallel()
		dcRepo := new(mockDispatchControlRepo)
		w := newComplianceWorker()
		w.Profile.Endorsement = worker.EndorsementTypeDoubleTriple
		w.Profile.HazmatExpiry = nil
		dc := newDispatchControl()
		dcRepo.On("GetOrCreate", mock.Anything, w.OrganizationID, w.BusinessUnitID).Return(dc, nil)

		rule := createHazmatComplianceRule(dcRepo)
		multiErr := runComplianceRule(t, rule, w)

		assert.False(t, multiErr.HasErrors())
		dcRepo.AssertExpectations(t)
	})
}

func TestGetComplianceErrorCode(t *testing.T) {
	t.Parallel()

	t.Run("returns compliance violation for Block level", func(t *testing.T) {
		t.Parallel()
		code := getComplianceErrorCode(dispatchcontrol.ComplianceEnforcementLevelBlock)
		assert.Equal(t, errortypes.ErrComplianceViolation, code)
	})

	t.Run("returns invalid for Warning level", func(t *testing.T) {
		t.Parallel()
		code := getComplianceErrorCode(dispatchcontrol.ComplianceEnforcementLevelWarning)
		assert.Equal(t, errortypes.ErrInvalid, code)
	})

	t.Run("returns invalid for Audit level", func(t *testing.T) {
		t.Parallel()
		code := getComplianceErrorCode(dispatchcontrol.ComplianceEnforcementLevelAudit)
		assert.Equal(t, errortypes.ErrInvalid, code)
	})
}
