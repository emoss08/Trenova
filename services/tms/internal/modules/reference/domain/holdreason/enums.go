package holdreason

type HoldType string

const (
	HoldTypeOperational = HoldType("OperationalHold")
	HoldTypeCompliance  = HoldType("ComplianceHold")
	HoldTypeCustomer    = HoldType("CustomerHold")
	HoldTypeFinance     = HoldType("FinanceHold")
)

type HoldSeverity string

const (
	HoldSeverityInformational = HoldSeverity("Informational")
	HoldSeverityAdvisory      = HoldSeverity("Advisory")
	HoldSeverityBlocking      = HoldSeverity("Blocking")
)
