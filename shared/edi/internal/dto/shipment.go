package dto

// Shipment is a minimal, TMS-oriented DTO produced from an inbound 204.
// It keeps names generic and leaves normalization/ID linking to the caller.
type Shipment struct {
    // Header
    CarrierSCAC string            `json:"carrier_scac"`
    ShipmentID  string            `json:"shipment_id"`
    ActionCode  string            `json:"action_code"`
    References  map[string]string `json:"references"` // common refs flattened where possible

    // Parties (header-level)
    BillTo   *Party `json:"bill_to,omitempty"`
    Shipper  *Party `json:"shipper,omitempty"`
    Consignee *Party `json:"consignee,omitempty"`

    // Execution
    Stops     []Stop     `json:"stops"`
    Equipment Equipment  `json:"equipment"`
    Notes     []string   `json:"notes"`
    // ReferencesRaw optionally includes the raw L11 qualifier/value arrays for auditing.
    ReferencesRaw map[string][]string `json:"references_raw,omitempty"`
    Totals   Totals     `json:"totals,omitempty"`
    Goods    []Commodity `json:"goods,omitempty"`
    ServiceLevel string        `json:"service_level,omitempty"`
    Accessorials []Accessorial `json:"accessorials,omitempty"`
}

type Party struct {
    Code       string   `json:"code,omitempty"`
    Name       string   `json:"name,omitempty"`
    IDCodeQual string   `json:"id_code_qual,omitempty"`
    IDCode     string   `json:"id_code,omitempty"`
    Address1   string   `json:"address1,omitempty"`
    Address2   string   `json:"address2,omitempty"`
    City       string   `json:"city,omitempty"`
    State      string   `json:"state,omitempty"`
    PostalCode string   `json:"postal_code,omitempty"`
    Country    string   `json:"country,omitempty"`
    Contacts   []string `json:"contacts,omitempty"`
}

type Stop struct {
    Sequence   int       `json:"sequence"`
    Type       string    `json:"type"` // pickup|delivery|other
    Location   Party     `json:"location"`
    Appointments []Appt  `json:"appointments,omitempty"`
    Notes      []string  `json:"notes,omitempty"`
}

type Appt struct {
    Qualifier string `json:"qualifier,omitempty"`
    Date      string `json:"date,omitempty"`
    Time      string `json:"time,omitempty"`
    DateTime  string `json:"datetime,omitempty"` // normalized ISO-8601 if enabled
}

type Equipment struct {
    Type string `json:"type,omitempty"`
    ID   string `json:"id,omitempty"`
}

type Totals struct {
    Weight     string `json:"weight,omitempty"`
    WeightUnit string `json:"weight_unit,omitempty"`
    Pieces     int    `json:"pieces,omitempty"`
}

type Commodity struct {
    Description string `json:"description,omitempty"`
    Code        string `json:"code,omitempty"`
}

type Accessorial struct {
    Code string `json:"code"`
    Name string `json:"name,omitempty"`
}
