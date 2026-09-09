package passwordresetservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubTokenRepo struct {
	created      []*tenant.PasswordResetToken
	redeemable   *tenant.PasswordResetToken
	findErr      error
	markUsedOK   bool
	markUsedErr  error
	invalidated  []pulid.ID
	countSince   int
	countErr     error
	markUsedCall int
}

func (r *stubTokenRepo) Create(_ context.Context, token *tenant.PasswordResetToken) error {
	r.created = append(r.created, token)
	return nil
}

func (r *stubTokenRepo) FindRedeemableByHash(
	_ context.Context,
	_ string,
	_ int64,
) (*tenant.PasswordResetToken, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.redeemable, nil
}

func (r *stubTokenRepo) MarkUsed(_ context.Context, _ pulid.ID, _ int64) (bool, error) {
	r.markUsedCall++
	return r.markUsedOK, r.markUsedErr
}

func (r *stubTokenRepo) InvalidateOutstanding(
	_ context.Context,
	userID pulid.ID,
	_ int64,
) error {
	r.invalidated = append(r.invalidated, userID)
	return nil
}

func (r *stubTokenRepo) CountSince(_ context.Context, _ pulid.ID, _ int64) (int, error) {
	return r.countSince, r.countErr
}

type stubUserRepo struct {
	repositories.UserRepository

	user       *tenant.User
	findErr    error
	getErr     error
	lookedUp   pulid.ID
	updated    *repositories.UpdateUserPasswordRequest
	updateErr  error
	findCalled int
}

func (r *stubUserRepo) GetByID(
	_ context.Context,
	req repositories.GetUserByIDRequest,
) (*tenant.User, error) {
	r.lookedUp = req.LookupUserID
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.user, nil
}

func (r *stubUserRepo) FindByEmail(_ context.Context, _ string) (*tenant.User, error) {
	r.findCalled++
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.user, nil
}

func (r *stubUserRepo) UpdatePassword(
	_ context.Context,
	req repositories.UpdateUserPasswordRequest,
) error {
	r.updated = &req
	return r.updateErr
}

type stubSessionRepo struct {
	repositories.SessionRepository

	deletedFor []pulid.ID
	deleteErr  error
}

func (r *stubSessionRepo) DeleteAllForUser(_ context.Context, userID pulid.ID) error {
	r.deletedFor = append(r.deletedFor, userID)
	return r.deleteErr
}

func activeUser() *tenant.User {
	return &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		CurrentOrganizationID: pulid.MustNew("org_"),
		Status:                domaintypes.StatusActive,
		Name:                  "Dana Whitfield",
		EmailAddress:          "dana@example.com",
	}
}

func newService(
	t *testing.T,
	users *stubUserRepo,
	tokens *stubTokenRepo,
	sessions *stubSessionRepo,
) *Service {
	t.Helper()

	cfg := &config.Config{}
	cfg.Security.PasswordReset.BaseURL = "https://app.example.com"

	return &Service{
		ur:       users,
		tokens:   tokens,
		sessions: sessions,
		cfg:      cfg,
		l:        zap.NewNop(),
	}
}

// The whole point of the token design: a request from a stranger must not touch the
// account. Nothing here writes a password.
func TestRequestReset_LeavesTheAccountUntouched(t *testing.T) {
	t.Parallel()

	users := &stubUserRepo{user: activeUser()}
	tokens := &stubTokenRepo{}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	require.NoError(t, svc.RequestReset(t.Context(), "dana@example.com"))

	assert.Nil(t, users.updated, "requesting a reset must not change the password")
	require.Len(t, tokens.created, 1)
	assert.NotEmpty(t, tokens.created[0].TokenHash)
}

// An unknown address, an inactive account and a rate-limited one must all be answered
// the same way, so the endpoint cannot be used to test whether an address is registered.
func TestRequestReset_IsSilentForEveryRejection(t *testing.T) {
	t.Parallel()

	locked := activeUser()
	locked.IsLocked = true

	inactive := activeUser()
	inactive.Status = domaintypes.StatusInactive

	tests := []struct {
		name   string
		users  *stubUserRepo
		tokens *stubTokenRepo
	}{
		{"unknown address", &stubUserRepo{findErr: errors.New("not found")}, &stubTokenRepo{}},
		{"no user returned", &stubUserRepo{user: nil}, &stubTokenRepo{}},
		{"locked account", &stubUserRepo{user: locked}, &stubTokenRepo{}},
		{"inactive account", &stubUserRepo{user: inactive}, &stubTokenRepo{}},
		{"rate limited", &stubUserRepo{user: activeUser()}, &stubTokenRepo{countSince: 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := newService(t, tt.users, tt.tokens, &stubSessionRepo{})

			require.NoError(t, svc.RequestReset(t.Context(), "dana@example.com"))
			assert.Empty(t, tt.tokens.created, "no token may be minted")
		})
	}
}

