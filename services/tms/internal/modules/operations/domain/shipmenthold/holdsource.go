package shipmenthold

type HoldSource string

const (
	HoldSourceUser = HoldSource("User")
	HoldSourceRule = HoldSource("Rule")
	HoldSourceAPI  = HoldSource("API")
	HoldSourceELD  = HoldSource("ELD")
	HoldSourceEDI  = HoldSource("EDI")
)
