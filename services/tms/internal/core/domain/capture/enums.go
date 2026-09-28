package capture

// DeviceStatus is whether a paired companion may still act for its user.
type DeviceStatus string

const (
	DeviceActive  = DeviceStatus("Active")
	DeviceRevoked = DeviceStatus("Revoked")
)

func (s DeviceStatus) IsValid() bool {
	switch s {
	case DeviceActive, DeviceRevoked:
		return true
	default:
		return false
	}
}

func AllDeviceStatuses() []DeviceStatus {
	return []DeviceStatus{DeviceActive, DeviceRevoked}
}

// Architecture is the processor the companion was built for. It decides which
// update package a device is offered.
//
// Only x64 exists. Scanner vendors do not ship TWAIN drivers for Windows on
// ARM, and Kofax VRS does not run there, so an ARM build would pair and then
// find nothing to scan with.
type Architecture string

const ArchitectureX64 = Architecture("x64")

func (a Architecture) IsValid() bool {
	return a == ArchitectureX64
}

func AllArchitectures() []Architecture {
	return []Architecture{ArchitectureX64}
}

// PairingStatus is where a device authorization grant is.
type PairingStatus string

const (
	PairingPending  = PairingStatus("Pending")
	PairingApproved = PairingStatus("Approved")
	PairingDenied   = PairingStatus("Denied")
	// PairingConsumed is an approved grant whose tokens were collected. A grant
	// is exchanged exactly once; a second poll with the same device code is
	// somebody replaying it.
	PairingConsumed = PairingStatus("Consumed")
	PairingExpired  = PairingStatus("Expired")
)

func (s PairingStatus) IsValid() bool {
	switch s {
	case PairingPending, PairingApproved, PairingDenied, PairingConsumed, PairingExpired:
		return true
	default:
		return false
	}
}

func (s PairingStatus) Terminal() bool {
	switch s {
	case PairingDenied, PairingConsumed, PairingExpired:
		return true
	case PairingPending, PairingApproved:
		return false
	default:
		return false
	}
}

func AllPairingStatuses() []PairingStatus {
	return []PairingStatus{
		PairingPending,
		PairingApproved,
		PairingDenied,
		PairingConsumed,
		PairingExpired,
	}
}

// Source is how the pages of a batch were acquired.
type Source string

const (
	SourceScan  = Source("Scan")
	SourcePrint = Source("Print")
)

func (s Source) IsValid() bool {
	switch s {
	case SourceScan, SourcePrint:
		return true
	default:
		return false
	}
}

