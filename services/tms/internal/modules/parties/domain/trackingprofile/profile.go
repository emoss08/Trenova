package trackingprofile

import (
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/mailtext"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	DefaultLinkTTLDays      = int16(30)
	MinLinkTTLDays          = int16(1)
	MaxLinkTTLDays          = int16(180)
	MaxRecipients           = 20
	MaxRecipientsLength     = 2000
	MaxRecipientEmailLength = 320
)

type Profile struct {
	keel.Table `keel:"customer_tracking_profiles"`

	ID                     pulid.ID `json:"id"                     keel:"id,pk,autoid:ctp_"`
	OrganizationID         pulid.ID `json:"organizationId"         keel:"organization_id,pk,tenant"`
	BusinessUnitID         pulid.ID `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	CustomerID             pulid.ID `json:"customerId"             keel:"customer_id"`
	LinkEnabled            bool     `json:"linkEnabled"            keel:"link_enabled"`
	VisibleFields          []Field  `json:"visibleFields"          keel:"visible_fields"`
	LinkTTLDays            int16    `json:"linkTtlDays"            keel:"link_ttl_days"`
	NotifyPickedUp         bool     `json:"notifyPickedUp"         keel:"notify_picked_up"`
	NotifyDelayed          bool     `json:"notifyDelayed"          keel:"notify_delayed"`
	NotifyDelivered        bool     `json:"notifyDelivered"        keel:"notify_delivered"`
	NotifyPodReady         bool     `json:"notifyPodReady"         keel:"notify_pod_ready"`
	IncludeLink            bool     `json:"includeLink"            keel:"include_link"`
	Recipients             string   `json:"recipients"             keel:"recipients,nullzero"`
	NotificationsEnabledAt *int64   `json:"notificationsEnabledAt" keel:"notifications_enabled_at"`
	Version                int64    `json:"version"                keel:"version,version"`
	CreatedAt              int64    `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64    `json:"updatedAt"              keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func NewDefault(orgID, buID, customerID pulid.ID) *Profile {
	return &Profile{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CustomerID:     customerID,
		LinkEnabled:    true,
		VisibleFields:  DefaultFields(),
		LinkTTLDays:    DefaultLinkTTLDays,
		IncludeLink:    true,
	}
}

func (p *Profile) GetID() string {
	return p.ID.String()
}

func (p *Profile) GetTableName() string {
	return "customer_tracking_profiles"
}

func (p *Profile) GetOrganizationID() pulid.ID {
	return p.OrganizationID
}

func (p *Profile) GetBusinessUnitID() pulid.ID {
	return p.BusinessUnitID
}

func (p *Profile) Shows(field Field) bool {
	return p != nil && slices.Contains(p.VisibleFields, field)
}

func (p *Profile) NotifiesAny() bool {
	return p != nil &&
		(p.NotifyPickedUp || p.NotifyDelayed || p.NotifyDelivered || p.NotifyPodReady)
}

func (p *Profile) LinkTTLSeconds() int64 {
	days := p.LinkTTLDays
	if days < MinLinkTTLDays || days > MaxLinkTTLDays {
		days = DefaultLinkTTLDays
	}
	return int64(days) * timeutils.SecondsPerDay
}

func (p *Profile) RecipientList() []string {
	return mailtext.SplitEmailList(p.Recipients)
}

// Normalize puts the profile in its canonical form before it is validated:
// fields deduplicated in their declared order and recipients normalized.
func (p *Profile) Normalize() {
	p.VisibleFields = canonicalFields(p.VisibleFields)
	if recipients := p.RecipientList(); len(recipients) > 0 {
		p.Recipients = strings.Join(recipients, ", ")
	} else {
		p.Recipients = ""
	}
	if p.LinkTTLDays == 0 {
		p.LinkTTLDays = DefaultLinkTTLDays
	}
}

// StampNotifications records when status emails were switched on, carried
// from the stored profile, so turning a rule on never reaches back to events
// that already happened.
func (p *Profile) StampNotifications(previous *Profile, now int64) {
	switch {
	case !p.NotifiesAny():
		p.NotificationsEnabledAt = nil
	case previous == nil || previous.NotificationsEnabledAt == nil:
		p.NotificationsEnabledAt = &now
	case p.switchedOnSince(previous):
		p.NotificationsEnabledAt = &now
	default:
		enabledAt := *previous.NotificationsEnabledAt
		p.NotificationsEnabledAt = &enabledAt
	}
}

func (p *Profile) switchedOnSince(previous *Profile) bool {
	return (p.NotifyPickedUp && !previous.NotifyPickedUp) ||
		(p.NotifyDelayed && !previous.NotifyDelayed) ||
		(p.NotifyDelivered && !previous.NotifyDelivered) ||
		(p.NotifyPodReady && !previous.NotifyPodReady)
}

func canonicalFields(fields []Field) []Field {
	result := make([]Field, 0, len(fields))
	for _, candidate := range FieldValues() {
		if slices.Contains(fields, candidate) {
			result = append(result, candidate)
		}
	}
	return result
}

func (p *Profile) Validate(multiErr *errortypes.MultiError) {
	p.Normalize()

	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.LinkTTLDays,
			validation.Min(MinLinkTTLDays).
				Error("A tracking link must last at least 1 day"),
			validation.Max(MaxLinkTTLDays).
				Error("A tracking link can last at most 180 days"),
		),
		validation.Field(&p.Recipients,
			validation.RuneLength(0, MaxRecipientsLength).
				Error("Recipients cannot be longer than 2000 characters"),
		),
	))
	recipients := p.RecipientList()
	if len(recipients) > MaxRecipients {
		multiErr.Add("recipients", errortypes.ErrInvalid, "List at most 20 recipients")
	}
	for _, recipient := range recipients {
		if err := validation.Validate(recipient,
			validation.RuneLength(1, MaxRecipientEmailLength),
			is.EmailFormat,
		); err != nil {
			multiErr.Add("recipients", errortypes.ErrInvalid,
				"Each recipient must be a valid email address")
			break
		}
	}
}

func (Profile) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_customer_tracking_profiles_customer",
			Message: "Customer already has tracking settings",
		},
		{
			Name:    "fk_customer_tracking_profiles_customer",
			Message: "Customer does not exist in your organization",
		},
	}
}
