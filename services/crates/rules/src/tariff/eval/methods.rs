//! Properties and methods on numbers, money, quantities, dates and date-times: the spec's
//! "Methods on values" table. Text methods are in `text`, list methods in `lists`.
//!
//! - Numbers: `round(places)` and `roundHalfEven(places)` (an `Int` at zero places), `floor`,
//!   `ceil` (both `Int`), `abs`, `min`, `max`, `clamp`, `pow(n)`, `sqrt`.
//! - Money: `round()` (to cents, half up), `roundHalfEven()`, `roundUp(to:)` and
//!   `roundDown(to:)` (to the next multiple up or down), `abs`, `min`, `max`, `clamp`.
//! - Quantities: `value`, `in(unit)`, `round(to:)` (nearest multiple, half up), `roundUp(to:)`,
//!   `roundDown(to:)`, `abs`, `min`, `max`, `clamp`. The increment is converted into the
//!   receiver's unit first, so `1_240 lb.roundUp(to: 1 cwt)` is `1300 lb`.
//! - Dates: `year`, `month`, `day`, `weekday` (a `Weekday`), `weekend`, `addDays(n)`; date-times
//!   also `hour`, `minute`, `date` and `addHours(n)`, all in the date-time's own offset.

use super::error::{Code, EvalError};
use super::operators::unit_error;
use super::time::{Date, DateTime};
use super::value::{Value, format_quantity};
use crate::message::{Arg, Message};
use crate::tariff::units::{Base, Dim, Quantity, Unit};
use rust_decimal::prelude::ToPrimitive;
use rust_decimal::{Decimal, MathematicalOps, RoundingStrategy};
use std::cmp::Ordering;
use std::rc::Rc;

/// The parameter names of a method on `receiver`, in order, for binding named arguments.
#[must_use]
pub fn params(receiver: &Value, method: &str) -> &'static [&'static str] {
    match method {
        "round" | "roundHalfEven" => match receiver {
            Value::Quantity(quantity) if quantity.unit.dim() != Dim::money() => &["to"],
            Value::Quantity(_) => &[],
            _ => &["places"],
        },
        "roundUp" | "roundDown" => &["to"],
        "min" | "max" => &["x"],
        "clamp" => &["lo", "hi"],
        "pow" | "addDays" | "addHours" => &["n"],
        "in" => &["unit"],
        "startsWith" | "endsWith" | "contains" => &["t"],
        "replace" => &["a", "b"],
        "split" => &["sep"],
        "matches" => &["pattern"],
        _ => &[],
    }
}

fn arg(args: &[Value], index: usize) -> &Value {
    args.get(index).unwrap_or(&Value::None)
}

fn bad_argument(method: &str, value: &Value) -> EvalError {
    EvalError::internal(Message::with(
        crate::message!("{0} cannot take {1}"),
        [Arg::from(method), Arg::from(value.kind())],
    ))
}

fn whole(method: &str, value: &Value) -> Result<i64, EvalError> {
    match value {
        Value::Int(number) => number.to_i64().ok_or_else(EvalError::overflow),
        Value::Number(number) if number.fract().is_zero() => {
            number.to_i64().ok_or_else(EvalError::overflow)
        }
        other => Err(bad_argument(method, other)),
    }
}

fn places(method: &str, args: &[Value]) -> Result<u32, EvalError> {
    match args.first() {
        None | Some(Value::None) => Ok(0),
        Some(value) => {
            let count = whole(method, value)?;
            u32::try_from(count)
                .ok()
                .filter(|places| *places <= 28)
                .ok_or_else(|| {
                    EvalError::with(
                        Code::Argument,
                        crate::message!("{0} takes 0 to 28 places, not {1}"),
                        [Arg::from(method), Arg::from(value.display())],
                    )
                })
        }
    }
}

