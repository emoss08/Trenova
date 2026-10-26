//! Trenova's side of every map: its documents, their records and enums, the lookups that turn
//! a partner's codes into Trenova records, and the events a flow sends on.
//!
//! Every field and variant carries its doc as a catalog message, so completion and hover read
//! in the person's language. The TMS converts each document to and from its own payloads
//! (`services/tms/internal/modules/edi/app/ladingdoc`), so the shapes here are the contract.

use trenova_module_kit::message;

use super::types::{Dim, Type};

/// A field's type, as the schema writes it.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Spec {
    /// `Text`
    Text,
    /// `Int`
    Int,
    /// `Decimal`
    Decimal,
    /// `Bool`
    Bool,
    /// `Date`
    Date,
    /// `DateTime`
    DateTime,
    /// `Money`
    Money,
    /// A measure.
    Measure(Dim),
    /// A record of the schema.
    Record(&'static str),
    /// An enum of the schema.
    Enum(&'static str),
    /// A lookup reference.
    Ref(&'static str),
    /// `[Text: Text]`
    Refs,
    /// A list of records.
    Records(&'static str),
    /// `[Text]`
    Texts,
    /// The organisation's custom fields on this record.
    Custom(&'static str),
}

impl Spec {
    /// The type.
    #[must_use]
    pub fn ty(self) -> Type {
        match self {
            Self::Text => Type::Text,
            Self::Int => Type::Int,
            Self::Decimal => Type::Decimal,
            Self::Bool => Type::Bool,
            Self::Date => Type::Date,
            Self::DateTime => Type::DateTime,
            Self::Money => Type::Money,
            Self::Measure(dim) => Type::Measure(dim),
            Self::Record(name) => Type::Record(name),
            Self::Enum(name) => Type::Enum(name),
            Self::Ref(name) => Type::Ref(name),
            Self::Refs => Type::Map(Box::new(Type::Text)),
            Self::Records(name) => Type::Record(name).list(),
            Self::Texts => Type::Text.list(),
            Self::Custom(_) => Type::Struct(Vec::new()),
        }
    }
}

/// One field of a record.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Field {
    /// Its name.
    pub name: &'static str,
    /// Its type.
    pub spec: Spec,
    /// Whether an inbound map must fill it.
    pub required: bool,
    /// What it holds, as a catalog message.
    pub doc: &'static str,
    /// The most characters it holds, when Trenova limits it.
    pub max_length: Option<u32>,
}

const fn field(name: &'static str, spec: Spec, required: bool, doc: &'static str) -> Field {
    Field {
        name,
        spec,
        required,
        doc,
        max_length: None,
    }
}

const fn limited(
    name: &'static str,
    spec: Spec,
    required: bool,
    doc: &'static str,
    max_length: u32,
) -> Field {
    Field {
        name,
        spec,
        required,
        doc,
        max_length: Some(max_length),
    }
}

/// A record or document.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Record {
    /// Its name.
    pub name: &'static str,
    /// What it is.
    pub doc: &'static str,
    /// Its fields.
    pub fields: &'static [Field],
    /// The transaction set it is, for a document.
    pub set: Option<&'static str>,
}

impl Record {
    /// A field by name.
    #[must_use]
    pub fn field(&self, name: &str) -> Option<&'static Field> {
        self.fields.iter().find(|field| field.name == name)
    }
}

/// An enum.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Enum {
    /// Its name.
    pub name: &'static str,
    /// What it is.
    pub doc: &'static str,
    /// Its variants and their docs.
    pub variants: &'static [(&'static str, &'static str)],
}

/// A kind of lookup: a Trenova record a partner's code maps to.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Lookup {
    /// Its name.
    pub name: &'static str,
    /// The mapping profile entity type the TMS resolves it through.
    pub entity: &'static str,
    /// What it is.
    pub doc: &'static str,
    /// Whether it carries a time zone.
    pub zoned: bool,
}

/// An event a flow can send on.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Event {
    /// Its name.
    pub name: &'static str,
    /// The document Trenova builds for it.
    pub document: &'static str,
    /// When it happens.
    pub doc: &'static str,
}

