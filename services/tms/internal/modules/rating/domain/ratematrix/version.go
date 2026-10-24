package ratematrix

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateMatrixVersion)(nil)

type VersionKey struct {
	RateMatrixID  pulid.ID `json:"rateMatrixId"`
	VersionNumber int64    `json:"versionNumber"`
}

func (v *RateMatrixVersion) Key() VersionKey {
	return VersionKey{RateMatrixID: v.RateMatrixID, VersionNumber: v.VersionNumber}
}

type RateMatrixVersion struct {
	bun.BaseModel `bun:"table:rate_matrix_versions,alias:rmv" json:"-"`
	keel.Table    `keel:"rate_matrix_versions"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:rmv_"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	RateMatrixID   pulid.ID  `json:"rateMatrixId"   bun:"rate_matrix_id,type:VARCHAR(100),notnull" keel:"rate_matrix_id"`
	VersionNumber  int64     `json:"versionNumber"  bun:"version_number,type:BIGINT,notnull" keel:"version_number"`
	Code           string    `json:"code"           bun:"code,type:VARCHAR(64),notnull" keel:"code"`
	Name           string    `json:"name"           bun:"name,type:VARCHAR(100),notnull" keel:"name"`
	Description    string    `json:"description"    bun:"description,type:TEXT,nullzero" keel:"description,nullzero"`
	ValueType      string    `json:"valueType"      bun:"value_type,type:VARCHAR(32),notnull" keel:"value_type"`
	CellCount      int64     `json:"cellCount"      bun:"cell_count,type:BIGINT,notnull,default:0" keel:"cell_count"`
	ChangeMessage  string    `json:"changeMessage"  bun:"change_message,type:TEXT,nullzero" keel:"change_message,nullzero"`
	SubmittedByID  *pulid.ID `json:"submittedById"  bun:"submitted_by_id,type:VARCHAR(100)" keel:"submitted_by_id"`
	PublishedByID  *pulid.ID `json:"publishedById"  bun:"published_by_id,type:VARCHAR(100)" keel:"published_by_id"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`

	PublishedBy *tenant.User           `json:"publishedBy,omitempty" bun:"rel:belongs-to,join:published_by_id=id" keel:"belongs:published_by_id"`
	Dimensions  []*RateMatrixDimension `json:"dimensions,omitempty"  bun:"-"`
}

func NewVersionFromMatrix(
	matrix *RateMatrix,
	versionNumber int64,
	publishedByID pulid.ID,
) *RateMatrixVersion {
	publisher := publishedByID
	return &RateMatrixVersion{
		OrganizationID: matrix.OrganizationID,
		BusinessUnitID: matrix.BusinessUnitID,
		RateMatrixID:   matrix.ID,
		VersionNumber:  versionNumber,
		Code:           matrix.Code,
		Name:           matrix.Name,
		Description:    matrix.Description,
		ValueType:      matrix.ValueType,
		CellCount:      matrix.DraftCellCount,
		ChangeMessage:  matrix.ReviewComment,
		SubmittedByID:  matrix.SubmittedByID,
		PublishedByID:  &publisher,
	}
}

type PublishedCopy struct {
	Version    *RateMatrixVersion
	Dimensions []*RateMatrixDimension
	Cells      []*RateMatrixCell
}

func CopyPublished(
	matrix *RateMatrix,
	cells []*RateMatrixCell,
	versionNumber int64,
	publishedByID pulid.ID,
) PublishedCopy {
	version := NewVersionFromMatrix(matrix, versionNumber, publishedByID)
	version.ID = pulid.MustNew("rmv_")
	version.CellCount = int64(len(cells))
	if publishedByID.IsNil() {
		version.PublishedByID = nil
	}

	dimensions := make([]*RateMatrixDimension, 0, len(matrix.Dimensions))
	for _, dimension := range matrix.Dimensions {
		if dimension == nil {
			continue
		}
		copied := *dimension
		copied.ID = pulid.MustNew("rmd_")
		copied.VersionNumber = versionNumber
		copied.RateMatrix = nil
		dimensions = append(dimensions, &copied)
	}

	copiedCells := make([]*RateMatrixCell, 0, len(cells))
	for _, cell := range cells {
		if cell == nil {
			continue
		}
		copied := *cell
		copied.ID = pulid.MustNew("rmc_")
		copied.VersionNumber = versionNumber
		copied.RateMatrix = nil
		copiedCells = append(copiedCells, &copied)
	}

	version.Dimensions = dimensions
	return PublishedCopy{Version: version, Dimensions: dimensions, Cells: copiedCells}
}

func (v *RateMatrixVersion) Pin() tarifftypes.TablePin {
	return tarifftypes.TablePin{
		Code:          v.Code,
		RateMatrixID:  v.RateMatrixID,
		VersionNumber: v.VersionNumber,
	}
}

func (v *RateMatrixVersion) OrderedDimensions() []*RateMatrixDimension {
	matrix := RateMatrix{Dimensions: v.Dimensions}
	return matrix.OrderedDimensions()
}

func (v *RateMatrixVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("rmv_")
		}
		if v.CreatedAt == 0 {
			v.CreatedAt = timeutils.NowUnix()
		}
	}
	return nil
}

func (v *RateMatrixVersion) GetID() pulid.ID {
	return v.ID
}

func (v *RateMatrixVersion) GetOrganizationID() pulid.ID {
	return v.OrganizationID
}

func (v *RateMatrixVersion) GetBusinessUnitID() pulid.ID {
	return v.BusinessUnitID
}

func (v *RateMatrixVersion) GetTableName() string {
	return "rate_matrix_versions"
}
