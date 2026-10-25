package ratematrix

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// RateMatrixDimension is one axis of a matrix.
//
// Position is the contract between a dimension and the cells beneath it: a
// dimension at position two is read from and written to a cell's third pair of
// columns. That is why position is validated to be dense and unique rather than
// merely present.
type RateMatrixDimension struct {
	keel.Table `keel:"rate_matrix_dimensions"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:rmd_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	RateMatrixID   pulid.ID `json:"rateMatrixId"   keel:"rate_matrix_id"`
	VersionNumber  int64    `json:"versionNumber"  keel:"version_number"`

	Position  int16         `json:"position"  keel:"position"`
	Kind      DimensionKind `json:"kind"      keel:"kind"`
	MatchMode MatchMode     `json:"matchMode" keel:"match_mode"`
	Label     string        `json:"label"     keel:"label,nullzero"`

	KeyNormalization KeyNormalization `json:"keyNormalization" keel:"key_normalization"`
	RangeOverflow    RangeOverflow    `json:"rangeOverflow"    keel:"range_overflow"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	RateMatrix *RateMatrix `json:"-" keel:"belongs:rate_matrix_id"`
}

func (rmd *RateMatrixDimension) BeforeUpdate(_ context.Context) error {
	rmd.applyModeDefaults()

	return nil
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

	if rmd.Kind.IsPostalRange() {
		rmd.validatePostalRange(multiErr)
		return
	}

	// Banding only means something for a quantity. A zone or a freight class
	// has no ordering to cut into ranges, and a cell keyed by a range on such
	// an axis could never be matched.
	if rmd.MatchMode == MatchModeRange && !rmd.Kind.IsNumeric() {
		multiErr.Add(
			"matchMode",
			errortypes.ErrInvalid,
			"Only a numeric dimension or a postal range can be matched by range",
		)
	}
}

func (rmd *RateMatrixDimension) validatePostalRange(multiErr *errortypes.MultiError) {
	if rmd.MatchMode != MatchModeRange {
		multiErr.Add(
			"matchMode",
			errortypes.ErrInvalid,
			"A postal range axis is matched by range",
		)
	}
	if rmd.RangeOverflow != "" && rmd.RangeOverflow != RangeOverflowError {
		multiErr.Add(
			"rangeOverflow",
			errortypes.ErrInvalid,
			"A postal range axis has no overflow policy: a postal code no range holds finds no rate",
		)
	}
}

func (rmd *RateMatrixDimension) SameShape(other *RateMatrixDimension) bool {
	if other == nil {
		return false
	}
	mine, theirs := *rmd, *other
	mine.applyModeDefaults()
	theirs.applyModeDefaults()
	return mine.Position == theirs.Position &&
		mine.Kind == theirs.Kind &&
		mine.MatchMode == theirs.MatchMode &&
		mine.Label == theirs.Label &&
		mine.KeyNormalization == theirs.KeyNormalization &&
		mine.RangeOverflow == theirs.RangeOverflow
}

func (rmd *RateMatrixDimension) IsPostalRange() bool {
	return rmd.Kind.IsPostalRange()
}

// DisplayLabel is the label if one was given, and the kind otherwise, so a
// matrix editor always has something to put in a column header.
func (rmd *RateMatrixDimension) DisplayLabel() string {
	if rmd.Label != "" {
		return rmd.Label
	}

	return rmd.Kind.String()
}

func (rmd *RateMatrixDimension) BeforeCreate(ctx context.Context) error {
	if rmd.KeyNormalization == "" {
		rmd.KeyNormalization = KeyNormalizationNone
	}
	if rmd.RangeOverflow == "" {
		rmd.RangeOverflow = RangeOverflowError
	}
	rmd.applyModeDefaults()

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
