//! Operators and access over values: arithmetic, comparison, membership, members, indexes,
//! slices, ranges and the methods dates and durations have.
//!
//! Every operator checks its operands' kinds at run time as well, because a value read through
//! `get`, an untyped relation or a predicate is only `Any` to the checker. An empty operand is an
//! error marked with which side was empty, so the interpreter can name the field it was read
//! from ([`EvalError::missing`]); `??`, `coalesce`, `== nil` and optional chaining are the ways
//! a formula handles an empty value, and they never reach these functions with one.
//!
//! Equality between two values of different kinds is an error, wherever the values came from:
//! `origin.zip == 75201` compares text with a number and would always be false. Nil compares
//! equal only to nil, without error.

use super::budget::Meter;
use super::error::Code;
use super::error::EvalError;
use super::error::cannot_compare;
use super::error::expected_boolean;
use super::error::expected_text;
use super::error::expected_whole_kind;
use super::error::invalid_operation;
use super::error::invalid_unary;
use super::error::kind_phrase;
use super::number;
use super::value::Value;
use super::value::format_number;
use crate::message::Arg;
use crate::message::Message;
use rust_decimal::Decimal;
use std::cmp::Ordering;
use std::rc::Rc;

/// An arithmetic operator.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Arithmetic {
    /// `+`
    Add,
    /// `-`
    Sub,
    /// `*`
    Mul,
    /// `/`
    Div,
    /// `%`
    Rem,
    /// `**` and `^`
    Pow,
}

impl Arithmetic {
    const fn symbol(self) -> &'static str {
        match self {
            Self::Add => "+",
            Self::Sub => "-",
            Self::Mul => "*",
            Self::Div => "/",
            Self::Rem => "%",
            Self::Pow => "**",
        }
    }
}

/// An ordering comparison.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Comparison {
    /// `<`
    Less,
    /// `<=`
    LessOrEqual,
    /// `>`
    Greater,
    /// `>=`
    GreaterOrEqual,
}

/// The error for an empty operand on `side` (0 left, 1 right).
fn empty(side: usize, error: EvalError) -> EvalError {
    error.empty_argument(side)
}

fn date_out_of_range() -> EvalError {
    EvalError::new(
        Code::Overflow,
        Message::new(crate::message!(
            "the date is out of range (years 0000 to 9999)"
        )),
    )
}

fn duration_out_of_range() -> EvalError {
    EvalError::new(
        Code::Overflow,
        Message::new(crate::message!(
            "the duration is out of range (about 292 years either way)"
        )),
    )
}

/// Scales a duration by a number, truncating to the nanosecond.
fn scale_duration(nanos: i64, factor: Decimal, divide: bool) -> Result<Value, EvalError> {
    let base = Decimal::from(nanos);
    let scaled = if divide {
        number::div(base, factor)?
    } else {
        base.checked_mul(factor).ok_or_else(duration_out_of_range)?
    };
    i64::try_from(scaled.trunc())
        .map(Value::Duration)
        .map_err(|_| duration_out_of_range())
}

