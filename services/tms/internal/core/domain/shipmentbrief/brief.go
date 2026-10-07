package shipmentbrief

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	MaxSegments        = 16
	MaxSegmentLength   = 400
	MaxWordedItems     = 40
	maxWordingTitle    = 120
	maxWordingReason   = 240
	maxModelIdentifier = 255
)

type Trigger string

const (
	TriggerScheduled = Trigger("Scheduled")
	TriggerOnDemand  = Trigger("OnDemand")
	TriggerCleared   = Trigger("Cleared")
)

func Triggers() []Trigger {
	return []Trigger{TriggerScheduled, TriggerOnDemand, TriggerCleared}
}

func (t Trigger) IsValid() bool {
	return slices.Contains(Triggers(), t)
}

func (t Trigger) String() string { return string(t) }

type Segment struct {
	Text   string                `json:"text"`
	Filter *shipment.QuickFilter `json:"filter,omitempty"`
}

type Wording struct {
	Title  string `json:"title"`
	Reason string `json:"reason"`
	Basis  string `json:"basis"`
}

func WordingBasis(title, reason string, impact []string) string {
	var builder strings.Builder
	builder.WriteString(title)
	builder.WriteByte(0)
	builder.WriteString(reason)
	for _, fact := range impact {
		builder.WriteByte(0)
		builder.WriteString(fact)
	}
	return hashutils.SHA256Hex(builder.String())
}

func (w Wording) Fits(title, reason string, impact []string) bool {
	return w.Title != "" && w.Reason != "" && w.Basis == WordingBasis(title, reason, impact)
}

type Facts struct {
	DeliveringToday int    `json:"deliveringToday"`
	Moving          int    `json:"moving"`
	Late            int    `json:"late"`
	Uncovered       int    `json:"uncovered"`
	Detention       int    `json:"detention"`
	ReadyToBill     int    `json:"readyToBill"`
	LowMargin       int    `json:"lowMargin"`
	LateReason      string `json:"lateReason,omitempty"`
	OpenSuggestions int    `json:"openSuggestions"`
	Timezone        string `json:"timezone"`
}

func (f Facts) OpenIssues() int {
	return f.Late + f.Uncovered + f.OpenSuggestions
}

var _ bun.BeforeAppendModelHook = (*Brief)(nil)

type Brief struct {
	bun.BaseModel `bun:"table:shipment_board_briefs,alias:sbb" json:"-"`

	ID              pulid.ID           `json:"id"              bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID  pulid.ID           `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID  pulid.ID           `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BriefDate       string             `json:"briefDate"       bun:"brief_date,type:VARCHAR(10),notnull"`
	Generation      int                `json:"generation"      bun:"generation,type:INTEGER,notnull"`
	Trigger         Trigger            `json:"trigger"         bun:"trigger,type:VARCHAR(20),notnull"`
	Segments        []Segment          `json:"segments"        bun:"segments,type:JSONB,notnull,default:'[]'"`
	Wording         map[string]Wording `json:"wording"         bun:"wording,type:JSONB,notnull,default:'{}'"`
	Facts           Facts              `json:"facts"           bun:"facts,type:JSONB,notnull,default:'{}'"`
	OpenIssues      int                `json:"openIssues"      bun:"open_issues,type:INTEGER,notnull,default:0"`
	Narrated        bool               `json:"narrated"        bun:"narrated,type:BOOLEAN,notnull,default:false"`
	ModelIdentifier string             `json:"modelIdentifier" bun:"model_identifier,type:VARCHAR(255),nullzero"`
	GeneratedAt     int64              `json:"generatedAt"     bun:"generated_at,type:BIGINT,notnull"`
	CreatedAt       int64              `json:"createdAt"       bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt       int64              `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull"`
}

func (b *Brief) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("sbb_")
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

func (b *Brief) Normalize() {
	if len(b.Segments) > MaxSegments {
		b.Segments = b.Segments[:MaxSegments]
	}
	for index := range b.Segments {
		b.Segments[index].Text = stringutils.TruncateRunes(b.Segments[index].Text, MaxSegmentLength)
	}
	if b.Segments == nil {
		b.Segments = []Segment{}
	}
	if b.Wording == nil {
		b.Wording = map[string]Wording{}
	}
	for key, wording := range b.Wording {
		b.Wording[key] = Wording{
			Title:  stringutils.Ellipsize(strings.TrimSpace(wording.Title), maxWordingTitle),
			Reason: stringutils.Ellipsize(strings.TrimSpace(wording.Reason), maxWordingReason),
			Basis:  wording.Basis,
		}
	}
	b.ModelIdentifier = stringutils.Ellipsize(b.ModelIdentifier, maxModelIdentifier)
	b.OpenIssues = b.Facts.OpenIssues()
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
		validation.Field(&b.Wording,
			validation.Length(0, MaxWordedItems).Error("Too many worded suggestions"),
		),
		validation.Field(&b.GeneratedAt, validation.Required.Error("Generated at is required")),
	))
}
