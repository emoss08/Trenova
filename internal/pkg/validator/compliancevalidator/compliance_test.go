// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package compliancevalidator_test

import (
	"context"
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/internal/pkg/validator/compliancevalidator"
	"github.com/emoss08/trenova/test/mocks"
	"github.com/emoss08/trenova/test/testutils"
)

var (
	ts  *testutils.TestSetup
	ctx = context.Background()
)

func TestMain(m *testing.M) {
	setup, err := testutils.NewTestSetup(ctx)
	if err != nil {
		panic(err)
	}

	ts = setup

	os.Exit(m.Run())
}

func TestComplianceValidator(t *testing.T) {
	workerProfile := ts.Fixture.MustRow("WorkerProfile.wp_1").(*worker.WorkerProfile)
	mockVef := &mocks.MockValidationEngineFactory{}

	hazmatRepo := repositories.NewHazmatExpirationRepository(
		repositories.HazmatExpirationRepositoryParams{
			Logger: logger.NewLogger(testutils.NewTestConfig()),
			DB:     ts.DB,
		},
	)

	shipmentControlRepo := repositories.NewShipmentControlRepository(
		repositories.ShipmentControlRepositoryParams{
			Logger: logger.NewLogger(testutils.NewTestConfig()),
			DB:     ts.DB,
		},
	)

	validator := compliancevalidator.NewValidator(compliancevalidator.ValidatorParams{
		HazmatExpRepo:           hazmatRepo,
		ShipmentControlRepo:     shipmentControlRepo,
		ValidationEngineFactory: mockVef,
	})

	scenarios := []struct {
		name           string
		modifyProfile  func(*worker.WorkerProfile)
		expectedErrors []struct {
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
			expectedErrors: []struct {
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
			expectedErrors: []struct {
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
		{
			name: "med_exam_is_required_every_24_months",
			modifyProfile: func(p *worker.WorkerProfile) {
				p.LastMVRCheck = timeutils.YearsFromNowUnix(1)
				p.MVRDueDate = timeutils.YearsFromNowUnixPointer(1)
				p.PhysicalDueDate = timeutils.YearsAgoUnixPointer(3)
			},
			expectedErrors: []struct {
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
		{
			name: "commercial_drivers_license_is_expired",
			modifyProfile: func(p *worker.WorkerProfile) {
				p.LicenseExpiry = timeutils.YearsAgoUnix(1)
				p.PhysicalDueDate = timeutils.MonthsAgoUnixPointer(1)
			},
			expectedErrors: []struct {
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

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			scenario.modifyProfile(workerProfile)
			me := errors.NewMultiError()

			validator.Validate(ctx, workerProfile, me)

			matcher := testutils.NewErrorMatcher(t, me)
			matcher.HasExactErrors(scenario.expectedErrors)
		})
	}
}
