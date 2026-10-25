package subscription

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MaxPlanKeyLength  = 50
	MaxStripeIDLength = 255
)

var (
	_ validationframework.TenantedEntity = (*Subscription)(nil)
)

type Subscription struct {
	keel.Table `keel:"organization_subscriptions"`

	ID                   pulid.ID `json:"id"                             keel:"id,pk,autoid:osub_"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"                 keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID `json:"organizationId"                 keel:"organization_id,pk,tenant"`
	PlanKey              string   `json:"planKey"                        keel:"plan_key"`
	Status               Status   `json:"status"                         keel:"status"`
	TrialEndsAt          int64    `json:"trialEndsAt"                    keel:"trial_ends_at"`
	ReadOnlyUntil        int64    `json:"readOnlyUntil"                  keel:"read_only_until"`
	StripeCustomerID     string   `json:"stripeCustomerId,omitempty"     keel:"stripe_customer_id,nullzero"`
	StripeSubscriptionID string   `json:"stripeSubscriptionId,omitempty" keel:"stripe_subscription_id,nullzero"`
	Version              int64    `json:"version"                        keel:"version,version"`
	CreatedAt            int64    `json:"createdAt"                      keel:"created_at,autocreate"`
	UpdatedAt            int64    `json:"updatedAt"                      keel:"updated_at,autoupdate"`
}

func (s *Subscription) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		s,
		validation.Field(&s.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&s.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(
			&s.PlanKey,
			validation.Required.Error("Plan is required"),
			validation.RuneLength(1, MaxPlanKeyLength).
				Error("Plan must be between 1 and 50 characters"),
		),
		validation.Field(
			&s.Status,
			validation.Required.Error("Status is required"),
			validation.By(func(any) error {
				if !s.Status.IsValid() {
					return validation.NewError("validation_invalid", "Status is invalid")
				}
				return nil
			}),
		),
		validation.Field(&s.TrialEndsAt, validation.Required.Error("Trial end is required")),
		validation.Field(
			&s.ReadOnlyUntil,
			validation.Required.Error("Read-only end is required"),
			validation.Min(s.TrialEndsAt).
				Error("Read-only end must not be before the trial end"),
		),
		validation.Field(
			&s.StripeCustomerID,
			validation.RuneLength(0, MaxStripeIDLength).
				Error("Stripe customer ID must be at most 255 characters"),
		),
		validation.Field(
			&s.StripeSubscriptionID,
			validation.RuneLength(0, MaxStripeIDLength).
				Error("Stripe subscription ID must be at most 255 characters"),
		),
	))
}

func (s *Subscription) EffectiveStatus(now int64) Status {
	switch s.Status {
	case StatusTrialing:
		if now >= s.ReadOnlyUntil {
			return StatusExpired
		}
		if now >= s.TrialEndsAt {
			return StatusReadOnly
		}
		return StatusTrialing
	case StatusReadOnly:
		if now >= s.ReadOnlyUntil {
			return StatusExpired
		}
		return StatusReadOnly
	case StatusActive, StatusExpired:
		return s.Status
	default:
		return s.Status
	}
}

func (s *Subscription) GetID() pulid.ID {
	return s.ID
}

func (s *Subscription) GetOrganizationID() pulid.ID {
	return s.OrganizationID
}

func (s *Subscription) GetBusinessUnitID() pulid.ID {
	return s.BusinessUnitID
}

func (s *Subscription) GetTableName() string {
	return "organization_subscriptions"
}
