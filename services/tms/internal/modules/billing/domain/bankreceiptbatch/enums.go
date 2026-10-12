package bankreceiptbatch

type Status string

const (
	StatusProcessing = Status("Processing")
	StatusCompleted  = Status("Completed")
)

func (s Status) String() string { return string(s) }
