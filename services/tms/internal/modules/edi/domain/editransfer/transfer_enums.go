package editransfer

type TransferStatus string

const (
	TransferStatusSubmitted       = TransferStatus("Submitted")
	TransferStatusMappingRequired = TransferStatus("MappingRequired")
	TransferStatusPendingApproval = TransferStatus("PendingApproval")
	TransferStatusProcessing      = TransferStatus("Processing")
	TransferStatusApproved        = TransferStatus("Approved")
	TransferStatusRejected        = TransferStatus("Rejected")
	TransferStatusExpired         = TransferStatus("Expired")
	TransferStatusCanceled        = TransferStatus("Canceled")
	TransferStatusFailed          = TransferStatus("Failed")
)

func (s TransferStatus) IsFinal() bool {
	switch s {
	case TransferStatusSubmitted,
		TransferStatusMappingRequired,
		TransferStatusPendingApproval,
		TransferStatusProcessing:
		return false
	case TransferStatusApproved,
		TransferStatusRejected,
		TransferStatusExpired,
		TransferStatusCanceled,
		TransferStatusFailed:
		return true
	default:
		return false
	}
}

func (s TransferStatus) IsActionable() bool {
	return !s.IsFinal() && s != TransferStatusProcessing
}

type LoadTenderPurposeCode string

const (
	LoadTenderPurposeOriginal = LoadTenderPurposeCode("00")
	LoadTenderPurposeCancel   = LoadTenderPurposeCode("01")
	LoadTenderPurposeChange   = LoadTenderPurposeCode("04")
)

func (s TransferStatus) IsValid() bool {
	switch s {
	case TransferStatusSubmitted,
		TransferStatusMappingRequired,
		TransferStatusPendingApproval,
		TransferStatusProcessing,
		TransferStatusApproved,
		TransferStatusRejected,
		TransferStatusExpired,
		TransferStatusCanceled,
		TransferStatusFailed:
		return true
	default:
		return false
	}
}

func (c LoadTenderPurposeCode) IsValid() bool {
	switch c {
	case LoadTenderPurposeOriginal,
		LoadTenderPurposeCancel,
		LoadTenderPurposeChange:
		return true
	default:
		return false
	}
}
