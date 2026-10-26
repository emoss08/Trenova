package userservice

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/sessionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*tenant.User]().
			WithModelName("User").
			Build(),
	}
}

type testDeps struct {
	repo        *tenantportsmock.MockUserRepository
	roleRepo    *permissionportsmock.MockRoleRepository
	sessionRepo *sessionportsmock.MockSessionRepository
	permissions *permissionportsmock.MockPermissionEngine
	storage     *kernelportsmock.MockClient
	audit       *auditportsmock.MockAuditService
	svc         *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := tenantportsmock.NewMockUserRepository(t)
	roleRepo := permissionportsmock.NewMockRoleRepository(t)
	sessionRepo := sessionportsmock.NewMockSessionRepository(t)
	permissions := permissionportsmock.NewMockPermissionEngine(t)
	storageClient := kernelportsmock.NewMockClient(t)
	auditSvc := auditportsmock.NewMockAuditService(t)
	svc := &Service{
		permissions:  permissions,
		l:            zap.NewNop(),
		repo:         repo,
		roleRepo:     roleRepo,
		sr:           sessionRepo,
		auditService: auditSvc,
		realtime:     &mocks.NoopRealtimeService{},
		storage:      storageClient,
		storageCfg: &config.StorageConfig{
			MaxFileSize:        5 * 1024 * 1024,
			PresignedURLExpiry: 15 * time.Minute,
			AllowedMIMETypes:   []string{"image/jpeg", "image/png", "image/webp"},
		},
		validator: newTestValidator(),
	}
	return &testDeps{
		repo:        repo,
		roleRepo:    roleRepo,
		sessionRepo: sessionRepo,
		permissions: permissions,
		storage:     storageClient,
		audit:       auditSvc,
		svc:         svc,
	}
}

// expectAuditLog arms the audit service for scenarios that mutate a user.
func (d *testDeps) expectAuditLog() {
	d.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil)
}

func newTestUser() *tenant.User {
	return &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		CurrentOrganizationID: pulid.MustNew("org_"),
		Status:                domaintypes.StatusActive,
		Name:                  "Test User",
		Username:              "testuser",
		EmailAddress:          "test@example.com",
		Timezone:              "America/New_York",
		Version:               1,
	}
}

func newTestUserWithPassword(t *testing.T, rawPassword string) *tenant.User {
	t.Helper()

	user := newTestUser()
	hashedPassword, err := user.GeneratePassword(rawPassword)
	require.NoError(t, err)
	user.Password = hashedPassword

	return user
}

func TestList_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*tenant.User]{
		Items:      []*tenant.User{newTestUser()},
		Total:      1,
		TotalExact: true,
	}
	req := &tenantports.ListUsersRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.List(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Items, 1)
	deps.repo.AssertExpectations(t)
}

func TestGetByID_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestUser()

	req := tenantports.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity.CurrentOrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: entity.ID,
		},
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)

	result, err := deps.svc.GetByID(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	deps.repo.AssertExpectations(t)
}

func TestSelectOptions_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*tenant.User]{
		Items:      []*tenant.User{newTestUser()},
		Total:      1,
		TotalExact: true,
	}
	req := &pagination.SelectQueryRequest{
		Query: "test",
	}

	deps.repo.On("SelectOptions", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.SelectOptions(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	deps.repo.AssertExpectations(t)
}

func TestBulkUpdateStatus_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	user1 := newTestUser()
	user2 := newTestUser()
	userIDs := []pulid.ID{user1.ID, user2.ID}
	tenantInfo := pagination.TenantInfo{
		OrgID:  user1.CurrentOrganizationID,
		BuID:   user1.BusinessUnitID,
		UserID: pulid.MustNew("usr_"),
	}

	req := &tenantports.BulkUpdateUserStatusRequest{
		TenantInfo: tenantInfo,
		UserIDs:    userIDs,
		Status:     domaintypes.StatusInactive,
	}

	originals := []*tenant.User{user1, user2}
	updated := []*tenant.User{
		{
			ID:                    user1.ID,
			Status:                domaintypes.StatusInactive,
			BusinessUnitID:        user1.BusinessUnitID,
			CurrentOrganizationID: user1.CurrentOrganizationID,
		},
		{
			ID:                    user2.ID,
			Status:                domaintypes.StatusInactive,
			BusinessUnitID:        user2.BusinessUnitID,
			CurrentOrganizationID: user2.CurrentOrganizationID,
		},
	}

	deps.repo.On("GetByIDs", mock.Anything, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenantInfo,
		UserIDs:    userIDs,
	}).Return(originals, nil)
	deps.repo.On("BulkUpdateStatus", mock.Anything, req).Return(updated, nil)
	deps.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, user1.ID).Return(nil).Once()
	deps.sessionRepo.EXPECT().DeleteAllForUser(mock.Anything, user2.ID).Return(nil).Once()
	deps.audit.On("LogActions", mock.Anything).Return(nil)

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestBulkUpdateStatus_GetByIDsError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	userIDs := []pulid.ID{pulid.MustNew("usr_")}

	req := &tenantports.BulkUpdateUserStatusRequest{
		TenantInfo: tenantInfo,
		UserIDs:    userIDs,
		Status:     domaintypes.StatusInactive,
	}

	deps.repo.On("GetByIDs", mock.Anything, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenantInfo,
		UserIDs:    userIDs,
	}).Return(nil, errors.New("get by ids failed"))

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "get by ids failed", err.Error())
	deps.repo.AssertNotCalled(t, "BulkUpdateStatus")
	deps.repo.AssertExpectations(t)
}

