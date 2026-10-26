//! Documents on the wire: values to JSON, and JSON to values by the type they must have.
//!
//! The TMS reads and writes these shapes (`ladingdoc`): money as `{amount, currency}`, measures
//! as `{value, unit}` in the unit Trenova stores (lb, ft, ft3, °F), instants as
//! `{at, zone}` with `at` in RFC 3339, dates as `YYYY-MM-DD`, decimals as strings, enums by
//! variant and references as `{id, code, label}`.

use std::collections::BTreeMap;
use std::rc::Rc;

use rust_decimal::Decimal;
use serde_json::{Map, Value as Json, json};

use super::value::{
    Reference, Unit, Value, convert, date_text, decimal_from_f64, decimal_text, instant_rfc3339,
    local,
};
use crate::lading::check::Program;
use crate::lading::schema;
use crate::lading::types::Type;
use crate::lading::values::{ZoneValue, parse_date, parse_instant, parse_time};

/// A value as JSON.
#[must_use]
pub fn to_json(value: &Value) -> Json {
    match value {
        Value::None | Value::Loop(..) | Value::Transaction(_) => Json::Null,
        Value::Bool(value) => Json::Bool(*value),
        Value::Int(value) | Value::Duration(value) => json!(value),
        Value::Decimal(value) | Value::Percent(value) => Json::String(decimal_text(*value)),
        Value::Text(text) => Json::String(text.to_string()),
        Value::Date(days) => Json::String(date_text(*days)),
        Value::Time(seconds) => Json::String(format!(
            "{:02}:{:02}:{:02}",
            seconds / 3600,
            seconds / 60 % 60,
            seconds % 60
        )),
        Value::DateTime(instant, zone) => {
            json!({ "at": instant_rfc3339(*instant, *zone), "zone": zone.name() })
        }
        Value::Zone(zone) => Json::String(zone.name()),
        Value::Money(amount, currency) => {
            json!({ "amount": format!("{:.2}", amount.round_dp(2)), "currency": currency.to_string() })
        }
        Value::Measure(amount, unit) => {
            let base = Unit::base(unit.dim);
            json!({ "value": decimal_text(convert(*amount, *unit, base)), "unit": base.name })
        }
        Value::List(items) => Json::Array(items.iter().map(to_json).collect()),
        Value::Map(fields) | Value::Struct(fields) | Value::Record(_, fields) => Json::Object(
            fields
                .iter()
                .filter(|(_, value)| !value.is_none())
                .map(|(key, value)| (key.clone(), to_json(value)))
                .collect::<Map<String, Json>>(),
        ),
        Value::Enum(_, variant) => Json::String((*variant).to_owned()),
        Value::Ref(reference) => {
            let mut object =
                json!({ "id": reference.id, "code": reference.code, "label": reference.label });
            if let Some(zone) = reference.zone {
                object["zone"] = Json::String(zone.name());
            }
            object
        }
        Value::Segment(tree, index) => Json::String(tree.segment_text(*index)),
        Value::Json(json) => json.as_ref().clone(),
    }
}

