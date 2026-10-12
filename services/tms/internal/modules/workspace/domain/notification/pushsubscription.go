package notification

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*PushSubscription)(nil)

type PushSubscription struct {
	bun.BaseModel `bun:"table:user_push_subscriptions,alias:ups" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Endpoint       string   `json:"endpoint"       bun:"endpoint,type:TEXT,notnull"`
	P256dh         string   `json:"p256dh"         bun:"p256dh,type:TEXT,notnull"`
	Auth           string   `json:"auth"           bun:"auth,type:TEXT,notnull"`
	UserAgent      string   `json:"userAgent"      bun:"user_agent,type:VARCHAR(255),nullzero"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (p *PushSubscription) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("ups_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}
	return nil
}

func (s *PushSubscription) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.UserID, validation.Required.Error("User is required")),
		// The endpoint is the push service URL the browser handed us; anything
		// that is not an absolute URL could never be posted to.
		validation.Field(&s.Endpoint,
			validation.Required.Error("Endpoint is required"),
			is.URL.Error("Endpoint must be a valid URL"),
		),
		// Both keys are required to encrypt a payload for this subscription, so
		// one without the other produces a notification that cannot be sealed.
		validation.Field(&s.P256dh, validation.Required.Error("Encryption key is required")),
		validation.Field(&s.Auth, validation.Required.Error("Auth secret is required")),
		validation.Field(&s.UserAgent,
			validation.Length(0, maxPushUserAgentLength).
				Error("User agent cannot be longer than 255 characters"),
		),
	))
}

const maxPushUserAgentLength = 255
