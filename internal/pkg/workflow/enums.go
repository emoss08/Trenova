package workflow

type Type string

const (
	TypeDocumentUpload  = Type("document_upload")
	TypeShipmentUpdated = Type("shipment_updated")
)

func (t Type) String() string {
	return string(t)
}
