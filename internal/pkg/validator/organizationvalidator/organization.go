package organizationvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"go.uber.org/fx"
)

// ValidatorParams defines the dependencies required for initializing the Validator.
// This includes the database connection and validation engine factory, organization repository, and logger.
type ValidatorParams struct {
	fx.In

	DB                      db.Connection
	Repo                    repositories.OrganizationRepository
	ValidationEngineFactory framework.ValidationEngineFactory
}

// Validator is a struct that contains the database connection and the validator.
// It provides methods to validate organizations and other related entities.
type Validator struct {
	db   db.Connection
	repo repositories.OrganizationRepository
	vef  framework.ValidationEngineFactory
}

// NewValidator initializes a new Validator with the provided dependencies.
//
// Parameters:
//   - p: ValidatorParams containing dependencies.
//
// Returns:
//   - *Validator: A new Validator instance.
func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db:   p.DB,
		repo: p.Repo,
		vef:  p.ValidationEngineFactory,
	}
}

// Validate validates an organization.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - org: The organization to validate.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) Validate(
	ctx context.Context,
	org *organization.Organization,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Basic validation rules (field presence, format, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				org.Validate(ctx, multiErr)
				return nil
			},
		),
	)

	// * Data integrity validation (uniqueness, references, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				return v.validateBucketName(ctx, org, multiErr)
			},
		),
	)

	return engine.Validate(ctx)
}

func (v *Validator) validateBucketName(
	ctx context.Context,
	org *organization.Organization,
	multiErr *errors.MultiError,
) error {
	if org.BucketName != "" {
		existingOrg, err := v.repo.GetByID(ctx, repositories.GetOrgByIDOptions{
			OrgID: org.ID,
		})
		if err != nil {
			return err
		}

		if existingOrg.BucketName != org.BucketName {
			multiErr.Add(
				"bucketName",
				errors.ErrInvalidOperation,
				"You cannot change the bucket name of an organization",
			)
		}
	}

	return nil
}
