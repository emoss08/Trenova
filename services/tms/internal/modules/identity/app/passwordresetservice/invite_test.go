package passwordresetservice

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/accounttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/passwordreset"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttemplateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/emailportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type inviteHarness struct {
	svc      *Service
	users    *stubUserRepo
	tokens   *stubTokenRepo
	rendered []*documenttemplateports.RenderMessageRequest
	sent     int
}

func newInviteHarness(t *testing.T, user *tenant.User) *inviteHarness {
	t.Helper()

	h := &inviteHarness{
		users: &stubUserRepo{
			user:  user,
			actor: &tenant.User{ID: pulid.MustNew("usr_"), Name: "Morgan Ellis"},
		},
		tokens: &stubTokenRepo{},
	}
	h.svc = newService(t, h.users, h.tokens, &stubSessionRepo{})
	h.svc.cfg.Security.PasswordReset.InviteTokenTTL = 72 * time.Hour

	templates := documenttemplateportsmock.NewMockDocumentTemplateResolver(t)
	templates.EXPECT().RenderMessage(mock.Anything, mock.MatchedBy(
		func(req *documenttemplateports.RenderMessageRequest) bool {
			h.rendered = append(h.rendered, req)
			return true
		},
	)).Return(&documenttemplateports.RenderedMessage{
		Subject: "Set up your account",
		HTML:    "<p>invite</p>",
		Text:    "invite",
	}, nil).Maybe()
	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{Name: "Acme"}, nil).Maybe()
	mail := emailportsmock.NewMockEmailService(t)
	mail.EXPECT().Send(mock.Anything, mock.Anything).
		Run(func(context.Context, *emailports.SendEmailRequest) { h.sent++ }).
		Return(nil, nil).Maybe()

	h.svc.templates = templates
	h.svc.orgs = orgs
	h.svc.emailService = mail

	return h
}

func inviteActor() pagination.TenantInfo {
	return pagination.TenantInfo{
		UserID: pulid.MustNew("usr_"),
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
	}
}

func signedInUser() *tenant.User {
	user := activeUser()
	user.LastLoginAt = new(timeutils.NowUnix())
	user.PasswordChangedAt = new(timeutils.NowUnix())
	return user
}

func TestSendInviteMintsAnInviteLinkThatLivesForTheInviteTTL(t *testing.T) {
	t.Parallel()

	user := activeUser()
	h := newInviteHarness(t, user)
	before := timeutils.NowUnix()

	require.NoError(
		t,
		h.svc.SendInvite(t.Context(), InviteRequest{User: user, Actor: inviteActor()}),
	)

	require.Len(t, h.tokens.created, 1)
	token := h.tokens.created[0]
	assert.Equal(t, passwordreset.PasswordResetPurposeInvite, token.Purpose)
	assert.Equal(t, user.ID, token.UserID)
	assert.Equal(t, user.CurrentOrganizationID, token.OrganizationID)
	assert.GreaterOrEqual(t, token.ExpiresAt, before+int64((72*time.Hour).Seconds()))
	assert.LessOrEqual(t, token.ExpiresAt, timeutils.NowUnix()+int64((72*time.Hour).Seconds()))
	assert.Contains(t, h.tokens.invalidated, user.ID)
	assert.Nil(t, h.users.updated, "an invitation never writes a password")
}

func TestSendInviteRendersTheInviteTemplateWithASetupLink(t *testing.T) {
	t.Parallel()

	user := activeUser()
	user.Username = "dwhitfield"
	h := newInviteHarness(t, user)

	require.NoError(
		t,
		h.svc.SendInvite(t.Context(), InviteRequest{User: user, Actor: inviteActor()}),
	)

	require.Len(t, h.rendered, 1)
	assert.Equal(t, 1, h.sent)
	req := h.rendered[0]
	assert.Equal(t, documenttemplate.KindAccountInviteEmail, req.Kind)
	data, ok := req.Data.(accounttemplate.AccountInviteContext)
	require.True(t, ok)
	assert.Equal(t, "Dana", data.FirstName)
	assert.Equal(t, "dwhitfield", data.Username)
	assert.Equal(t, "Morgan Ellis", data.InvitedByName)
	assert.Equal(t, "Acme", data.CompanyName)
	assert.Equal(t, 3, data.ExpiresInDays)
	assert.Equal(t, 72, data.ExpiresInHours)

	link, err := url.Parse(string(data.SetupURL))
	require.NoError(t, err)
	assert.Equal(t, "app.example.com", link.Host)
	assert.Equal(t, "/auth/reset", link.Path)
	assert.Equal(t, "invite", link.Query().Get("purpose"))
	assert.NotEmpty(t, link.Query().Get("token"))
}

