package tenantbootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

var ErrNotAMember = errors.New("the person is not a live member of the organization")

type MemberRequest struct {
	UserID         pulid.ID
	OrganizationID pulid.ID
	Now            int64
}

func LockMember(ctx context.Context, db *keel.DB, req MemberRequest) (pulid.ID, error) {
	if req.UserID.IsNil() || req.OrganizationID.IsNil() {
		return pulid.Nil, ErrNotAMember
	}
	now := req.Now
	if now <= 0 {
		now = timeutils.NowUnix()
	}

	m := tenantdb.OrganizationMemberships
	sa := tenantdb.SystemActors
	membership, err := tenantscope.Scoped(
		ctx,
		m.Pick(m.BusinessUnitID).Where(
			m.UserID.Eq(req.UserID),
			m.OrganizationID.Eq(req.OrganizationID),
			keel.Or(m.ExpiresAt.IsNull(), m.ExpiresAt.Gt(now)),
			keel.Not(keel.Exists(tenantscope.Scoped(ctx, sa.Where(sa.UserID.EqCol(m.UserID))))),
		),
	).Lock(keel.ForShare).First(ctx, db)
	switch {
	case errors.Is(err, keel.ErrNotFound):
		return pulid.Nil, ErrNotAMember
	case err != nil:
		return pulid.Nil, fmt.Errorf("check the membership of user %s: %w", req.UserID, err)
	}

	return membership.BusinessUnitID, nil
}