const LOAD_TENDER: &[Field] = &[
    field(
        "key",
        Spec::Text,
        true,
        message!("The partner's identifier for the load, which every later message quotes"),
    ),
    field(
        "purpose",
        Spec::Enum("TenderPurpose"),
        false,
        message!("Whether this is a new tender, a change or a cancellation"),
    ),
    field(
        "bol",
        Spec::Text,
        false,
        message!("The bill of lading number"),
    ),
    field(
        "proNumber",
        Spec::Text,
        false,
        message!("The carrier's PRO number"),
    ),
    field(
        "poNumber",
        Spec::Text,
        false,
        message!("The purchase order number"),
    ),
    field(
        "customer",
        Spec::Ref("Customer"),
        false,
        message!("The customer the load is for"),
    ),
    field(
        "serviceType",
        Spec::Ref("ServiceType"),
        false,
        message!("The service type"),
    ),
    field(
        "shipmentType",
        Spec::Ref("ShipmentType"),
        false,
        message!("The shipment type"),
    ),
    field(
        "rating",
        Spec::Ref("RatingTemplate"),
        false,
        message!("The rating template that prices the load"),
    ),
    field(
        "payment",
        Spec::Enum("Payment"),
        false,
        message!("Who pays the freight"),
    ),
    field(
        "equipment",
        Spec::Text,
        false,
        message!("The equipment the load needs, as the partner names it"),
    ),
    field(
        "weight",
        Spec::Measure(Dim::Weight),
        false,
        message!("The load's weight"),
    ),
    field(
        "pieces",
        Spec::Int,
        false,
        message!("How many pieces the load has"),
    ),
    field(
        "temperature",
        Spec::Record("TemperatureRange"),
        false,
        message!("The temperature the load must be kept at"),
    ),
    field(
        "total",
        Spec::Money,
        false,
        message!("The total charge the partner tenders"),
    ),
    field(
        "freightCharge",
        Spec::Money,
        false,
        message!("The linehaul charge"),
    ),
    field(
        "otherCharges",
        Spec::Money,
        false,
        message!("Charges other than linehaul"),
    ),
    field(
        "refs",
        Spec::Refs,
        false,
        message!("Reference numbers by qualifier"),
    ),
    field("notes", Spec::Texts, false, message!("Notes for the load")),
    field(
        "stops",
        Spec::Records("Stop"),
        true,
        message!("The stops, pickups and deliveries, in order"),
    ),
    field(
        "commodities",
        Spec::Records("CommodityLine"),
        false,
        message!("What the load carries"),
    ),
    field(
        "charges",
        Spec::Records("Charge"),
        false,
        message!("Accessorial charges"),
    ),
    field(
        "custom",
        Spec::Custom("shipment"),
        false,
        message!("The organisation's custom fields on shipments"),
    ),
];

const TENDER_RESPONSE: &[Field] = &[
    field(
        "key",
        Spec::Text,
        true,
        message!("The load the response answers"),
    ),
    field(
        "decision",
        Spec::Enum("Decision"),
        true,
        message!("Whether the tender is accepted or declined"),
    ),
    limited(
        "reason",
        Spec::Text,
        false,
        message!("Why a tender is declined"),
        200,
    ),
    field(
        "scac",
        Spec::Text,
        false,
        message!("The responding carrier's SCAC"),
    ),
    field(
        "respondedAt",
        Spec::DateTime,
        true,
        message!("When the response was made"),
    ),
    field(
        "refs",
        Spec::Refs,
        false,
        message!("Reference numbers by qualifier"),
    ),
];

const SHIPMENT_STATUS: &[Field] = &[
    field(
        "key",
        Spec::Text,
        true,
        message!("The load the status is about"),
    ),
    field(
        "bol",
        Spec::Text,
        false,
        message!("The bill of lading number"),
    ),
    field(
        "proNumber",
        Spec::Text,
        false,
        message!("The carrier's PRO number"),
    ),
    field(
        "shipperReference",
        Spec::Text,
        false,
        message!("The shipper's reference"),
    ),
    field(
        "weight",
        Spec::Measure(Dim::Weight),
        false,
        message!("The load's weight"),
    ),
    field(
        "pieces",
        Spec::Int,
        false,
        message!("How many pieces the load has"),
    ),
    field(
        "equipment",
        Spec::Text,
        false,
        message!("The trailer or container number"),
    ),
    field(
        "refs",
        Spec::Refs,
        false,
        message!("Reference numbers by qualifier"),
    ),
    field(
        "events",
        Spec::Records("StatusEvent"),
        true,
        message!("What happened, in order"),
    ),
];

