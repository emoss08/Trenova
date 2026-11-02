package userservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/sessiontest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func expectSettingsSave(deps *testDeps, user *tenant.User) pagination.TenantInfo {
	tenantInfo := pagination.TenantInfo{
		OrgID:  user.CurrentOrganizationID,
		BuID:   user.BusinessUnitID,
		UserID: user.ID,
	}
	deps.expectAuditLog()
	deps.repo.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	}).Return(user, nil).Once()
	deps.repo.On("UpdateColumns", mock.Anything, mock.Anything).
		Return(func(_ context.Context, req tenantports.UpdateUserColumnsRequest) *tenant.User {
			return req.Entity
		}, func(context.Context, tenantports.UpdateUserColumnsRequest) error {
			return nil
		}).Once()
	deps.repo.On("GetByID", mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		LookupUserID:       user.ID,
		IncludeMemberships: true,
	}).Return(user, nil).Once()
	return tenantInfo
}

func TestUpdateMySettings_LanguageChangeReachesTheCurrentSession(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	user := newTestUser()
	tenantInfo := expectSettingsSave(deps, user)

	sess := &session.Session{ID: pulid.MustNew("ses_"), UserID: user.ID, Locale: "en"}
	deps.sessionRepo.EXPECT().Update(mock.Anything, sess.ID, mock.Anything).
		RunAndReturn(sessiontest.UpdateStored(sess)).
		Once()

	_, err := deps.svc.UpdateMySettings(t.Context(), tenantInfo, UpdateMySettingsRequest{
		Timezone:   "America/Chicago",
		TimeFormat: domaintypes.TimeFormat24Hour,
		Locale:     "es",
		SessionID:  sess.ID,
	})

	require.NoError(t, err)
	require.Equal(t, "es", sess.Locale)
}

func TestUpdateMySettings_LeavesAnotherUsersSessionAlone(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	user := newTestUser()
	tenantInfo := expectSettingsSave(deps, user)

	sess := &session.Session{ID: pulid.MustNew("ses_"), UserID: pulid.MustNew("usr_"), Locale: "en"}
	deps.sessionRepo.EXPECT().Update(mock.Anything, sess.ID, mock.Anything).
		RunAndReturn(sessiontest.UpdateStored(sess)).
		Once()

	_, err := deps.svc.UpdateMySettings(t.Context(), tenantInfo, UpdateMySettingsRequest{
		Timezone:   "America/Chicago",
		TimeFormat: domaintypes.TimeFormat24Hour,
		Locale:     "es",
		SessionID:  sess.ID,
	})

	require.NoError(t, err)
	require.Equal(t, "en", sess.Locale)
}

func TestUpdateMySettings_SavesTheLanguageWhenTheSessionCannotBeRefreshed(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	user := newTestUser()
	tenantInfo := expectSettingsSave(deps, user)

	sessionID := pulid.MustNew("ses_")
	deps.sessionRepo.EXPECT().Update(mock.Anything, sessionID, mock.Anything).
		Return(nil, context.DeadlineExceeded).Once()

	updated, err := deps.svc.UpdateMySettings(t.Context(), tenantInfo, UpdateMySettingsRequest{
		Timezone:   "America/Chicago",
		TimeFormat: domaintypes.TimeFormat24Hour,
		Locale:     "es",
		SessionID:  sessionID,
	})

	require.NoError(t, err)
	require.Equal(t, "es", updated.Locale)
}