/// A property of a number, quantity, date or date-time, or `None` when it has none by that
/// name.
#[must_use]
pub fn property(value: &Value, name: &str) -> Option<Value> {
    match (value, name) {
        (Value::Quantity(quantity), "value") => Some(Value::Number(quantity.value.normalize())),
        (Value::Date(date), _) => date_property(*date, name),
        (Value::DateTime(at), "hour") => Some(Value::int(at.hour())),
        (Value::DateTime(at), "minute") => Some(Value::int(at.minute())),
        (Value::DateTime(at), "date") => Some(Value::Date(at.date())),
        (Value::DateTime(at), _) => date_property(at.date(), name),
        _ => None,
    }
}

fn date_property(date: Date, name: &str) -> Option<Value> {
    let civil = date.civil();
    Some(match name {
        "year" => Value::int(civil.year),
        "month" => Value::int(civil.month),
        "day" => Value::int(civil.day),
        "weekday" => Value::variant(date.weekday()),
        "weekend" => Value::Bool(date.weekend()),
        _ => return None,
    })
}

/// Calls a method on a number, quantity, date or date-time, with its arguments in parameter
/// order and, for `in`, the unit read from the tree. `None` when the receiver has no method by
/// that name.
#[must_use]
pub fn call(
    receiver: &Value,
    name: &str,
    args: &[Value],
    unit: Option<&Unit>,
) -> Option<Result<Value, EvalError>> {
    match receiver {
        Value::Int(number) | Value::Number(number) => number_method(receiver, *number, name, args),
        Value::Quantity(quantity) => quantity_method(receiver, quantity, name, args, unit),
        Value::Date(date) => date_method(*date, name, args),
        Value::DateTime(at) => date_time_method(*at, name, args),
        _ => None,
    }
}

fn number_method(
    receiver: &Value,
    number: Decimal,
    name: &str,
    args: &[Value],
) -> Option<Result<Value, EvalError>> {
    let rounded = |strategy| -> Result<Value, EvalError> {
        let places = places(name, args)?;
        let value = number.round_dp_with_strategy(places, strategy).normalize();
        Ok(if places == 0 {
            Value::Int(value)
        } else {
            Value::Number(value)
        })
    };
    Some(match name {
        "round" => rounded(RoundingStrategy::MidpointAwayFromZero),
        "roundHalfEven" => rounded(RoundingStrategy::MidpointNearestEven),
        "floor" => Ok(Value::Int(number.floor().normalize())),
        "ceil" => Ok(Value::Int(number.ceil().normalize())),
        "abs" => Ok(same_kind(receiver, number.abs())),
        "min" | "max" | "clamp" => choose(receiver, name, args),
        "pow" => pow(receiver, number, args),
        "sqrt" => {
            if number.is_sign_negative() && !number.is_zero() {
                Err(EvalError::with(
                    Code::Argument,
                    crate::message!("{0} has no square root"),
                    [Arg::from(receiver.display())],
                ))
            } else {
                number
                    .sqrt()
                    .map(|root| Value::Number(root.normalize()))
                    .ok_or_else(EvalError::overflow)
            }
        }
        _ => return None,
    })
}

fn same_kind(receiver: &Value, number: Decimal) -> Value {
    match receiver {
        Value::Int(_) => Value::Int(number.normalize()),
        _ => Value::Number(number.normalize()),
    }
}

fn pow(receiver: &Value, number: Decimal, args: &[Value]) -> Result<Value, EvalError> {
    let exponent = whole("pow", arg(args, 0))?;
    if number.is_zero() && exponent < 0 {
        return Err(EvalError::divide_by_zero());
    }
    let value = number
        .checked_powi(exponent)
        .ok_or_else(EvalError::overflow)?
        .normalize();
    Ok(if matches!(receiver, Value::Int(_)) && exponent >= 0 {
        Value::Int(value)
    } else {
        Value::Number(value)
    })
}

