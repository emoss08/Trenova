package authzlint

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const resolverDir = "../resolver"

var authOnlyAllowlist = map[string]string{
	"QueryResolver.SelectOptions": "select options are deliberately readable by every " +
		"authenticated user in the tenant: forms on pages a user cannot open still need " +
		"their dropdowns, and the payload is id, label and display metadata only",

	"QueryResolver.Notifications":               "returns only the calling user's own notifications",
	"QueryResolver.NotificationUnreadCount":     "returns only the calling user's own notifications",
	"MutationResolver.MarkNotificationsRead":    "acts only on the calling user's own notifications",
	"MutationResolver.MarkNotificationsUnread":  "acts only on the calling user's own notifications",
	"MutationResolver.MarkAllNotificationsRead": "acts only on the calling user's own notifications",
	"MutationResolver.DismissNotifications":     "acts only on the calling user's own notifications",
	"MutationResolver.RestoreNotifications":     "acts only on the calling user's own notifications",

	"QueryResolver.SidebarPreferences":          "returns only the calling user's own preferences",
	"QueryResolver.SidebarCustomizationOptions": "returns only the calling user's own preferences",
	"MutationResolver.UpdateSidebarPreferences": "acts only on the calling user's own preferences",

	"QueryResolver.HomeLayout":          "returns only the calling user's own home layout",
	"QueryResolver.HomeWidgetCatalog":   "static catalog of widgets; carries no resource data",
	"MutationResolver.UpdateHomeLayout": "acts only on the calling user's own home layout",
	"MutationResolver.ResetHomeLayout":  "acts only on the calling user's own home layout",

	"QueryResolver.TableConfiguration":              "table layouts are UI preferences, not resource data",
	"QueryResolver.TableConfigurations":             "table layouts are UI preferences, not resource data",
	"QueryResolver.DefaultTableConfiguration":       "table layouts are UI preferences, not resource data",
	"MutationResolver.CreateTableConfiguration":     "table layouts are UI preferences, not resource data",
	"MutationResolver.UpdateTableConfiguration":     "table layouts are UI preferences, not resource data",
	"MutationResolver.PatchTableConfiguration":      "table layouts are UI preferences, not resource data",
	"MutationResolver.DeleteTableConfiguration":     "table layouts are UI preferences, not resource data",
	"MutationResolver.SetDefaultTableConfiguration": "table layouts are UI preferences, not resource data",

	"QueryResolver.CaptureAgentRelease": "public release metadata, served unauthenticated at " +
		"/api/v1/capture/releases/latest; carries no tenant data",

	"QueryResolver.TelematicsStatus": "org-wide integration health indicator shown in the " +
		"application shell; carries no resource data",

	"QueryResolver.MyAIFeedback": "returns only the caller's own ratings of AI output; " +
		"no one else's rating is ever read",
	"MutationResolver.SetMyAIFeedback": "rates AI output for the caller only; the service " +
		"refuses a target the caller could not read (their own thread or briefing, " +
		"insight:read, watchtower:read plus the item source's read)",
	"MutationResolver.ClearMyAIFeedback": "removes only the caller's own rating; the delete " +
		"is scoped to the caller's user id",

	"MutationResolver.CreateSettlementDispute":   "a driver disputing their own settlement",
	"MutationResolver.WithdrawSettlementDispute": "a driver withdrawing their own dispute",
}

var selfScopedName = regexp.MustCompile(`(^|[a-z])My[A-Z]`)

func isAllowedAuthOnly(root RootResolver) bool {
	if _, ok := authOnlyAllowlist[root.Key()]; ok {
		return true
	}

	return selfScopedName.MatchString(root.Name)
}

func TestEveryRootResolverIsAuthorized(t *testing.T) {
	t.Parallel()

	roots, err := Analyze(resolverDir)
	require.NoError(t, err)
	require.NotEmpty(t, roots, "no root resolvers found; is %s the resolver package?", resolverDir)

	seen := make(map[string]struct{}, len(roots))
	var unchecked, unlistedAuthOnly, overListed []string

	for _, root := range roots {
		seen[root.Key()] = struct{}{}

		switch root.Verdict {
		case VerdictNone:
			unchecked = append(unchecked, root.File+": "+root.Key())
		case VerdictAuthOnly:
			if !isAllowedAuthOnly(root) {
				unlistedAuthOnly = append(unlistedAuthOnly, root.File+": "+root.Key())
			}
		case VerdictPermission:
			if _, listed := authOnlyAllowlist[root.Key()]; listed {
				overListed = append(overListed, root.Key())
			}
		}
	}

	assert.Empty(t, unchecked,
		"root resolvers with no authentication or permission check at all:\n  %s",
		strings.Join(unchecked, "\n  "))

	assert.Empty(t, unlistedAuthOnly,
		"root resolvers that authenticate but never reach a permission check. "+
			"Either add one, name the resolver My<Thing> if it only ever touches the "+
			"caller's own data, or add it to authOnlyAllowlist with a reason explaining "+
			"why every authenticated user may call it:\n  %s",
		strings.Join(unlistedAuthOnly, "\n  "))

	assert.Empty(t, overListed,
		"allowlisted resolvers that now perform a permission check; remove them from "+
			"authOnlyAllowlist so the list stays accurate:\n  %s",
		strings.Join(overListed, "\n  "))

	var stale []string
	for key := range authOnlyAllowlist {
		if _, ok := seen[key]; !ok {
			stale = append(stale, key)
		}
	}
	assert.Empty(t, stale, "authOnlyAllowlist names resolvers that no longer exist:\n  %s",
		strings.Join(stale, "\n  "))
}

