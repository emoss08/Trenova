package task

const (
	TypeSendReport = "send:report"
	TypeNormalTask = "normal:task"
	TypeCleanup    = "cleanup:task"
)

type ReportPayload struct {
	ReportID int
}

type NormalTaskPayload struct {
	TaskID int
}

type CleanupPayload struct {
	TaskName string
}
