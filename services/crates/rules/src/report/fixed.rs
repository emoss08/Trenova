//! Exact decimal rounding, fixed-point text and division with the semantics of Go's
//! `shopspring/decimal`, which the server's money and report code was written against.
//!
//! `rust_decimal` holds 28 significant digits and its operators round when a result needs more.
//! The rules here must not: `StringFixed(2)` of `1.005` is "1.01" because the digits are read as
//! written, and a percent shifted by two places or a compact number divided by a power of ten
//! must not lose the digit that decides the rounding. So this module works on the digits
//! themselves. [`Digits`] is an unsigned decimal of any scale (a shift only moves the point),
//! [`Digits::fixed`] rounds half away from zero to a number of places and writes it, and
//! [`div_round`] is shopspring's `Div`: the quotient rounded half away from zero to 16 places,
//! computed by long division so it is exact before it is rounded.
//!
//! Report formatting and charge allocation both use it; it lives with report formatting because
//! that is where `StringFixed` is the contract.

use rust_decimal::Decimal;
use rust_decimal::RoundingStrategy;

/// The number of places shopspring's `Div` rounds a quotient to (`DivisionPrecision`).
pub const DIVISION_PRECISION: u32 = 16;

/// An unsigned decimal as its digits and a scale: the value is `digits × 10^-scale`.
///
/// The scale may be negative or larger than 28, so shifting the point never rounds.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Digits {
    /// ASCII digits, most significant first, without leading zeros; `"0"` for zero.
    digits: Vec<u8>,
    scale: i64,
}

impl Digits {
    /// The magnitude of a decimal.
    #[must_use]
    pub fn of(value: Decimal) -> Self {
        Self::from_parts(value.mantissa().unsigned_abs(), i64::from(value.scale()))
    }

    /// `mantissa × 10^-scale`.
    #[must_use]
    pub fn from_parts(mantissa: u128, scale: i64) -> Self {
        Self {
            digits: mantissa.to_string().into_bytes(),
            scale,
        }
    }

    /// The value times `10^places` (a negative `places` divides).
    #[must_use]
    pub fn shift(mut self, places: i64) -> Self {
        self.scale -= places;
        self
    }

    /// Whether the value is zero.
    #[must_use]
    pub fn is_zero(&self) -> bool {
        self.digits.iter().all(|digit| *digit == b'0')
    }

    /// Whether the value is at least `10^power`.
    #[must_use]
    pub fn at_least_pow10(&self, power: i64) -> bool {
        if self.is_zero() {
            return false;
        }
        let length = i64::try_from(self.digits.len()).unwrap_or(i64::MAX);
        length - self.scale > power
    }

    /// The whole part, truncated toward zero, or `None` when it does not fit in a `u128`.
    #[must_use]
    pub fn whole(&self) -> Option<u128> {
        let text = self.fixed_truncated();
        text.parse().ok()
    }

    fn fixed_truncated(&self) -> String {
        if self.scale <= 0 {
            let mut text = String::from_utf8_lossy(&self.digits).into_owned();
            if !self.is_zero() {
                for _ in 0..(-self.scale) {
                    text.push('0');
                }
            }
            return text;
        }
        let drop = usize::try_from(self.scale).unwrap_or(usize::MAX);
        if drop >= self.digits.len() {
            return String::from("0");
        }
        String::from_utf8_lossy(&self.digits[..self.digits.len() - drop]).into_owned()
    }

    /// The value rounded half away from zero to `places` decimals and written with exactly that
    /// many, as `StringFixed` writes a magnitude: `"1.01"`, `"0.00"`, `"1235"`.
    #[must_use]
    pub fn fixed(&self, places: u32) -> String {
        let places = i64::from(places);
        let mut kept: Vec<u8>;
        if self.scale > places {
            let drop = usize::try_from(self.scale - places).unwrap_or(usize::MAX);
            let length = self.digits.len();
            let round_up;
            if drop > length {
                kept = Vec::new();
                round_up = false;
            } else {
                let split = length - drop;
                round_up = self.digits[split] >= b'5';
                kept = self.digits[..split].to_vec();
            }
            if round_up {
                increment(&mut kept);
            }
        } else {
            kept = self.digits.clone();
            let pad = usize::try_from(places - self.scale).unwrap_or(0);
            kept.extend(std::iter::repeat_n(b'0', pad));
        }
        let first = kept
            .iter()
            .position(|digit| *digit != b'0')
            .unwrap_or(kept.len());
        let mut kept = kept.split_off(first);
        let places = usize::try_from(places).unwrap_or(0);
        if kept.len() <= places {
            let mut padded = vec![b'0'; places + 1 - kept.len()];
            padded.append(&mut kept);
            kept = padded;
        }
        let point = kept.len() - places;
        let mut text = String::with_capacity(kept.len() + 1);
        text.push_str(&String::from_utf8_lossy(&kept[..point]));
        if places > 0 {
            text.push('.');
            text.push_str(&String::from_utf8_lossy(&kept[point..]));
        }
        text
    }
}

