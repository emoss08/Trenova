package detentionpolicyservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePolicyRepo struct {
	detentionports.DetentionPolicyRepository

	overlapping []*detentionpolicy.DetentionPolicy
}

func (fakePolicyRepo) CodeExists(
	context.Context,
	*detentionports.DetentionPolicyCodeExistsRequest,
) (bool, error) {
	return false, nil
}

func (r fakePolicyRepo) FindOverlapping(
	context.Context,
	*detentionports.FindOverlappingDetentionPoliciesRequest,
) ([]*detentionpolicy.DetentionPolicy, error) {
	return r.overlapping, nil
}

type fakeControls struct {
	tenantports.ShipmentControlRepository

	control *shipmentcontrol.ShipmentControl
}

func (c fakeControls) Get(
	context.Context,
	tenantports.GetShipmentControlRequest,
) (*shipmentcontrol.ShipmentControl, error) {
	if c.control == nil {
		return nil, errortypes.NewNotFoundError(
			"ShipmentControl not found within your organization",
		)
	}

	return c.control, nil
}

const validatorNow int64 = 1_800_000_000

func newValidator(
	defaultPolicyID *pulid.ID,
	overlapping ...*detentionpolicy.DetentionPolicy,
) *Validator {
	return &Validator{
		repo: fakePolicyRepo{overlapping: overlapping},
		controls: fakeControls{
			control: &shipmentcontrol.ShipmentControl{DefaultDetentionPolicyID: defaultPolicyID},
		},
		now: func() int64 { return validatorNow },
	}
}

func storedPolicy(code string) *detentionpolicy.DetentionPolicy {
	return &detentionpolicy.DetentionPolicy{
		ID:                      pulid.MustNew("dtp_"),
		OrganizationID:          pulid.MustNew("org_"),
		BusinessUnitID:          pulid.MustNew("bu_"),
		Name:                    code,
		Code:                    code,
		Status:                  detentionpolicy.PolicyStatusActive,
		ClockStartBasis:         detentionpolicy.ClockStartBasisArrival,
		LateArrivalRule:         detentionpolicy.LateArrivalRuleNoEffect,
		RoundingMode:            detentionpolicy.RoundingModeUp,
		BillingIncrementMinutes: 15,
		BillingFreeMinutes:      120,
		RateSource:              detentionpolicy.RateSourceAccessorial,
		AccessorialChargeID:     pulid.MustNew("acc_"),
		NotificationRequirement: detentionpolicy.NotificationRequirementNone,
		UnnotifiedBehavior:      detentionpolicy.UnnotifiedBehaviorBill,
		DayBoundaryMode:         detentionpolicy.CapScopePerStop,
		Currency:                "USD",
	}
}

func TestValidateUpdate_DefaultPolicyMustStayActive(t *testing.T) {
	t.Parallel()

	entity := storedPolicy("STD")
	entity.Status = detentionpolicy.PolicyStatusInactive

	result := validationtest.Refused(
		t,
		newValidator(&entity.ID).ValidateUpdate(t.Context(), entity),
	)

	refusal := result.Refuses("status", errortypes.ErrInvalid)
	assert.Equal(t, defaultMustStayActive, refusal.Message)
	assert.True(t, result.Exactly(validationtest.At("status", errortypes.ErrInvalid)))
}

func TestValidateUpdate_DefaultPolicyMustCoverAllFreight(t *testing.T) {
	t.Parallel()

	entity := storedPolicy("STD")
	entity.CustomerID = new(pulid.MustNew("cus_"))

	result := validationtest.Refused(
		t,
		newValidator(&entity.ID).ValidateUpdate(t.Context(), entity),
	)

	refusal := result.Refuses("customerId", errortypes.ErrInvalid)
	assert.Equal(t, defaultMustCoverAll, refusal.Message)
	assert.True(t, result.Exactly(validationtest.At("customerId", errortypes.ErrInvalid)))
}

func TestValidateUpdate_DefaultPolicyCannotBeLimitedToAppointmentStops(t *testing.T) {
	t.Parallel()

	entity := storedPolicy("STD")
	entity.AppointmentStopsOnly = true

	result := validationtest.Refused(
		t,
		newValidator(&entity.ID).ValidateUpdate(t.Context(), entity),
	)

	assert.True(t, result.Exactly(validationtest.At("appointmentStopsOnly", errortypes.ErrInvalid)))
}

