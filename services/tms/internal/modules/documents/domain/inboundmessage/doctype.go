package inboundmessage

var attachmentDocumentTypeCodes = map[AttachmentKind]string{ //nolint:exhaustive // Unknown and Other files have no document type to file under
	AttachmentProofOfDelivery:  "POD",
	AttachmentBillOfLading:     "BOL",
	AttachmentRateConfirmation: "RATECONF",
	AttachmentInvoice:          "INVOICE",
}

func (k AttachmentKind) DocumentTypeCode() (string, bool) {
	code, ok := attachmentDocumentTypeCodes[k]

	return code, ok
}
