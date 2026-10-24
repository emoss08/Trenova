package extractioneval

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*ExtractionResult)(nil)
	_ pagination.CursorEntity        = (*ExtractionResult)(nil)
	_ domaintypes.PostgresSearchable = (*ExtractionResult)(nil)
)

type ExtractionResult struct {
	bun.BaseModel `bun:"table:extraction_eval_results,alias:eeres" json:"-"`
	keel.Table    `keel:"extraction_eval_results"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:eeres_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`

	RunID     pulid.ID     `json:"runId"     bun:"run_id,type:VARCHAR(100),notnull" keel:"run_id"`
	CaseID    pulid.ID     `json:"caseId"    bun:"case_id,type:VARCHAR(100),notnull" keel:"case_id"`
	CaseTitle string       `json:"caseTitle" bun:"case_title,type:VARCHAR(200),notnull" keel:"case_title"`
	Ordinal   int          `json:"ordinal"   bun:"ordinal,type:INTEGER,notnull" keel:"ordinal"`
	Status    ResultStatus `json:"status"    bun:"status,type:VARCHAR(20),notnull,default:'Pending'" keel:"status"`

	Model        string                     `json:"model"        bun:"model,type:VARCHAR(255),nullzero" keel:"model,nullzero"`
	ProviderID   *pulid.ID                  `json:"providerId"   bun:"provider_id,type:VARCHAR(100),nullzero" keel:"provider_id"`
	Predicted    *aicorrection.Snapshot     `json:"predicted"    bun:"predicted,type:JSONB,nullzero" keel:"predicted,jsonb"`
	FieldResults []aicorrection.FieldResult `json:"fieldResults" bun:"field_results,type:JSONB,notnull,default:'[]'" keel:"field_results,jsonb"`

	ScoredCount      int     `json:"scoredCount"      bun:"scored_count,type:INTEGER,notnull,default:0" keel:"scored_count"`
	CorrectCount     int     `json:"correctCount"     bun:"correct_count,type:INTEGER,notnull,default:0" keel:"correct_count"`
	CorrectedCount   int     `json:"correctedCount"   bun:"corrected_count,type:INTEGER,notnull,default:0" keel:"corrected_count"`
	MissedCount      int     `json:"missedCount"      bun:"missed_count,type:INTEGER,notnull,default:0" keel:"missed_count"`
	UnconfirmedCount int     `json:"unconfirmedCount" bun:"unconfirmed_count,type:INTEGER,notnull,default:0" keel:"unconfirmed_count"`
	UnscoredCount    int     `json:"unscoredCount"    bun:"unscored_count,type:INTEGER,notnull,default:0" keel:"unscored_count"`
	Accuracy         float64 `json:"accuracy"         bun:"accuracy,type:DOUBLE PRECISION,notnull,default:0" keel:"accuracy"`

	LatencyMs    int64           `json:"latencyMs"    bun:"latency_ms,type:BIGINT,notnull,default:0" keel:"latency_ms"`
	InputTokens  int64           `json:"inputTokens"  bun:"input_tokens,type:BIGINT,notnull,default:0" keel:"input_tokens"`
	OutputTokens int64           `json:"outputTokens" bun:"output_tokens,type:BIGINT,notnull,default:0" keel:"output_tokens"`
	CostUSD      decimal.Decimal `json:"costUsd"      bun:"cost_usd,type:NUMERIC(14,6),notnull,default:0" keel:"cost_usd"`
	ErrorMessage string          `json:"errorMessage" bun:"error_message,type:TEXT,nullzero" keel:"error_message,nullzero"`
	StartedAt    *int64          `json:"startedAt"    bun:"started_at,type:BIGINT,nullzero" keel:"started_at"`
	CompletedAt  *int64          `json:"completedAt"  bun:"completed_at,type:BIGINT,nullzero" keel:"completed_at"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (r *ExtractionResult) ApplyScore(results []aicorrection.FieldResult) {
	r.FieldResults = results
	t := aicorrection.TallyResults(results)
	r.ScoredCount = t.Scored
	r.CorrectCount = t.Correct
	r.CorrectedCount = t.Corrected
	r.MissedCount = t.Missed
	r.UnconfirmedCount = t.Unconfirmed
	r.UnscoredCount = t.Unscored
	r.Accuracy = aicorrection.Accuracy(t.Correct, t.Scored)
}

func (r *ExtractionResult) GetID() pulid.ID { return r.ID }

func (r *ExtractionResult) GetCreatedAt() int64 { return r.CreatedAt }

func (r *ExtractionResult) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *ExtractionResult) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *ExtractionResult) GetTableName() string { return "extraction_eval_results" }

func (r *ExtractionResult) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "eeres",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "case_title", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (r *ExtractionResult) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("eeres_")
		}
		if r.CreatedAt == 0 {
			r.CreatedAt = now
		}
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}
