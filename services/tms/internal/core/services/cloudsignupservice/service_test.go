package cloudsignupservice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/cloudsignup"
	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const strongPassword = "Haul-the-freight-2026!"

type fakeDB struct{}

func (fakeDB) DB() *bun.DB { return nil }

func (fakeDB) DBForContext(context.Context) bun.IDB { return nil }

func (fakeDB) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	txCtx, hooks := ports.WithAfterCommitHooks(ctx)
	if err := fn(txCtx, bun.Tx{}); err != nil {
		return err
	}
	hooks.Run(ctx)
	return nil
}

func (fakeDB) HealthCheck(context.Context) error { return nil }

func (fakeDB) IsHealthy(context.Context) bool { return true }

func (fakeDB) Close() error { return nil }

type recordingEvents struct {
	mu      sync.Mutex
	records []services.AuthEventRecord
}

func (r *recordingEvents) Record(_ context.Context, rec *services.AuthEventRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, *rec)
}

func (r *recordingEvents) providers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.records))
	for _, rec := range r.records {
		out = append(out, rec.Provider+":"+rec.ErrorCode)
	}
	return out
}

type recordingAuditor struct {
	mu      sync.Mutex
	changes []services.SecurityChange
}

func (r *recordingAuditor) RecordChange(_ context.Context, change *services.SecurityChange) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.changes = append(r.changes, *change)
}

type deps struct {
	signups       *mocks.MockCloudSignupRepository
	subscriptions *mocks.MockSubscriptionRepository
	onboarding    *mocks.MockOnboardingRepository
	bootstrap     *mocks.MockTenantBootstrapRepository
	users         *mocks.MockUserRepository
	plans         *mocks.MockPlanService
	auth          *mocks.MockAuthService
	turnstile     *mocks.MockTurnstileVerifier
	email         *mocks.MockPlatformEmailService
	events        *recordingEvents
	auditor       *recordingAuditor
	cfg           *config.Config
	svc           *Service
}

func testConfig() *config.Config {
	return &config.Config{
		Platform: config.PlatformConfig{
			Mode: config.PlatformModeCloud,
			Cloud: config.PlatformCloudConfig{
				Signup: config.CloudSignupConfig{
					Enabled:              true,
					MaxActiveTenants:     10,
					MaxSignupsPerDay:     5,
					PerIPPerHour:         3,
					VerificationTokenTTL: time.Hour,
					BlockDisposableEmail: true,
				},
				Turnstile: config.CloudTurnstileConfig{Enabled: true},
				Trial: config.CloudTrialConfig{
					Lifetime:      720 * time.Hour,
					ReadOnlyGrace: 336 * time.Hour,
				},
			},
		},
	}
}

func setup(t *testing.T, mutate ...func(*config.Config)) *deps {
	t.Helper()

	cfg := testConfig()
	for _, fn := range mutate {
		fn(cfg)
	}

	d := &deps{
		signups:       mocks.NewMockCloudSignupRepository(t),
		subscriptions: mocks.NewMockSubscriptionRepository(t),
		onboarding:    mocks.NewMockOnboardingRepository(t),
		bootstrap:     mocks.NewMockTenantBootstrapRepository(t),
		users:         mocks.NewMockUserRepository(t),
		plans:         mocks.NewMockPlanService(t),
		auth:          mocks.NewMockAuthService(t),
		turnstile:     mocks.NewMockTurnstileVerifier(t),
		email:         mocks.NewMockPlatformEmailService(t),
		events:        &recordingEvents{},
		auditor:       &recordingAuditor{},
		cfg:           cfg,
	}

	d.svc = New(Params{
		DB:            fakeDB{},
		Signups:       d.signups,
		Subscriptions: d.subscriptions,
		Onboarding:    d.onboarding,
		Bootstrap:     d.bootstrap,
		Users:         d.users,
		Plans:         d.plans,
		Auth:          d.auth,
		AuthEvents:    d.events,
		Security:      d.auditor,
		Turnstile:     d.turnstile,
		Email:         d.email,
		Config:        cfg,
		Logger:        zap.NewNop(),
	}).(*Service)

	return d
}

