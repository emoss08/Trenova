package documentuploadjobs

import (
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

type FinalizeUploadPayload struct {
	temporaltype.BasePayload

	SessionID pulid.ID `json:"sessionId"`
}

type FinalizeUploadResult struct {
	SessionID   pulid.ID  `json:"sessionId"`
	DocumentID  *pulid.ID `json:"documentId,omitempty"`
	Status      string    `json:"status"`
	PreviewPath string    `json:"previewPath,omitempty"`
}

type ReconcileUploadsPayload struct {
	temporaltype.BasePayload

	StaleAfterSeconds   int64 `json:"staleAfterSeconds"`
	PendingAfterSeconds int64 `json:"pendingAfterSeconds"`
	Limit               int   `json:"limit"`
}

type ReconcileUploadsResult struct {
	StaleSessionsProcessed int `json:"staleSessionsProcessed"`
	FinalizationsStarted   int `json:"finalizationsStarted"`
	SessionsExpired        int `json:"sessionsExpired"`
	PreviewRetriesStarted  int `json:"previewRetriesStarted"`
}

type CleanupDocumentStoragePayload struct {
	temporaltype.BasePayload

	DocumentID pulid.ID `json:"documentId"`
	Paths      []string `json:"paths"`
}
