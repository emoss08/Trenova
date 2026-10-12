package inboundmessage

var attachmentDocumentTypeCodes = map[AttachmentKind]string{
	AttachmentProofOfDelivery:  "POD",
	AttachmentBillOfLading:     "BOL",
	AttachmentRateConfirmation: "RATECONF",
	AttachmentInvoice:          "INVOICE",
}

func (k AttachmentKind) DocumentTypeCode() (string, bool) {
	code, ok := attachmentDocumentTypeCodes[k]

	return code, ok
}