func ctxWithMeta(t *testing.T) context.Context {
	t.Helper()
	return requestmeta.With(t.Context(), requestmeta.New("req-42", "198.51.100.4", "agent/1.0"))
}

func (d *deps) passTurnstile(action string) {
	d.turnstile.EXPECT().Enabled().Return(true)
	d.turnstile.EXPECT().Verify(mock.Anything, &services.TurnstileVerification{
		Token:          "ts-token",
		RemoteIP:       "198.51.100.4",
		ExpectedAction: action,
	}).Return(nil)
}

func (d *deps) noUser(email string) {
	d.users.EXPECT().FindByEmail(mock.Anything, email).Return(
		nil,
		errortypes.NewValidationError("emailAddress", errortypes.ErrNotFound, "not found"),
	)
}

func validSignup() *services.CloudSignupRequest {
	return &services.CloudSignupRequest{
		Name:           "  Dana   Whitfield ",
		EmailAddress:   "Dana.Whitfield+trial@Gmail.com",
		Password:       strongPassword,
		CompanyName:    "Acme Freight, LLC",
		AcceptTerms:    true,
		TurnstileToken: "ts-token",
	}
}

func fieldsOf(t *testing.T, err error) []string {
	t.Helper()
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	return fields
}

func TestSignupDisabled(t *testing.T) {
	t.Parallel()

	d := setup(t, func(cfg *config.Config) { cfg.Platform.Cloud.Signup.Enabled = false })
	_, err := d.svc.Signup(t.Context(), validSignup())
	require.True(t, errortypes.IsNotFoundError(err))
	_, err = d.svc.Resend(t.Context(), &services.CloudSignupResendRequest{})
	require.True(t, errortypes.IsNotFoundError(err))
	_, err = d.svc.Verify(t.Context(), &services.CloudSignupVerifyRequest{Token: "x"})
	require.True(t, errortypes.IsNotFoundError(err))

	d = setup(t, func(cfg *config.Config) { cfg.Platform.Mode = config.PlatformModeSelfHosted })
	assert.False(t, d.svc.Enabled())
}

func TestSignupHoneypotIsAcceptedAndDropped(t *testing.T) {
	t.Parallel()

	d := setup(t)
	req := validSignup()
	req.Website = "https://spam.example"

	resp, err := d.svc.Signup(ctxWithMeta(t), req)
	require.NoError(t, err)
	assert.Equal(t, "pending", resp.Status)
	assert.Equal(t, []string{"signup_rejected:honeypot"}, d.events.providers())
}

func TestSignupTurnstileFailures(t *testing.T) {
	t.Parallel()

	for name, verifyErr := range map[string]error{
		"rejected":    services.ErrTurnstileRejected,
		"unavailable": services.ErrTurnstileUnavailable,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			d := setup(t)
			d.turnstile.EXPECT().Enabled().Return(true)
			d.turnstile.EXPECT().Verify(mock.Anything, mock.Anything).Return(verifyErr)

			_, err := d.svc.Signup(ctxWithMeta(t), validSignup())
			var fieldErr *errortypes.Error
			require.ErrorAs(t, err, &fieldErr)
			assert.Equal(t, "turnstileToken", fieldErr.Field)
		})
	}
}

func TestSignupValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*services.CloudSignupRequest)
		cfg    func(*config.Config)
		fields []string
	}{
		{
			name: "everything missing",
			mutate: func(r *services.CloudSignupRequest) {
				r.Name, r.CompanyName, r.EmailAddress, r.Password, r.AcceptTerms = "", "", "", "", false
			},
			fields: []string{"name", "companyName", "emailAddress", "acceptTerms", "password"},
		},
		{
			name:   "disposable domain",
			mutate: func(r *services.CloudSignupRequest) { r.EmailAddress = "dana@mailinator.com" },
			fields: []string{"emailAddress"},
		},
		{
			name:   "outside the allowlist",
			mutate: func(*services.CloudSignupRequest) {},
			cfg: func(cfg *config.Config) {
				cfg.Platform.Cloud.Signup.AllowedEmailDomains = []string{"trenova.app"}
			},
			fields: []string{"emailAddress"},
		},
		{
			name:   "short password",
			mutate: func(r *services.CloudSignupRequest) { r.Password = "short-pass" },
			fields: []string{"password"},
		},
		{
			name:   "password contains the local part",
			mutate: func(r *services.CloudSignupRequest) { r.Password = "x-dana.whitfield+trial-9" },
			fields: []string{"password"},
		},
		{
			name:   "common password",
			mutate: func(r *services.CloudSignupRequest) { r.Password = "password1234" },
			fields: []string{"password"},
		},
		{
			name:   "password beyond bcrypt",
			mutate: func(r *services.CloudSignupRequest) { r.Password = strings.Repeat("Ab3!", 19) },
			fields: []string{"password"},
		},
		{
			name:   "long company name",
			mutate: func(r *services.CloudSignupRequest) { r.CompanyName = strings.Repeat("a", 101) },
			fields: []string{"companyName"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mutations := []func(*config.Config){}
			if tt.cfg != nil {
				mutations = append(mutations, tt.cfg)
			}
			d := setup(t, mutations...)
			d.passTurnstile(services.TurnstileActionSignup)
			req := validSignup()
			tt.mutate(req)

			_, err := d.svc.Signup(ctxWithMeta(t), req)
			assert.ElementsMatch(t, tt.fields, fieldsOf(t, err))
			assert.Equal(t, []string{"signup_rejected:invalid_input"}, d.events.providers())
		})
	}
}

func TestSignupForExistingUserSendsNoticeOnly(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.passTurnstile(services.TurnstileActionSignup)
	d.users.EXPECT().FindByEmail(mock.Anything, "dana.whitfield+trial@gmail.com").
		Return(&tenant.User{Name: "Dana Whitfield", Locale: "es"}, nil)
	d.email.EXPECT().SendSignupExistingAccount(mock.Anything, &services.SignupExistingAccountEmail{
		To:     "dana.whitfield+trial@gmail.com",
		Locale: i18n.ES,
		Name:   "Dana Whitfield",
	}).Return(nil)

	ctx := i18n.WithLocale(ctxWithMeta(t), i18n.ZhTW)
	resp, err := d.svc.Signup(ctx, validSignup())
	require.NoError(t, err)
	assert.Equal(t, "pending", resp.Status)
	assert.Equal(t, []string{"signup_rejected:existing_account"}, d.events.providers())
}

func TestExistingAccountNoticeFallsBackToTheRequestLanguage(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.passTurnstile(services.TurnstileActionSignup)
	d.users.EXPECT().FindByEmail(mock.Anything, "dana.whitfield+trial@gmail.com").
		Return(&tenant.User{Name: "Dana Whitfield"}, nil)
	d.email.EXPECT().SendSignupExistingAccount(mock.Anything, mock.MatchedBy(
		func(msg *services.SignupExistingAccountEmail) bool { return msg.Locale == i18n.ZhTW },
	)).Return(nil)

	_, err := d.svc.Signup(i18n.WithLocale(ctxWithMeta(t), i18n.ZhTW), validSignup())
	require.NoError(t, err)
}

func TestSignupCreatesAPendingRequest(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.passTurnstile(services.TurnstileActionSignup)
	d.noUser("dana.whitfield+trial@gmail.com")
	d.signups.EXPECT().GetPendingByEmail(mock.Anything, "danawhitfield@gmail.com").
		Return(nil, errortypes.NewNotFoundError("Signup not found"))

	var created *cloudsignup.CloudSignup
	d.signups.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, entity *cloudsignup.CloudSignup) (*cloudsignup.CloudSignup, error) {
			created = entity
			entity.ID = pulid.MustNew("csu_")
			return entity, nil
		},
	)

	var mailed *services.SignupVerificationEmail
	d.email.EXPECT().SendSignupVerification(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, msg *services.SignupVerificationEmail) error {
			mailed = msg
			return nil
		},
	)

	before := time.Now().Unix()
	_, err := d.svc.Signup(i18n.WithLocale(ctxWithMeta(t), i18n.ES), validSignup())
	require.NoError(t, err)

	require.NotNil(t, mailed)
	assert.Equal(t, i18n.ES, mailed.Locale)
	require.NotNil(t, created)
	assert.Equal(t, "dana.whitfield+trial@gmail.com", created.EmailAddress)
	assert.Equal(t, "danawhitfield@gmail.com", created.EmailNormalized)
	assert.Equal(t, "Dana Whitfield", created.Name)
	assert.Equal(t, "Acme Freight, LLC", created.CompanyName)
	assert.Equal(t, "198.51.100.4", created.ClientIP)
	assert.Equal(t, "agent/1.0", created.UserAgent)
	assert.Equal(t, 1, created.Attempts)
	assert.Equal(t, cloudsignup.StatusPending, created.Status)
	assert.InDelta(t, before+3_600, created.ExpiresAt, 5)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(created.PasswordHash), []byte(strongPassword)))

	require.NotNil(t, mailed)
	assert.Equal(t, created.TokenHash, tokenutils.Hash(mailed.Token))
	assert.Equal(t, created.EmailAddress, mailed.To)
	assert.Equal(t, created.ExpiresAt, mailed.ExpiresAt)
	assert.Equal(t, []string{"signup_requested:"}, d.events.providers())
}

