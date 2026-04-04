package ailog

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type Model string

const (
	ModelGPT5Nano         Model = "gpt-5-nano"
	ModelGPT5Nano20250807 Model = "gpt-5-nano-2025-08-07"
	ModelGPT5Mini         Model = "gpt-5-mini"
	ModelGPT5Mini20250807 Model = "gpt-5-mini-2025-08-07"
	ModelModerationLatest Model = "omni-moderation-latest"
)

type Operation string

const (
	OperationClassifyLocation            Operation = "ClassifyLocation"
	OperationDocumentIntelligenceRoute   Operation = "DocumentIntelligenceRoute"
	OperationDocumentIntelligenceExtract Operation = "DocumentIntelligenceExtract"
	OperationShipmentImportChat          Operation = "ShipmentImportChat"
)

type Log struct {
	bun.BaseModel `bun:"table:ai_logs,alias:ail" json:"-"`

	ID               pulid.ID  `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID   pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID   pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	UserID           pulid.ID  `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Prompt           string    `json:"prompt"         bun:"prompt,type:TEXT,notnull"`
	Response         string    `json:"response"       bun:"response,type:TEXT,notnull"`
	Model            Model     `json:"model"          bun:"model,type:model_enum,notnull"`
	Operation        Operation `json:"operation"      bun:"operation,type:operation_enum,notnull"`
	Object           string    `json:"object"         bun:"object,type:VARCHAR(100),notnull"`
	ServiceTier      string    `json:"serviceTier"    bun:"service_tier,type:VARCHAR(100),notnull"`
	PromptTokens     int       `json:"promptTokens"   bun:"prompt_tokens,notnull"`
	CompletionTokens int       `json:"completionTokens" bun:"completion_tokens,notnull"`
	TotalTokens      int       `json:"totalTokens"    bun:"total_tokens,notnull"`
	ReasoningTokens  int       `json:"reasoningTokens" bun:"reasoning_tokens,notnull"`
	SearchVector     string    `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Timestamp        int64     `json:"timestamp"      bun:"timestamp,notnull,default:extract(epoch from current_timestamp)::bigint"`
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
