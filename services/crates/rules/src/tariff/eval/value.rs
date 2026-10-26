//! Run-time values: what the interpreter computes, the record holds and the bill is made of.
//!
//! A value is cheap to clone: text, lists, records and quantities sit behind reference counts.
//! Money, rates, percents and quantities are one kind, [`Value::Quantity`]: an exact decimal in
//! the unit it was written in, converted only when two units meet (`units::Quantity`). A percent
//! is a quantity in `%`; money is a quantity whose unit measures `Money` (`$`, `$/pallet`). A
//! count written with a label (`3 pallet`) is an `Int`, since labels fold into the value.
//!
//! The evaluator is dynamically typed over the syntax tree: it trusts that the program checked,
//! and an operation on values of the wrong kind is an internal error, never a guess.

use super::error::EvalError;
use super::time::{Date, DateTime, Time};
use crate::tariff::schema::variant_rank;
use crate::tariff::units::{Dim, Quantity, Unit};
use rust_decimal::Decimal;
use std::cmp::Ordering;
use std::fmt::Write as _;
use std::rc::Rc;
use trenova_module_kit::message::{Arg, Message};

/// A run-time value.
#[derive(Debug, Clone)]
pub enum Value {
    /// `none`.
    None,
    /// `true` or `false`.
    Bool(bool),
    /// A whole number, held as a decimal with no fraction.
    Int(Decimal),
    /// An exact decimal.
    Number(Decimal),
    /// Money, a rate, a percent or a quantity with its unit.
    Quantity(Rc<Quantity>),
    /// Text.
    Text(Rc<str>),
    /// A calendar date.
    Date(Date),
    /// An instant with its UTC offset.
    DateTime(DateTime),
    /// A time of day.
    Time(Time),
    /// A list.
    List(Rc<Vec<Value>>),
    /// A record.
    Record(Rc<Record>),
    /// An enum value: its variant's name. The checker guarantees both sides of a comparison
    /// belong to one enum.
    Enum(Rc<str>),
    /// `tables.Code`: a rate table, read with `[…]`, `.cell`, `.band` and `.interpolate`.
    Table(Rc<str>),
}

/// A record's fields, in a stable order.
#[derive(Debug, Clone, Default)]
pub struct Record {
    /// The fields, each name once.
    pub fields: Vec<(Rc<str>, Value)>,
}

impl Record {
    /// A record from its fields.
    #[must_use]
    pub const fn new(fields: Vec<(Rc<str>, Value)>) -> Self {
        Self { fields }
    }

    /// The field named `name`.
    #[must_use]
    pub fn get(&self, name: &str) -> Option<&Value> {
        self.fields
            .iter()
            .find(|(field, _)| &**field == name)
            .map(|(_, value)| value)
    }

    /// Sets a field, adding it when the record has none by that name.
    pub fn set(&mut self, name: &str, value: Value) {
        if let Some(slot) = self.fields.iter_mut().find(|(field, _)| &**field == name) {
            slot.1 = value;
        } else {
            self.fields.push((Rc::from(name), value));
        }
    }
}

/// The unit `$`.
#[must_use]
pub fn dollars() -> Unit {
    Unit::parse("$").unwrap_or_default()
}

/// The unit `%`.
#[must_use]
pub fn percent_unit() -> Unit {
    Unit::parse("%").unwrap_or_default()
}

impl Value {
    /// Text.
    #[must_use]
    pub fn text(text: &str) -> Self {
        Self::Text(Rc::from(text))
    }

    /// A whole number.
    #[must_use]
    pub fn int(value: i64) -> Self {
        Self::Int(Decimal::from(value))
    }

    /// Money in the rating currency.
    #[must_use]
    pub fn money(amount: Decimal) -> Self {
        Self::Quantity(Rc::new(Quantity::new(amount, dollars())))
    }