func TestSignupRefreshesAnExistingPendingRequest(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.passTurnstile(services.TurnstileActionSignup)
	d.noUser("dana.whitfield+trial@gmail.com")
	pending := &cloudsignup.CloudSignup{ID: pulid.MustNew("csu_"), Attempts: 2}
	d.signups.EXPECT().GetPendingByEmail(mock.Anything, "danawhitfield@gmail.com").Return(pending, nil)
	d.signups.EXPECT().Refresh(mock.Anything, mock.MatchedBy(func(req *repositories.RefreshCloudSignupRequest) bool {
		return req.ID == pending.ID && req.Name == "Dana Whitfield" && len(req.TokenHash) == 64
	})).Return(&cloudsignup.CloudSignup{ID: pending.ID, EmailAddress: "x@example.com"}, nil)
	d.email.EXPECT().SendSignupVerification(mock.Anything, mock.Anything).Return(nil)

	_, err := d.svc.Signup(ctxWithMeta(t), validSignup())
	require.NoError(t, err)
}

func TestSignupStopsSendingAfterTheAttemptCap(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.passTurnstile(services.TurnstileActionSignup)
	d.noUser("dana.whitfield+trial@gmail.com")
	d.signups.EXPECT().GetPendingByEmail(mock.Anything, mock.Anything).
		Return(&cloudsignup.CloudSignup{ID: pulid.MustNew("csu_"), Attempts: maxSendAttempts}, nil)

	resp, err := d.svc.Signup(ctxWithMeta(t), validSignup())
	require.NoError(t, err)
	assert.Equal(t, "pending", resp.Status)
	assert.Equal(t, []string{"signup_rejected:send_limit_reached"}, d.events.providers())
}

func TestSignupRecoversFromAConcurrentCreate(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.passTurnstile(services.TurnstileActionSignup)
	d.noUser("dana.whitfield+trial@gmail.com")
	pending := &cloudsignup.CloudSignup{ID: pulid.MustNew("csu_"), Attempts: 1}
	d.signups.EXPECT().GetPendingByEmail(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Signup not found")).Once()
	d.signups.EXPECT().Create(mock.Anything, mock.Anything).
		Return(nil, &pgconn.PgError{Code: "23505", ConstraintName: "idx_cloud_signups_pending_email"})
	d.signups.EXPECT().GetPendingByEmail(mock.Anything, mock.Anything).Return(pending, nil).Once()
	d.signups.EXPECT().Refresh(mock.Anything, mock.Anything).Return(pending, nil)
	d.email.EXPECT().SendSignupVerification(mock.Anything, mock.Anything).Return(nil)

	_, err := d.svc.Signup(ctxWithMeta(t), validSignup())
	require.NoError(t, err)
}

func TestSignupEmailFailureStillAnswersAccepted(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.passTurnstile(services.TurnstileActionSignup)
	d.noUser("dana.whitfield+trial@gmail.com")
	d.signups.EXPECT().GetPendingByEmail(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Signup not found"))
	d.signups.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, entity *cloudsignup.CloudSignup) (*cloudsignup.CloudSignup, error) {
			return entity, nil
		},
	)
	d.email.EXPECT().SendSignupVerification(mock.Anything, mock.Anything).
		Return(errors.New("resend down"))

	resp, err := d.svc.Signup(ctxWithMeta(t), validSignup())
	require.NoError(t, err)
	assert.Equal(t, "pending", resp.Status)
}

