// Package insightjobs recomputes home-screen insights on a schedule.
//
// Generation is scheduled rather than done on page load for three reasons that
// all matter: a refresh runs several aggregate queries over a month of freight,
// it may call a model, and it has to supersede what it found last time. None of
// that belongs on the critical path of someone opening their home screen, and
// none of it should happen a hundred times because a hundred people logged in.
package insightjobs

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/temporaljobs"
)

const (
	// InsightRefreshWorkflowName is the registered workflow name.
	InsightRefreshWorkflowName = "InsightRefreshWorkflow"
	// RefreshOrganizationInsightsWorkflowName refreshes one organization.
	RefreshOrganizationInsightsWorkflowName = "RefreshOrganizationInsightsWorkflow"
)

// InsightRefreshInput carries a sweep across continue-as-new: where the next
// page starts, and the one instant every organization in the sweep describes.
type InsightRefreshInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Now   int64                        `json:"now,omitempty"`
}

// ListOrganizationsInput asks for one page of organizations.
type ListOrganizationsInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Limit int                          `json:"limit"`
}

// OrganizationInsightsInput refreshes one organization as of the sweep's
// instant.
type OrganizationInsightsInput struct {
	temporaljobs.TenantWorkItem

	Now int64 `json:"now"`
}

// OrganizationInsightsResult is what one organization's refresh did.
type OrganizationInsightsResult struct {
	Created    int      `json:"created"`
	Resolved   int      `json:"resolved"`
	Suppressed int      `json:"suppressed"`
	Narrated   int      `json:"narrated"`
	Failed     []string `json:"failed,omitempty"`
}

// InsightRefreshResult reports the sweep across every tenant.
type InsightRefreshResult struct {
	OrganizationsProcessed int `json:"organizationsProcessed"`
	InsightsCreated        int `json:"insightsCreated"`
	InsightsResolved       int `json:"insightsResolved"`
	InsightsSuppressed     int `json:"insightsSuppressed"`
	InsightsNarrated       int `json:"insightsNarrated"`
	// FailedOrganizations names tenants whose refresh could not complete. The
	// sweep still succeeds: one organization's broken data must not stop every
	// other organization's insights from updating.
	FailedOrganizations []string `json:"failedOrganizations"`
	// FailedDetectors names detector failures across all tenants, deduplicated,
	// so a rule that is broken everywhere is visible as one line rather than
	// buried in per-tenant noise.
	FailedDetectors []string `json:"failedDetectors"`
}

func newInsightRefreshResult() *InsightRefreshResult {
	return &InsightRefreshResult{
		FailedOrganizations: make([]string, 0),
		FailedDetectors:     make([]string, 0),
	}
}

// absorb adds one organization's refresh to the sweep's.
func (r *InsightRefreshResult) absorb(organization *OrganizationInsightsResult) {
	r.OrganizationsProcessed++
	r.InsightsCreated += organization.Created
	r.InsightsResolved += organization.Resolved
	r.InsightsSuppressed += organization.Suppressed
	r.InsightsNarrated += organization.Narrated

	for _, failed := range organization.Failed {
		// A detector broken by a schema change fails for every tenant. One
		// line naming it is actionable; five hundred identical lines are not.
		if !slices.Contains(r.FailedDetectors, failed) {
			r.FailedDetectors = append(r.FailedDetectors, failed)
		}
	}
}
