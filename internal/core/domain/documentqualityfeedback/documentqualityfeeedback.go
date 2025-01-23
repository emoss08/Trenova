package documentqualityfeedback

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*DocumentQualityFeedback)(nil)

type DocumentQualityFeedback struct {
	bun.BaseModel `bun:"table:document_quality_feedback,alias:dqf" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId" bun:"user_id,type:VARCHAR(100),notnull"`

	// Feedback Details
	DocumentURL  string       `json:"documentUrl" bun:"document_url,type:TEXT,notnull"`
	FeedbackType FeedbackType `json:"feedbackType" bun:"feedback_type,type:feedback_type_enum,notnull"`
	Comment      string       `json:"comment" bun:"comment,type:TEXT"`

	// Quality Metrics at Time of Assessment
	QualityScore    float64 `json:"qualityScore" bun:"quality_score,type:DECIMAL(5,2),notnull"`
	ConfidenceScore float64 `json:"confidenceScore" bun:"confidence_score,type:DECIMAL(5,2),notnull"`
	Sharpness       float64 `json:"sharpness" bun:"sharpness,type:DECIMAL(10,2),notnull"`
	TextDensity     float64 `json:"textDensity" bun:"text_density,type:DECIMAL(5,2),notnull"`
	WordCount       int     `json:"wordCount" bun:"word_count,type:INTEGER,notnull"`

	// Training Flags
	UsedForTraining bool   `json:"usedForTraining" bun:"used_for_training,type:BOOLEAN,notnull,default:false"`
	TrainedAt       *int64 `json:"trainedAt,omitempty" bun:"trained_at,type:BIGINT"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	User         *user.User                 `json:"user,omitempty" bun:"rel:belongs-to,join:user_id=id"`
}

func (f *DocumentQualityFeedback) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if f.ID.IsNil() {
			f.ID = pulid.MustNew("dqf_")
		}

		f.CreatedAt = now
	case *bun.UpdateQuery:
		f.UpdatedAt = now
	}

	return nil
}