func TestSendInviteRefusesAnInactiveAccount(t *testing.T) {
	t.Parallel()

	user := activeUser()
	user.Status = domaintypes.StatusInactive
	h := newInviteHarness(t, user)

	require.Error(t, h.svc.SendInvite(t.Context(), InviteRequest{User: user, Actor: inviteActor()}))
	assert.Empty(t, h.tokens.created)
	assert.Empty(t, h.rendered)
}

func TestRequestResetForUserResendsAnInviteToSomebodyWhoNeverSetUpTheirAccount(t *testing.T) {
	t.Parallel()

	user := activeUser()
	h := newInviteHarness(t, user)

	require.NoError(t, h.svc.RequestResetForUser(t.Context(), adminRequest(user.ID)))

	require.Len(t, h.tokens.created, 1)
	assert.Equal(t, passwordreset.PasswordResetPurposeInvite, h.tokens.created[0].Purpose)
	require.Len(t, h.rendered, 1)
	assert.Equal(t, documenttemplate.KindAccountInviteEmail, h.rendered[0].Kind)
}

func TestRequestResetForUserSendsAResetToSomebodyWhoHasSignedIn(t *testing.T) {
	t.Parallel()

	user := signedInUser()
	h := newInviteHarness(t, user)

	require.NoError(t, h.svc.RequestResetForUser(t.Context(), adminRequest(user.ID)))

	require.Len(t, h.tokens.created, 1)
	assert.Equal(t, passwordreset.PasswordResetPurposeReset, h.tokens.created[0].Purpose)
	require.Len(t, h.rendered, 1)
	assert.Equal(t, documenttemplate.KindPasswordResetEmail, h.rendered[0].Kind)
}

func TestSelfServiceResetAlwaysMintsAResetLink(t *testing.T) {
	t.Parallel()

	user := activeUser()
	h := newInviteHarness(t, user)

	require.NoError(t, h.svc.RequestReset(t.Context(), user.EmailAddress))

	require.Len(t, h.tokens.created, 1)
	assert.Equal(t, passwordreset.PasswordResetPurposeReset, h.tokens.created[0].Purpose)
}

func TestResetPasswordRedeemsAnInviteAndClearsTheForcedChange(t *testing.T) {
	t.Parallel()

	user := activeUser()
	user.MustChangePassword = true
	token := redeemableToken(user)
	token.Purpose = passwordreset.PasswordResetPurposeInvite
	users := &stubUserRepo{user: user}
	tokens := &stubTokenRepo{redeemable: token, markUsedOK: true}
	events := &recordingAuthEvents{}
	svc := newService(t, users, tokens, &stubSessionRepo{})
	svc.authEvents = events

	require.NoError(t, svc.ResetPassword(t.Context(), "raw-token", "a-good-password"))

	require.NotNil(t, users.updated)
	assert.Equal(t, user.ID, users.updated.UserID)
	assert.False(t, users.updated.MustChangePassword)
	assert.NotEqual(t, "a-good-password", users.updated.Password)
	require.Len(t, events.records, 1)
	assert.Equal(t, iamports.AuthEventProviderInviteAccept, events.records[0].Provider)
	assert.Equal(t, iam.AuthEventOutcomeSuccess, events.records[0].Outcome)
	assert.Equal(t, user.ID, events.records[0].UserID)
}

func TestResetPasswordRecordsAResetForAResetLink(t *testing.T) {
	t.Parallel()

	user := activeUser()
	token := redeemableToken(user)
	token.Purpose = passwordreset.PasswordResetPurposeReset
	events := &recordingAuthEvents{}
	svc := newService(
		t,
		&stubUserRepo{user: user},
		&stubTokenRepo{redeemable: token, markUsedOK: true},
		&stubSessionRepo{},
	)
	svc.authEvents = events

	require.NoError(t, svc.ResetPassword(t.Context(), "raw-token", "a-good-password"))

	require.Len(t, events.records, 1)
	assert.Equal(t, iamports.AuthEventProviderPasswordResetConfirm, events.records[0].Provider)
}

func TestLinkURLMarksAnInvite(t *testing.T) {
	t.Parallel()

	svc := newService(t, &stubUserRepo{}, &stubTokenRepo{}, &stubSessionRepo{})

	link, err := svc.linkURL("raw-token", passwordreset.PasswordResetPurposeInvite)

	require.NoError(t, err)
	assert.Equal(t, "https://app.example.com/auth/reset?purpose=invite&token=raw-token", link)
}

func TestWholeDaysOnlyCountsExactDays(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 7, wholeDays(168))
	assert.Equal(t, 0, wholeDays(36))
	assert.Equal(t, 0, wholeDays(0))
}