/// `left op right`.
pub fn arithmetic(
    op: Arithmetic,
    left: &Value,
    right: &Value,
    meter: &mut Meter,
) -> Result<Value, EvalError> {
    let mismatch = || {
        let error = invalid_operation(op.symbol(), left.kind_key(), right.kind_key());
        if left.is_nil() {
            empty(0, error)
        } else if right.is_nil() {
            empty(1, error)
        } else {
            error
        }
    };
    match (op, left, right) {
        (_, Value::Number(a), Value::Number(b)) => match op {
            Arithmetic::Add => number::add(*a, *b),
            Arithmetic::Sub => number::sub(*a, *b),
            Arithmetic::Mul => number::mul(*a, *b),
            Arithmetic::Div => number::div(*a, *b),
            Arithmetic::Rem => number::rem(*a, *b),
            Arithmetic::Pow => number::pow(*a, *b),
        }
        .map(Value::Number),
        (Arithmetic::Add, Value::Text(a), Value::Text(b)) => {
            meter.text(a.len() + b.len())?;
            let mut joined = String::with_capacity(a.len() + b.len());
            joined.push_str(a);
            joined.push_str(b);
            Ok(Value::Text(Rc::from(joined)))
        }
        (Arithmetic::Add, Value::Date(date), Value::Duration(nanos))
        | (Arithmetic::Add, Value::Duration(nanos), Value::Date(date)) => date
            .add_nanos(*nanos)
            .map(Value::Date)
            .ok_or_else(date_out_of_range),
        (Arithmetic::Sub, Value::Date(date), Value::Duration(nanos)) => nanos
            .checked_neg()
            .and_then(|back| date.add_nanos(back))
            .map(Value::Date)
            .ok_or_else(date_out_of_range),
        (Arithmetic::Sub, Value::Date(a), Value::Date(b)) => a
            .since(b)
            .map(Value::Duration)
            .ok_or_else(duration_out_of_range),
        (Arithmetic::Add, Value::Duration(a), Value::Duration(b)) => a
            .checked_add(*b)
            .map(Value::Duration)
            .ok_or_else(duration_out_of_range),
        (Arithmetic::Sub, Value::Duration(a), Value::Duration(b)) => a
            .checked_sub(*b)
            .map(Value::Duration)
            .ok_or_else(duration_out_of_range),
        (Arithmetic::Mul, Value::Duration(nanos), Value::Number(factor))
        | (Arithmetic::Mul, Value::Number(factor), Value::Duration(nanos)) => {
            scale_duration(*nanos, *factor, false)
        }
        (Arithmetic::Div, Value::Duration(nanos), Value::Number(divisor)) => {
            scale_duration(*nanos, *divisor, true)
        }
        (Arithmetic::Div, Value::Duration(a), Value::Duration(b)) => {
            number::div(Decimal::from(*a), Decimal::from(*b)).map(Value::Number)
        }
        _ => Err(mismatch()),
    }
}

/// `-value` and `+value`.
pub fn sign(negate: bool, value: &Value) -> Result<Value, EvalError> {
    let symbol = if negate { "-" } else { "+" };
    match value {
        Value::Number(number) if negate => Ok(Value::Number(-*number)),
        Value::Number(_) | Value::Duration(_) if !negate => Ok(value.clone()),
        Value::Duration(nanos) => nanos
            .checked_neg()
            .map(Value::Duration)
            .ok_or_else(duration_out_of_range),
        Value::Nil => Err(empty(0, invalid_unary(symbol, "nil"))),
        other => Err(invalid_unary(symbol, other.kind_key())),
    }
}

/// `not value`.
pub fn not(value: &Value) -> Result<Value, EvalError> {
    match value {
        Value::Bool(flag) => Ok(Value::Bool(!flag)),
        Value::Nil => Err(empty(0, expected_boolean("nil"))),
        other => Err(expected_boolean(other.kind_key())),
    }
}

/// `left == right`: nil equals only nil; other values of different kinds may not be compared.
pub fn equal(left: &Value, right: &Value) -> Result<bool, EvalError> {
    if left.is_nil() || right.is_nil() {
        return Ok(left.is_nil() && right.is_nil());
    }
    if left.kind() != right.kind() {
        return Err(cannot_compare(left.kind_key(), right.kind_key()));
    }
    Ok(left.same(right))
}

/// `left < right` and the rest: numbers, text, dates and durations, each with its own kind.
pub fn compare(op: Comparison, left: &Value, right: &Value) -> Result<bool, EvalError> {
    let Some(order) = left.compare(right) else {
        let error = cannot_compare(left.kind_key(), right.kind_key());
        return Err(if left.is_nil() {
            empty(0, error)
        } else if right.is_nil() {
            empty(1, error)
        } else {
            error
        });
    };
    Ok(match op {
        Comparison::Less => order == Ordering::Less,
        Comparison::LessOrEqual => order != Ordering::Greater,
        Comparison::Greater => order == Ordering::Greater,
        Comparison::GreaterOrEqual => order != Ordering::Less,
    })
}

fn text_operand(value: &Value, side: usize) -> Result<&str, EvalError> {
    match value {
        Value::Text(text) => Ok(text),
        Value::Nil => Err(empty(side, expected_text("nil"))),
        other => Err(expected_text(other.kind_key())),
    }
}

