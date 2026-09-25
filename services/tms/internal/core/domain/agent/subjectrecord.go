package agent

import (
	"fmt"

	"github.com/emoss08/trenova/shared/pulid"
)

type subjectRecord struct {
	prefix  string
	noun    string
	article string
}

var subjectRecords = map[SubjectType]subjectRecord{
	SubjectBillingQueueItem:     {"bqi_", "billing queue item", "a"},
	SubjectShipmentMove:         {"sm_", "shipment move", "a"},
	SubjectAssistantThread:      {"athr_", "conversation", "a"},
	SubjectShipment:             {"shp_", "shipment", "a"},
	SubjectDocument:             {"doc_", "document", "a"},
	SubjectOrganization:         {"org_", "organization", "an"},
	SubjectInsight:              {"inst_", "insight", "an"},
	SubjectBankReceipt:          {"brcpt_", "bank receipt", "a"},
	SubjectDetentionOccurrence:  {"dto_", "detention occurrence", "a"},
	SubjectWorker:               {"wrk_", "worker", "a"},
	SubjectCarrierIntelEvent:    {"cievt_", "carrier intelligence event", "a"},
	SubjectEDIInboundFile:       {"ediinf_", "EDI inbound file", "an"},
	SubjectInboundMessage:       {"imsg_", "inbound message", "an"},
	SubjectReport:               {"rd_", "report", "a"},
	SubjectDashboard:            {"rdb_", "dashboard", "a"},
	SubjectAccountingConnection: {"acctc_", "accounting connection", "an"},
	SubjectAccountingSyncRecord: {"acctsr_", "accounting sync record", "an"},
	SubjectFormulaTemplate:      {"ft_", "formula template", "a"},
}

var subjectsByPrefix = func() map[string]SubjectType {
	out := make(map[string]SubjectType, len(subjectRecords))
	for subject, record := range subjectRecords {
		out[record.prefix] = subject
	}

	return out
}()

// IDPrefix is the prefix every id of this kind of record carries, so an id
// can be told apart from another kind's before anything is read.
func (s SubjectType) IDPrefix() string {
	return subjectRecords[s].prefix
}

// Noun is what one record of this kind is called in a sentence: "report",
// "EDI inbound file".
func (s SubjectType) Noun() string {
	if record, ok := subjectRecords[s]; ok {
		return record.noun
	}

	return string(s)
}

// NounWithArticle is Noun as a sentence introduces it: "a report", "an
// insight".
func (s SubjectType) NounWithArticle() string {
	if record, ok := subjectRecords[s]; ok {
		return record.article + " " + record.noun
	}

	return "a " + string(s)
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
