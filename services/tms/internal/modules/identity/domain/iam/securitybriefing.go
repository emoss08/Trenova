package iam

import (
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
)

type SecurityFindingSeverity string

const (
	SecurityFindingCritical SecurityFindingSeverity = "critical"
	SecurityFindingWarning  SecurityFindingSeverity = "warning"
	SecurityFindingNotice   SecurityFindingSeverity = "notice"
	SecurityFindingPass     SecurityFindingSeverity = "pass"
)

type SecurityFindingCode string

const (
	SecurityFindingMFACoverage    SecurityFindingCode = "mfa_coverage"
	SecurityFindingMFAPolicy      SecurityFindingCode = "mfa_policy"
	SecurityFindingSSO            SecurityFindingCode = "sso"
	SecurityFindingSCIM           SecurityFindingCode = "scim"
	SecurityFindingDormant        SecurityFindingCode = "dormant_accounts"
	SecurityFindingLocked         SecurityFindingCode = "locked_accounts"
	SecurityFindingFailedSignIns  SecurityFindingCode = "failed_sign_ins"
	SecurityFindingRiskDenials    SecurityFindingCode = "risk_denials"
	SecurityFindingAPIKeyExpiry   SecurityFindingCode = "api_key_expiry"
	SecurityFindingAPIKeyUnused   SecurityFindingCode = "api_key_unused"
	SecurityFindingAPIKeyExpiring SecurityFindingCode = "api_key_expiring"
	SecurityFindingRoleActivation SecurityFindingCode = "role_activation"
	SecurityFindingIdleTimeout    SecurityFindingCode = "idle_timeout"
)

const (
	SecuritySignInWindowDays         int64 = 7
	SecurityDormantAfterDays         int64 = 90
	SecurityAPIKeyUnusedAfterDays    int64 = 90
	SecurityAPIKeyExpiringWithinDays int64 = 14
	SecurityFailedSignInWarningAt    int64 = 10
	SecurityUnenrolledListLimit            = 25

	SecurityMFACriticalUnenrolledNumerator   int64 = 1
	SecurityMFACriticalUnenrolledDenominator int64 = 2

	secondsPerDay int64 = 24 * 60 * 60
)

var securityFindingSeverityRank = map[SecurityFindingSeverity]int{
	SecurityFindingCritical: 0,
	SecurityFindingWarning:  1,
	SecurityFindingNotice:   2,
	SecurityFindingPass:     3,
}

var securityFindingCodeRank = map[SecurityFindingCode]int{
	SecurityFindingMFACoverage:    0,
	SecurityFindingMFAPolicy:      1,
	SecurityFindingSSO:            2,
	SecurityFindingSCIM:           3,
	SecurityFindingDormant:        4,
	SecurityFindingLocked:         5,
	SecurityFindingFailedSignIns:  6,
	SecurityFindingRiskDenials:    7,
	SecurityFindingAPIKeyExpiry:   8,
	SecurityFindingAPIKeyUnused:   9,
	SecurityFindingAPIKeyExpiring: 10,
	SecurityFindingRoleActivation: 11,
	SecurityFindingIdleTimeout:    12,
}

type SecurityFinding struct {
	Code     SecurityFindingCode     `json:"code"`
	Severity SecurityFindingSeverity `json:"severity"`
	Count    int64                   `json:"count"`
	Total    int64                   `json:"total"`
}

type SecuritySessionPolicy struct {
	MaxAgeSeconds      int64 `json:"maxAgeSeconds"`
	IdleTimeoutSeconds int64 `json:"idleTimeoutSeconds"`
	IdleWarningSeconds int64 `json:"idleWarningSeconds"`
}

type SecurityMemberSummary struct {
	Active             int64 `json:"active"             bun:"active"`
	Locked             int64 `json:"locked"             bun:"locked"`
	NeverSignedIn      int64 `json:"neverSignedIn"      bun:"never_signed_in"`
	Dormant            int64 `json:"dormant"            bun:"dormant"`
	MustChangePassword int64 `json:"mustChangePassword" bun:"must_change_password"`
}

type SecurityMemberRef struct {
	UserID       pulid.ID `json:"userId"       bun:"user_id"`
	Name         string   `json:"name"         bun:"name"`
	EmailAddress string   `json:"emailAddress" bun:"email_address"`
	LastLoginAt  *int64   `json:"lastLoginAt"  bun:"last_login_at"`
}

type SecurityMFACoverage struct {
	Enrolled   int64                `json:"enrolled"`
	Total      int64                `json:"total"`
	Unenrolled []*SecurityMemberRef `json:"unenrolled"`
}

type SecuritySignInSummary struct {
	Succeeded  int64  `json:"succeeded"`
	Failed     int64  `json:"failed"`
	Denied     int64  `json:"denied"`
	Throttled  int64  `json:"throttled"`
	RiskDenied *int64 `json:"riskDenied"`
}

type SecurityAPIKeySummary struct {
	Active        int64 `json:"active"        bun:"active"`
	WithoutExpiry int64 `json:"withoutExpiry" bun:"without_expiry"`
	Unused        int64 `json:"unused"        bun:"unused"`
	ExpiringSoon  int64 `json:"expiringSoon"  bun:"expiring_soon"`
}

