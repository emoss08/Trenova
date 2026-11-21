package tenantbootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnsureSystemActor(
	ctx context.Context,
	db *keel.DB,
	scope tenantscope.Scope,
) (pulid.ID, bool, error) {
	sa := tenantdb.SystemActors
	existing, err := tenantscope.Scoped(
		ctx,
		sa.Pick(sa.UserID).Where(sa.BusinessUnitID.Eq(scope.BusinessUnitID)),
	).First(ctx, db)
	switch {
	case err == nil:
		return existing.UserID, false, nil
	case !errors.Is(err, keel.ErrNotFound):
		return pulid.Nil, false, fmt.Errorf("check system actor: %w", err)
	}

	user := tenant.NewSystemActorUser(scope.BusinessUnitID, scope.OrganizationID)
	password, err := user.GenerateUnusablePassword()
	if err != nil {
		return pulid.Nil, false, fmt.Errorf("hash the system actor's unusable password: %w", err)
	}
	user.Password = password
	now := scope.NowUnix()
	user.CreatedAt = now
	user.UpdatedAt = now

	if err = tenantdb.Users.Create(ctx, db, user); err != nil {
		return pulid.Nil, false, fmt.Errorf("create system actor: %w", err)
	}
	if err = sa.Create(ctx, db, &tenant.SystemActor{
		BusinessUnitID: scope.BusinessUnitID,
		UserID:         user.ID,
		CreatedAt:      now,
	}); err != nil {
		return pulid.Nil, false, fmt.Errorf("record system actor: %w", err)
	}

	return user.ID, true, nil
}

func NotSystemActor(ctx context.Context) keel.Pred {
	u := tenantdb.Users
	sa := tenantdb.SystemActors
	return keel.Not(keel.Exists(tenantscope.Scoped(ctx, sa.Where(sa.UserID.EqCol(u.ID)))))
}
