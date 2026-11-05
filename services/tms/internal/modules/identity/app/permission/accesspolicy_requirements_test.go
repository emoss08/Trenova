package permission

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func policy(effect iam.PolicyEffect, priority int, conditions map[string]string) *iam.AccessPolicy {
	return &iam.AccessPolicy{Effect: effect, Priority: priority, Conditions: conditions}
}

func sessionAt(aal int, mfaAt int64, fal int) *permissionports.PermissionCheckRequest {
	return &permissionports.PermissionCheckRequest{
		PrincipalType: permissionports.PrincipalTypeUser,
		ContextAttributes: permissionports.RequestContextAttributes{
			AuthenticatorAAL:   aal,
			MFAAuthenticatedAt: mfaAt,
			FederationFAL:      fal,
		},
	}
}

func TestEvaluateAccessPolicies_Requirements(t *testing.T) {
	t.Parallel()

	password := sessionAt(1, 0, 1)
	withMFA := sessionAt(2, 1_700_000_000, 1)
	federated := sessionAt(1, 0, 2)
	noSession := &permissionports.PermissionCheckRequest{
		PrincipalType: permissionports.PrincipalTypeUser,
	}
	apiKey := &permissionports.PermissionCheckRequest{
		PrincipalType: permissionports.PrincipalTypeAPIKey,
	}
	requireMFA := map[string]string{"mfaRequired": "true"}

	tests := []struct {
		name     string
		policies []*iam.AccessPolicy
		req      *permissionports.PermissionCheckRequest
		want     iam.PolicyEffect
	}{
		{
			name:     "an Allow requiring MFA refuses a password session",
			policies: []*iam.AccessPolicy{policy(iam.PolicyEffectAllow, 1, requireMFA)},
			req:      password,
			want:     iam.PolicyEffectDeny,
		},
		{
			name:     "an Allow requiring MFA admits an MFA session",
			policies: []*iam.AccessPolicy{policy(iam.PolicyEffectAllow, 1, requireMFA)},
			req:      withMFA,
			want:     iam.PolicyEffectAllow,
		},
		{
			name:     "a Deny requiring MFA refuses a password session",
			policies: []*iam.AccessPolicy{policy(iam.PolicyEffectDeny, 1, requireMFA)},
			req:      password,
			want:     iam.PolicyEffectDeny,
		},
		{
			name:     "a Deny requiring MFA stands down for an MFA session",
			policies: []*iam.AccessPolicy{policy(iam.PolicyEffectDeny, 1, requireMFA)},
			req:      withMFA,
			want:     "",
		},
		{
			name: "a satisfied Deny does not claim its priority from a lower policy",
			policies: []*iam.AccessPolicy{
				policy(iam.PolicyEffectDeny, 10, requireMFA),
				policy(iam.PolicyEffectDeny, 1, nil),
			},
			req:  withMFA,
			want: iam.PolicyEffectDeny,
		},
		{
			name: "the two-policy form still refuses a password session",
			policies: []*iam.AccessPolicy{
				policy(iam.PolicyEffectAllow, 10, requireMFA),
				policy(iam.PolicyEffectDeny, 1, nil),
			},
			req:  password,
			want: iam.PolicyEffectDeny,
		},
		{
			name: "the two-policy form still admits an MFA session",
			policies: []*iam.AccessPolicy{
				policy(iam.PolicyEffectAllow, 10, requireMFA),
				policy(iam.PolicyEffectDeny, 1, nil),
			},
			req:  withMFA,
			want: iam.PolicyEffectAllow,
		},
		{
			name: "mfaRequired false asks nothing of the session",
			policies: []*iam.AccessPolicy{
				policy(iam.PolicyEffectDeny, 1, map[string]string{"mfaRequired": "false"}),
			},
			req:  withMFA,
			want: iam.PolicyEffectDeny,
		},
		{
			name: "an assurance level below the minimum refuses",
			policies: []*iam.AccessPolicy{
				policy(iam.PolicyEffectAllow, 1, map[string]string{"authenticatorAalMin": "2"}),
			},
			req:  password,
			want: iam.PolicyEffectDeny,
		},
		{
			name: "an assurance level at the minimum admits",
			policies: []*iam.AccessPolicy{
				policy(iam.PolicyEffectAllow, 1, map[string]string{"authenticatorAalMin": "2"}),
			},
			req:  withMFA,
			want: iam.PolicyEffectAllow,
		},
		{
			name: "a federation level below the minimum refuses",
			policies: []*iam.AccessPolicy{
				policy(iam.PolicyEffectAllow, 1, map[string]string{"federationFalMin": "2"}),
			},
			req:  password,
			want: iam.PolicyEffectDeny,
		},
		{
			name: "a federation level at the minimum admits",
			policies: []*iam.AccessPolicy{
				policy(iam.PolicyEffectAllow, 1, map[string]string{"federationFalMin": "2"}),
			},
			req:  federated,
			want: iam.PolicyEffectAllow,
		},
		{
			name: "an unreadable minimum refuses rather than admits",
			policies: []*iam.AccessPolicy{
				policy(iam.PolicyEffectAllow, 1, map[string]string{"authenticatorAalMin": "two"}),
			},
			req:  withMFA,
			want: iam.PolicyEffectDeny,
		},
		{
			name:     "a check with no session is not refused for lacking one",
			policies: []*iam.AccessPolicy{policy(iam.PolicyEffectAllow, 1, requireMFA)},
			req:      noSession,
			want:     iam.PolicyEffectAllow,
		},
		{
			name:     "a Deny requiring MFA leaves background work alone",
			policies: []*iam.AccessPolicy{policy(iam.PolicyEffectDeny, 1, requireMFA)},
			req:      noSession,
			want:     "",
		},
		{
			name:     "an API key has no session to hold to MFA",
			policies: []*iam.AccessPolicy{policy(iam.PolicyEffectDeny, 1, requireMFA)},
			req:      apiKey,
			want:     "",
		},
		{
			name: "a requirement binds only the principals the policy is scoped to",
			policies: []*iam.AccessPolicy{policy(iam.PolicyEffectAllow, 1, map[string]string{
				"mfaRequired":   "true",
				"principalType": string(permissionports.PrincipalTypeAPIKey),
			})},
			req:  password,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, evaluateAccessPolicies(tt.policies, tt.req))
		})
	}
}

