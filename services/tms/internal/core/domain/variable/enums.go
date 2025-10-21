package variable

import "fmt"

type Context string

const (
	ContextInvoice      Context = "Invoice"
	ContextCustomer     Context = "Customer"
	ContextShipment     Context = "Shipment"
	ContextOrganization Context = "Organization"
	ContextSystem       Context = "System"
)

func (v Context) String() string {
	return string(v)
}

func (v Context) IsValid() bool {
	switch v {
	case ContextInvoice,
		ContextCustomer,
		ContextShipment,
		ContextOrganization,
		ContextSystem:
		return true
	}
	return false
}

func ParseContext(s string) (Context, error) {
	ctx := Context(s)
	if !ctx.IsValid() {
		return "", fmt.Errorf("invalid context: %s", s)
	}
	return ctx, nil
}

type ValueType string

const (
	ValueTypeString   ValueType = "String"
	ValueTypeNumber   ValueType = "Number"
	ValueTypeDate     ValueType = "Date"
	ValueTypeBoolean  ValueType = "Boolean"
	ValueTypeCurrency ValueType = "Currency"
)

func (v ValueType) String() string {
	return string(v)
}

func (v ValueType) IsValid() bool {
	switch v {
	case ValueTypeString,
		ValueTypeNumber,
		ValueTypeDate,
		ValueTypeBoolean,
		ValueTypeCurrency:
		return true
	}
	return false
}
