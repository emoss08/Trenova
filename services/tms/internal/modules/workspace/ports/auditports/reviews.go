package auditports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrReviewUnchanged = errors.New("the audit review was not in the expected state")

type ListReviewsRequest struct {
	TenantInfo    pagination.TenantInfo
	Limit         int
	ReviewedSince int64
}

type ReviewList struct {
	Open          []*audit.Review
	OpenCount     int
	Reviewed      []*audit.Review
	ReviewedCount int
}

type SetReviewedRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	ReviewerID pulid.ID
	At         int64
	Reviewed   bool
}

type ReviewRepository interface {
	Raise(ctx context.Context, review *audit.Review) (*audit.Review, bool, error)
	List(ctx context.Context, req *ListReviewsRequest) (*ReviewList, error)
	GetByID(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID) (*audit.Review, error)
	GetByEntryIDs(
		ctx context.Context,
		tenant pagination.TenantInfo,
		entryIDs []pulid.ID,
	) ([]*audit.Review, error)
	SetReviewed(ctx context.Context, req *SetReviewedRequest) (*audit.Review, error)
}