/// `min(x)`, `max(x)` and `clamp(lo, hi)` on anything ordered. Mixing an `Int` with a `Number`
/// gives a `Number`.
fn choose(receiver: &Value, name: &str, args: &[Value]) -> Result<Value, EvalError> {
    let pick = |left: &Value, right: &Value, keep_greater: bool| -> Result<Value, EvalError> {
        let order = left.compare(right)?;
        let chosen = if (order == Ordering::Less) == keep_greater {
            right
        } else {
            left
        };
        Ok(widen(chosen, left, right))
    };
    match name {
        "min" => pick(receiver, arg(args, 0), false),
        "max" => pick(receiver, arg(args, 0), true),
        _ => {
            let (low, high) = (arg(args, 0), arg(args, 1));
            if low.compare(high)? == Ordering::Greater {
                return Err(EvalError::with(
                    Code::Argument,
                    crate::message!("clamp needs its low bound {0} at or below its high bound {1}"),
                    [Arg::from(low.display()), Arg::from(high.display())],
                ));
            }
            let raised = pick(receiver, low, true)?;
            pick(&raised, high, false)
        }
    }
}

fn widen(chosen: &Value, left: &Value, right: &Value) -> Value {
    let number = |value: &Value| matches!(value, Value::Number(_));
    match chosen {
        Value::Int(value) if number(left) || number(right) => Value::Number(*value),
        other => other.clone(),
    }
}

fn quantity_value(value: Decimal, unit: &Unit) -> Value {
    Value::Quantity(Rc::new(Quantity::new(value.normalize(), unit.clone())))
}

fn quantity_method(
    receiver: &Value,
    quantity: &Quantity,
    name: &str,
    args: &[Value],
    unit: Option<&Unit>,
) -> Option<Result<Value, EvalError>> {
    let money = quantity.unit.dim().exponent(Base::Money) == 1;
    Some(match name {
        "in" => convert(quantity, unit),
        "round" | "roundHalfEven" if money && args.is_empty() => {
            let strategy = if name == "round" {
                RoundingStrategy::MidpointAwayFromZero
            } else {
                RoundingStrategy::MidpointNearestEven
            };
            Ok(quantity_value(
                quantity.value.round_dp_with_strategy(2, strategy),
                &quantity.unit,
            ))
        }
        "round" => to_multiple(quantity, name, args, RoundingStrategy::MidpointAwayFromZero),
        "roundUp" => to_multiple(quantity, name, args, RoundingStrategy::ToPositiveInfinity),
        "roundDown" => to_multiple(quantity, name, args, RoundingStrategy::ToNegativeInfinity),
        "abs" => Ok(quantity_value(quantity.value.abs(), &quantity.unit)),
        "min" | "max" | "clamp" => choose(receiver, name, args),
        _ => return None,
    })
}

/// `q.in(unit)`: the quantity in another unit of its dimension. A rate also converts by its
/// denominator: `$0.185/lb`.in(cwt) is `$18.50/cwt` ([`Unit::conversion_target`]).
fn convert(quantity: &Quantity, unit: Option<&Unit>) -> Result<Value, EvalError> {
    let Some(unit) = unit else {
        return Err(EvalError::internal(Message::new(crate::message!(
            "in needs a unit, such as cwt or lb/ft3"
        ))));
    };
    if let Some(target) = Unit::conversion_target(quantity.unit.dim(), unit) {
        return quantity
            .to(&target)
            .map(|q| quantity_value(q.value, &q.unit))
            .map_err(|e| unit_error(&e));
    }
    Err(EvalError::with(
        Code::Conversion,
        crate::message!("{0} cannot be converted to {1}"),
        [
            Arg::from(format_quantity(quantity)),
            Arg::from(unit.to_string()),
        ],
    ))
}

fn to_multiple(
    quantity: &Quantity,
    name: &str,
    args: &[Value],
    strategy: RoundingStrategy,
) -> Result<Value, EvalError> {
    let Value::Quantity(step) = arg(args, 0) else {
        return Err(bad_argument(name, arg(args, 0)));
    };
    let increment = step
        .to(&quantity.unit)
        .map_err(|error| unit_error(&error))?
        .value;
    if increment <= Decimal::ZERO {
        return Err(EvalError::with(
            Code::Argument,
            crate::message!("{0} needs an increment above zero, not {1}"),
            [Arg::from(name), Arg::from(format_quantity(step))],
        ));
    }
    let steps = quantity
        .value
        .checked_div(increment)
        .ok_or_else(EvalError::overflow)?
        .round_dp_with_strategy(0, strategy);
    let value = steps
        .checked_mul(increment)
        .ok_or_else(EvalError::overflow)?;
    Ok(quantity_value(value, &quantity.unit))
}

