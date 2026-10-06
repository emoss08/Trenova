package tenant

type OperationType string

const (
	OperationTypeAsset     = OperationType("asset")
	OperationTypeBrokerage = OperationType("brokerage")
	OperationTypeBoth      = OperationType("both")
)

func AllOperationTypes() []OperationType {
	return []OperationType{OperationTypeAsset, OperationTypeBrokerage, OperationTypeBoth}
}

func (o OperationType) String() string {
	return string(o)
}

func (o OperationType) IsValid() bool {
	switch o {
	case OperationTypeAsset, OperationTypeBrokerage, OperationTypeBoth:
		return true
	default:
		return false
	}
}

func (o OperationType) RunsAssets() bool {
	return o == OperationTypeAsset || o == OperationTypeBoth
}

func (o OperationType) RunsBrokerage() bool {
	return o == OperationTypeBrokerage || o == OperationTypeBoth
}

func OperationTypeOf(brokerageEnabled, assetOperationsEnabled bool) OperationType {
	switch {
	case brokerageEnabled && assetOperationsEnabled:
		return OperationTypeBoth
	case brokerageEnabled:
		return OperationTypeBrokerage
	default:
		return OperationTypeAsset
	}
}

func (o OperationType) Capabilities() (brokerageEnabled, assetOperationsEnabled bool) {
	return o.RunsBrokerage(), o.RunsAssets()
}

func (o OperationType) CoverageNoun() string {
	switch o {
	case OperationTypeAsset:
		return "a driver"
	case OperationTypeBrokerage:
		return "a carrier"
	case OperationTypeBoth:
		return "coverage"
	default:
		return "coverage"
	}
}