/// Reads JSON as a value of `ty`; the error names the path that does not fit.
#[allow(clippy::too_many_lines)]
pub fn from_json(program: &Program, json: &Json, ty: &Type, path: &str) -> Result<Value, String> {
    if json.is_null() {
        return Ok(Value::None);
    }
    let wrong = |what: &str| Err(format!("{path}: expected {what}, found {json}"));
    match ty {
        Type::Optional(inner) => from_json(program, json, inner, path),
        Type::Any | Type::Json => Ok(Value::Json(Rc::new(json.clone()))),
        Type::Text | Type::Code(_) => match json {
            Json::String(text) => Ok(Value::text(text)),
            Json::Number(number) => Ok(Value::text(number.to_string())),
            _ => wrong("text"),
        },
        Type::Int => match json
            .as_i64()
            .or_else(|| json.as_str().and_then(|text| text.parse().ok()))
        {
            Some(value) => Ok(Value::Int(value)),
            None => wrong("a whole number"),
        },
        Type::Decimal | Type::Percent => match decimal(json) {
            Some(value) if matches!(ty, Type::Percent) => Ok(Value::Percent(value)),
            Some(value) => Ok(Value::Decimal(value)),
            None => wrong("a number"),
        },
        Type::Bool => json
            .as_bool()
            .map_or_else(|| wrong("true or false"), |value| Ok(Value::Bool(value))),
        Type::Date => json
            .as_str()
            .and_then(parse_date)
            .map_or_else(|| wrong("a date"), |days| Ok(Value::Date(days))),
        Type::Time => json
            .as_str()
            .and_then(parse_time)
            .map_or_else(|| wrong("a time"), |seconds| Ok(Value::Time(seconds))),
        Type::DateTime => date_time(json).map_or_else(|| wrong("a date and time"), Ok),
        Type::Zone => json
            .as_str()
            .and_then(ZoneValue::parse)
            .map_or_else(|| wrong("a time zone"), |zone| Ok(Value::Zone(zone))),
        Type::Duration => json
            .as_i64()
            .map_or_else(|| wrong("seconds"), |seconds| Ok(Value::Duration(seconds))),
        Type::Money => {
            let (amount, currency) = match json {
                Json::Object(object) => (
                    object.get("amount").and_then(decimal),
                    object
                        .get("currency")
                        .and_then(Json::as_str)
                        .unwrap_or("USD")
                        .to_owned(),
                ),
                other => (decimal(other), "USD".to_owned()),
            };
            amount.map_or_else(
                || wrong("money"),
                |amount| Ok(Value::Money(amount, Rc::from(currency.as_str()))),
            )
        }
        Type::Measure(dim) => {
            let base = Unit::base(*dim);
            let (value, unit) = match json {
                Json::Object(object) => (
                    object.get("value").and_then(decimal),
                    object
                        .get("unit")
                        .and_then(Json::as_str)
                        .and_then(Unit::named)
                        .unwrap_or(base),
                ),
                other => (decimal(other), base),
            };
            match value {
                Some(value) if unit.dim == *dim => Ok(Value::Measure(value, unit)),
                _ => wrong(dim.name()),
            }
        }
        Type::Enum(name) => {
            let variant = json.as_str().and_then(|text| {
                schema::enumeration(name)?
                    .variants
                    .iter()
                    .find(|(variant, _)| *variant == text)
                    .map(|(variant, _)| *variant)
            });
            variant.map_or_else(|| wrong(name), |variant| Ok(Value::Enum(name, variant)))
        }
        Type::Ref(kind) => {
            let Json::Object(object) = json else {
                return wrong("a reference with id, code and label");
            };
            let text = |key: &str| {
                object
                    .get(key)
                    .and_then(Json::as_str)
                    .unwrap_or_default()
                    .to_owned()
            };
            Ok(Value::Ref(Rc::new(Reference {
                kind,
                id: text("id"),
                code: text("code"),
                label: text("label"),
                zone: object
                    .get("zone")
                    .and_then(Json::as_str)
                    .and_then(ZoneValue::parse),
            })))
        }
        Type::List(item) => {
            let Json::Array(items) = json else {
                return wrong("a list");
            };
            let mut values = Vec::with_capacity(items.len());
            for (index, item_json) in items.iter().enumerate() {
                values.push(from_json(
                    program,
                    item_json,
                    item,
                    &format!("{path}[{index}]"),
                )?);
            }
            Ok(Value::list(values))
        }
        Type::Map(value_ty) => {
            let Json::Object(object) = json else {
                return wrong("an object");
            };
            let mut fields = BTreeMap::new();
            for (key, value) in object {
                fields.insert(
                    key.clone(),
                    from_json(program, value, value_ty, &join(path, key))?,
                );
            }
            Ok(Value::Map(Rc::new(fields)))
        }
        Type::Record(name) => {
            let Json::Object(object) = json else {
                return wrong(name);
            };
            let Some(record) = schema::record(name) else {
                return wrong(name);
            };
            let mut fields = BTreeMap::new();
            for (key, value) in object {
                let Some(field) = record.field(key) else {
                    return Err(format!("{}: {name} has no field {key}", join(path, key)));
                };
                let field_ty = program.field_type(field);
                fields.insert(
                    field.name.to_owned(),
                    from_json(program, value, &field_ty, &join(path, key))?,
                );
            }
            for field in record.fields {
                if field.required && fields.get(field.name).is_none_or(Value::is_none) {
                    return Err(format!(
                        "{}: {name} needs {}",
                        join(path, field.name),
                        field.name
                    ));
                }
            }
            Ok(Value::Record(record.name, Rc::new(fields)))
        }
        Type::Struct(shape) => {
            let Json::Object(object) = json else {
                return wrong("an object");
            };
            let mut fields = BTreeMap::new();
            for (key, field_ty) in shape {
                let value = object.get(key).map_or(Ok(Value::None), |value| {
                    from_json(program, value, field_ty, &join(path, key))
                })?;
                fields.insert(key.clone(), value);
            }
            Ok(Value::Struct(Rc::new(fields)))
        }
        Type::Segment(_)
        | Type::SegmentId(_)
        | Type::Loop(_)
        | Type::LoopId(_)
        | Type::Transaction(_)
        | Type::Lambda(..)
        | Type::None => wrong("a value a document holds"),
    }
}

