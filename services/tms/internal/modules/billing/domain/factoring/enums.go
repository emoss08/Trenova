package factoring

type AgreementStatus string

const (
	AgreementStatusDraft  = AgreementStatus("Draft")
	AgreementStatusActive = AgreementStatus("Active")
	AgreementStatusEnded  = AgreementStatus("Ended")
)

func (s AgreementStatus) IsValid() bool {
	switch s {
	case AgreementStatusDraft, AgreementStatusActive, AgreementStatusEnded:
		return true
	default:
		return false
	}
}

type Recourse string

const (
	RecourseFull = Recourse("Recourse")
	RecourseNone = Recourse("NonRecourse")
)

func (r Recourse) IsValid() bool {
	switch r {
	case RecourseFull, RecourseNone:
		return true
	default:
		return false
	}
}

type AccountingTreatment string

const (
	AccountingTreatmentSale             = AccountingTreatment("Sale")
	AccountingTreatmentSecuredBorrowing = AccountingTreatment("SecuredBorrowing")
)

func (t AccountingTreatment) IsValid() bool {
	switch t {
	case AccountingTreatmentSale, AccountingTreatmentSecuredBorrowing:
		return true
	default:
		return false
	}
}

type NoaStatus string

const (
	NoaStatusDraft        = NoaStatus("Draft")
	NoaStatusSent         = NoaStatus("Sent")
	NoaStatusAcknowledged = NoaStatus("Acknowledged")
	NoaStatusReleased     = NoaStatus("Released")
)

func (s NoaStatus) IsValid() bool {
	switch s {
	case NoaStatusDraft, NoaStatusSent, NoaStatusAcknowledged, NoaStatusReleased:
		return true
	default:
		return false
	}
}

func AllNoaStatuses() []NoaStatus {
	return []NoaStatus{NoaStatusDraft, NoaStatusSent, NoaStatusAcknowledged, NoaStatusReleased}
}

type Verdict string

const (
	VerdictReady     = Verdict("Ready")
	VerdictNotReady  = Verdict("NotReady")
	VerdictSubmitted = Verdict("Submitted")
	VerdictFunded    = Verdict("Funded")
	VerdictExcluded  = Verdict("Excluded")
)

func (v Verdict) IsValid() bool {
	switch v {
	case VerdictReady, VerdictNotReady, VerdictSubmitted, VerdictFunded, VerdictExcluded:
		return true
	default:
		return false
	}
}

func QueueVerdicts() []Verdict {
	return []Verdict{VerdictReady, VerdictNotReady, VerdictSubmitted, VerdictFunded}
}

type CheckCode string

const (
	CheckEligible            = CheckCode("Eligible")
	CheckFactored            = CheckCode("Factored")
	CheckAgreement           = CheckCode("Agreement")
	CheckNoa                 = CheckCode("Noa")
	CheckDispute             = CheckCode("Dispute")
	CheckBillingDocument     = CheckCode("BillingDocument")
	CheckBillingValidation   = CheckCode("BillingValidation")
	CheckAgreementDocument   = CheckCode("AgreementDocument")
	CheckShipmentUnavailable = CheckCode("ShipmentUnavailable")
	CheckChargedBack         = CheckCode("ChargedBack")
)

func (c CheckCode) IsValid() bool {
	switch c {
	case CheckEligible, CheckFactored, CheckAgreement, CheckNoa, CheckDispute,
		CheckBillingDocument, CheckBillingValidation, CheckAgreementDocument,
		CheckShipmentUnavailable, CheckChargedBack:
		return true
	default:
		return false
	}
}

type FixAction string

const (
	FixNone                     = FixAction("")
	FixSetUpAgreement           = FixAction("SetUpAgreement")
	FixSendNoa                  = FixAction("SendNoa")
	FixRecordNoaAcknowledgement = FixAction("RecordNoaAcknowledgement")
	FixResolveDispute           = FixAction("ResolveDispute")
	FixRequestDocument          = FixAction("RequestDocument")
	FixReviewShipment           = FixAction("ReviewShipment")
)

func (a FixAction) IsValid() bool {
	switch a {
	case FixNone, FixSetUpAgreement, FixSendNoa, FixRecordNoaAcknowledgement,
		FixResolveDispute, FixRequestDocument, FixReviewShipment:
		return true
	default:
		return false
	}
}

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

type FundingOutcome string

const (
	FundingOutcomeFunded   = FundingOutcome("Funded")
	FundingOutcomeRejected = FundingOutcome("Rejected")
)

func (o FundingOutcome) IsValid() bool {
	return o == FundingOutcomeFunded || o == FundingOutcomeRejected
}

type ReserveEntryKind string

const (
	ReserveEntryFunding            = ReserveEntryKind("Funding")
	ReserveEntryRelease            = ReserveEntryKind("ReserveRelease")
	ReserveEntryChargeback         = ReserveEntryKind("Chargeback")
	ReserveEntryChargebackReversal = ReserveEntryKind("ChargebackReversal")
)

func (k ReserveEntryKind) IsValid() bool {
	switch k {
	case ReserveEntryFunding, ReserveEntryRelease, ReserveEntryChargeback,
		ReserveEntryChargebackReversal:
		return true
	default:
		return false
	}
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
