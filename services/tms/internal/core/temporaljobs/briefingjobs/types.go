// Package briefingjobs writes the morning briefing.
//
// It runs hourly rather than once a night, because the hour that matters
// is the organization's own: a company with an office in Newark and one in
// Reno wants each to open a page written for its own morning, not one of
// them reading yesterday. Each firing writes for the organizations whose
// local clock has just reached their briefing hour, so every tenant is
// written exactly once a day whatever its timezone.
package briefingjobs

import "github.com/emoss08/trenova/internal/core/temporaljobs"

const (
	// DailyBriefingWorkflowName writes the morning for every organization
	// whose local briefing hour has just come round.
	DailyBriefingWorkflowName = "DailyBriefingWorkflow"
	// WriteOrganizationBriefingWorkflowName writes one organization's
	// morning.
	WriteOrganizationBriefingWorkflowName = "WriteOrganizationBriefingWorkflow"
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

// DueOrganizationsInput asks which organizations' briefing hour has come as
// of the sweep's instant.
type DueOrganizationsInput struct {
	Now int64 `json:"now"`
}

// DueOrganizations are the organizations to write for this hour, and those
// whose hour could not be decided.
type DueOrganizations struct {
	Due    []temporaljobs.TenantWorkItem `json:"due"`
	Failed []string                      `json:"failed,omitempty"`
}

// OrganizationBriefingInput writes one organization's morning as of the
// sweep's instant.
type OrganizationBriefingInput struct {
	temporaljobs.TenantWorkItem

	Now int64 `json:"now"`
}

// OrganizationBriefingResult is what one organization's morning came to.
type OrganizationBriefingResult struct {
	Written  int `json:"written"`
	Narrated int `json:"narrated"`
}
