//! Runtime values: exact numbers, measures with units, money, dates and instants in zones,
//! lists, records, references, X12 nodes and JSON.

use std::cmp::Ordering;
use std::collections::BTreeMap;
use std::fmt::Write as _;
use std::rc::Rc;

use rust_decimal::Decimal;
use rust_decimal::prelude::{FromPrimitive, ToPrimitive};

use super::tree::{Node, Tree};
use crate::lading::types::Dim;
use crate::lading::tz::civil_from_days;
use crate::lading::values::ZoneValue;

/// A unit of a measure.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Unit {
    /// How it is written.
    pub name: &'static str,
    /// What it measures.
    pub dim: Dim,
    /// How many of the dimension's base unit (lb, ft, ft3) one is, scaled by 10^12; unused for
    /// temperatures.
    factor: i64,
}

const fn unit(name: &'static str, dim: Dim, factor: i64) -> Unit {
    Unit { name, dim, factor }
}

/// Every unit.
pub const UNITS: &[Unit] = &[
    unit("lb", Dim::Weight, 1_000_000_000_000),
    unit("kg", Dim::Weight, 2_204_622_621_850),
    unit("oz", Dim::Weight, 62_500_000_000),
    unit("t", Dim::Weight, 2_204_622_621_850_000),
    unit("ton", Dim::Weight, 2_000_000_000_000_000),
    unit("ft", Dim::Length, 1_000_000_000_000),
    unit("in", Dim::Length, 83_333_333_333),
    unit("yd", Dim::Length, 3_000_000_000_000),
    unit("mi", Dim::Length, 5_280_000_000_000_000),
    unit("cm", Dim::Length, 32_808_398_950),
    unit("m", Dim::Length, 3_280_839_895_013),
    unit("km", Dim::Length, 3_280_839_895_013_123),
    unit("ft3", Dim::Volume, 1_000_000_000_000),
    unit("m3", Dim::Volume, 35_314_666_721_489),
    unit("gal", Dim::Volume, 133_680_555_556),
    unit("l", Dim::Volume, 35_314_666_721),
    unit("°F", Dim::Temperature, 0),
    unit("°C", Dim::Temperature, 0),
];

impl Unit {
    /// A unit by name.
    #[must_use]
    pub fn named(name: &str) -> Option<Self> {
        UNITS.iter().copied().find(|unit| unit.name == name)
    }

    /// The unit Trenova stores a dimension in.
    #[must_use]
    pub fn base(dim: Dim) -> Self {
        let name = match dim {
            Dim::Weight => "lb",
            Dim::Length => "ft",
            Dim::Volume => "ft3",
            Dim::Temperature => "°F",
        };
        Self::named(name).unwrap_or(UNITS[0])
    }

    /// The unit an X12 unit code stands for in a dimension.
    #[must_use]
    pub fn from_code(dim: Dim, code: &str) -> Option<Self> {
        let name = match (dim, code.trim()) {
            (Dim::Weight, "L") => "lb",
            (Dim::Weight, "K") => "kg",
            (Dim::Weight, "E") => "t",
            (Dim::Length, "F") => "ft",
            (Dim::Length, "I") => "in",
            (Dim::Length, "M") => "m",
            (Dim::Volume, "E") => "ft3",
            (Dim::Volume, "X") => "m3",
            (Dim::Volume, "G") => "gal",
            (Dim::Temperature, "F") => "°F",
            (Dim::Temperature, "C") => "°C",
            _ => return None,
        };
        Self::named(name)
    }
}

/// Converts a measure between units of one dimension.
#[must_use]
pub fn convert(value: Decimal, from: Unit, to: Unit) -> Decimal {
    if from == to {
        return value;
    }
    if from.dim == Dim::Temperature {
        let nine = Decimal::from(9);
        let five = Decimal::from(5);
        let thirty_two = Decimal::from(32);
        return match (from.name, to.name) {
            ("°C", "°F") => value * nine / five + thirty_two,
            ("°F", "°C") => (value - thirty_two) * five / nine,
            _ => value,
        };
    }
    let ratio = Decimal::from(from.factor) / Decimal::from(to.factor);
    value
        .checked_mul(ratio)
        .unwrap_or(value)
        .round_dp(9)
        .normalize()
}

/// A Trenova record a lookup found.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Reference {
    /// The kind.
    pub kind: &'static str,
    /// Its ID.
    pub id: String,
    /// The partner's code it was found by.
    pub code: String,
    /// Its name.
    pub label: String,
    /// Its time zone, for a location.
    pub zone: Option<ZoneValue>,
}

