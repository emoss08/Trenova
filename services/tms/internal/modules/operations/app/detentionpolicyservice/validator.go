package detentionpolicyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Repo detentionports.DetentionPolicyRepository
}

type Validator struct {
	repo detentionports.DetentionPolicyRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{repo: p.Repo}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
) *errortypes.MultiError {
	return v.validate(ctx, entity, true)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
) *errortypes.MultiError {
	return v.validate(ctx, entity, false)
}

func (v *Validator) validate(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
	isCreate bool,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	entity.Validate(multiErr)
	v.validateUniqueCode(ctx, entity, isCreate, multiErr)
	v.validateNoOverlap(ctx, entity, isCreate, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) validateUniqueCode(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
	isCreate bool,
	multiErr *errortypes.MultiError,
) {
	if entity.Code == "" {
		return
	}

	req := &detentionports.DetentionPolicyCodeExistsRequest{
		TenantInfo: tenantOf(entity),
		Code:       entity.Code,
	}

	if !isCreate {
		id := entity.ID
		req.ExcludeID = &id
	}

	exists, err := v.repo.CodeExists(ctx, req)
	if err != nil {
		multiErr.Add("code", errortypes.ErrSystemError, "Failed to verify code uniqueness")
		return
	}

	if exists {
		multiErr.Add("code", errortypes.ErrDuplicate,
			"A detention policy with this code already exists in your organization")
	}
}

// validateNoOverlap blocks two active policies that pin the same scope over
// intersecting dates. Resolution would otherwise fall through to a tie-break
// and the carrier would not know which contract terms were charged.
func (v *Validator) validateNoOverlap(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
	isCreate bool,
	multiErr *errortypes.MultiError,
) {
	if entity.Status != detentionpolicy.PolicyStatusActive {
		return
	}

	req := &detentionports.FindOverlappingDetentionPoliciesRequest{
		TenantInfo:         tenantOf(entity),
		CustomerID:         entity.CustomerID,
		LocationID:         entity.LocationID,
		IsOrgDefault:       entity.IsOrgDefault,
		EffectiveStartDate: entity.EffectiveStartDate,
		EffectiveEndDate:   entity.EffectiveEndDate,
	}

	if !isCreate {
		id := entity.ID
		req.ExcludeID = &id
	}

	conflicts, err := v.repo.FindOverlapping(ctx, req)
	if err != nil {
		multiErr.Add("effectiveStartDate", errortypes.ErrSystemError,
			"Failed to verify overlapping policies")
		return
	}

	for _, conflict := range conflicts {
		if conflict == nil || conflict.Priority != entity.Priority {
			continue
		}

		field := "effectiveStartDate"
		if entity.IsOrgDefault {
			field = "isOrgDefault"
		}

		multiErr.Add(
			field,
			errortypes.ErrInvalid,
			"Policy {0} already covers this scope for an overlapping date range. Change the dates, narrow the scope, or set a different priority.",
			conflict.Code,
		)
		return
	}
}

func tenantOf(entity *detentionpolicy.DetentionPolicy) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.GetOrganizationID(),
		BuID:  entity.GetBusinessUnitID(),
	}
}
