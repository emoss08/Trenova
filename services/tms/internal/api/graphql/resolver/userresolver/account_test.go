package userresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingMFA struct {
	sessionports.MFAService

	statusTenant pagination.TenantInfo
	statusAAL    int
	begin        *sessionports.BeginTOTPEnrollmentRequest
	confirm      *sessionports.ConfirmTOTPEnrollmentRequest
	disable      *sessionports.DisableTOTPRequest
	regenerate   *sessionports.RegenerateRecoveryCodesRequest
	reset        *sessionports.ResetUserMFARequest
}

func (m *recordingMFA) Status(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	aal int,
) (*sessionports.MFAStatus, error) {
	m.statusTenant = tenantInfo
	m.statusAAL = aal
	return &sessionports.MFAStatus{TOTPEnabled: true, RecoveryCodesRemaining: 4}, nil
}

func (m *recordingMFA) BeginTOTPEnrollment(
	_ context.Context,
	req *sessionports.BeginTOTPEnrollmentRequest,
) (*sessionports.TOTPEnrollment, error) {
	m.begin = req
	return &sessionports.TOTPEnrollment{Secret: "SECRET"}, nil
}

func (m *recordingMFA) ConfirmTOTPEnrollment(
	_ context.Context,
	req *sessionports.ConfirmTOTPEnrollmentRequest,
) (*sessionports.RecoveryCodesResponse, error) {
	m.confirm = req
	return &sessionports.RecoveryCodesResponse{RecoveryCodes: []string{"a", "b"}}, nil
}

func (m *recordingMFA) DisableTOTP(_ context.Context, req *sessionports.DisableTOTPRequest) error {
	m.disable = req
	return nil
}

func (m *recordingMFA) RegenerateRecoveryCodes(
	_ context.Context,
	req *sessionports.RegenerateRecoveryCodesRequest,
) (*sessionports.RecoveryCodesResponse, error) {
	m.regenerate = req
	return &sessionports.RecoveryCodesResponse{RecoveryCodes: []string{"c"}}, nil
}

func (m *recordingMFA) ResetUserMFA(
	_ context.Context,
	req *sessionports.ResetUserMFARequest,
) error {
	m.reset = req
	return nil
}

func accountAuth() *authctx.AuthContext {
	userID := pulid.MustNew("usr_")
	return &authctx.AuthContext{
		PrincipalType:    authctx.PrincipalTypeUser,
		PrincipalID:      userID,
		UserID:           userID,
		OrganizationID:   pulid.MustNew("org_"),
		BusinessUnitID:   pulid.MustNew("bu_"),
		SessionID:        pulid.MustNew("ses_"),
		AuthenticatorAAL: 2,
	}
}

func accountDeps(engine *resolvertest.GrantingPermissionEngine, mfa *recordingMFA) *Deps {
	return &Deps{
		Core:       &resolvercore.Core{L: zap.NewNop(), PermissionEngine: engine},
		MFAService: mfa,
	}
}

func TestAdministrativeUserResolversRefuseWithoutTheirPermission(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_").String()
	cases := []struct {
		name string
		op   permission.Operation
		call func(ctx context.Context, d *Deps) error
	}{
		{"bulkUpdateUserStatus", permission.OpUpdate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).BulkUpdateUserStatus(
				ctx,
				gqlmodel.BulkUpdateUserStatusInput{
					UserIds: []string{userID},
					Status:  domaintypes.StatusInactive,
				},
			)
			return err
		}},
		{"resetUserPassword", permission.OpUpdate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).ResetUserPassword(ctx, userID)
			return err
		}},
		{
			"replaceUserOrganizationMemberships",
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).ReplaceUserOrganizationMemberships(
					ctx,
					userID,
					nil,
				)
				return err
			},
		},
		{"resetUserMfa", permission.OpUpdate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).ResetUserMfa(ctx, userID)
			return err
		}},
		{"userRoleAssignments", permission.OpRead, func(ctx context.Context, d *Deps) error {
			_, err := (&QueryResolver{d}).UserRoleAssignments(ctx, userID)
			return err
		}},
		{
			"userOrganizationMemberships",
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := (&QueryResolver{d}).UserOrganizationMemberships(ctx, userID)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := &resolvertest.GrantingPermissionEngine{}
			mfa := &recordingMFA{}
			ctx := gqlctx.WithAuthContext(t.Context(), accountAuth())

			err := tc.call(ctx, accountDeps(engine, mfa))

			require.Error(t, err)
			assert.True(t, errortypes.IsAuthorizationError(err))
			require.Len(t, engine.Requests, 1)
			assert.Equal(t, permission.ResourceUser.String(), engine.Requests[0].Resource)
			assert.Equal(t, tc.op, engine.Requests[0].Operation)
			assert.Nil(t, mfa.reset)
		})
	}
}

func TestResetUserMfaResetsTheNamedPersonWithTheCallersTenant(t *testing.T) {
	t.Parallel()

	auth := accountAuth()
	engine := &resolvertest.GrantingPermissionEngine{
		Granted: map[string]bool{
			permission.ResourceUser.String() + "|" + string(permission.OpUpdate): true,
		},
	}
	mfa := &recordingMFA{}
	target := pulid.MustNew("usr_")

	ok, err := (&MutationResolver{accountDeps(engine, mfa)}).ResetUserMfa(
		gqlctx.WithAuthContext(t.Context(), auth),
		target.String(),
	)

	require.NoError(t, err)
	assert.True(t, ok)
	require.NotNil(t, mfa.reset)
	assert.Equal(t, target, mfa.reset.TargetUserID)
	assert.Equal(t, auth.OrganizationID, mfa.reset.TenantInfo.OrgID)
	assert.Equal(t, auth.UserID, mfa.reset.TenantInfo.UserID)
}

