//! Compile-time values: `const` declarations and `input` defaults are folded here, so the
//! checker can refuse one that needs a record to compute, and `tariff.check` can report each
//! input's default as a wire value.
//!
//! A constant expression is a literal, `none`, an enum variant, another constant, a list or a
//! record of constant expressions, negation and the arithmetic operators on them (with the same
//! unit algebra the evaluator uses). Text is constant when it has no interpolation.

use crate::span::Span;
use crate::tariff::syntax::ast::{
    BinaryOp, DateLit, DateTimeLit, Expr, ExprKind, NodeId, TimeLit, UnaryOp,
};
use crate::tariff::types::Type;
use crate::tariff::units::{Quantity, Unit};
use rust_decimal::Decimal;
use serde_json::{Map, Value as Json, json};
use std::fmt::Write as _;

/// A folded value.
#[derive(Debug, Clone, PartialEq)]
pub enum Value {
    /// `true` or `false`.
    Bool(bool),
    /// A whole number.
    Int(Decimal),
    /// A decimal.
    Number(Decimal),
    /// A percent, in points: `18%` is 18.
    Percent(Decimal),
    /// Money, a rate or a quantity, with its unit.
    Quantity(Quantity),
    /// Text.
    Text(String),
    /// A date.
    Date(DateLit),
    /// A date-time.
    DateTime(DateTimeLit),
    /// A time of day.
    Time(TimeLit),
    /// An enum variant, by name.
    Enum(String),
    /// `none`.
    None,
    /// A list.
    List(Vec<Value>),
    /// A record's fields, in the order written.
    Record(Vec<(String, Value)>),
}

/// Why an expression has no compile-time value.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FoldError {
    /// The part at this span needs a record, a call or something else computed at run time.
    NotConstant(Span),
    /// The arithmetic at this span overflows 28 significant digits.
    Overflow(Span),
    /// The division at this span divides by zero.
    DivideByZero(Span),
    /// A constant this one reads has no value; that one is reported where it is declared.
    Upstream,
}

/// Folds expressions to values.
#[allow(missing_debug_implementations)]
pub struct Folder<'a> {
    /// The value of a constant by the name it is read by, or `None` when the name is not a
    /// constant; `Err` when that constant has no value.
    pub constant: &'a dyn Fn(&Expr) -> Option<Result<Value, FoldError>>,
    /// The checked type of an expression node.
    pub type_of: &'a dyn Fn(NodeId) -> Option<Type>,
}

impl Folder<'_> {
    /// The value of an expression.
    ///
    /// # Errors
    ///
    /// When it is not a compile-time value, overflows or divides by zero.
    pub fn fold(&self, expr: &Expr) -> Result<Value, FoldError> {
        match &expr.kind {
            ExprKind::Number { value, unit, .. } => Ok(self.number(expr, *value, unit.as_ref())),
            ExprKind::Money { value, per } => {
                let text = per
                    .as_ref()
                    .map_or_else(|| String::from("$"), |per| format!("$/{}", per.text()));
                let unit = Unit::parse(&text).map_err(|_| FoldError::NotConstant(expr.span))?;
                Ok(Value::Quantity(Quantity::new(*value, unit)))
            }
            ExprKind::Percent(points) => Ok(Value::Percent(*points)),
            ExprKind::Text(text) => text
                .literal()
                .map(Value::Text)
                .ok_or(FoldError::NotConstant(expr.span)),
            ExprKind::Date(date) => Ok(Value::Date(*date)),
            ExprKind::DateTime(at) => Ok(Value::DateTime(*at)),
            ExprKind::Time(time) => Ok(Value::Time(*time)),
            ExprKind::Bool(value) => Ok(Value::Bool(*value)),
            ExprKind::None => Ok(Value::None),
            ExprKind::Path { head, member } => match (self.type_of)(expr.id) {
                Some(Type::Enum(_)) => Ok(Value::Enum(
                    member
                        .as_ref()
                        .map_or_else(|| head.text.clone(), |m| m.text.clone()),
                )),
                _ => Err(FoldError::NotConstant(expr.span)),
            },
            ExprKind::Name(_) | ExprKind::Field { .. } => {
                (self.constant)(expr).unwrap_or(Err(FoldError::NotConstant(expr.span)))
            }
            ExprKind::List(items) => items
                .iter()
                .map(|item| self.fold(item))
                .collect::<Result<Vec<_>, _>>()
                .map(Value::List),
            ExprKind::Record(fields) => fields
                .iter()
                .map(|field| Ok((field.name.text.clone(), self.fold(&field.value)?)))
                .collect::<Result<Vec<_>, FoldError>>()
                .map(Value::Record),
            ExprKind::Unary {
                op: UnaryOp::Neg,
                operand,
                ..
            } => negate(self.fold(operand)?, expr.span),
            ExprKind::Unary {
                op: UnaryOp::Not,
                operand,
                ..
            } => match self.fold(operand)? {
                Value::Bool(value) => Ok(Value::Bool(!value)),
                _ => Err(FoldError::NotConstant(expr.span)),
            },
            ExprKind::Binary {
                op, left, right, ..
            } => {
                let left_value = self.fold(left)?;
                let right_value = self.fold(right)?;
                let result = (self.type_of)(expr.id).unwrap_or(Type::Error);
                binary(*op, &left_value, &right_value, &result, expr.span)
            }
            _ => Err(FoldError::NotConstant(expr.span)),
        }
    }

    fn number(
        &self,
        expr: &Expr,
        value: Decimal,
        unit: Option<&crate::tariff::syntax::ast::UnitExpr>,
    ) -> Value {
        let integer = matches!((self.type_of)(expr.id), Some(Type::Int));
        match unit.and_then(|unit| Unit::parse(&unit.text()).ok()) {
            Some(unit) if !unit.dim().is_none() => Value::Quantity(Quantity::new(value, unit)),
            _ if integer => Value::Int(value),
            _ => Value::Number(value),
        }
    }
}