const FREIGHT_INVOICE: &[Field] = &[
    field("number", Spec::Text, true, message!("The invoice number")),
    field("date", Spec::Date, true, message!("The invoice date")),
    field(
        "deliveryDate",
        Spec::Date,
        false,
        message!("When the freight was delivered"),
    ),
    field(
        "key",
        Spec::Text,
        false,
        message!("The load the invoice bills"),
    ),
    field(
        "bol",
        Spec::Text,
        false,
        message!("The bill of lading number"),
    ),
    field(
        "proNumber",
        Spec::Text,
        false,
        message!("The carrier's PRO number"),
    ),
    field(
        "weight",
        Spec::Measure(Dim::Weight),
        false,
        message!("The billed weight"),
    ),
    field(
        "billTo",
        Spec::Record("Party"),
        false,
        message!("Who the invoice is addressed to"),
    ),
    field("total", Spec::Money, true, message!("The amount due")),
    field(
        "lines",
        Spec::Records("InvoiceLine"),
        false,
        message!("The charges, line by line"),
    ),
    field(
        "shipments",
        Spec::Records("InvoicedShipment"),
        false,
        message!("The shipments a consolidated invoice bills"),
    ),
    field(
        "refs",
        Spec::Refs,
        false,
        message!("Reference numbers by qualifier"),
    ),
];

const STOP: &[Field] = &[
    field(
        "sequence",
        Spec::Int,
        true,
        message!("The stop's position, from 1"),
    ),
    field(
        "type",
        Spec::Enum("StopType"),
        true,
        message!("Whether it is a pickup or a delivery"),
    ),
    field(
        "location",
        Spec::Ref("Location"),
        false,
        message!("The Trenova location"),
    ),
    field("name", Spec::Text, false, message!("The place's name")),
    field(
        "code",
        Spec::Text,
        false,
        message!("The partner's code for the place"),
    ),
    field(
        "address",
        Spec::Record("Address"),
        false,
        message!("The place's address"),
    ),
    field(
        "window",
        Spec::Record("Window"),
        true,
        message!("When the stop is scheduled"),
    ),
    field(
        "weight",
        Spec::Measure(Dim::Weight),
        false,
        message!("The weight picked up or delivered here"),
    ),
    field(
        "pieces",
        Spec::Int,
        false,
        message!("The pieces picked up or delivered here"),
    ),
    field(
        "refs",
        Spec::Refs,
        false,
        message!("Reference numbers by qualifier"),
    ),
    field("notes", Spec::Texts, false, message!("Notes for the stop")),
];

const WINDOW: &[Field] = &[
    field(
        "start",
        Spec::DateTime,
        true,
        message!("When the window opens"),
    ),
    field("end", Spec::DateTime, false, message!("When it closes")),
];

const ADDRESS: &[Field] = &[
    field("line1", Spec::Text, true, message!("The street address")),
    field(
        "line2",
        Spec::Text,
        false,
        message!("More of the street address"),
    ),
    field("city", Spec::Text, true, message!("The city")),
    field(
        "state",
        Spec::Text,
        false,
        message!("The state or province code"),
    ),
    field("postal", Spec::Text, false, message!("The postal code")),
    field("country", Spec::Text, false, message!("The country code")),
];

const PARTY: &[Field] = &[
    field("name", Spec::Text, true, message!("The name")),
    field(
        "address",
        Spec::Record("Address"),
        false,
        message!("The address"),
    ),
];

const COMMODITY_LINE: &[Field] = &[
    field(
        "commodity",
        Spec::Ref("Commodity"),
        false,
        message!("The Trenova commodity"),
    ),
    field("description", Spec::Text, true, message!("What it is")),
    field(
        "weight",
        Spec::Measure(Dim::Weight),
        false,
        message!("Its weight"),
    ),
    field("pieces", Spec::Int, false, message!("How many pieces")),
    field(
        "hazmat",
        Spec::Bool,
        false,
        message!("Whether it is hazardous material"),
    ),
];

const CHARGE: &[Field] = &[
    field(
        "accessorial",
        Spec::Ref("Accessorial"),
        false,
        message!("The Trenova accessorial charge"),
    ),
    field(
        "code",
        Spec::Text,
        false,
        message!("The partner's charge code"),
    ),
    field(
        "description",
        Spec::Text,
        false,
        message!("What the charge is for"),
    ),
    field("amount", Spec::Money, true, message!("The amount")),
];

const TEMPERATURE_RANGE: &[Field] = &[
    field(
        "min",
        Spec::Measure(Dim::Temperature),
        false,
        message!("The lowest temperature allowed"),
    ),
    field(
        "max",
        Spec::Measure(Dim::Temperature),
        false,
        message!("The highest temperature allowed"),
    ),
];