    /// A percent: `18` is 18%.
    #[must_use]
    pub fn percent(value: Decimal) -> Self {
        Self::Quantity(Rc::new(Quantity::new(value, percent_unit())))
    }

    /// A value in `unit`. A unit that measures nothing (no unit, or only count labels) gives a
    /// plain number: an `Int` when `integer` and whole, else a `Number`.
    #[must_use]
    pub fn quantity(value: Decimal, unit: Unit) -> Self {
        if unit.dim().is_none() && !is_percent_unit(&unit) {
            let folded = fold_labels(value, &unit);
            return Self::Number(folded.normalize());
        }
        Self::Quantity(Rc::new(Quantity::new(value, unit)))
    }

    /// A list.
    #[must_use]
    pub fn list(items: Vec<Self>) -> Self {
        Self::List(Rc::new(items))
    }

    /// A record from `(name, value)` pairs.
    #[must_use]
    pub fn record(fields: Vec<(&str, Self)>) -> Self {
        Self::Record(Rc::new(Record::new(
            fields
                .into_iter()
                .map(|(name, value)| (Rc::from(name), value))
                .collect(),
        )))
    }

    /// An enum value.
    #[must_use]
    pub fn variant(name: &str) -> Self {
        Self::Enum(Rc::from(name))
    }

    /// Whether this is `none`.
    #[must_use]
    pub const fn is_none(&self) -> bool {
        matches!(self, Self::None)
    }

    /// The quantity inside, when this is one.
    #[must_use]
    pub fn as_quantity(&self) -> Option<&Quantity> {
        match self {
            Self::Quantity(quantity) => Some(quantity),
            _ => None,
        }
    }

    /// The amount of money, in `$`, when this is money.
    #[must_use]
    pub fn as_money(&self) -> Option<Decimal> {
        let quantity = self.as_quantity()?;
        if quantity.unit.dim() != Dim::money() {
            return None;
        }
        quantity.to(&dollars()).ok().map(|money| money.value)
    }

    /// The decimal inside an `Int` or `Number`.
    #[must_use]
    pub const fn as_decimal(&self) -> Option<Decimal> {
        match self {
            Self::Int(value) | Self::Number(value) => Some(*value),
            _ => None,
        }
    }

    /// What kind of value this is, as a phrase for messages.
    #[must_use]
    pub fn kind(&self) -> Message {
        Message::new(match self {
            Self::None => trenova_module_kit::message!("nothing (none)"),
            Self::Bool(_) => trenova_module_kit::message!("true or false"),
            Self::Int(_) => trenova_module_kit::message!("a whole number"),
            Self::Number(_) => trenova_module_kit::message!("a number"),
            Self::Quantity(quantity) if is_percent_unit(&quantity.unit) => {
                trenova_module_kit::message!("a percent")
            }
            Self::Quantity(quantity) if quantity.unit.dim() == Dim::money() => {
                trenova_module_kit::message!("money")
            }
            Self::Quantity(quantity)
                if quantity
                    .unit
                    .dim()
                    .exponent(crate::tariff::units::Base::Money)
                    == 1 =>
            {
                trenova_module_kit::message!("a rate")
            }
            Self::Quantity(_) => trenova_module_kit::message!("a quantity"),
            Self::Text(_) => trenova_module_kit::message!("text"),
            Self::Date(_) => trenova_module_kit::message!("a date"),
            Self::DateTime(_) => trenova_module_kit::message!("a date-time"),
            Self::Time(_) => trenova_module_kit::message!("a time of day"),
            Self::List(_) => trenova_module_kit::message!("a list"),
            Self::Record(_) => trenova_module_kit::message!("a record"),
            Self::Enum(_) => trenova_module_kit::message!("an enum value"),
            Self::Table(_) => trenova_module_kit::message!("a rate table"),
        })
    }