fn negate(value: Value, span: Span) -> Result<Value, FoldError> {
    Ok(match value {
        Value::Int(value) => Value::Int(-value),
        Value::Number(value) => Value::Number(-value),
        Value::Percent(value) => Value::Percent(-value),
        Value::Quantity(quantity) => Value::Quantity(Quantity::new(-quantity.value, quantity.unit)),
        _ => return Err(FoldError::NotConstant(span)),
    })
}

/// A plain number of a value, when it is one.
const fn plain(value: &Value) -> Option<Decimal> {
    match value {
        Value::Int(value) | Value::Number(value) => Some(*value),
        _ => None,
    }
}

fn hundred() -> Decimal {
    Decimal::ONE_HUNDRED
}

fn arithmetic(op: BinaryOp, a: Decimal, b: Decimal, span: Span) -> Result<Decimal, FoldError> {
    let out = match op {
        BinaryOp::Add => a.checked_add(b),
        BinaryOp::Sub => a.checked_sub(b),
        BinaryOp::Mul => a.checked_mul(b),
        BinaryOp::Div | BinaryOp::Rem if b.is_zero() => return Err(FoldError::DivideByZero(span)),
        BinaryOp::Div => a.checked_div(b),
        BinaryOp::Rem => a.checked_rem(b),
        _ => return Err(FoldError::NotConstant(span)),
    };
    out.map(|value| value.normalize())
        .ok_or(FoldError::Overflow(span))
}

fn as_quantity(value: &Value) -> Option<Quantity> {
    match value {
        Value::Quantity(quantity) => Some(quantity.clone()),
        Value::Int(value) | Value::Number(value) => Some(Quantity::new(*value, Unit::none())),
        Value::Percent(points) => Unit::parse("%")
            .ok()
            .map(|unit| Quantity::new(*points, unit)),
        _ => None,
    }
}

