// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package documentqualityconfig

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/pretrainedmodels"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*DocumentQualityConfig)(nil)
	_ domain.Validatable        = (*DocumentQualityConfig)(nil)
)

type DocumentQualityConfig struct {
	bun.BaseModel `bun:"table:document_quality_configs,alias:dqc" json:"-"`

	ID                pulid.ID `json:"id"                bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID    pulid.ID `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	IsActive          bool     `json:"isActive"          bun:"is_active,type:BOOLEAN,notnull,default:true"`
	MinWordCount      int      `json:"minWordCount"      bun:"min_word_count,type:INTEGER,notnull,default:50"`
	MinDPI            int      `json:"minDpi"            bun:"min_dpi,type:INTEGER,notnull,default:200"`
	MinBrightness     float64  `json:"minBrightness"     bun:"min_brightness,type:NUMERIC(5,2),notnull,default:40"`
	MaxBrightness     float64  `json:"maxBrightness"     bun:"max_brightness,type:NUMERIC(5,2),notnull,default:220"`
	MinContrast       float64  `json:"minContrast"       bun:"min_contrast,type:NUMERIC(5,2),notnull,default:40"`
	MinSharpness      float64  `json:"minSharpness"      bun:"min_sharpness,type:NUMERIC(5,2),notnull,default:50"`
	AutoRejectScore   float64  `json:"autoRejectScore"   bun:"auto_reject_score,type:NUMERIC(3,2),notnull,default:0.2"`
	ManualReviewScore float64  `json:"manualReviewScore" bun:"manual_review_score,type:NUMERIC(3,2),notnull,default:0.4"`
	MinConfidence     float64  `json:"minConfidence"     bun:"min_confidence,type:NUMERIC(3,2),notnull,default:0.7"`
	MinTextDensity    float64  `json:"minTextDensity"    bun:"min_text_density,type:NUMERIC(5,2),notnull,default:0.1"`
	ModelID           pulid.ID `json:"modelId"           bun:"model_id,type:VARCHAR(100),notnull"`
	AllowTraining     bool     `json:"allowTraining"     bun:"allow_training,type:BOOLEAN,notnull,default:true"`
	Version           int64    `json:"version"           bun:"version,type:BIGINT"`
	CreatedAt         int64    `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64    `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	Model *pretrainedmodels.PretrainedModel `json:"model,omitempty" bun:"rel:belongs-to,join:model_id=id"`
}

func (c *DocumentQualityConfig) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, c,
		// Ensure minimum DPI is set
		validation.Field(&c.MinDPI,
			validation.Required.Error("Min DPI is required"),
			validation.Min(1).Error("Min DPI must be at least 1"),
		),

		// Ensure minimum brightness is set and within range
		validation.Field(&c.MinBrightness,
			validation.Required.Error("Min Brightness is required"),
			validation.Min(40).Error("Min Brightness must be at least 40"),
			validation.Max(220).Error("Min Brightness must be at most 220"),
		),

		// Ensure maximum brightness is set and within range
		validation.Field(&c.MaxBrightness,
			validation.Required.Error("Max Brightness is required"),
			validation.Min(40).Error("Max Brightness must be at least 40"),
			validation.Max(220).Error("Max Brightness must be at most 220"),
		),

		// Ensure minimum contrast is set and within range
		validation.Field(&c.MinContrast,
			validation.Required.Error("Min Contrast is required"),
			validation.Min(1).Error("Min Contrast must be at least 1"),
		),

		// Ensure minimum sharpness is set
		validation.Field(&c.MinSharpness,
			validation.Required.Error("Min Sharpness is required"),
			validation.Min(1).Error("Min Sharpness must be at least 1"),
		),

		// Ensure minimum word count is set
		validation.Field(&c.MinWordCount,
			validation.Required.Error("Min Word Count is required"),
			validation.Min(1).Error("Min Word Count must be at least 1"),
		),

		// Ensure minimum text density is set
		validation.Field(&c.MinTextDensity,
			validation.Required.Error("Min Text Density is required"),
			validation.Min(0.01).Error("Min Text Density must be at least 0.01"),
		),

		// Ensure model ID is set
		validation.Field(&c.ModelID,
			validation.Required.Error("Model ID is required"),
		),

		// Ensure allow training is set
		validation.Field(&c.AllowTraining,
			validation.Required.Error("Allow Training is required"),
			validation.In(true, false).Error("Allow Training must be either true or false"),
		),

		// Ensure auto reject score is set and within range
		validation.Field(&c.AutoRejectScore,
			validation.Required.Error("Auto Reject Score is required"),
			validation.Min(0.01).Error("Auto Reject Score must be at least 0.01"),
			validation.Max(0.99).Error("Auto Reject Score must be at most 0.99"),
		),

		// Ensure manual review score is set and within range
		validation.Field(&c.ManualReviewScore,
			validation.Required.Error("Manual Review Score is required"),
			validation.Min(0.01).Error("Manual Review Score must be at least 0.01"),
			validation.Max(0.99).Error("Manual Review Score must be at most 0.99"),
		),

		// Ensure minimum confidence is set and within range
		validation.Field(&c.MinConfidence,
			validation.Required.Error("Min Confidence is required"),
			validation.Min(0.01).Error("Min Confidence must be at least 0.01"),
			validation.Max(0.99).Error("Min Confidence must be at most 0.99"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (c *DocumentQualityConfig) DBValidate(ctx context.Context, tx bun.IDB) *errors.MultiError {
	multiErr := errors.NewMultiError()

	c.Validate(ctx, multiErr)
	c.ValidateUniqueness(ctx, tx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (c *DocumentQualityConfig) ValidateUniqueness(
	ctx context.Context,
	tx bun.IDB,
	multiErr *errors.MultiError,
) {
	vb := queryutils.NewUniquenessValidator(c.GetTableName()).
		WithTenant(c.BusinessUnitID, c.OrganizationID).
		WithModelName("Document Quality Config").
		WithFieldAndTemplate("organization_id", c.OrganizationID.String(),
			"Document Quality Config with organization ID ':value' already exists in the business unit. An Organization can only have one Document Quality Config.",
			map[string]string{
				"value": c.OrganizationID.String(),
			})

	if c.ID.IsNotNil() {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", c.ID.String())
	} else {
		vb.WithOperation(queryutils.OperationCreate)
	}

	queryutils.CheckFieldUniqueness(ctx, tx, vb.Build(), multiErr)
}

func (c *DocumentQualityConfig) GetID() string {
	return c.ID.String()
}

func (c *DocumentQualityConfig) GetTableName() string {
	return "document_quality_configs"
}

// Misc
func (c *DocumentQualityConfig) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("dqc_")
		}

		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
