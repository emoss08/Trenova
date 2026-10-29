package accountingsyncpush

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/pulid"
)

type WaitError struct {
	Reason    *accountingsync.SyncError
	Delay     time.Duration
	DependsOn pulid.ID
}

func (w *WaitError) Error() string { return w.Reason.Message }

func WaitFor(dependsOn pulid.ID, message string) *WaitError {
	return &WaitError{
		Reason: &accountingsync.SyncError{
			Category: accountingsync.SyncErrorMapping,
			Message:  message,
		},
		Delay:     dependencyWait,
		DependsOn: dependsOn,
	}
}

type NoopError struct {
	Reason string
}

func (n *NoopError) Error() string { return n.Reason }

func Blocked(
	category accountingsync.SyncErrorCategory,
	message, resolution string,
) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category:   category,
		Message:    message,
		Resolution: resolution,
	}
}

func WaitingOn(dependency *accountingsync.AccountingSyncRecord, label string) error {
	if dependency.Status.NeedsAttention() {
		message := label + " has not reached the books"
		if dependency.ErrorMessage != "" {
			message += ": " + dependency.ErrorMessage
		}
		wait := WaitFor(dependency.ID, message)
		wait.Reason.Resolution = "Fix " + label + " first; this record follows it"
		wait.Delay = accountingsync.SyncAuthWait
		return wait
	}
	return WaitFor(dependency.ID, "Waiting for "+label+" to reach the books")
}

func SharedDocumentError(sess *Session, label string) *accountingsync.SyncError {
	return Blocked(
		accountingsync.SyncErrorConflict,
		label+" was recorded in "+sess.ProviderName+
			" as part of one payment with other documents, so Trenova cannot change it alone",
		"Edit that payment in "+sess.ProviderName+", then skip this record",
	)
}
