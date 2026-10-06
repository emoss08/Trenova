package cloud_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/rlslint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var systemScopeAllowed = map[string]string{
	"internal/cloud/aitraining/aitrainingrepository/exports.go:exportRepository.ListConsentingOrganizations":                "list organizations that consented to training export, across tenants",
	"internal/cloud/aitraining/aitrainingrepository/records.go:recordRepository.ListWithdrawn":                              "find exported examples whose organization has since withdrawn consent",
	"internal/cloud/cloudplan/subscriptionrepository/repository.go:repository.CountByStatus":                                "count cloud subscriptions by status across every organization",
	"internal/cloud/cloudplan/subscriptionrepository/repository.go:repository.ListDue":                                      "list cloud subscriptions due a lifecycle transition across every organization",
	"internal/cloud/cloudplan/subscriptionrepository/repository.go:repository.ListExpired":                                  "list expired cloud subscriptions across every organization so the sweep can finish their purge",
	"internal/cloud/controlplane/tenantprovisioningrepository/tenant_provisioning.go:repository.UpsertProvisioningSnapshot": "provision business units and organizations from the control plane",
	"internal/cloud/lifecycle/tenantpurgerepository/repository.go:repository.DeleteTenant":                                  "delete an expired cloud organization, its business unit and its signup record, which no tenant scope may delete",
	"internal/cloud/lifecycle/tenantpurgerepository/repository.go:repository.ListMembers":                                   "list an expired cloud organization's members and their memberships elsewhere, which span tenants",
	"internal/cloud/lifecycle/tenantpurgerepository/repository.go:repository.OrganizationProfile":                           "read a cloud organization's name and timezone for its lifecycle email from the sweep, which holds no tenant scope",
	"internal/cloud/lifecycle/tenantpurgerepository/repository.go:repository.PurgeRows":                                     "delete every row an expired cloud organization owns, discovered from the catalog across all tenant tables",
	"internal/cloud/lifecycle/tenantpurgerepository/repository.go:repository.PurgeUser":                                     "remove or deactivate a purged organization's user, whose row and other memberships span tenants",
	"internal/cloud/networkpulse/networkpulserepository/networkpulse.go:repository.GetNetworkPulse":                         "compute the instance-wide network pulse across every organization",
	"internal/cloud/seedaccounts/seedaccountrepository/repository.go:repository.CountReferences":                            "count rows of every tenant that reference a legacy seeded account, discovered from the catalog, to decide whether it can be deleted",
	"internal/cloud/seedaccounts/seedaccountrepository/repository.go:repository.FindSystemUser":                             "find the instance system user to attribute the retirement to, from an operator command that holds no tenant scope",
	"internal/cloud/seedaccounts/seedaccountrepository/repository.go:repository.FindUsers":                                  "find the legacy seeded accounts by username and email, which belong to whichever tenants the seed created",
	"internal/cloud/seedaccounts/seedaccountrepository/repository.go:repository.InTransaction":                              "hold the retirement of every legacy seeded account and demo organization in one transaction across the tenants they span",
	"internal/cloud/seedaccounts/seedaccountrepository/repository.go:repository.InspectOrganizations":                       "inspect the seed-created demo organizations and every catalog table that references them before deciding to delete them",
	"internal/cloud/seedaccounts/seedaccountrepository/repository.go:repository.RemoveOrganization":                         "delete a seed-created demo organization that holds nothing but seed defaults, which no tenant scope may delete",
	"internal/cloud/seedaccounts/seedaccountrepository/repository.go:repository.RemoveUser":                                 "delete or disable a legacy seeded account, whose row and references span tenants",
	"internal/cloud/seedaccounts/seedaccountrepository/repository.go:repository.StripUser":                                  "remove a legacy seeded account's memberships, roles, factors, reset links and API keys in every organization it joined",
	"internal/cloud/signup/cloudsignuprepository/repository.go:repository.CountProvisionedSince":                            "count cloud signups provisioned since a time for the daily signup cap",
	"internal/cloud/signup/cloudsignuprepository/repository.go:repository.Create":                                           "record a cloud signup request, which belongs to no organization yet",
	"internal/cloud/signup/cloudsignuprepository/repository.go:repository.Expire":                                           "expire unverified cloud signups past their token lifetime",
	"internal/cloud/signup/cloudsignuprepository/repository.go:repository.GetPendingByEmail":                                "find the pending cloud signup for an email address before any organization exists",
	"internal/cloud/signup/cloudsignuprepository/repository.go:repository.GetPendingByTokenHash":                            "lock a cloud signup by its verification token before any organization exists",
	"internal/cloud/signup/cloudsignuprepository/repository.go:repository.IncrementAttempts":                                "count verification attempts against a pending cloud signup",
	"internal/cloud/signup/cloudsignuprepository/repository.go:repository.MarkProvisioned":                                  "record the organization a verified cloud signup provisioned",
	"internal/cloud/signup/cloudsignuprepository/repository.go:repository.Refresh":                                          "replace the token on a pending cloud signup before any organization exists",
	"internal/cloud/signup/cloudsignuprepository/repository.go:repository.Reject":                                           "reject a pending cloud signup",
	"internal/cloud/signup/cloudsignuprepository/repository.go:repository.Touch":                                            "reissue the verification token of a pending cloud signup",
	"internal/cloud/signup/cloudsignupservice/verify.go:Service.provision":                                                  "provision a verified cloud signup's business unit, organization, owner, subscription and onboarding in one transaction before the tenant exists",
}

func serviceRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func TestEveryCloudSystemScopeIsReviewed(t *testing.T) {
	t.Parallel()

	sites, err := rlslint.Scan(serviceRoot(t), filepath.Join("internal", "cloud"))
	require.NoError(t, err)

	seen := make(map[string]struct{}, len(sites))
	unreviewed := make([]string, 0)
	unexplained := make([]string, 0)
	for _, site := range sites {
		seen[site.Key] = struct{}{}
		if _, ok := systemScopeAllowed[site.Key]; !ok {
			unreviewed = append(unreviewed, site.Position+" ("+site.Key+")")
		}
		if site.Reason == "" {
			unexplained = append(unexplained, site.Position)
		}
	}

	stale := make([]string, 0)
	for key, why := range systemScopeAllowed {
		if strings.TrimSpace(why) == "" {
			unexplained = append(unexplained, key+" (allowlist entry)")
		}
		if _, ok := seen[key]; !ok {
			stale = append(stale, key)
		}
	}

	assert.Empty(t, unreviewed,
		"dbscope.WithSystem bypasses tenant isolation; add each new cloud caller to "+
			"systemScopeAllowed with why it must see every tenant:\n%s", strings.Join(unreviewed, "\n"))
	assert.Empty(t, unexplained,
		"dbscope.WithSystem needs a non-empty string literal or package constant as its reason:\n%s",
		strings.Join(unexplained, "\n"))
	assert.Empty(t, stale, "allowlist entries with no remaining caller:\n%s", strings.Join(stale, "\n"))
}

func TestCloudRepositoryMethodsRunInScopedTransactions(t *testing.T) {
	t.Parallel()

	root := serviceRoot(t)
	violations, err := rlslint.ScanRepositories(filepath.Join(root, "internal", "cloud"), root)
	require.NoError(t, err)

	found := make([]string, 0, len(violations))
	for _, v := range violations {
		found = append(found, v.Position+" "+v.Key+": "+v.Problem)
	}
	assert.Empty(t, found,
		"repository methods must run their statements in a tenant-scoped transaction "+
			"(see docs/engineering/row-level-security.md):\n%s", strings.Join(found, "\n"))
}