fn date_method(date: Date, name: &str, args: &[Value]) -> Option<Result<Value, EvalError>> {
    match name {
        "addDays" => Some(whole(name, arg(args, 0)).and_then(|days| {
            date.add_days(days)
                .map(Value::Date)
                .ok_or_else(EvalError::overflow)
        })),
        _ => None,
    }
}

fn date_time_method(at: DateTime, name: &str, args: &[Value]) -> Option<Result<Value, EvalError>> {
    let shift = |seconds: Result<i64, EvalError>| {
        seconds.and_then(|seconds| {
            at.add_seconds(seconds)
                .map(Value::DateTime)
                .ok_or_else(EvalError::overflow)
        })
    };
    match name {
        "addDays" => Some(shift(whole(name, arg(args, 0)).and_then(|days| {
            days.checked_mul(86_400).ok_or_else(EvalError::overflow)
        }))),
        "addHours" => Some(shift(hours_in_seconds(arg(args, 0)))),
        _ => None,
    }
}

/// A whole count of hours, as seconds.
fn hours_in_seconds(value: &Value) -> Result<i64, EvalError> {
    whole("addHours", value)?
        .checked_mul(3600)
        .ok_or_else(EvalError::overflow)
}

#[cfg(test)]
mod tests {
    use super::{call, property};
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

    fn run(receiver: &Value, name: &str, args: &[Value]) -> String {
        call(receiver, name, args, None).map_or_else(
            || String::from("unknown"),
            |result| result.map_or_else(|error| error.code.as_str().to_owned(), |v| v.display()),
        )
    }

    #[test]
    fn numbers_round_and_compare() {
        assert_eq!(run(&Value::Number(d("2.5")), "round", &[]), "3");
        assert_eq!(run(&Value::Number(d("2.5")), "roundHalfEven", &[]), "2");
        assert_eq!(
            run(&Value::Number(d("2.345")), "round", &[Value::int(2)]),
            "2.35"
        );
        assert_eq!(run(&Value::Number(d("-2.5")), "floor", &[]), "-3");
        assert_eq!(run(&Value::int(2), "pow", &[Value::int(10)]), "1024");
        assert_eq!(
            run(&Value::int(5), "clamp", &[Value::int(1), Value::int(3)]),
            "3"
        );
        assert_eq!(run(&Value::int(-4), "sqrt", &[]), "argument");
    }

    #[test]
    fn quantities_round_to_increments_in_their_own_unit() {
        assert_eq!(
            run(&q("1240", "lb"), "roundUp", &[q("100", "lb")]),
            "1300 lb"
        );
        assert_eq!(
            run(&q("1240", "lb"), "roundUp", &[q("1", "cwt")]),
            "1300 lb"
        );
        assert_eq!(
            run(
                &Value::money(d("451.2")),
                "roundUp",
                &[Value::money(d("5"))]
            ),
            "$455.00"
        );
        assert_eq!(run(&Value::money(d("2.345")), "round", &[]), "$2.35");
        let cwt = Unit::parse("cwt").expect("a unit");
        assert_eq!(
            call(&q("0.185", "$/lb"), "in", &[], Some(&cwt)).map(|r| r.map(|v| v.display()).ok()),
            Some(Some(String::from("$18.50/cwt")))
        );
    }

    #[test]
    fn date_times_read_their_own_clock() {
        let at =
            Value::DateTime(DateTime::parse("2026-10-24T18:30:00-05:00").expect("a date-time"));
        assert_eq!(
            property(&at, "hour").map(|v| v.display()).as_deref(),
            Some("18")
        );
        assert_eq!(
            property(&at, "weekday").map(|v| v.display()).as_deref(),
            Some("Sat")
        );
        assert_eq!(
            property(&at, "weekend").map(|v| v.display()).as_deref(),
            Some("true")
        );
        assert_eq!(
            run(&at, "addHours", &[Value::Int(d("2"))]),
            "2026-10-24T20:30:00-05:00"
        );
    }
}
