package subscription

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	MaxPlanKeyLength  = 50
	MaxStripeIDLength = 255
)

var (
	_ bun.BeforeAppendModelHook          = (*Subscription)(nil)
	_ validationframework.TenantedEntity = (*Subscription)(nil)
)

type Subscription struct {
	bun.BaseModel `bun:"table:organization_subscriptions,alias:osub" json:"-"`

	ID                   pulid.ID `json:"id"                             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"                 bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID `json:"organizationId"                 bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	PlanKey              string   `json:"planKey"                        bun:"plan_key,type:VARCHAR(50),notnull"`
	Status               Status   `json:"status"                         bun:"status,type:VARCHAR(20),notnull"`
	TrialEndsAt          int64    `json:"trialEndsAt"                    bun:"trial_ends_at,type:BIGINT,notnull"`
	ReadOnlyUntil        int64    `json:"readOnlyUntil"                  bun:"read_only_until,type:BIGINT,notnull"`
	StripeCustomerID     string   `json:"stripeCustomerId,omitempty"     bun:"stripe_customer_id,type:VARCHAR(255),nullzero"`
	StripeSubscriptionID string   `json:"stripeSubscriptionId,omitempty" bun:"stripe_subscription_id,type:VARCHAR(255),nullzero"`
	Version              int64    `json:"version"                        bun:"version,type:BIGINT,notnull"`
	CreatedAt            int64    `json:"createdAt"                      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64    `json:"updatedAt"                      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

func (s *Subscription) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("osub_")
		}
		if s.CreatedAt == 0 {
			s.CreatedAt = now
		}
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
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