const STATUS_EVENT: &[Field] = &[
    field(
        "kind",
        Spec::Enum("EventKind"),
        true,
        message!("What happened"),
    ),
    field(
        "reason",
        Spec::Text,
        false,
        message!("The partner's reason code, as sent"),
    ),
    field("at", Spec::DateTime, true, message!("When it happened")),
    field(
        "local",
        Spec::Bool,
        false,
        message!("Whether the time is the clock where it happened, read in the stop's zone"),
    ),
    field("stop", Spec::Int, false, message!("The stop's sequence")),
    field(
        "city",
        Spec::Text,
        false,
        message!("Where it happened: the city"),
    ),
    field(
        "state",
        Spec::Text,
        false,
        message!("The state or province code"),
    ),
    field("country", Spec::Text, false, message!("The country code")),
    field("latitude", Spec::Decimal, false, message!("The latitude")),
    field("longitude", Spec::Decimal, false, message!("The longitude")),
    field(
        "equipment",
        Spec::Text,
        false,
        message!("The trailer or container number"),
    ),
    field(
        "eta",
        Spec::DateTime,
        false,
        message!("The new estimated arrival"),
    ),
];

const INVOICE_LINE: &[Field] = &[
    field(
        "sequence",
        Spec::Int,
        true,
        message!("The line's position, from 1"),
    ),
    field("code", Spec::Text, false, message!("The charge code")),
    field(
        "accessorial",
        Spec::Ref("Accessorial"),
        false,
        message!("The Trenova accessorial charge"),
    ),
    field(
        "description",
        Spec::Text,
        false,
        message!("What the charge is for"),
    ),
    field("amount", Spec::Money, true, message!("The amount")),
    field("rate", Spec::Decimal, false, message!("The rate")),
    field(
        "weight",
        Spec::Measure(Dim::Weight),
        false,
        message!("The weight it is charged on"),
    ),
];

const INVOICED_SHIPMENT: &[Field] = &[
    field(
        "proNumber",
        Spec::Text,
        false,
        message!("The carrier's PRO number"),
    ),
    field(
        "bol",
        Spec::Text,
        false,
        message!("The bill of lading number"),
    ),
    field(
        "poNumber",
        Spec::Text,
        false,
        message!("The purchase order number"),
    ),
    field(
        "deliveryDate",
        Spec::Date,
        false,
        message!("When it was delivered"),
    ),
    field("amount", Spec::Money, false, message!("What it is billed")),
];

/// Every document, the targets of inbound maps and sources of outbound ones.
pub const DOCUMENTS: &[Record] = &[
    Record {
        name: "loadTender",
        doc: message!("A load offered for hauling (204)"),
        fields: LOAD_TENDER,
        set: Some("204"),
    },
    Record {
        name: "tenderResponse",
        doc: message!("An answer to a tender (990)"),
        fields: TENDER_RESPONSE,
        set: Some("990"),
    },
    Record {
        name: "shipmentStatus",
        doc: message!("What happened to a load (214)"),
        fields: SHIPMENT_STATUS,
        set: Some("214"),
    },
    Record {
        name: "freightInvoice",
        doc: message!("A bill for freight (210)"),
        fields: FREIGHT_INVOICE,
        set: Some("210"),
    },
];

/// Every record that is not a document.
pub const RECORDS: &[Record] = &[
    Record {
        name: "Stop",
        doc: message!("A pickup or delivery"),
        fields: STOP,
        set: None,
    },
    Record {
        name: "Window",
        doc: message!("When something is scheduled"),
        fields: WINDOW,
        set: None,
    },
    Record {
        name: "Address",
        doc: message!("A postal address"),
        fields: ADDRESS,
        set: None,
    },
    Record {
        name: "Party",
        doc: message!("A named party and its address"),
        fields: PARTY,
        set: None,
    },
    Record {
        name: "CommodityLine",
        doc: message!("A commodity on a load"),
        fields: COMMODITY_LINE,
        set: None,
    },
    Record {
        name: "Charge",
        doc: message!("An accessorial charge on a load"),
        fields: CHARGE,
        set: None,
    },
    Record {
        name: "TemperatureRange",
        doc: message!("A temperature range"),
        fields: TEMPERATURE_RANGE,
        set: None,
    },
    Record {
        name: "StatusEvent",
        doc: message!("One thing that happened to a load"),
        fields: STATUS_EVENT,
        set: None,
    },
    Record {
        name: "InvoiceLine",
        doc: message!("One charge on an invoice"),
        fields: INVOICE_LINE,
        set: None,
    },
    Record {
        name: "InvoicedShipment",
        doc: message!("One shipment on a consolidated invoice"),
        fields: INVOICED_SHIPMENT,
        set: None,
    },
];