func TestResend(t *testing.T) {
	t.Parallel()

	t.Run("unknown address answers accepted", func(t *testing.T) {
		t.Parallel()
		d := setup(t)
		d.passTurnstile(services.TurnstileActionSignupResend)
		d.signups.EXPECT().GetPendingByEmail(mock.Anything, "dana@example.com").
			Return(nil, errortypes.NewNotFoundError("Signup not found"))

		resp, err := d.svc.Resend(ctxWithMeta(t), &services.CloudSignupResendRequest{
			EmailAddress:   "Dana@Example.com",
			TurnstileToken: "ts-token",
		})
		require.NoError(t, err)
		assert.Equal(t, "pending", resp.Status)
	})

	t.Run("invalid address is a field error", func(t *testing.T) {
		t.Parallel()
		d := setup(t)
		d.passTurnstile(services.TurnstileActionSignupResend)

		_, err := d.svc.Resend(ctxWithMeta(t), &services.CloudSignupResendRequest{
			EmailAddress:   "nope",
			TurnstileToken: "ts-token",
		})
		var fieldErr *errortypes.Error
		require.ErrorAs(t, err, &fieldErr)
		assert.Equal(t, "emailAddress", fieldErr.Field)
	})

	t.Run("reissues the token", func(t *testing.T) {
		t.Parallel()
		d := setup(t)
		d.passTurnstile(services.TurnstileActionSignupResend)
		pending := &cloudsignup.CloudSignup{ID: pulid.MustNew("csu_"), Attempts: 1}
		d.signups.EXPECT().GetPendingByEmail(mock.Anything, "dana@example.com").Return(pending, nil)

		var touched *repositories.TouchCloudSignupRequest
		d.signups.EXPECT().Touch(mock.Anything, mock.Anything).RunAndReturn(
			func(_ context.Context, req *repositories.TouchCloudSignupRequest) (*cloudsignup.CloudSignup, error) {
				touched = req
				return &cloudsignup.CloudSignup{
					ID:           pending.ID,
					EmailAddress: "dana@example.com",
					TokenHash:    req.TokenHash,
					ExpiresAt:    req.ExpiresAt,
				}, nil
			},
		)
		var mailed *services.SignupVerificationEmail
		d.email.EXPECT().SendSignupVerification(mock.Anything, mock.Anything).RunAndReturn(
			func(_ context.Context, msg *services.SignupVerificationEmail) error {
				mailed = msg
				return nil
			},
		)

		_, err := d.svc.Resend(ctxWithMeta(t), &services.CloudSignupResendRequest{
			EmailAddress:   "dana@example.com",
			TurnstileToken: "ts-token",
		})
		require.NoError(t, err)
		require.NotNil(t, touched)
		assert.Equal(t, touched.TokenHash, tokenutils.Hash(mailed.Token))
		assert.Equal(t, "198.51.100.4", touched.ClientIP)
	})

	t.Run("stops after the cap", func(t *testing.T) {
		t.Parallel()
		d := setup(t)
		d.passTurnstile(services.TurnstileActionSignupResend)
		d.signups.EXPECT().GetPendingByEmail(mock.Anything, mock.Anything).
			Return(&cloudsignup.CloudSignup{Attempts: maxSendAttempts}, nil)

		_, err := d.svc.Resend(ctxWithMeta(t), &services.CloudSignupResendRequest{
			EmailAddress:   "dana@example.com",
			TurnstileToken: "ts-token",
		})
		require.NoError(t, err)
	})
}

func pendingSignup(expiresAt int64) *cloudsignup.CloudSignup {
	return &cloudsignup.CloudSignup{
		ID:              pulid.MustNew("csu_"),
		EmailAddress:    "dana.whitfield@acme-freight.com",
		EmailNormalized: "dana.whitfield@acme-freight.com",
		Name:            "Dana Whitfield",
		CompanyName:     "Acme Freight, LLC",
		PasswordHash:    "$2a$10$hash",
		Status:          cloudsignup.StatusPending,
		ExpiresAt:       expiresAt,
	}
}

