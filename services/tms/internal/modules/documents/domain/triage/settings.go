package triage

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MaxTemplateNameLength  = 100
	MaxTemplateBodyLength  = 10000
	MaxViewNameLength      = 80
	MaxViewQueryLength     = 200
	MinTargetMinutes       = 5
	MaxTargetMinutes       = 30 * 24 * 60
	DefaultEndOfDayMinute  = 17 * 60
	MinutesPerDay          = 24 * 60
	MaxViewsPerUser        = 50
	MaxTemplatesPerTenant  = 200
	MaxRulesPerTenant      = 500
	DefaultHandOffBatchMax = 5000
)

var defaultTargetMinutes = map[inboundmessage.Classification]int{
	inboundmessage.ClassificationTender:           120,
	inboundmessage.ClassificationStatusRequest:    60,
	inboundmessage.ClassificationRateConfirmation: 120,
	inboundmessage.ClassificationProofOfDelivery:  480,
	inboundmessage.ClassificationInvoice:          1440,
	inboundmessage.ClassificationDetentionDispute: 1440,
	inboundmessage.ClassificationFactorRemittance: 1440,
	inboundmessage.ClassificationOther:            2880,
}

func DefaultTargetMinutes(class inboundmessage.Classification) int {
	if minutes, ok := defaultTargetMinutes[class]; ok {
		return minutes
	}

	return defaultTargetMinutes[inboundmessage.ClassificationOther]
}

type ResponseTarget struct {
	keel.Table `keel:"triage_response_targets"`

	OrganizationID pulid.ID                      `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID                      `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Classification inboundmessage.Classification `json:"classification" keel:"classification,pk"`
	Minutes        int                           `json:"minutes" keel:"minutes"`
	UpdatedByID    pulid.ID                      `json:"updatedById" keel:"updated_by_id,nullzero"`
	CreatedAt      int64                         `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64                         `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (t *ResponseTarget) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.Classification,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[inboundmessage.Classification]("Kind is not a known one"),
		),
		validation.Field(&t.Minutes,
			validation.Required.Error("Target is required"),
			validation.Min(MinTargetMinutes).Error("A target is at least five minutes"),
			validation.Max(MaxTargetMinutes).Error("A target is at most thirty days"),
		),
	))
}

type Targets map[inboundmessage.Classification]int

func ResolveTargets(rows []*ResponseTarget) Targets {
	targets := make(Targets, len(defaultTargetMinutes))
	for class, minutes := range defaultTargetMinutes {
		targets[class] = minutes
	}
	for _, row := range rows {
		targets[row.Classification] = row.Minutes
	}

	return targets
}

func (t Targets) For(class inboundmessage.Classification) int {
	if minutes, ok := t[class]; ok {
		return minutes
	}

	return DefaultTargetMinutes(class)
}

type Settings struct {
	keel.Table `keel:"triage_settings"`

	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OnCallUserID   pulid.ID `json:"onCallUserId" keel:"on_call_user_id,nullzero"`
	EndOfDayMinute int      `json:"endOfDayMinute" keel:"end_of_day_minute"`
	Version        int64    `json:"version" keel:"version,version"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func DefaultSettings(orgID, buID pulid.ID) *Settings {
	return &Settings{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		EndOfDayMinute: DefaultEndOfDayMinute,
	}
}

func (s *Settings) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.EndOfDayMinute,
			validation.Min(0).Error("End of day must be a time of day"),
			validation.Max(MinutesPerDay-1).Error("End of day must be a time of day"),
		),
	))
}

func (s *Settings) BeforeCreate(_ context.Context) error {
	if s.EndOfDayMinute == 0 {
		s.EndOfDayMinute = DefaultEndOfDayMinute
	}

	return nil
}

type ReplyTemplate struct {
	keel.Table `keel:"triage_reply_templates"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:ttpl_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	Name           string   `json:"name" keel:"name"`
	Body           string   `json:"body" keel:"body"`
	CreatedByID    pulid.ID `json:"createdById" keel:"created_by_id,nullzero"`
	Version        int64    `json:"version" keel:"version,version"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (t *ReplyTemplate) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.RuneLength(1, MaxTemplateNameLength),
		),
		validation.Field(&t.Body,
			validation.Required.Error("A template needs some text"),
			validation.RuneLength(1, MaxTemplateBodyLength),
		),
	))
}

func (t *ReplyTemplate) BeforeCreate(ctx context.Context) error {
	t.Name = stringutils.StripNUL(t.Name)
	t.Body = stringutils.StripNUL(t.Body)

	return nil
}

func (t *ReplyTemplate) BeforeUpdate(ctx context.Context) error {
	t.Name = stringutils.StripNUL(t.Name)
	t.Body = stringutils.StripNUL(t.Body)

	return nil
}

func (t *ReplyTemplate) GetID() pulid.ID      { return t.ID }
func (t *ReplyTemplate) GetCreatedAt() int64  { return t.CreatedAt }
func (t *ReplyTemplate) GetTableName() string { return "triage_reply_templates" }

type View struct {
	keel.Table `keel:"triage_views"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:tview_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	UserID         pulid.ID `json:"userId" keel:"user_id"`
	Name           string   `json:"name" keel:"name"`
	Query          string   `json:"query" keel:"query"`
	Position       int      `json:"position" keel:"position"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (v *View) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.Name,
			validation.Required.Error("Name is required"),
			validation.RuneLength(1, MaxViewNameLength),
		),
		validation.Field(&v.Query,
			validation.Required.Error("A view needs words to search for"),
			validation.RuneLength(1, MaxViewQueryLength),
		),
	))
}

func (v *View) BeforeCreate(ctx context.Context) error {
	v.Name = stringutils.StripNUL(v.Name)
	v.Query = stringutils.StripNUL(v.Query)

	return nil
}

func (v *View) BeforeUpdate(ctx context.Context) error {
	v.Name = stringutils.StripNUL(v.Name)
	v.Query = stringutils.StripNUL(v.Query)

	return nil
}

func (v *View) GetID() pulid.ID      { return v.ID }
func (v *View) GetCreatedAt() int64  { return v.CreatedAt }
func (v *View) GetTableName() string { return "triage_views" }

type SenderHint struct {
	keel.Table `keel:"inbound_sender_hints"`

	OrganizationID  pulid.ID                      `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID  pulid.ID                      `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	SenderDomain    string                        `json:"senderDomain" keel:"sender_domain,pk"`
	Classification  inboundmessage.Classification `json:"classification" keel:"classification,pk"`
	Corrections     int                           `json:"corrections" keel:"corrections"`
	LastCorrectedAt int64                         `json:"lastCorrectedAt" keel:"last_corrected_at"`
}
