package pretrainedmodels

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*PretrainedModel)(nil)

type PretrainedModel struct {
	bun.BaseModel `bun:"table:pretrained_models,alias:pm" json:"-"`

	// Primary identifiers
	ID      pulid.ID  `json:"id" bun:"id,pk,type:VARCHAR(100)"`
	Name    string    `json:"name" bun:"name,type:VARCHAR(100),notnull"`
	Version string    `json:"version" bun:"version,type:VARCHAR(50),notnull"`
	Type    ModelType `json:"type" bun:"type,type:model_type_enum,notnull"`

	// Model Details
	Description string      `json:"description" bun:"description,type:TEXT,notnull"`
	Status      ModelStatus `json:"status" bun:"status,type:model_status_enum,notnull,default:'Stable'"`
	Path        string      `json:"path" bun:"path,type:VARCHAR(255),notnull"`
	IsDefault   bool        `json:"isDefault" bun:"is_default,type:BOOLEAN,notnull,default:false"`
	IsActive    bool        `json:"isActive" bun:"is_active,type:BOOLEAN,notnull,default:true"`

	// Training Info
	TrainedAt       int64          `json:"trainedAt" bun:"trained_at,nullzero,notnull"`
	TrainingMetrics map[string]any `json:"trainingMetrics" bun:"training_metrics,type:JSONB,default:'{}'"`

	// Metadata
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

// Misc
func (pm *PretrainedModel) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if pm.ID.IsNil() {
			pm.ID = pulid.MustNew("pm_")
		}

		pm.CreatedAt = now
	case *bun.UpdateQuery:
		pm.UpdatedAt = now
	}

	return nil
}
