// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package sequencestore

// SequenceType represents different types of sequences
type SequenceType string

const (
	SequenceTypeProNumber     SequenceType = "pro_number"
	SequenceTypeConsolidation SequenceType = "consolidation"
	SequenceTypeInvoice       SequenceType = "invoice"
	SequenceTypeWorkOrder     SequenceType = "work_order"
)
