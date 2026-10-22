//! Tariff values on the wire: the spec's "Wire values" table, both ways.
//!
//! | Value | Wire form |
//! |---|---|
//! | `Bool`, `Text` | JSON `true`/`false`, a JSON string |
//! | `Int`, `Number` | `{"$n": "1200"}` |
//! | Money, quantities, rates, percents | `{"$q": "2.35", "u": "$/mi"}`, `{"$q": "18", "u": "%"}` |
//! | `Date`, `DateTime` | `{"$date": "2026-10-22"}`, `{"$date": "2026-10-22T08:00:00-05:00"}` |
//! | Enum variants | a JSON string, read as the declared enum |
//! | `none` | `null` |
//! | Lists, records | JSON arrays and objects (record keys never start with `$`) |
//!
//! [`decode`] reads a value against the type it must have and refuses anything else, naming the
//! JSON path: a quantity whose unit measures the wrong dimension, a variant the enum lacks, a
//! record with a missing required field or an unknown one, `null` where the type is not
//! optional, a fraction where a whole number is needed. Numbers are decimal strings read
//! exactly, never through a float. A record field whose type is optional may be left out.
//!
//! [`encode`] is canonical: decimals normalised (`"2.5"`, never `"2.50"`), units written as
//! `Unit`'s `Display`, date-times with seconds and their offset, records' keys in order. The
//! checker writes input defaults in this same form, and the bill and the trace use it.

use crate::message::{Arg, Message};
use crate::tariff::eval::time::{Date, DateTime};
use crate::tariff::eval::value::{Record, Value, format_decimal};
use crate::tariff::types::{RecordType, Type};
use crate::tariff::units::Unit;
use rust_decimal::Decimal;
use serde_json::{Map, Value as Json};
use std::rc::Rc;

/// Why a wire value does not fit its type.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WireError {
    /// Where in the JSON: `shipment.stops[0].weight`.
    pub path: String,
    /// What is wrong.
    pub message: Message,
}

impl WireError {
    fn new(path: &str, template: &'static str, args: impl IntoIterator<Item = Arg>) -> Self {
        Self {
            path: path.to_owned(),
            message: Message::with(template, args),
        }
    }
}

/// How deep a wire value may nest; deeper is refused rather than walked.
const MAX_DEPTH: usize = 64;

/// Reads `json` as a value of type `ty`. `path` names the value in errors (`shipment`,
/// `input rate`).
///
/// # Errors
///
/// When the value does not fit the type.
pub fn decode(json: &Json, ty: &Type, path: &str) -> Result<Value, WireError> {
    decode_at(json, ty, path, 0)
}

fn decode_at(json: &Json, ty: &Type, path: &str, depth: usize) -> Result<Value, WireError> {
    if depth > MAX_DEPTH {
        return Err(WireError::new(
            path,
            crate::message!("The value nests deeper than {0} levels"),
            [Arg::from(MAX_DEPTH)],
        ));
    }
    if let Type::Optional(inner) = ty {
        return if json.is_null() {
            Ok(Value::None)
        } else {
            decode_at(json, inner, path, depth)
        };
    }
    if json.is_null() {
        return Err(expected(path, ty, json));
    }
    match ty {
        Type::Bool => json
            .as_bool()
            .map(Value::Bool)
            .ok_or_else(|| expected(path, ty, json)),
        Type::Text => json
            .as_str()
            .map(Value::text)
            .ok_or_else(|| expected(path, ty, json)),
        Type::Int => {
            let value = number(json, ty, path)?;
            if !value.fract().is_zero() {
                return Err(WireError::new(
                    path,
                    crate::message!("{0} is not a whole number"),
                    [Arg::from(format_decimal(value))],
                ));
            }
            Ok(Value::Int(value.trunc().normalize()))
        }
        Type::Number => Ok(Value::Number(number(json, ty, path)?)),
        Type::Percent => {
            let (value, unit) = quantity(json, ty, path)?;
            if unit.to_string() != "%" {
                return Err(WireError::new(
                    path,
                    crate::message!("A percent is written with the unit \"%\", not \"{0}\""),
                    [Arg::from(unit.to_string())],
                ));
            }
            Ok(Value::percent(value))
        }
        Type::Quantity(expected_type) => {
            let (value, unit) = quantity(json, ty, path)?;
            if unit.dim() != expected_type.dim || unit.dim().is_none() {
                return Err(WireError::new(
                    path,
                    crate::message!("{0} is not a unit of {1}"),
                    [Arg::from(unit.to_string()), Arg::from(ty.to_string())],
                ));
            }
            Ok(Value::quantity(value, unit))
        }
        Type::Date => date(json, ty, path).map(Value::Date),
        Type::DateTime => date_time(json, ty, path).map(Value::DateTime),
        Type::Enum(enumeration) => {
            let text = json.as_str().ok_or_else(|| expected(path, ty, json))?;
            if enumeration.has(text) {
                Ok(Value::variant(text))
            } else {
                Err(WireError::new(
                    path,
                    crate::message!("\"{0}\" is not a {1}"),
                    [Arg::from(text), Arg::from(ty.to_string())],
                ))
            }
        }
        Type::List(item) => {
            let items = json.as_array().ok_or_else(|| expected(path, ty, json))?;
            items
                .iter()
                .enumerate()
                .map(|(index, entry)| {
                    decode_at(entry, item, &format!("{path}[{index}]"), depth + 1)
                })
                .collect::<Result<Vec<_>, _>>()
                .map(Value::list)
        }
        Type::Record(record) => decode_record(json, ty, record, path, depth),
        Type::Optional(_) | Type::Never | Type::Error => Err(WireError::new(
            path,
            crate::message!("No value has the type {0}"),
            [Arg::from(ty.to_string())],
        )),
    }
}