func TestRequestReset_RetiresOlderLinksSoOnlyTheNewestWorks(t *testing.T) {
	t.Parallel()

	user := activeUser()
	users := &stubUserRepo{user: user}
	tokens := &stubTokenRepo{}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	require.NoError(t, svc.RequestReset(t.Context(), "dana@example.com"))

	assert.Contains(t, tokens.invalidated, user.ID)
}

func TestRequestReset_StoresOnlyTheDigest(t *testing.T) {
	t.Parallel()

	users := &stubUserRepo{user: activeUser()}
	tokens := &stubTokenRepo{}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	require.NoError(t, svc.RequestReset(t.Context(), "dana@example.com"))

	require.Len(t, tokens.created, 1)
	stored := tokens.created[0].TokenHash
	assert.Len(t, stored, 64, "a SHA-256 hex digest, not a usable token")
	assert.NotEqual(t, "", stored)
}

func TestRequestReset_IgnoresABlankAddress(t *testing.T) {
	t.Parallel()

	users := &stubUserRepo{user: activeUser()}
	svc := newService(t, users, &stubTokenRepo{}, &stubSessionRepo{})

	require.NoError(t, svc.RequestReset(t.Context(), "   "))
	assert.Zero(t, users.findCalled)
}

func redeemableToken(user *tenant.User) *tenant.PasswordResetToken {
	return &tenant.PasswordResetToken{
		ID:             pulid.MustNew("prt_"),
		UserID:         user.ID,
		BusinessUnitID: user.BusinessUnitID,
		OrganizationID: user.CurrentOrganizationID,
		TokenHash:      tokenutils.Hash("raw-token"),
		ExpiresAt:      timeutils.NowUnix() + 600,
		User:           user,
	}
}

func TestResetPassword_SetsThePasswordAndEndsEverySession(t *testing.T) {
	t.Parallel()

	user := activeUser()
	users := &stubUserRepo{user: user}
	tokens := &stubTokenRepo{redeemable: redeemableToken(user), markUsedOK: true}
	sessions := &stubSessionRepo{}
	svc := newService(t, users, tokens, sessions)

	require.NoError(t, svc.ResetPassword(t.Context(), "raw-token", "a-good-password"))

	require.NotNil(t, users.updated)
	assert.NotEqual(t, "a-good-password", users.updated.Password, "the password is hashed")
	assert.False(t, users.updated.MustChangePassword)
	assert.Contains(t, sessions.deletedFor, user.ID)
}

// Two tabs opened from one email must not both reset. The repository reports whether
// the redeeming UPDATE actually matched a row, and a loser must be refused.
func TestResetPassword_RefusesAlreadyRedeemedToken(t *testing.T) {
	t.Parallel()

	user := activeUser()
	users := &stubUserRepo{user: user}
	tokens := &stubTokenRepo{redeemable: redeemableToken(user), markUsedOK: false}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	err := svc.ResetPassword(t.Context(), "raw-token", "a-good-password")

	require.Error(t, err)
	assert.Nil(t, users.updated, "a losing racer must not write a password")
}

func TestResetPassword_RedeemsBeforeWritingThePassword(t *testing.T) {
	t.Parallel()

	user := activeUser()
	users := &stubUserRepo{user: user}
	tokens := &stubTokenRepo{redeemable: redeemableToken(user), markUsedOK: true}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	require.NoError(t, svc.ResetPassword(t.Context(), "raw-token", "a-good-password"))

	assert.Equal(t, 1, tokens.markUsedCall)
}

func TestResetPassword_RejectsAnUnknownOrExpiredToken(t *testing.T) {
	t.Parallel()

	users := &stubUserRepo{user: activeUser()}
	tokens := &stubTokenRepo{findErr: errors.New("not found")}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	err := svc.ResetPassword(t.Context(), "raw-token", "a-good-password")

	require.Error(t, err)
	assert.Nil(t, users.updated)
}

func TestResetPassword_RejectsABlankToken(t *testing.T) {
	t.Parallel()

	svc := newService(t, &stubUserRepo{}, &stubTokenRepo{}, &stubSessionRepo{})

	require.Error(t, svc.ResetPassword(t.Context(), "", "a-good-password"))
}

func TestResetPassword_EnforcesAMinimumPasswordLength(t *testing.T) {
	t.Parallel()

	user := activeUser()
	users := &stubUserRepo{user: user}
	tokens := &stubTokenRepo{redeemable: redeemableToken(user), markUsedOK: true}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	err := svc.ResetPassword(t.Context(), "raw-token", "short")

	require.Error(t, err)
	assert.Nil(t, users.updated)
	assert.Zero(t, tokens.markUsedCall, "a rejected password must not burn the link")
}

