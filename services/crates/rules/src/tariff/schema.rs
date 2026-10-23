//! The data a tariff reads: the schema roots, their record and enum types, and the shapes of
//! the rate tables and charge codes the host describes.
//!
//! The roots are `shipment`, `lane`, `customer`, `fuel`, `rating` and `tables`. Their types are
//! fixed by the language (the reference lists them under "Data"), so the browser and the server
//! check against the same schema without fetching it; the host only says which unit the
//! organisation stores distances in, and describes its rate tables and accessorial codes.
//!
//! The Go host builds the record it sends from these same names; `tariff.catalog` publishes the
//! schema with its docs for the editor and for Nova.

use super::docs::Doc;
use super::types::{EnumType, FieldDef, RecordType, Type};
use super::units::{Base, Dim, Unit};
use crate::message;
use serde::Deserialize;
use std::sync::Arc;

/// The doc of `tables`, the root that holds the organisation's active rate matrices.
pub const TABLES_DOC: &str = message!("The organisation's active rate matrices, by code.");

/// The schema enums, by name, with their variants in order.
pub const ENUMS: &[(&str, &[&str])] = &[
    (
        "ShipmentStatus",
        &[
            "New",
            "PartiallyAssigned",
            "Assigned",
            "InTransit",
            "Delayed",
            "PartiallyCompleted",
            "ReadyToInvoice",
            "Completed",
            "Invoiced",
            "Canceled",
        ],
    ),
    (
        "StopKind",
        &["Pickup", "Delivery", "SplitPickup", "SplitDelivery"],
    ),
    ("StopStatus", &["New", "InTransit", "Completed", "Canceled"]),
    (
        "FreightClass",
        &[
            "Class50",
            "Class55",
            "Class60",
            "Class65",
            "Class70",
            "Class77_5",
            "Class85",
            "Class92_5",
            "Class100",
            "Class110",
            "Class125",
            "Class150",
            "Class175",
            "Class200",
            "Class250",
            "Class300",
            "Class400",
            "Class500",
        ],
    ),
    ("RatingMode", &["Sell", "Buy"]),
    (
        "LocationCategory",
        &[
            "Terminal",
            "Warehouse",
            "DistributionCenter",
            "TruckStop",
            "RestArea",
            "CustomerLocation",
            "Port",
            "RailYard",
            "MaintenanceFacility",
        ],
    ),
    (
        "FacilityType",
        &[
            "CrossDock",
            "StorageWarehouse",
            "ColdStorage",
            "HazmatFacility",
            "IntermodalFacility",
        ],
    ),
    (
        "Weekday",
        &["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"],
    ),
];

/// How the organisation stores what the schema exposes, as the host sends it.
#[derive(Debug, Clone, Deserialize, Default)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Settings {
    /// `mi` or `km`; absent is `mi`.
    #[serde(default)]
    pub distance_unit: Option<String>,
}

/// One axis of a rate table.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Axis {
    /// `Exact` or `Range`.
    #[serde(rename = "match")]
    pub match_mode: String,
    /// What the axis varies by, as Go's `ratematrix.DimensionKind` (`Zone`, `WeightBreak`, …).
    pub kind: String,
    /// The name the author gave the axis, for completion and messages.
    #[serde(default)]
    pub label: String,
    /// The unit a range axis's bounds are stored in (`lb`, `mi`, `ft`); absent for exact axes
    /// and for counts.
    #[serde(default)]
    pub unit: Option<String>,
    /// How an exact axis normalises keys, as Go's `ratematrix.KeyNormalization`: `None`,
    /// `Trim`, `Upper` or `Zip3`; empty is `None`.
    #[serde(default)]
    pub normalization: String,
    /// What a range axis does with a key no band covers, as Go's `ratematrix.RangeOverflow`:
    /// `Error`, `ClampToTopBand` or `Nearest`; empty is `Error`.
    #[serde(default)]
    pub overflow: String,
}