func TestBulkUpdateStatus_RepoError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	user1 := newTestUser()
	userIDs := []pulid.ID{user1.ID}
	tenantInfo := pagination.TenantInfo{
		OrgID:  user1.CurrentOrganizationID,
		BuID:   user1.BusinessUnitID,
		UserID: pulid.MustNew("usr_"),
	}

	req := &tenantports.BulkUpdateUserStatusRequest{
		TenantInfo: tenantInfo,
		UserIDs:    userIDs,
		Status:     domaintypes.StatusInactive,
	}

	deps.repo.On("GetByIDs", mock.Anything, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenantInfo,
		UserIDs:    userIDs,
	}).Return([]*tenant.User{user1}, nil)
	deps.repo.On("BulkUpdateStatus", mock.Anything, req).
		Return(nil, errors.New("bulk update failed"))

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "bulk update failed", err.Error())
	deps.repo.AssertExpectations(t)
	deps.audit.AssertNotCalled(t, "LogActions")
}

func TestGetOrganizations_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	userID := pulid.MustNew("usr_")
	currentOrgID := pulid.MustNew("org_")
	otherOrgID := pulid.MustNew("org_")

	memberships := []*tenant.OrganizationMembership{
		{
			OrganizationID: currentOrgID,
			BusinessUnitID: pulid.MustNew("bu_"),
			IsDefault:      true,
			Organization:   &tenant.Organization{Name: "Current Org"},
		},
		{
			OrganizationID: otherOrgID,
			BusinessUnitID: pulid.MustNew("bu_"),
			IsDefault:      false,
			Organization:   &tenant.Organization{Name: "Other Org"},
		},
	}

	deps.repo.On("GetOrganizations", mock.Anything, userID).Return(memberships, nil)

	result, err := deps.svc.GetOrganizations(ctx, userID, currentOrgID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, currentOrgID, result[0].ID)
	assert.Equal(t, "Current Org", result[0].Name)
	assert.True(t, result[0].IsDefault)
	assert.True(t, result[0].IsCurrent)
	assert.Equal(t, otherOrgID, result[1].ID)
	assert.Equal(t, "Other Org", result[1].Name)
	assert.False(t, result[1].IsDefault)
	assert.False(t, result[1].IsCurrent)
	deps.repo.AssertExpectations(t)
}

func TestGetOrganizations_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	userID := pulid.MustNew("usr_")
	currentOrgID := pulid.MustNew("org_")

	deps.repo.On("GetOrganizations", mock.Anything, userID).Return(nil, errors.New("repo error"))

	result, err := deps.svc.GetOrganizations(ctx, userID, currentOrgID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "repo error", err.Error())
	deps.repo.AssertExpectations(t)
}

