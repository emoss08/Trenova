package compliancejobs

const CredentialExpirySweepWorkflowName = "CredentialExpirySweepWorkflow" //nolint:gosec // Temporal workflow name, not a credential.

type CredentialExpirySweepResult struct {
	WorkersChecked      int `json:"workersChecked"`
	DriverNotifications int `json:"driverNotifications"`
	ComplianceAlerts    int `json:"complianceAlerts"`
	Failed              int `json:"failed"`
}