impl Axis {
    /// The type of the key a tariff gives this axis.
    #[must_use]
    pub fn key_type(&self, schema: &Schema) -> Type {
        match self.kind.as_str() {
            "WeightBreak" => Type::of(Base::Weight),
            "Distance" => Type::of(Base::Distance),
            "LinearFeet" => Type::of(Base::Length),
            "PieceCount" => Type::Int,
            "Quantity" => Type::Number,
            "FreightClass" => schema
                .enumeration("FreightClass")
                .map_or(Type::Text, Type::Enum),
            _ => Type::Text,
        }
    }
}

/// A rate table as the checker sees it: its code, axes and value type. The evaluator receives
/// the same shape with its cells.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TableShape {
    /// The matrix code a tariff names: `tables.Zones`.
    pub code: String,
    /// The matrix's name, for hover.
    #[serde(default)]
    pub name: String,
    /// The axes in position order: one to four.
    pub axes: Vec<Axis>,
    /// What a cell holds, as a Tariff type: `Money`, `Money/cwt`, `Money/mi`, `Percent`,
    /// `Number`.
    pub value_type: String,
    /// The cells, which the evaluator reads; the checker needs only the shape, so a host may
    /// leave them out for `tariff.check` and the editor.
    #[serde(default)]
    pub cells: Option<Vec<super::eval::tables::CellWire>>,
    /// The exact-axis key combinations whose cells `cells` holds, one normalised key per exact
    /// axis in axis order; absent when `cells` is the whole table. A lookup outside them is a
    /// need the host answers by loading those cells and evaluating again.
    #[serde(default)]
    pub loaded: Option<Vec<Vec<String>>>,
}

impl TableShape {
    /// The type of a cell's value, or `None` when `value_type` is not one a table may hold.
    #[must_use]
    pub fn value(&self) -> Option<Type> {
        value_type(&self.value_type)
    }
}

/// Reads a table's value type: `Money`, `Money/<unit>`, `Percent`, `Number` or `Int`.
#[must_use]
pub fn value_type(text: &str) -> Option<Type> {
    match text.trim() {
        "Money" => Some(Type::money()),
        "Percent" => Some(Type::Percent),
        "Number" => Some(Type::Number),
        "Int" => Some(Type::Int),
        other => {
            let per = other.strip_prefix("Money/")?;
            let unit = Unit::parse(&format!("$/{per}")).ok()?;
            Some(Type::quantity_in(&unit))
        }
    }
}

/// An accessorial charge code a tariff may bill with `charge … as CODE`.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct ChargeCode {
    /// The code, up to ten characters: `FSC`.
    pub code: String,
    /// What it is, for hover and completion.
    #[serde(default)]
    pub description: String,
    /// Whether it may be billed; an inactive code is refused.
    pub active: bool,
}

/// The schema: every root, record type and enum a file can name before it declares anything.
#[derive(Debug, Clone)]
pub struct Schema {
    /// The roots, by name, with their types and docs.
    pub roots: Vec<(&'static str, Type, Doc)>,
    /// The named record types: `Stop`, `Commodity`.
    pub records: Vec<Arc<RecordType>>,
    /// The enums.
    pub enums: Vec<Arc<EnumType>>,
}

impl Schema {
    /// The schema, with distances in the organisation's unit.
    #[must_use]
    pub fn new(settings: &Settings) -> Self {
        let builder = Builder::new(settings);
        let stop = builder.stop();
        let commodity = builder.commodity();
        Self {
            roots: vec![
                (
                    "shipment",
                    builder.shipment(&stop, &commodity),
                    Doc::message(message!("The shipment being rated.")),
                ),
                (
                    "lane",
                    Builder::lane(),
                    Doc::message(message!("Where it goes.")),
                ),
                (
                    "customer",
                    Builder::customer(),
                    Doc::message(message!("Who is billed.")),
                ),
                (
                    "fuel",
                    Builder::fuel(),
                    Doc::message(message!("The fuel index.")),
                ),
                (
                    "rating",
                    builder.rating(),
                    Doc::message(message!("The rating itself.")),
                ),
            ],
            records: vec![stop, commodity],
            enums: builder.enums,
        }
    }

    /// The root named `name`.
    #[must_use]
    pub fn root(&self, name: &str) -> Option<&Type> {
        self.roots
            .iter()
            .find(|(root, _, _)| *root == name)
            .map(|(_, ty, _)| ty)
    }

