//! The operator table: what each arithmetic, comparison and equality operator gives for the
//! types of its operands, following the reference's "Units and dimensions".
//!
//! - `+` and `-` take one dimension; `Percent ± Percent` is a `Percent`; `Int ± Int` an `Int`.
//! - `*` and `/` combine dimensions with [`Dim::times`] and [`Dim::per`]: a rate times the
//!   quantity it is per is money, money over a quantity is a rate, a quantity over the same
//!   dimension is a `Number`, and any quantity times or over a number keeps its dimension. The
//!   result must be a dimension a type can name: a single dimension, one per another, or none.
//! - `Money × Number|Percent|Int` and `Money / Number` are money; `Percent × Percent` has no
//!   meaning; a percent of a `Number` is a `Number`, and a whole number of percents (`4 ×
//!   0.5%`) is a `Percent`.
//! - `Int / Int` is a `Number`; `%` takes `Int`s only.
//! - Temperatures only compare and subtract.
//! - `Date - Date` and `DateTime - DateTime` are durations.
//!
//! Optionals are handled before these rules: an operand that may be empty is the `optional`
//! error where it is written.

use super::describe::{quotient, single};
use crate::tariff::syntax::ast::BinaryOp;
use crate::tariff::types::{QuantityType, Type};
use crate::tariff::units::{Base, Dim, Unit};

/// Why an operator does not apply to its operands.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Problem {
    /// The operands are kinds the operator does not combine.
    Kinds,
    /// `Percent × Percent`.
    PercentTimesPercent,
    /// A temperature added, multiplied or divided.
    Temperature,
    /// `%` on something other than two `Int`s.
    Remainder,
    /// An order comparison on values that have no order.
    Unordered,
    /// A product or quotient whose dimension no type can name (`Weight × Weight`).
    Unnamed,
}

/// The type of `left op right` for an arithmetic operator (`+ - * / %`).
///
/// # Errors
///
/// The [`Problem`] when the operator does not apply.
pub fn arithmetic(op: BinaryOp, left: &Type, right: &Type) -> Result<Type, Problem> {
    if matches!(left, Type::Error | Type::Never) || matches!(right, Type::Error | Type::Never) {
        return Ok(Type::Error);
    }
    match op {
        BinaryOp::Add | BinaryOp::Sub => additive(op, left, right),
        BinaryOp::Mul => multiply(left, right),
        BinaryOp::Div => divide(left, right),
        BinaryOp::Rem => match (left, right) {
            (Type::Int, Type::Int) => Ok(Type::Int),
            _ => Err(Problem::Remainder),
        },
        _ => Err(Problem::Kinds),
    }
}

fn additive(op: BinaryOp, left: &Type, right: &Type) -> Result<Type, Problem> {
    match (left, right) {
        (Type::Int, Type::Int) => Ok(Type::Int),
        (Type::Int | Type::Number, Type::Int | Type::Number) => Ok(Type::Number),
        (Type::Percent, Type::Percent) => Ok(Type::Percent),
        (Type::Quantity(a), Type::Quantity(b)) if a.dim == b.dim => {
            if a.dim.has_temperature() && op == BinaryOp::Add {
                return Err(Problem::Temperature);
            }
            Ok(Type::Quantity(QuantityType {
                dim: a.dim,
                display: a.display.clone().or_else(|| b.display.clone()),
            }))
        }
        (Type::Date, Type::Date) | (Type::DateTime, Type::DateTime) if op == BinaryOp::Sub => {
            Ok(Type::of(Base::Duration))
        }
        _ => Err(Problem::Kinds),
    }
}

/// Whether a quantity is money shown per a count (`$75/stop`), which a count multiplies into
/// plain money.
fn counted(quantity: &QuantityType) -> bool {
    quantity.dim == Dim::money()
        && quantity.display.as_ref().is_some_and(|unit| {
            unit.factors()
                .iter()
                .any(|(simple, exponent)| *exponent < 0 && simple.base.is_none())
        })
}

/// Whether a type is a plain number of either kind.
const fn number(ty: &Type) -> bool {
    matches!(ty, Type::Int | Type::Number)
}

