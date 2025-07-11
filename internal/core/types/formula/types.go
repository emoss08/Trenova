package formula

// * ContextType represents the type of formula context
type ContextType string

const (
	ContextTypeBuiltIn = ContextType("BUILT_IN")
	ContextTypeCustom  = ContextType("CUSTOM")
)

// * ValueType represents the data type of a formula value
type ValueType string

const (
	ValueTypeNumber  = ValueType("NUMBER")
	ValueTypeString  = ValueType("STRING")
	ValueTypeBoolean = ValueType("BOOLEAN")
	ValueTypeDate    = ValueType("DATE")
	ValueTypeArray   = ValueType("ARRAY")
	ValueTypeObject  = ValueType("OBJECT")
)

// * FormulaAction represents actions that can be performed on formulas
type FormulaAction string

const (
	FormulaActionCreate  = FormulaAction("CREATE")
	FormulaActionRead    = FormulaAction("READ")
	FormulaActionUpdate  = FormulaAction("UPDATE")
	FormulaActionDelete  = FormulaAction("DELETE")
	FormulaActionTest    = FormulaAction("TEST")
	FormulaActionApprove = FormulaAction("APPROVE")
)

// * AdjustmentType represents types of pricing adjustments
type AdjustmentType string

const (
	AdjustmentTypeDiscount   = AdjustmentType("DISCOUNT")
	AdjustmentTypeSurcharge  = AdjustmentType("SURCHARGE")
	AdjustmentTypeMultiplier = AdjustmentType("MULTIPLIER")
)

// * BuiltInContextName represents the names of built-in contexts
type BuiltInContextName string

const (
	BuiltInContextEquipmentType = BuiltInContextName("equipmentType")
	BuiltInContextHazmat        = BuiltInContextName("hazmat")
	BuiltInContextTemperature   = BuiltInContextName("temperature")
	BuiltInContextRoute         = BuiltInContextName("route")
	BuiltInContextTime          = BuiltInContextName("time")
	BuiltInContextShipment      = BuiltInContextName("shipment")
	BuiltInContextCustomer      = BuiltInContextName("customer")
)
