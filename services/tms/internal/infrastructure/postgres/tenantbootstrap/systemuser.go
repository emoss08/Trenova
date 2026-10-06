package tenantbootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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
	db bun.IDB,
	params SystemUserParams,
) (pulid.ID, bool, error) {
	cols := buncolgen.UserColumns
	existing := new(tenant.User)
	err := db.NewSelect().
		Model(existing).
		Column(cols.ID.Bare()).
		Where("lower("+cols.Username.Qualified()+") = ?", tenant.SystemUsername).
		Limit(1).
		Scan(ctx)
	switch {
	case err == nil:
		return existing.ID, false, nil
	case !dberror.IsNotFoundError(err):
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

	if _, err = db.NewInsert().Model(user).Exec(ctx); err != nil {
		return pulid.Nil, false, fmt.Errorf("create system user: %w", err)
	}
	if err = params.Scope.record(ctx, "users", user.ID); err != nil {
		return pulid.Nil, false, err
	}

	return user.ID, true, nil
}
