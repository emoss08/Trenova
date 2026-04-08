package invoice

const daySeconds = int64(24 * 60 * 60)

func DueDateFromPaymentTerm(invoiceDate int64, term PaymentTerm) *int64 {
	if invoiceDate <= 0 {
		return nil
	}

	var days int64
	switch term {
	case PaymentTermDueOnReceipt:
		days = 0
	case PaymentTermNet10:
		days = 10
	case PaymentTermNet15:
		days = 15
	case PaymentTermNet30:
		days = 30
	case PaymentTermNet45:
		days = 45
	case PaymentTermNet60:
		days = 60
	case PaymentTermNet90:
		days = 90
	default:
		return nil
	}

	dueDate := invoiceDate + (days * daySeconds)
	return &dueDate
}
