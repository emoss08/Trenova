package ratetableservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratetable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Repo repositories.RateTableRepository
}

type Validator struct {
	repo repositories.RateTableRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{repo: p.Repo}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *ratetable.RateTable,
) *errortypes.MultiError {
	return v.validate(ctx, entity, true)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *ratetable.RateTable,
) *errortypes.MultiError {
	return v.validate(ctx, entity, false)
}

func (v *Validator) validate(
	ctx context.Context,
	entity *ratetable.RateTable,
	isCreate bool,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	entity.Validate(multiErr)
	v.validateUniqueKey(ctx, entity, isCreate, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) validateUniqueKey(
	ctx context.Context,
	entity *ratetable.RateTable,
	isCreate bool,
	multiErr *errortypes.MultiError,
) {
	if entity.Key == "" {
		return
	}

	existing, err := v.repo.GetByKeys(ctx, &repositories.GetRateTablesByKeysRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
		Keys: []string{entity.Key},
	})
	if err != nil {
		multiErr.Add("key", errortypes.ErrSystemError, "Failed to verify key uniqueness")
		return
	}

	for _, other := range existing {
		if isCreate || other.ID != entity.ID {
			multiErr.Add(
				"key",
				errortypes.ErrDuplicate,
				"Rate table with this key already exists in your organization",
			)
			return
		}
	}
}