fn multiply(left: &Type, right: &Type) -> Result<Type, Problem> {
    match (left, right) {
        (Type::Int, Type::Int) => Ok(Type::Int),
        (l, r) if number(l) && number(r) => Ok(Type::Number),
        (Type::Percent, Type::Percent) => Err(Problem::PercentTimesPercent),
        (Type::Percent, Type::Int) | (Type::Int, Type::Percent) => Ok(Type::Percent),
        (Type::Percent, Type::Number) | (Type::Number, Type::Percent) => Ok(Type::Number),
        (Type::Quantity(q), other) | (other, Type::Quantity(q))
            if number(other) || matches!(other, Type::Percent) =>
        {
            if q.dim.has_temperature() {
                return Err(Problem::Temperature);
            }
            if counted(q) {
                return Ok(Type::money());
            }
            Ok(Type::Quantity(q.clone()))
        }
        (Type::Quantity(a), Type::Quantity(b)) => {
            if a.dim.has_temperature() || b.dim.has_temperature() {
                return Err(Problem::Temperature);
            }
            named(a.dim.times(b.dim), None)
        }
        _ => Err(Problem::Kinds),
    }
}

fn divide(left: &Type, right: &Type) -> Result<Type, Problem> {
    match (left, right) {
        (l, r) if number(l) && number(r) => Ok(Type::Number),
        (Type::Percent, Type::Percent) => Ok(Type::Number),
        (Type::Percent, Type::Int) => Ok(Type::Percent),
        (Type::Percent, Type::Number) | (Type::Int | Type::Number, Type::Percent) => {
            Ok(Type::Number)
        }
        (Type::Quantity(q), other) if number(other) || matches!(other, Type::Percent) => {
            if q.dim.has_temperature() {
                return Err(Problem::Temperature);
            }
            Ok(Type::Quantity(q.clone()))
        }
        (Type::Quantity(a), Type::Quantity(b)) => {
            if a.dim.has_temperature() || b.dim.has_temperature() {
                return Err(Problem::Temperature);
            }
            named(a.dim.per(b.dim), rate_unit(a, b))
        }
        (other, Type::Quantity(q)) if number(other) => {
            if q.dim.has_temperature() {
                return Err(Problem::Temperature);
            }
            named(Dim::NONE.per(q.dim), None)
        }
        _ => Err(Problem::Kinds),
    }
}

/// The unit a quotient is shown in when both sides are simple units: `$450 / 300 mi` is shown
/// per mile.
fn rate_unit(top: &QuantityType, bottom: &QuantityType) -> Option<Unit> {
    let simple = |quantity: &QuantityType| -> Option<String> {
        let unit = quantity.display.as_ref()?;
        match unit.factors() {
            [(simple, 1)] => Some(simple.symbol.to_owned()),
            _ => None,
        }
    };
    let top_text = simple(top).or_else(|| (top.dim == Dim::money()).then(|| String::from("$")))?;
    let bottom_text = simple(bottom)?;
    Unit::parse(&format!("{top_text}/{bottom_text}")).ok()
}

/// The type of a product or quotient's dimension, when a type can name it.
fn named(dim: Option<Dim>, display: Option<Unit>) -> Result<Type, Problem> {
    let dim = dim.ok_or(Problem::Unnamed)?;
    if dim.is_none() {
        return Ok(Type::Number);
    }
    if single(dim).is_some() || quotient(dim).is_some() {
        return Ok(match display {
            Some(unit) if unit.dim() == dim => Type::quantity_in(&unit),
            _ => Type::quantity(dim),
        });
    }
    Err(Problem::Unnamed)
}

/// Whether values of a type have an order (`<`, `sortBy`, `min`, ranges): numbers, percents,
/// money and quantities, dates and date-times.
#[must_use]
pub const fn ordered(ty: &Type) -> bool {
    matches!(
        ty,
        Type::Int
            | Type::Number
            | Type::Percent
            | Type::Quantity(_)
            | Type::Date
            | Type::DateTime
            | Type::Error
            | Type::Never
    )
}

/// Checks an order comparison (`< <= > >=`) of two values.
///
/// # Errors
///
/// [`Problem::Unordered`] for values with no order, [`Problem::Kinds`] for two different kinds.
pub fn compare(left: &Type, right: &Type) -> Result<(), Problem> {
    if matches!(left, Type::Error | Type::Never) || matches!(right, Type::Error | Type::Never) {
        return Ok(());
    }
    if !ordered(left) || !ordered(right) {
        return if left == right {
            Err(Problem::Unordered)
        } else {
            Err(Problem::Kinds)
        };
    }
    if comparable(left, right) {
        Ok(())
    } else {
        Err(Problem::Kinds)
    }
}

