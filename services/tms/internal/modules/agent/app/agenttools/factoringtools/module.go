package factoringtools

func Providers() []any {
	return []any{
		NewDraftFactoringNoaTool,
		NewSendFactoringNoaTool,
		NewRecordFactoringNoaAcknowledgementTool,
		NewReleaseFactoringNoaTool,
		NewSetInvoiceFactoringOverrideTool,
		NewRequestMissingFactoringDocumentTool,
	}
}
