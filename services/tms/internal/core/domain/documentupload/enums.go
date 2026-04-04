package documentupload

type FailureCode string

const (
	FailureCodeSuspendedByNewerSession  = FailureCode("SUPERSEDED_BY_NEWER_SESSION")
	FailureCodeUploadFinalizationFailed = FailureCode("UPLOAD_FINALIZATION_FAILED")
	FailureMultipartUploadMissing       = FailureCode("MULTIPART_UPLOAD_MISSING")
	FailureSessionExpired               = FailureCode("SESSION_EXPIRED")
	FailureDocumentCreateFailed         = FailureCode("DOCUMENT_CREATE_FAILED")
	FailureNoUploadParts                = FailureCode("NO_UPLOAD_PARTS")
	FailureMultipartCompleteFailed      = FailureCode("MULTIPART_COMPLETE_FAILED")
	FailureFileInfoFailed               = FailureCode("FILE_INFO_FAILED")
	FailureFileSizeMismatch             = FailureCode("FILE_SIZE_MISMATCH")
	FailureDownloadFailed               = FailureCode("DOWNLOAD_FAILED")
	FailureReadFailed                   = FailureCode("READ_FAILED")
	FailureExtractionFailed             = FailureCode("EXTRACTION_FAILED")
)

func (f FailureCode) String() string {
	return string(f)
}

type Status string

const (
	StatusInitiated   = Status("Initiated")
	StatusUploading   = Status("Uploading")
	StatusUploaded    = Status("Uploaded")
	StatusVerifying   = Status("Verifying")
	StatusFinalizing  = Status("Finalizing")
	StatusPaused      = Status("Paused")
	StatusCompleting  = Status("Completing")
	StatusCompleted   = Status("Completed")
	StatusAvailable   = Status("Available")
	StatusQuarantined = Status("Quarantined")
	StatusFailed      = Status("Failed")
	StatusCanceled    = Status("Canceled")
	StatusExpired     = Status("Expired")
)

func (s Status) String() string {
	return string(s)
}

type Strategy string

const (
	StrategySingle    = Strategy("single")
	StrategyMultipart = Strategy("multipart")
)

func (s Strategy) String() string {
	return string(s)
}
