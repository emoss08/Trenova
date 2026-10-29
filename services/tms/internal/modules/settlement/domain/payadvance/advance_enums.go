package payadvance

type AdvanceStatus string

const (
	AdvanceStatusOutstanding        = AdvanceStatus("Outstanding")
	AdvanceStatusPartiallyRecovered = AdvanceStatus("PartiallyRecovered")
	AdvanceStatusRecovered          = AdvanceStatus("Recovered")
	AdvanceStatusWrittenOff         = AdvanceStatus("WrittenOff")
)

func (a AdvanceStatus) String() string { return string(a) }

func (a AdvanceStatus) IsValid() bool {
	switch a {
	case AdvanceStatusOutstanding, AdvanceStatusPartiallyRecovered,
		AdvanceStatusRecovered, AdvanceStatusWrittenOff:
		return true
	default:
		return false
	}
}

type AdvanceSource string

const (
	AdvanceSourceCash         = AdvanceSource("Cash")
	AdvanceSourceEFSMoneyCode = AdvanceSource("EFSMoneyCode")
	AdvanceSourceComdataCode  = AdvanceSource("ComdataCode")
	AdvanceSourceFuelCard     = AdvanceSource("FuelCard")
	AdvanceSourceOther        = AdvanceSource("Other")
)

func (a AdvanceSource) String() string { return string(a) }

func (a AdvanceSource) IsValid() bool {
	switch a {
	case AdvanceSourceCash, AdvanceSourceEFSMoneyCode, AdvanceSourceComdataCode,
		AdvanceSourceFuelCard, AdvanceSourceOther:
		return true
	default:
		return false
	}
}
