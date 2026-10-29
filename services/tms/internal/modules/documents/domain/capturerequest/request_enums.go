package capturerequest

import (
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
)

// RequestMode is what a person asked the companion to do from the web app.
type RequestMode string

const (
	RequestModeScan  = RequestMode("Scan")
	RequestModePrint = RequestMode("Print")
)

func (m RequestMode) IsValid() bool {
	switch m {
	case RequestModeScan, RequestModePrint:
		return true
	default:
		return false
	}
}

func AllRequestModes() []RequestMode {
	return []RequestMode{RequestModeScan, RequestModePrint}
}

// Source is the batch source a request of this mode produces.
func (m RequestMode) Source() capture.Source {
	if m == RequestModePrint {
		return capture.SourcePrint
	}

	return capture.SourceScan
}

// RequestStatus follows a request from the browser to the device and back.
type RequestStatus string

const (
	RequestPending    = RequestStatus("Pending")
	RequestDelivered  = RequestStatus("Delivered")
	RequestInProgress = RequestStatus("InProgress")
	RequestCompleted  = RequestStatus("Completed")
	RequestCanceled   = RequestStatus("Canceled")
	RequestExpired    = RequestStatus("Expired")
	RequestFailed     = RequestStatus("Failed")
)

func (s RequestStatus) IsValid() bool {
	switch s {
	case RequestPending,
		RequestDelivered,
		RequestInProgress,
		RequestCompleted,
		RequestCanceled,
		RequestExpired,
		RequestFailed:
		return true
	default:
		return false
	}
}

func (s RequestStatus) Terminal() bool {
	switch s {
	case RequestCompleted, RequestCanceled, RequestExpired, RequestFailed:
		return true
	case RequestPending, RequestDelivered, RequestInProgress:
		return false
	default:
		return false
	}
}

// CanMoveTo is the request lifecycle. A device reports progress, and a report
// that would move a request backwards, or out of a terminal state, is refused
// rather than applied, so a late or replayed report cannot reopen a request the
// person already cancelled.
func (s RequestStatus) CanMoveTo(next RequestStatus) bool {
	if s.Terminal() || s == next {
		return false
	}

	switch next {
	case RequestDelivered:
		return s == RequestPending
	case RequestInProgress:
		return s == RequestPending || s == RequestDelivered
	case RequestCompleted:
		return s == RequestDelivered || s == RequestInProgress
	case RequestCanceled, RequestExpired, RequestFailed:
		return true
	case RequestPending:
		return false
	default:
		return false
	}
}

func AllRequestStatuses() []RequestStatus {
	return []RequestStatus{
		RequestPending,
		RequestDelivered,
		RequestInProgress,
		RequestCompleted,
		RequestCanceled,
		RequestExpired,
		RequestFailed,
	}
}

// RequestFailureCode is why a device could not do what was asked. The device
// reports it; the browser turns it into words.
type RequestFailureCode string

const (
	FailureSourceUnavailable = RequestFailureCode("SOURCE_UNAVAILABLE")
	FailureSourceBusy        = RequestFailureCode("SOURCE_BUSY")
	FailurePaperJam          = RequestFailureCode("PAPER_JAM")
	FailureFeederEmpty       = RequestFailureCode("FEEDER_EMPTY")
	FailureCanceledByUser    = RequestFailureCode("CANCELED_BY_USER")
	FailureDriverError       = RequestFailureCode("DRIVER_ERROR")
	FailureUploadFailed      = RequestFailureCode("UPLOAD_FAILED")
	FailureNotDelivered      = RequestFailureCode("NOT_DELIVERED")
	FailureInternal          = RequestFailureCode("INTERNAL")
)

func (c RequestFailureCode) IsValid() bool {
	switch c {
	case FailureSourceUnavailable,
		FailureSourceBusy,
		FailurePaperJam,
		FailureFeederEmpty,
		FailureCanceledByUser,
		FailureDriverError,
		FailureUploadFailed,
		FailureNotDelivered,
		FailureInternal:
		return true
	default:
		return false
	}
}

func AllRequestFailureCodes() []RequestFailureCode {
	return []RequestFailureCode{
		FailureSourceUnavailable,
		FailureSourceBusy,
		FailurePaperJam,
		FailureFeederEmpty,
		FailureCanceledByUser,
		FailureDriverError,
		FailureUploadFailed,
		FailureNotDelivered,
		FailureInternal,
	}
}

// Lifetime is how long a request of this mode waits for its device.
func (m RequestMode) Lifetime() int64 {
	if m == RequestModePrint {
		return PrintRequestLifetimeSeconds
	}

	return ScanRequestLifetimeSeconds
}