    /// The schema enum named `name`.
    #[must_use]
    pub fn enumeration(&self, name: &str) -> Option<Arc<EnumType>> {
        self.enums
            .iter()
            .find(|enumeration| enumeration.name == name)
            .cloned()
    }

    /// The schema record type named `name`.
    #[must_use]
    pub fn record(&self, name: &str) -> Option<Arc<RecordType>> {
        self.records
            .iter()
            .find(|record| record.name.as_deref() == Some(name))
            .cloned()
    }
}

struct Builder {
    enums: Vec<Arc<EnumType>>,
    distance: Type,
    weight: Type,
    length: Type,
    volume: Type,
    duration: Type,
    temperature: Type,
    density: Type,
}

impl Builder {
    fn new(settings: &Settings) -> Self {
        let enums = ENUMS
            .iter()
            .map(|(name, variants)| {
                Arc::new(EnumType {
                    name: (*name).to_owned(),
                    variants: variants.iter().map(|v| ((*v).to_owned(), None)).collect(),
                    doc: None,
                })
            })
            .collect();
        let distance = Unit::parse(settings.distance_unit.as_deref().unwrap_or("mi"))
            .ok()
            .filter(|unit| unit.dim() == Dim::of(Base::Distance))
            .map_or_else(|| Type::of(Base::Distance), |unit| Type::quantity_in(&unit));
        Self {
            enums,
            distance,
            weight: Type::quantity_in(&unit("lb")),
            length: Type::quantity_in(&unit("ft")),
            volume: Type::quantity_in(&unit("ft3")),
            duration: Type::quantity_in(&unit("h")),
            temperature: Type::quantity_in(&unit("°F")),
            density: Type::quantity_in(&unit("lb/ft3")),
        }
    }

    fn enumeration(&self, name: &str) -> Type {
        self.enums
            .iter()
            .find(|enumeration| enumeration.name == name)
            .map_or(Type::Error, |found| Type::Enum(Arc::clone(found)))
    }

    fn stop(&self) -> Arc<RecordType> {
        let location = record(&[
            ("city", Type::Text, message!("The city.")),
            ("state", Type::Text, message!("The state or province code.")),
            ("zip", Type::Text, message!("The postal code.")),
            (
                "zip3",
                Type::Text,
                message!("The first three characters of the postal code."),
            ),
            ("country", Type::Text, message!("The ISO country code.")),
            ("timezone", Type::Text, message!("The IANA time zone.")),
            (
                "category",
                opt(self.enumeration("LocationCategory")),
                message!("What the location is: a terminal, a warehouse, a port, …"),
            ),
            (
                "facilityType",
                opt(self.enumeration("FacilityType")),
                message!("What kind of facility the location is: cross-dock, cold storage, …"),
            ),
            (
                "requiresAppointment",
                Type::Bool,
                message!("Whether the location requires an appointment for every visit."),
            ),
        ]);
        let window = record(&[
            (
                "start",
                opt(Type::DateTime),
                message!("When the window opens."),
            ),
            (
                "end",
                opt(Type::DateTime),
                message!("When the window closes."),
            ),
        ]);
        Arc::new(RecordType {
            name: Some(String::from("Stop")),
            fields: fields(&[
                (
                    "sequence",
                    Type::Int,
                    message!("The stop's place in the route, from 1."),
                ),
                (
                    "kind",
                    self.enumeration("StopKind"),
                    message!("Pickup, delivery or a split."),
                ),
                (
                    "status",
                    self.enumeration("StopStatus"),
                    message!("Where the stop is in its life."),
                ),
                (
                    "appointment",
                    Type::Bool,
                    message!("Whether the stop is by appointment."),
                ),
                ("window", window, message!("The scheduled window.")),
                (
                    "arrival",
                    opt(Type::DateTime),
                    message!("When the equipment arrived."),
                ),
                (
                    "departure",
                    opt(Type::DateTime),
                    message!("When the equipment left."),
                ),
                (
                    "dwell",
                    opt(self.duration.clone()),
                    message!("Departure minus arrival, when both are recorded."),
                ),
                (
                    "pieces",
                    opt(Type::Int),
                    message!("Pieces handled at the stop."),
                ),
                (
                    "weight",
                    opt(self.weight.clone()),
                    message!("Weight handled at the stop."),
                ),
                (
                    "legDistance",
                    opt(self.distance.clone()),
                    message!(
                        "The distance from the previous stop, in the organisation's unit; none on the first stop."
                    ),
                ),
                ("location", location, message!("Where the stop is.")),
            ]),
        })
    }

