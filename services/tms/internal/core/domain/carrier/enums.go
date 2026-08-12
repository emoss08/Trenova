package carrier

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
	StatusDoNotUse = Status("DoNotUse")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusDoNotUse:
		return true
	}
	return false
}

type Type string

const (
	TypeCommon   = Type("Common")
	TypeContract = Type("Contract")
	TypeBroker   = Type("Broker")
	TypeExempt   = Type("Exempt")
)

func (t Type) String() string {
	return string(t)
}

func (t Type) IsValid() bool {
	switch t {
	case TypeCommon, TypeContract, TypeBroker, TypeExempt:
		return true
	}
	return false
}

type ComplianceStatus string

const (
	ComplianceStatusPending      = ComplianceStatus("Pending")
	ComplianceStatusQualified    = ComplianceStatus("Qualified")
	ComplianceStatusDisqualified = ComplianceStatus("Disqualified")
	ComplianceStatusExpired      = ComplianceStatus("Expired")
)

func (cs ComplianceStatus) String() string {
	return string(cs)
}

func (cs ComplianceStatus) IsValid() bool {
	switch cs {
	case ComplianceStatusPending, ComplianceStatusQualified,
		ComplianceStatusDisqualified, ComplianceStatusExpired:
		return true
	}
	return false
}

type SafetyRating string

const (
	SafetyRatingSatisfactory   = SafetyRating("Satisfactory")
	SafetyRatingConditional    = SafetyRating("Conditional")
	SafetyRatingUnsatisfactory = SafetyRating("Unsatisfactory")
	SafetyRatingNotRated       = SafetyRating("NotRated")
)

func (sr SafetyRating) String() string {
	return string(sr)
}

func (sr SafetyRating) IsValid() bool {
	switch sr {
	case SafetyRatingSatisfactory, SafetyRatingConditional,
		SafetyRatingUnsatisfactory, SafetyRatingNotRated:
		return true
	}
	return false
}

type TaxIDType string

const (
	TaxIDTypeEIN = TaxIDType("EIN")
	TaxIDTypeSSN = TaxIDType("SSN")
)

func (t TaxIDType) String() string {
	return string(t)
}

func (t TaxIDType) IsValid() bool {
	switch t {
	case TaxIDTypeEIN, TaxIDTypeSSN:
		return true
	}
	return false
}

type PaymentMethod string

const (
	PaymentMethodCheck     = PaymentMethod("Check")
	PaymentMethodACHManual = PaymentMethod("ACHManual")
)

func (pm PaymentMethod) String() string {
	return string(pm)
}

func (pm PaymentMethod) IsValid() bool {
	switch pm {
	case PaymentMethodCheck, PaymentMethodACHManual:
		return true
	}
	return false
}

type InsurancePolicyType string

const (
	InsurancePolicyTypeAutoLiability    = InsurancePolicyType("AutoLiability")
	InsurancePolicyTypeCargoLiability   = InsurancePolicyType("CargoLiability")
	InsurancePolicyTypeGeneralLiability = InsurancePolicyType("GeneralLiability")
	InsurancePolicyTypeWorkersComp      = InsurancePolicyType("WorkersComp")
	InsurancePolicyTypeUmbrella         = InsurancePolicyType("Umbrella")
)

func (pt InsurancePolicyType) String() string {
	return string(pt)
}

func (pt InsurancePolicyType) IsValid() bool {
	switch pt {
	case InsurancePolicyTypeAutoLiability, InsurancePolicyTypeCargoLiability,
		InsurancePolicyTypeGeneralLiability, InsurancePolicyTypeWorkersComp,
		InsurancePolicyTypeUmbrella:
		return true
	}
	return false
}

// RequiredInsurancePolicyTypes are the coverages a carrier must hold, unexpired,
// before a shipment move may be covered by it.
var RequiredInsurancePolicyTypes = []InsurancePolicyType{
	InsurancePolicyTypeAutoLiability,
	InsurancePolicyTypeCargoLiability,
}
