// Package ratematrix stores tariffs that vary along more than one axis.
//
// A rate table answers "what does this one number map to". A tariff usually
// asks something wider: what does Chicago to Atlanta cost at four thousand
// pounds on a reefer. Each of those is an axis, and the answer lives at their
// intersection.
//
// Cells are looked up one at a time through an index. A matrix is never loaded
// whole — a class tariff runs to hundreds of thousands of cells, and rating now
// happens on every shipment write.
package ratematrix

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*RateMatrix)(nil)
	_ validationframework.TenantedEntity = (*RateMatrix)(nil)
	_ domaintypes.PostgresSearchable     = (*RateMatrix)(nil)
)

const (
	maxCodeLength        = 64
	maxNameLength        = 100
	maxDescriptionLength = 500
	maxRoundingPrecision = 6
)

type RateMatrix struct {
	bun.BaseModel             `bun:"table:rate_matrices,alias:rmx" json:"-"`
	pagination.CursorValueSet `bun:",embed"                        json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Code        string             `json:"code"        bun:"code,type:VARCHAR(64),notnull"`
	Name        string             `json:"name"        bun:"name,type:VARCHAR(100),notnull"`
	Description string             `json:"description" bun:"description,type:TEXT,nullzero"`
	Currency    string             `json:"currency"    bun:"currency,type:VARCHAR(3),notnull,default:'USD'"`
	Status      domaintypes.Status `json:"status"      bun:"status,type:status_enum,notnull,default:'Active'"`

	// FormulaTemplateID says what a cell's number means. The template prices the
	// shipment with the matched cell's value bound as its base rate, so the same
	// grid is a per-mile tariff or a flat table depending on which template the
	// matrix names.
	FormulaTemplateID pulid.ID `json:"formulaTemplateId" bun:"formula_template_id,type:VARCHAR(100),notnull"`

	RoundingMode      ratetypes.RoundingMode `json:"roundingMode"      bun:"rounding_mode,type:rate_rounding_mode_enum,notnull,default:'HalfUp'"`
	RoundingPrecision int16                  `json:"roundingPrecision" bun:"rounding_precision,type:SMALLINT,notnull,default:2"`

	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit    *tenant.BusinessUnit             `json:"-"                         bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization    *tenant.Organization             `json:"-"                         bun:"rel:belongs-to,join:organization_id=id"`
	FormulaTemplate *formulatemplate.FormulaTemplate `json:"formulaTemplate,omitempty" bun:"rel:belongs-to,join:formula_template_id=id"`
	Dimensions      []*RateMatrixDimension           `json:"dimensions,omitempty"      bun:"rel:has-many,join:id=rate_matrix_id"`
}

// applyDefaults fills in the values the database would have defaulted, so a
// payload that omits them validates the same way it will be stored.
func (rm *RateMatrix) applyDefaults() {
	if rm.Status == "" {
		rm.Status = domaintypes.StatusActive
	}
	if rm.Currency == "" {
		rm.Currency = money.DefaultCurrencyCode
	}
	if rm.RoundingMode == "" {
		rm.RoundingMode = ratetypes.RoundingModeHalfUp
	}
}

func (rm *RateMatrix) Validate(multiErr *errortypes.MultiError) {
	rm.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(rm,
		validation.Field(&rm.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, maxCodeLength).
				Error("Code must be between 1 and 64 characters"),
			validation.Match(domaintypes.IdentifierPattern).
				Error("Code must start with a letter and contain only letters, digits, and underscores"),
		),
		validation.Field(&rm.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxNameLength).
				Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&rm.Description,
			validation.Length(0, maxDescriptionLength).
				Error("Description cannot be longer than 500 characters"),
		),
		validation.Field(&rm.FormulaTemplateID,
			validation.Required.Error("Formula template is required"),
		),
		validation.Field(&rm.Currency,
			validation.Required.Error("Currency is required"),
			validation.Length(3, 3).Error("Currency must be a three letter code"),
		),
		validation.Field(&rm.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status is invalid"),
		),
		validation.Field(&rm.RoundingMode,
			validation.Required.Error("Rounding mode is required"),
			domainvalidation.ValidEnum[ratetypes.RoundingMode]("Rounding mode is invalid"),
		),
		validation.Field(&rm.RoundingPrecision,
			validation.Min(int16(0)).Error("Rounding precision cannot be negative"),
			validation.Max(int16(maxRoundingPrecision)).
				Error("Rounding precision cannot exceed 6"),
		),
	))

	rm.validateDimensions(multiErr)
}

