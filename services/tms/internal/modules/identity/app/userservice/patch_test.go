package userservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type patchDeps struct {
	*testDeps
	security *securityaudittest.Recorder
	target   *tenant.User
	actor    pagination.TenantInfo
}

func setupPatchTest(t *testing.T) *patchDeps {
	t.Helper()

	deps := setupTest(t)
	security := &securityaudittest.Recorder{}
	deps.svc.auditor = security
	deps.svc.validator = NewTestValidatorWithRepository(deps.repo)

	target := newTestUser()
	target.Name = "Target User"
	target.IsLocked = true
	target.MustChangePassword = true

	return &patchDeps{
		testDeps: deps,
		security: security,
		target:   target,
		actor: pagination.TenantInfo{
			OrgID:  target.CurrentOrganizationID,
			BuID:   target.BusinessUnitID,
			UserID: pulid.MustNew("usr_"),
		},
	}
}

func (d *patchDeps) expectLoad(user *tenant.User) {
	d.repo.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo:         d.actor,
		LookupUserID:       user.ID,
		IncludeMemberships: true,
	}).Return(user, nil)
}

func (d *patchDeps) expectSave() *tenantports.UpdateUserColumnsRequest {
	captured := &tenantports.UpdateUserColumnsRequest{}
	d.repo.On("FindIdentityConflicts", mock.Anything, mock.Anything).
		Return(tenantports.UserIdentityConflicts{}, nil).Maybe()
	d.repo.On("UpdateColumns", mock.Anything, mock.Anything).
		Return(func(_ context.Context, req tenantports.UpdateUserColumnsRequest) *tenant.User {
			*captured = req
			saved := *req.Entity
			return &saved
		}, func(context.Context, tenantports.UpdateUserColumnsRequest) error {
			return nil
		}).Once()
	return captured
}

func TestPatch_WritesFalseTogglesItWasSent(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	original := *d.target
	d.expectLoad(d.target)
	captured := d.expectSave()
	d.expectAuditLog()

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply: func(user *tenant.User) error {
			user.IsLocked = false
			user.MustChangePassword = false
			return nil
		},
	})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"is_locked", "must_change_password"}, captured.Columns)
	assert.False(t, captured.Entity.IsLocked)
	assert.False(t, captured.Entity.MustChangePassword)
	assert.Equal(t, original.Name, captured.Entity.Name)
}

func TestPatch_WritesAnEmptyStringItWasSent(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	d.target.JobTitle = "Dispatcher"
	d.expectLoad(d.target)
	captured := d.expectSave()
	d.expectAuditLog()

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply: func(user *tenant.User) error {
			user.JobTitle = ""
			return nil
		},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"job_title"}, captured.Columns)
	assert.Empty(t, captured.Entity.JobTitle)
}

func TestPatch_AuditsAgainstTheTargetUser(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	d.expectLoad(d.target)
	d.expectSave()
	var logged *auditports.LogActionParams
	d.audit.On("LogAction", mock.MatchedBy(func(p *auditports.LogActionParams) bool {
		logged = p
		return true
	}), mock.Anything).Return(nil).Once()

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply: func(user *tenant.User) error {
			user.Name = "Renamed User"
			return nil
		},
	})

	require.NoError(t, err)
	require.NotNil(t, logged)
	assert.Equal(t, d.target.ID.String(), logged.ResourceID)
	assert.Equal(t, d.actor.UserID, logged.UserID)
	assert.Equal(t, "Target User", logged.PreviousState["name"])
	assert.Equal(t, d.target.ID.String(), logged.PreviousState["id"])
	assert.Equal(t, "Renamed User", logged.CurrentState["name"])
	d.repo.AssertNotCalled(t, "GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo:         d.actor,
		IncludeMemberships: true,
	})
}