func TestResetPassword_RefusesAnInactiveAccount(t *testing.T) {
	t.Parallel()

	user := activeUser()
	user.Status = domaintypes.StatusInactive
	users := &stubUserRepo{user: user}
	tokens := &stubTokenRepo{redeemable: redeemableToken(user), markUsedOK: true}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	require.Error(t, svc.ResetPassword(t.Context(), "raw-token", "a-good-password"))
	assert.Nil(t, users.updated)
}

func TestResetURL_RefusesToGuessTheOrigin(t *testing.T) {
	t.Parallel()

	svc := newService(t, &stubUserRepo{}, &stubTokenRepo{}, &stubSessionRepo{})
	svc.cfg.Security.PasswordReset.BaseURL = ""

	_, err := svc.resetURL("raw-token")

	require.ErrorIs(t, err, errNoResetBaseURL)
}

func TestResetURL_BuildsTheLinkFromTheConfiguredOrigin(t *testing.T) {
	t.Parallel()

	svc := newService(t, &stubUserRepo{}, &stubTokenRepo{}, &stubSessionRepo{})

	url, err := svc.resetURL("raw-token")

	require.NoError(t, err)
	assert.Equal(t, "https://app.example.com/auth/reset?token=raw-token", url)
}

func TestFirstName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Dana", firstName("Dana Whitfield"))
	assert.Equal(t, "Dana", firstName("Dana"))
	assert.Empty(t, firstName("   "))
}

func adminRequest(target pulid.ID) AdminResetRequest {
	return AdminResetRequest{
		TargetUserID: target,
		Actor: pagination.TenantInfo{
			UserID: pulid.MustNew("usr_"),
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
		},
	}
}

// The admin button must not become a way to set somebody's password, or to lock them
// out of one. It mints a link, exactly like the self-service path.
func TestRequestResetForUser_OnlyMintsALink(t *testing.T) {
	t.Parallel()

	user := activeUser()
	users := &stubUserRepo{user: user}
	tokens := &stubTokenRepo{}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	require.NoError(t, svc.RequestResetForUser(t.Context(), adminRequest(user.ID)))

	assert.Nil(t, users.updated, "an administrator reset must not write a password")
	require.Len(t, tokens.created, 1)
	assert.Equal(t, user.ID, tokens.created[0].UserID)
	assert.Contains(t, tokens.invalidated, user.ID)
}

func TestRequestResetForUser_LooksUpTheTargetNotTheActor(t *testing.T) {
	t.Parallel()

	user := activeUser()
	users := &stubUserRepo{user: user}
	svc := newService(t, users, &stubTokenRepo{}, &stubSessionRepo{})

	req := adminRequest(user.ID)
	require.NoError(t, svc.RequestResetForUser(t.Context(), req))

	assert.Equal(t, req.TargetUserID, users.lookedUp)
	assert.NotEqual(t, req.Actor.UserID, users.lookedUp)
}

// The self-service path is deliberately silent about every rejection. This one is not:
// the caller is already authenticated, so there is no oracle to protect, and a button
// that reports success while nothing was sent is worse than an error.
func TestRequestResetForUser_ReportsAMissingUser(t *testing.T) {
	t.Parallel()

	users := &stubUserRepo{getErr: errors.New("not found")}
	tokens := &stubTokenRepo{}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	err := svc.RequestResetForUser(t.Context(), adminRequest(pulid.MustNew("usr_")))

	require.Error(t, err)
	assert.Empty(t, tokens.created)
}

func TestRequestResetForUser_RefusesAnInactiveAccount(t *testing.T) {
	t.Parallel()

	user := activeUser()
	user.Status = domaintypes.StatusInactive
	users := &stubUserRepo{user: user}
	tokens := &stubTokenRepo{}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	require.Error(t, svc.RequestResetForUser(t.Context(), adminRequest(user.ID)))
	assert.Empty(t, tokens.created)
}

// The hourly cap stops an anonymous stranger flooding an inbox. An admin clicking this
// is usually helping somebody who already burned their own allowance, so it must not
// apply to them.
func TestRequestResetForUser_IgnoresTheAnonymousHourlyCap(t *testing.T) {
	t.Parallel()

	user := activeUser()
	users := &stubUserRepo{user: user}
	tokens := &stubTokenRepo{countSince: 99}
	svc := newService(t, users, tokens, &stubSessionRepo{})

	require.NoError(t, svc.RequestResetForUser(t.Context(), adminRequest(user.ID)))

	assert.Len(t, tokens.created, 1)
}
