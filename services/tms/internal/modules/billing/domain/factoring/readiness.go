package factoring

import (
	"github.com/emoss08/trenova/shared/pulid"
)

type Check struct {
	Code             CheckCode `json:"code"`
	Passed           bool      `json:"passed"`
	Message          string    `json:"message"`
	Fix              FixAction `json:"fix,omitempty"`
	DocumentTypeID   pulid.ID  `json:"documentTypeId,omitempty"`
	DocumentTypeName string    `json:"documentTypeName,omitempty"`
	ShipmentID       pulid.ID  `json:"shipmentId,omitempty"`
	ShipmentPro      string    `json:"shipmentPro,omitempty"`
	Detail           string    `json:"detail,omitempty"`
}

type Readiness struct {
	Verdict Verdict `json:"verdict"`
	Checks  []Check `json:"checks"`
}

func (r *Readiness) FailedCount() int {
	failed := 0
	for idx := range r.Checks {
		if !r.Checks[idx].Passed {
			failed++
		}
	}
	return failed
}

func (r *Readiness) Failed() []Check {
	out := make([]Check, 0, len(r.Checks))
	for idx := range r.Checks {
		if !r.Checks[idx].Passed {
			out = append(out, r.Checks[idx])
		}
	}
	return out
}

type DocumentNeed struct {
	TypeID pulid.ID
	Name   string
}

type ShipmentValidation struct {
	Code    string
	Message string
}

type ShipmentFacts struct {
	ShipmentID         pulid.ID
	ProNumber          string
	Available          bool
	MissingDocuments   []DocumentNeed
	ValidationFailures []ShipmentValidation
	AcceptedTypeIDs    map[pulid.ID]struct{}
}

type InvoiceFacts struct {
	Posted             bool
	Voided             bool
	CreditMemo         bool
	Excluded           bool
	OpenBalanceMinor   int64
	FactoringCompanyID pulid.ID
}

type ReadinessInputs struct {
	At                     int64
	Invoice                InvoiceFacts
	Agreement              *Agreement
	Noa                    *Noa
	OpenDispute            bool
	Shipments              []ShipmentFacts
	InvoiceAcceptedTypeIDs map[pulid.ID]struct{}
	AgreementDocuments     []DocumentNeed
}

func EvaluateReadiness(in *ReadinessInputs) Readiness {
	if check, ok := eligibility(&in.Invoice); !ok {
		return Readiness{Verdict: VerdictExcluded, Checks: []Check{check}}
	}

	checks := make([]Check, 0, 6+len(in.Shipments)+len(in.AgreementDocuments))
	checks = append(checks,
		Check{Code: CheckEligible, Passed: true, Message: "Posted with an open balance"},
		Check{Code: CheckFactored, Passed: true, Message: "Assigned to a factor"},
		agreementCheck(in),
		noaCheck(in),
		disputeCheck(in),
	)
	checks = append(checks, shipmentChecks(in.Shipments)...)
	checks = append(checks, agreementDocumentChecks(in)...)

	readiness := Readiness{Verdict: VerdictReady, Checks: checks}
	if readiness.FailedCount() > 0 {
		readiness.Verdict = VerdictNotReady
	}
	return readiness
}

func eligibility(inv *InvoiceFacts) (Check, bool) {
	switch {
	case inv.CreditMemo:
		return Check{Code: CheckEligible, Message: "Credit memos are not factored"}, false
	case inv.Voided:
		return Check{Code: CheckEligible, Message: "The invoice is voided"}, false
	case !inv.Posted:
		return Check{Code: CheckEligible, Message: "The invoice is not posted yet"}, false
	case inv.OpenBalanceMinor <= 0:
		return Check{Code: CheckEligible, Message: "The invoice has no open balance"}, false
	case inv.Excluded:
		return Check{
			Code:    CheckEligible,
			Message: "The invoice is excluded from factoring and stays on normal receivables",
		}, false
	case inv.FactoringCompanyID.IsNil():
		return Check{Code: CheckFactored, Message: "The customer is not factored"}, false
	default:
		return Check{}, true
	}
}

func agreementCheck(in *ReadinessInputs) Check {
	if in.Agreement.InForce(in.At) {
		return Check{Code: CheckAgreement, Passed: true, Message: "An agreement is in force"}
	}
	return Check{
		Code:    CheckAgreement,
		Message: "The factor has no active agreement in force",
		Fix:     FixSetUpAgreement,
	}
}