func TestSwitchOrganization_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	sessionID := pulid.MustNew("ses_")
	userID := pulid.MustNew("usr_")
	targetOrgID := pulid.MustNew("org_")
	targetBuID := pulid.MustNew("bu_")

	sess := &session.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	req := tenantports.SwitchOrganizationRequest{
		SessionID:      sessionID,
		OrganizationID: targetOrgID,
	}

	memberships := []*tenant.OrganizationMembership{
		{
			OrganizationID: targetOrgID,
			BusinessUnitID: targetBuID,
			IsDefault:      false,
		},
	}

	updatedUser := &tenant.User{
		ID:                    userID,
		CurrentOrganizationID: targetOrgID,
		BusinessUnitID:        targetBuID,
	}

	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(sess, nil)
	deps.repo.On("GetOrganizations", mock.Anything, userID).Return(memberships, nil)
	deps.repo.On("UpdateCurrentOrganization", mock.Anything, userID, targetOrgID, targetBuID).
		Return(nil)
	deps.sessionRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	deps.repo.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			UserID: userID,
			OrgID:  targetOrgID,
			BuID:   targetBuID,
		},
		IncludeMemberships: true,
	}).Return(updatedUser, nil)

	result, err := deps.svc.SwitchOrganization(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, userID, result.ID)
	assert.Equal(t, targetOrgID, result.CurrentOrganizationID)
	deps.sessionRepo.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
}

func TestSwitchOrganization_InvalidSession(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	sessionID := pulid.MustNew("ses_")
	targetOrgID := pulid.MustNew("org_")

	req := tenantports.SwitchOrganizationRequest{
		SessionID:      sessionID,
		OrganizationID: targetOrgID,
	}

	deps.sessionRepo.On("Get", mock.Anything, sessionID).
		Return(nil, errors.New("session not found"))

	result, err := deps.svc.SwitchOrganization(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Invalid session")
	deps.sessionRepo.AssertExpectations(t)
	deps.repo.AssertNotCalled(t, "GetOrganizations")
}

func TestSwitchOrganization_UnauthorizedOrg(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	sessionID := pulid.MustNew("ses_")
	userID := pulid.MustNew("usr_")
	targetOrgID := pulid.MustNew("org_")

	sess := &session.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	req := tenantports.SwitchOrganizationRequest{
		SessionID:      sessionID,
		OrganizationID: targetOrgID,
	}

	memberships := []*tenant.OrganizationMembership{
		{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
	}

	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(sess, nil)
	deps.repo.On("GetOrganizations", mock.Anything, userID).Return(memberships, nil)

	result, err := deps.svc.SwitchOrganization(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "You do not have access to this organization")
	deps.sessionRepo.AssertExpectations(t)
	deps.repo.AssertExpectations(t)
	deps.roleRepo.AssertExpectations(t)
	deps.repo.AssertNotCalled(t, "UpdateCurrentOrganization")
}

func TestUpdateMySettings_Success(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.expectAuditLog()
	ctx := t.Context()
	user := newTestUser()
	tenantInfo := pagination.TenantInfo{
		OrgID:  user.CurrentOrganizationID,
		BuID:   user.BusinessUnitID,
		UserID: user.ID,
	}

	deps.repo.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	}).Return(user, nil).Once()
	deps.repo.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  user.CurrentOrganizationID,
			BuID:   user.BusinessUnitID,
			UserID: user.ID,
		},
		LookupUserID:       user.ID,
		IncludeMemberships: true,
	}).Return(user, nil).Once()
	deps.repo.On("UpdateColumns", mock.Anything, mock.MatchedBy(func(req tenantports.UpdateUserColumnsRequest) bool {
		updated := req.Entity
		return assert.ObjectsAreEqual([]string{"timezone", "time_format"}, req.Columns) &&
			updated.Name == user.Name &&
			updated.Username == user.Username &&
			updated.EmailAddress == user.EmailAddress &&
			updated.Timezone == "America/Chicago" &&
			updated.TimeFormat == domaintypes.TimeFormat24Hour
	})).Return(func(_ context.Context, req tenantports.UpdateUserColumnsRequest) *tenant.User {
		return req.Entity
	}, func(context.Context, tenantports.UpdateUserColumnsRequest) error {
		return nil
	}).Once()

	result, err := deps.svc.UpdateMySettings(ctx, tenantInfo, UpdateMySettingsRequest{
		Timezone:   "America/Chicago",
		TimeFormat: domaintypes.TimeFormat24Hour,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, user.Name, result.Name)
	assert.Equal(t, user.Username, result.Username)
	assert.Equal(t, user.EmailAddress, result.EmailAddress)
	assert.Equal(t, domaintypes.TimeFormat24Hour, result.TimeFormat)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertCalled(t, "LogAction", mock.Anything, mock.Anything)
}