func TestPatch_RefusesToLetAnAdminLockThemselves(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	self := newTestUser()
	d.actor.UserID = self.ID
	d.actor.OrgID = self.CurrentOrganizationID
	d.actor.BuID = self.BusinessUnitID
	d.expectLoad(self)

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     self.ID,
		Apply: func(user *tenant.User) error {
			user.IsLocked = true
			return nil
		},
	})

	assertFieldError(t, err, "isLocked", errortypes.ErrForbidden)
	d.repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
}

func TestPatch_RefusesToLetAnAdminDeactivateThemselves(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	self := newTestUser()
	d.actor.UserID = self.ID
	d.actor.OrgID = self.CurrentOrganizationID
	d.actor.BuID = self.BusinessUnitID
	d.expectLoad(self)

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     self.ID,
		Apply: func(user *tenant.User) error {
			user.Status = domaintypes.StatusInactive
			return nil
		},
	})

	assertFieldError(t, err, "status", errortypes.ErrForbidden)
	d.repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
}

func TestPatch_LetsAnAdminLockSomebodyElse(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	d.target.IsLocked = false
	d.expectLoad(d.target)
	captured := d.expectSave()
	d.expectAuditLog()
	d.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, d.target.ID).Return(nil).Once()

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply: func(user *tenant.User) error {
			user.IsLocked = true
			return nil
		},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"is_locked"}, captured.Columns)
	change := d.security.Only(t)
	assert.Equal(t, permission.OpUpdate, change.Operation)
	assert.Equal(t, d.target.ID.String(), change.ResourceID)
	assert.Equal(t, false, jsonutils.MustToJSON(change.Before)["isLocked"])
	assert.Equal(t, true, jsonutils.MustToJSON(change.After)["isLocked"])
}

func TestPatch_IgnoresAttemptsToChangeImmutableFields(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	d.expectLoad(d.target)
	original := *d.target

	updated, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply: func(user *tenant.User) error {
			user.Username = "hijacked"
			user.CurrentOrganizationID = pulid.MustNew("org_")
			user.Password = "plain"
			return nil
		},
	})

	require.NoError(t, err)
	assert.Equal(t, original.Username, updated.Username)
	assert.Equal(t, original.CurrentOrganizationID, updated.CurrentOrganizationID)
	d.repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
	d.audit.AssertNotCalled(t, "LogAction", mock.Anything, mock.Anything)
}

func TestPatch_RefusesAnEmailAddressAnotherUserHas(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	d.expectLoad(d.target)
	d.repo.On("FindIdentityConflicts", mock.Anything, tenantports.UserIdentityConflictsRequest{
		EmailAddress:  "taken@example.com",
		Username:      d.target.Username,
		ExcludeUserID: d.target.ID,
	}).Return(tenantports.UserIdentityConflicts{EmailAddress: true}, nil).Once()

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply: func(user *tenant.User) error {
			user.EmailAddress = "Taken@Example.com"
			return nil
		},
	})

	assertFieldError(t, err, "emailAddress", errortypes.ErrDuplicate)
	d.repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
}

func TestPatch_ReturnsTheApplyError(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)
	d.expectLoad(d.target)
	refusal := errortypes.NewValidationError("name", errortypes.ErrRequired, "Name cannot be cleared")

	_, err := d.svc.Patch(t.Context(), &PatchUserRequest{
		TenantInfo: d.actor,
		UserID:     d.target.ID,
		Apply:      func(*tenant.User) error { return refusal },
	})

	require.ErrorIs(t, err, refusal)
	d.repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
}

func TestBulkUpdateStatusRefusesToDeactivateTheCaller(t *testing.T) {
	t.Parallel()

	d := setupPatchTest(t)

	_, err := d.svc.BulkUpdateStatus(t.Context(), &tenantports.BulkUpdateUserStatusRequest{
		TenantInfo: d.actor,
		UserIDs:    []pulid.ID{d.target.ID, d.actor.UserID},
		Status:     domaintypes.StatusInactive,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "You cannot deactivate your own account")
	d.repo.AssertNotCalled(t, "BulkUpdateStatus", mock.Anything, mock.Anything)
}
