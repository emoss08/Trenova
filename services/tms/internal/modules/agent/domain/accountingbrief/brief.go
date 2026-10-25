package accountingbrief

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MaxSegments          = 16
	MaxSegmentLength     = 400
	maxModelIdentifier   = 255
	maxGenerationsPerDay = 8
)

type Trigger string

const (
	TriggerScheduled = Trigger("Scheduled")
	TriggerOnDemand  = Trigger("OnDemand")
	TriggerRefreshed = Trigger("Refreshed")
)

func Triggers() []Trigger {
	return []Trigger{TriggerScheduled, TriggerOnDemand, TriggerRefreshed}
}

func (t Trigger) IsValid() bool {
	return slices.Contains(Triggers(), t)
}

func (t Trigger) String() string { return string(t) }

type Segment struct {
	Text string `json:"text"`
	Link *Link  `json:"link,omitempty"`
}

func CanRefresh(generation int) bool {
	return generation < maxGenerationsPerDay
}

type Brief struct {
	keel.Table `keel:"accounting_briefs"`

	ID              pulid.ID  `json:"id"              keel:"id,pk,autoid:acb_"`
	BusinessUnitID  pulid.ID  `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID  `json:"organizationId"  keel:"organization_id,pk,tenant"`
	BriefDate       string    `json:"briefDate"       keel:"brief_date"`
	Generation      int       `json:"generation"      keel:"generation"`
	Trigger         Trigger   `json:"trigger"         keel:"trigger"`
	Segments        []Segment `json:"segments"        keel:"segments,jsonb"`
	Facts           Facts     `json:"facts"           keel:"facts,jsonb"`
	Narrated        bool      `json:"narrated"        keel:"narrated"`
	ModelIdentifier string    `json:"modelIdentifier" keel:"model_identifier,nullzero"`
	GeneratedAt     int64     `json:"generatedAt"     keel:"generated_at"`
	VerifiedAt      int64     `json:"verifiedAt"      keel:"verified_at"`
	CreatedAt       int64     `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64     `json:"updatedAt"       keel:"updated_at,autoupdate"`
}

func (b *Brief) CheckedAt() int64 {
	return max(b.GeneratedAt, b.VerifiedAt)
}

func (b *Brief) Normalize() {
	if len(b.Segments) > MaxSegments {
		b.Segments = b.Segments[:MaxSegments]
	}
	for index := range b.Segments {
		b.Segments[index].Text = stringutils.TruncateRunes(b.Segments[index].Text, MaxSegmentLength)
		if link := b.Segments[index].Link; link != nil && !link.IsValid() {
			b.Segments[index].Link = nil
		}
	}
	if b.Segments == nil {
		b.Segments = []Segment{}
	}
	b.Facts.normalize()
	b.ModelIdentifier = stringutils.Ellipsize(b.ModelIdentifier, maxModelIdentifier)
	if b.VerifiedAt == 0 {
		b.VerifiedAt = b.GeneratedAt
	}
}

func (b *Brief) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&b.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&b.BriefDate,
			validation.Required.Error("Brief date is required"),
			validation.Length(10, 10).Error("Brief date must be a calendar day"),
		),
		validation.Field(&b.Generation, validation.Min(1).Error("Generation starts at one")),
		validation.Field(&b.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[Trigger]("Trigger is invalid"),
		),
		validation.Field(&b.Segments,
			validation.Required.Error("A brief says something"),
			validation.Length(1, MaxSegments).Error("Too many segments"),
		),
		validation.Field(&b.GeneratedAt, validation.Required.Error("Generated at is required")),
	))
}
