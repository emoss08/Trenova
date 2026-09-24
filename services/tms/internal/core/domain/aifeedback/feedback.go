package aifeedback

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Feedback)(nil)
	_ validationframework.TenantedEntity = (*Feedback)(nil)
	_ pagination.CursorEntity            = (*Feedback)(nil)
	_ domaintypes.PostgresSearchable     = (*Feedback)(nil)
)

const (
	MaxCommentRunes    = 1000
	MaxTargetPartRunes = 100
	PatternKeyLength   = 64
)

type Feedback struct {
	bun.BaseModel `bun:"table:ai_feedback,alias:aifb" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	UserID     pulid.ID   `json:"userId"     bun:"user_id,type:VARCHAR(100),notnull"`
	TargetType TargetType `json:"targetType" bun:"target_type,type:VARCHAR(30),notnull"`
	TargetID   pulid.ID   `json:"targetId"   bun:"target_id,type:VARCHAR(100),notnull"`
	TargetPart string     `json:"targetPart" bun:"target_part,type:VARCHAR(100),notnull,default:''"`

	ThreadID *pulid.ID `json:"threadId" bun:"thread_id,type:VARCHAR(100),nullzero"`
	TurnID   *pulid.ID `json:"turnId"   bun:"turn_id,type:VARCHAR(100),nullzero"`
	RunID    *pulid.ID `json:"runId"    bun:"run_id,type:VARCHAR(100),nullzero"`

	AgentDefinitionID *pulid.ID         `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	DefinitionVersion *int64            `json:"definitionVersion" bun:"definition_version,type:BIGINT,nullzero"`
	DetectorKey       string            `json:"detectorKey"       bun:"detector_key,type:VARCHAR(100),nullzero"`
	Task              string            `json:"task"              bun:"task,type:VARCHAR(100),nullzero"`
	Model             string            `json:"model"             bun:"model,type:VARCHAR(255),nullzero"`
	ProviderID        *pulid.ID         `json:"providerId"        bun:"provider_id,type:VARCHAR(100),nullzero"`
	PromptHash        string            `json:"promptHash"        bun:"prompt_hash,type:VARCHAR(64),nullzero"`
	ToolSpecHash      string            `json:"toolSpecHash"      bun:"tool_spec_hash,type:VARCHAR(64),nullzero"`
	FingerprintSource FingerprintSource `json:"fingerprintSource" bun:"fingerprint_source,type:VARCHAR(20),notnull"`

	Rating  Rating   `json:"rating"  bun:"rating,type:SMALLINT,notnull"`
	Reasons []Reason `json:"reasons" bun:"reasons,type:TEXT[],array,notnull,default:'{}'"`
	Comment string   `json:"comment" bun:"comment,type:TEXT,nullzero"`

	TurnSnapshot *TurnSnapshot `json:"turnSnapshot" bun:"turn_snapshot,type:JSONB,nullzero"`
	PatternKey   string        `json:"patternKey"   bun:"pattern_key,type:VARCHAR(64),notnull"`
	EvalCaseID   *pulid.ID     `json:"evalCaseId"   bun:"eval_case_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (f *Feedback) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(f,
		validation.Field(&f.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&f.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&f.UserID, validation.Required.Error("User is required")),
		validation.Field(&f.TargetType,
			validation.Required.Error("Target type is required"),
			domainvalidation.ValidEnum[TargetType]("Target type is invalid"),
		),
		validation.Field(&f.TargetID, validation.Required.Error("Target is required")),
		validation.Field(&f.TargetPart,
			validation.Length(0, MaxTargetPartRunes).
				Error(fmt.Sprintf("Target part must be at most %d characters", MaxTargetPartRunes)),
		),
		validation.Field(&f.FingerprintSource,
			validation.Required.Error("Fingerprint source is required"),
			domainvalidation.ValidEnum[FingerprintSource]("Fingerprint source is invalid"),
		),
		validation.Field(&f.Comment,
			validation.Length(0, MaxCommentRunes).
				Error(fmt.Sprintf("Comment must be at most %d characters", MaxCommentRunes)),
		),
		validation.Field(&f.PatternKey,
			validation.Required.Error("Pattern key is required"),
			validation.Length(PatternKeyLength, PatternKeyLength).Error("Pattern key is invalid"),
		),
	))

	if !f.Rating.IsValid() {
		multiErr.Add("rating", errortypes.ErrInvalid, "Rating must be a thumbs up or a thumbs down")
	}

	switch {
	case f.TargetType.RequiresPart() && strings.TrimSpace(f.TargetPart) == "":
		multiErr.Add("targetPart", errortypes.ErrRequired, "Name the part of the target rated")
	case !f.TargetType.RequiresPart() && f.TargetPart != "":
		multiErr.Add("targetPart", errortypes.ErrInvalid, "This target has no parts to rate")
	}

	f.validateReasons(multiErr)
}

func (f *Feedback) validateReasons(multiErr *errortypes.MultiError) {
	seen := make(map[Reason]struct{}, len(f.Reasons))
	for index, reason := range f.Reasons {
		field := fmt.Sprintf("reasons[%d]", index)
		if !reason.IsValid() {
			multiErr.Add(field, errortypes.ErrInvalid, "Reason is invalid")
			continue
		}
		if f.Rating.IsValid() && !reason.MatchesRating(f.Rating) {
			multiErr.Add(field, errortypes.ErrInvalid, "Reason does not match the rating")
			continue
		}
		if _, dup := seen[reason]; dup {
			multiErr.Add(field, errortypes.ErrInvalid, "Reason is listed twice")
			continue
		}
		seen[reason] = struct{}{}
	}
}

func (f *Feedback) FirstReason() Reason {
	if len(f.Reasons) == 0 {
		return ""
	}

	return f.Reasons[0]
}

func (f *Feedback) Negative() bool { return f.Rating == RatingNegative }

func (f *Feedback) GetID() pulid.ID { return f.ID }

func (f *Feedback) GetCreatedAt() int64 { return f.CreatedAt }

func (f *Feedback) GetOrganizationID() pulid.ID { return f.OrganizationID }

func (f *Feedback) GetBusinessUnitID() pulid.ID { return f.BusinessUnitID }

func (f *Feedback) GetTableName() string { return "ai_feedback" }

func (f *Feedback) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "aifb",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "comment", Type: domaintypes.FieldTypeText},
			{Name: "model", Type: domaintypes.FieldTypeText},
			{Name: "task", Type: domaintypes.FieldTypeText},
			{Name: "target_type", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (f *Feedback) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if f.ID.IsNil() {
			f.ID = pulid.MustNew("aifb_")
		}
		if f.CreatedAt == 0 {
			f.CreatedAt = now
		}
		f.UpdatedAt = now
	case *bun.UpdateQuery:
		f.UpdatedAt = now
	}

	return nil
}