fn increment(digits: &mut Vec<u8>) {
    for digit in digits.iter_mut().rev() {
        if *digit == b'9' {
            *digit = b'0';
        } else {
            *digit += 1;
            return;
        }
    }
    digits.insert(0, b'1');
}

/// A signed decimal of any scale: what [`div_round`] returns.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Signed {
    /// Whether the value is below zero. Never set for zero.
    pub negative: bool,
    /// The magnitude.
    pub magnitude: Digits,
}

impl Signed {
    /// The value rounded half away from zero to `places` and parsed back into a decimal, which
    /// is shopspring's `Round(places)` applied after a division. `None` when it does not fit.
    #[must_use]
    pub fn round(&self, places: u32) -> Option<Decimal> {
        let text = self.magnitude.fixed(places);
        let value: Decimal = text.parse().ok()?;
        Some(if self.negative { -value } else { value })
    }
}

/// shopspring's `a.Div(b)`: the quotient rounded half away from zero to 16 decimal places,
/// computed exactly. `None` when `b` is zero.
#[must_use]
pub fn div_round(dividend: Decimal, divisor: Decimal) -> Option<Signed> {
    if divisor.is_zero() {
        return None;
    }
    let negative = dividend.is_sign_negative() != divisor.is_sign_negative();
    let Some(magnitude) = div_magnitude(
        dividend.mantissa().unsigned_abs(),
        dividend.scale(),
        divisor.mantissa().unsigned_abs(),
        divisor.scale(),
    ) else {
        return approximate(dividend, divisor);
    };
    Some(Signed {
        negative: negative && !magnitude.is_zero(),
        magnitude,
    })
}

/// `(numerator × 10^-numerator_scale) / (denominator × 10^-denominator_scale)`, rounded half away
/// from zero to 16 places by long division. `None` when the denominator is zero or, once
/// aligned, too long for a `u128` step.
#[must_use]
pub fn div_magnitude(
    numerator: u128,
    numerator_scale: u32,
    denominator: u128,
    denominator_scale: u32,
) -> Option<Digits> {
    if denominator == 0 {
        return None;
    }
    let places = i64::from(DIVISION_PRECISION);
    let exponent = i64::from(denominator_scale) - i64::from(numerator_scale) + places;
    let mut digits = numerator.to_string().into_bytes();
    let denominator = if exponent >= 0 {
        digits.extend(std::iter::repeat_n(
            b'0',
            usize::try_from(exponent).unwrap_or(0),
        ));
        denominator
    } else {
        u32::try_from(-exponent)
            .ok()
            .and_then(|power| 10u128.checked_pow(power))
            .and_then(|power| denominator.checked_mul(power))?
    };
    if denominator > u128::MAX / 10 {
        return None;
    }
    let mut quotient = Vec::with_capacity(digits.len());
    let mut remainder: u128 = 0;
    for digit in digits {
        remainder = remainder * 10 + u128::from(digit - b'0');
        let next = remainder / denominator;
        remainder %= denominator;
        quotient.push(b'0' + u8::try_from(next).unwrap_or(0));
    }
    if remainder >= denominator - remainder {
        increment(&mut quotient);
    }
    let digits = match quotient.iter().position(|digit| *digit != b'0') {
        Some(first) => quotient.split_off(first),
        None => vec![b'0'],
    };
    Some(Digits {
        digits,
        scale: places,
    })
}

/// The quotient through `rust_decimal`, for operands whose digits do not fit the exact long
/// division. Neither a money column nor a report value reaches it: it needs a divisor with more
/// than 37 significant digits once aligned.
fn approximate(dividend: Decimal, divisor: Decimal) -> Option<Signed> {
    let quotient = dividend
        .checked_div(divisor)?
        .round_dp_with_strategy(DIVISION_PRECISION, RoundingStrategy::MidpointAwayFromZero);
    Some(Signed {
        negative: quotient.is_sign_negative() && !quotient.is_zero(),
        magnitude: Digits::of(quotient),
    })
}

/// shopspring's `StringFixed(places)`: rounded half away from zero, written with exactly
/// `places` decimals, and signed only when the rounded value is not zero.
#[must_use]
pub fn string_fixed(value: Decimal, places: u32) -> String {
    let text = Digits::of(value).fixed(places);
    if value.is_sign_negative() && text.bytes().any(|digit| matches!(digit, b'1'..=b'9')) {
        format!("-{text}")
    } else {
        text
    }
}

