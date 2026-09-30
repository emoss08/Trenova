package invoiceadjustmentservice

// batchInlineThreshold is how many lines a bulk request may carry before it is
// worth processing each adjustment on its own rather than in one transaction.
const batchInlineThreshold = 25

// adjustmentDocumentResourceType is the document bucket a generated file is
// filed under when it was produced by an adjustment rather than by hand.
const adjustmentDocumentResourceType = "invoice_adjustment"