    fn commodity(&self) -> Arc<RecordType> {
        let dimensions = record(&[
            ("length", opt(self.length.clone()), message!("The length.")),
            ("width", opt(self.length.clone()), message!("The width.")),
            ("height", opt(self.length.clone()), message!("The height.")),
        ]);
        Arc::new(RecordType {
            name: Some(String::from("Commodity")),
            fields: fields(&[
                ("name", Type::Text, message!("The commodity's name.")),
                (
                    "freightClass",
                    opt(self.enumeration("FreightClass")),
                    message!("Its NMFC class."),
                ),
                (
                    "weight",
                    self.weight.clone(),
                    message!("The line's weight."),
                ),
                ("pieces", Type::Int, message!("The line's pieces.")),
                (
                    "dimensions",
                    dimensions,
                    message!("Its dimensions, when recorded."),
                ),
                (
                    "volume",
                    opt(self.volume.clone()),
                    message!("Its volume, from the dimensions."),
                ),
                (
                    "density",
                    opt(self.density.clone()),
                    message!("Its density, from weight and volume."),
                ),
                (
                    "hazmat",
                    Type::Bool,
                    message!("Whether it is hazardous material."),
                ),
                (
                    "stackable",
                    Type::Bool,
                    message!("Whether it can be stacked."),
                ),
            ]),
        })
    }

    fn shipment(&self, stop: &Arc<RecordType>, commodity: &Arc<RecordType>) -> Type {
        let mut list = self.freight_fields();
        list.extend(Self::route_fields(stop, commodity));
        Type::Record(Arc::new(RecordType {
            name: None,
            fields: fields(&list),
        }))
    }

    fn route_fields(
        stop: &Arc<RecordType>,
        commodity: &Arc<RecordType>,
    ) -> Vec<(&'static str, Type, &'static str)> {
        let reference = record(&[
            ("code", Type::Text, message!("The code.")),
            ("description", Type::Text, message!("The description.")),
        ]);
        let stop = Type::Record(Arc::clone(stop));
        vec![
            (
                "serviceType",
                reference.clone(),
                message!("The service type."),
            ),
            (
                "shipmentType",
                opt(reference.clone()),
                message!("The shipment type."),
            ),
            (
                "tractorType",
                opt(reference.clone()),
                message!("The tractor's equipment type."),
            ),
            (
                "trailerType",
                opt(reference),
                message!("The trailer's equipment type."),
            ),
            (
                "stops",
                Type::list(stop.clone()),
                message!("Every stop, in route order."),
            ),
            (
                "commodities",
                Type::list(Type::Record(Arc::clone(commodity))),
                message!("Every commodity line."),
            ),
            (
                "pickup",
                opt(stop.clone()),
                message!("The first pickup stop."),
            ),
            ("delivery", opt(stop), message!("The last delivery stop.")),
            (
                "accessorials",
                Type::list(record(&[
                    (
                        "code",
                        Type::Text,
                        message!("The accessorial charge's catalog code."),
                    ),
                    (
                        "quantity",
                        Type::Number,
                        message!("How many were asked for."),
                    ),
                ])),
                message!(
                    "The accessorials a user entered on the shipment, never the lines a tariff or program generated."
                ),
            ),
        ]
    }

