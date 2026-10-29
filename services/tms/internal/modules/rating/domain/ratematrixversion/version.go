package ratematrixversion

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixcell"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type VersionKey struct {
	RateMatrixID  pulid.ID `json:"rateMatrixId"`
	VersionNumber int64    `json:"versionNumber"`
}

func (v *RateMatrixVersion) Key() VersionKey {
	return VersionKey{RateMatrixID: v.RateMatrixID, VersionNumber: v.VersionNumber}
}

type RateMatrixVersion struct {
	keel.Table `keel:"rate_matrix_versions"`

	ID             pulid.ID  `json:"id"             keel:"id,pk,autoid:rmv_"`
	OrganizationID pulid.ID  `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	RateMatrixID   pulid.ID  `json:"rateMatrixId"   keel:"rate_matrix_id"`
	VersionNumber  int64     `json:"versionNumber"  keel:"version_number"`
	Code           string    `json:"code"           keel:"code"`
	Name           string    `json:"name"           keel:"name"`
	Description    string    `json:"description"    keel:"description,nullzero"`
	ValueType      string    `json:"valueType"      keel:"value_type"`
	CellCount      int64     `json:"cellCount"      keel:"cell_count"`
	ChangeMessage  string    `json:"changeMessage"  keel:"change_message,nullzero"`
	SubmittedByID  *pulid.ID `json:"submittedById"  keel:"submitted_by_id"`
	PublishedByID  *pulid.ID `json:"publishedById"  keel:"published_by_id"`
	CreatedAt      int64     `json:"createdAt"      keel:"created_at,autocreate"`

	PublishedBy *tenant.User                      `json:"publishedBy,omitempty" keel:"belongs:published_by_id"`
	Dimensions  []*ratematrix.RateMatrixDimension `json:"dimensions,omitempty"`
}

func NewVersionFromMatrix(
	matrix *ratematrix.RateMatrix,
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
	Dimensions []*ratematrix.RateMatrixDimension
	Cells      []*ratematrixcell.RateMatrixCell
}

func CopyPublished(
	matrix *ratematrix.RateMatrix,
	cells []*ratematrixcell.RateMatrixCell,
	versionNumber int64,
	publishedByID pulid.ID,
) PublishedCopy {
	version := NewVersionFromMatrix(matrix, versionNumber, publishedByID)
	version.ID = pulid.MustNew("rmv_")
	version.CellCount = int64(len(cells))
	if publishedByID.IsNil() {
		version.PublishedByID = nil
	}

	dimensions := make([]*ratematrix.RateMatrixDimension, 0, len(matrix.Dimensions))
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

	copiedCells := make([]*ratematrixcell.RateMatrixCell, 0, len(cells))
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

func (v *RateMatrixVersion) OrderedDimensions() []*ratematrix.RateMatrixDimension {
	matrix := ratematrix.RateMatrix{Dimensions: v.Dimensions}
	return matrix.OrderedDimensions()
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
