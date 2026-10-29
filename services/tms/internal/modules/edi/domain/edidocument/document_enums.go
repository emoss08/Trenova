package edidocument

type DocumentDirection string

const (
	DocumentDirectionInbound  = DocumentDirection("Inbound")
	DocumentDirectionOutbound = DocumentDirection("Outbound")
)

type EDIStandard string

const (
	EDIStandardX12 = EDIStandard("X12")
)

type TransactionSet string

const (
	TransactionSet204 = TransactionSet("204")
	TransactionSet210 = TransactionSet("210")
	TransactionSet214 = TransactionSet("214")
	TransactionSet990 = TransactionSet("990")
	TransactionSet997 = TransactionSet("997")
	TransactionSet999 = TransactionSet("999")
)

type DocumentStatus string

const (
	DocumentStatusActive   = DocumentStatus("Active")
	DocumentStatusInactive = DocumentStatus("Inactive")
)

type ValidationSeverity string

const (
	ValidationSeverityInfo    = ValidationSeverity("Info")
	ValidationSeverityWarning = ValidationSeverity("Warning")
	ValidationSeverityError   = ValidationSeverity("Error")
)

type ControlNumberKind string

const (
	ControlNumberKindInterchange = ControlNumberKind("Interchange")
	ControlNumberKindGroup       = ControlNumberKind("Group")
	ControlNumberKindTransaction = ControlNumberKind("Transaction")
)

type AcknowledgmentType string

const (
	AcknowledgmentTypeNone = AcknowledgmentType("None")
	AcknowledgmentType997  = AcknowledgmentType("997")
	AcknowledgmentType999  = AcknowledgmentType("999")
)

func (d DocumentDirection) IsValid() bool {
	switch d {
	case DocumentDirectionInbound,
		DocumentDirectionOutbound:
		return true
	default:
		return false
	}
}

func (s EDIStandard) IsValid() bool {
	switch s {
	case EDIStandardX12:
		return true
	default:
		return false
	}
}

func (t TransactionSet) IsAcknowledgment() bool {
	switch t {
	case TransactionSet997, TransactionSet999:
		return true
	case TransactionSet204, TransactionSet210, TransactionSet214, TransactionSet990:
		return false
	default:
		return false
	}
}

func (t TransactionSet) IsValid() bool {
	switch t {
	case TransactionSet204,
		TransactionSet210,
		TransactionSet214,
		TransactionSet990,
		TransactionSet997,
		TransactionSet999:
		return true
	default:
		return false
	}
}

func (s DocumentStatus) IsValid() bool {
	switch s {
	case DocumentStatusActive,
		DocumentStatusInactive:
		return true
	default:
		return false
	}
}

func (s ValidationSeverity) IsValid() bool {
	switch s {
	case ValidationSeverityInfo,
		ValidationSeverityWarning,
		ValidationSeverityError:
		return true
	default:
		return false
	}
}

func (k ControlNumberKind) IsValid() bool {
	switch k {
	case ControlNumberKindInterchange,
		ControlNumberKindGroup,
		ControlNumberKindTransaction:
		return true
	default:
		return false
	}
}

func (t AcknowledgmentType) IsValid() bool {
	switch t {
	case AcknowledgmentTypeNone,
		AcknowledgmentType997,
		AcknowledgmentType999:
		return true
	default:
		return false
	}
}
