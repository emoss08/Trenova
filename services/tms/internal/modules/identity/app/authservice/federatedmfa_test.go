package authservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sso"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFederatedMFAIsTrustedOnlyWhenAllowedAndAssertedExactly(t *testing.T) {
	t.Parallel()

	trusting := &sso.SSOConfig{
		AllowFederatedMFA: true,
		FederatedMFAACRs:  []string{"http://schemas.openid.net/pape/policies/2007/06/multi-factor"},
	}
	refusing := &sso.SSOConfig{AllowFederatedMFA: false}

	cases := []struct {
		name   string
		cfg    *sso.SSOConfig
		claims oidcClaims
		want   bool
	}{
		{"mfa amr", trusting, oidcClaims{AMR: []string{"pwd", "mfa"}}, true},
		{"otp amr", trusting, oidcClaims{AMR: []string{"otp"}}, true},
		{"hardware key amr", trusting, oidcClaims{AMR: []string{"hwk"}}, true},
		{"password only", trusting, oidcClaims{AMR: []string{"pwd"}}, false},
		{"amr that only contains mfa", trusting, oidcClaims{AMR: []string{"notmfa"}}, false},
		{"amr with stray whitespace", trusting, oidcClaims{AMR: []string{" mfa"}}, false},
		{"no claims", trusting, oidcClaims{}, false},
		{
			"configured acr",
			trusting,
			oidcClaims{ACR: "http://schemas.openid.net/pape/policies/2007/06/multi-factor"},
			true,
		},
		{"acr containing aal2", trusting, oidcClaims{ACR: "urn:example:aal2"}, false},
		{"acr containing level2", trusting, oidcClaims{ACR: "level2-but-not-really"}, false},
		{"acr silver", trusting, oidcClaims{ACR: "urn:mace:incommon:iap:silver"}, false},
		{"provider not trusted", refusing, oidcClaims{AMR: []string{"mfa"}}, false},
		{"no configuration", nil, oidcClaims{AMR: []string{"mfa"}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, federatedMFASatisfied(tc.cfg, tc.claims))
			aal, mfaAt := assuranceFromOIDCClaims(tc.cfg, tc.claims)
			if tc.want {
				assert.Equal(t, mfaAssuranceLevel, aal)
				assert.Positive(t, mfaAt)
			} else {
				assert.Equal(t, 1, aal)
				assert.Zero(t, mfaAt)
			}
		})
	}
}

func TestSSOChallengeCarriesTheFederationIntoTheSession(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	challenges := newMemoryChallenges()
	deps.svc.mfa = &fakeMFA{active: true, validCode: "123456"}
	deps.svc.challenges = challenges

	attempt := newAuthAttempt(ssoAuthEventProvider(sso.SSOProviderOkta))
	challenge, err := deps.svc.issueMFAChallenge(
		t.Context(),
		usr,
		nil,
		attempt,
		&iamports.MFAChallengeFederation{
			AuthProvider:    string(sso.SSOProviderOkta),
			ExternalSubject: "okta|00u1",
			FederationFAL:   2,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, challenge)
	assert.True(t, challenge.MFARequired)
	assert.Empty(t, challenge.SessionToken)

	var created *session.Session
	deps.userRepo.On("FindByIDForLogin", mock.Anything, usr.ID).Return(usr, nil)
	deps.portalRepo.On("ExistsWorkerForUser", mock.Anything, mock.Anything).Return(false, nil)
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { created = args.Get(1).(*session.Session) }).
		Return(nil)
	deps.userRepo.On("UpdateLastLoginAt", mock.Anything, usr.ID).Return(nil)

	_, err = deps.svc.VerifyMFAChallenge(t.Context(), tenantports.VerifyMFAChallengeRequest{
		ChallengeToken: challenge.MFAChallengeToken,
		Code:           "123456",
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, string(sso.SSOProviderOkta), created.AuthProvider)
	assert.Equal(t, "okta|00u1", created.ExternalSubject)
	assert.Equal(t, 2, created.FederationFAL)
	assert.Equal(t, mfaAssuranceLevel, created.AuthenticatorAAL)
	assert.Positive(t, created.MFAAuthenticatedAt)
}
