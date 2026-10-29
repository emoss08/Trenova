package factoringsubmission

type SubmissionStatus string

const (
	SubmissionStatusSending   = SubmissionStatus("Sending")
	SubmissionStatusSent      = SubmissionStatus("Sent")
	SubmissionStatusFailed    = SubmissionStatus("Failed")
	SubmissionStatusCancelled = SubmissionStatus("Cancelled")
	SubmissionStatusPartially = SubmissionStatus("PartiallyFunded")
	SubmissionStatusFunded    = SubmissionStatus("Funded")
	SubmissionStatusRejected  = SubmissionStatus("Rejected")
)

func (s SubmissionStatus) IsValid() bool {
	switch s {
	case SubmissionStatusSending, SubmissionStatusSent, SubmissionStatusFailed,
		SubmissionStatusCancelled, SubmissionStatusPartially, SubmissionStatusFunded,
		SubmissionStatusRejected:
		return true
	default:
		return false
	}
}

func (s SubmissionStatus) AwaitsFunding() bool {
	return s == SubmissionStatusSent || s == SubmissionStatusPartially
}

func AwaitingFundingStatuses() []SubmissionStatus {
	return []SubmissionStatus{SubmissionStatusSent, SubmissionStatusPartially}
}

func (s SubmissionStatus) IsLive() bool {
	return s == SubmissionStatusSending || s == SubmissionStatusSent ||
		s == SubmissionStatusFailed
}

func AllSubmissionStatuses() []SubmissionStatus {
	return []SubmissionStatus{
		SubmissionStatusSending,
		SubmissionStatusSent,
		SubmissionStatusFailed,
		SubmissionStatusCancelled,
		SubmissionStatusPartially,
		SubmissionStatusFunded,
		SubmissionStatusRejected,
	}
}

type SubmissionItemStatus string

const (
	SubmissionItemStatusPending     = SubmissionItemStatus("Pending")
	SubmissionItemStatusReleased    = SubmissionItemStatus("Released")
	SubmissionItemStatusFunded      = SubmissionItemStatus("Funded")
	SubmissionItemStatusRejected    = SubmissionItemStatus("Rejected")
	SubmissionItemStatusSettled     = SubmissionItemStatus("Settled")
	SubmissionItemStatusChargedBack = SubmissionItemStatus("ChargedBack")
)

func (s SubmissionItemStatus) IsValid() bool {
	switch s {
	case SubmissionItemStatusPending, SubmissionItemStatusReleased, SubmissionItemStatusFunded,
		SubmissionItemStatusRejected, SubmissionItemStatusSettled,
		SubmissionItemStatusChargedBack:
		return true
	default:
		return false
	}
}

func (s SubmissionItemStatus) IsLive() bool {
	return s == SubmissionItemStatusPending || s == SubmissionItemStatusFunded
}

func LiveSubmissionItemStatuses() []SubmissionItemStatus {
	return []SubmissionItemStatus{SubmissionItemStatusPending, SubmissionItemStatusFunded}
}

type TransportMethod string

const (
	TransportMethodEmail = TransportMethod("Email")
	TransportMethodSftp  = TransportMethod("Sftp")
)

func (m TransportMethod) IsValid() bool {
	switch m {
	case TransportMethodEmail, TransportMethodSftp:
		return true
	default:
		return false
	}
}

func (m TransportMethod) String() string {
	return string(m)
}

type SftpAuthMethod string

const (
	SftpAuthPassword   = SftpAuthMethod("Password")
	SftpAuthPrivateKey = SftpAuthMethod("PrivateKey")
)

func (m SftpAuthMethod) IsValid() bool {
	switch m {
	case SftpAuthPassword, SftpAuthPrivateKey:
		return true
	default:
		return false
	}
}
