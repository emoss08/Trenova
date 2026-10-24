package customertracking

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Profile struct {
	keel.Table `keel:"customer_tracking_profiles"`

	ID                     pulid.ID `json:"id" keel:"id,pk"`
	OrganizationID         pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID         pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	CustomerID             pulid.ID `json:"customerId" keel:"customer_id"`
	LinkEnabled            bool     `json:"linkEnabled" keel:"link_enabled"`
	VisibleFields          []Field  `json:"visibleFields" keel:"visible_fields"`
	LinkTTLDays            int16    `json:"linkTtlDays" keel:"link_ttl_days"`
	NotifyPickedUp         bool     `json:"notifyPickedUp" keel:"notify_picked_up"`
	NotifyDelayed          bool     `json:"notifyDelayed" keel:"notify_delayed"`
	NotifyDelivered        bool     `json:"notifyDelivered" keel:"notify_delivered"`
	NotifyPodReady         bool     `json:"notifyPodReady" keel:"notify_pod_ready"`
	IncludeLink            bool     `json:"includeLink" keel:"include_link"`
	Recipients             string   `json:"recipients" keel:"recipients,nullzero"`
	NotificationsEnabledAt *int64   `json:"notificationsEnabledAt" keel:"notifications_enabled_at"`
	Version                int64    `json:"version" keel:"version,version"`
	CreatedAt              int64    `json:"createdAt" keel:"created_at"`
	UpdatedAt              int64    `json:"updatedAt" keel:"updated_at"`
}

func (p *Profile) BeforeCreate(_ context.Context) error {
	if p.LinkTTLDays == 0 {
		p.LinkTTLDays = 30
	}

	return nil
}

func NewProfileID() pulid.ID {
	return pulid.MustNew("ctp_")
}

func DefaultProfile(orgID, buID, customerID pulid.ID) *Profile {
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

func (p *Profile) Shows(field Field) bool {
	return p != nil && slices.Contains(p.VisibleFields, field)
}

func (p *Profile) Notifies(event Event) bool {
	if p == nil {
		return false
	}
	switch event {
	case EventPickedUp:
		return p.NotifyPickedUp
	case EventDelayed:
		return p.NotifyDelayed
	case EventDelivered:
		return p.NotifyDelivered
	case EventPodReady:
		return p.NotifyPodReady
	default:
		return false
	}
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
	return int64(days) * SecondsPerDay
}

func (p *Profile) RecipientList() []string {
	return stringutils.SplitEmailList(p.Recipients)
}

// Normalize puts the profile in its canonical form before it is validated and
// stored: fields deduplicated in their declared order, recipients normalized,
// and the instant notifications were switched on stamped from the previous
// version, so turning a rule on never reaches back to events that already
// happened.
func (p *Profile) Normalize(previous *Profile, now int64) {
	p.VisibleFields = canonicalFields(p.VisibleFields)
	if recipients := p.RecipientList(); len(recipients) > 0 {
		p.Recipients = strings.Join(recipients, ", ")
	} else {
		p.Recipients = ""
	}
	if p.LinkTTLDays == 0 {
		p.LinkTTLDays = DefaultLinkTTLDays
	}
	switch {
	case !p.NotifiesAny():
		p.NotificationsEnabledAt = nil
	case previous == nil || previous.NotificationsEnabledAt == nil:
		p.NotificationsEnabledAt = &now
	default:
		p.NotificationsEnabledAt = previous.NotificationsEnabledAt
		for _, event := range EventValues() {
			if p.Notifies(event) && !previous.Notifies(event) {
				p.NotificationsEnabledAt = &now
				break
			}
		}
	}
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
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&p.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&p.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(&p.LinkTTLDays,
			validation.Min(MinLinkTTLDays).Error("A tracking link must last at least 1 day"),
			validation.Max(MaxLinkTTLDays).Error("A tracking link can last at most 180 days"),
		),
		validation.Field(&p.Recipients,
			validation.RuneLength(0, MaxRecipientsLength).
				Error("Recipients cannot be longer than 2000 characters"),
		),
	))
	for i, field := range p.VisibleFields {
		if !field.IsValid() {
			multiErr.WithIndex("visibleFields", i).
				Add("", errortypes.ErrInvalid, "Visible field is not one Trenova knows")
		}
	}
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