func TestRequestInSession(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	ctx := authctx.WithSessionAttributes(t.Context(), authctx.SessionAttributes{
		UserID:             userID,
		AuthenticatorAAL:   2,
		MFAAuthenticatedAt: 1_700_000_000,
	})

	tests := []struct {
		name string
		req  *permissionports.PermissionCheckRequest
		want int
	}{
		{
			name: "a service check for the session's person takes its assurance",
			req: &permissionports.PermissionCheckRequest{
				PrincipalType: permissionports.PrincipalTypeUser,
				UserID:        userID,
			},
			want: 2,
		},
		{
			name: "a check that names no principal type is the same person",
			req:  &permissionports.PermissionCheckRequest{UserID: userID},
			want: 2,
		},
		{
			name: "a check about someone else never borrows the session",
			req: &permissionports.PermissionCheckRequest{
				PrincipalType: permissionports.PrincipalTypeUser,
				UserID:        pulid.MustNew("usr_"),
			},
			want: 0,
		},
		{
			name: "an API key never borrows the session",
			req: &permissionports.PermissionCheckRequest{
				PrincipalType: permissionports.PrincipalTypeAPIKey,
				UserID:        userID,
			},
			want: 0,
		},
		{
			name: "attributes the caller set are kept",
			req: &permissionports.PermissionCheckRequest{
				PrincipalType: permissionports.PrincipalTypeUser,
				UserID:        userID,
				ContextAttributes: permissionports.RequestContextAttributes{
					AuthenticatorAAL: 1,
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := requestInSession(ctx, tt.req)

			assert.Equal(t, tt.want, got.ContextAttributes.AuthenticatorAAL)
		})
	}
}

func TestRequestInSession_DoesNotChangeTheCallersRequest(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	ctx := authctx.WithSessionAttributes(t.Context(), authctx.SessionAttributes{
		UserID:           userID,
		AuthenticatorAAL: 2,
	})
	req := &permissionports.PermissionCheckRequest{UserID: userID}

	requestInSession(ctx, req)

	assert.Zero(t, req.ContextAttributes.AuthenticatorAAL)
}

func TestEvaluateAccessPolicies_AnUnprovenTurnIsRefusedAnMFARequirement(t *testing.T) {
	t.Parallel()

	req := sessionAt(authctx.UnprovenAssurance, 0, 0)

	assert.Equal(t, iam.PolicyEffectDeny, evaluateAccessPolicies(
		[]*iam.AccessPolicy{
			policy(iam.PolicyEffectAllow, 1, map[string]string{"mfaRequired": "true"}),
		},
		req,
	))
}