/// A value.
#[derive(Debug, Clone)]
pub enum Value {
    /// `none`.
    None,
    /// A Bool.
    Bool(bool),
    /// An Int.
    Int(i64),
    /// A Decimal.
    Decimal(Decimal),
    /// Text.
    Text(Rc<str>),
    /// A date, days since 1970-01-01.
    Date(i64),
    /// A time of day, seconds since midnight.
    Time(u32),
    /// An instant, seconds since the epoch, and the zone it is in.
    DateTime(i64, ZoneValue),
    /// A zone.
    Zone(ZoneValue),
    /// A length of time, in seconds.
    Duration(i64),
    /// A percent, as written (`5%` is 5).
    Percent(Decimal),
    /// Money: the amount and the currency.
    Money(Decimal, Rc<str>),
    /// A measure in a unit.
    Measure(Decimal, Unit),
    /// A list.
    List(Rc<Vec<Value>>),
    /// Text keys to values.
    Map(Rc<BTreeMap<String, Value>>),
    /// A record of the schema, by name.
    Record(&'static str, Rc<BTreeMap<String, Value>>),
    /// A record written in place, or a composite element.
    Struct(Rc<BTreeMap<String, Value>>),
    /// An enum variant: the enum and the variant.
    Enum(&'static str, &'static str),
    /// A lookup's record.
    Ref(Rc<Reference>),
    /// A segment of a message read.
    Segment(Rc<Tree>, usize),
    /// A loop instance of a message read.
    Loop(Rc<Tree>, Node),
    /// The top of a transaction read.
    Transaction(Rc<Tree>),
    /// A JSON value.
    Json(Rc<serde_json::Value>),
}

impl Value {
    /// Text from a string.
    #[must_use]
    pub fn text(text: impl AsRef<str>) -> Self {
        Self::Text(Rc::from(text.as_ref()))
    }

    /// A list.
    #[must_use]
    pub fn list(items: Vec<Self>) -> Self {
        Self::List(Rc::new(items))
    }

    /// Whether it is none.
    #[must_use]
    pub fn is_none(&self) -> bool {
        matches!(self, Self::None)
    }

    /// The text, for text.
    #[must_use]
    pub fn as_text(&self) -> Option<&str> {
        match self {
            Self::Text(text) => Some(text),
            _ => None,
        }
    }

    /// A number as a decimal.
    #[must_use]
    pub fn as_decimal(&self) -> Option<Decimal> {
        match self {
            Self::Int(value) => Some(Decimal::from(*value)),
            Self::Decimal(value) | Self::Percent(value) => Some(*value),
            _ => None,
        }
    }

    /// The items of a list (none and a missing list are empty).
    #[must_use]
    pub fn items(&self) -> &[Self] {
        match self {
            Self::List(items) => items,
            _ => &[],
        }
    }

    /// A field of a record or struct.
    #[must_use]
    pub fn field(&self, name: &str) -> Self {
        match self {
            Self::Record(_, fields) | Self::Struct(fields) | Self::Map(fields) => {
                fields.get(name).cloned().unwrap_or(Self::None)
            }
            _ => Self::None,
        }
    }

