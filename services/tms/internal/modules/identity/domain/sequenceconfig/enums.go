package sequenceconfig

type SequenceType string

const (
	SequenceTypeProNumber            = SequenceType("pro_number")
	SequenceTypeConsolidation        = SequenceType("consolidation")
	SequenceTypeOrder                = SequenceType("order")
	SequenceTypeInvoice              = SequenceType("invoice")
	SequenceTypeCreditMemo           = SequenceType("credit_memo")
	SequenceTypeDebitMemo            = SequenceType("debit_memo")
	SequenceTypeWorkOrder            = SequenceType("work_order")
	SequenceTypeJournalBatch         = SequenceType("journal_batch")
	SequenceTypeJournalEntry         = SequenceType("journal_entry")
	SequenceTypeManualJournalRequest = SequenceType("manual_journal_request")
	SequenceTypeLocationCode         = SequenceType("location_code")
	SequenceTypeDriverSettlement     = SequenceType("driver_settlement")
	SequenceTypeCarrierSettlement    = SequenceType("carrier_settlement")
	SequenceTypeInvoiceRun           = SequenceType("invoice_run")
	SequenceTypeFactoringSubmission  = SequenceType("factoring_submission")
)

func (v SequenceType) IsValid() bool {
	switch v {
	case SequenceTypeProNumber,
		SequenceTypeConsolidation,
		SequenceTypeOrder,
		SequenceTypeInvoice,
		SequenceTypeInvoiceRun,
		SequenceTypeCreditMemo,
		SequenceTypeDebitMemo,
		SequenceTypeWorkOrder,
		SequenceTypeJournalBatch,
		SequenceTypeJournalEntry,
		SequenceTypeManualJournalRequest,
		SequenceTypeLocationCode,
		SequenceTypeDriverSettlement,
		SequenceTypeCarrierSettlement,
		SequenceTypeFactoringSubmission:
		return true
	default:
		return false
	}
}
