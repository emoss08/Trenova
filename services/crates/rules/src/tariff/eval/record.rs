//! The record a program rates: the host's, decoded strictly against the schema, or the sample
//! record tests and the Studio's preview start from.
//!
//! The host sends `{"shipment": …, "lane": …, "customer": …, "fuel": …, "rating": …}` in the
//! wire form. Decoding is strict: a missing root, an unknown root or field, a value of the wrong
//! type or unit is an error naming its path, so a host bug surfaces instead of rating on a guess.
//!
//! The sample record is the schema's defaults: an empty `New` shipment with one pickup and one
//! delivery stop and no commodities, a blank lane, a sample customer, no fuel price, and a sell
//! rating on 2026-01-05 at noon UTC in US dollars. Tests change it with `given`, and build stops
//! and commodities with the test constructors ([`stop`], [`commodity`]).

use super::operators::duration_between;
use super::time::{Date, DateTime};
use super::value::{Record, Value};
use crate::message::{Arg, Message};
use crate::tariff::schema::{Schema, Settings};
use crate::tariff::units::{Base, Dim, Unit};
use crate::tariff::wire::{WireError, decode};
use rust_decimal::Decimal;
use serde_json::Value as Json;
use std::rc::Rc;

/// The schema roots a record holds, in order.
pub const ROOTS: [&str; 5] = ["shipment", "lane", "customer", "fuel", "rating"];

/// Decodes the host's record against the schema.
///
/// # Errors
///
/// When a root is missing or unknown, or a value does not fit its type.
pub fn decode_record(json: &Json, schema: &Schema) -> Result<Value, WireError> {
    let Some(object) = json.as_object() else {
        return Err(WireError {
            path: String::new(),
            message: Message::new(crate::message!(
                "The record must be an object holding shipment, lane, customer, fuel and rating"
            )),
        });
    };
    if let Some(unknown) = object.keys().find(|key| !ROOTS.contains(&key.as_str())) {
        return Err(WireError {
            path: unknown.clone(),
            message: Message::with(
                crate::message!("{0} is not a root of the record"),
                [Arg::from(unknown.clone())],
            ),
        });
    }
    let mut fields = Vec::with_capacity(ROOTS.len());
    for root in ROOTS {
        let (Some(ty), Some(value)) = (schema.root(root), object.get(root)) else {
            return Err(WireError {
                path: root.to_owned(),
                message: Message::with(crate::message!("The record has no {0}"), [Arg::from(root)]),
            });
        };
        fields.push((Rc::from(root), decode(value, ty, root)?));
    }
    Ok(Value::Record(Rc::new(Record::new(fields))))
}

fn unit(text: &str) -> Unit {
    Unit::parse(text).unwrap_or_default()
}

fn zero(text: &str) -> Value {
    Value::quantity(Decimal::ZERO, unit(text))
}

fn reference(code: &str, description: &str) -> Value {
    Value::record(vec![
        ("code", Value::text(code)),
        ("description", Value::text(description)),
    ])
}

/// The distance unit the organisation stores.
#[must_use]
pub fn distance_unit(settings: &Settings) -> Unit {
    settings
        .distance_unit
        .as_deref()
        .and_then(|text| Unit::parse(text).ok())
        .filter(|unit| unit.dim() == Dim::of(Base::Distance))
        .unwrap_or_else(|| unit("mi"))
}

/// The sample stop of a kind (`Pickup`, `Delivery`, `SplitPickup`, `SplitDelivery`). `sequence`
/// `None` leaves the sequence for [`settle`] to fill from the stop's position.
#[must_use]
pub fn stop(kind: &str, sequence: Option<i64>) -> Value {
    let location = Value::record(vec![
        ("city", Value::text("")),
        ("state", Value::text("")),
        ("zip", Value::text("")),
        ("zip3", Value::text("")),
        ("country", Value::text("")),
        ("timezone", Value::text("UTC")),
    ]);
    Value::record(vec![
        ("sequence", sequence.map_or(Value::None, Value::int)),
        ("kind", Value::variant(kind)),
        ("status", Value::variant("New")),
        ("appointment", Value::Bool(false)),
        (
            "window",
            Value::record(vec![("start", Value::None), ("end", Value::None)]),
        ),
        ("arrival", Value::None),
        ("departure", Value::None),
        ("dwell", Value::None),
        ("pieces", Value::None),
        ("weight", Value::None),
        ("location", location),
    ])
}

