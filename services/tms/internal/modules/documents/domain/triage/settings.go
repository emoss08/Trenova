package triage

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	bun.BaseModel `bun:"table:triage_response_targets,alias:ttgt" json:"-"`
	keel.Table    `keel:"triage_response_targets"`

	OrganizationID pulid.ID                      `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID                      `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	Classification inboundmessage.Classification `json:"classification" bun:"classification,pk,type:VARCHAR(30),notnull" keel:"classification,pk"`
	Minutes        int                           `json:"minutes"        bun:"minutes,type:INTEGER,notnull" keel:"minutes"`
	UpdatedByID    pulid.ID                      `json:"updatedById"    bun:"updated_by_id,type:VARCHAR(100),nullzero" keel:"updated_by_id,nullzero"`
	CreatedAt      int64                         `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64                         `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (t *ResponseTarget) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
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
	bun.BaseModel `bun:"table:triage_settings,alias:tset" json:"-"`
	keel.Table    `keel:"triage_settings"`

	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OnCallUserID   pulid.ID `json:"onCallUserId"   bun:"on_call_user_id,type:VARCHAR(100),nullzero" keel:"on_call_user_id,nullzero"`
	EndOfDayMinute int      `json:"endOfDayMinute" bun:"end_of_day_minute,type:INTEGER,notnull,default:1020" keel:"end_of_day_minute"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (s *Settings) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		s.CreatedAt = now
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}

type ReplyTemplate struct {
	bun.BaseModel `bun:"table:triage_reply_templates,alias:ttpl" json:"-"`
	keel.Table    `keel:"triage_reply_templates"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:ttpl_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	Name           string   `json:"name"           bun:"name,type:VARCHAR(100),notnull" keel:"name"`
	Body           string   `json:"body"           bun:"body,type:TEXT,notnull" keel:"body"`
	CreatedByID    pulid.ID `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero" keel:"created_by_id,nullzero"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (t *ReplyTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	t.Name = stringutils.StripNUL(t.Name)
	t.Body = stringutils.StripNUL(t.Body)
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("ttpl_")
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}

func (t *ReplyTemplate) BeforeCreate(ctx context.Context) error {
	return t.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (t *ReplyTemplate) GetID() pulid.ID      { return t.ID }
func (t *ReplyTemplate) GetCreatedAt() int64  { return t.CreatedAt }
func (t *ReplyTemplate) GetTableName() string { return "triage_reply_templates" }

type View struct {
	bun.BaseModel `bun:"table:triage_views,alias:tview" json:"-"`
	keel.Table    `keel:"triage_views"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:tview_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull" keel:"user_id"`
	Name           string   `json:"name"           bun:"name,type:VARCHAR(80),notnull" keel:"name"`
	Query          string   `json:"query"          bun:"query,type:VARCHAR(200),notnull" keel:"query"`
	Position       int      `json:"position"       bun:"position,type:INTEGER,notnull,default:0" keel:"position"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (v *View) BeforeAppendModel(_ context.Context, query bun.Query) error {
	v.Name = stringutils.StripNUL(v.Name)
	v.Query = stringutils.StripNUL(v.Query)
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("tview_")
		}
		v.CreatedAt = now
		v.UpdatedAt = now
	case *bun.UpdateQuery:
		v.UpdatedAt = now
	}

	return nil
}

func (v *View) BeforeCreate(ctx context.Context) error {
	return v.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (v *View) GetID() pulid.ID      { return v.ID }
func (v *View) GetCreatedAt() int64  { return v.CreatedAt }
func (v *View) GetTableName() string { return "triage_views" }

type SenderHint struct {
	bun.BaseModel `bun:"table:inbound_sender_hints,alias:ishint" json:"-"`
	keel.Table    `keel:"inbound_sender_hints"`

	OrganizationID  pulid.ID                      `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID  pulid.ID                      `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	SenderDomain    string                        `json:"senderDomain"    bun:"sender_domain,pk,type:VARCHAR(255),notnull" keel:"sender_domain,pk"`
	Classification  inboundmessage.Classification `json:"classification"  bun:"classification,pk,type:VARCHAR(30),notnull" keel:"classification,pk"`
	Corrections     int                           `json:"corrections"     bun:"corrections,type:INTEGER,notnull,default:0" keel:"corrections"`
	LastCorrectedAt int64                         `json:"lastCorrectedAt" bun:"last_corrected_at,type:BIGINT,notnull" keel:"last_corrected_at"`
}
