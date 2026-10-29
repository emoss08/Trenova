package driverpay

type PayeeClassification string

const (
	PayeeClassificationCompanyDriver = PayeeClassification("CompanyDriver")
	PayeeClassificationOwnerOperator = PayeeClassification("OwnerOperator")
)

func (p PayeeClassification) String() string { return string(p) }

func (p PayeeClassification) IsValid() bool {
	switch p {
	case PayeeClassificationCompanyDriver, PayeeClassificationOwnerOperator:
		return true
	default:
		return false
	}
}

type ComponentKind string

const (
	ComponentKindLinehaul      = ComponentKind("Linehaul")
	ComponentKindFuelSurcharge = ComponentKind("FuelSurcharge")
	ComponentKindStopPay       = ComponentKind("StopPay")
	ComponentKindDetention     = ComponentKind("Detention")
	ComponentKindLayover       = ComponentKind("Layover")
	ComponentKindBreakdown     = ComponentKind("Breakdown")
	ComponentKindTarp          = ComponentKind("Tarp")
	ComponentKindHazmat        = ComponentKind("Hazmat")
	ComponentKindBonus         = ComponentKind("Bonus")
	ComponentKindCustom        = ComponentKind("Custom")
)

func (c ComponentKind) String() string { return string(c) }

func (c ComponentKind) IsValid() bool {
	switch c {
	case ComponentKindLinehaul, ComponentKindFuelSurcharge, ComponentKindStopPay,
		ComponentKindDetention, ComponentKindLayover, ComponentKindBreakdown,
		ComponentKindTarp, ComponentKindHazmat, ComponentKindBonus, ComponentKindCustom:
		return true
	default:
		return false
	}
}

type CalcMethod string

const (
	CalcMethodPerLoadedMile    = CalcMethod("PerLoadedMile")
	CalcMethodPerEmptyMile     = CalcMethod("PerEmptyMile")
	CalcMethodPerTotalMile     = CalcMethod("PerTotalMile")
	CalcMethodPercentOfRevenue = CalcMethod("PercentOfRevenue")
	CalcMethodFlatPerShipment  = CalcMethod("FlatPerShipment")
	CalcMethodPerStop          = CalcMethod("PerStop")
	CalcMethodPerHour          = CalcMethod("PerHour")
	CalcMethodPerDay           = CalcMethod("PerDay")
	CalcMethodPerEvent         = CalcMethod("PerEvent")
)

func (c CalcMethod) String() string { return string(c) }

func (c CalcMethod) IsValid() bool {
	switch c {
	case CalcMethodPerLoadedMile, CalcMethodPerEmptyMile, CalcMethodPerTotalMile,
		CalcMethodPercentOfRevenue, CalcMethodFlatPerShipment, CalcMethodPerStop,
		CalcMethodPerHour, CalcMethodPerDay, CalcMethodPerEvent:
		return true
	default:
		return false
	}
}

func (c CalcMethod) IsPerMile() bool {
	switch c { //nolint:exhaustive // only per-mile methods matter; default covers the rest
	case CalcMethodPerLoadedMile, CalcMethodPerEmptyMile, CalcMethodPerTotalMile:
		return true
	default:
		return false
	}
}

type RevenueBasis string

const (
	RevenueBasisLinehaul                  = RevenueBasis("Linehaul")
	RevenueBasisLinehaulPlusFuelSurcharge = RevenueBasis("LinehaulPlusFuelSurcharge")
	RevenueBasisTotalRevenue              = RevenueBasis("TotalRevenue")
)

func (r RevenueBasis) String() string { return string(r) }

func (r RevenueBasis) IsValid() bool {
	switch r {
	case RevenueBasisLinehaul, RevenueBasisLinehaulPlusFuelSurcharge, RevenueBasisTotalRevenue:
		return true
	default:
		return false
	}
}
