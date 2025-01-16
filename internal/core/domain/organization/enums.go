package organization

type Type string

const (
	// Brokerage is an organization that is a brokerage.
	TypeBrokerage = Type("Brokerage")

	// Carrier is an organization that is a carrier.
	TypeCarrier = Type("Carrier")

	// BrokerageCarrier is an organization that is both a brokerage and a carrier.
	TypeBrokerageCarrier = Type("BrokerageCarrier")
)