fn decode_record(
    json: &Json,
    ty: &Type,
    record: &RecordType,
    path: &str,
    depth: usize,
) -> Result<Value, WireError> {
    let object = json.as_object().ok_or_else(|| expected(path, ty, json))?;
    if let Some(unknown) = object.keys().find(|key| record.field(key).is_none()) {
        return Err(WireError::new(
            &join(path, unknown),
            crate::message!("{0} is not a field of {1}"),
            [Arg::from(unknown.clone()), Arg::from(ty.to_string())],
        ));
    }
    let mut fields = Vec::with_capacity(record.fields.len());
    for field in &record.fields {
        let at = join(path, &field.name);
        let value = match object.get(&field.name) {
            Some(entry) => decode_at(entry, &field.ty, &at, depth + 1)?,
            None if field.ty.is_optional() => Value::None,
            None => {
                return Err(WireError::new(
                    &at,
                    crate::message!("The required field {0} is missing"),
                    [Arg::from(field.name.clone())],
                ));
            }
        };
        fields.push((Rc::from(field.name.as_str()), value));
    }
    Ok(Value::Record(Rc::new(Record::new(fields))))
}

fn join(path: &str, field: &str) -> String {
    if path.is_empty() {
        field.to_owned()
    } else {
        format!("{path}.{field}")
    }
}

/// What a JSON value is, as a phrase.
fn json_kind(json: &Json) -> Message {
    Message::new(match json {
        Json::Null => crate::message!("null"),
        Json::Bool(_) => crate::message!("true or false"),
        Json::Number(_) => crate::message!("a JSON number"),
        Json::String(_) => crate::message!("text"),
        Json::Array(_) => crate::message!("a list"),
        Json::Object(object) if object.contains_key("$n") => crate::message!("a number"),
        Json::Object(object) if object.contains_key("$q") => crate::message!("a quantity"),
        Json::Object(object) if object.contains_key("$date") => crate::message!("a date"),
        Json::Object(_) => crate::message!("a record"),
    })
}

fn expected(path: &str, ty: &Type, json: &Json) -> WireError {
    WireError::new(
        path,
        crate::message!("Expected {0}, got {1}"),
        [Arg::from(ty.to_string()), Arg::from(json_kind(json))],
    )
}

/// The one tagged member of `{"$x": …}`, refusing any other key.
fn tagged<'a>(
    json: &'a Json,
    tag: &str,
    extra: Option<&str>,
    ty: &Type,
    path: &str,
) -> Result<&'a Map<String, Json>, WireError> {
    let object = json.as_object().ok_or_else(|| expected(path, ty, json))?;
    if !object.contains_key(tag) {
        return Err(expected(path, ty, json));
    }
    if let Some(unknown) = object
        .keys()
        .find(|key| key.as_str() != tag && Some(key.as_str()) != extra)
    {
        return Err(WireError::new(
            path,
            crate::message!("{0} does not belong in this value"),
            [Arg::from(unknown.clone())],
        ));
    }
    Ok(object)
}

fn decimal_text(text: &Json, path: &str) -> Result<Decimal, WireError> {
    let Some(text) = text.as_str() else {
        return Err(WireError::new(
            path,
            crate::message!("A number is written as a decimal string, such as \"12.5\""),
            [],
        ));
    };
    parse_plain_decimal(text).ok_or_else(|| {
        WireError::new(
            path,
            crate::message!("\"{0}\" is not a decimal number"),
            [Arg::from(text)],
        )
    })
}

