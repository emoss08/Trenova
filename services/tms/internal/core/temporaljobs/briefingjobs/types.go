// Package briefingjobs writes the morning briefing.
//
// It runs hourly rather than once a night, because the hour that matters
// is the organization's own: a company with an office in Newark and one in
// Reno wants each to open a page written for its own morning, not one of
// them reading yesterday. Each firing writes for the organizations whose
// local clock has just reached their briefing hour, so every tenant is
// written exactly once a day whatever its timezone.
package briefingjobs

const (
	// DailyBriefingWorkflowName writes the morning for every organization
	// whose local briefing hour has just come round.
	DailyBriefingWorkflowName = "DailyBriefingWorkflow"
	// BriefingRetentionWorkflowName removes briefings nobody reads back to.
	BriefingRetentionWorkflowName = "BriefingRetentionWorkflow"
)

// DailyBriefingResult reports one hour's sweep.
type DailyBriefingResult struct {
	OrganizationsDue int `json:"organizationsDue"`
	BriefingsWritten int `json:"briefingsWritten"`
	// BriefingsNarrated counts the pages a model wrote wording for. A page
	// it did not is still a complete briefing, so this is a health figure
	// rather than a failure count.
	BriefingsNarrated   int      `json:"briefingsNarrated"`
	FailedOrganizations []string `json:"failedOrganizations"`
}

// BriefingRetentionResult reports what the retention sweep removed.
type BriefingRetentionResult struct {
	Deleted int `json:"deleted"`
}