/// Whether two ordered values are of one kind: two numbers, two percents, two quantities of one
/// dimension, two dates or two date-times.
fn comparable(left: &Type, right: &Type) -> bool {
    match (left, right) {
        (l, r) if number(l) && number(r) => true,
        (Type::Quantity(a), Type::Quantity(b)) => a.dim == b.dim,
        (l, r) => l == r,
    }
}

/// Checks an equality (`==`, `!=`): two values of one type, where `T` and `T?` are one type
/// and `Int` widens to `Number`.
///
/// # Errors
///
/// [`Problem::Kinds`] when the types differ.
pub fn equality(left: &Type, right: &Type) -> Result<(), Problem> {
    if matches!(left, Type::Error | Type::Never) || matches!(right, Type::Error | Type::Never) {
        return Ok(());
    }
    if left.join(right).is_some() {
        Ok(())
    } else {
        Err(Problem::Kinds)
    }
}

/// Whether a value of a type may be summed or averaged: numbers, percents, money and quantities
/// other than temperatures.
#[must_use]
pub fn summable(ty: &Type) -> bool {
    match ty {
        Type::Int | Type::Number | Type::Percent | Type::Error | Type::Never => true,
        Type::Quantity(quantity) => !quantity.dim.has_temperature(),
        _ => false,
    }
}

#[cfg(test)]
mod tests {
    use super::{Problem, arithmetic, compare, equality};
    use crate::tariff::syntax::ast::BinaryOp;
    use crate::tariff::types::Type;
    use crate::tariff::units::{Base, Unit};

    fn unit(text: &str) -> Type {
        Type::quantity_in(&Unit::parse(text).unwrap_or_default())
    }

    #[test]
    fn rates_times_quantities_are_money() {
        assert_eq!(
            arithmetic(BinaryOp::Mul, &unit("$/cwt"), &unit("lb")),
            Ok(Type::money())
        );
        assert_eq!(
            arithmetic(BinaryOp::Mul, &unit("lb"), &unit("$/cwt")),
            Ok(Type::money())
        );
        assert_eq!(
            arithmetic(BinaryOp::Div, &Type::money(), &unit("mi")).map(|t| t.to_string()),
            Ok(String::from("Money/mi"))
        );
        assert_eq!(
            arithmetic(BinaryOp::Div, &unit("lb"), &unit("cwt")),
            Ok(Type::Number)
        );
        assert_eq!(
            arithmetic(BinaryOp::Mul, &unit("lb"), &unit("lb")),
            Err(Problem::Unnamed)
        );
        assert_eq!(
            arithmetic(BinaryOp::Add, &Type::money(), &unit("lb")),
            Err(Problem::Kinds)
        );
    }

    #[test]
    fn numbers_percents_and_temperatures() {
        assert_eq!(
            arithmetic(BinaryOp::Div, &Type::Int, &Type::Int),
            Ok(Type::Number)
        );
        assert_eq!(
            arithmetic(BinaryOp::Rem, &Type::Number, &Type::Int),
            Err(Problem::Remainder)
        );
        assert_eq!(
            arithmetic(BinaryOp::Mul, &Type::Percent, &Type::Percent),
            Err(Problem::PercentTimesPercent)
        );
        assert_eq!(
            arithmetic(BinaryOp::Mul, &Type::Number, &Type::Percent),
            Ok(Type::Number)
        );
        assert_eq!(
            arithmetic(BinaryOp::Mul, &Type::Percent, &Type::Int),
            Ok(Type::Percent)
        );
        assert_eq!(
            arithmetic(BinaryOp::Add, &unit("°F"), &unit("°C")),
            Err(Problem::Temperature)
        );
        assert_eq!(
            arithmetic(BinaryOp::Sub, &unit("°F"), &unit("°C")),
            Ok(unit("°F"))
        );
        assert_eq!(
            arithmetic(BinaryOp::Sub, &Type::DateTime, &Type::DateTime),
            Ok(Type::of(Base::Duration))
        );
        assert_eq!(compare(&Type::Text, &Type::Text), Err(Problem::Unordered));
        assert_eq!(compare(&unit("lb"), &unit("kg")), Ok(()));
        assert_eq!(
            equality(&Type::optional(Type::money()), &Type::money()),
            Ok(())
        );
    }
}
