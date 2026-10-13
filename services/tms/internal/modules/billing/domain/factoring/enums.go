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

type Verdict string

const (
	VerdictReady    = Verdict("Ready")
	VerdictNotReady = Verdict("NotReady")
	VerdictExcluded = Verdict("Excluded")
)

func (v Verdict) IsValid() bool {
	switch v {
	case VerdictReady, VerdictNotReady, VerdictExcluded:
		return true
	default:
		return false
	}
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
)

func (c CheckCode) IsValid() bool {
	switch c {
	case CheckEligible, CheckFactored, CheckAgreement, CheckNoa, CheckDispute,
		CheckBillingDocument, CheckBillingValidation, CheckAgreementDocument,
		CheckShipmentUnavailable:
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
