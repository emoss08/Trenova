package formulalibraryports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetLibraryByIDRequest struct {
	TenantInfo pagination.TenantInfo
	LibraryID  pulid.ID
}

type GetLibrariesByIDsRequest struct {
	TenantInfo pagination.TenantInfo
	LibraryIDs []pulid.ID
}

type ListLibraryConnectionRequest struct {
	Filter                *pagination.QueryOptions
	Cursor                pagination.CursorInfo
	FormulaLibraryColumns []string
}

type ListApprovedLibrariesRequest struct {
	TenantInfo pagination.TenantInfo
	Limit      int
}

type FormulaLibraryRepository interface {
	Create(
		ctx context.Context,
		entity *formulalibrary.FormulaLibrary,
	) (*formulalibrary.FormulaLibrary, error)
	Update(
		ctx context.Context,
		entity *formulalibrary.FormulaLibrary,
	) (*formulalibrary.FormulaLibrary, error)
	GetByID(
		ctx context.Context,
		req *GetLibraryByIDRequest,
	) (*formulalibrary.FormulaLibrary, error)
	GetByIDs(
		ctx context.Context,
		req *GetLibrariesByIDsRequest,
	) ([]*formulalibrary.FormulaLibrary, error)
	ListConnection(
		ctx context.Context,
		req *ListLibraryConnectionRequest,
	) (*pagination.CursorListResult[*formulalibrary.FormulaLibrary], error)
	NamesLike(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		prefix string,
	) ([]string, error)
}

type GetLibraryVersionRequest struct {
	TenantInfo    pagination.TenantInfo
	LibraryID     pulid.ID
	VersionNumber int64
}

type ListLibraryVersionsRequest struct {
	TenantInfo pagination.TenantInfo
	LibraryID  pulid.ID
	Limit      int
}

type GetLibraryVersionRangeRequest struct {
	TenantInfo  pagination.TenantInfo
	LibraryID   pulid.ID
	FromVersion int64
	ToVersion   int64
}

type LatestApprovedByNamesRequest struct {
	TenantInfo pagination.TenantInfo
	Names      []string
}

type GetVersionsByPinsRequest struct {
	TenantInfo pagination.TenantInfo
	Pins       []tarifftypes.Import
}

type ListLibraryAuthorsRequest struct {
	TenantInfo   pagination.TenantInfo
	LibraryID    pulid.ID
	AfterVersion int64
}

type GetLatestLibraryVersionByStatusRequest struct {
	TenantInfo pagination.TenantInfo
	LibraryID  pulid.ID
	Status     formulatemplate.Status
}

type FormulaLibraryVersionRepository interface {
	Create(
		ctx context.Context,
		version *formulalibrary.FormulaLibraryVersion,
	) (*formulalibrary.FormulaLibraryVersion, error)
	GetByLibraryAndVersion(
		ctx context.Context,
		req *GetLibraryVersionRequest,
	) (*formulalibrary.FormulaLibraryVersion, error)
	List(
		ctx context.Context,
		req *ListLibraryVersionsRequest,
	) ([]*formulalibrary.FormulaLibraryVersion, error)
	GetVersionRange(
		ctx context.Context,
		req *GetLibraryVersionRangeRequest,
	) ([]*formulalibrary.FormulaLibraryVersion, error)
	LatestApprovedByNames(
		ctx context.Context,
		req *LatestApprovedByNamesRequest,
	) ([]*formulalibrary.FormulaLibraryVersion, error)
	ListLatestApproved(
		ctx context.Context,
		req *ListApprovedLibrariesRequest,
	) ([]*formulalibrary.FormulaLibraryVersion, error)
	GetByPins(
		ctx context.Context,
		req *GetVersionsByPinsRequest,
	) ([]*formulalibrary.FormulaLibraryVersion, error)
	ListAuthorIDsSince(
		ctx context.Context,
		req *ListLibraryAuthorsRequest,
	) ([]pulid.ID, error)
	GetLatestByStatus(
		ctx context.Context,
		req *GetLatestLibraryVersionByStatusRequest,
	) (*formulalibrary.FormulaLibraryVersion, error)
}

type ListLibraryReviewsRequest struct {
	TenantInfo pagination.TenantInfo
	LibraryID  pulid.ID
	Limit      int
}

type ListStaleLibrarySubmissionsRequest struct {
	SubmittedBefore int64
	Limit           int
}

type FormulaLibraryReviewRepository interface {
	Create(
		ctx context.Context,
		entity *formulalibrary.FormulaLibraryReview,
	) (*formulalibrary.FormulaLibraryReview, error)
	ListByLibrary(
		ctx context.Context,
		req *ListLibraryReviewsRequest,
	) ([]*formulalibrary.FormulaLibraryReview, error)
	Latest(
		ctx context.Context,
		libraryID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*formulalibrary.FormulaLibraryReview, error)
	ListStaleSubmissions(
		ctx context.Context,
		req *ListStaleLibrarySubmissionsRequest,
	) ([]*formulalibrary.FormulaLibrary, error)
}
