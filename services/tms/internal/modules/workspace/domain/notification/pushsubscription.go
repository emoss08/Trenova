package notification

import (
	"context"
	"net/url"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/urlutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*PushSubscription)(nil)

type PushSubscription struct {
	bun.BaseModel `bun:"table:user_push_subscriptions,alias:ups" json:"-"`
	keel.Table    `keel:"user_push_subscriptions"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:ups_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull" keel:"user_id"`
	Endpoint       string   `json:"endpoint"       bun:"endpoint,type:TEXT,notnull" keel:"endpoint"`
	P256dh         string   `json:"p256dh"         bun:"p256dh,type:TEXT,notnull" keel:"p256dh"`
	Auth           string   `json:"auth"           bun:"auth,type:TEXT,notnull" keel:"auth"`
	UserAgent      string   `json:"userAgent"      bun:"user_agent,type:VARCHAR(255),nullzero" keel:"user_agent,nullzero"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (s *PushSubscription) ValidateEndpointHost(
	multiErr *errortypes.MultiError,
	allowedHosts []string,
) {
	if !PushEndpointAllowed(s.Endpoint, allowedHosts) {
		multiErr.Add(
			"endpoint",
			errortypes.ErrInvalid,
			"Endpoint must be an https address of a supported push service",
		)
	}
}

func PushEndpointAllowed(endpoint string, allowedHosts []string) bool {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Opaque != "" {
		return false
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return false
	}
	return urlutils.HostMatches(parsed.Hostname(), allowedHosts)
}
