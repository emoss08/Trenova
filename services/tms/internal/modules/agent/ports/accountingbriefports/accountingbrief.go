package accountingbriefports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetLatestAccountingBriefRequest struct {
	TenantInfo pagination.TenantInfo
	BriefDate  string
}

type MarkAccountingBriefVerifiedRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	VerifiedAt int64
}

type DeleteAccountingBriefsBeforeRequest struct {
	BeforeDate string
	Limit      int
}

type AccountingBriefRepository interface {
	Latest(
		ctx context.Context,
		req *GetLatestAccountingBriefRequest,
	) (*accountingbrief.Brief, error)
	Insert(ctx context.Context, brief *accountingbrief.Brief) (*accountingbrief.Brief, error)
	MarkVerified(ctx context.Context, req *MarkAccountingBriefVerifiedRequest) error
	DeleteBefore(ctx context.Context, req *DeleteAccountingBriefsBeforeRequest) (int, error)
}
