// Package capturejobs reads captured batches and files what they become.
//
// Reading is a workflow rather than part of the seal request because rendering
// a two-hundred-page stack takes longer than a device should wait on an HTTP
// call, and a worker that dies halfway must be able to start again without
// the device sending anything twice.
package capturejobs

import (
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/shared/pulid"
)

// The payloads are the service's own, re-exported so the workflows read
// naturally without the two packages importing each other.

type ProcessBatchPayload = captureservice.ProcessBatchPayload

type ProcessResult = captureservice.ProcessResult

type FileItemPayload = captureservice.FileItemPayload

type MaintenanceResult = captureservice.MaintenanceResult

// AutoFilePayload files one item the processing run found needs no person.
type AutoFilePayload struct {
	ProcessBatchPayload
	ItemID pulid.ID `json:"itemId"`
}

// RecordFiledPayload reports the document a filing produced.
type RecordFiledPayload struct {
	FileItemPayload
	DocumentID pulid.ID `json:"documentId"`
}

// RecordFailedPayload reports that a filing could not finish.
type RecordFailedPayload struct {
	FileItemPayload
	Message string `json:"message"`
}
