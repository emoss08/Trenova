package accountingconnection

type SetupStep string

const (
	SetupStepMode      = SetupStep("Mode")
	SetupStepMappings  = SetupStep("Mappings")
	SetupStepStartDate = SetupStep("StartDate")
	SetupStepComplete  = SetupStep("Complete")
)

func (s SetupStep) String() string { return string(s) }

func (s SetupStep) IsValid() bool {
	switch s {
	case SetupStepMode, SetupStepMappings, SetupStepStartDate, SetupStepComplete:
		return true
	default:
		return false
	}
}

func AllSetupSteps() []SetupStep {
	return []SetupStep{SetupStepMode, SetupStepMappings, SetupStepStartDate, SetupStepComplete}
}
