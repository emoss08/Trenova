package compliancejobs

const CredentialExpirySweepWorkflowName = "CredentialExpirySweepWorkflow" //nolint:gosec // Temporal workflow name, not a credential.

type CredentialExpirySweepResult struct {
	CredentialsChecked  int                          `json:"credentialsChecked"`
	WorkersChecked      int                          `json:"workersChecked"`
	DriverNotifications int                          `json:"driverNotifications"`
	ComplianceAlerts    int                          `json:"complianceAlerts"`
	Failed              int                          `json:"failed"`
	Training            *TrainingReminderSweepResult `json:"training,omitempty"`
	Safety              *SafetyRollupSweepResult     `json:"safety,omitempty"`
	DrugAlcohol         *DrugAlcoholSweepResult      `json:"drugAlcohol,omitempty"`
	Digest              *DriverDigestSweepResult     `json:"digest,omitempty"`
}

// TrainingReminderSweepResult counts what the training pass of the nightly
// sweep did: due-date nudges to drivers, overdue/lapsed alerts to the office,
// and completions flipped to Expired.
type TrainingReminderSweepResult struct {
	RecordsChecked      int `json:"recordsChecked"`
	DriverNotifications int `json:"driverNotifications"`
	ComplianceAlerts    int `json:"complianceAlerts"`
	Expired             int `json:"expired"`
	Failed              int `json:"failed"`
}
