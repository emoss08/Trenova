package dispatchjobs

const HorizonPlanSweepWorkflowName = "HorizonPlanSweepWorkflow"

// TenantHorizonPlan is one organization's outcome from a scheduled planning pass.
// It is deliberately a summary rather than the plan itself: the plan is already
// persisted as an agent run with per-move proposals, and this exists so the sweep's
// history shows whether the planner is earning its keep.
type TenantHorizonPlan struct {
	OrganizationID string `json:"organizationId"`
	BusinessUnitID string `json:"businessUnitId"`
	RunID          string `json:"runId,omitempty"`

	MovesPlanned   int `json:"movesPlanned"`
	MovesUncovered int `json:"movesUncovered"`
	ToursBuilt     int `json:"toursBuilt"`
	ChainedMoves   int `json:"chainedMoves"`
	TotalScore     int `json:"totalScore"`

	TotalDeadheadMiles float64 `json:"totalDeadheadMiles"`
	ShadowMode         bool    `json:"shadowMode"`

	// ProposalsRetired is how many of this pass's proposals were moved out of the
	// pending queue. A scheduled plan is evidence, not work for a dispatcher.
	ProposalsRetired int    `json:"proposalsRetired"`
	Error            string `json:"error,omitempty"`
}

type HorizonPlanSweepResult struct {
	TenantsScanned int                  `json:"tenantsScanned"`
	TenantsPlanned int                  `json:"tenantsPlanned"`
	TenantsFailed  int                  `json:"tenantsFailed"`
	MovesPlanned   int                  `json:"movesPlanned"`
	MovesUncovered int                  `json:"movesUncovered"`
	ToursBuilt     int                  `json:"toursBuilt"`
	ChainedMoves   int                  `json:"chainedMoves"`
	TenantOutcomes []*TenantHorizonPlan `json:"tenantOutcomes,omitempty"`
}
