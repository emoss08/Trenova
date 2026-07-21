package recurringshipmentjobs

const DispatchDueRecurringShipmentsWorkflowName = "DispatchDueRecurringShipmentsWorkflow"

type DispatchDueRecurringShipmentsResult struct {
	Dispatched  int   `json:"dispatched"`
	Skipped     int   `json:"skipped"`
	Failed      int   `json:"failed"`
	CompletedAt int64 `json:"completedAt"`
}