func TestMyTwoFactorResolversActOnlyOnTheCaller(t *testing.T) {
	t.Parallel()

	auth := accountAuth()
	mfa := &recordingMFA{}
	deps := accountDeps(&resolvertest.GrantingPermissionEngine{}, mfa)
	ctx := gqlctx.WithAuthContext(t.Context(), auth)
	name := "Phone"
	code := "123456"

	status, err := (&QueryResolver{deps}).MyMfaStatus(ctx)
	require.NoError(t, err)
	assert.True(t, status.TOTPEnabled)
	assert.Equal(t, auth.UserID, mfa.statusTenant.UserID)
	assert.Equal(t, 2, mfa.statusAAL)

	enrollment, err := (&MutationResolver{deps}).BeginMyTotpEnrollment(ctx, "hunter2")
	require.NoError(t, err)
	assert.Equal(t, "SECRET", enrollment.Secret)
	assert.Equal(t, "hunter2", mfa.begin.Password)
	assert.Equal(t, auth.UserID, mfa.begin.TenantInfo.UserID)

	codes, err := (&MutationResolver{deps}).ConfirmMyTotpEnrollment(ctx, "654321", &name)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, codes.RecoveryCodes)
	assert.Equal(t, auth.SessionID, mfa.confirm.SessionID)
	assert.Equal(t, "654321", mfa.confirm.Code)
	assert.Equal(t, "Phone", mfa.confirm.Name)

	disabled, err := (&MutationResolver{deps}).DisableMyTotp(ctx, gqlmodel.DisableMyTotpInput{
		Password: "hunter2",
		Code:     &code,
	})
	require.NoError(t, err)
	assert.True(t, disabled)
	assert.Equal(t, "hunter2", mfa.disable.Password)
	assert.Equal(t, "123456", mfa.disable.Code)
	assert.Empty(t, mfa.disable.RecoveryCode)
	assert.Equal(t, auth.UserID, mfa.disable.TenantInfo.UserID)

	regenerated, err := (&MutationResolver{deps}).RegenerateMyRecoveryCodes(ctx, "111111")
	require.NoError(t, err)
	assert.Equal(t, []string{"c"}, regenerated.RecoveryCodes)
	assert.Equal(t, "111111", mfa.regenerate.Code)
}

func TestMyTwoFactorResolversRefuseAnAgent(t *testing.T) {
	t.Parallel()

	auth := accountAuth()
	auth.PrincipalType = authctx.PrincipalTypeAgent
	mfa := &recordingMFA{}
	deps := accountDeps(&resolvertest.GrantingPermissionEngine{}, mfa)

	_, err := (&MutationResolver{deps}).BeginMyTotpEnrollment(
		gqlctx.WithAuthContext(t.Context(), auth),
		"hunter2",
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Nil(t, mfa.begin)
}

func TestMySignInsRefusesANonPositiveLimit(t *testing.T) {
	t.Parallel()

	zero := 0
	_, err := (&QueryResolver{accountDeps(&resolvertest.GrantingPermissionEngine{}, nil)}).MySignIns(
		gqlctx.WithAuthContext(t.Context(), accountAuth()),
		&zero,
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestAvailabilityUpdateFromInputMapsTheHandOff(t *testing.T) {
	t.Parallel()

	delegate := pulid.MustNew("usr_")
	focus := 1_700_000_000
	raw := " " + delegate.String() + " "
	note := "Back Monday"

	update, err := availabilityUpdateFromInput(&gqlmodel.UpdateMyAvailabilityInput{
		Status:         tenant.AvailabilityAway,
		FocusUntil:     &focus,
		DelegateUserID: &raw,
		AwayNote:       &note,
	})

	require.NoError(t, err)
	assert.Equal(t, tenant.AvailabilityAway, update.Status)
	require.NotNil(t, update.FocusUntil)
	assert.Equal(t, int64(focus), *update.FocusUntil)
	assert.Nil(t, update.BackOn)
	require.NotNil(t, update.DelegateUserID)
	assert.Equal(t, delegate, *update.DelegateUserID)
	assert.Equal(t, "Back Monday", update.AwayNote)

	blank := "  "
	update, err = availabilityUpdateFromInput(&gqlmodel.UpdateMyAvailabilityInput{
		Status:         tenant.AvailabilityAvailable,
		DelegateUserID: &blank,
	})
	require.NoError(t, err)
	assert.Nil(t, update.DelegateUserID)

	bad := "not-an-id"
	_, err = availabilityUpdateFromInput(&gqlmodel.UpdateMyAvailabilityInput{
		Status:         tenant.AvailabilityAway,
		DelegateUserID: &bad,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestAvatarIDsFromInputBoundsTheRequest(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("usr_")
	ids, err := avatarIDsFromInput([]string{first.String(), " ", ""})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{first}, ids)

	tooMany := make([]string, maxAvatarIDs+1)
	for i := range tooMany {
		tooMany[i] = pulid.MustNew("usr_").String()
	}
	_, err = avatarIDsFromInput(tooMany)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))

	_, err = avatarIDsFromInput([]string{"nope"})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestSelfUserReportsTheSessionTenant(t *testing.T) {
	t.Parallel()

	auth := accountAuth()
	user := selfUser(auth, &tenant.User{
		CurrentOrganizationID: pulid.MustNew("org_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
	})

	assert.Equal(t, auth.OrganizationID, user.CurrentOrganizationID)
	assert.Equal(t, auth.BusinessUnitID, user.BusinessUnitID)
}
