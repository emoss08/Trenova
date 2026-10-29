package driverportalrecordsservice

// PortalFeatures is the driver-facing view of the org's DashControl: which
// portal capabilities the carrier has enabled for its drivers.
type PortalFeatures struct {
	RequireLoadAcknowledgment  bool `json:"requireLoadAcknowledgment"`
	AllowLoadRefusals          bool `json:"allowLoadRefusals"`
	AllowStopActions           bool `json:"allowStopActions"`
	AllowLoadDocumentUpload    bool `json:"allowLoadDocumentUpload"`
	AllowLoadComments          bool `json:"allowLoadComments"`
	ShowLoadPay                bool `json:"showLoadPay"`
	ShowPayEstimates           bool `json:"showPayEstimates"`
	AllowExpenseSubmission     bool `json:"allowExpenseSubmission"`
	RequireExpenseReceipt      bool `json:"requireExpenseReceipt"`
	AllowSettlementDisputes    bool `json:"allowSettlementDisputes"`
	AllowProfileDocumentUpload bool `json:"allowProfileDocumentUpload"`
	AllowContactInfoEdit       bool `json:"allowContactInfoEdit"`
	AllowPtoRequests           bool `json:"allowPtoRequests"`
	PtoBalances                bool `json:"ptoBalances"`
	// LeaveBalance is on only once the driver has a leave case. An empty FMLA
	// card reads as an entitlement they are owed rather than one they have not
	// started drawing on.
	LeaveBalance bool `json:"leaveBalance"`
	// Schedule is on once the driver is actually on a shift. An empty rota
	// reads as a roster nobody filled in rather than as a job with no fixed
	// pattern, which is what an owner-operator has.
	Schedule bool `json:"schedule"`
	// ShiftSwaps follows Schedule: a day you are not rostered on is not a day
	// you can offer anybody.
	ShiftSwaps bool `json:"shiftSwaps"`
	// RequireContactChangeApproval is surfaced so the app can say "this will
	// be sent to your carrier" before the driver saves, not after.
	RequireContactChangeApproval bool `json:"requireContactChangeApproval"`
	// PoliciesOutstanding is how many signatures the driver still owes.
	PoliciesOutstanding int `json:"policiesOutstanding"`
}
