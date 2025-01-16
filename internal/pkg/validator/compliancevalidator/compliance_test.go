package compliancevalidator_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/trenova-app/transport/internal/core/domain/businessunit"
	"github.com/trenova-app/transport/internal/core/domain/organization"
	"github.com/trenova-app/transport/internal/core/domain/usstate"
	"github.com/trenova-app/transport/internal/core/domain/worker"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/utils/timeutils"
	"github.com/trenova-app/transport/internal/pkg/validator/compliancevalidator"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/trenova-app/transport/test/testutils"
)

type ComplianceValidatorTestSuite struct {
	testutils.BaseSuite
	validator *compliancevalidator.Validator
}

func TestWorkerSuite(t *testing.T) {
	suite.Run(t, new(ComplianceValidatorTestSuite))
}

func (s *ComplianceValidatorTestSuite) SetupTest() {
	s.BaseSuite.LoadTestDB()
	s.validator = s.Validators.ComplianceValidator
}

func (s *ComplianceValidatorTestSuite) createValidWorkerProfile() *worker.WorkerProfile {
	workerID := s.GetFixture("Worker.worker_1").(*worker.Worker).ID
	businessUnitID := s.GetFixture("BusinessUnit.trenova").(*businessunit.BusinessUnit).ID
	organizationID := s.GetFixture("Organization.trenova").(*organization.Organization).ID
	licenseStateID := s.GetFixture("UsState.fl").(*usstate.UsState).ID

	now := timeutils.NowUnix()
	sixMonthsFromNow := timeutils.MonthsFromNowUnix(6)

	return &worker.WorkerProfile{
		ID:                     pulid.MustNew("wp"),
		WorkerID:               workerID,
		BusinessUnitID:         businessUnitID,
		OrganizationID:         organizationID,
		LicenseStateID:         licenseStateID,
		LastMVRCheck:           now,
		MVRDueDate:             &sixMonthsFromNow,
		PhysicalDueDate:        &sixMonthsFromNow,
		HireDate:               now,
		LastDrugTest:           now,
		LicenseNumber:          "1234567890",
		DOB:                    timeutils.YearsAgoUnix(30),
		Endorsement:            worker.EndorsementNone,
		LastComplianceCheck:    now,
		HazmatExpiry:           timeutils.YearsFromNowUnix(1),
		LicenseExpiry:          timeutils.YearsFromNowUnix(2),
		TerminationDate:        nil,
		ComplianceStatus:       worker.ComplianceStatusCompliant,
		IsQualified:            true,
		DisqualificationReason: "",
		Version:                1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func (s *ComplianceValidatorTestSuite) TestValidateMVRCompliance() {
	scenarios := []struct {
		name          string
		modifyProfile func(*worker.WorkerProfile)
		expectErrors  []struct {
			Field   string
			Code    errors.ErrorCode
			Message string
		}
	}{
		{
			name: "annual_mvr_is_overdue",
			modifyProfile: func(p *worker.WorkerProfile) {
				p.LastMVRCheck = timeutils.YearsAgoUnix(2)
			},
			expectErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "profile.lastMVRCheck",
					Code:    errors.ErrComplianceViolation,
					Message: "Annual MVR Check is overdue (49 CFR § 391.25(c)(2))",
				},
			},
		},
		{
			name: "mvr_renewal_is_overdue",
			modifyProfile: func(p *worker.WorkerProfile) {
				p.LastMVRCheck = timeutils.YearsFromNowUnix(1)
				p.MVRDueDate = timeutils.YearsAgoUnixPointer(2)
			},
			expectErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "profile.mvrDueDate",
					Code:    errors.ErrComplianceViolation,
					Message: "MVR renewal is overdue (49 CFR § 391.25(c)(2))",
				},
			},
		},
	}

	for _, tt := range scenarios {
		s.Run(tt.name, func() {
			profile := s.createValidWorkerProfile()
			if tt.modifyProfile != nil {
				tt.modifyProfile(profile)
			}

			multiErr := errors.NewMultiError()
			s.validator.ValidateWorkerCompliance(s.Ctx, profile, multiErr)

			matcher := testutils.NewErrorMatcher(s.T(), multiErr)
			matcher.HasExactErrors(tt.expectErrors)
		})
	}
}

func (s *ComplianceValidatorTestSuite) TestMedicalCertificate() {
	scenarios := []struct {
		name          string
		modifyProfile func(*worker.WorkerProfile)
		expectErrors  []struct {
			Field   string
			Code    errors.ErrorCode
			Message string
		}
	}{
		{
			name: "med_exam_is_required_every_24_months",
			modifyProfile: func(p *worker.WorkerProfile) {
				p.LastMVRCheck = timeutils.YearsFromNowUnix(1)
				p.MVRDueDate = timeutils.YearsFromNowUnixPointer(1)
				p.PhysicalDueDate = timeutils.YearsAgoUnixPointer(3)
			},
			expectErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "profile.physicalDueDate",
					Code:    errors.ErrComplianceViolation,
					Message: "Medical examination is required at least every 24 months (49 CFR § 391.45)",
				},
			},
		},
	}

	for _, tt := range scenarios {
		s.Run(tt.name, func() {
			profile := s.createValidWorkerProfile()
			if tt.modifyProfile != nil {
				tt.modifyProfile(profile)
			}

			multiErr := errors.NewMultiError()
			s.validator.ValidateWorkerCompliance(s.Ctx, profile, multiErr)

			matcher := testutils.NewErrorMatcher(s.T(), multiErr)
			matcher.HasExactErrors(tt.expectErrors)
		})
	}
}