/// Every enum.
pub const ENUMS: &[Enum] = &[
    Enum {
        name: "TenderPurpose",
        doc: message!("What a tender asks"),
        variants: &[
            ("Original", message!("A new load")),
            ("Change", message!("A change to a load already tendered")),
            ("Cancel", message!("The load is withdrawn")),
        ],
    },
    Enum {
        name: "StopType",
        doc: message!("What happens at a stop"),
        variants: &[
            ("Pickup", message!("Freight is picked up")),
            ("Delivery", message!("Freight is delivered")),
        ],
    },
    Enum {
        name: "Payment",
        doc: message!("Who pays the freight"),
        variants: &[
            ("Prepaid", message!("The shipper pays")),
            ("Collect", message!("The consignee pays")),
            ("ThirdParty", message!("A third party pays")),
        ],
    },
    Enum {
        name: "Decision",
        doc: message!("An answer to a tender"),
        variants: &[
            ("Accept", message!("The load is accepted")),
            ("Decline", message!("The load is declined")),
        ],
    },
    Enum {
        name: "EventKind",
        doc: message!("What happened to a load"),
        variants: &[
            ("Dispatched", message!("A driver is on the way")),
            ("ArrivedPickup", message!("Arrived at a pickup")),
            ("Loaded", message!("Loaded")),
            ("DepartedPickup", message!("Left a pickup")),
            ("EnRoute", message!("On the road")),
            ("ArrivedDelivery", message!("Arrived at a delivery")),
            ("Delivered", message!("Delivered")),
            ("DepartedDelivery", message!("Left a delivery")),
            ("Delayed", message!("Running late")),
            ("AppointmentSet", message!("An appointment was made")),
            ("EtaChanged", message!("The estimated arrival changed")),
            ("Exception", message!("Something went wrong")),
        ],
    },
];

/// Every lookup kind.
pub const LOOKUPS: &[Lookup] = &[
    Lookup {
        name: "Customer",
        entity: "Customer",
        doc: message!("A customer"),
        zoned: false,
    },
    Lookup {
        name: "Location",
        entity: "Location",
        doc: message!("A location"),
        zoned: true,
    },
    Lookup {
        name: "Commodity",
        entity: "Commodity",
        doc: message!("A commodity"),
        zoned: false,
    },
    Lookup {
        name: "Accessorial",
        entity: "AccessorialCharge",
        doc: message!("An accessorial charge"),
        zoned: false,
    },
    Lookup {
        name: "ServiceType",
        entity: "ServiceType",
        doc: message!("A service type"),
        zoned: false,
    },
    Lookup {
        name: "ShipmentType",
        entity: "ShipmentType",
        doc: message!("A shipment type"),
        zoned: false,
    },
    Lookup {
        name: "RatingTemplate",
        entity: "FormulaTemplate",
        doc: message!("A rating template"),
        zoned: false,
    },
    Lookup {
        name: "ServiceFailureReason",
        entity: "ServiceFailureReasonCode",
        doc: message!("A service failure reason"),
        zoned: false,
    },
];

/// Every event a flow can send on.
pub const EVENTS: &[Event] = &[
    Event {
        name: "TenderOffered",
        document: "loadTender",
        doc: message!("A load is offered to the partner"),
    },
    Event {
        name: "TenderCancelled",
        document: "loadTender",
        doc: message!("A load offered to the partner is withdrawn"),
    },
    Event {
        name: "TenderAccepted",
        document: "tenderResponse",
        doc: message!("A tender from the partner is accepted"),
    },
    Event {
        name: "TenderDeclined",
        document: "tenderResponse",
        doc: message!("A tender from the partner is declined"),
    },
    Event {
        name: "Dispatched",
        document: "shipmentStatus",
        doc: message!("The load is dispatched"),
    },
    Event {
        name: "StopArrived",
        document: "shipmentStatus",
        doc: message!("The driver arrives at a stop"),
    },
    Event {
        name: "StopDeparted",
        document: "shipmentStatus",
        doc: message!("The driver leaves a stop"),
    },
    Event {
        name: "Delivered",
        document: "shipmentStatus",
        doc: message!("The load is delivered"),
    },
    Event {
        name: "EtaChanged",
        document: "shipmentStatus",
        doc: message!("A stop's estimated arrival changes"),
    },
    Event {
        name: "ServiceFailure",
        document: "shipmentStatus",
        doc: message!("A service failure is recorded"),
    },
    Event {
        name: "InvoicePosted",
        document: "freightInvoice",
        doc: message!("An invoice is posted"),
    },
    Event {
        name: "InvoiceVoided",
        document: "freightInvoice",
        doc: message!("An invoice is voided"),
    },
];

