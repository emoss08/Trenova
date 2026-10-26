//! Arithmetic on values: the spec's "Units and dimensions" rules, exactly.
//!
//! - `Int + - *` give an `Int`; `Int / Int` gives a `Number`; `%` takes `Int`s.
//! - `+`, `-` take one dimension and give the left operand's unit: `500 lb + 2 cwt` is `700 lb`.
//! - `×` and `÷` combine units with `units::Quantity::mul`/`div`, converting where a base meets
//!   itself in another unit, so `$18.50/cwt × 1_200 lb` is `$222`, and a quotient whose units
//!   cancel is a plain `Number`.
//! - A percent scaled by a number stays a percent (`0.5% × 3` is `1.5%`) unless the static type
//!   says the product is a number; a percent times money is money.
//! - `DateTime - DateTime` is a duration, `Date - Date` whole days, `Time - Time` the duration
//!   between two times of one day (negative when the right side is later).
//!
//! Every result is exact or an error: overflow past 28 significant digits and division by zero
//! never wrap, round or panic.

use super::error::{Code, EvalError};
use super::time::{Date, DateTime};
use super::value::{Value, is_percent, percent_unit};
use crate::tariff::units::{Quantity, Unit, UnitError};
use rust_decimal::Decimal;
use std::rc::Rc;
use trenova_module_kit::message::{Arg, Message};

/// What a unit error means at run time.
#[must_use]
pub fn unit_error(error: &UnitError) -> EvalError {
    match error {
        UnitError::Overflow => EvalError::overflow(),
        UnitError::Mismatch { from, to } => EvalError::with(
            Code::Conversion,
            trenova_module_kit::message!("{0} cannot be converted to {1}"),
            [Arg::from(from.clone()), Arg::from(to.clone())],
        ),
        UnitError::Temperature => EvalError::with(
            Code::Conversion,
            trenova_module_kit::message!("Temperatures can only be compared and subtracted"),
            [],
        ),
        UnitError::Unknown(text) => EvalError::with(
            Code::Conversion,
            trenova_module_kit::message!("{0} is not a unit"),
            [Arg::from(text.clone())],
        ),
    }
}

fn wrong(symbol: &str, left: &Value, right: &Value) -> EvalError {
    EvalError::internal(Message::with(
        trenova_module_kit::message!("{0} cannot be applied to {1} and {2}"),
        [
            Arg::from(symbol),
            Arg::from(left.kind()),
            Arg::from(right.kind()),
        ],
    ))
}

fn checked(value: Option<Decimal>) -> Result<Decimal, EvalError> {
    value.ok_or_else(EvalError::overflow)
}

/// An `Int` result, refused past 28 significant digits.
fn int(value: Option<Decimal>) -> Result<Value, EvalError> {
    Ok(Value::Int(checked(value)?.normalize()))
}

fn plain(value: Decimal) -> Quantity {
    Quantity::new(value, Unit::none())
}

fn quantity_result(result: Result<Quantity, UnitError>) -> Result<Value, EvalError> {
    let quantity = result.map_err(|error| unit_error(&error))?;
    Ok(Value::quantity(quantity.value.normalize(), quantity.unit))
}

/// `left + right`.
///
/// # Errors
///
/// On overflow, or operands the checker would refuse.
pub fn add(left: &Value, right: &Value) -> Result<Value, EvalError> {
    match (left, right) {
        (Value::Int(a), Value::Int(b)) => int(a.checked_add(*b)),
        (Value::Int(a) | Value::Number(a), Value::Int(b) | Value::Number(b)) => {
            Ok(Value::Number(checked(a.checked_add(*b))?))
        }
        (Value::Quantity(a), Value::Quantity(b)) => match a.add(b) {
            Ok(sum) => Ok(Value::Quantity(Rc::new(sum))),
            Err(UnitError::Overflow) => Err(EvalError::overflow()),
            Err(_) => Err(wrong("+", left, right)),
        },
        _ => Err(wrong("+", left, right)),
    }
}

/// `left - right`.
///
/// # Errors
///
/// On overflow, or operands the checker would refuse.
pub fn sub(left: &Value, right: &Value) -> Result<Value, EvalError> {
    match (left, right) {
        (Value::Int(a), Value::Int(b)) => int(a.checked_sub(*b)),
        (Value::Int(a) | Value::Number(a), Value::Int(b) | Value::Number(b)) => {
            Ok(Value::Number(checked(a.checked_sub(*b))?))
        }
        (Value::Quantity(a), Value::Quantity(b)) => match a.sub(b) {
            Ok(difference) => Ok(Value::Quantity(Rc::new(difference))),
            Err(UnitError::Overflow) => Err(EvalError::overflow()),
            Err(_) => Err(wrong("-", left, right)),
        },
        (Value::DateTime(a), Value::DateTime(b)) => duration_between(*b, *a),
        (Value::Date(a), Value::Date(b)) => days_between(*b, *a),
        (Value::Time(a), Value::Time(b)) => Ok(duration_of_seconds(a.seconds() - b.seconds())),
        _ => Err(wrong("-", left, right)),
    }
}