func TestAnalyzeReportsCoverage(t *testing.T) {
	t.Parallel()

	roots, err := Analyze(resolverDir)
	require.NoError(t, err)

	counts := map[Verdict]int{}
	for _, root := range roots {
		counts[root.Verdict]++
	}

	t.Logf("root resolvers: %d  permission=%d  auth-only=%d  none=%d",
		len(roots), counts[VerdictPermission], counts[VerdictAuthOnly], counts[VerdictNone])

	assert.Zero(t, counts[VerdictNone])
	assert.Greater(t, counts[VerdictPermission], counts[VerdictAuthOnly],
		"the overwhelming majority of root resolvers must be permission-checked")
}

func TestSelfScopedNameRule(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"MyLoads", "CancelMyExpense", "MarkAllMyNotificationsRead"} {
		assert.True(t, selfScopedName.MatchString(name), name)
	}
	for _, name := range []string{"Myths", "Shipments", "DummyData", "ApproveWorkerPto"} {
		assert.False(t, selfScopedName.MatchString(name), name)
	}
}

// What agents may do without a person names every tool's rule and every
// agent's reach, so both reads are held to reading agents.
func TestAgentSafetyResolversAreAuthorized(t *testing.T) {
	t.Parallel()

	roots, err := Analyze(resolverDir)
	require.NoError(t, err)

	verdicts := make(map[string]Verdict, len(roots))
	for _, root := range roots {
		verdicts[root.Key()] = root.Verdict
	}

	for _, key := range []string{
		"QueryResolver.AgentToolPolicies",
		"QueryResolver.AgentToolPolicyConnection",
		"QueryResolver.AgentSafetySummary",
		"QueryResolver.AgentSafety",
	} {
		verdict, ok := verdicts[key]
		require.True(t, ok, "%s is not a root resolver", key)
		assert.Equal(t, VerdictPermission, verdict, "%s must reach a permission check", key)
		_, listed := authOnlyAllowlist[key]
		assert.False(t, listed, "%s must not be allowlisted as auth-only", key)
	}
}

// Who may use which agent is decided by these operations. The self-scoped
// ones are named My<Thing>, which would let them pass on authentication
// alone, so each is held to a permission check of its own.
func TestAgentAccessResolversAreAuthorized(t *testing.T) {
	t.Parallel()

	roots, err := Analyze(resolverDir)
	require.NoError(t, err)

	verdicts := make(map[string]Verdict, len(roots))
	for _, root := range roots {
		verdicts[root.Key()] = root.Verdict
	}

	for _, key := range []string{
		"QueryResolver.MyAgents",
		"QueryResolver.SuggestedAgentAudience",
		"MutationResolver.SetAgentAccess",
		"MutationResolver.SetRoleAgentAccess",
		"MutationResolver.DecideMyProposal",
		// Self-scoped by name, so it would pass on authentication alone. It
		// runs the plan's writes, so it must hold assistant:update before the
		// service checks the thread is the caller's and the agent is theirs.
		"MutationResolver.DecideMyPlan",
		"QueryResolver.AgentAccessPreview",
		"QueryResolver.PendingDecisions",
		"QueryResolver.PendingDecisionSummary",
	} {
		verdict, ok := verdicts[key]
		require.True(t, ok, "%s is not a root resolver", key)
		assert.Equal(t, VerdictPermission, verdict, "%s must reach a permission check", key)
		_, listed := authOnlyAllowlist[key]
		assert.False(t, listed, "%s must not be allowlisted as auth-only", key)
	}
}

func TestAgentQualityResolversAreAuthorized(t *testing.T) {
	t.Parallel()

	roots, err := Analyze(resolverDir)
	require.NoError(t, err)

	verdicts := make(map[string]Verdict, len(roots))
	for _, root := range roots {
		verdicts[root.Key()] = root.Verdict
	}

	for _, key := range []string{
		"QueryResolver.AgentQualityOverview",
		"QueryResolver.AgentQuality",
		"QueryResolver.AgentQualityAgents",
		"QueryResolver.AgentWorstRatedAnswers",
		"QueryResolver.AgentSuiteRuns",
		"QueryResolver.AgentSuiteRun",
		"QueryResolver.AgentSuiteRunCases",
		"QueryResolver.AgentQualityControl",
		"MutationResolver.RunAgentSuite",
		"MutationResolver.UpdateAgentQualityControl",
	} {
		verdict, ok := verdicts[key]
		require.True(t, ok, "%s is not a root resolver", key)
		assert.Equal(t, VerdictPermission, verdict, "%s must reach a permission check", key)
		_, listed := authOnlyAllowlist[key]
		assert.False(t, listed, "%s must not be allowlisted as auth-only", key)
	}
}