/// A text operator.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TextTest {
    /// `contains`
    Contains,
    /// `startsWith`
    StartsWith,
    /// `endsWith`
    EndsWith,
}

/// `text contains part`, `text startsWith prefix`, `text endsWith suffix`.
pub fn text_test(test: TextTest, left: &Value, right: &Value) -> Result<bool, EvalError> {
    let haystack = text_operand(left, 0)?;
    let needle = text_operand(right, 1)?;
    Ok(match test {
        TextTest::Contains => haystack.contains(needle),
        TextTest::StartsWith => haystack.starts_with(needle),
        TextTest::EndsWith => haystack.ends_with(needle),
    })
}

/// `needle in haystack`: an equal item of a list, a key of a map; nothing is in nil.
pub fn membership(needle: &Value, haystack: &Value) -> Result<bool, EvalError> {
    match haystack {
        Value::Nil => Ok(false),
        Value::List(list) => Ok(list.items().iter().any(|item| item.same(needle))),
        Value::Map(map) => Ok(map_key(needle, 0)
            .map(|key| map.get(&key).is_some())
            .unwrap_or(false)),
        other => Err(EvalError::with(
            Code::Type,
            crate::message!("operator \"in\" not defined on {0}"),
            [kind_phrase(other.kind_key())],
        )),
    }
}

/// A value used as a map key: text as written, numbers and booleans as `string()` writes them.
pub fn map_key(key: &Value, side: usize) -> Result<String, EvalError> {
    match key {
        Value::Text(text) => Ok(text.to_string()),
        Value::Number(number) => Ok(format_number(*number)),
        Value::Bool(flag) => Ok(String::from(if *flag { "true" } else { "false" })),
        Value::Nil => Err(empty(side, map_key_error("nil"))),
        other => Err(map_key_error(other.kind_key())),
    }
}

fn map_key_error(kind: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!("a map key must be text, got {0}"),
        [kind_phrase(kind)],
    )
}

fn cannot_fetch(what: String, kind: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!("cannot fetch {0} from {1}"),
        [Arg::from(what), kind_phrase(kind)],
    )
}

/// `value.name`: a map's field, nil when the map has none.
pub fn member(value: &Value, name: &str) -> Result<Value, EvalError> {
    match value {
        Value::Map(map) => Ok(map.get(name).cloned().unwrap_or(Value::Nil)),
        Value::Nil => Err(empty(0, cannot_fetch(name.to_owned(), "nil"))),
        other => Err(cannot_fetch(name.to_owned(), other.kind_key())),
    }
}

/// A whole number used as a position.
pub fn position(value: &Value, side: usize) -> Result<i64, EvalError> {
    match value {
        Value::Number(number) => number::to_i64(*number),
        Value::Nil => Err(empty(side, expected_whole_kind("nil"))),
        other => Err(expected_whole_kind(other.kind_key())),
    }
}

fn out_of_range(index: i64, length: usize) -> EvalError {
    EvalError::with(
        Code::Index,
        crate::message!("index out of range: {0} (array length is {1})"),
        [Arg::from(index.to_string()), Arg::from(length.to_string())],
    )
}

fn resolve_index(index: i64, length: usize) -> Option<usize> {
    let length_signed = i64::try_from(length).ok()?;
    let resolved = if index < 0 {
        length_signed.checked_add(index)?
    } else {
        index
    };
    usize::try_from(resolved).ok().filter(|at| *at < length)
}

/// `value[index]`: a list item or a character (negative indexes count from the end), or a map
/// entry (nil when absent). An index outside a list or text is an error.
pub fn index(value: &Value, key: &Value, meter: &mut Meter) -> Result<Value, EvalError> {
    match value {
        Value::List(list) => {
            let at = position(key, 1)?;
            resolve_index(at, list.items().len())
                .map(|at| list.items()[at].clone())
                .ok_or_else(|| out_of_range(at, list.items().len()))
        }
        Value::Text(text) => {
            let at = position(key, 1)?;
            let length = text.chars().count();
            let character = resolve_index(at, length)
                .and_then(|at| text.chars().nth(at))
                .ok_or_else(|| out_of_range(at, length))?;
            meter.text(character.len_utf8())?;
            Ok(Value::Text(Rc::from(character.to_string())))
        }
        Value::Map(map) => {
            let key = map_key(key, 1)?;
            Ok(map.get(&key).cloned().unwrap_or(Value::Nil))
        }
        Value::Nil => Err(empty(0, cannot_fetch(key.display(), "nil"))),
        other => Err(cannot_fetch(key.display(), other.kind_key())),
    }
}

