package documenttype

type DocumentClassification string

const (
	ClassificationPublic     = DocumentClassification("Public")
	ClassificationPrivate    = DocumentClassification("Private")
	ClassificationSensitive  = DocumentClassification("Sensitive")
	ClassificationRegulatory = DocumentClassification("Regulatory")
)

func (dc DocumentClassification) String() string {
	return string(dc)
}

type DocumentCategory string

const (
	CategoryShipment   = DocumentCategory("Shipment")   // BOL, POD, etc...
	CategoryWorker     = DocumentCategory("Worker")     // Worker docs, licenses
	CategoryRegulatory = DocumentCategory("Regulatory") // Regulatory docs, certificates, etc...
	CategoryProfile    = DocumentCategory("Profile")    // Profile photos, etc...
	CategoryBranding   = DocumentCategory("Branding")   // Branding files, etc...
	CategoryInvoice    = DocumentCategory("Invoice")    // Invoice files, etc...
	CategoryContract   = DocumentCategory("Contract")   // Contract files, etc...
	CategoryOther      = DocumentCategory("Other")      // Other files, etc...
)

func (dc DocumentCategory) String() string {
	return string(dc)
}
