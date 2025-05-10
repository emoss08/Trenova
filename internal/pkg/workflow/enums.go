package workflow

type Type string

const (
	TypeShipmentUpdated = Type("shipment_updated")
	TypeMarkReadyToBill = Type("mark_ready_to_bill")
)

func (t Type) String() string {
	return string(t)
}