    /// Whether two values are equal: numbers by value, quantities across units and scales
    /// (`1 cwt == 100 lb`, `$426 == $426.00`), date-times by instant, records field by field,
    /// lists item by item, enums by variant. `none` equals only `none`.
    ///
    /// # Errors
    ///
    /// When the values are of kinds that cannot be compared, which the checker rules out.
    pub fn equals(&self, other: &Self) -> Result<bool, EvalError> {
        Ok(match (self, other) {
            (Self::None, Self::None) => true,
            (Self::None, _) | (_, Self::None) => false,
            (Self::Bool(left), Self::Bool(right)) => left == right,
            (Self::Int(left) | Self::Number(left), Self::Int(right) | Self::Number(right)) => {
                left == right
            }
            (Self::Quantity(left), Self::Quantity(right)) => {
                left.compare(right).map_err(|_| mismatch(self, other))? == Ordering::Equal
            }
            (Self::Text(left), Self::Text(right)) => left == right,
            (Self::Date(left), Self::Date(right)) => left == right,
            (Self::DateTime(left), Self::DateTime(right)) => left == right,
            (Self::Time(left), Self::Time(right)) => left == right,
            (Self::Enum(left), Self::Enum(right)) | (Self::Table(left), Self::Table(right)) => {
                left == right
            }
            (Self::List(left), Self::List(right)) => {
                if left.len() != right.len() {
                    return Ok(false);
                }
                for (left, right) in left.iter().zip(right.iter()) {
                    if !left.equals(right)? {
                        return Ok(false);
                    }
                }
                true
            }
            (Self::Record(left), Self::Record(right)) => {
                if left.fields.len() != right.fields.len() {
                    return Ok(false);
                }
                for (name, value) in &left.fields {
                    match right.get(name) {
                        Some(other) if value.equals(other)? => {}
                        _ => return Ok(false),
                    }
                }
                true
            }
            _ => return Err(mismatch(self, other)),
        })
    }

    /// Orders two values for `<`, `sortBy`, `min` and `max`: numbers, percents, quantities of
    /// one dimension, dates, date-times, times of day and variants of an ordered enum (by
    /// declaration order). Text has no order here; see [`Value::range_compare`].
    ///
    /// # Errors
    ///
    /// When the values have no order between them, which the checker rules out.
    pub fn compare(&self, other: &Self) -> Result<Ordering, EvalError> {
        match (self, other) {
            (Self::Int(left) | Self::Number(left), Self::Int(right) | Self::Number(right)) => {
                Ok(left.cmp(right))
            }
            (Self::Quantity(left), Self::Quantity(right)) => {
                left.compare(right).map_err(|_| unordered(self, other))
            }
            (Self::Date(left), Self::Date(right)) => Ok(left.cmp(right)),
            (Self::DateTime(left), Self::DateTime(right)) => Ok(left.cmp_instant(*right)),
            (Self::Time(left), Self::Time(right)) => Ok(left.cmp(right)),
            (Self::Enum(left), Self::Enum(right)) => {
                match (variant_rank(left), variant_rank(right)) {
                    (Some(left), Some(right)) => Ok(left.cmp(&right)),
                    _ => Err(unordered(self, other)),
                }
            }
            _ => Err(unordered(self, other)),
        }
    }

    /// Orders a value against a range's end (`in lo..hi`, a range pattern, a `where` range): as
    /// [`Value::compare`], and text too, in Unicode scalar order, character by character.
    ///
    /// # Errors
    ///
    /// When the values have no order between them, which the checker rules out.
    pub fn range_compare(&self, other: &Self) -> Result<Ordering, EvalError> {
        match (self, other) {
            (Self::Text(left), Self::Text(right)) => Ok(left.cmp(right)),
            _ => self.compare(other),
        }
    }

