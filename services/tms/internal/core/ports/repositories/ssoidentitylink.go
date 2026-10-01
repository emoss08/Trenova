package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetSSOIdentityLinkBySubjectRequest struct {
	SSOConfigID pulid.ID
	Issuer      string
	Subject     string
}

type GetSSOIdentityLinkByUserRequest struct {
	SSOConfigID pulid.ID
	Issuer      string
	UserID      pulid.ID
}

type SSOIdentityLinkRepository interface {
	GetBySubject(
		ctx context.Context,
		req GetSSOIdentityLinkBySubjectRequest,
	) (*tenant.SSOIdentityLink, error)
	GetByUser(
		ctx context.Context,
		req GetSSOIdentityLinkByUserRequest,
	) (*tenant.SSOIdentityLink, error)
	Create(ctx context.Context, link *tenant.SSOIdentityLink) error
	RecordLogin(ctx context.Context, link *tenant.SSOIdentityLink, at int64) error
}
