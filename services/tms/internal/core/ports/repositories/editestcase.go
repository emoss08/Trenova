package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDITestCasesRequest struct {
	Filter                   *pagination.QueryOptions `json:"filter"`
	Cursor                   pagination.CursorInfo    `json:"-"`
	PartnerDocumentProfileID pulid.ID                 `json:"partnerDocumentProfileId"`
}

type GetEDITestCaseByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DeleteEDITestCaseRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type EDITestCaseRepository interface {
	ListTestCases(
		ctx context.Context,
		req *ListEDITestCasesRequest,
	) (*pagination.ListResult[*edi.EDITestCase], error)
	ListTestCasesCursor(
		ctx context.Context,
		req *ListEDITestCasesRequest,
	) (*pagination.CursorListResult[*edi.EDITestCase], error)
	GetTestCaseByID(ctx context.Context, req GetEDITestCaseByIDRequest) (*edi.EDITestCase, error)
	CreateTestCase(ctx context.Context, entity *edi.EDITestCase) (*edi.EDITestCase, error)
	UpdateTestCase(ctx context.Context, entity *edi.EDITestCase) (*edi.EDITestCase, error)
	DeleteTestCase(ctx context.Context, req DeleteEDITestCaseRequest) error
}