/// Go's slice bounds: negative counts from the end, then both clamp into the value.
fn slice_bounds(from: Option<i64>, to: Option<i64>, length: usize) -> (usize, usize) {
    let length_signed = i64::try_from(length).unwrap_or(i64::MAX);
    let clamp = |bound: i64| {
        let bound = if bound < 0 {
            length_signed.saturating_add(bound)
        } else {
            bound
        };
        usize::try_from(bound.clamp(0, length_signed)).unwrap_or(0)
    };
    let end = to.map_or(length, clamp);
    let start = from.map_or(0, clamp).min(end);
    (start, end)
}

/// `value[from:to]` on a list or text (in characters).
pub fn slice(
    value: &Value,
    from: Option<i64>,
    to: Option<i64>,
    meter: &mut Meter,
) -> Result<Value, EvalError> {
    match value {
        Value::List(list) => {
            let (start, end) = slice_bounds(from, to, list.items().len());
            let items = list.items()[start..end].to_vec();
            meter.items(items.len())?;
            Value::list(items).map_err(|_| too_deep())
        }
        Value::Text(text) => {
            let length = text.chars().count();
            let (start, end) = slice_bounds(from, to, length);
            let piece: String = text.chars().skip(start).take(end - start).collect();
            meter.text(piece.len())?;
            Ok(Value::Text(Rc::from(piece)))
        }
        Value::Nil => Err(empty(0, cannot_slice("nil"))),
        other => Err(cannot_slice(other.kind_key())),
    }
}

fn cannot_slice(kind: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!("cannot slice {0}"),
        [kind_phrase(kind)],
    )
}

/// A value nested deeper than the language allows.
#[must_use]
pub fn too_deep() -> EvalError {
    EvalError::new(
        Code::Budget,
        Message::new(crate::message!(
            "the formula built lists or maps nested too deeply"
        )),
    )
}

/// `from..to`: the whole numbers from `from` to `to` inclusive, empty when `to < from`.
pub fn range(from: &Value, to: &Value, meter: &mut Meter) -> Result<Value, EvalError> {
    let start = position(from, 0)?;
    let end = position(to, 1)?;
    if end < start {
        return Value::list(Vec::new()).map_err(|_| too_deep());
    }
    let count = u64::try_from(i128::from(end) - i128::from(start) + 1).unwrap_or(u64::MAX);
    meter.items_would_fit(count)?;
    let count = usize::try_from(count).unwrap_or(usize::MAX);
    meter.items(count)?;
    meter.steps(count)?;
    Value::list((start..=end).map(Value::int).collect()).map_err(|_| too_deep())
}

/// `value.Method()` on a date or a duration.
pub fn method(value: &Value, name: &str) -> Result<Value, EvalError> {
    match value {
        Value::Date(date) => {
            let civil = date.civil();
            let field = match name {
                "Year" => civil.year,
                "Month" => civil.month,
                "Day" => civil.day,
                "Hour" => civil.hour,
                "Minute" => civil.minute,
                "Second" => civil.second,
                "Weekday" => civil.weekday,
                "YearDay" => civil.year_day,
                "Unix" => date.unix(),
                _ => return Err(cannot_fetch(name.to_owned(), "date")),
            };
            Ok(Value::int(field))
        }
        Value::Duration(nanos) => {
            let unit: i64 = match name {
                "Hours" => 3_600_000_000_000,
                "Minutes" => 60_000_000_000,
                "Seconds" => 1_000_000_000,
                _ => return Err(cannot_fetch(name.to_owned(), "duration")),
            };
            number::div(Decimal::from(*nanos), Decimal::from(unit)).map(Value::Number)
        }
        Value::Nil => Err(empty(0, cannot_fetch(name.to_owned(), "nil"))),
        other => Err(cannot_fetch(name.to_owned(), other.kind_key())),
    }
}

