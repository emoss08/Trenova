package carriersettlementjobs

const (
	GenerateCarrierSettlementBatchesWorkflowName = "GenerateCarrierSettlementBatchesWorkflow"
)

type GenerateCarrierSettlementBatchesResult struct {
	OrganizationsChecked int   `json:"organizationsChecked"`
	BatchesGenerated     int   `json:"batchesGenerated"`
	Failed               int   `json:"failed"`
	CompletedAt          int64 `json:"completedAt"`
}
