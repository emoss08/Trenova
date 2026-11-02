package billingvocab

type BillType string

const (
	BillTypeInvoice    = BillType("Invoice")
	BillTypeCreditMemo = BillType("CreditMemo")
	BillTypeDebitMemo  = BillType("DebitMemo")
)
