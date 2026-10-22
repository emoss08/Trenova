//! Exact decimal arithmetic: every operation is checked, overflow is an error, never a wrap.
//!
//! Numbers are `rust_decimal::Decimal`: 96 bits of mantissa, 28 significant digits. Division is
//! exact to the last digit that fits. `**` and `pow` are exact for whole exponents (repeated
//! squaring, `1 / x^n` for negative ones); a fractional exponent goes through `rust_decimal`'s
//! `exp(b * ln(a))`, with `x ** 0.5` taken as the square root so the common case is exact.
//! The rounding family rounds in decimal, so `round(2.675, 2)` is `2.68` and `round(1.005, 2)` is
//! `1.01`, as a person expects and binary floating point does not deliver.

use super::error::EvalError;
use super::error::divide_by_zero;
use super::error::expected_whole_value;
use super::error::non_finite;
use super::error::number_overflow;
use super::value::format_number;
use rust_decimal::Decimal;
use rust_decimal::MathematicalOps;
use rust_decimal::RoundingStrategy;

/// How a rounding function treats the digit it drops.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Rounding {
    /// Half away from zero (`round`).
    HalfAwayFromZero,
    /// Towards positive infinity (`roundUp`, `ceil`).
    Up,
    /// Towards negative infinity (`roundDown`, `floor`).
    Down,
    /// Half to even, the banker's rounding (`roundHalfEven`).
    HalfEven,
}

impl Rounding {
    const fn strategy(self) -> RoundingStrategy {
        match self {
            Self::HalfAwayFromZero => RoundingStrategy::MidpointAwayFromZero,
            Self::Up => RoundingStrategy::ToPositiveInfinity,
            Self::Down => RoundingStrategy::ToNegativeInfinity,
            Self::HalfEven => RoundingStrategy::MidpointNearestEven,
        }
    }
}

/// `a + b`.
pub fn add(left: Decimal, right: Decimal) -> Result<Decimal, EvalError> {
    left.checked_add(right).ok_or_else(number_overflow)
}

/// `a - b`.
pub fn sub(left: Decimal, right: Decimal) -> Result<Decimal, EvalError> {
    left.checked_sub(right).ok_or_else(number_overflow)
}

/// `a * b`.
pub fn mul(left: Decimal, right: Decimal) -> Result<Decimal, EvalError> {
    left.checked_mul(right).ok_or_else(number_overflow)
}

/// `a / b`; division by zero is an error.
pub fn div(left: Decimal, right: Decimal) -> Result<Decimal, EvalError> {
    if right.is_zero() {
        return Err(divide_by_zero());
    }
    left.checked_div(right).ok_or_else(number_overflow)
}

/// `a % b` on whole numbers; the sign follows the dividend, as Go's `%`.
pub fn rem(left: Decimal, right: Decimal) -> Result<Decimal, EvalError> {
    whole(left)?;
    whole(right)?;
    if right.is_zero() {
        return Err(divide_by_zero());
    }
    left.checked_rem(right).ok_or_else(number_overflow)
}

/// Fails unless the number is whole.
pub fn whole(value: Decimal) -> Result<Decimal, EvalError> {
    if value.fract().is_zero() {
        Ok(value)
    } else {
        Err(expected_whole_value(format_number(value)))
    }
}

/// The number as an `i64`, when it is whole and fits.
pub fn to_i64(value: Decimal) -> Result<i64, EvalError> {
    whole(value)?;
    i64::try_from(value.trunc()).map_err(|_| expected_whole_value(format_number(value)))
}

/// `base ** exponent`.
pub fn pow(base: Decimal, exponent: Decimal) -> Result<Decimal, EvalError> {
    if exponent.fract().is_zero() {
        let Ok(power) = i64::try_from(exponent.trunc()) else {
            return Err(number_overflow());
        };
        if base.is_zero() && power < 0 {
            return Err(non_finite());
        }
        return base.checked_powi(power).ok_or_else(number_overflow);
    }
    if base.is_sign_negative() && !base.is_zero() {
        return Err(non_finite());
    }
    let half = Decimal::new(5, 1);
    if exponent == half {
        return sqrt(base);
    }
    if exponent == -half {
        let root = sqrt(base)?;
        return div(Decimal::ONE, root).map_err(|_| non_finite());
    }
    if base.is_zero() {
        return if exponent.is_sign_negative() {
            Err(non_finite())
        } else {
            Ok(Decimal::ZERO)
        };
    }
    base.checked_powd(exponent).ok_or_else(number_overflow)
}