    /// The truth of a condition: none is false.
    #[must_use]
    pub fn truthy(&self) -> bool {
        matches!(self, Self::Bool(true))
    }
}

/// Two values are equal: numbers by value, measures across units, instants by instant.
#[must_use]
pub fn equal(left: &Value, right: &Value) -> bool {
    match (left, right) {
        (Value::None, Value::None) => true,
        (Value::None, _) | (_, Value::None) => false,
        (Value::Bool(a), Value::Bool(b)) => a == b,
        (Value::Text(a), Value::Text(b)) => a == b,
        (Value::Date(a), Value::Date(b))
        | (Value::Duration(a), Value::Duration(b))
        | (Value::DateTime(a, _), Value::DateTime(b, _)) => a == b,
        (Value::Time(a), Value::Time(b)) => a == b,
        (Value::Zone(a), Value::Zone(b)) => a.name() == b.name(),
        (Value::Money(a, x), Value::Money(b, y)) => a == b && x == y,
        (Value::Measure(..), Value::Measure(..)) => compare(left, right) == Some(Ordering::Equal),
        (Value::Enum(a, x), Value::Enum(b, y)) => a == b && x == y,
        (Value::Ref(a), Value::Ref(b)) => a.kind == b.kind && a.id == b.id,
        (Value::List(a), Value::List(b)) => {
            a.len() == b.len() && a.iter().zip(b.iter()).all(|(x, y)| equal(x, y))
        }
        (Value::Map(a), Value::Map(b))
        | (Value::Struct(a), Value::Struct(b))
        | (Value::Record(_, a), Value::Record(_, b)) => {
            a.len() == b.len()
                && a.iter()
                    .zip(b.iter())
                    .all(|((k, x), (l, y))| k == l && equal(x, y))
        }
        (Value::Segment(a, x), Value::Segment(b, y)) => Rc::ptr_eq(a, b) && x == y,
        (Value::Json(a), Value::Json(b)) => a == b,
        _ => match (left.as_decimal(), right.as_decimal()) {
            (Some(a), Some(b)) => a == b,
            _ => false,
        },
    }
}

/// The order of two values, when they have one: none has none.
#[must_use]
pub fn compare(left: &Value, right: &Value) -> Option<Ordering> {
    match (left, right) {
        (Value::Text(a), Value::Text(b)) => Some(a.cmp(b)),
        (Value::Date(a), Value::Date(b))
        | (Value::Duration(a), Value::Duration(b))
        | (Value::DateTime(a, _), Value::DateTime(b, _)) => Some(a.cmp(b)),
        (Value::Time(a), Value::Time(b)) => Some(a.cmp(b)),
        (Value::Money(a, x), Value::Money(b, y)) if x == y => Some(a.cmp(b)),
        (Value::Measure(a, x), Value::Measure(b, y)) if x.dim == y.dim => {
            Some(convert(*a, *x, Unit::base(x.dim)).cmp(&convert(*b, *y, Unit::base(y.dim))))
        }
        _ => Some(left.as_decimal()?.cmp(&right.as_decimal()?)),
    }
}

/// A decimal as written: no trailing zeros, no exponent.
#[must_use]
pub fn decimal_text(value: Decimal) -> String {
    let normalized = value.normalize();
    normalized.to_string()
}

/// A date as `2026-01-10`.
#[must_use]
pub fn date_text(days: i64) -> String {
    let (year, month, day) = civil_from_days(days);
    format!("{year:04}-{month:02}-{day:02}")
}

/// A time as `14:30` or `14:30:15`.
#[must_use]
pub fn time_text(seconds: u32) -> String {
    let (hours, minutes, rest) = (seconds / 3600, seconds / 60 % 60, seconds % 60);
    if rest == 0 {
        format!("{hours:02}:{minutes:02}")
    } else {
        format!("{hours:02}:{minutes:02}:{rest:02}")
    }
}

/// The local date and time of an instant in its zone, and the offset.
#[must_use]
pub fn local(instant: i64, zone: ZoneValue) -> (i64, u32, i32) {
    let offset = zone.offset_at(instant);
    let local = instant + i64::from(offset) * 60;
    let days = local.div_euclid(86_400);
    let seconds = u32::try_from(local.rem_euclid(86_400)).unwrap_or(0);
    (days, seconds, offset)
}

/// An instant as RFC 3339 in its zone's offset.
#[must_use]
pub fn instant_rfc3339(instant: i64, zone: ZoneValue) -> String {
    let (days, seconds, offset) = local(instant, zone);
    let mut text = format!(
        "{}T{:02}:{:02}:{:02}",
        date_text(days),
        seconds / 3600,
        seconds / 60 % 60,
        seconds % 60
    );
    if offset == 0 {
        text.push('Z');
    } else {
        let sign = if offset < 0 { '-' } else { '+' };
        let offset = offset.unsigned_abs();
        let _ = write!(text, "{sign}{:02}:{:02}", offset / 60, offset % 60);
    }
    text
}

/// A duration as `2 h`, `90 min` or `45 s`.
#[must_use]
pub fn duration_text(seconds: i64) -> String {
    if seconds != 0 && seconds % 86_400 == 0 {
        format!("{} d", seconds / 86_400)
    } else if seconds != 0 && seconds % 3600 == 0 {
        format!("{} h", seconds / 3600)
    } else if seconds % 60 == 0 {
        format!("{} min", seconds / 60)
    } else {
        format!("{seconds} s")
    }
}

/// Money as `$1,850.00` for dollars, `1,850.00 CAD` otherwise.
#[must_use]
pub fn money_text(amount: Decimal, currency: &str) -> String {
    let rounded = amount.round_dp(2);
    let negative = rounded.is_sign_negative() && !rounded.is_zero();
    let text = format!("{:.2}", rounded.abs());
    let (whole, cents) = text.split_once('.').unwrap_or((&text, "00"));
    let mut grouped = String::new();
    for (index, digit) in whole.chars().enumerate() {
        if index > 0 && (whole.len() - index) % 3 == 0 {
            grouped.push(',');
        }
        grouped.push(digit);
    }
    let sign = if negative { "-" } else { "" };
    if currency == "USD" {
        format!("{sign}${grouped}.{cents}")
    } else {
        format!("{sign}{grouped}.{cents} {currency}")
    }
}

/// A value written as text, as interpolation and `text()` write it.
#[must_use]
pub fn display(value: &Value) -> String {
    match value {
        Value::None | Value::Loop(..) | Value::Transaction(_) => String::new(),
        Value::Bool(value) => value.to_string(),
        Value::Int(value) => value.to_string(),
        Value::Decimal(value) => decimal_text(*value),
        Value::Text(text) => text.to_string(),
        Value::Date(days) => date_text(*days),
        Value::Time(seconds) => time_text(*seconds),
        Value::DateTime(instant, zone) => {
            let (days, seconds, _) = local(*instant, *zone);
            format!("{} {} {}", date_text(days), time_text(seconds), zone.name())
        }
        Value::Zone(zone) => zone.name(),
        Value::Duration(seconds) => duration_text(*seconds),
        Value::Percent(value) => format!("{}%", decimal_text(*value)),
        Value::Money(amount, currency) => money_text(*amount, currency),
        Value::Measure(amount, unit) => format!("{} {}", grouped(*amount), unit.name),
        Value::List(items) => items.iter().map(display).collect::<Vec<_>>().join(", "),
        Value::Map(fields) | Value::Struct(fields) | Value::Record(_, fields) => fields
            .iter()
            .map(|(key, value)| format!("{key}: {}", display(value)))
            .collect::<Vec<_>>()
            .join(", "),
        Value::Enum(_, variant) => (*variant).to_owned(),
        Value::Ref(reference) => reference.label.clone(),
        Value::Segment(tree, index) => tree.segment_text(*index),
        Value::Json(json) => match json.as_ref() {
            serde_json::Value::String(text) => text.clone(),
            other => other.to_string(),
        },
    }
}

/// A number with thousands separators: `45,000`.
fn grouped(value: Decimal) -> String {
    let text = decimal_text(value);
    let (whole, fraction) = text
        .split_once('.')
        .map_or((text.as_str(), None), |(a, b)| (a, Some(b)));
    let (sign, digits) = whole
        .strip_prefix('-')
        .map_or(("", whole), |rest| ("-", rest));
    let mut out = String::from(sign);
    for (index, digit) in digits.chars().enumerate() {
        if index > 0 && (digits.len() - index) % 3 == 0 {
            out.push(',');
        }
        out.push(digit);
    }
    if let Some(fraction) = fraction {
        out.push('.');
        out.push_str(fraction);
    }
    out
}

/// A decimal from an f64 when JSON carries a number.
#[must_use]
pub fn decimal_from_f64(value: f64) -> Option<Decimal> {
    Decimal::from_f64(value)
}

/// A decimal as an i64 when it is whole.
#[must_use]
pub fn whole(value: Decimal) -> Option<i64> {
    if value.fract().is_zero() {
        value.to_i64()
    } else {
        None
    }
}

#[cfg(test)]
mod tests {
    use rust_decimal::Decimal;

