package accountingbrief

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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

var _ bun.BeforeAppendModelHook = (*Brief)(nil)

type Brief struct {
	bun.BaseModel `bun:"table:accounting_briefs,alias:acb" json:"-"`

	ID              pulid.ID  `json:"id"              bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID  pulid.ID  `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID  pulid.ID  `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BriefDate       string    `json:"briefDate"       bun:"brief_date,type:VARCHAR(10),notnull"`
	Generation      int       `json:"generation"      bun:"generation,type:INTEGER,notnull"`
	Trigger         Trigger   `json:"trigger"         bun:"trigger,type:VARCHAR(20),notnull"`
	Segments        []Segment `json:"segments"        bun:"segments,type:JSONB,notnull,default:'[]'"`
	Facts           Facts     `json:"facts"           bun:"facts,type:JSONB,notnull,default:'{}'"`
	Narrated        bool      `json:"narrated"        bun:"narrated,type:BOOLEAN,notnull,default:false"`
	ModelIdentifier string    `json:"modelIdentifier" bun:"model_identifier,type:VARCHAR(255),nullzero"`
	GeneratedAt     int64     `json:"generatedAt"     bun:"generated_at,type:BIGINT,notnull"`
	VerifiedAt      int64     `json:"verifiedAt"      bun:"verified_at,type:BIGINT,notnull"`
	CreatedAt       int64     `json:"createdAt"       bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt       int64     `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull"`
}

func (b *Brief) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("acb_")
		}
		if b.CreatedAt == 0 {
			b.CreatedAt = now
		}
		b.UpdatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}

	return nil
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