func AllSources() []Source {
	return []Source{SourceScan, SourcePrint}
}

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
func (m RequestMode) Source() Source {
	if m == RequestModePrint {
		return SourcePrint
	}

	return SourceScan
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

// BatchStatus follows one acquisition from its first page to the last item
// filed.
type BatchStatus string

const (
	// BatchReceiving is open: pages are still arriving.
	BatchReceiving = BatchStatus("Receiving")
	// BatchSealed has every page it said it would and is waiting to be read.
	BatchSealed     = BatchStatus("Sealed")
	BatchProcessing = BatchStatus("Processing")
	// BatchReady has been split into items and is waiting on a person.
	BatchReady          = BatchStatus("Ready")
	BatchPartiallyFiled = BatchStatus("PartiallyFiled")
	BatchFiled          = BatchStatus("Filed")
	BatchDiscarded      = BatchStatus("Discarded")
	BatchExpired        = BatchStatus("Expired")
	BatchFailed         = BatchStatus("Failed")
)

func (s BatchStatus) IsValid() bool {
	switch s {
	case BatchReceiving,
		BatchSealed,
		BatchProcessing,
		BatchReady,
		BatchPartiallyFiled,
		BatchFiled,
		BatchDiscarded,
		BatchExpired,
		BatchFailed:
		return true
	default:
		return false
	}
}

// Terminal reports whether a batch is finished with. A failed batch is not:
// its pages are still there, and a person can still split and file them.
func (s BatchStatus) Terminal() bool {
	switch s {
	case BatchFiled, BatchDiscarded, BatchExpired:
		return true
	case BatchReceiving,
		BatchSealed,
		BatchProcessing,
		BatchReady,
		BatchPartiallyFiled,
		BatchFailed:
		return false
	default:
		return false
	}
}

// Editable reports whether a person may change how the batch is split.
func (s BatchStatus) Editable() bool {
	switch s {
	case BatchReady, BatchPartiallyFiled, BatchFailed:
		return true
	case BatchReceiving,
		BatchSealed,
		BatchProcessing,
		BatchFiled,
		BatchDiscarded,
		BatchExpired:
		return false
	default:
		return false
	}
}

func AllBatchStatuses() []BatchStatus {
	return []BatchStatus{
		BatchReceiving,
		BatchSealed,
		BatchProcessing,
		BatchReady,
		BatchPartiallyFiled,
		BatchFiled,
		BatchDiscarded,
		BatchExpired,
		BatchFailed,
	}
}

// PageStatus is whether the server has read a page yet.
type PageStatus string

const (
	PageReceived  = PageStatus("Received")
	PageProcessed = PageStatus("Processed")
	PageFailed    = PageStatus("Failed")
)

func (s PageStatus) IsValid() bool {
	switch s {
	case PageReceived, PageProcessed, PageFailed:
		return true
	default:
		return false
	}
}

func AllPageStatuses() []PageStatus {
	return []PageStatus{PageReceived, PageProcessed, PageFailed}
}

// ItemStatus follows one proposed document to the document it became.
type ItemStatus string

const (
	ItemProposed  = ItemStatus("Proposed")
	ItemFiling    = ItemStatus("Filing")
	ItemFiled     = ItemStatus("Filed")
	ItemDiscarded = ItemStatus("Discarded")
	ItemFailed    = ItemStatus("Failed")
)

func (s ItemStatus) IsValid() bool {
	switch s {
	case ItemProposed, ItemFiling, ItemFiled, ItemDiscarded, ItemFailed:
		return true
	default:
		return false
	}
}

// Open reports whether an item still holds its pages for a person to act on.
func (s ItemStatus) Open() bool {
	return s == ItemProposed || s == ItemFailed
}

func AllItemStatuses() []ItemStatus {
	return []ItemStatus{ItemProposed, ItemFiling, ItemFiled, ItemDiscarded, ItemFailed}
}

// SuggestionSource is where an item's proposed destination came from. It is
// shown next to the suggestion, because a destination read off a cover sheet
// and one guessed from the text deserve different amounts of trust.
type SuggestionSource string

const (
	SuggestionCoverSheet = SuggestionSource("CoverSheet")
	SuggestionRequest    = SuggestionSource("Request")
	SuggestionClassifier = SuggestionSource("Classifier")
	SuggestionPerson     = SuggestionSource("Person")
)

func (s SuggestionSource) IsValid() bool {
	switch s {
	case SuggestionCoverSheet, SuggestionRequest, SuggestionClassifier, SuggestionPerson:
		return true
	default:
		return false
	}
}

func AllSuggestionSources() []SuggestionSource {
	return []SuggestionSource{
		SuggestionCoverSheet,
		SuggestionRequest,
		SuggestionClassifier,
		SuggestionPerson,
	}
}

// PixelType is the colour depth a profile asks a scanner for.
type PixelType string

const (
	PixelBlackWhite = PixelType("BlackWhite")
	PixelGrayscale  = PixelType("Grayscale")
	PixelColor      = PixelType("Color")
)

func (p PixelType) IsValid() bool {
	switch p {
	case PixelBlackWhite, PixelGrayscale, PixelColor:
		return true
	default:
		return false
	}
}

func AllPixelTypes() []PixelType {
	return []PixelType{PixelBlackWhite, PixelGrayscale, PixelColor}
}

// SeparatorStrategy is one way a stack of paper says where a document ends.
// A profile may combine them; any one of them firing splits the stack.
type SeparatorStrategy string

const (
	// SeparatorPatchCode splits on a patch-code sheet the scanner detected.
	SeparatorPatchCode = SeparatorStrategy("PatchCode")
	// SeparatorCoverSheet splits on a Trenova cover sheet and routes what
	// follows it to the record the sheet names.
	SeparatorCoverSheet = SeparatorStrategy("CoverSheet")
	// SeparatorBlankPage splits on a page with no content on it.
	SeparatorBlankPage = SeparatorStrategy("BlankPage")
	// SeparatorFixedPageCount splits every N pages.
	SeparatorFixedPageCount = SeparatorStrategy("FixedPageCount")
)

func (s SeparatorStrategy) IsValid() bool {
	switch s {
	case SeparatorPatchCode, SeparatorCoverSheet, SeparatorBlankPage, SeparatorFixedPageCount:
		return true
	default:
		return false
	}
}

func AllSeparatorStrategies() []SeparatorStrategy {
	return []SeparatorStrategy{
		SeparatorPatchCode,
		SeparatorCoverSheet,
		SeparatorBlankPage,
		SeparatorFixedPageCount,
	}
}

// ProfileStatus is whether a profile is offered to people who scan.
type ProfileStatus string

const (
	ProfileActive   = ProfileStatus("Active")
	ProfileInactive = ProfileStatus("Inactive")
)

func (s ProfileStatus) IsValid() bool {
	switch s {
	case ProfileActive, ProfileInactive:
		return true
	default:
		return false
	}
}

func AllProfileStatuses() []ProfileStatus {
	return []ProfileStatus{ProfileActive, ProfileInactive}
}

// SourceProtocol is how the companion talks to a scanner.
type SourceProtocol string

const (
	ProtocolTWAIN = SourceProtocol("TWAIN")
	ProtocolWIA   = SourceProtocol("WIA")
)

func (p SourceProtocol) IsValid() bool {
	switch p {
	case ProtocolTWAIN, ProtocolWIA:
		return true
	default:
		return false
	}
}
