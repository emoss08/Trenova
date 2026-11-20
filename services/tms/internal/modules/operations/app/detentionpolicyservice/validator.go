package detentionpolicyservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	defaultMustStayActive = "This policy is the organization's default detention policy and must stay active. Choose another default in Shipment Controls first."
	defaultMustCoverAll   = "This policy is the organization's default detention policy and must cover all freight. Choose another default in Shipment Controls before narrowing it."
	defaultCannotDelete   = "This policy is the organization's default detention policy. Choose another default in Shipment Controls before deleting it."
	defaultMustNotLapse   = "This policy is the organization's default detention policy and must not expire. Choose another default in Shipment Controls before giving it an end date."
	defaultMustApplyNow   = "This policy is the organization's default detention policy and must already be in effect. Choose another default in Shipment Controls before moving its start date into the future."
)

type ValidatorParams struct {
	fx.In

	Repo                detentionports.DetentionPolicyRepository
	ShipmentControlRepo tenantports.ShipmentControlRepository
	Now                 timeutils.Clock
}

type Validator struct {
	repo     detentionports.DetentionPolicyRepository
	controls tenantports.ShipmentControlRepository
	now      timeutils.Clock
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{repo: p.Repo, controls: p.ShipmentControlRepo, now: p.Now}
}

// ValidateDelete refuses to delete the policy shipment control names as the
// organization's default, which the engine falls back to for every stop no
// other policy covers.
func (v *Validator) ValidateDelete(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
) error {
	isDefault, err := v.isOrganizationDefault(ctx, entity)
	if err != nil {
		return err
	}
	if isDefault {
		return errortypes.NewBusinessError(defaultCannotDelete)
	}

	return nil
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
) error {
	return v.validate(ctx, entity, true)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
) error {
	return v.validate(ctx, entity, false)
}

func (v *Validator) validate(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
	isCreate bool,
) error {
	multiErr := errortypes.NewMultiError()

	entity.Validate(multiErr)
	v.validateUniqueCode(ctx, entity, isCreate, multiErr)
	v.validateNoOverlap(ctx, entity, isCreate, multiErr)
	if !isCreate {
		v.validateOrganizationDefault(ctx, entity, multiErr)
	}

	return multiErr.Err()
}

// validateOrganizationDefault keeps the policy shipment control names as the
// default in the shape the engine relies on: active, covering all freight, and
// in effect from today on, so no stop is ever left without a policy.
func (v *Validator) validateOrganizationDefault(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
	multiErr *errortypes.MultiError,
) {
	isDefault, err := v.isOrganizationDefault(ctx, entity)
	if err != nil {
		multiErr.AddInternal(err)
		return
	}
	if !isDefault {
		return
	}

	if entity.Status != detentionpolicy.PolicyStatusActive {
		multiErr.Add("status", errortypes.ErrInvalid, defaultMustStayActive)
	}
	if field := narrowingField(entity); field != "" {
		multiErr.Add(field, errortypes.ErrInvalid, defaultMustCoverAll)
	}
	if entity.EffectiveEndDate != nil {
		multiErr.Add("effectiveEndDate", errortypes.ErrInvalid, defaultMustNotLapse)
	}
	if entity.EffectiveStartDate != nil && *entity.EffectiveStartDate > v.now() {
		multiErr.Add("effectiveStartDate", errortypes.ErrInvalid, defaultMustApplyNow)
	}
}

func (v *Validator) isOrganizationDefault(
	ctx context.Context,
	entity *detentionpolicy.DetentionPolicy,
) (bool, error) {
	if entity.ID.IsNil() {
		return false, nil
	}

	defaultID, err := organizationDefaultID(ctx, v.controls, tenantOf(entity))
	if err != nil {
		return false, err
	}

	return defaultID.IsNotNil() && defaultID == entity.ID, nil
}

func narrowingField(entity *detentionpolicy.DetentionPolicy) string {
	switch {
	case entity.LocationID != nil && entity.LocationID.IsNotNil():
		return "locationId"
	case entity.CustomerID != nil && entity.CustomerID.IsNotNil():
		return "customerId"
	case len(entity.ShipmentTypeIDs) > 0:
		return "shipmentTypeIds"
	case len(entity.ServiceTypeIDs) > 0:
		return "serviceTypeIds"
	case len(entity.CommodityIDs) > 0:
		return "commodityIds"
	case len(entity.StopTypes) > 0:
		return "stopTypes"
	case entity.AppointmentStopsOnly:
		return "appointmentStopsOnly"
	default:
		return ""
	}
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
		multiErr.AddInternal(fmt.Errorf("failed to verify code uniqueness: %w", err))
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
		EffectiveStartDate: entity.EffectiveStartDate,
		EffectiveEndDate:   entity.EffectiveEndDate,
	}

	if !isCreate {
		id := entity.ID
		req.ExcludeID = &id
	}

	conflicts, err := v.repo.FindOverlapping(ctx, req)
	if err != nil {
		multiErr.AddInternal(fmt.Errorf("failed to verify overlapping policies: %w", err))
		return
	}

	for _, conflict := range conflicts {
		if conflict == nil || conflict.Priority != entity.Priority ||
			!conflict.SameScope(entity) {
			continue
		}

		multiErr.Add(
			"effectiveStartDate",
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