#[cfg(test)]
mod tests {
    use super::{
        Arithmetic, Comparison, arithmetic, compare, equal, index, member, membership, method,
        range, slice,
    };
    use crate::formula::eval::budget::{Limits, Meter};
    use crate::formula::eval::error::Code;
    use crate::formula::eval::time::parse_date;
    use crate::formula::eval::tree::parse_value;
    use crate::formula::eval::value::Value;

    fn meter() -> Meter {
        Meter::new(Limits::DEFAULT)
    }

    fn tree(text: &str) -> Value {
        parse_value(text).expect("tree")
    }

    #[test]
    fn arithmetic_is_exact_and_typed() {
        let sum = arithmetic(Arithmetic::Add, &tree("0.1"), &tree("0.2"), &mut meter());
        assert!(sum.expect("adds").same(&tree("0.3")));
        let error = arithmetic(Arithmetic::Add, &tree("\"a\""), &tree("1"), &mut meter())
            .expect_err("text plus number");
        assert_eq!(error.code, Code::Type);
        let empty =
            arithmetic(Arithmetic::Mul, &Value::Nil, &tree("2"), &mut meter()).expect_err("nil");
        assert_eq!(empty.empty_argument, Some(0));
    }

    #[test]
    fn dates_and_durations_combine() {
        let start = Value::Date(parse_date("2026-01-01").expect("date"));
        let hour = Value::Duration(3_600_000_000_000);
        let later = arithmetic(Arithmetic::Add, &start, &hour, &mut meter()).expect("adds");
        assert!(compare(Comparison::Greater, &later, &start).expect("compares"));
        let gap = arithmetic(Arithmetic::Sub, &later, &start, &mut meter()).expect("subtracts");
        assert!(gap.same(&hour));
        let doubled = arithmetic(Arithmetic::Mul, &hour, &tree("2"), &mut meter()).expect("x2");
        assert!(doubled.same(&Value::Duration(7_200_000_000_000)));
        assert!(method(&hour, "Minutes").expect("method").same(&tree("60")));
        assert!(method(&start, "Weekday").expect("method").same(&tree("4")));
    }

    #[test]
    fn equality_refuses_mixed_kinds_but_not_nil() {
        assert!(equal(&tree("\"75201\""), &tree("75201")).is_err());
        assert_eq!(equal(&Value::Nil, &tree("1")), Ok(false));
        assert_eq!(equal(&Value::Nil, &Value::Nil), Ok(true));
        assert_eq!(equal(&tree("[1, 2]"), &tree("[1, 2.0]")), Ok(true));
    }

    #[test]
    fn access_counts_characters() {
        let text = tree("\"🚚 é\"");
        assert!(
            index(&text, &tree("-1"), &mut meter())
                .expect("char")
                .same(&tree("\"é\""))
        );
        assert!(
            slice(&text, Some(0), Some(1), &mut meter())
                .expect("slice")
                .same(&tree("\"🚚\""))
        );
        let list = tree("[1, 2, 3]");
        assert_eq!(
            index(&list, &tree("3"), &mut meter()).map_err(|error| error.code),
            Err(Code::Index)
        );
        assert!(
            slice(&list, Some(-2), None, &mut meter())
                .expect("slice")
                .same(&tree("[2, 3]"))
        );
        assert!(
            member(&tree("{\"a\": 1}"), "b")
                .expect("member")
                .same(&Value::Nil)
        );
        assert_eq!(membership(&tree("2"), &tree("[1, 2]")), Ok(true));
        assert_eq!(membership(&tree("\"a\""), &tree("{\"a\": 1}")), Ok(true));
    }

    #[test]
    fn ranges_are_bounded_by_the_item_budget() {
        let mut small = Meter::new(Limits {
            items: 10,
            ..Limits::DEFAULT
        });
        assert!(range(&tree("1"), &tree("10"), &mut small).is_ok());
        assert_eq!(
            range(&tree("1"), &tree("100000000000"), &mut meter()).map_err(|error| error.code),
            Err(Code::Budget)
        );
        assert!(
            range(&tree("3"), &tree("1"), &mut meter())
                .expect("empty")
                .same(&tree("[]"))
        );
    }
}
