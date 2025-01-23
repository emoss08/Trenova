package dbtests_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/internal/pkg/validator/compliancevalidator"
	"github.com/emoss08/trenova/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestComplianceValidator(t *testing.T) {
	ctx := context.Background()
	db := testutils.GetTestDB()

	fixture, err := db.Fixture(ctx)
	require.NotNil(t, fixture, "fixture not found")
	require.NoError(t, err, "error loading fixture")

	workerProfile := fixture.MustRow("WorkerProfile.wp_1").(*worker.WorkerProfile)

	hazmatRepo := repositories.NewHazmatExpirationRepository(repositories.HazmatExpirationRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     db,
	})

	validator := compliancevalidator.NewValidator(compliancevalidator.ValidatorParams{
		HazmatExpRepo: hazmatRepo,
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
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			me := errors.NewMultiError()

			scenario.modifyProfile(workerProfile)

			validator.ValidateWorkerCompliance(ctx, workerProfile, me)

			matcher := testutils.NewErrorMatcher(t, me)
			matcher.HasExactErrors(scenario.expectedErrors)
		})
	}
}
