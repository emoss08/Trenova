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
	CategoryShipment   = DocumentCategory("Shipment")
	CategoryWorker     = DocumentCategory("Worker")
	CategoryRegulatory = DocumentCategory("Regulatory")
	CategoryProfile    = DocumentCategory("Profile")
	CategoryBranding   = DocumentCategory("Branding")
	CategoryInvoice    = DocumentCategory("Invoice")
	CategoryContract   = DocumentCategory("Contract")
	CategoryOther      = DocumentCategory("Other")
)

func (dc DocumentCategory) String() string {
	return string(dc)
}