type SecuritySSOSummary struct {
	EnabledProviders     int64  `json:"enabledProviders"`
	Enforced             bool   `json:"enforced"`
	EnforcedProviderName string `json:"enforcedProviderName"`
	TrustsFederatedMFA   bool   `json:"trustsFederatedMfa"`
}

type SecurityProvisioningSummary struct {
	ActiveDirectories int64 `json:"activeDirectories"`
}

type SecurityPolicySummary struct {
	Enabled                int64 `json:"enabled"`
	RequiringMFA           int64 `json:"requiringMfa"`
	RoleActivationRequired bool  `json:"roleActivationRequired"`
}

type SecurityBriefing struct {
	GeneratedAt              int64                        `json:"generatedAt"`
	SignInWindowDays         int64                        `json:"signInWindowDays"`
	DormantAfterDays         int64                        `json:"dormantAfterDays"`
	APIKeyUnusedAfterDays    int64                        `json:"apiKeyUnusedAfterDays"`
	APIKeyExpiringWithinDays int64                        `json:"apiKeyExpiringWithinDays"`
	Findings                 []SecurityFinding            `json:"findings"`
	Session                  SecuritySessionPolicy        `json:"session"`
	Members                  *SecurityMemberSummary       `json:"members"`
	MFA                      *SecurityMFACoverage         `json:"mfa"`
	SignIns                  *SecuritySignInSummary       `json:"signIns"`
	APIKeys                  *SecurityAPIKeySummary       `json:"apiKeys"`
	SSO                      *SecuritySSOSummary          `json:"sso"`
	Provisioning             *SecurityProvisioningSummary `json:"provisioning"`
	Policies                 *SecurityPolicySummary       `json:"policies"`
}

func NewSecurityBriefing(now int64) *SecurityBriefing {
	return &SecurityBriefing{
		GeneratedAt:              now,
		SignInWindowDays:         SecuritySignInWindowDays,
		DormantAfterDays:         SecurityDormantAfterDays,
		APIKeyUnusedAfterDays:    SecurityAPIKeyUnusedAfterDays,
		APIKeyExpiringWithinDays: SecurityAPIKeyExpiringWithinDays,
		Findings:                 []SecurityFinding{},
	}
}

func (b *SecurityBriefing) SignInSince() int64 {
	return b.GeneratedAt - b.SignInWindowDays*secondsPerDay
}

func (b *SecurityBriefing) DormantBefore() int64 {
	return b.GeneratedAt - b.DormantAfterDays*secondsPerDay
}

func (b *SecurityBriefing) APIKeyUnusedBefore() int64 {
	return b.GeneratedAt - b.APIKeyUnusedAfterDays*secondsPerDay
}

func (b *SecurityBriefing) APIKeyExpiringBefore() int64 {
	return b.GeneratedAt + b.APIKeyExpiringWithinDays*secondsPerDay
}

func BuildSecurityFindings(b *SecurityBriefing) []SecurityFinding {
	if b == nil {
		return []SecurityFinding{}
	}

	findings := make([]SecurityFinding, 0, len(securityFindingCodeRank))
	add := func(finding *SecurityFinding) {
		if finding != nil {
			findings = append(findings, *finding)
		}
	}

	add(mfaCoverageFinding(b.MFA, b.SSO))
	add(mfaPolicyFinding(b.Policies))
	add(ssoFinding(b.SSO))
	add(scimFinding(b.Provisioning, b.SSO))
	add(dormantFinding(b.Members))
	add(lockedFinding(b.Members))
	add(failedSignInFinding(b.SignIns))
	add(riskDenialFinding(b.SignIns))
	add(apiKeyExpiryFinding(b.APIKeys))
	add(apiKeyUnusedFinding(b.APIKeys))
	add(apiKeyExpiringFinding(b.APIKeys))
	add(roleActivationFinding(b.Policies))
	add(idleTimeoutFinding(b.Session))

	slices.SortStableFunc(findings, compareSecurityFindings)

	return findings
}

func compareSecurityFindings(left, right SecurityFinding) int {
	severity := securityFindingSeverityRank[left.Severity] -
		securityFindingSeverityRank[right.Severity]
	if severity != 0 {
		return severity
	}
	return securityFindingCodeRank[left.Code] - securityFindingCodeRank[right.Code]
}

func passOr(failing bool, severity SecurityFindingSeverity) SecurityFindingSeverity {
	if failing {
		return severity
	}
	return SecurityFindingPass
}

func mfaCoverageFinding(mfa *SecurityMFACoverage, sso *SecuritySSOSummary) *SecurityFinding {
	if mfa == nil || mfa.Total == 0 {
		return nil
	}

	unenrolled := max(mfa.Total-mfa.Enrolled, 0)
	finding := &SecurityFinding{
		Code:  SecurityFindingMFACoverage,
		Count: unenrolled,
		Total: mfa.Total,
	}

	switch {
	case unenrolled == 0:
		finding.Severity = SecurityFindingPass
	case sso != nil && sso.Enforced:
		finding.Severity = SecurityFindingNotice
	case unenrolled*SecurityMFACriticalUnenrolledDenominator >=
		mfa.Total*SecurityMFACriticalUnenrolledNumerator:
		finding.Severity = SecurityFindingCritical
	default:
		finding.Severity = SecurityFindingWarning
	}

	return finding
}

