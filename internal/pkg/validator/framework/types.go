package framework

// ValidationStage represents a stage in the validation process
type ValidationStage int

const (
	// ValidationStageBasic represents basic validation (field presence, format, etc.)
	ValidationStageBasic ValidationStage = iota
	// ValidationStageDataIntegrity represents data integrity validation (uniqueness, references, etc.)
	ValidationStageDataIntegrity
	// ValidationStageBusinessRules represents business rules validation (domain-specific rules)
	ValidationStageBusinessRules
	// ValidationStageCompliance represents compliance validation (regulatory requirements)
	ValidationStageCompliance
)

// ValidationPriority represents the priority of a validation
type ValidationPriority int

const (
	// ValidationPriorityHigh represents high priority validation (must pass)
	ValidationPriorityHigh ValidationPriority = iota
	// ValidationPriorityMedium represents medium priority validation (should pass)
	ValidationPriorityMedium
	// ValidationPriorityLow represents low priority validation (nice to pass)
	ValidationPriorityLow
)