// StampDimensions re-seats on every axis the values it inherits from its
// matrix: tenancy and its parent. A payload describes the axes, never their
// ownership.
func (rm *RateMatrix) StampDimensions() {
	for _, dimension := range rm.Dimensions {
		if dimension == nil {
			continue
		}

		dimension.RateMatrixID = rm.ID
		dimension.OrganizationID = rm.OrganizationID
		dimension.BusinessUnitID = rm.BusinessUnitID
	}
}

func (rm *RateMatrix) validateDimensions(multiErr *errortypes.MultiError) {
	rm.StampDimensions()

	if len(rm.Dimensions) == 0 {
		multiErr.Add(
			"dimensions",
			errortypes.ErrRequired,
			"A matrix must have at least one dimension",
		)
		return
	}

	if len(rm.Dimensions) > MaxDimensions {
		multiErr.Add(
			"dimensions",
			errortypes.ErrInvalid,
			"A matrix cannot have more than four dimensions",
		)
		return
	}

	positions := make(map[int16]struct{}, len(rm.Dimensions))

	for i, dimension := range rm.Dimensions {
		if dimension == nil {
			continue
		}

		dimensionErr := multiErr.WithIndex("dimensions", i)
		dimension.Validate(dimensionErr)

		// Position is what binds a dimension to a column on every cell. Two
		// dimensions claiming the same slot would silently make one of them
		// unreachable, and every cell keyed against it would stop matching.
		if _, dup := positions[dimension.Position]; dup {
			dimensionErr.Add(
				"position",
				errortypes.ErrDuplicate,
				"Two dimensions cannot share a position",
			)
		}
		positions[dimension.Position] = struct{}{}
	}

	for position := range int16(len(rm.Dimensions)) { //nolint:gosec // bounded by MaxDimensions
		if _, ok := positions[position]; !ok {
			multiErr.Add(
				"dimensions",
				errortypes.ErrInvalid,
				"Dimension positions must run from zero without gaps",
			)
			break
		}
	}
}

// OrderedDimensions returns the dimensions by position, which is the order a
// caller must supply lookup values in.
func (rm *RateMatrix) OrderedDimensions() []*RateMatrixDimension {
	ordered := make([]*RateMatrixDimension, MaxDimensions)

	for _, dimension := range rm.Dimensions {
		if dimension == nil || dimension.Position < 0 || dimension.Position >= MaxDimensions {
			continue
		}
		ordered[dimension.Position] = dimension
	}

	trimmed := make([]*RateMatrixDimension, 0, MaxDimensions)
	for _, dimension := range ordered {
		if dimension == nil {
			break
		}
		trimmed = append(trimmed, dimension)
	}

	return trimmed
}

func (rm *RateMatrix) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rm.ID.IsNil() {
			rm.ID = pulid.MustNew("rmx_")
		}
		rm.CreatedAt = now
		rm.UpdatedAt = now
	case *bun.UpdateQuery:
		rm.UpdatedAt = now
	}

	return nil
}

func (rm *RateMatrix) GetID() pulid.ID {
	return rm.ID
}

func (rm *RateMatrix) GetCreatedAt() int64 {
	return rm.CreatedAt
}

func (rm *RateMatrix) GetOrganizationID() pulid.ID {
	return rm.OrganizationID
}

func (rm *RateMatrix) GetBusinessUnitID() pulid.ID {
	return rm.BusinessUnitID
}

func (rm *RateMatrix) GetTableName() string {
	return "rate_matrices"
}

func (rm *RateMatrix) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "rmx",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}