func TestVerifyRejectsUnknownTokens(t *testing.T) {
	t.Parallel()

	d := setup(t)
	_, err := d.svc.Verify(t.Context(), &services.CloudSignupVerifyRequest{Token: "  "})
	require.ErrorIs(t, err, errInvalidVerificationToken)

	d.signups.EXPECT().GetPendingByTokenHash(mock.Anything, tokenutils.Hash("tok")).
		Return(nil, errortypes.NewNotFoundError("Signup not found"))

	_, err = d.svc.Verify(t.Context(), &services.CloudSignupVerifyRequest{Token: "tok"})
	require.ErrorIs(t, err, errInvalidVerificationToken)
	assert.Equal(t, []string{
		"signup_rejected:invalid_token",
		"signup_rejected:invalid_token",
	}, d.events.providers())
}

func TestVerifyExpiredTokenRejectsTheSignup(t *testing.T) {
	t.Parallel()

	d := setup(t)
	signup := pendingSignup(time.Now().Unix() - 10)
	d.signups.EXPECT().GetPendingByTokenHash(mock.Anything, mock.Anything).Return(signup, nil)
	d.signups.EXPECT().Reject(mock.Anything, &repositories.RejectCloudSignupRequest{
		ID:     signup.ID,
		Reason: reasonExpiredToken,
	}).Return(nil)

	_, err := d.svc.Verify(t.Context(), &services.CloudSignupVerifyRequest{Token: "tok"})
	require.ErrorIs(t, err, errInvalidVerificationToken)
}

func TestVerifyPausesAtTheDailyCap(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.bootstrap.EXPECT().LockProvisioning(mock.Anything).Return(nil)
	d.signups.EXPECT().GetPendingByTokenHash(mock.Anything, mock.Anything).
		Return(pendingSignup(time.Now().Unix()+600), nil)
	d.signups.EXPECT().CountProvisionedSince(mock.Anything, mock.MatchedBy(func(since int64) bool {
		return since%86_400 == 0 && since <= time.Now().Unix()
	})).Return(5, nil)

	_, err := d.svc.Verify(t.Context(), &services.CloudSignupVerifyRequest{Token: "tok"})
	require.True(t, errortypes.IsPlanRestrictionError(err))
	var restriction *errortypes.PlanRestrictionError
	require.ErrorAs(t, err, &restriction)
	params := restriction.Params()
	assert.Equal(t, errortypes.PlanRestrictionReasonSignupsPaused, params["reason"])
	assert.Equal(t, []string{"signup_rejected:signups_paused"}, d.events.providers())
}

