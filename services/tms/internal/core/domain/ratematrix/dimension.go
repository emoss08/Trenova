package ratematrix

import (
	"context"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateMatrixDimension)(nil)

// RateMatrixDimension is one axis of a matrix.
//
// Position is the contract between a dimension and the cells beneath it: a
// dimension at position two is read from and written to a cell's third pair of
// columns. That is why position is validated to be dense and unique rather than
// merely present.
type RateMatrixDimension struct {
	bun.BaseModel `bun:"table:rate_matrix_dimensions,alias:rmd" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateMatrixID   pulid.ID `json:"rateMatrixId"   bun:"rate_matrix_id,type:VARCHAR(100),notnull"`

	Position  int16         `json:"position"  bun:"position,type:SMALLINT,notnull"`
	Kind      DimensionKind `json:"kind"      bun:"kind,type:rate_matrix_dimension_kind_enum,notnull"`
	MatchMode MatchMode     `json:"matchMode" bun:"match_mode,type:rate_matrix_match_mode_enum,notnull"`
	Label     string        `json:"label"     bun:"label,type:VARCHAR(100),nullzero"`

	KeyNormalization KeyNormalization `json:"keyNormalization" bun:"key_normalization,type:rate_matrix_key_normalization_enum,notnull,default:'None'"`
	RangeOverflow    RangeOverflow    `json:"rangeOverflow"    bun:"range_overflow,type:rate_matrix_range_overflow_enum,notnull,default:'Error'"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	RateMatrix *RateMatrix `json:"-" bun:"rel:belongs-to,join:rate_matrix_id=id"`
}

func (rmd *RateMatrixDimension) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(rmd,
		validation.Field(&rmd.Position,
			validation.Min(int16(0)).Error("Position cannot be negative"),
			validation.Max(int16(MaxDimensions-1)).Error("Position cannot exceed three"),
		),
		validation.Field(&rmd.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[DimensionKind]("Kind is invalid"),
		),
		validation.Field(&rmd.MatchMode,
			validation.Required.Error("Match mode is required"),
			domainvalidation.ValidEnum[MatchMode]("Match mode is invalid"),
		),
		validation.Field(&rmd.Label,
			validation.Length(0, 100).Error("Label cannot be longer than 100 characters"),
		),
		validation.Field(&rmd.KeyNormalization,
			validation.When(
				rmd.KeyNormalization != "",
				domainvalidation.ValidEnum[KeyNormalization]("Key normalization is invalid"),
			),
		),
		validation.Field(&rmd.RangeOverflow,
			validation.When(
				rmd.RangeOverflow != "",
				domainvalidation.ValidEnum[RangeOverflow]("Range overflow is invalid"),
			),
		),
	))

	if rmd.MatchMode != MatchModeExact && rmd.KeyNormalization != "" &&
		rmd.KeyNormalization != KeyNormalizationNone {
		multiErr.Add(
			"keyNormalization",
			errortypes.ErrInvalid,
			"Key normalization only applies to an axis matched by exact key",
		)
	}

	if rmd.MatchMode != MatchModeRange && rmd.RangeOverflow != "" &&
		rmd.RangeOverflow != RangeOverflowError {
		multiErr.Add(
			"rangeOverflow",
			errortypes.ErrInvalid,
			"Range overflow only applies to an axis matched by band",
		)
	}

	// Banding only means something for a quantity. A zone or a freight class
	// has no ordering to cut into ranges, and a cell keyed by a range on such
	// an axis could never be matched.
	if rmd.MatchMode == MatchModeRange && !rmd.Kind.IsNumeric() {
		multiErr.Add(
			"matchMode",
			errortypes.ErrInvalid,
			"Only a numeric dimension can be matched by range",
		)
	}
}

// DisplayLabel is the label if one was given, and the kind otherwise, so a
// matrix editor always has something to put in a column header.
func (rmd *RateMatrixDimension) DisplayLabel() string {
	if rmd.Label != "" {
		return rmd.Label
	}

	return rmd.Kind.String()
}

func (rmd *RateMatrixDimension) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	rmd.applyModeDefaults()

	switch query.(type) {
	case *bun.InsertQuery:
		if rmd.ID.IsNil() {
			rmd.ID = pulid.MustNew("rmd_")
		}
		rmd.CreatedAt = now
		rmd.UpdatedAt = now
	case *bun.UpdateQuery:
		rmd.UpdatedAt = now
	}

	return nil
}

func (rmd *RateMatrixDimension) applyModeDefaults() {
	if rmd.KeyNormalization == "" {
		rmd.KeyNormalization = KeyNormalizationNone
	}
	if rmd.RangeOverflow == "" {
		rmd.RangeOverflow = RangeOverflowError
	}
}

func (rmd *RateMatrixDimension) GetID() pulid.ID {
	return rmd.ID
}

func (rmd *RateMatrixDimension) GetOrganizationID() pulid.ID {
	return rmd.OrganizationID
}

func (rmd *RateMatrixDimension) GetBusinessUnitID() pulid.ID {
	return rmd.BusinessUnitID
}

func (rmd *RateMatrixDimension) GetTableName() string {
	return "rate_matrix_dimensions"
}
