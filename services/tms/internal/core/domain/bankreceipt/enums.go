package bankreceipt

type Status string

const (
	StatusImported  = Status("Imported")
	StatusMatched   = Status("Matched")
	StatusException = Status("Exception")
)

func (s Status) String() string { return string(s) }
