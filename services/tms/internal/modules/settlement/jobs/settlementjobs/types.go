package settlementjobs

const (
	GenerateSettlementBatchesWorkflowName = "GenerateSettlementBatchesWorkflow"
	AccrueEscrowInterestWorkflowName      = "AccrueEscrowInterestWorkflow"
)

type GenerateSettlementBatchesResult struct {
	OrganizationsChecked int   `json:"organizationsChecked"`
	BatchesGenerated     int   `json:"batchesGenerated"`
	Skipped              int   `json:"skipped"`
	Failed               int   `json:"failed"`
	CompletedAt          int64 `json:"completedAt"`
}

type AccrueEscrowInterestResult struct {
	AccountsAccrued int   `json:"accountsAccrued"`
	Failed          int   `json:"failed"`
	CompletedAt     int64 `json:"completedAt"`
}
