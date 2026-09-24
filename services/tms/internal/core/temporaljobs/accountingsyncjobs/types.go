package accountingsyncjobs

const (
	CheckAccountingConnectionsWorkflowName = "CheckAccountingConnectionsWorkflow"
	healthPageSize                         = 25
	healthMaxPages                         = 40
)

type HealthSweepResult struct {
	Listed  int `json:"listed"`
	Checked int `json:"checked"`
	Failed  int `json:"failed"`
	Pages   int `json:"pages"`
}
