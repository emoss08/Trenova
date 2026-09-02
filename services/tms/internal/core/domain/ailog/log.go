package ailog

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Log struct {
	bun.BaseModel `bun:"table:ai_logs,alias:ail" json:"-"`

	ID               pulid.ID  `json:"id"               bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID   pulid.ID  `json:"organizationId"   bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID   pulid.ID  `json:"businessUnitId"   bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	UserID           pulid.ID  `json:"userId"           bun:"user_id,type:VARCHAR(100),notnull"`
	Prompt           string    `json:"prompt"           bun:"prompt,type:TEXT,notnull"`
	Response         string    `json:"response"         bun:"response,type:TEXT,notnull"`
	Model            Model     `json:"model"            bun:"model,type:model_enum,notnull"`
	Operation        Operation `json:"operation"        bun:"operation,type:operation_enum,notnull"`
	Object           string    `json:"object"           bun:"object,type:VARCHAR(100),notnull"`
	ServiceTier      string    `json:"serviceTier"      bun:"service_tier,type:VARCHAR(100),notnull"`
	PromptTokens     int       `json:"promptTokens"     bun:"prompt_tokens,notnull"`
	CompletionTokens int       `json:"completionTokens" bun:"completion_tokens,notnull"`
	TotalTokens      int       `json:"totalTokens"      bun:"total_tokens,notnull"`
	ReasoningTokens  int       `json:"reasoningTokens"  bun:"reasoning_tokens,notnull"`
	SearchVector     string    `json:"-"                bun:"search_vector,type:TSVECTOR,scanonly"`
	Timestamp        int64     `json:"timestamp"        bun:"timestamp,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (l *Log) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("ail_")
		}
		if l.Timestamp == 0 {
			l.Timestamp = timeutils.NowUnix()
		}
	}

	return nil
}

func (l *Log) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(l,
		validation.Field(&l.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&l.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&l.UserID, validation.Required.Error("User is required")),
		validation.Field(&l.Prompt, validation.Required.Error("Prompt is required")),
		validation.Field(&l.Model,
			validation.Required.Error("Model is required"),
			validation.In(
				ModelGPT5Nano,
				ModelGPT5Nano20250807,
				ModelGPT5Mini,
				ModelGPT5Mini20250807,
				ModelModerationLatest,
				ModelClaudeOpus5,
			).Error("Model is not one this system calls"),
		),
		validation.Field(&l.Operation,
			validation.Required.Error("Operation is required"),
			validation.In(
				OperationClassifyLocation,
				OperationDocumentIntelligenceRoute,
				OperationDocumentIntelligenceExtract,
				OperationShipmentImportChat,
				OperationFormulaGenerate,
				OperationFormulaExplain,
			).Error("Operation is not one this system records"),
		),
		validation.Field(&l.Object,
			validation.Required.Error("Object is required"),
			validation.Length(1, maxObjectLength).
				Error("Object cannot be longer than 100 characters"),
		),
		validation.Field(&l.ServiceTier,
			validation.Length(0, maxServiceTierLength).
				Error("Service tier cannot be longer than 100 characters"),
		),
		// Token counts drive spend reporting, so a negative one would understate
		// what an organization has actually used.
		validation.Field(&l.PromptTokens,
			validation.Min(0).Error("Prompt tokens cannot be negative"),
		),
		validation.Field(&l.CompletionTokens,
			validation.Min(0).Error("Completion tokens cannot be negative"),
		),
		validation.Field(&l.TotalTokens,
			validation.Min(0).Error("Total tokens cannot be negative"),
		),
		validation.Field(&l.ReasoningTokens,
			validation.Min(0).Error("Reasoning tokens cannot be negative"),
		),
	))
}

const (
	maxObjectLength      = 100
	maxServiceTierLength = 100
)
