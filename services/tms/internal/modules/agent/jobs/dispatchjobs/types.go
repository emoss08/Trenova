package dispatchjobs

const HorizonPlanSweepWorkflowName = "HorizonPlanSweepWorkflow"

type TenantHorizonPlan struct {
	OrganizationID string `json:"organizationId"`
	BusinessUnitID string `json:"businessUnitId"`
	RunID          string `json:"runId,omitempty"`

	MovesPlanned   int   `json:"movesPlanned"`
	MovesUncovered int   `json:"movesUncovered"`
	MovesAtRisk    int   `json:"movesAtRisk"`
	ToursBuilt     int   `json:"toursBuilt"`
	ChainedMoves   int   `json:"chainedMoves"`
	CostCents      int64 `json:"costCents"`
	UncoveredCents int64 `json:"uncoveredCents"`

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
	MovesAtRisk    int                  `json:"movesAtRisk"`
	ToursBuilt     int                  `json:"toursBuilt"`
	ChainedMoves   int                  `json:"chainedMoves"`
	TenantOutcomes []*TenantHorizonPlan `json:"tenantOutcomes,omitempty"`
}

// CoverageRefreshResult is what one debounced coverage refresh did.
type CoverageRefreshResult struct {
	Skipped        bool   `json:"skipped,omitempty"`
	RunID          string `json:"runId,omitempty"`
	MovesProposed  int    `json:"movesProposed"`
	MovesUncovered int    `json:"movesUncovered"`
	MovesAtRisk    int    `json:"movesAtRisk"`
}
