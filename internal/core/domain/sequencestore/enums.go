package sequencestore

// SequenceType represents different types of sequences
type SequenceType string

const (
	SequenceTypeProNumber     SequenceType = "pro_number"
	SequenceTypeConsolidation SequenceType = "consolidation"
	SequenceTypeInvoice       SequenceType = "invoice"
	SequenceTypeWorkOrder     SequenceType = "work_order"
)