    /// The value as text interpolation writes it: `Int` plain, `Number` normalised, money
    /// `$1234.50` (at least two decimals, no grouping), rates `$2.35/mi`, quantities `500 lb`,
    /// percents `18%`, dates and date-times ISO 8601, times `17:30` (`07:30:15` with seconds),
    /// enums as their variant, `true`/`false`.
    /// Lists read `[a, b]` and records `{ name: value }`, in interpolation (which the checker
    /// allows when no item or field may be empty) and on the trace.
    #[must_use]
    pub fn display(&self) -> String {
        let mut out = String::new();
        self.write_display(&mut out);
        out
    }

    fn write_display(&self, out: &mut String) {
        match self {
            Self::None => out.push_str("none"),
            Self::Bool(flag) => out.push_str(if *flag { "true" } else { "false" }),
            Self::Int(value) | Self::Number(value) => out.push_str(&format_decimal(*value)),
            Self::Quantity(quantity) => out.push_str(&format_quantity(quantity)),
            Self::Text(text) => out.push_str(text),
            Self::Date(date) => out.push_str(&date.iso()),
            Self::DateTime(at) => out.push_str(&at.iso()),
            Self::Time(time) => out.push_str(&time.display()),
            Self::Enum(variant) => out.push_str(variant),
            Self::Table(code) => {
                let _ = write!(out, "tables.{code}");
            }
            Self::List(items) => {
                out.push('[');
                for (index, item) in items.iter().enumerate() {
                    if index > 0 {
                        out.push_str(", ");
                    }
                    item.write_display(out);
                }
                out.push(']');
            }
            Self::Record(record) => {
                out.push_str("{ ");
                for (index, (name, value)) in record.fields.iter().enumerate() {
                    if index > 0 {
                        out.push_str(", ");
                    }
                    out.push_str(name);
                    out.push_str(": ");
                    value.write_display(out);
                }
                out.push_str(" }");
            }
        }
    }
}

/// Whether a unit is exactly `%`.
#[must_use]
pub fn is_percent_unit(unit: &Unit) -> bool {
    matches!(unit.factors(), [(simple, 1)] if simple.symbol == "%")
}

/// Whether a quantity is a percent.
#[must_use]
pub fn is_percent(quantity: &Quantity) -> bool {
    is_percent_unit(&quantity.unit)
}

/// The value with every dimensionless label's factor folded in (`%` is ×0.01; counts are ×1).
fn fold_labels(value: Decimal, unit: &Unit) -> Decimal {
    let mut folded = value;
    for (simple, exponent) in unit.factors() {
        if simple.base.is_none() {
            let factor = simple.factor();
            for _ in 0..exponent.unsigned_abs() {
                folded = if *exponent > 0 {
                    folded.checked_mul(factor)
                } else {
                    folded.checked_div(factor)
                }
                .unwrap_or(folded);
            }
        }
    }
    folded
}

/// A decimal in plain notation, normalised: `1200`, `2.5`, `0`.
#[must_use]
pub fn format_decimal(value: Decimal) -> String {
    if value.is_zero() {
        return String::from("0");
    }
    value.normalize().to_string()
}

/// A quantity as interpolation writes it: `$1234.50`, `$2.35/mi`, `-$5.00`, `500 lb`, `18%`.
#[must_use]
pub fn format_quantity(quantity: &Quantity) -> String {
    if is_percent(quantity) {
        return format!("{}%", format_decimal(quantity.value));
    }
    if let Some(((first, 1), _)) = quantity.unit.factors().split_first().map(|(f, r)| (*f, r))
        && first.symbol == "$"
    {
        let unit = quantity.unit.to_string();
        let per = unit.strip_prefix('$').unwrap_or_default();
        let sign = if quantity.value.is_sign_negative() && !quantity.value.is_zero() {
            "-"
        } else {
            ""
        };
        return format!("{sign}${}{per}", money_digits(quantity.value.abs()));
    }
    format!("{} {}", format_decimal(quantity.value), quantity.unit)
}

