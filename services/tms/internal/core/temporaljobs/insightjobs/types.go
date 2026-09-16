// Package insightjobs recomputes home-screen insights on a schedule.
//
// Generation is scheduled rather than done on page load for three reasons that
// all matter: a refresh runs several aggregate queries over a month of freight,
// it may call a model, and it has to supersede what it found last time. None of
// that belongs on the critical path of someone opening their home screen, and
// none of it should happen a hundred times because a hundred people logged in.
package insightjobs

// InsightRefreshWorkflowName is the registered workflow name.
const InsightRefreshWorkflowName = "InsightRefreshWorkflow"

// InsightRefreshResult reports the sweep across every tenant.
type InsightRefreshResult struct {
	OrganizationsProcessed int      `json:"organizationsProcessed"`
	InsightsCreated        int      `json:"insightsCreated"`
	InsightsResolved       int      `json:"insightsResolved"`
	InsightsSuppressed     int      `json:"insightsSuppressed"`
	InsightsNarrated       int      `json:"insightsNarrated"`
	// FailedOrganizations names tenants whose refresh could not complete. The
	// sweep still succeeds: one organization's broken data must not stop every
	// other organization's insights from updating.
	FailedOrganizations []string `json:"failedOrganizations"`
	// FailedDetectors names detector failures across all tenants, deduplicated,
	// so a rule that is broken everywhere is visible as one line rather than
	// buried in per-tenant noise.
	FailedDetectors []string `json:"failedDetectors"`
}
