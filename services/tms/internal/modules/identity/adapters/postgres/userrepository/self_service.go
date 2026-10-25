package userrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/zap"
)

type columnSetter func(*tenant.User) keel.Assignment

var updatableUserColumns = userColumnSetters()

func userColumnSetters() map[string]columnSetter {
	u := tenantdb.Users
	setters := []struct {
		col keel.Col
		set columnSetter
	}{
		{u.Status.Col, func(e *tenant.User) keel.Assignment { return u.Status.To(e.Status) }},
		{u.Name.Col, func(e *tenant.User) keel.Assignment { return u.Name.To(e.Name) }},
		{u.EmailAddress.Col, func(e *tenant.User) keel.Assignment {
			return u.EmailAddress.To(e.EmailAddress)
		}},
		{u.Timezone.Col, func(e *tenant.User) keel.Assignment { return u.Timezone.To(e.Timezone) }},
		{u.TimeFormat.Col, func(e *tenant.User) keel.Assignment {
			return u.TimeFormat.To(e.TimeFormat)
		}},
		{u.Locale.Col, func(e *tenant.User) keel.Assignment { return u.Locale.To(e.Locale) }},
		{u.IsLocked.Col, func(e *tenant.User) keel.Assignment { return u.IsLocked.To(e.IsLocked) }},
		{u.MustChangePassword.Col, func(e *tenant.User) keel.Assignment {
			return u.MustChangePassword.To(e.MustChangePassword)
		}},
		{u.JobTitle.Col, func(e *tenant.User) keel.Assignment { return u.JobTitle.To(e.JobTitle) }},
		{u.PhoneNumber.Col, func(e *tenant.User) keel.Assignment {
			return u.PhoneNumber.To(e.PhoneNumber)
		}},
		{u.AvatarStyle.Col, func(e *tenant.User) keel.Assignment {
			return u.AvatarStyle.To(e.AvatarStyle)
		}},
		{u.ProfilePicURL.Col, func(e *tenant.User) keel.Assignment {
			return u.ProfilePicURL.To(e.ProfilePicURL)
		}},
		{u.ThumbnailURL.Col, func(e *tenant.User) keel.Assignment {
			return u.ThumbnailURL.To(e.ThumbnailURL)
		}},
		{u.AssistantHidden.Col, func(e *tenant.User) keel.Assignment {
			return u.AssistantHidden.To(e.AssistantHidden)
		}},
		{u.AvailabilityStatus.Col, func(e *tenant.User) keel.Assignment {
			return u.AvailabilityStatus.To(e.AvailabilityStatus)
		}},
		{u.FocusUntil.Col, func(e *tenant.User) keel.Assignment {
			if e.FocusUntil == nil {
				return u.FocusUntil.ToNull()
			}
			return u.FocusUntil.To(*e.FocusUntil)
		}},
		{u.AwayBackOn.Col, func(e *tenant.User) keel.Assignment {
			if e.AwayBackOn == nil {
				return u.AwayBackOn.ToNull()
			}
			return u.AwayBackOn.To(*e.AwayBackOn)
		}},
		{u.AwayDelegateUserID.Col, func(e *tenant.User) keel.Assignment {
			if e.AwayDelegateUserID == nil {
				return u.AwayDelegateUserID.ToNull()
			}
			return u.AwayDelegateUserID.To(*e.AwayDelegateUserID)
		}},
		{u.AwayNote.Col, func(e *tenant.User) keel.Assignment { return u.AwayNote.To(e.AwayNote) }},
	}

	table := tenantdb.UsersSchema()
	byName := make(map[string]columnSetter, len(setters))
	for _, s := range setters {
		byName[table.Columns[s.col.Index].Name] = s.set
	}

	return byName
}

func (ur *repository) UpdateColumns(
	ctx context.Context,
	req tenantports.UpdateUserColumnsRequest,
) (*tenant.User, error) {
	entity := req.Entity
	sets := make([]keel.Assignment, 0, len(req.Columns))
	for _, column := range req.Columns {
		set, ok := updatableUserColumns[column]
		if !ok {
			return nil, fmt.Errorf("user column %q cannot be updated through UpdateColumns", column)
		}
		sets = append(sets, set(entity))
	}

	return dbtx.Write(ctx, ur.db, func(ctx context.Context) (*tenant.User, error) {
		u := tenantdb.Users
		updated, err := u.Where(
			u.ID.Eq(entity.ID),
			u.Version.Eq(entity.Version),
			u.CurrentOrganizationID.Eq(entity.CurrentOrganizationID),
		).
			Pick(u.Version, u.UpdatedAt).
			UpdateReturning(ctx, ur.db.Keel(ctx), sets...)
		if err != nil {
			ur.l.Error(
				"failed to update user columns",
				zap.String("userID", entity.ID.String()),
				zap.Strings("columns", req.Columns),
				zap.Error(err),
			)
			return nil, err
		}

		if err = dberror.CheckAffected(
			int64(len(updated)),
			"User",
			entity.ID.String(),
		); err != nil {
			return nil, err
		}

		entity.Version = updated[0].Version
		entity.UpdatedAt = updated[0].UpdatedAt

		return entity, nil
	})
}
