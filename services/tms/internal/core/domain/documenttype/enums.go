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

// CodeDetentionNotice files a rendered detention notice against its shipment.
//
// It is a constant because three places have to agree on it — the seed that
// creates it for a new organization, the migration that backfills it for existing
// ones, and the send path that looks it up. The column is varchar(10), so the
// name is abbreviated rather than spelled out.
const CodeDetentionNotice = "DETNOTICE"

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

// CodeRateConfirmation files a rendered carrier rate confirmation against its
// shipment. Like CodeDetentionNotice, the seed, the backfill migration, and the
// generate path all have to agree on it.
const CodeRateConfirmation = "RATECON"
