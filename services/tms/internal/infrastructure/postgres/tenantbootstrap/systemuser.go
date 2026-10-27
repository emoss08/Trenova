package tenantbootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	SystemUserName     = "System Account"
	SystemUserEmail    = "system@trenova.app"
	systemUserTimezone = "America/Los_Angeles"
)

var ErrSystemUserPasswordRequired = errors.New(
	"tenant bootstrap: the system user password must be set in config (system.systemUserPassword)",
)

type SystemUserParams struct {
	Scope    Scope
	Password string
}

func EnsureSystemUser(
	ctx context.Context,
	db *keel.DB,
	params SystemUserParams,
) (pulid.ID, bool, error) {
	u := tenantdb.Users
	existing, err := scoped(ctx, u.Pick(u.ID).Where(u.Username.Lower().Eq(tenant.SystemUsername))).
		First(ctx, db)
	switch {
	case err == nil:
		return existing.ID, false, nil
	case !errors.Is(err, keel.ErrNotFound):
		return pulid.Nil, false, fmt.Errorf("check system user: %w", err)
	}

	if strings.TrimSpace(params.Password) == "" {
		return pulid.Nil, false, ErrSystemUserPasswordRequired
	}

	now := params.Scope.now()
	user := &tenant.User{
		BusinessUnitID:        params.Scope.BusinessUnitID,
		CurrentOrganizationID: params.Scope.OrganizationID,
		Name:                  SystemUserName,
		Username:              tenant.SystemUsername,
		EmailAddress:          SystemUserEmail,
		Status:                domaintypes.StatusActive,
		Timezone:              systemUserTimezone,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	hashed, err := user.GeneratePassword(params.Password)
	if err != nil {
		return pulid.Nil, false, fmt.Errorf("hash system user password: %w", err)
	}
	user.Password = hashed

	if err = u.Create(ctx, db, user); err != nil {
		return pulid.Nil, false, fmt.Errorf("create system user: %w", err)
	}

	return user.ID, true, nil
}
