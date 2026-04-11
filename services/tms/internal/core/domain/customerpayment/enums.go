package customerpayment

type Status string

const (
	StatusPosted   = Status("Posted")
	StatusReversed = Status("Reversed")
)

func (s Status) String() string { return string(s) }

type Method string

const (
	MethodACH   = Method("ACH")
	MethodCheck = Method("Check")
	MethodWire  = Method("Wire")
	MethodCard  = Method("Card")
	MethodCash  = Method("Cash")
	MethodOther = Method("Other")
)

func (m Method) String() string { return string(m) }

func (m Method) IsValid() bool {
	switch m {
	case MethodACH, MethodCheck, MethodWire, MethodCard, MethodCash, MethodOther:
		return true
	default:
		return false
	}
}
