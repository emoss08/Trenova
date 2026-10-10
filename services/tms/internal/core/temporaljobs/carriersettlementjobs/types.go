package carriersettlementjobs

const (
	GenerateCarrierSettlementBatchesWorkflowName = "GenerateCarrierSettlementBatchesWorkflow"
)

type GenerateCarrierSettlementBatchesResult struct {
	OrganizationsChecked int   `json:"organizationsChecked"`
	BatchesGenerated     int   `json:"batchesGenerated"`
	Skipped              int   `json:"skipped"`
	Failed               int   `json:"failed"`
	CompletedAt          int64 `json:"completedAt"`
}