fn join(path: &str, key: &str) -> String {
    if path.is_empty() {
        key.to_owned()
    } else {
        format!("{path}.{key}")
    }
}

fn decimal(json: &Json) -> Option<Decimal> {
    match json {
        Json::String(text) => text.parse().ok(),
        Json::Number(number) => number
            .as_i64()
            .map(Decimal::from)
            .or_else(|| number.as_f64().and_then(decimal_from_f64)),
        _ => None,
    }
}

fn date_time(json: &Json) -> Option<Value> {
    let (text, zone) = match json {
        Json::String(text) => (text.as_str(), None),
        Json::Object(object) => (
            object.get("at").and_then(Json::as_str)?,
            object
                .get("zone")
                .and_then(Json::as_str)
                .and_then(ZoneValue::parse),
        ),
        _ => return None,
    };
    let (instant, written) = parse_instant(text)?;
    Some(Value::DateTime(instant, zone.unwrap_or(written)))
}

/// The local date and time of an instant, for display: `(days, seconds)`.
#[must_use]
pub fn local_parts(instant: i64, zone: ZoneValue) -> (i64, u32) {
    let (days, seconds, _) = local(instant, zone);
    (days, seconds)
}

#[cfg(test)]
mod tests {
    use serde_json::json;

    use super::{from_json, to_json};
    use crate::lading::check::Program;
    use crate::lading::types::Type;

    #[test]
    fn documents_round_trip() {
        let program = Program::default();
        let document = json!({
            "key": "L-1",
            "events": [{ "kind": "Delivered", "at": { "at": "2026-01-10T08:30:00-06:00", "zone": "America/Chicago" } }],
            "weight": { "value": "1000", "unit": "kg" }
        });
        let value =
            from_json(&program, &document, &Type::Record("shipmentStatus"), "").expect("reads");
        let back = to_json(&value);
        assert_eq!(back["events"][0]["at"]["at"], "2026-01-10T08:30:00-06:00");
        assert_eq!(back["weight"]["unit"], "lb");
        assert_eq!(back["weight"]["value"], "2204.62262185");
        let missing = from_json(
            &program,
            &json!({ "key": "L-1" }),
            &Type::Record("shipmentStatus"),
            "",
        );
        assert!(missing.expect_err("events is required").contains("events"));
    }
}