/// An amount with at least two decimals: `1234.50`, `0.185`.
#[must_use]
pub fn money_digits(amount: Decimal) -> String {
    let normal = amount.normalize();
    if normal.scale() < 2 {
        format!("{normal:.2}")
    } else {
        normal.to_string()
    }
}

fn mismatch(left: &Value, right: &Value) -> EvalError {
    EvalError::internal(Message::with(
        trenova_module_kit::message!("{0} cannot be compared with {1}"),
        [Arg::from(left.kind()), Arg::from(right.kind())],
    ))
}

fn unordered(left: &Value, right: &Value) -> EvalError {
    EvalError::internal(Message::with(
        trenova_module_kit::message!("{0} and {1} have no order between them"),
        [Arg::from(left.kind()), Arg::from(right.kind())],
    ))
}

#[cfg(test)]
mod tests {
    use super::Value;
    use crate::tariff::units::Unit;
    use rust_decimal::Decimal;
    use std::str::FromStr;

    fn d(text: &str) -> Decimal {
        Decimal::from_str(text).expect("a decimal")
    }

    fn q(value: &str, unit: &str) -> Value {
        Value::quantity(d(value), Unit::parse(unit).expect("a unit"))
    }

    #[test]
    fn interpolation_formats_each_kind() {
        assert_eq!(Value::money(d("1234.5")).display(), "$1234.50");
        assert_eq!(Value::money(d("-5")).display(), "-$5.00");
        assert_eq!(q("2.35", "$/mi").display(), "$2.35/mi");
        assert_eq!(q("0.185", "$/lb").display(), "$0.185/lb");
        assert_eq!(q("500", "lb").display(), "500 lb");
        assert_eq!(Value::percent(d("18")).display(), "18%");
        assert_eq!(Value::Number(d("2.50")).display(), "2.5");
        assert_eq!(Value::int(1200).display(), "1200");
        assert_eq!(Value::Bool(true).display(), "true");
    }

    #[test]
    fn equality_crosses_units_and_scales() {
        assert!(q("1", "cwt").equals(&q("100", "lb")).unwrap_or(false));
        assert!(
            Value::money(d("426"))
                .equals(&Value::money(d("426.00")))
                .unwrap_or(false)
        );
        assert!(
            Value::int(2)
                .equals(&Value::Number(d("2.0")))
                .unwrap_or(false)
        );
        assert!(!Value::None.equals(&Value::int(0)).unwrap_or(true));
        assert!(Value::text("a").equals(&Value::int(1)).is_err());
        assert!(q("1", "lb").equals(&q("1", "mi")).is_err());
    }

    #[test]
    fn ordered_enums_times_and_text_compare() {
        use std::cmp::Ordering;
        let order = |a: &Value, b: &Value| a.compare(b).ok();
        assert_eq!(
            order(&Value::variant("Class77_5"), &Value::variant("Class85")),
            Some(Ordering::Less)
        );
        assert_eq!(
            order(&Value::variant("Sun"), &Value::variant("Mon")),
            Some(Ordering::Greater)
        );
        assert_eq!(
            order(&Value::variant("Pickup"), &Value::variant("Delivery")),
            None
        );
        let range = |a: &Value, b: &Value| a.range_compare(b).ok();
        assert_eq!(
            range(&Value::text("30301"), &Value::text("31999")),
            Some(Ordering::Less)
        );
        assert_eq!(
            range(&Value::text("9"), &Value::text("10")),
            Some(Ordering::Greater)
        );
        assert_eq!(order(&Value::text("a"), &Value::text("b")), None);
        let time = |text: &str| Value::Time(super::Time::parse(text).expect("a time"));
        assert_eq!(order(&time("08:00"), &time("17:00")), Some(Ordering::Less));
        assert_eq!(time("17:30").display(), "17:30");
    }

    #[test]
    fn count_labels_fold_into_plain_numbers() {
        assert!(matches!(q("3", "pallet"), Value::Number(_)));
    }
}