/// Reads a plain decimal: an optional `-`, digits and an optional fraction. No exponent, no
/// `+`, no spaces, at most 28 significant digits.
#[must_use]
pub fn parse_plain_decimal(text: &str) -> Option<Decimal> {
    let digits = text.strip_prefix('-').unwrap_or(text);
    let (whole, fraction) = digits.split_once('.').unwrap_or((digits, ""));
    let valid = !whole.is_empty()
        && whole.bytes().all(|byte| byte.is_ascii_digit())
        && (fraction.is_empty() || fraction.bytes().all(|byte| byte.is_ascii_digit()))
        && !(digits.contains('.') && fraction.is_empty());
    if !valid {
        return None;
    }
    text.parse::<Decimal>().ok().map(|value| value.normalize())
}

fn number(json: &Json, ty: &Type, path: &str) -> Result<Decimal, WireError> {
    let object = tagged(json, "$n", None, ty, path)?;
    decimal_text(object.get("$n").unwrap_or(&Json::Null), path)
}

fn quantity(json: &Json, ty: &Type, path: &str) -> Result<(Decimal, Unit), WireError> {
    let object = tagged(json, "$q", Some("u"), ty, path)?;
    let value = decimal_text(object.get("$q").unwrap_or(&Json::Null), path)?;
    let Some(text) = object.get("u").and_then(Json::as_str) else {
        return Err(WireError::new(
            path,
            crate::message!("A quantity needs its unit under \"u\""),
            [],
        ));
    };
    let unit = Unit::parse(text).map_err(|_| {
        WireError::new(
            path,
            crate::message!("\"{0}\" is not a unit"),
            [Arg::from(text)],
        )
    })?;
    Ok((value, unit))
}

fn date_text<'a>(json: &'a Json, ty: &Type, path: &str) -> Result<&'a str, WireError> {
    let object = tagged(json, "$date", None, ty, path)?;
    object
        .get("$date")
        .and_then(Json::as_str)
        .ok_or_else(|| expected(path, ty, json))
}

fn date(json: &Json, ty: &Type, path: &str) -> Result<Date, WireError> {
    let text = date_text(json, ty, path)?;
    Date::parse(text).ok_or_else(|| {
        WireError::new(
            path,
            crate::message!("\"{0}\" is not a date (YYYY-MM-DD)"),
            [Arg::from(text)],
        )
    })
}

fn date_time(json: &Json, ty: &Type, path: &str) -> Result<DateTime, WireError> {
    let text = date_text(json, ty, path)?;
    DateTime::parse(text).ok_or_else(|| {
        WireError::new(
            path,
            crate::message!(
                "\"{0}\" is not a date-time with its offset (YYYY-MM-DDTHH:MM:SS±HH:MM)"
            ),
            [Arg::from(text)],
        )
    })
}

/// Writes a value in its canonical wire form.
#[must_use]
pub fn encode(value: &Value) -> Json {
    match value {
        Value::None => Json::Null,
        Value::Bool(flag) => Json::Bool(*flag),
        Value::Int(number) | Value::Number(number) => {
            tag("$n", Json::String(format_decimal(*number)))
        }
        Value::Quantity(quantity) => {
            let mut object = Map::new();
            object.insert(
                String::from("$q"),
                Json::String(format_decimal(quantity.value)),
            );
            object.insert(String::from("u"), Json::String(quantity.unit.to_string()));
            Json::Object(object)
        }
        Value::Text(text) => Json::String(text.to_string()),
        Value::Date(date) => tag("$date", Json::String(date.iso())),
        Value::DateTime(at) => tag("$date", Json::String(at.iso())),
        Value::List(items) => Json::Array(items.iter().map(encode).collect()),
        Value::Record(record) => Json::Object(
            record
                .fields
                .iter()
                .map(|(name, value)| (name.to_string(), encode(value)))
                .collect(),
        ),
        Value::Enum(variant) => Json::String(variant.to_string()),
        Value::Table(code) => Json::String(code.to_string()),
    }
}

fn tag(name: &str, value: Json) -> Json {
    let mut object = Map::new();
    object.insert(name.to_owned(), value);
    Json::Object(object)
}

#[cfg(test)]
mod tests {
    use super::{decode, encode};
    use crate::tariff::types::Type;
    use crate::tariff::units::Unit;
    use serde_json::json;

    #[test]
    fn round_trips_a_rate() {
        let ty = Type::quantity_in(&Unit::parse("$/mi").expect("a unit"));
        let value = decode(&json!({"$q": "2.350", "u": "$/mi"}), &ty, "rate").expect("a rate");
        assert_eq!(encode(&value), json!({"$q": "2.35", "u": "$/mi"}));
    }

    #[test]
    fn refuses_the_wrong_dimension() {
        let ty = Type::quantity_in(&Unit::parse("$/mi").expect("a unit"));
        let error = decode(&json!({"$q": "2", "u": "$/cwt"}), &ty, "rate").expect_err("refused");
        assert_eq!(error.message.english(), "$/cwt is not a unit of Money/mi");
        assert_eq!(error.path, "rate");
    }
}
