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
	"queryResolver.SelectOptions": "select options are deliberately readable by every " +
		"authenticated user in the tenant: forms on pages a user cannot open still need " +
		"their dropdowns, and the payload is id, label and display metadata only",

	"queryResolver.Notifications":               "returns only the calling user's own notifications",
	"queryResolver.NotificationUnreadCount":     "returns only the calling user's own notifications",
	"mutationResolver.MarkNotificationsRead":    "acts only on the calling user's own notifications",
	"mutationResolver.MarkNotificationsUnread":  "acts only on the calling user's own notifications",
	"mutationResolver.MarkAllNotificationsRead": "acts only on the calling user's own notifications",
	"mutationResolver.DismissNotifications":     "acts only on the calling user's own notifications",
	"mutationResolver.RestoreNotifications":     "acts only on the calling user's own notifications",

	"queryResolver.SidebarPreferences":          "returns only the calling user's own preferences",
	"queryResolver.SidebarCustomizationOptions": "returns only the calling user's own preferences",
	"mutationResolver.UpdateSidebarPreferences": "acts only on the calling user's own preferences",

	"queryResolver.HomeLayout":          "returns only the calling user's own home layout",
	"queryResolver.HomeWidgetCatalog":   "static catalog of widgets; carries no resource data",
	"mutationResolver.UpdateHomeLayout": "acts only on the calling user's own home layout",
	"mutationResolver.ResetHomeLayout":  "acts only on the calling user's own home layout",

	"queryResolver.TableConfiguration":              "table layouts are UI preferences, not resource data",
	"queryResolver.TableConfigurations":             "table layouts are UI preferences, not resource data",
	"queryResolver.DefaultTableConfiguration":       "table layouts are UI preferences, not resource data",
	"mutationResolver.CreateTableConfiguration":     "table layouts are UI preferences, not resource data",
	"mutationResolver.UpdateTableConfiguration":     "table layouts are UI preferences, not resource data",
	"mutationResolver.PatchTableConfiguration":      "table layouts are UI preferences, not resource data",
	"mutationResolver.DeleteTableConfiguration":     "table layouts are UI preferences, not resource data",
	"mutationResolver.SetDefaultTableConfiguration": "table layouts are UI preferences, not resource data",

	"queryResolver.TelematicsStatus": "org-wide integration health indicator shown in the " +
		"application shell; carries no resource data",

	"mutationResolver.CreateSettlementDispute":   "a driver disputing their own settlement",
	"mutationResolver.WithdrawSettlementDispute": "a driver withdrawing their own dispute",
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
