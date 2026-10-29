package agent

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
)

// subjectRecordKinds is the kind of record each subject is, as the permission
// registry names it; its prefix and its noun are read from there. The client
// is generated from this table (client/packages/graphql/scripts/
// generate-record-kinds.mjs), so it stays a map of plain literals.
var subjectRecordKinds = map[SubjectType]recordkind.Kind{
	SubjectBillingQueueItem:     "billing_queue_item",
	SubjectShipmentMove:         "shipment_move",
	SubjectAssistantThread:      "assistant_thread",
	SubjectShipment:             "shipment",
	SubjectDocument:             "document",
	SubjectOrganization:         "organization",
	SubjectInsight:              "insight",
	SubjectBankReceipt:          "bank_receipt",
	SubjectDetentionOccurrence:  "detention_occurrence",
	SubjectWorker:               "worker",
	SubjectCarrierIntelEvent:    "carrier_intelligence_event",
	SubjectEDIInboundFile:       "edi_inbound_file",
	SubjectInboundMessage:       "inbound_message",
	SubjectReport:               "report",
	SubjectDashboard:            "dashboard",
	SubjectAccountingConnection: "accounting_connection",
	SubjectAccountingSyncRecord: "accounting_sync_record",
	SubjectAccountingInbound:    "accounting_inbound_change",
	SubjectAccountingDrift:      "accounting_drift_finding",
	SubjectFormulaTemplate:      "formula_template",
	SubjectFormulaLibrary:       "formula_library",
	SubjectInvoice:              "invoice",
	SubjectInvoiceDispute:       "invoice_dispute",
	SubjectFactoringNoa:         "factoring_noa",
	SubjectFactoringSubmission:  "factoring_submission",
	SubjectFactoringRemittance:  "factoring_remittance",
}

var subjectsByPrefix = func() map[string]SubjectType {
	out := make(map[string]SubjectType, len(subjectRecordKinds))
	for subject, kind := range subjectRecordKinds {
		if prefix, ok := kind.IDPrefix(); ok {
			out[prefix] = subject
		}
	}

	return out
}()

// RecordKind is the kind of record this subject is.
func (s SubjectType) RecordKind() (recordkind.Kind, bool) {
	kind, ok := subjectRecordKinds[s]

	return kind, ok
}

// IDPrefix is the prefix every id of this kind of record carries, so an id
// can be told apart from another kind's before anything is read.
func (s SubjectType) IDPrefix() string {
	prefix, _ := subjectRecordKinds[s].IDPrefix()

	return prefix
}

// Noun is what one record of this kind is called in a sentence: "report",
// "EDI inbound file".
func (s SubjectType) Noun() string {
	if kind, ok := subjectRecordKinds[s]; ok {
		return kind.Noun()
	}

	return string(s)
}

// NounWithArticle is Noun as a sentence introduces it: "a report", "an
// insight".
func (s SubjectType) NounWithArticle() string {
	return inflect.WithArticle(s.Noun())
}

// SubjectTypeOfID is the kind of record an id's prefix belongs to.
func SubjectTypeOfID(id pulid.ID) (SubjectType, bool) {
	subject, ok := subjectsByPrefix[id.Prefix()]

	return subject, ok
}

// CheckID reports whether id names a record of this kind and, when it does
// not, says what the id is and which kind to use instead. A report filed as
// an insight used to be recorded as given and pointed at nothing.
func (s SubjectType) CheckID(id pulid.ID) error {
	want := s.IDPrefix()
	if want == "" {
		return fmt.Errorf("subject type %q is not a kind of record a case can be about", s)
	}
	if id.Prefix() == want {
		return nil
	}

	if actual, ok := SubjectTypeOfID(id); ok {
		return fmt.Errorf(
			"%s is %s, not %s; use subjectType %s",
			id, actual.NounWithArticle(), s.NounWithArticle(), actual,
		)
	}

	return fmt.Errorf(
		"%s is not the id of %s: %s ids start with %q",
		id, s.NounWithArticle(), s.Noun(), want,
	)
}
