package userrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	usersEmailAddressIndex = "idx_users_email_address"
	usersUsernameIndex     = "idx_users_username"
)

func (ur *repository) Create(
	ctx context.Context,
	req tenantports.CreateUserRequest,
) (*tenant.User, error) {
	return dbtx.Write(ctx, ur.db, func(ctx context.Context) (*tenant.User, error) {
		log := ur.l.With(
			zap.String("operation", "Create"),
			zap.String("organizationID", req.User.CurrentOrganizationID.String()),
		)

		if err := quotaservice.EnforceAll(
			ctx,
			ur.quota,
			platformplanports.QuotaRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: req.Membership.OrganizationID,
					BuID:  req.Membership.BusinessUnitID,
				},
				Meter:    platformcatalog.MeterUserSeats,
				Quantity: 1,
			},
		); err != nil {
			return nil, err
		}

		db := ur.db.Keel(ctx)

		if err := tenantdb.Users.Create(ctx, db, req.User); err != nil {
			log.Error("failed to insert user", zap.Error(err))
			return nil, userUniqueViolation(err)
		}

		req.Membership.UserID = req.User.ID
		if err := tenantdb.OrganizationMemberships.Create(ctx, db, req.Membership); err != nil {
			log.Error("failed to insert organization membership", zap.Error(err))
			return nil, err
		}

		req.User.Memberships = []*tenant.OrganizationMembership{req.Membership}

		return req.User, nil
	})
}

func (ur *repository) FindIdentityConflicts(
	ctx context.Context,
	req tenantports.UserIdentityConflictsRequest,
) (tenantports.UserIdentityConflicts, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"email addresses and usernames are unique across every tenant, as sign-in requires",
	)
	return dbtx.Read(
		ctx,
		ur.db,
		func(ctx context.Context) (tenantports.UserIdentityConflicts, error) {
			var conflicts tenantports.UserIdentityConflicts
			if req.EmailAddress == "" && req.Username == "" {
				return conflicts, nil
			}

			u := tenantdb.Users
			emailMatch := keel.DynFunc("lower", u.EmailAddress).Eq(keel.DynFunc("lower", keel.DynLit(req.EmailAddress).Cast(keel.TypeString)))
			usernameMatch := keel.DynFunc("lower", u.Username).Eq(keel.DynFunc("lower", keel.DynLit(req.Username).Cast(keel.TypeString)))

			row, err := keel.IntoOne[identityMatches](
				ctx,
				ur.db.Keel(ctx),
				u.AllTenants().
					Pick(
						keel.DynFunc("bool_or", emailMatch).Coalesce(false),
						keel.DynFunc("bool_or", usernameMatch).Coalesce(false),
					).
					Where(
						keel.Or(emailMatch, usernameMatch),
						keel.When(req.ExcludeUserID.IsNotNil(), u.ID.Ne(req.ExcludeUserID)),
					),
			)
			if err != nil {
				ur.l.Error("failed to check user identity conflicts", zap.Error(err))
				return conflicts, err
			}

			conflicts.EmailAddress = req.EmailAddress != "" && row.EmailAddress
			conflicts.Username = req.Username != "" && row.Username

			return conflicts, nil
		},
	)
}

type identityMatches struct {
	EmailAddress bool
	Username     bool
}

func userUniqueViolation(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	me := errortypes.NewMultiError()
	switch dberror.ExtractConstraintName(err) {
	case usersEmailAddressIndex:
		me.AddError(tenant.NewDuplicateEmailAddressError())
	case usersUsernameIndex:
		me.AddError(tenant.NewDuplicateUsernameError())
	default:
		return err
	}

	return me
}