/// The states a flow's `cancel unless` names.
pub const STATES: &[(&str, &str)] = &[
    ("dispatched", message!("The load has been dispatched")),
    ("delivered", message!("The load has been delivered")),
    ("invoiced", message!("The load has been invoiced")),
];

/// A document by name.
#[must_use]
pub fn document(name: &str) -> Option<&'static Record> {
    DOCUMENTS.iter().find(|record| record.name == name)
}

/// A record or document by name.
#[must_use]
pub fn record(name: &str) -> Option<&'static Record> {
    RECORDS
        .iter()
        .chain(DOCUMENTS)
        .find(|record| record.name == name)
}

/// An enum by name.
#[must_use]
pub fn enumeration(name: &str) -> Option<&'static Enum> {
    ENUMS.iter().find(|item| item.name == name)
}

/// The enum a variant belongs to.
#[must_use]
pub fn enum_of_variant(variant: &str) -> Option<&'static Enum> {
    ENUMS
        .iter()
        .find(|item| item.variants.iter().any(|(name, _)| *name == variant))
}

/// A lookup kind by name.
#[must_use]
pub fn lookup(name: &str) -> Option<&'static Lookup> {
    LOOKUPS.iter().find(|item| item.name == name)
}

/// An event by name.
#[must_use]
pub fn event(name: &str) -> Option<&'static Event> {
    EVENTS.iter().find(|item| item.name == name)
}

/// The fields of a lookup reference.
#[must_use]
pub fn ref_fields(kind: &str) -> Vec<(&'static str, Type, &'static str)> {
    let mut fields = vec![
        ("id", Type::Text, message!("The Trenova record's ID")),
        (
            "code",
            Type::Text,
            message!("The partner's code it was found by"),
        ),
        ("label", Type::Text, message!("The record's name")),
    ];
    if lookup(kind).is_some_and(|item| item.zoned) {
        fields.push((
            "zone",
            Type::Zone.optional(),
            message!("The location's time zone"),
        ));
    }
    fields
}

/// The type a written type name stands for, outside generics and lists.
#[must_use]
pub fn named_type(name: &str) -> Option<Type> {
    Some(match name {
        "Text" => Type::Text,
        "Int" => Type::Int,
        "Decimal" => Type::Decimal,
        "Bool" => Type::Bool,
        "Date" => Type::Date,
        "Time" => Type::Time,
        "DateTime" => Type::DateTime,
        "Zone" => Type::Zone,
        "Duration" => Type::Duration,
        "Percent" => Type::Percent,
        "Money" => Type::Money,
        "Weight" => Type::Measure(Dim::Weight),
        "Length" => Type::Measure(Dim::Length),
        "Volume" => Type::Measure(Dim::Volume),
        "Temperature" => Type::Measure(Dim::Temperature),
        "Json" => Type::Json,
        _ => {
            if let Some(item) = enumeration(name) {
                Type::Enum(item.name)
            } else if let Some(item) = lookup(name) {
                Type::Ref(item.name)
            } else {
                Type::Record(RECORDS.iter().find(|record| record.name == name)?.name)
            }
        }
    })
}

#[cfg(test)]
mod tests {
    use super::{DOCUMENTS, ENUMS, EVENTS, RECORDS, Spec, document, record};

    #[test]
    fn every_record_a_field_names_exists() {
        for owner in DOCUMENTS.iter().chain(RECORDS) {
            for field in owner.fields {
                match field.spec {
                    Spec::Record(name) | Spec::Records(name) => {
                        assert!(
                            record(name).is_some(),
                            "{}.{} names {name}",
                            owner.name,
                            field.name
                        );
                    }
                    Spec::Enum(name) => {
                        assert!(ENUMS.iter().any(|item| item.name == name), "{name}");
                    }
                    _ => {}
                }
            }
        }
    }

    #[test]
    fn every_event_builds_a_document() {
        for event in EVENTS {
            assert!(document(event.document).is_some(), "{}", event.name);
        }
    }
}