/// `left × right`. `percent` says what a percent scaled by a number is: a percent (`true`) or
/// the plain number it multiplies as (`false`).
///
/// # Errors
///
/// On overflow, or operands the checker would refuse.
pub fn mul(left: &Value, right: &Value, percent: bool) -> Result<Value, EvalError> {
    match (left, right) {
        (Value::Int(a), Value::Int(b)) => int(a.checked_mul(*b)),
        (Value::Int(a) | Value::Number(a), Value::Int(b) | Value::Number(b)) => {
            Ok(Value::Number(checked(a.checked_mul(*b))?))
        }
        (Value::Quantity(q), Value::Int(n) | Value::Number(n))
        | (Value::Int(n) | Value::Number(n), Value::Quantity(q)) => {
            if is_percent(q) && percent {
                return Ok(Value::percent(checked(q.value.checked_mul(*n))?));
            }
            if matches!(left, Value::Quantity(_)) {
                quantity_result(q.mul(&plain(*n)))
            } else {
                quantity_result(plain(*n).mul(q))
            }
        }
        (Value::Quantity(a), Value::Quantity(b)) => quantity_result(a.mul(b)),
        _ => Err(wrong("*", left, right)),
    }
}

/// `left / right`. `percent` is as for [`mul`].
///
/// # Errors
///
/// On division by zero, overflow, or operands the checker would refuse.
pub fn div(left: &Value, right: &Value, percent: bool) -> Result<Value, EvalError> {
    let divisor_zero = match right {
        Value::Int(value) | Value::Number(value) => value.is_zero(),
        Value::Quantity(quantity) => quantity.value.is_zero(),
        _ => false,
    };
    if divisor_zero {
        return Err(EvalError::divide_by_zero());
    }
    match (left, right) {
        (Value::Int(a) | Value::Number(a), Value::Int(b) | Value::Number(b)) => {
            Ok(Value::Number(checked(a.checked_div(*b))?.normalize()))
        }
        (Value::Quantity(q), Value::Int(n) | Value::Number(n)) => {
            if is_percent(q) && percent {
                return Ok(Value::percent(
                    checked(q.value.checked_div(*n))?.normalize(),
                ));
            }
            quantity_result(q.div(&plain(*n)))
        }
        (Value::Int(n) | Value::Number(n), Value::Quantity(q)) => quantity_result(plain(*n).div(q)),
        (Value::Quantity(a), Value::Quantity(b)) => quantity_result(a.div(b)),
        _ => Err(wrong("/", left, right)),
    }
}

/// `left % right`, on `Int`s.
///
/// # Errors
///
/// On division by zero, or operands the checker would refuse.
pub fn rem(left: &Value, right: &Value) -> Result<Value, EvalError> {
    match (left, right) {
        (Value::Int(_), Value::Int(b)) if b.is_zero() => Err(EvalError::divide_by_zero()),
        (Value::Int(a), Value::Int(b)) => int(a.checked_rem(*b)),
        _ => Err(wrong("%", left, right)),
    }
}

/// `-value`.
///
/// # Errors
///
/// On an operand the checker would refuse.
pub fn neg(value: &Value) -> Result<Value, EvalError> {
    match value {
        Value::Int(number) => Ok(Value::Int(-*number)),
        Value::Number(number) => Ok(Value::Number(-*number)),
        Value::Quantity(quantity) => Ok(Value::Quantity(Rc::new(Quantity::new(
            -quantity.value,
            quantity.unit.clone(),
        )))),
        other => Err(EvalError::internal(Message::with(
            trenova_module_kit::message!("{0} cannot be negated"),
            [Arg::from(other.kind())],
        ))),
    }
}

/// The duration from `start` to `end`: in hours when that is exact to two decimals, else in
/// minutes.
///
/// # Errors
///
/// When the duration does not fit.
pub fn duration_between(start: DateTime, end: DateTime) -> Result<Value, EvalError> {
    let seconds = end
        .unix()
        .checked_sub(start.unix())
        .ok_or_else(EvalError::overflow)?;
    Ok(duration_of_seconds(seconds))
}

