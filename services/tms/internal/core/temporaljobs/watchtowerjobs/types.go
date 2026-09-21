// Package watchtowerjobs keeps the watchtower honest.
//
// The feed is a projection: every source upserts and resolves live as its
// records open and close. A projection that is only ever written live drifts,
// because a process that dies between the write and the projection leaves an
// item nobody will ever close. So the feed is also swept: a nightly reconcile
// asks each source what is open right now, adds what is missing and resolves
// what the source no longer reports, and a backfill does the same on demand
// for an organization whose tower has never been filled.
//
// What has resolved is kept for a month so a person can look back at a week
// they missed, then removed; the sources it was drawn from are permanent.
package watchtowerjobs

const (
	// WatchtowerReconcileWorkflowName corrects every tenant's feed against
	// its sources.
	WatchtowerReconcileWorkflowName = "WatchtowerReconcileWorkflow"
	// WatchtowerBackfillWorkflowName fills one tenant's feed, or every
	// tenant's, from the sources' current state.
	WatchtowerBackfillWorkflowName = "WatchtowerBackfillWorkflow"
	// WatchtowerRetentionWorkflowName removes items resolved long enough ago
	// that nobody is looking back at them.
	WatchtowerRetentionWorkflowName = "WatchtowerRetentionWorkflow"
)

// WatchtowerBackfillInput names the tenant to fill. An empty organization
// fills every one of them, which is what the scheduled reconcile does.
type WatchtowerBackfillInput struct {
	OrganizationID string `json:"organizationId"`
	BusinessUnitID string `json:"businessUnitId"`
}

// WatchtowerSweepResult reports one sweep across the tenants it covered.
type WatchtowerSweepResult struct {
	OrganizationsProcessed int `json:"organizationsProcessed"`
	Upserted               int `json:"upserted"`
	Resolved               int `json:"resolved"`
	// FailedOrganizations names tenants whose sweep could not run at all.
	// One organization's broken source must not stop every other tenant's
	// feed from being corrected.
	FailedOrganizations []string `json:"failedOrganizations"`
	// FailedSources names the sources that could not be read, deduplicated
	// across tenants: a source broken everywhere is one line, not five
	// hundred. A source that fails resolves nothing, so a tower whose
	// source is down keeps what it had rather than emptying.
	FailedSources []string `json:"failedSources"`
}

// WatchtowerRetentionResult reports what the retention sweep removed.
type WatchtowerRetentionResult struct {
	Deleted int `json:"deleted"`
}