func mfaPolicyFinding(policies *SecurityPolicySummary) *SecurityFinding {
	if policies == nil {
		return nil
	}

	return &SecurityFinding{
		Code:     SecurityFindingMFAPolicy,
		Severity: passOr(policies.RequiringMFA == 0, SecurityFindingNotice),
		Count:    policies.RequiringMFA,
		Total:    policies.Enabled,
	}
}

func ssoFinding(sso *SecuritySSOSummary) *SecurityFinding {
	if sso == nil {
		return nil
	}

	return &SecurityFinding{
		Code:     SecurityFindingSSO,
		Severity: passOr(sso.EnabledProviders == 0 || !sso.Enforced, SecurityFindingNotice),
		Count:    sso.EnabledProviders,
	}
}

func scimFinding(
	provisioning *SecurityProvisioningSummary,
	sso *SecuritySSOSummary,
) *SecurityFinding {
	if provisioning == nil || sso == nil || sso.EnabledProviders == 0 {
		return nil
	}

	return &SecurityFinding{
		Code:     SecurityFindingSCIM,
		Severity: passOr(provisioning.ActiveDirectories == 0, SecurityFindingNotice),
		Count:    provisioning.ActiveDirectories,
	}
}

func dormantFinding(members *SecurityMemberSummary) *SecurityFinding {
	if members == nil {
		return nil
	}

	return &SecurityFinding{
		Code:     SecurityFindingDormant,
		Severity: passOr(members.Dormant > 0, SecurityFindingWarning),
		Count:    members.Dormant,
		Total:    members.Active,
	}
}

func lockedFinding(members *SecurityMemberSummary) *SecurityFinding {
	if members == nil || members.Locked == 0 {
		return nil
	}

	return &SecurityFinding{
		Code:     SecurityFindingLocked,
		Severity: SecurityFindingNotice,
		Count:    members.Locked,
		Total:    members.Active,
	}
}

func failedSignInFinding(signIns *SecuritySignInSummary) *SecurityFinding {
	if signIns == nil {
		return nil
	}

	failed := signIns.Failed + signIns.Denied + signIns.Throttled
	finding := &SecurityFinding{
		Code:  SecurityFindingFailedSignIns,
		Count: failed,
		Total: failed + signIns.Succeeded,
	}

	switch {
	case failed >= SecurityFailedSignInWarningAt:
		finding.Severity = SecurityFindingWarning
	case failed > 0:
		finding.Severity = SecurityFindingNotice
	default:
		finding.Severity = SecurityFindingPass
	}

	return finding
}

func riskDenialFinding(signIns *SecuritySignInSummary) *SecurityFinding {
	if signIns == nil || signIns.RiskDenied == nil || *signIns.RiskDenied == 0 {
		return nil
	}

	return &SecurityFinding{
		Code:     SecurityFindingRiskDenials,
		Severity: SecurityFindingWarning,
		Count:    *signIns.RiskDenied,
	}
}

func apiKeyExpiryFinding(keys *SecurityAPIKeySummary) *SecurityFinding {
	if keys == nil || keys.Active == 0 {
		return nil
	}

	return &SecurityFinding{
		Code:     SecurityFindingAPIKeyExpiry,
		Severity: passOr(keys.WithoutExpiry > 0, SecurityFindingWarning),
		Count:    keys.WithoutExpiry,
		Total:    keys.Active,
	}
}

func apiKeyUnusedFinding(keys *SecurityAPIKeySummary) *SecurityFinding {
	if keys == nil || keys.Active == 0 {
		return nil
	}

	return &SecurityFinding{
		Code:     SecurityFindingAPIKeyUnused,
		Severity: passOr(keys.Unused > 0, SecurityFindingWarning),
		Count:    keys.Unused,
		Total:    keys.Active,
	}
}

func apiKeyExpiringFinding(keys *SecurityAPIKeySummary) *SecurityFinding {
	if keys == nil || keys.ExpiringSoon == 0 {
		return nil
	}

	return &SecurityFinding{
		Code:     SecurityFindingAPIKeyExpiring,
		Severity: SecurityFindingNotice,
		Count:    keys.ExpiringSoon,
		Total:    keys.Active,
	}
}

func roleActivationFinding(policies *SecurityPolicySummary) *SecurityFinding {
	if policies == nil {
		return nil
	}

	return &SecurityFinding{
		Code:     SecurityFindingRoleActivation,
		Severity: passOr(!policies.RoleActivationRequired, SecurityFindingNotice),
	}
}

func idleTimeoutFinding(session SecuritySessionPolicy) *SecurityFinding {
	return &SecurityFinding{
		Code:     SecurityFindingIdleTimeout,
		Severity: passOr(session.IdleTimeoutSeconds == 0, SecurityFindingWarning),
	}
}