/// A whole number of seconds as a duration value: hours when exact to two decimals, else
/// minutes.
#[must_use]
pub fn duration_of_seconds(seconds: i64) -> Value {
    let seconds = Decimal::from(seconds);
    let (divisor, unit) = if (seconds % Decimal::from(36)).is_zero() {
        (Decimal::from(3600), "h")
    } else {
        (Decimal::from(60), "min")
    };
    let value = seconds
        .checked_div(divisor)
        .map_or(Decimal::ZERO, |value| value.normalize());
    Value::quantity(value, Unit::parse(unit).unwrap_or_default())
}

fn days_between(start: Date, end: Date) -> Result<Value, EvalError> {
    let days = end
        .days()
        .checked_sub(start.days())
        .ok_or_else(EvalError::overflow)?;
    Ok(Value::quantity(
        Decimal::from(days),
        Unit::parse("d").unwrap_or_default(),
    ))
}

/// Whether a value is a percent and the other a plain number, the one case the static type
/// decides.
#[must_use]
pub fn scales_a_percent(left: &Value, right: &Value) -> bool {
    let percent = |value: &Value| matches!(value, Value::Quantity(q) if is_percent(q));
    let number = |value: &Value| matches!(value, Value::Int(_) | Value::Number(_));
    (percent(left) && number(right)) || (number(left) && percent(right))
}

/// `0%`.
#[must_use]
pub fn zero_percent() -> Value {
    Value::Quantity(Rc::new(Quantity::new(Decimal::ZERO, percent_unit())))
}

#[cfg(test)]
mod tests {
    use super::{add, div, mul, rem, sub};
    use crate::tariff::eval::time::DateTime;
    use crate::tariff::eval::value::Value;
    use crate::tariff::units::Unit;
    use rust_decimal::Decimal;
    use std::str::FromStr;

    fn d(text: &str) -> Decimal {
        Decimal::from_str(text).expect("a decimal")
    }

    fn q(value: &str, unit: &str) -> Value {
        Value::quantity(d(value), Unit::parse(unit).expect("a unit"))
    }

    fn shown(result: Result<Value, crate::tariff::eval::error::EvalError>) -> String {
        result.map_or_else(
            |error| error.code.as_str().to_owned(),
            |value| value.display(),
        )
    }

    #[test]
    fn rates_price_quantities_across_units() {
        assert_eq!(
            shown(mul(&q("18.50", "$/cwt"), &q("1200", "lb"), true)),
            "$222.00"
        );
        assert_eq!(
            shown(mul(&q("1200", "lb"), &q("18.50", "$/cwt"), true)),
            "$222.00"
        );
        assert_eq!(shown(add(&q("500", "lb"), &q("2", "cwt"))), "700 lb");
        assert_eq!(shown(div(&q("1200", "lb"), &q("1", "cwt"), true)), "12");
    }

    #[test]
    fn percents_scale_and_fold() {
        assert_eq!(shown(mul(&q("0.5", "%"), &Value::int(3), true)), "1.5%");
        assert_eq!(shown(mul(&Value::int(3), &q("0.5", "%"), false)), "0.015");
        assert_eq!(
            shown(mul(&Value::money(d("100")), &q("18", "%"), true)),
            "$18.00"
        );
    }

    #[test]
    fn whole_numbers_stay_whole() {
        assert!(matches!(
            add(&Value::int(2), &Value::int(3)),
            Ok(Value::Int(_))
        ));
        assert!(matches!(
            div(&Value::int(3), &Value::int(2), true),
            Ok(Value::Number(_))
        ));
        assert_eq!(shown(rem(&Value::int(7), &Value::int(0))), "divideByZero");
        assert_eq!(
            shown(div(&Value::money(d("1")), &q("0", "mi"), true)),
            "divideByZero"
        );
        let big = Value::Int(Decimal::MAX);
        assert_eq!(shown(add(&big, &Value::int(1))), "overflow");
    }

    #[test]
    fn date_times_subtract_to_durations() {
        let start = Value::DateTime(DateTime::parse("2026-01-05T08:00Z").expect("a date-time"));
        let end = Value::DateTime(DateTime::parse("2026-01-05T11:30Z").expect("a date-time"));
        assert_eq!(shown(sub(&end, &start)), "3.5 h");
        let odd = Value::DateTime(DateTime::parse("2026-01-05T08:07Z").expect("a date-time"));
        assert_eq!(shown(sub(&odd, &start)), "7 min");
    }
}
