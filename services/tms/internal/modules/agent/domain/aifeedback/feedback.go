package aifeedback

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
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
	keel.Table `keel:"ai_feedback"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:aifb_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	UserID     pulid.ID   `json:"userId" keel:"user_id"`
	TargetType TargetType `json:"targetType" keel:"target_type"`
	TargetID   pulid.ID   `json:"targetId" keel:"target_id"`
	TargetPart string     `json:"targetPart" keel:"target_part"`

	ThreadID *pulid.ID `json:"threadId" keel:"thread_id"`
	TurnID   *pulid.ID `json:"turnId" keel:"turn_id"`
	RunID    *pulid.ID `json:"runId" keel:"run_id"`

	AgentDefinitionID *pulid.ID         `json:"agentDefinitionId" keel:"agent_definition_id"`
	DefinitionVersion *int64            `json:"definitionVersion" keel:"definition_version"`
	DetectorKey       string            `json:"detectorKey" keel:"detector_key,nullzero"`
	Task              string            `json:"task" keel:"task,nullzero"`
	Model             string            `json:"model" keel:"model,nullzero"`
	ProviderID        *pulid.ID         `json:"providerId" keel:"provider_id"`
	PromptHash        string            `json:"promptHash" keel:"prompt_hash,nullzero"`
	ToolSpecHash      string            `json:"toolSpecHash" keel:"tool_spec_hash,nullzero"`
	FingerprintSource FingerprintSource `json:"fingerprintSource" keel:"fingerprint_source"`

	Rating  Rating   `json:"rating" keel:"rating"`
	Reasons []Reason `json:"reasons" keel:"reasons"`
	Comment string   `json:"comment" keel:"comment,nullzero"`

	TurnSnapshot *TurnSnapshot `json:"turnSnapshot" keel:"turn_snapshot,jsonb"`
	PatternKey   string        `json:"patternKey" keel:"pattern_key"`
	EvalCaseID   *pulid.ID     `json:"evalCaseId" keel:"eval_case_id"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
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
			validation.RuneLength(0, MaxCommentRunes).
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

func (f *Feedback) BeforeCreate(_ context.Context) error {
	if f.Reasons == nil {
		f.Reasons = []Reason{}
	}

	return nil
}
