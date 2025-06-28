package consolidation

type GroupStatus string

const (
	GroupStatusNew        = GroupStatus("New")
	GroupStatusInProgress = GroupStatus("InProgress")
	GroupStatusCompleted  = GroupStatus("Completed")
	GroupStatusCanceled   = GroupStatus("Canceled")
)
