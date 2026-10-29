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
