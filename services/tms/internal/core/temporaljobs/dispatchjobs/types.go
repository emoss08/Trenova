package dispatchjobs

const HorizonPlanSweepWorkflowName = "HorizonPlanSweepWorkflow"

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