/// shopspring's `RoundBank(places)`: half to even.
#[must_use]
pub fn round_bank(value: Decimal, places: u32) -> Decimal {
    value.round_dp_with_strategy(places, RoundingStrategy::MidpointNearestEven)
}

/// shopspring's `Round(places)`: half away from zero.
#[must_use]
pub fn round_half_away(value: Decimal, places: u32) -> Decimal {
    value.round_dp_with_strategy(places, RoundingStrategy::MidpointAwayFromZero)
}

/// Writes a plain decimal string with its whole part grouped by thousands with commas, the way
/// the report formatter groups and the way `toLocaleString` writes English numbers.
#[must_use]
pub fn group_thousands(digits: &str) -> String {
    let (sign, unsigned) = match digits.strip_prefix('-') {
        Some(rest) => ("-", rest),
        None => ("", digits),
    };
    let (whole, fraction) = match unsigned.split_once('.') {
        Some((whole, fraction)) => (whole, Some(fraction)),
        None => (unsigned, None),
    };
    if whole.len() <= 3 {
        return digits.to_owned();
    }
    let mut out = String::with_capacity(digits.len() + whole.len() / 3);
    out.push_str(sign);
    let lead = whole.len() % 3;
    if lead > 0 {
        out.push_str(&whole[..lead]);
    }
    let mut index = lead;
    while index < whole.len() {
        if index > 0 {
            out.push(',');
        }
        out.push_str(&whole[index..index + 3]);
        index += 3;
    }
    if let Some(fraction) = fraction {
        out.push('.');
        out.push_str(fraction);
    }
    out
}

#[cfg(test)]
mod tests {
    use super::Digits;
    use super::div_round;
    use super::group_thousands;
    use super::string_fixed;
    use rust_decimal::Decimal;

    fn dec(text: &str) -> Decimal {
        text.parse().unwrap_or_default()
    }

    #[test]
    fn string_fixed_rounds_half_away_from_zero_exactly() {
        assert_eq!(string_fixed(dec("1.005"), 2), "1.01");
        assert_eq!(string_fixed(dec("2.675"), 2), "2.68");
        assert_eq!(string_fixed(dec("-2.675"), 2), "-2.68");
        assert_eq!(string_fixed(dec("-0.001"), 2), "0.00");
        assert_eq!(string_fixed(dec("1234.5"), 0), "1235");
        assert_eq!(string_fixed(dec("0.5"), 0), "1");
        assert_eq!(string_fixed(dec("0.4999"), 0), "0");
        assert_eq!(string_fixed(dec("9.995"), 2), "10.00");
        assert_eq!(string_fixed(dec("42"), 3), "42.000");
        assert_eq!(string_fixed(dec("0"), 2), "0.00");
    }

    #[test]
    fn shifting_never_rounds() {
        assert_eq!(Digits::of(dec("0.1834")).shift(2).fixed(1), "18.3");
        assert_eq!(Digits::of(dec("1284000")).shift(-6).fixed(1), "1.3");
        assert_eq!(Digits::of(dec("5")).shift(3).fixed(0), "5000");
        assert_eq!(
            Digits::of(dec("0.000004")).shift(-6).fixed(10),
            "0.0000000000"
        );
        assert!(Digits::of(dec("1000")).at_least_pow10(3));
        assert!(!Digits::of(dec("999.999")).at_least_pow10(3));
    }

    #[test]
    fn division_matches_shopspring() {
        let quotient = div_round(dec("1500"), dec("2850")).map(|q| q.magnitude.fixed(16));
        assert_eq!(quotient.as_deref(), Some("0.5263157894736842"));
        let quotient = div_round(dec("150000"), dec("2850")).and_then(|q| q.round(6));
        assert_eq!(quotient, Some(dec("52.631579")));
        let quotient = div_round(dec("-1"), dec("3")).map(|q| (q.negative, q.magnitude.fixed(16)));
        assert_eq!(quotient, Some((true, String::from("0.3333333333333333"))));
        assert!(div_round(dec("1"), dec("0")).is_none());
        let tiny = div_round(dec("0.0000000000000000000001"), dec("7"))
            .map(|q| (q.negative, q.magnitude.fixed(16)));
        assert_eq!(tiny, Some((false, String::from("0.0000000000000000"))));
    }

    #[test]
    fn groups_thousands() {
        assert_eq!(group_thousands("1234567.891"), "1,234,567.891");
        assert_eq!(group_thousands("-1000"), "-1,000");
        assert_eq!(group_thousands("999"), "999");
    }
}