/// The square root of a non-negative number.
pub fn sqrt(value: Decimal) -> Result<Decimal, EvalError> {
    if value.is_zero() {
        return Ok(Decimal::ZERO);
    }
    if value.is_sign_negative() {
        return Err(EvalError::with(
            super::error::Code::Argument,
            crate::message!("sqrt of negative number {0}"),
            [crate::message::Arg::from(format_number(value))],
        ));
    }
    value.sqrt().ok_or_else(number_overflow)
}

/// Rounds to `places` decimals (negative places round to tens, hundreds and so on).
pub fn round(value: Decimal, places: i64, rounding: Rounding) -> Result<Decimal, EvalError> {
    if places >= 0 {
        let places = u32::try_from(places).unwrap_or(u32::MAX).min(28);
        return Ok(value.round_dp_with_strategy(places, rounding.strategy()));
    }
    let exponent = u32::try_from(-places).unwrap_or(u32::MAX).min(28);
    let factor = Decimal::from_i128_with_scale(10i128.pow(exponent), 0);
    let scaled = div(value, factor)?;
    mul(
        scaled.round_dp_with_strategy(0, rounding.strategy()),
        factor,
    )
}

#[cfg(test)]
mod tests {
    use super::{Rounding, div, pow, rem, round, sqrt};
    use crate::formula::eval::error::Code;
    use rust_decimal::Decimal;

    fn number(text: &str) -> Decimal {
        crate::wire::parse_decimal(text).expect("decimal")
    }

    #[test]
    fn rounding_is_decimal() {
        let cases = [
            ("2.675", 2, Rounding::HalfAwayFromZero, "2.68"),
            ("1.005", 2, Rounding::HalfAwayFromZero, "1.01"),
            ("-2.5", 0, Rounding::HalfAwayFromZero, "-3"),
            ("1234.5", -2, Rounding::HalfAwayFromZero, "1200"),
            ("2.001", 2, Rounding::Up, "2.01"),
            ("2.999", 2, Rounding::Down, "2.99"),
            ("2.665", 2, Rounding::HalfEven, "2.66"),
            ("2.675", 2, Rounding::HalfEven, "2.68"),
            ("1250", -2, Rounding::HalfEven, "1200"),
            ("3.14159", 12, Rounding::HalfAwayFromZero, "3.14159"),
            ("3.14159", -12, Rounding::HalfAwayFromZero, "0"),
        ];
        for (value, places, rounding, want) in cases {
            let got = round(number(value), places, rounding).expect("rounds");
            assert_eq!(got, number(want), "{value} {places} {rounding:?}");
        }
    }

    #[test]
    fn powers_are_exact_for_whole_exponents() {
        assert_eq!(pow(number("2"), number("3")), Ok(number("8")));
        assert_eq!(pow(number("2"), number("-1")), Ok(number("0.5")));
        assert_eq!(pow(number("9"), number("0.5")), Ok(number("3")));
        assert_eq!(pow(number("1.05"), number("2")), Ok(number("1.1025")));
        assert_eq!(
            pow(number("-8"), number("0.5")).map_err(|error| error.code),
            Err(Code::NonFinite)
        );
        assert_eq!(
            pow(number("0"), number("-1")).map_err(|error| error.code),
            Err(Code::NonFinite)
        );
        assert_eq!(
            pow(number("10"), number("40")).map_err(|error| error.code),
            Err(Code::Overflow)
        );
    }

    #[test]
    fn division_and_remainder_refuse_zero() {
        assert_eq!(
            div(Decimal::ONE, Decimal::ZERO).map_err(|error| error.code),
            Err(Code::DivideByZero)
        );
        assert_eq!(rem(number("7"), number("-3")), Ok(number("1")));
        assert_eq!(rem(number("-7"), number("3")), Ok(number("-1")));
        assert_eq!(
            rem(number("7.5"), number("2")).map_err(|error| error.code),
            Err(Code::Argument)
        );
        assert_eq!(
            sqrt(number("-4")).map_err(|error| error.code),
            Err(Code::Argument)
        );
        assert_eq!(sqrt(number("16")), Ok(number("4")));
    }
}