func TestVerifyPausesAtTheActiveTenantCap(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.bootstrap.EXPECT().LockProvisioning(mock.Anything).Return(nil)
	d.signups.EXPECT().GetPendingByTokenHash(mock.Anything, mock.Anything).
		Return(pendingSignup(time.Now().Unix()+600), nil)
	d.signups.EXPECT().CountProvisionedSince(mock.Anything, mock.Anything).Return(0, nil)
	d.subscriptions.EXPECT().CountByStatus(mock.Anything, &repositories.CountSubscriptionsByStatusRequest{
		Statuses: []subscription.Status{subscription.StatusTrialing, subscription.StatusReadOnly},
	}).Return(10, nil)

	_, err := d.svc.Verify(t.Context(), &services.CloudSignupVerifyRequest{Token: "tok"})
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestVerifyRefusesAnAddressThatBecameAUser(t *testing.T) {
	t.Parallel()

	d := setup(t, func(cfg *config.Config) {
		cfg.Platform.Cloud.Signup.MaxActiveTenants = 0
		cfg.Platform.Cloud.Signup.MaxSignupsPerDay = 0
	})
	signup := pendingSignup(time.Now().Unix() + 600)
	d.bootstrap.EXPECT().LockProvisioning(mock.Anything).Return(nil)
	d.signups.EXPECT().GetPendingByTokenHash(mock.Anything, mock.Anything).Return(signup, nil)
	d.users.EXPECT().FindByEmail(mock.Anything, signup.EmailAddress).
		Return(&tenant.User{Name: "Dana"}, nil)
	d.signups.EXPECT().Reject(mock.Anything, &repositories.RejectCloudSignupRequest{
		ID:     signup.ID,
		Reason: reasonEmailInUse,
	}).Return(nil)

	_, err := d.svc.Verify(t.Context(), &services.CloudSignupVerifyRequest{Token: "tok"})
	require.ErrorIs(t, err, errEmailInUse)
}

func TestVerifyTreatsAUserEmailRaceAsInUse(t *testing.T) {
	t.Parallel()

	d := setup(t, func(cfg *config.Config) {
		cfg.Platform.Cloud.Signup.MaxActiveTenants = 0
		cfg.Platform.Cloud.Signup.MaxSignupsPerDay = 0
	})
	signup := pendingSignup(time.Now().Unix() + 600)
	d.bootstrap.EXPECT().LockProvisioning(mock.Anything).Return(nil)
	d.signups.EXPECT().GetPendingByTokenHash(mock.Anything, mock.Anything).Return(signup, nil)
	d.noUser(signup.EmailAddress)
	d.bootstrap.EXPECT().Bootstrap(mock.Anything, mock.Anything).
		Return(nil, &pgconn.PgError{Code: "23505", ConstraintName: usersEmailIndex})
	d.signups.EXPECT().Reject(mock.Anything, mock.Anything).Return(nil)

	_, err := d.svc.Verify(t.Context(), &services.CloudSignupVerifyRequest{Token: "tok"})
	require.ErrorIs(t, err, errEmailInUse)
}

func TestVerifyProvisionsTheWorkspaceAndSignsIn(t *testing.T) {
	t.Parallel()

	d := setup(t)
	signup := pendingSignup(time.Now().Unix() + 600)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	ownerID := pulid.MustNew("usr_")

	d.bootstrap.EXPECT().LockProvisioning(mock.Anything).Return(nil)
	d.signups.EXPECT().GetPendingByTokenHash(mock.Anything, tokenutils.Hash("good-token")).
		Return(signup, nil)
	d.signups.EXPECT().CountProvisionedSince(mock.Anything, mock.Anything).Return(1, nil)
	d.subscriptions.EXPECT().CountByStatus(mock.Anything, mock.Anything).Return(2, nil)
	d.noUser(signup.EmailAddress)

	var bootstrapReq *repositories.BootstrapTenantRequest
	d.bootstrap.EXPECT().Bootstrap(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, req *repositories.BootstrapTenantRequest) (*repositories.BootstrapTenantResult, error) {
			bootstrapReq = req
			org := req.Organization
			org.ID = orgID
			org.BusinessUnitID = buID
			owner := req.Owner
			owner.ID = ownerID
			owner.CurrentOrganizationID = orgID
			owner.BusinessUnitID = buID
			owner.Username = "dana.whitfield"
			return &repositories.BootstrapTenantResult{
				BusinessUnit: &tenant.BusinessUnit{ID: buID},
				Organization: org,
				Owner:        owner,
				AdminRoleID:  pulid.MustNew("rol_"),
			}, nil
		},
	)

	var sub *subscription.Subscription
	d.subscriptions.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, entity *subscription.Subscription) (*subscription.Subscription, error) {
			sub = entity
			return entity, nil
		},
	)
	d.onboarding.EXPECT().Create(mock.Anything, mock.MatchedBy(func(o *onboarding.Onboarding) bool {
		return o.OrganizationID == orgID && o.BusinessUnitID == buID && o.Status == onboarding.StatusPending
	})).RunAndReturn(func(_ context.Context, o *onboarding.Onboarding) (*onboarding.Onboarding, error) {
		return o, nil
	})
	d.signups.EXPECT().MarkProvisioned(mock.Anything, mock.MatchedBy(
		func(req *repositories.MarkCloudSignupProvisionedRequest) bool {
			return req.ID == signup.ID && req.OrganizationID == orgID &&
				req.BusinessUnitID == buID && req.UserID == ownerID && req.VerifiedAt > 0
		},
	)).Return(nil)
	d.plans.EXPECT().Invalidate(orgID).Return()
	d.email.EXPECT().SendWelcome(mock.Anything, mock.MatchedBy(func(msg *services.WelcomeEmail) bool {
		return msg.To == signup.EmailAddress && msg.CompanyName == "Acme Freight, LLC" &&
			msg.Locale == i18n.ZhCN
	})).Return(nil)
	d.auth.EXPECT().CreateSessionForUser(mock.Anything, mock.MatchedBy(
		func(req *services.CreateSessionForUserRequest) bool {
			return req.User.ID == ownerID
		},
	)).Return(&services.LoginResponse{SessionID: "sess", SessionToken: "token"}, nil)

	before := time.Now().Unix()
	resp, err := d.svc.Verify(
		i18n.WithLocale(t.Context(), i18n.ZhCN),
		&services.CloudSignupVerifyRequest{Token: " good-token "},
	)
	require.NoError(t, err)
	assert.Equal(t, "sess", resp.SessionID)

	require.NotNil(t, bootstrapReq)
	assert.Equal(t, "Acme Freight, LLC", bootstrapReq.BusinessUnitName)
	assert.Equal(t, "acme-freight-llc", bootstrapReq.LoginSlugBase)
	assert.Equal(t, "dana.whitfield", bootstrapReq.UsernameBase)
	assert.Equal(t, placeholderState, bootstrapReq.StateAbbreviation)
	assert.Equal(t, onboarding.PlaceholderSCAC, bootstrapReq.Organization.ScacCode)
	assert.Equal(t, onboarding.PlaceholderDOTNumber, bootstrapReq.Organization.DOTNumber)
	assert.Equal(t, onboarding.PlaceholderPostalCode, bootstrapReq.Organization.PostalCode)
	assert.Equal(t, "Acme Freight, LLC", bootstrapReq.Organization.Name)
	assert.True(t, bootstrapReq.Organization.BrokerageEnabled)
	assert.True(t, bootstrapReq.Organization.AssetOperationsEnabled)
	assert.Equal(t, signup.PasswordHash, bootstrapReq.Owner.Password)
	assert.Equal(t, signup.EmailAddress, bootstrapReq.Owner.EmailAddress)
	assert.Equal(t, domaintypes.StatusActive, bootstrapReq.Owner.Status)
	assert.False(t, bootstrapReq.Owner.MustChangePassword)
	assert.Equal(t, string(i18n.ZhCN), bootstrapReq.Owner.Locale)

	require.NotNil(t, sub)
	assert.Equal(t, string(platformplan.PlanKeyFreeDemo), sub.PlanKey)
	assert.Equal(t, subscription.StatusTrialing, sub.Status)
	assert.InDelta(t, before+int64((720*time.Hour).Seconds()), sub.TrialEndsAt, 5)
	assert.Equal(t, sub.TrialEndsAt+int64((336*time.Hour).Seconds()), sub.ReadOnlyUntil)

	require.Len(t, d.auditor.changes, 1)
	change := d.auditor.changes[0]
	assert.Equal(t, orgID, change.OrganizationID)
	assert.Equal(t, ownerID, change.Actor.UserID)
	assert.Equal(t, "assign", string(change.Operation))

	providers := d.events.providers()
	assert.Contains(t, providers, "signup_verified:")
	assert.Contains(t, providers, "signup_provisioned:")
	for _, rec := range d.events.records {
		if rec.Provider == services.AuthEventProviderSignupProvisioned {
			assert.Equal(t, orgID, rec.OrganizationID)
			assert.Equal(t, ownerID, rec.UserID)
			assert.Equal(t, iam.AuthEventOutcomeSuccess, rec.Outcome)
		}
	}
}

func TestUsernameBase(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "dana.whitfield", usernameBase(mustParse(t, "Dana.Whitfield@example.com")))
	assert.Equal(t, "opsfreight", usernameBase(mustParse(t, "ops+freight@example.com")))
	assert.Len(t, usernameBase(mustParse(t, strings.Repeat("a", 40)+"@example.com")), 20)
	assert.Equal(t, "a-b", usernameBase(mustParse(t, "-a-b-@example.com")))
}

func mustParse(t *testing.T, raw string) emailutils.Address {
	t.Helper()
	addr, err := emailutils.Parse(raw)
	require.NoError(t, err)
	return addr
}