func TestChangeMyPassword_Success(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.expectAuditLog()
	ctx := t.Context()
	user := newTestUserWithPassword(t, "current-password")
	user.MustChangePassword = true
	tenantInfo := pagination.TenantInfo{
		OrgID:  user.CurrentOrganizationID,
		BuID:   user.BusinessUnitID,
		UserID: user.ID,
	}

	deps.repo.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo: tenantInfo,
	}).Return(user, nil).Once()
	deps.repo.On("UpdatePassword", mock.Anything, mock.MatchedBy(func(req tenantports.UpdateUserPasswordRequest) bool {
		return req.UserID == user.ID &&
			req.OrganizationID == user.CurrentOrganizationID &&
			req.BusinessUnitID == user.BusinessUnitID &&
			req.MustChangePassword == false &&
			req.Password != "" &&
			req.Password != "new-password"
	})).
		Return(nil).
		Once()
	deps.repo.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	}).Return(user, nil).Once()

	result, err := deps.svc.ChangeMyPassword(ctx, tenantInfo, ChangeMyPasswordRequest{
		CurrentPassword: "current-password",
		NewPassword:     "new-password",
		ConfirmPassword: "new-password",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, user.ID, result.ID)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertCalled(t, "LogAction", mock.Anything, mock.Anything)
}

func TestChangeMyPassword_WrongCurrentPassword(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	ctx := t.Context()
	user := newTestUserWithPassword(t, "current-password")
	tenantInfo := pagination.TenantInfo{
		OrgID:  user.CurrentOrganizationID,
		BuID:   user.BusinessUnitID,
		UserID: user.ID,
	}

	deps.repo.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo: tenantInfo,
	}).Return(user, nil).Once()

	result, err := deps.svc.ChangeMyPassword(ctx, tenantInfo, ChangeMyPasswordRequest{
		CurrentPassword: "wrong-password",
		NewPassword:     "new-password",
		ConfirmPassword: "new-password",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "UpdatePassword")
	deps.audit.AssertNotCalled(t, "LogAction", mock.Anything, mock.Anything)
}

func TestChangeMyPassword_ValidationFailure(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)

	result, err := deps.svc.ChangeMyPassword(
		t.Context(),
		pagination.TenantInfo{},
		ChangeMyPasswordRequest{
			CurrentPassword: "same-password",
			NewPassword:     "same-password",
			ConfirmPassword: "different-password",
		},
	)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "GetByID")
	deps.repo.AssertNotCalled(t, "UpdatePassword")
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := tenantportsmock.NewMockUserRepository(t)
	sessionRepo := sessionportsmock.NewMockSessionRepository(t)
	storageClient := kernelportsmock.NewMockClient(t)
	auditSvc := auditportsmock.NewMockAuditService(t)
	validator := newTestValidator()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			MaxFileSize:        5 * 1024 * 1024,
			PresignedURLExpiry: 15 * time.Minute,
			AllowedMIMETypes:   []string{"image/jpeg", "image/png", "image/webp"},
		},
	}

	svc := New(Params{
		Logger:            zap.NewNop(),
		Repo:              repo,
		SessionRepository: sessionRepo,
		AuditService:      auditSvc,
		Realtime:          &mocks.NoopRealtimeService{},
		Storage:           storageClient,
		Config:            cfg,
		Validator:         validator,
	})

	require.NotNil(t, svc)
}

func TestNewTestValidator(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	require.NotNil(t, v)
}

func TestChangeMyPassword_RefusesAPasswordThePolicyRejects(t *testing.T) {
	t.Parallel()

	user := newTestUserWithPassword(t, "current-password")
	localPart, _, _ := strings.Cut(user.EmailAddress, "@")
	for name, newPassword := range map[string]string{
		"too short":          "short-pw9",
		"common":             "password1234",
		"contains the email": "x-" + localPart + "-2026!",
		"over 72 bytes":      strings.Repeat("ab1-", 19),
		"only spaces":        strings.Repeat(" ", 14),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deps := setupTest(t)
			tenantInfo := pagination.TenantInfo{
				OrgID:  user.CurrentOrganizationID,
				BuID:   user.BusinessUnitID,
				UserID: user.ID,
			}
			deps.repo.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
				TenantInfo: tenantInfo,
			}).Return(user, nil).Once()

			result, err := deps.svc.ChangeMyPassword(t.Context(), tenantInfo, ChangeMyPasswordRequest{
				CurrentPassword: "current-password",
				NewPassword:     newPassword,
				ConfirmPassword: newPassword,
			})

			require.Nil(t, result)
			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			require.Len(t, multiErr.Errors, 1)
			assert.Equal(t, "newPassword", multiErr.Errors[0].Field)
			deps.repo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything)
		})
	}
}
