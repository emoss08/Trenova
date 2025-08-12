package tx204

// Minimal typed model for an X12 204 Load Tender (004010-oriented).

type LoadTender struct {
    Control    Control             `json:"control"`
    Header     Header              `json:"header"`
    Parties    map[string]Party    `json:"parties"`   // key by N1 entity code: BT, SH, ST, CN, SF, etc.
    Stops      []Stop              `json:"stops"`
    Equipment  Equipment           `json:"equipment"`
    Notes      []string            `json:"notes"`
    Totals     Totals              `json:"totals"`
    Commodities []Commodity        `json:"commodities"`
}

type Control struct {
    STControl string `json:"st_control"`
}

type Header struct {
    CarrierSCAC string              `json:"carrier_scac"`
    ShipmentID  string              `json:"shipment_id"`
    ActionCode  string              `json:"action_code"`
    References  map[string][]string `json:"references"` // qualifier -> values
}

type Party struct {
    Code       string   `json:"code"` // N1-01 e.g., BT/SH/ST
    Name       string   `json:"name"`
    IDCodeQual string   `json:"id_code_qual"`
    IDCode     string   `json:"id_code"`
    Address1   string   `json:"address1"`
    Address2   string   `json:"address2"`
    City       string   `json:"city"`
    State      string   `json:"state"`
    PostalCode string   `json:"postal_code"`
    Country    string   `json:"country"`
    Contacts   []string `json:"contacts"`
}

type Stop struct {
    Sequence   int       `json:"sequence"`
    Type       string    `json:"type"` // LD (pickup), UL (delivery), etc.
    Appointments []Appt  `json:"appointments"`
    Location   Party     `json:"location"`
    Notes      []string  `json:"notes"`
}

type Appt struct {
    Qualifier string `json:"qualifier"` // e.g., 133 (pickup), 132 (delivery)
    Date      string `json:"date"`
    Time      string `json:"time"`
}

type Equipment struct {
    Type    string `json:"type"`
    ID      string `json:"id"`
    TypeCode string `json:"type_code,omitempty"`
    Description string `json:"description,omitempty"`
    Length string `json:"length,omitempty"`
    Width  string `json:"width,omitempty"`
    Height string `json:"height,omitempty"`
    DimUnit string `json:"dim_unit,omitempty"`
}

type Totals struct {
    Weight     string `json:"weight"`
    WeightUnit string `json:"weight_unit"`
    Pieces     int    `json:"pieces"`
}

type Commodity struct {
    Description string `json:"description"`
    Code        string `json:"code"`
}
