// Package ladingdoc converts between Trenova's EDI payloads and the documents Lading maps read
// and write (services/crates/edi/src/lading/schema.rs is the contract).
package ladingdoc

const (
	UnitPounds     = "lb"
	UnitFahrenheit = "°F"
	DefaultCurency = "USD"
)

const (
	PurposeOriginal = "Original"
	PurposeChange   = "Change"
	PurposeCancel   = "Cancel"
)

const (
	StopPickup   = "Pickup"
	StopDelivery = "Delivery"
)

const (
	DecisionAccept  = "Accept"
	DecisionDecline = "Decline"
)

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency,omitempty"`
}

type Measure struct {
	Value string `json:"value"`
	Unit  string `json:"unit"`
}

type DateTime struct {
	At   string `json:"at"`
	Zone string `json:"zone,omitempty"`
}

type Ref struct {
	ID    string `json:"id,omitempty"`
	Code  string `json:"code,omitempty"`
	Label string `json:"label,omitempty"`
	Zone  string `json:"zone,omitempty"`
}

type Address struct {
	Line1   string `json:"line1"`
	Line2   string `json:"line2,omitempty"`
	City    string `json:"city"`
	State   string `json:"state,omitempty"`
	Postal  string `json:"postal,omitempty"`
	Country string `json:"country,omitempty"`
}

type Window struct {
	Start DateTime  `json:"start"`
	End   *DateTime `json:"end,omitempty"`
}

type Stop struct {
	Sequence int64             `json:"sequence"`
	Type     string            `json:"type"`
	Location *Ref              `json:"location,omitempty"`
	Name     string            `json:"name,omitempty"`
	Code     string            `json:"code,omitempty"`
	Address  *Address          `json:"address,omitempty"`
	Window   Window            `json:"window"`
	Weight   *Measure          `json:"weight,omitempty"`
	Pieces   *int64            `json:"pieces,omitempty"`
	Refs     map[string]string `json:"refs,omitempty"`
	Notes    []string          `json:"notes,omitempty"`
}

type CommodityLine struct {
	Commodity   *Ref     `json:"commodity,omitempty"`
	Description string   `json:"description"`
	Weight      *Measure `json:"weight,omitempty"`
	Pieces      *int64   `json:"pieces,omitempty"`
	Hazmat      *bool    `json:"hazmat,omitempty"`
}

type Charge struct {
	Accessorial *Ref   `json:"accessorial,omitempty"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
	Amount      Money  `json:"amount"`
}

type TemperatureRange struct {
	Min *Measure `json:"min,omitempty"`
	Max *Measure `json:"max,omitempty"`
}

type LoadTender struct {
	Key           string            `json:"key"`
	Purpose       string            `json:"purpose,omitempty"`
	BOL           string            `json:"bol,omitempty"`
	ProNumber     string            `json:"proNumber,omitempty"`
	PONumber      string            `json:"poNumber,omitempty"`
	Customer      *Ref              `json:"customer,omitempty"`
	ServiceType   *Ref              `json:"serviceType,omitempty"`
	ShipmentType  *Ref              `json:"shipmentType,omitempty"`
	Rating        *Ref              `json:"rating,omitempty"`
	Payment       string            `json:"payment,omitempty"`
	Equipment     string            `json:"equipment,omitempty"`
	Weight        *Measure          `json:"weight,omitempty"`
	Pieces        *int64            `json:"pieces,omitempty"`
	Temperature   *TemperatureRange `json:"temperature,omitempty"`
	Total         *Money            `json:"total,omitempty"`
	FreightCharge *Money            `json:"freightCharge,omitempty"`
	OtherCharges  *Money            `json:"otherCharges,omitempty"`
	Refs          map[string]string `json:"refs,omitempty"`
	Notes         []string          `json:"notes,omitempty"`
	Stops         []Stop            `json:"stops"`
	Commodities   []CommodityLine   `json:"commodities,omitempty"`
	Charges       []Charge          `json:"charges,omitempty"`
	Custom        map[string]any    `json:"custom,omitempty"`
}

type TenderResponse struct {
	Key         string            `json:"key"`
	Decision    string            `json:"decision"`
	Reason      string            `json:"reason,omitempty"`
	SCAC        string            `json:"scac,omitempty"`
	RespondedAt DateTime          `json:"respondedAt"`
	Refs        map[string]string `json:"refs,omitempty"`
}

type StatusEvent struct {
	Kind      string    `json:"kind"`
	Reason    string    `json:"reason,omitempty"`
	At        DateTime  `json:"at"`
	Local     bool      `json:"local,omitempty"`
	Stop      *int64    `json:"stop,omitempty"`
	City      string    `json:"city,omitempty"`
	State     string    `json:"state,omitempty"`
	Country   string    `json:"country,omitempty"`
	Latitude  string    `json:"latitude,omitempty"`
	Longitude string    `json:"longitude,omitempty"`
	Equipment string    `json:"equipment,omitempty"`
	ETA       *DateTime `json:"eta,omitempty"`
}

type ShipmentStatus struct {
	Key              string            `json:"key"`
	BOL              string            `json:"bol,omitempty"`
	ProNumber        string            `json:"proNumber,omitempty"`
	ShipperReference string            `json:"shipperReference,omitempty"`
	Weight           *Measure          `json:"weight,omitempty"`
	Pieces           *int64            `json:"pieces,omitempty"`
	Equipment        string            `json:"equipment,omitempty"`
	Refs             map[string]string `json:"refs,omitempty"`
	Events           []StatusEvent     `json:"events"`
}

type Party struct {
	Name    string   `json:"name"`
	Address *Address `json:"address,omitempty"`
}

type InvoiceLine struct {
	Sequence    int64    `json:"sequence"`
	Code        string   `json:"code,omitempty"`
	Accessorial *Ref     `json:"accessorial,omitempty"`
	Description string   `json:"description,omitempty"`
	Amount      Money    `json:"amount"`
	Rate        string   `json:"rate,omitempty"`
	Weight      *Measure `json:"weight,omitempty"`
}

type InvoicedShipment struct {
	ProNumber    string `json:"proNumber,omitempty"`
	BOL          string `json:"bol,omitempty"`
	PONumber     string `json:"poNumber,omitempty"`
	DeliveryDate string `json:"deliveryDate,omitempty"`
	Amount       *Money `json:"amount,omitempty"`
}

type FreightInvoice struct {
	Number       string             `json:"number"`
	Date         string             `json:"date"`
	DeliveryDate string             `json:"deliveryDate,omitempty"`
	Key          string             `json:"key,omitempty"`
	BOL          string             `json:"bol,omitempty"`
	ProNumber    string             `json:"proNumber,omitempty"`
	Weight       *Measure           `json:"weight,omitempty"`
	BillTo       *Party             `json:"billTo,omitempty"`
	Total        Money              `json:"total"`
	Lines        []InvoiceLine      `json:"lines,omitempty"`
	Shipments    []InvoicedShipment `json:"shipments,omitempty"`
	Refs         map[string]string  `json:"refs,omitempty"`
}