/// The sample commodity: unnamed, no class, `0 lb`, no pieces, no dimensions, not hazardous,
/// not stackable.
#[must_use]
pub fn commodity() -> Value {
    Value::record(vec![
        ("name", Value::text("")),
        ("freightClass", Value::None),
        ("weight", zero("lb")),
        ("pieces", Value::int(0)),
        (
            "dimensions",
            Value::record(vec![
                ("length", Value::None),
                ("width", Value::None),
                ("height", Value::None),
            ]),
        ),
        ("volume", Value::None),
        ("density", Value::None),
        ("hazmat", Value::Bool(false)),
        ("stackable", Value::Bool(false)),
    ])
}

fn place() -> Value {
    Value::record(vec![
        ("city", Value::text("")),
        ("state", Value::text("")),
        ("zip", Value::text("")),
        ("zip3", Value::text("")),
        ("country", Value::text("")),
        ("zones", Value::list(Vec::new())),
    ])
}

/// The sample record.
#[must_use]
pub fn sample(settings: &Settings) -> Value {
    let stops = vec![stop("Pickup", Some(1)), stop("Delivery", Some(2))];
    let shipment = Value::record(vec![
        ("proNumber", Value::text("SAMPLE")),
        ("status", Value::variant("New")),
        ("weight", Value::None),
        ("pieces", Value::None),
        (
            "distance",
            Value::quantity(Decimal::ZERO, distance_unit(settings)),
        ),
        ("duration", zero("h")),
        ("linearFeet", zero("ft")),
        ("volume", zero("ft3")),
        ("density", Value::None),
        ("hazmat", Value::Bool(false)),
        ("reefer", Value::Bool(false)),
        (
            "temperature",
            Value::record(vec![("min", Value::None), ("max", Value::None)]),
        ),
        ("freightClass", Value::None),
        ("highestClass", Value::None),
        ("serviceType", reference("STD", "Standard")),
        ("shipmentType", Value::None),
        ("tractorType", Value::None),
        ("trailerType", Value::None),
        ("pickup", stops.first().cloned().unwrap_or(Value::None)),
        ("delivery", stops.last().cloned().unwrap_or(Value::None)),
        ("stops", Value::list(stops)),
        ("commodities", Value::list(Vec::new())),
    ]);
    let lane = Value::record(vec![
        ("origin", place()),
        ("destination", place()),
        ("interstate", Value::Bool(false)),
        ("crossBorder", Value::Bool(false)),
    ]);
    let rating_date = Date::from_civil(2026, 1, 5)
        .and_then(|date| DateTime::from_local(date, 12, 0, 0, 0))
        .map_or(Value::None, Value::DateTime);
    Value::record(vec![
        ("shipment", shipment),
        ("lane", lane),
        (
            "customer",
            Value::record(vec![
                ("name", Value::text("Sample customer")),
                ("code", Value::text("SAMPLE")),
            ]),
        ),
        (
            "fuel",
            Value::record(vec![
                ("price", Value::None),
                ("priceDate", Value::None),
                ("index", Value::None),
            ]),
        ),
        (
            "rating",
            Value::record(vec![
                ("date", rating_date),
                ("mode", Value::variant("Sell")),
                ("sellTotal", Value::None),
                ("currency", Value::text("USD")),
            ]),
        ),
    ])
}

