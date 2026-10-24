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
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
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

	DraftVersion int64 = 0
)

type RateMatrix struct {
	bun.BaseModel `bun:"table:rate_matrices,alias:rmx" json:"-"`
	keel.Table    `                                             keel:"rate_matrices"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"               keel:"id,pk,autoid:rmx_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"  keel:"organization_id,pk,tenant"`

	Code        string                 `json:"code"        bun:"code,type:VARCHAR(64),notnull"                                    keel:"code"`
	Name        string                 `json:"name"        bun:"name,type:VARCHAR(100),notnull"                                   keel:"name"`
	Description string                 `json:"description" bun:"description,type:TEXT,nullzero"                                   keel:"description,nullzero"`
	ValueType   string                 `json:"valueType"   bun:"value_type,type:VARCHAR(32),notnull"                              keel:"value_type"`
	Status      formulatemplate.Status `json:"status"      bun:"status,type:formula_template_status_enum,notnull,default:'Draft'" keel:"status"`

	SubmittedByID          *pulid.ID  `json:"submittedById"          bun:"submitted_by_id,type:VARCHAR(100)"                      keel:"submitted_by_id"`
	SubmittedAt            *int64     `json:"submittedAt"            bun:"submitted_at,type:BIGINT"                               keel:"submitted_at"`
	ApprovedByID           *pulid.ID  `json:"approvedById"           bun:"approved_by_id,type:VARCHAR(100)"                       keel:"approved_by_id"`
	ApprovedAt             *int64     `json:"approvedAt"             bun:"approved_at,type:BIGINT"                                keel:"approved_at"`
	ReviewComment          string     `json:"reviewComment"          bun:"review_comment,type:TEXT,nullzero"                      keel:"review_comment,nullzero"`
	PublishedVersionNumber int64      `json:"publishedVersionNumber" bun:"published_version_number,type:BIGINT,notnull,default:0" keel:"published_version_number"`
	PublishedAt            *int64     `json:"publishedAt"            bun:"published_at,type:BIGINT"                               keel:"published_at"`
	HasDraftChanges        bool       `json:"hasDraftChanges"        bun:"has_draft_changes,type:BOOLEAN,notnull"                 keel:"has_draft_changes"`
	DraftCellCount         int64      `json:"draftCellCount"         bun:"draft_cell_count,type:BIGINT,notnull,default:0"         keel:"draft_cell_count"`
	DraftEditorIDs         []pulid.ID `json:"draftEditorIds"         bun:"draft_editor_ids,type:JSONB,notnull,default:'[]'"       keel:"draft_editor_ids,jsonb"`

	Version      int64  `json:"version"   bun:"version,type:BIGINT"                                                                  keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"                                                 keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"                                                      keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit   `json:"-"                    bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization   `json:"-"                    bun:"rel:belongs-to,join:organization_id=id"`
	Dimensions   []*RateMatrixDimension `json:"dimensions,omitempty" bun:"rel:has-many,join:id=rate_matrix_id"     keel:"many:rate_matrix_id"`
}

// applyDefaults fills in the values the database would have defaulted, so a
// payload that omits them validates the same way it will be stored.
func (rm *RateMatrix) applyDefaults() {
	if rm.Status == "" {
		rm.Status = formulatemplate.StatusDraft
	}
	if rm.DraftEditorIDs == nil {
		rm.DraftEditorIDs = []pulid.ID{}
	}
	rm.ValueType = strings.TrimSpace(rm.ValueType)
}

func (rm *RateMatrix) IsPublished() bool {
	return rm.PublishedVersionNumber > 0
}

func (rm *RateMatrix) IsServed() bool {
	return rm.IsPublished() && rm.Status != formulatemplate.StatusInactive
}

func (rm *RateMatrix) RecordEditor(userID pulid.ID) {
	if userID.IsNil() || slices.Contains(rm.DraftEditorIDs, userID) {
		return
	}
	rm.DraftEditorIDs = append(slices.Clone(rm.DraftEditorIDs), userID)
}

func (rm *RateMatrix) EditedBy(userID pulid.ID) bool {
	return slices.Contains(rm.DraftEditorIDs, userID)
}

func (rm *RateMatrix) ClearApproval() {
	rm.SubmittedByID = nil
	rm.SubmittedAt = nil
	rm.ApprovedByID = nil
	rm.ApprovedAt = nil
	rm.ReviewComment = ""
}

func (rm *RateMatrix) HasMaterialChange(other *RateMatrix) bool {
	if other == nil {
		return true
	}
	if rm.ValueType != other.ValueType || rm.Code != other.Code ||
		len(rm.Dimensions) != len(other.Dimensions) {
		return true
	}
	mine, theirs := rm.OrderedDimensions(), other.OrderedDimensions()
	if len(mine) != len(theirs) {
		return true
	}
	for i := range mine {
		if !mine[i].SameShape(theirs[i]) {
			return true
		}
	}
	return false
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
		validation.Field(&rm.ValueType,
			validation.Required.Error("Value type is required"),
			validation.By(validateValueType),
		),
		validation.Field(&rm.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				formulatemplate.StatusDraft,
				formulatemplate.StatusInReview,
				formulatemplate.StatusActive,
				formulatemplate.StatusInactive,
			).Error("Status is invalid"),
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
		dimension.VersionNumber = DraftVersion
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

	if rm.DraftEditorIDs == nil {
		rm.DraftEditorIDs = []pulid.ID{}
	}

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

func (rm *RateMatrix) BeforeCreate(ctx context.Context) error {
	return rm.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (rm *RateMatrix) BeforeSave(_ context.Context) error {
	if rm.DraftEditorIDs == nil {
		rm.DraftEditorIDs = []pulid.ID{}
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