fn binary(
    op: BinaryOp,
    left: &Value,
    right: &Value,
    result: &Type,
    span: Span,
) -> Result<Value, FoldError> {
    let percent_points = |value: Decimal| match result {
        Type::Percent => Ok(Value::Percent(value)),
        Type::Int => Ok(Value::Int(value)),
        _ => Ok(Value::Number(value)),
    };
    match (left, right) {
        (Value::Int(a), Value::Int(b)) if op != BinaryOp::Div => {
            Ok(Value::Int(arithmetic(op, *a, *b, span)?))
        }
        (l, r) if plain(l).is_some() && plain(r).is_some() => Ok(Value::Number(arithmetic(
            op,
            plain(l).unwrap_or_default(),
            plain(r).unwrap_or_default(),
            span,
        )?)),
        (Value::Percent(a), Value::Percent(b)) if matches!(op, BinaryOp::Add | BinaryOp::Sub) => {
            Ok(Value::Percent(arithmetic(op, *a, *b, span)?))
        }
        (Value::Percent(a), Value::Percent(b)) if op == BinaryOp::Div => {
            Ok(Value::Number(arithmetic(op, *a, *b, span)?))
        }
        (Value::Percent(points), other) | (other, Value::Percent(points))
            if plain(other).is_some() && op == BinaryOp::Mul =>
        {
            let product = arithmetic(op, *points, plain(other).unwrap_or_default(), span)?;
            match result {
                Type::Percent => percent_points(product),
                _ => Ok(Value::Number(arithmetic(
                    BinaryOp::Div,
                    product,
                    hundred(),
                    span,
                )?)),
            }
        }
        (Value::Percent(points), other) if plain(other).is_some() && op == BinaryOp::Div => {
            let quotient = arithmetic(op, *points, plain(other).unwrap_or_default(), span)?;
            match result {
                Type::Percent => percent_points(quotient),
                _ => Ok(Value::Number(arithmetic(
                    BinaryOp::Div,
                    quotient,
                    hundred(),
                    span,
                )?)),
            }
        }
        (other, Value::Percent(points)) if plain(other).is_some() && op == BinaryOp::Div => {
            let fraction = arithmetic(BinaryOp::Div, *points, hundred(), span)?;
            Ok(Value::Number(arithmetic(
                op,
                plain(other).unwrap_or_default(),
                fraction,
                span,
            )?))
        }
        (Value::Date(a), Value::Date(b)) if op == BinaryOp::Sub => {
            let days = days_from_civil(i64::from(a.year), i64::from(a.month), i64::from(a.day))
                - days_from_civil(i64::from(b.year), i64::from(b.month), i64::from(b.day));
            let unit = Unit::parse("d").map_err(|_| FoldError::NotConstant(span))?;
            Ok(Value::Quantity(Quantity::new(Decimal::from(days), unit)))
        }
        (Value::Time(a), Value::Time(b)) if op == BinaryOp::Sub => {
            let seconds = Decimal::from(a.seconds() - b.seconds());
            let (divisor, unit) = if (seconds % Decimal::from(36)).is_zero() {
                (Decimal::from(3_600), "h")
            } else {
                (Decimal::from(60), "min")
            };
            let unit = Unit::parse(unit).map_err(|_| FoldError::NotConstant(span))?;
            let value = arithmetic(BinaryOp::Div, seconds, divisor, span)?;
            Ok(Value::Quantity(Quantity::new(value, unit)))
        }
        _ => quantities(op, left, right, span),
    }
}

fn quantities(op: BinaryOp, left: &Value, right: &Value, span: Span) -> Result<Value, FoldError> {
    let (Some(a), Some(b)) = (as_quantity(left), as_quantity(right)) else {
        return Err(FoldError::NotConstant(span));
    };
    let out = match op {
        BinaryOp::Add => a.add(&b),
        BinaryOp::Sub => a.sub(&b),
        BinaryOp::Mul => a.mul(&b),
        BinaryOp::Div if b.value.is_zero() => return Err(FoldError::DivideByZero(span)),
        BinaryOp::Div => a.div(&b),
        _ => return Err(FoldError::NotConstant(span)),
    }
    .map_err(|_| FoldError::Overflow(span))?;
    if out.unit.is_none() {
        return Ok(Value::Number(out.value.normalize()));
    }
    Ok(Value::Quantity(Quantity::new(
        out.value.normalize(),
        out.unit,
    )))
}

/// Orders two folded values of one kind, for checks such as `minimum` above `maximum` and an
/// empty range: numbers, percents, quantities, dates, date-times, times of day, text (in
/// Unicode scalar order, as ranges compare it) and the variants of an ordered enum.
#[must_use]
pub fn compare(a: &Value, b: &Value) -> Option<std::cmp::Ordering> {
    match (a, b) {
        (Value::Quantity(x), Value::Quantity(y)) => x.compare(y).ok(),
        (Value::Percent(x), Value::Percent(y)) => Some(x.cmp(y)),
        (Value::Date(x), Value::Date(y)) => Some(date_ordinal(*x).cmp(&date_ordinal(*y))),
        (Value::DateTime(x), Value::DateTime(y)) => {
            Some(date_time_ordinal(*x).cmp(&date_time_ordinal(*y)))
        }
        (Value::Time(x), Value::Time(y)) => Some(x.seconds().cmp(&y.seconds())),
        (Value::Text(x), Value::Text(y)) => Some(x.cmp(y)),
        (Value::Enum(x), Value::Enum(y)) => Some(
            crate::tariff::schema::variant_rank(x)?.cmp(&crate::tariff::schema::variant_rank(y)?),
        ),
        (x, y) => Some(plain(x)?.cmp(&plain(y)?)),
    }
}

/// The value in the reference's wire form: `{"$n": "1200"}`, `{"$q": "2.35", "u": "$/mi"}`,
/// `{"$date": "2026-10-22"}`, `{"$time": "17:30:00"}`, variants as strings, `none` as `null`.
#[must_use]
pub fn wire(value: &Value) -> Json {
    match value {
        Value::Bool(value) => Json::Bool(*value),
        Value::Int(value) | Value::Number(value) => json!({ "$n": value.normalize().to_string() }),
        Value::Percent(points) => json!({ "$q": points.normalize().to_string(), "u": "%" }),
        Value::Quantity(quantity) => json!({
            "$q": quantity.value.normalize().to_string(),
            "u": quantity.unit.to_string(),
        }),
        Value::Text(text) | Value::Enum(text) => Json::String(text.clone()),
        Value::Date(date) => json!({ "$date": date_text(*date) }),
        Value::DateTime(at) => json!({ "$date": date_time_text(*at) }),
        Value::Time(time) => json!({ "$time": time_text(*time) }),
        Value::None => Json::Null,
        Value::List(items) => Json::Array(items.iter().map(wire).collect()),
        Value::Record(fields) => {
            let mut map = Map::new();
            for (name, value) in fields {
                map.insert(name.clone(), wire(value));
            }
            Json::Object(map)
        }
    }
}

