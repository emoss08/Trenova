package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetEmailProfileByIDRequest struct {
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
	ProfileID  pulid.ID
	ExpandData bool
}

type ListEmailProfileRequest struct {
	Filter          *pagination.QueryOptions `json:"filter"          form:"filter"`
	ExcludeInactive bool                     `json:"excludeInactive" form:"excludeInactive"`
}

type EmailProfileRepository interface {
	Create(ctx context.Context, profile *email.EmailProfile) (*email.EmailProfile, error)
	Update(ctx context.Context, profile *email.EmailProfile) (*email.EmailProfile, error)
	Get(ctx context.Context, req GetEmailProfileByIDRequest) (*email.EmailProfile, error)
	List(
		ctx context.Context,
		req *ListEmailProfileRequest,
	) (*pagination.ListResult[*email.EmailProfile], error)
	GetDefault(ctx context.Context, orgID, buID pulid.ID) (*email.EmailProfile, error)
}
