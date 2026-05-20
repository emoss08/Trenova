package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDITestCasesRequest struct {
	Filter                   *pagination.QueryOptions `json:"filter"`
	PartnerDocumentProfileID pulid.ID                 `json:"partnerDocumentProfileId"`
}

type GetEDITestCaseByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type EDITestCaseRepository interface {
	ListTestCases(
		ctx context.Context,
		req *ListEDITestCasesRequest,
	) (*pagination.ListResult[*edi.EDITestCase], error)
	GetTestCaseByID(ctx context.Context, req GetEDITestCaseByIDRequest) (*edi.EDITestCase, error)
	CreateTestCase(ctx context.Context, entity *edi.EDITestCase) (*edi.EDITestCase, error)
}
