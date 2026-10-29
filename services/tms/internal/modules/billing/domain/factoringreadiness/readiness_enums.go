package factoringreadiness

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
