package ailog

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*AILog)(nil)
	_ domaintypes.PostgresSearchable = (*AILog)(nil)
)

type AILog struct {
	bun.BaseModel `bun:"table:ai_logs,alias:ailog"`

	ID               pulid.ID  `json:"id"               bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID   pulid.ID  `json:"organizationId"   bun:"organization_id,type:VARCHAR(100)"`
	BusinessUnitID   pulid.ID  `json:"businessUnitId"   bun:"business_unit_id,type:VARCHAR(100)"`
	UserID           pulid.ID  `json:"userId"           bun:"user_id,type:VARCHAR(100)"`
	Prompt           string    `json:"prompt"           bun:"prompt,type:TEXT"`
	Operation        Operation `json:"operation"        bun:"operation,type:operation_enum"`
	Response         string    `json:"response"         bun:"response,type:TEXT"`
	Model            Model     `json:"model"            bun:"model,type:model_enum"`
	Object           string    `json:"object"           bun:"object,type:VARCHAR(100)"`
	ServiceTier      string    `json:"serviceTier"      bun:"service_tier,type:VARCHAR(100)"`
	PromptTokens     int64     `json:"promptTokens"     bun:"prompt_tokens,type:INT"`
	CompletionTokens int64     `json:"completionTokens" bun:"completion_tokens,type:INT"`
	SearchVector     string    `json:"-"                bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank             string    `json:"-"                bun:"rank,type:VARCHAR(100),scanonly"`
	TotalTokens      int64     `json:"totalTokens"      bun:"total_tokens,type:INT"`
	ReasoningTokens  int64     `json:"reasoningTokens"  bun:"reasoning_tokens,type:INT"`
	Timestamp        int64     `json:"timestamp"        bun:"timestamp,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
	User         *tenant.User         `bun:"rel:belongs-to,join:user_id=id"          json:"user,omitempty"`
}

func (al *AILog) GetTableName() string {
	return "ai_logs"
}

func (al *AILog) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ailog",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "prompt", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "operation", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{Name: "response", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{Name: "model", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
		Relationships: []*domaintypes.RelationshipDefinition{
			{
				Field:        "User",
				Type:         domaintypes.RelationshipTypeBelongsTo,
				TargetTable:  "users",
				ForeignKey:   "user_id",
				ReferenceKey: "id",
				Alias:        "u",
				Queryable:    true,
				TargetEntity: (*tenant.User)(nil),
			},
		},
	}
}

func (al *AILog) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	if _, ok := query.(*bun.InsertQuery); ok {
		if al.ID.IsNil() {
			al.ID = pulid.MustNew("ailog_")
		}

		al.Timestamp = now
	}

	return nil
}