func TestValidateUpdate_DefaultPolicyMustNotExpire(t *testing.T) {
	t.Parallel()

	entity := storedPolicy("STD")
	entity.EffectiveEndDate = new(validatorNow + 86_400)

	result := validationtest.Refused(
		t,
		newValidator(&entity.ID).ValidateUpdate(t.Context(), entity),
	)

	refusal := result.Refuses("effectiveEndDate", errortypes.ErrInvalid)
	assert.Equal(t, defaultMustNotLapse, refusal.Message)
	assert.True(t, result.Exactly(validationtest.At("effectiveEndDate", errortypes.ErrInvalid)))
}

func TestValidateUpdate_DefaultPolicyMustAlreadyBeInEffect(t *testing.T) {
	t.Parallel()

	entity := storedPolicy("STD")
	entity.EffectiveStartDate = new(validatorNow + 1)

	result := validationtest.Refused(
		t,
		newValidator(&entity.ID).ValidateUpdate(t.Context(), entity),
	)

	refusal := result.Refuses("effectiveStartDate", errortypes.ErrInvalid)
	assert.Equal(t, defaultMustApplyNow, refusal.Message)
	assert.True(t, result.Exactly(validationtest.At("effectiveStartDate", errortypes.ErrInvalid)))
}

func TestValidateUpdate_DefaultPolicyMayHaveStartedAlready(t *testing.T) {
	t.Parallel()

	entity := storedPolicy("STD")
	entity.EffectiveStartDate = new(validatorNow)

	err := newValidator(&entity.ID).ValidateUpdate(t.Context(), entity)

	assert.True(t, validationtest.Outcome(t, err).Valid())
}

func TestValidateUpdate_OtherPoliciesMayBeScopedAndDeactivated(t *testing.T) {
	t.Parallel()

	defaultID := pulid.MustNew("dtp_")
	entity := storedPolicy("ACME")
	entity.CustomerID = new(pulid.MustNew("cus_"))
	entity.Status = detentionpolicy.PolicyStatusInactive
	entity.EffectiveStartDate = new(validatorNow + 86_400)
	entity.EffectiveEndDate = new(validatorNow + 2*86_400)

	err := newValidator(&defaultID).ValidateUpdate(t.Context(), entity)

	assert.True(t, validationtest.Outcome(t, err).Valid())
}

func TestValidateUpdate_NoShipmentControlMeansNoDefault(t *testing.T) {
	t.Parallel()

	entity := storedPolicy("STD")
	entity.Status = detentionpolicy.PolicyStatusInactive
	v := &Validator{repo: fakePolicyRepo{}, controls: fakeControls{}}

	assert.True(t, validationtest.Outcome(t, v.ValidateUpdate(t.Context(), entity)).Valid())
}

func TestValidateCreate_DoesNotReadShipmentControl(t *testing.T) {
	t.Parallel()

	entity := storedPolicy("NEW")
	entity.Status = detentionpolicy.PolicyStatusDraft
	v := &Validator{repo: fakePolicyRepo{}}

	assert.True(t, validationtest.Outcome(t, v.ValidateCreate(t.Context(), entity)).Valid())
}

func TestValidateDelete_RefusesTheDefaultPolicy(t *testing.T) {
	t.Parallel()

	entity := storedPolicy("STD")

	err := newValidator(&entity.ID).ValidateDelete(t.Context(), entity)

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Equal(t, defaultCannotDelete, business.Message)
}

func TestValidateDelete_AllowsAnyOtherPolicy(t *testing.T) {
	t.Parallel()

	defaultID := pulid.MustNew("dtp_")

	require.NoError(t, newValidator(&defaultID).ValidateDelete(t.Context(), storedPolicy("ACME")))
	require.NoError(t, newValidator(nil).ValidateDelete(t.Context(), storedPolicy("ACME")))
}

func TestValidateNoOverlap_IgnoresPoliciesWithADifferentScope(t *testing.T) {
	t.Parallel()

	reefer := storedPolicy("REEFER")
	reefer.ShipmentTypeIDs = []pulid.ID{pulid.MustNew("spt_")}
	entity := storedPolicy("ALL")

	err := newValidator(nil, reefer).ValidateCreate(t.Context(), entity)

	assert.True(t, validationtest.Outcome(t, err).Valid(),
		"an all-freight policy and a reefer-only one at the same priority are not ambiguous")
}

func TestValidateNoOverlap_RefusesTheSameScopeAtTheSamePriority(t *testing.T) {
	t.Parallel()

	existing := storedPolicy("OLD")
	entity := storedPolicy("NEW")

	result := validationtest.Refused(
		t,
		newValidator(nil, existing).ValidateCreate(t.Context(), entity),
	)

	assert.True(t, result.Exactly(validationtest.At("effectiveStartDate", errortypes.ErrInvalid)))
}
