package iam

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(value int64) *int64 {
	return &value
}

func findingByCode(findings []SecurityFinding, code SecurityFindingCode) (SecurityFinding, bool) {
	for _, finding := range findings {
		if finding.Code == code {
			return finding, true
		}
	}
	return SecurityFinding{}, false
}

func briefingWithIdleTimeout() *SecurityBriefing {
	b := NewSecurityBriefing(1_000_000)
	b.Session = SecuritySessionPolicy{
		MaxAgeSeconds:      43_200,
		IdleTimeoutSeconds: 1_800,
		IdleWarningSeconds: 120,
	}
	return b
}

func TestBuildSecurityFindings_SkipsSectionsTheCallerCannotSee(t *testing.T) {
	t.Parallel()

	findings := BuildSecurityFindings(briefingWithIdleTimeout())

	require.Len(t, findings, 1)
	assert.Equal(t, SecurityFinding{
		Code:     SecurityFindingIdleTimeout,
		Severity: SecurityFindingPass,
	}, findings[0])
}

func TestBuildSecurityFindings_NilBriefingHasNoFindings(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []SecurityFinding{}, BuildSecurityFindings(nil))
}

func TestBuildSecurityFindings_MFACoverage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		mfa      *SecurityMFACoverage
		sso      *SecuritySSOSummary
		want     SecurityFinding
		expected bool
	}{
		{
			name:     "no members has no finding",
			mfa:      &SecurityMFACoverage{Enrolled: 0, Total: 0},
			expected: false,
		},
		{
			name: "every member enrolled passes",
			mfa:  &SecurityMFACoverage{Enrolled: 4, Total: 4},
			want: SecurityFinding{
				Code:     SecurityFindingMFACoverage,
				Severity: SecurityFindingPass,
				Count:    0,
				Total:    4,
			},
			expected: true,
		},
		{
			name: "exactly half unenrolled is critical",
			mfa:  &SecurityMFACoverage{Enrolled: 5, Total: 10},
			want: SecurityFinding{
				Code:     SecurityFindingMFACoverage,
				Severity: SecurityFindingCritical,
				Count:    5,
				Total:    10,
			},
			expected: true,
		},
		{
			name: "just under half unenrolled is a warning",
			mfa:  &SecurityMFACoverage{Enrolled: 6, Total: 10},
			want: SecurityFinding{
				Code:     SecurityFindingMFACoverage,
				Severity: SecurityFindingWarning,
				Count:    4,
				Total:    10,
			},
			expected: true,
		},
		{
			name: "enforced single sign-on lowers missing enrolment to a notice",
			mfa:  &SecurityMFACoverage{Enrolled: 0, Total: 10},
			sso:  &SecuritySSOSummary{EnabledProviders: 1, Enforced: true},
			want: SecurityFinding{
				Code:     SecurityFindingMFACoverage,
				Severity: SecurityFindingNotice,
				Count:    10,
				Total:    10,
			},
			expected: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := briefingWithIdleTimeout()
			b.MFA = tc.mfa
			b.SSO = tc.sso

			got, ok := findingByCode(BuildSecurityFindings(b), SecurityFindingMFACoverage)
			require.Equal(t, tc.expected, ok)
			if tc.expected {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestBuildSecurityFindings_FailedSignIns(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		signIns SecuritySignInSummary
		want    SecurityFinding
	}{
		{
			name:    "no failures passes",
			signIns: SecuritySignInSummary{Succeeded: 20},
			want: SecurityFinding{
				Code:     SecurityFindingFailedSignIns,
				Severity: SecurityFindingPass,
				Total:    20,
			},
		},
		{
			name:    "nine failures is a notice",
			signIns: SecuritySignInSummary{Succeeded: 20, Failed: 5, Denied: 3, Throttled: 1},
			want: SecurityFinding{
				Code:     SecurityFindingFailedSignIns,
				Severity: SecurityFindingNotice,
				Count:    9,
				Total:    29,
			},
		},
		{
			name:    "ten failures is a warning",
			signIns: SecuritySignInSummary{Succeeded: 20, Failed: 6, Denied: 3, Throttled: 1},
			want: SecurityFinding{
				Code:     SecurityFindingFailedSignIns,
				Severity: SecurityFindingWarning,
				Count:    10,
				Total:    30,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := briefingWithIdleTimeout()
			b.SignIns = &tc.signIns

			got, ok := findingByCode(BuildSecurityFindings(b), SecurityFindingFailedSignIns)
			require.True(t, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildSecurityFindings_RiskDenials(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		riskDenied *int64
		expected   bool
	}{
		{name: "risk decisions not visible", riskDenied: nil, expected: false},
		{name: "no denials", riskDenied: int64Ptr(0), expected: false},
		{name: "denials", riskDenied: int64Ptr(2), expected: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := briefingWithIdleTimeout()
			b.SignIns = &SecuritySignInSummary{RiskDenied: tc.riskDenied}

			got, ok := findingByCode(BuildSecurityFindings(b), SecurityFindingRiskDenials)
			require.Equal(t, tc.expected, ok)
			if tc.expected {
				assert.Equal(t, SecurityFinding{
					Code:     SecurityFindingRiskDenials,
					Severity: SecurityFindingWarning,
					Count:    2,
				}, got)
			}
		})
	}
}

func TestBuildSecurityFindings_SingleSignOnAndDirectory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		sso          *SecuritySSOSummary
		provisioning *SecurityProvisioningSummary
		wantSSO      SecurityFindingSeverity
		wantSCIM     SecurityFindingSeverity
	}{
		{
			name:         "no provider is a notice and directory is not judged",
			sso:          &SecuritySSOSummary{},
			provisioning: &SecurityProvisioningSummary{},
			wantSSO:      SecurityFindingNotice,
		},
		{
			name:         "provider not enforced and no directory",
			sso:          &SecuritySSOSummary{EnabledProviders: 1},
			provisioning: &SecurityProvisioningSummary{},
			wantSSO:      SecurityFindingNotice,
			wantSCIM:     SecurityFindingNotice,
		},
		{
			name:         "enforced provider with a directory",
			sso:          &SecuritySSOSummary{EnabledProviders: 1, Enforced: true},
			provisioning: &SecurityProvisioningSummary{ActiveDirectories: 1},
			wantSSO:      SecurityFindingPass,
			wantSCIM:     SecurityFindingPass,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := briefingWithIdleTimeout()
			b.SSO = tc.sso
			b.Provisioning = tc.provisioning
			findings := BuildSecurityFindings(b)

			sso, ok := findingByCode(findings, SecurityFindingSSO)
			require.True(t, ok)
			assert.Equal(t, tc.wantSSO, sso.Severity)

			scim, ok := findingByCode(findings, SecurityFindingSCIM)
			require.Equal(t, tc.wantSCIM != "", ok)
			if ok {
				assert.Equal(t, tc.wantSCIM, scim.Severity)
			}
		})
	}
}

func TestBuildSecurityFindings_MembersAndKeys(t *testing.T) {
	t.Parallel()

	b := briefingWithIdleTimeout()
	b.Members = &SecurityMemberSummary{Active: 30, Locked: 2, Dormant: 3}
	b.APIKeys = &SecurityAPIKeySummary{Active: 4, WithoutExpiry: 1, Unused: 0, ExpiringSoon: 2}
	findings := BuildSecurityFindings(b)

	cases := []SecurityFinding{
		{Code: SecurityFindingDormant, Severity: SecurityFindingWarning, Count: 3, Total: 30},
		{Code: SecurityFindingLocked, Severity: SecurityFindingNotice, Count: 2, Total: 30},
		{Code: SecurityFindingAPIKeyExpiry, Severity: SecurityFindingWarning, Count: 1, Total: 4},
		{Code: SecurityFindingAPIKeyUnused, Severity: SecurityFindingPass, Count: 0, Total: 4},
		{Code: SecurityFindingAPIKeyExpiring, Severity: SecurityFindingNotice, Count: 2, Total: 4},
	}
	for _, want := range cases {
		got, ok := findingByCode(findings, want.Code)
		require.True(t, ok, want.Code)
		assert.Equal(t, want, got)
	}
}

func TestBuildSecurityFindings_QuietMembersAndNoKeysOmitOptionalChecks(t *testing.T) {
	t.Parallel()

	b := briefingWithIdleTimeout()
	b.Members = &SecurityMemberSummary{Active: 30}
	b.APIKeys = &SecurityAPIKeySummary{}
	findings := BuildSecurityFindings(b)

	for _, code := range []SecurityFindingCode{
		SecurityFindingLocked,
		SecurityFindingAPIKeyExpiry,
		SecurityFindingAPIKeyUnused,
		SecurityFindingAPIKeyExpiring,
	} {
		_, ok := findingByCode(findings, code)
		assert.False(t, ok, code)
	}

	dormant, ok := findingByCode(findings, SecurityFindingDormant)
	require.True(t, ok)
	assert.Equal(t, SecurityFindingPass, dormant.Severity)
}

func TestBuildSecurityFindings_PoliciesAndSession(t *testing.T) {
	t.Parallel()

	b := NewSecurityBriefing(1_000_000)
	b.Policies = &SecurityPolicySummary{Enabled: 3}
	findings := BuildSecurityFindings(b)

	assert.Equal(t, []SecurityFinding{
		{Code: SecurityFindingIdleTimeout, Severity: SecurityFindingWarning},
		{Code: SecurityFindingMFAPolicy, Severity: SecurityFindingNotice, Count: 0, Total: 3},
		{Code: SecurityFindingRoleActivation, Severity: SecurityFindingNotice},
	}, findings)

	b.Policies = &SecurityPolicySummary{Enabled: 3, RequiringMFA: 1, RoleActivationRequired: true}
	b.Session.IdleTimeoutSeconds = 900
	assert.Equal(t, []SecurityFinding{
		{Code: SecurityFindingMFAPolicy, Severity: SecurityFindingPass, Count: 1, Total: 3},
		{Code: SecurityFindingRoleActivation, Severity: SecurityFindingPass},
		{Code: SecurityFindingIdleTimeout, Severity: SecurityFindingPass},
	}, BuildSecurityFindings(b))
}

func TestBuildSecurityFindings_OrdersBySeverityThenCheck(t *testing.T) {
	t.Parallel()

	b := briefingWithIdleTimeout()
	b.MFA = &SecurityMFACoverage{Enrolled: 1, Total: 10}
	b.Members = &SecurityMemberSummary{Active: 10, Dormant: 2, Locked: 1}
	b.SSO = &SecuritySSOSummary{}
	b.APIKeys = &SecurityAPIKeySummary{Active: 1, WithoutExpiry: 1, Unused: 1}

	got := make([]SecurityFindingCode, 0)
	for _, finding := range BuildSecurityFindings(b) {
		got = append(got, finding.Code)
	}

	assert.Equal(t, []SecurityFindingCode{
		SecurityFindingMFACoverage,
		SecurityFindingDormant,
		SecurityFindingAPIKeyExpiry,
		SecurityFindingAPIKeyUnused,
		SecurityFindingSSO,
		SecurityFindingLocked,
		SecurityFindingIdleTimeout,
	}, got)
}

func TestSecurityBriefing_Cutoffs(t *testing.T) {
	t.Parallel()

	b := NewSecurityBriefing(100 * secondsPerDay)

	assert.Equal(t, 93*secondsPerDay, b.SignInSince())
	assert.Equal(t, 10*secondsPerDay, b.DormantBefore())
	assert.Equal(t, 10*secondsPerDay, b.APIKeyUnusedBefore())
	assert.Equal(t, 114*secondsPerDay, b.APIKeyExpiringBefore())
}