func noaCheck(in *ReadinessInputs) Check {
	switch {
	case in.Noa.IsEffective(in.At):
		return Check{
			Code:    CheckNoa,
			Passed:  true,
			Message: "The customer acknowledged the notice of assignment",
		}
	case in.Noa == nil || in.Noa.Status == NoaStatusReleased:
		return Check{
			Code:    CheckNoa,
			Message: "No notice of assignment has been sent to the customer",
			Fix:     FixSendNoa,
		}
	case in.Noa.Status == NoaStatusAcknowledged:
		return Check{
			Code:    CheckNoa,
			Message: "The notice of assignment has expired",
			Fix:     FixSendNoa,
		}
	case in.Noa.Status == NoaStatusDraft:
		return Check{
			Code:    CheckNoa,
			Message: "The notice of assignment is drafted but not sent",
			Fix:     FixSendNoa,
		}
	default:
		return Check{
			Code:    CheckNoa,
			Message: "The customer has not returned the signed notice of assignment",
			Fix:     FixRecordNoaAcknowledgement,
		}
	}
}

func disputeCheck(in *ReadinessInputs) Check {
	if !in.OpenDispute {
		return Check{Code: CheckDispute, Passed: true, Message: "No open dispute"}
	}
	return Check{
		Code:    CheckDispute,
		Message: "The customer is disputing the invoice",
		Fix:     FixResolveDispute,
	}
}

func shipmentChecks(shipments []ShipmentFacts) []Check {
	checks := make([]Check, 0, len(shipments))
	for idx := range shipments {
		s := &shipments[idx]
		if !s.Available {
			checks = append(checks, Check{
				Code:        CheckShipmentUnavailable,
				Message:     "The shipment's billing readiness could not be read",
				Fix:         FixReviewShipment,
				ShipmentID:  s.ShipmentID,
				ShipmentPro: s.ProNumber,
			})
			continue
		}
		clean := true
		for _, need := range s.MissingDocuments {
			clean = false
			checks = append(checks, Check{
				Code:             CheckBillingDocument,
				Message:          "Missing " + need.Name,
				Fix:              FixRequestDocument,
				DocumentTypeID:   need.TypeID,
				DocumentTypeName: need.Name,
				ShipmentID:       s.ShipmentID,
				ShipmentPro:      s.ProNumber,
			})
		}
		for _, failure := range s.ValidationFailures {
			clean = false
			checks = append(checks, Check{
				Code:        CheckBillingValidation,
				Message:     failure.Message,
				Fix:         FixReviewShipment,
				ShipmentID:  s.ShipmentID,
				ShipmentPro: s.ProNumber,
				Detail:      failure.Code,
			})
		}
		if clean {
			checks = append(checks, Check{
				Code:        CheckBillingDocument,
				Passed:      true,
				Message:     "Billing documents and rates check out",
				ShipmentID:  s.ShipmentID,
				ShipmentPro: s.ProNumber,
			})
		}
	}
	return checks
}

func agreementDocumentChecks(in *ReadinessInputs) []Check {
	checks := make([]Check, 0, len(in.AgreementDocuments))
	for _, need := range in.AgreementDocuments {
		if hasAcceptedType(in, need.TypeID) {
			checks = append(checks, Check{
				Code:             CheckAgreementDocument,
				Passed:           true,
				Message:          need.Name + " is on file",
				DocumentTypeID:   need.TypeID,
				DocumentTypeName: need.Name,
			})
			continue
		}
		check := Check{
			Code:             CheckAgreementDocument,
			Message:          "The factor needs " + need.Name,
			Fix:              FixRequestDocument,
			DocumentTypeID:   need.TypeID,
			DocumentTypeName: need.Name,
		}
		if len(in.Shipments) == 1 {
			check.ShipmentID = in.Shipments[0].ShipmentID
			check.ShipmentPro = in.Shipments[0].ProNumber
		}
		checks = append(checks, check)
	}
	return checks
}

func hasAcceptedType(in *ReadinessInputs, typeID pulid.ID) bool {
	if _, ok := in.InvoiceAcceptedTypeIDs[typeID]; ok {
		return true
	}
	if len(in.Shipments) == 0 {
		return false
	}
	for idx := range in.Shipments {
		if _, ok := in.Shipments[idx].AcceptedTypeIDs[typeID]; !ok {
			return false
		}
	}
	return true
}
