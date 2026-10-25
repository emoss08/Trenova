package userservice

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

type PatchUserRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Apply      func(*tenant.User) error
}

type patchableColumn struct {
	column  keel.Col
	changed func(before, after *tenant.User) bool
	access  bool
}

var patchableColumns = []patchableColumn{
	{
		column:  tenantdb.Users.Status.Col,
		changed: func(b, a *tenant.User) bool { return b.Status != a.Status },
		access:  true,
	},
	{
		column:  tenantdb.Users.Name.Col,
		changed: func(b, a *tenant.User) bool { return b.Name != a.Name },
	},
	{
		column:  tenantdb.Users.EmailAddress.Col,
		changed: func(b, a *tenant.User) bool { return b.EmailAddress != a.EmailAddress },
		access:  true,
	},
	{
		column:  tenantdb.Users.Timezone.Col,
		changed: func(b, a *tenant.User) bool { return b.Timezone != a.Timezone },
	},
	{
		column:  tenantdb.Users.TimeFormat.Col,
		changed: func(b, a *tenant.User) bool { return b.TimeFormat != a.TimeFormat },
	},
	{
		column:  tenantdb.Users.Locale.Col,
		changed: func(b, a *tenant.User) bool { return b.Locale != a.Locale },
	},
	{
		column:  tenantdb.Users.IsLocked.Col,
		changed: func(b, a *tenant.User) bool { return b.IsLocked != a.IsLocked },
		access:  true,
	},
	{
		column: tenantdb.Users.MustChangePassword.Col,
		changed: func(b, a *tenant.User) bool {
			return b.MustChangePassword != a.MustChangePassword
		},
		access: true,
	},
	{
		column:  tenantdb.Users.JobTitle.Col,
		changed: func(b, a *tenant.User) bool { return b.JobTitle != a.JobTitle },
	},
	{
		column:  tenantdb.Users.PhoneNumber.Col,
		changed: func(b, a *tenant.User) bool { return b.PhoneNumber != a.PhoneNumber },
	},
}

type userChanges struct {
	columns []string
	access  []string
}

func diffPatchedUser(before, after *tenant.User) userChanges {
	changes := userChanges{columns: make([]string, 0, len(patchableColumns))}
	for i := range patchableColumns {
		candidate := &patchableColumns[i]
		if !candidate.changed(before, after) {
			continue
		}
		changes.columns = append(changes.columns, columnName(candidate.column))
		if candidate.access {
			changes.access = append(changes.access, columnName(candidate.column))
		}
	}

	return changes
}

func (s *Service) Patch(ctx context.Context, req *PatchUserRequest) (*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "Patch"),
		zap.String("actorID", req.TenantInfo.UserID.String()),
		zap.String("userID", req.UserID.String()),
	)

	user, err := s.repo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:         req.TenantInfo,
		LookupUserID:       req.UserID,
		IncludeMemberships: true,
	})
	if err != nil {
		return nil, err
	}

	original := *user
	if err = req.Apply(user); err != nil {
		return nil, err
	}

	restoreImmutableFields(user, &original)
	user.Name = strings.TrimSpace(user.Name)
	user.EmailAddress = stringutils.NormalizeEmailAddress(user.EmailAddress)

	if multiErr := guardSelfAccess(req.TenantInfo.UserID, &original, user); multiErr != nil {
		return nil, multiErr
	}

	changes := diffPatchedUser(&original, user)
	if len(changes.columns) == 0 {
		if err = s.endLingeringSessions(ctx, req.TenantInfo, &original); err != nil {
			return nil, err
		}
		return &original, nil
	}

	if multiErr := s.validator.ValidateUpdate(ctx, user); multiErr != nil {
		return nil, multiErr
	}

	updated, err := s.updateColumns(ctx, &columnUpdate{
		tenantInfo: req.TenantInfo,
		user:       user,
		original:   &original,
		columns:    changes.columns,
		comment:    "User updated",
	})
	if err != nil {
		log.Error("failed to patch user", zap.Error(err))
		return nil, err
	}

	if len(changes.access) > 0 {
		s.recordAccessChange(ctx, &accessChange{
			tenantInfo: req.TenantInfo,
			before:     &original,
			after:      updated,
			columns:    changes.access,
		})
	}

	if lostSignInAccess(&original, updated) {
		if err = s.endAllSessions(ctx, req.TenantInfo, updated.ID); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

func restoreImmutableFields(user, original *tenant.User) {
	user.ID = original.ID
	user.BusinessUnitID = original.BusinessUnitID
	user.CurrentOrganizationID = original.CurrentOrganizationID
	user.Username = original.Username
	user.Password = original.Password
	user.Version = original.Version
}

func guardSelfAccess(actorID pulid.ID, before, after *tenant.User) *errortypes.MultiError {
	if actorID.IsNil() || actorID != before.ID {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	if after.IsLocked && !before.IsLocked {
		multiErr.Add("isLocked", errortypes.ErrForbidden, "You cannot lock your own account")
	}
	if after.Status != domaintypes.StatusActive && before.Status == domaintypes.StatusActive {
		multiErr.Add("status", errortypes.ErrForbidden, "You cannot deactivate your own account")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type accessChange struct {
	tenantInfo pagination.TenantInfo
	before     *tenant.User
	after      *tenant.User
	columns    []string
}

func (s *Service) recordAccessChange(ctx context.Context, change *accessChange) {
	if s.auditor == nil {
		return
	}

	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceUser,
		ResourceID:     change.after.ID.String(),
		Operation:      permission.OpUpdate,
		Actor:          permissionports.UserActor(change.tenantInfo).AuditActor(),
		OrganizationID: change.tenantInfo.OrgID,
		BusinessUnitID: change.tenantInfo.BuID,
		Before:         accessState(change.before),
		After:          accessState(change.after),
		Comment:        "User sign-in access changed",
		Metadata:       map[string]any{"changedColumns": change.columns},
	})
}

func accessState(user *tenant.User) map[string]any {
	return map[string]any{
		"status":             user.Status,
		"emailAddress":       user.EmailAddress,
		"isLocked":           user.IsLocked,
		"mustChangePassword": user.MustChangePassword,
	}
}
