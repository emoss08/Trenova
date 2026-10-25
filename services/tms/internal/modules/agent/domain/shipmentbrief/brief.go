package shipmentbrief

import (
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	LateReason      string `json:"lateReason,omitempty"`
	OpenSuggestions int    `json:"openSuggestions"`
	Timezone        string `json:"timezone"`
}

func (f Facts) OpenIssues() int {
	return f.Late + f.Uncovered + f.OpenSuggestions
}

type Brief struct {
	keel.Table `keel:"shipment_board_briefs"`

	ID              pulid.ID           `json:"id"              keel:"id,pk,autoid:sbb_"`
	BusinessUnitID  pulid.ID           `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID           `json:"organizationId"  keel:"organization_id,pk,tenant"`
	BriefDate       string             `json:"briefDate"       keel:"brief_date"`
	Generation      int                `json:"generation"      keel:"generation"`
	Trigger         Trigger            `json:"trigger"         keel:"trigger"`
	Segments        []Segment          `json:"segments"        keel:"segments,jsonb"`
	Wording         map[string]Wording `json:"wording"         keel:"wording,jsonb"`
	Facts           Facts              `json:"facts"           keel:"facts,jsonb"`
	OpenIssues      int                `json:"openIssues"      keel:"open_issues"`
	Narrated        bool               `json:"narrated"        keel:"narrated"`
	ModelIdentifier string             `json:"modelIdentifier" keel:"model_identifier,nullzero"`
	GeneratedAt     int64              `json:"generatedAt"     keel:"generated_at"`
	CreatedAt       int64              `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64              `json:"updatedAt"       keel:"updated_at,autoupdate"`
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
