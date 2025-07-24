/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