/// `2026-10-22`.
#[must_use]
pub fn date_text(date: DateLit) -> String {
    format!("{:04}-{:02}-{:02}", date.year, date.month, date.day)
}

/// `2026-10-22T08:00:00-05:00`, or `…Z` at offset zero.
#[must_use]
pub fn date_time_text(at: DateTimeLit) -> String {
    let mut out = format!(
        "{}T{:02}:{:02}:{:02}",
        date_text(at.date),
        at.hour,
        at.minute,
        at.second
    );
    if at.offset_minutes == 0 {
        out.push('Z');
    } else {
        let sign = if at.offset_minutes < 0 { '-' } else { '+' };
        let minutes = at.offset_minutes.unsigned_abs();
        let _ = write!(out, "{sign}{:02}:{:02}", minutes / 60, minutes % 60);
    }
    out
}

/// `17:30:00`.
#[must_use]
pub fn time_text(time: TimeLit) -> String {
    format!("{:02}:{:02}:{:02}", time.hour, time.minute, time.second)
}

/// Days since 1970-01-01 of a proleptic Gregorian date.
#[must_use]
pub fn days_from_civil(year: i64, month: i64, day: i64) -> i64 {
    let year = if month <= 2 { year - 1 } else { year };
    let era = year.div_euclid(400);
    let year_of_era = year - era * 400;
    let month_index = (month + 9) % 12;
    let day_of_year = (153 * month_index + 2) / 5 + day - 1;
    let day_of_era = year_of_era * 365 + year_of_era / 4 - year_of_era / 100 + day_of_year;
    era * 146_097 + day_of_era - 719_468
}

/// The proleptic Gregorian date of a day count since 1970-01-01.
#[must_use]
pub fn civil_from_days(days: i64) -> (i64, i64, i64) {
    let shifted = days.saturating_add(719_468);
    let era = shifted.div_euclid(146_097);
    let day_of_era = shifted - era * 146_097;
    let year_of_era =
        (day_of_era - day_of_era / 1460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
    let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let month_index = (5 * day_of_year + 2) / 153;
    let day = day_of_year - (153 * month_index + 2) / 5 + 1;
    let month = if month_index < 10 {
        month_index + 3
    } else {
        month_index - 9
    };
    let year = year_of_era + era * 400 + i64::from(month <= 2);
    (year, month, day)
}

/// A date's position on a line of days, for date ranges in `match`.
#[must_use]
pub fn date_ordinal(date: DateLit) -> i64 {
    days_from_civil(
        i64::from(date.year),
        i64::from(date.month),
        i64::from(date.day),
    )
}

/// A date-time's position on a line of seconds since the epoch, in UTC.
#[must_use]
pub fn date_time_ordinal(at: DateTimeLit) -> i64 {
    date_ordinal(at.date) * 86_400
        + i64::from(at.hour) * 3_600
        + i64::from(at.minute) * 60
        + i64::from(at.second)
        - i64::from(at.offset_minutes) * 60
}

#[cfg(test)]
mod tests {
    use super::{Value, civil_from_days, days_from_civil, wire};
    use crate::tariff::units::{Quantity, Unit};
    use rust_decimal::Decimal;

    #[test]
    fn wire_values_follow_the_reference() {
        let rate = Value::Quantity(Quantity::new(
            Decimal::new(235, 2),
            Unit::parse("$/mi").unwrap_or_default(),
        ));
        assert_eq!(wire(&rate).to_string(), r#"{"$q":"2.35","u":"$/mi"}"#);
        assert_eq!(
            wire(&Value::Int(Decimal::from(1200))).to_string(),
            r#"{"$n":"1200"}"#
        );
        assert_eq!(
            wire(&Value::Percent(Decimal::from(18))).to_string(),
            r#"{"$q":"18","u":"%"}"#
        );
        assert_eq!(wire(&Value::None).to_string(), "null");
    }

    #[test]
    fn civil_dates_round_trip() {
        let days = days_from_civil(2026, 10, 22);
        assert_eq!(civil_from_days(days), (2026, 10, 22));
        assert_eq!(days_from_civil(1970, 1, 1), 0);
    }
}