/// A stop with its dwell worked out from its arrival and departure, when both are present and
/// no dwell was given.
#[must_use]
pub fn with_dwell(stop: Value, dwell_given: bool) -> Value {
    let Value::Record(record) = &stop else {
        return stop;
    };
    if dwell_given {
        return stop;
    }
    let (Some(Value::DateTime(arrival)), Some(Value::DateTime(departure))) =
        (record.get("arrival"), record.get("departure"))
    else {
        return stop;
    };
    let Ok(dwell) = duration_between(*arrival, *departure) else {
        return stop;
    };
    let mut changed = (**record).clone();
    changed.set("dwell", dwell);
    Value::Record(Rc::new(changed))
}

fn kind_of(stop: &Value) -> Option<Rc<str>> {
    match stop {
        Value::Record(record) => match record.get("kind") {
            Some(Value::Enum(kind)) => Some(Rc::clone(kind)),
            _ => None,
        },
        _ => None,
    }
}

/// Settles the shipment after a test's `given`s: each stop without a sequence takes its 1-based
/// position, and `pickup` and `delivery` become the first pickup-kind and the last
/// delivery-kind stop, unless a `given` set them.
#[must_use]
pub fn settle(record: &Value, pickup_given: bool, delivery_given: bool) -> Value {
    let Value::Record(roots) = record else {
        return record.clone();
    };
    let Some(Value::Record(shipment)) = roots.get("shipment") else {
        return record.clone();
    };
    let mut shipment = (**shipment).clone();
    let stops: Vec<Value> = match shipment.get("stops") {
        Some(Value::List(stops)) => stops
            .iter()
            .enumerate()
            .map(|(index, stop)| numbered(stop, index))
            .collect(),
        _ => Vec::new(),
    };
    if !pickup_given {
        let first = stops
            .iter()
            .find(|stop| matches!(kind_of(stop).as_deref(), Some("Pickup" | "SplitPickup")));
        shipment.set("pickup", first.cloned().unwrap_or(Value::None));
    }
    if !delivery_given {
        let last = stops
            .iter()
            .rev()
            .find(|stop| matches!(kind_of(stop).as_deref(), Some("Delivery" | "SplitDelivery")));
        shipment.set("delivery", last.cloned().unwrap_or(Value::None));
    }
    for field in ["pickup", "delivery"] {
        if let Some(stop) = shipment.get(field).cloned() {
            shipment.set(field, numbered(&stop, 0));
        }
    }
    shipment.set("stops", Value::list(stops));
    let mut roots = (**roots).clone();
    roots.set("shipment", Value::Record(Rc::new(shipment)));
    Value::Record(Rc::new(roots))
}

fn numbered(stop: &Value, index: usize) -> Value {
    match stop {
        Value::Record(record) if matches!(record.get("sequence"), Some(Value::None)) => {
            let mut changed = (**record).clone();
            changed.set(
                "sequence",
                Value::Int(Decimal::from(index).saturating_add(Decimal::ONE)),
            );
            Value::Record(Rc::new(changed))
        }
        other => other.clone(),
    }
}

#[cfg(test)]
mod tests {
    use super::{decode_record, sample};
    use crate::tariff::schema::{Schema, Settings};
    use crate::tariff::wire::encode;

    #[test]
    fn the_sample_record_decodes_against_the_schema() {
        let settings = Settings::default();
        let schema = Schema::new(&settings);
        let json = encode(&sample(&settings));
        let decoded = decode_record(&json, &schema).expect("the sample fits the schema");
        assert_eq!(encode(&decoded), json);
    }

    #[test]
    fn unknown_roots_and_fields_are_refused() {
        let settings = Settings::default();
        let schema = Schema::new(&settings);
        let mut json = encode(&sample(&settings));
        if let Some(object) = json.as_object_mut() {
            object.insert(String::from("extra"), serde_json::Value::Null);
        }
        assert_eq!(
            decode_record(&json, &schema)
                .map_err(|e| e.path)
                .err()
                .as_deref(),
            Some("extra")
        );
    }
}
