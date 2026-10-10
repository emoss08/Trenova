package kernelports

// RunStepOwnerKind says which ledger a step belongs to. One table serves both
// because the rule is the same for both; a column cannot carry two foreign
// keys, so the rows are swept on a retention schedule rather than cascaded.
type RunStepOwnerKind string