    fn freight_fields(&self) -> Vec<(&'static str, Type, &'static str)> {
        let temperature = record(&[
            (
                "min",
                opt(self.temperature.clone()),
                message!("The lowest allowed temperature."),
            ),
            (
                "max",
                opt(self.temperature.clone()),
                message!("The highest allowed temperature."),
            ),
        ]);
        vec![
            (
                "proNumber",
                Type::Text,
                message!("The shipment's PRO number."),
            ),
            (
                "status",
                self.enumeration("ShipmentStatus"),
                message!("Where the shipment is in its life."),
            ),
            (
                "weight",
                opt(self.weight.clone()),
                message!("The commodity lines' weight, else the weight entered on the shipment."),
            ),
            (
                "pieces",
                opt(Type::Int),
                message!("The pieces, when recorded."),
            ),
            (
                "distance",
                self.distance.clone(),
                message!("The route's distance, in the organisation's unit."),
            ),
            (
                "duration",
                self.duration.clone(),
                message!("The route's planned driving time."),
            ),
            (
                "linearFeet",
                self.length.clone(),
                message!("The trailer length the freight takes."),
            ),
            (
                "volume",
                self.volume.clone(),
                message!("The freight's volume."),
            ),
            (
                "density",
                opt(self.density.clone()),
                message!("The freight's density, when it has weight and volume."),
            ),
            (
                "hazmat",
                Type::Bool,
                message!("Whether any commodity is hazardous material."),
            ),
            (
                "reefer",
                Type::Bool,
                message!("Whether the shipment is temperature controlled."),
            ),
            (
                "temperature",
                temperature,
                message!("The temperature range, for reefer loads."),
            ),
            (
                "freightClass",
                opt(self.enumeration("FreightClass")),
                message!("The class of the heaviest commodity line."),
            ),
            (
                "highestClass",
                opt(self.enumeration("FreightClass")),
                message!("The highest class of any commodity line."),
            ),
        ]
    }

    fn lane() -> Type {
        let place = record(&[
            ("city", Type::Text, message!("The city.")),
            ("state", Type::Text, message!("The state or province code.")),
            ("zip", Type::Text, message!("The postal code.")),
            (
                "zip3",
                Type::Text,
                message!("The first three characters of the postal code."),
            ),
            ("country", Type::Text, message!("The ISO country code.")),
            (
                "zones",
                Type::list(Type::Text),
                message!("The rating zones the place is in."),
            ),
        ]);
        record(&[
            (
                "origin",
                place.clone(),
                message!("Where the shipment is picked up."),
            ),
            ("destination", place, message!("Where it is delivered.")),
            (
                "interstate",
                Type::Bool,
                message!("Whether origin and destination states differ."),
            ),
            (
                "crossBorder",
                Type::Bool,
                message!("Whether origin and destination countries differ."),
            ),
        ])
    }

    fn customer() -> Type {
        record(&[
            ("name", Type::Text, message!("The customer's name.")),
            ("code", Type::Text, message!("The customer's code.")),
        ])
    }

    fn fuel() -> Type {
        record(&[
            (
                "price",
                opt(Type::quantity_in(&unit("$/gal"))),
                message!("The fuel index price as of the rating date."),
            ),
            (
                "priceDate",
                opt(Type::Date),
                message!("The date of that price."),
            ),
            (
                "index",
                opt(Type::Text),
                message!("The index the price comes from."),
            ),
        ])
    }

    fn rating(&self) -> Type {
        record(&[
            (
                "date",
                Type::DateTime,
                message!("When the shipment is rated as of."),
            ),
            (
                "mode",
                self.enumeration("RatingMode"),
                message!("Sell to the customer or buy from the carrier."),
            ),
            (
                "sellTotal",
                opt(Type::money()),
                message!("The sell side's total, when rating the buy side."),
            ),
            (
                "currency",
                Type::Text,
                message!("The rating currency's ISO code."),
            ),
            (
                "holidays",
                Type::list(record(&[
                    ("date", Type::Date, message!("The holiday's date.")),
                    ("name", Type::Text, message!("The holiday's name.")),
                ])),
                message!(
                    "The organisation's holidays from the year before the rating date's year to the year after, in date order."
                ),
            ),
        ])
    }
}

fn opt(inner: Type) -> Type {
    Type::optional(inner)
}

fn unit(text: &str) -> Unit {
    Unit::parse(text).unwrap_or_default()
}

fn fields(list: &[(&str, Type, &'static str)]) -> Vec<FieldDef> {
    list.iter()
        .map(|(name, ty, doc)| FieldDef {
            name: (*name).to_owned(),
            ty: ty.clone(),
            doc: Some(Doc::message(doc)),
        })
        .collect()
}

fn record(list: &[(&str, Type, &'static str)]) -> Type {
    Type::Record(Arc::new(RecordType {
        name: None,
        fields: fields(list),
    }))
}
