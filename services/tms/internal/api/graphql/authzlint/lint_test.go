package authzlint

import (
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
	"QueryResolver.MyTableLayout":               "returns only the calling user's own table layout",
	"MutationResolver.SaveMyTableLayout":        "saves only the calling user's own table layout",
	"MutationResolver.ResetMyTableLayout":       "removes only the calling user's own table layout",
	"QueryResolver.MyBulkEdit":                  "returns only a bulk edit the calling user started",
	"QueryResolver.MyBulkEdits":                 "returns only bulk edits the calling user started",

	"QueryResolver.HomeLayout":          "returns only the calling user's own home layout",
	"QueryResolver.HomeWidgetCatalog":   "static catalog of widgets; carries no resource data",
	"MutationResolver.UpdateHomeLayout": "acts only on the calling user's own home layout",
	"MutationResolver.ResetHomeLayout":  "acts only on the calling user's own home layout",

	"MutationResolver.CreateTableConfiguration": "anyone may keep a private view; the service " +
		"refuses to publish one to the organization without table_configuration:create",
	"MutationResolver.UpdateTableConfiguration": "only the owner may change a view, and the " +
		"service refuses to leave it published without table_configuration:update",
	"MutationResolver.PatchTableConfiguration": "only the owner may change a view, and the " +
		"service refuses to leave it published without table_configuration:update",
	"MutationResolver.DeleteTableConfiguration": "the service lets only the owner delete a view, " +
		"which withdraws it from everyone it was shared with",
	"MutationResolver.SetDefaultTableConfiguration": "sets the caller's own default; the service " +
		"refuses a view the caller does not own",

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

// selfScoped lists, by the reason each holds, the resolvers that act only on the
// caller's own records and so need no permission beyond being signed in. Every
// name is listed: a resolver called My<Thing> earns no exemption by its name.
var selfScoped = map[string][]string{
	"the driver portal resolves the worker linked to the calling user " +
		"(driverportalservice.ResolveWorker) and reads or acts only on that worker": {
		"MutationResolver.AcknowledgeMyDisciplinaryAction",
		"MutationResolver.AcknowledgeMyPolicy",
		"MutationResolver.AcknowledgeMyReview",
		"MutationResolver.AcknowledgeMyTraining",
		"MutationResolver.CancelMyExpense",
		"MutationResolver.CancelMyPTO",
		"MutationResolver.CreateMyLoadComment",
		"MutationResolver.DismissMyNotifications",
		"MutationResolver.MarkAllMyNotificationsRead",
		"MutationResolver.MarkMyNotificationsRead",
		"MutationResolver.MarkMyNotificationsUnread",
		"MutationResolver.ProposeMyShiftSwap",
		"MutationResolver.RecordMyStopAction",
		"MutationResolver.RequestMyPTO",
		"MutationResolver.RespondToMyAssignment",
		"MutationResolver.RespondToMyShiftSwap",
		"MutationResolver.RestoreMyNotifications",
		"MutationResolver.SetMyAvailability",
		"MutationResolver.StartMyTraining",
		"MutationResolver.SubmitMyExpense",
		"MutationResolver.UpdateMyContactInfo",
		"MutationResolver.WithdrawMyProfileChange",
		"QueryResolver.MyAdvances",
		"QueryResolver.MyAvailability",
		"QueryResolver.MyComplianceProfile",
		"QueryResolver.MyCredentials",
		"QueryResolver.MyDisciplinaryActions",
		"QueryResolver.MyDisputes",
		"QueryResolver.MyEscrow",
		"QueryResolver.MyExpenses",
		"QueryResolver.MyHosDailyLogs",
		"QueryResolver.MyHosState",
		"QueryResolver.MyHosViolations",
		"QueryResolver.MyLeave",
		"QueryResolver.MyLoadComments",
		"QueryResolver.MyLoadPayEstimate",
		"QueryResolver.MyLoads",
		"QueryResolver.MyNotificationUnreadCount",
		"QueryResolver.MyNotifications",
		"QueryResolver.MyPTO",
		"QueryResolver.MyPTOBalances",
		"QueryResolver.MyPeriodSummary",
		"QueryResolver.MyPolicies",
		"QueryResolver.MyPolicyDocumentURL",
		"QueryResolver.MyPortalFeatures",
		"QueryResolver.MyPortalProfile",
		"QueryResolver.MyProfileChangeRequests",
		"QueryResolver.MyRecentPayEvents",
		"QueryResolver.MyRecognitions",
		"QueryResolver.MyReviews",
		"QueryResolver.MySafetyScorecard",
		"QueryResolver.MySchedule",
		"QueryResolver.MySettlement",
		"QueryResolver.MySettlements",
		"QueryResolver.MyShiftSwaps",
		"QueryResolver.MyTotalCompensation",
		"QueryResolver.MyTraining",
		"QueryResolver.MyYtdPay",
	},
	"capture devices are listed and revoked as the caller's own: the service is " +
		"asked with Mine set and refuses a device another user registered": {
		"MutationResolver.RevokeMyCaptureDevice",
		"QueryResolver.MyCaptureDevices",
	},
	"reports what capture the calling user may use, from their own session": {
		"QueryResolver.MyCaptureAccess",
	},
}

func isAllowedAuthOnly(root RootResolver) bool {
	_, ok := allowedAuthOnly()[root.Key()]
	return ok
}

func allowedAuthOnly() map[string]string {
	allowed := make(map[string]string, len(authOnlyAllowlist)+64)
	for key, reason := range authOnlyAllowlist {
		allowed[key] = reason
	}
	for reason, keys := range selfScoped {
		for _, key := range keys {
			allowed[key] = reason
		}
	}
	return allowed
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
			if _, listed := allowedAuthOnly()[root.Key()]; listed {
				overListed = append(overListed, root.Key())
			}
		}
	}

	assert.Empty(t, unchecked,
		"root resolvers with no authentication or permission check at all:\n  %s",
		strings.Join(unchecked, "\n  "))

	assert.Empty(t, unlistedAuthOnly,
		"root resolvers that authenticate but never reach a permission check. "+
			"Either add one, list it in selfScoped under the reason it only ever touches "+
			"the caller's own data, or add it to authOnlyAllowlist with a reason explaining "+
			"why every authenticated user may call it:\n  %s",
		strings.Join(unlistedAuthOnly, "\n  "))

	assert.Empty(t, overListed,
		"allowlisted resolvers that now perform a permission check; remove them from "+
			"authOnlyAllowlist so the list stays accurate:\n  %s",
		strings.Join(overListed, "\n  "))

	var stale []string
	for key := range allowedAuthOnly() {
		if _, ok := seen[key]; !ok {
			stale = append(stale, key)
		}
	}
	assert.Empty(t, stale, "authOnlyAllowlist or selfScoped names resolvers that no longer exist:\n  %s",
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

func TestASelfScopedNameEarnsNoExemption(t *testing.T) {
	t.Parallel()

	assert.False(t, isAllowedAuthOnly(RootResolver{Receiver: "QueryResolver", Name: "MyUnlistedThing"}))
	assert.False(t, isAllowedAuthOnly(RootResolver{Receiver: "MutationResolver", Name: "CancelMyThing"}))
	assert.True(t, isAllowedAuthOnly(RootResolver{Receiver: "QueryResolver", Name: "MyLoads"}))
}

func TestSelfScopedResolversAreListedOnce(t *testing.T) {
	t.Parallel()

	seen := make(map[string]string)
	for reason, keys := range selfScoped {
		for _, key := range keys {
			previous, dup := seen[key]
			assert.Falsef(t, dup, "%s is listed under %q and %q", key, previous, reason)
			_, alsoListed := authOnlyAllowlist[key]
			assert.Falsef(t, alsoListed, "%s is in both selfScoped and authOnlyAllowlist", key)
			seen[key] = reason
		}
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
		"QueryResolver.AgentToolRuleConnection",
		"QueryResolver.AgentSafetySummary",
		"QueryResolver.AgentSafety",
		"QueryResolver.AgentToolHolders",
		"QueryResolver.AgentToolRuleImpact",
		"MutationResolver.SaveAgentToolRule",
	} {
		verdict, ok := verdicts[key]
		require.True(t, ok, "%s is not a root resolver", key)
		assert.Equal(t, VerdictPermission, verdict, "%s must reach a permission check", key)
		_, listed := allowedAuthOnly()[key]
		assert.False(t, listed, "%s must not be allowlisted as auth-only", key)
	}
}

// Who may use which agent is decided by these operations. The self-scoped
// ones are named My<Thing> yet still reach the writes deciding does, so each
// is held to a permission check of its own.
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
		"MutationResolver.DecideMyProposals",
		// Self-scoped, yet it runs the plan's writes, so it must hold
		// assistant:update before the service checks the thread is the
		// caller's and the agent is theirs.
		"MutationResolver.DecideMyPlan",
		// Undoing or hurrying an approval reaches the same writes deciding
		// does, so it is held to the same check.
		"MutationResolver.UndoMyDecision",
		"MutationResolver.CommitMyDecisionNow",
		"QueryResolver.AgentAccessPreview",
		"QueryResolver.PendingDecisions",
		"QueryResolver.PendingDecisionSummary",
	} {
		verdict, ok := verdicts[key]
		require.True(t, ok, "%s is not a root resolver", key)
		assert.Equal(t, VerdictPermission, verdict, "%s must reach a permission check", key)
		_, listed := allowedAuthOnly()[key]
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
		"QueryResolver.AgentQualityAgentConnection",
		"QueryResolver.AgentWorstRatedAnswerConnection",
		"QueryResolver.AgentSuiteRunConnection",
		"QueryResolver.AgentSuiteRun",
		"QueryResolver.AgentSuiteRunCaseConnection",
		"QueryResolver.AgentQualityControl",
		"MutationResolver.RunAgentSuite",
		"MutationResolver.UpdateAgentQualityControl",
	} {
		verdict, ok := verdicts[key]
		require.True(t, ok, "%s is not a root resolver", key)
		assert.Equal(t, VerdictPermission, verdict, "%s must reach a permission check", key)
		_, listed := allowedAuthOnly()[key]
		assert.False(t, listed, "%s must not be allowlisted as auth-only", key)
	}
}