    use super::{Unit, Value, compare, convert, display, equal, money_text};

    #[test]
    fn measures_convert_and_compare_across_units() {
        let kg = Unit::named("kg").expect("kg");
        let lb = Unit::named("lb").expect("lb");
        assert_eq!(
            convert(Decimal::from(1000), kg, lb).round_dp(2),
            Decimal::new(220_462, 2)
        );
        assert!(equal(
            &Value::Measure(Decimal::from(1), Unit::named("ton").expect("ton")),
            &Value::Measure(Decimal::from(2000), lb)
        ));
        assert_eq!(
            compare(
                &Value::Measure(Decimal::from(1), kg),
                &Value::Measure(Decimal::from(2), lb)
            ),
            Some(std::cmp::Ordering::Greater)
        );
        let celsius = Unit::named("°C").expect("C");
        let fahrenheit = Unit::named("°F").expect("F");
        assert_eq!(
            convert(Decimal::from(100), celsius, fahrenheit),
            Decimal::from(212)
        );
    }

    #[test]
    fn values_display_like_the_language() {
        assert_eq!(money_text(Decimal::new(185_000, 2), "USD"), "$1,850.00");
        assert_eq!(money_text(Decimal::new(-5, 1), "CAD"), "-0.50 CAD");
        assert_eq!(
            display(&Value::Measure(
                Decimal::from(45_000),
                Unit::named("lb").expect("lb")
            )),
            "45,000 lb"
        );
        assert_eq!(display(&Value::Decimal(Decimal::new(12_500, 3))), "12.5");
    }
}
