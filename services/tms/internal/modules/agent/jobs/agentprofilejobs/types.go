// Package agentprofilejobs writes the profiles agents are told about an
// organization and the people in it: every night for each organization and
// each person who talked to an agent lately, and on demand when someone asks
// or guidance changes.
package agentprofilejobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	SweepWorkflowName        = "AgentProfileSweepWorkflow"
	OrganizationWorkflowName = "AgentProfileOrganizationWorkflow"
	ProfileWorkflowName      = "AgentProfileWorkflow"

	// ActiveUserDays is how recently a person must have talked to an agent for
	// the nightly pass to keep their profile.
	ActiveUserDays = 14

	// userConcurrency bounds how many people's profiles one organization writes
	// at once, and organizationConcurrency how many organizations a sweep
	// works on at once.
	userConcurrency         = 4
	organizationConcurrency = 4
)

type SweepInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Now   int64                        `json:"now,omitempty"`
}

type ListOrganizationsInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Limit int                          `json:"limit"`
}

type OrganizationInput struct {
	temporaljobs.TenantWorkItem

	Now int64 `json:"now"`
}

type ActiveUsersInput struct {
	temporaljobs.TenantWorkItem

	Since int64 `json:"since"`
}

// ModelReply is the model step's answer. NoProvider means the organization has
// no AI provider for the task: nothing is written and nothing failed.
type ModelReply struct {
	Result     *agentports.StructuredCompletionResult `json:"result,omitempty"`
	NoProvider bool                                   `json:"noProvider,omitempty"`
}

type ProfileResult struct {
	Written int `json:"written"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

func (r *ProfileResult) add(other *ProfileResult) {
	if other == nil {
		return
	}
	r.Written += other.Written
	r.Failed += other.Failed
	r.Skipped += other.Skipped
}

func (r *ProfileResult) absorb(outcome *agentprofileports.GenerationOutcome) {
	switch {
	case outcome == nil || outcome.ProfileID.IsNil():
		r.Skipped++
	case outcome.Status == agentprofile.StatusFailed:
		r.Failed++
	default:
		r.Written++
	}
}

type OrganizationResult struct {
	ProfileResult

	People      int        `json:"people"`
	FailedUsers []pulid.ID `json:"failedUsers,omitempty"`
}

type SweepResult struct {
	ProfileResult

	Organizations       int      `json:"organizations"`
	People              int      `json:"people"`
	FailedOrganizations []string `json:"failedOrganizations,omitempty"`
}