func (s *ComplianceValidatorTestSuite) TestDrugAndAlcoholCompliance() {
	scenarios := []struct {
		name          string
		modifyProfile func(*worker.WorkerProfile)
		expectErrors  []struct {
			Field   string
			Code    errors.ErrorCode
			Message string
		}
	}{
		{
			name: "pre_employment_drug_test_is_required",
			modifyProfile: func(p *worker.WorkerProfile) {
				p.HireDate = timeutils.YearsFromNowUnix(1)
				p.LastDrugTest = timeutils.MonthsAgoUnix(1)
			},
			expectErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "profile.lastDrugTest",
					Code:    errors.ErrComplianceViolation,
					Message: "Pre-employment drug test is required (49 CFR § 382.301(a))",
				},
			},
		},
	}

	for _, tt := range scenarios {
		s.Run(tt.name, func() {
			profile := s.createValidWorkerProfile()
			if tt.modifyProfile != nil {
				tt.modifyProfile(profile)
			}

			multiErr := errors.NewMultiError()
			s.validator.ValidateWorkerCompliance(s.Ctx, profile, multiErr)

			matcher := testutils.NewErrorMatcher(s.T(), multiErr)
			matcher.HasExactErrors(tt.expectErrors)
		})
	}
}

func (s *ComplianceValidatorTestSuite) TestValidateDriverQualification() {
	scenarios := []struct {
		name          string
		modifyProfile func(*worker.WorkerProfile)
		expectErrors  []struct {
			Field   string
			Code    errors.ErrorCode
			Message string
		}
	}{
		{
			name: "commercial_drivers_license_is_expired",
			modifyProfile: func(p *worker.WorkerProfile) {
				p.LicenseExpiry = timeutils.YearsAgoUnix(1)
			},
			expectErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "profile.licenseExpiry",
					Code:    errors.ErrComplianceViolation,
					Message: "Commercial driver's license is expired (49 CFR § 391.11(b)(5))",
				},
			},
		},
	}

	for _, tt := range scenarios {
		s.Run(tt.name, func() {
			profile := s.createValidWorkerProfile()
			if tt.modifyProfile != nil {
				tt.modifyProfile(profile)
			}

			multiErr := errors.NewMultiError()
			s.validator.ValidateWorkerCompliance(s.Ctx, profile, multiErr)

			matcher := testutils.NewErrorMatcher(s.T(), multiErr)
			matcher.HasExactErrors(tt.expectErrors)
		})
	}
}

func (s *ComplianceValidatorTestSuite) TestValidateHazmatCompliance() {
	scenarios := []struct {
		name          string
		modifyProfile func(*worker.WorkerProfile)
		expectErrors  []struct {
			Field   string
			Code    errors.ErrorCode
			Message string
		}
	}{
		{
			name: "hazmat_endorsement_is_expired",
			modifyProfile: func(p *worker.WorkerProfile) {
				p.HazmatExpiry = timeutils.YearsAgoUnix(2)
				p.Endorsement = worker.EndorsementTankerHazmat
			},
			expectErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "profile.licenseExpiry",
					Code:    errors.ErrComplianceViolation,
					Message: "Hazmat endorsement is expired (49 CFR § 383.93)",
				},
			},
		},
		{
			name: "hazmat_endorsement_exceeds_max_expiration_period",
			modifyProfile: func(p *worker.WorkerProfile) {
				p.HazmatExpiry = timeutils.YearsFromNowUnix(6)
				p.Endorsement = worker.EndorsementTankerHazmat
			},
			expectErrors: []struct {
				Field   string
				Code    errors.ErrorCode
				Message string
			}{
				{
					Field:   "profile.hazmatExpiry",
					Code:    errors.ErrComplianceViolation,
					Message: "Hazmat endorsement exceeds the maximum allowed period of 4 years (49 CFR § 383.93)",
				},
			},
		},
	}

	for _, tt := range scenarios {
		s.Run(tt.name, func() {
			profile := s.createValidWorkerProfile()
			if tt.modifyProfile != nil {
				tt.modifyProfile(profile)
			}

			multiErr := errors.NewMultiError()
			s.validator.ValidateWorkerCompliance(s.Ctx, profile, multiErr)

			matcher := testutils.NewErrorMatcher(s.T(), multiErr)
			matcher.Debug()
			matcher.HasExactErrors(tt.expectErrors)
		})
	}
}
