package onboarding

type Status string

const (
	StatusPending   = Status("pending")
	StatusCompleted = Status("completed")
)

func AllStatuses() []Status {
	return []Status{StatusPending, StatusCompleted}
}

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusCompleted:
		return true
	default:
		return false
	}
}

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
