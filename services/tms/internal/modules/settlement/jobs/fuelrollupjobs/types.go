package fuelrollupjobs

const (
	RefreshWorkflowName = "RefreshFuelTractorDaysWorkflow"
	RecentScheduleID    = "fuel-tractor-days-recent"
	TrailingScheduleID  = "fuel-tractor-days-trailing"
)

type RefreshInput struct {
	Days int `json:"days"`
}

type RefreshResult struct {
	Rows int64 `json:"rows"`
}
