package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/sso"
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
	) (*sso.SSOIdentityLink, error)
	GetByUser(
		ctx context.Context,
		req GetSSOIdentityLinkByUserRequest,
	) (*sso.SSOIdentityLink, error)
	Create(ctx context.Context, link *sso.SSOIdentityLink) error
	RecordLogin(ctx context.Context, link *sso.SSOIdentityLink, at int64) error
}
