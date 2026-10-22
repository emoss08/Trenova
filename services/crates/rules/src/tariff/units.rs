//! Units and dimensions: the exact algebra behind Tariff's money, rates and quantities.
//!
//! A quantity is a decimal value with a unit (`1_200 lb`, `$2.35/mi`, `18%`). The checker reasons
//! about [`Dim`]s, the dimension a unit measures (`Money/Weight`), and the evaluator about
//! [`Unit`]s, the unit a value is written in. Values are kept in the unit they were written in
//! and converted only when two different units meet, into the left operand's unit, so every
//! result that is a terminating decimal is exact: `$18.50/cwt × 1_200 lb` converts the pounds
//! into hundredweight (×0.01, exact) rather than the rate into a per-kilogram figure that never
//! terminates.
//!
//! Conversion factors are to SI and exact by definition: 1 lb is 0.45359237 kg, 1 mi is
//! 1,609.344 m, 1 ft is 0.3048 m, 1 gal is 0.003785411784 m³. The one inexact conversion is
//! between °F and °C (a factor of 5/9), which rounds at 28 significant digits; temperatures are
//! for comparison and difference, not arithmetic.
//!
//! Dimensionless units (`%`, and the count labels `pallet`, `piece`, `stop`) fold into the value
//! when multiplied: `18% × $100` is `$18`, and `$75/stop × 3` is `$225`. `Money/stop` therefore
//! measures `Money`, and its `/stop` is a label for people reading the source.

use rust_decimal::Decimal;
use std::fmt;
use std::str::FromStr;

/// A base dimension. Distance (road miles) and length (linear feet, dimensions) are kept apart
/// on purpose: a rate per mile must never price a linear foot.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum Base {
    /// Money in the rating currency.
    Money,
    /// Mass.
    Weight,
    /// Road distance.
    Distance,
    /// Linear length.
    Length,
    /// Volume, cubic or liquid.
    Volume,
    /// Elapsed time.
    Duration,
    /// Temperature (comparison and difference only).
    Temperature,
}

const BASES: [Base; 7] = [
    Base::Money,
    Base::Weight,
    Base::Distance,
    Base::Length,
    Base::Volume,
    Base::Duration,
    Base::Temperature,
];

impl Base {
    /// The dimension's name as types spell it (`Weight`).
    #[must_use]
    pub const fn name(self) -> &'static str {
        match self {
            Self::Money => "Money",
            Self::Weight => "Weight",
            Self::Distance => "Distance",
            Self::Length => "Length",
            Self::Volume => "Volume",
            Self::Duration => "Duration",
            Self::Temperature => "Temperature",
        }
    }

    const fn index(self) -> usize {
        self as usize
    }

    /// The base dimension a type name names.
    #[must_use]
    pub fn from_name(name: &str) -> Option<Self> {
        BASES.into_iter().find(|base| base.name() == name)
    }
}

/// A dimension: the exponent of each base dimension. `Money/Weight` is money to the first and
/// weight to the minus first; a bare number is all zeros.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Default)]
pub struct Dim([i8; 7]);

impl Dim {
    /// A plain number.
    pub const NONE: Self = Self([0; 7]);

    /// One base dimension to the first power.
    #[must_use]
    pub const fn of(base: Base) -> Self {
        let mut exponents = [0; 7];
        exponents[base.index()] = 1;
        Self(exponents)
    }

    /// Money.
    #[must_use]
    pub const fn money() -> Self {
        Self::of(Base::Money)
    }

    /// The exponent of a base dimension.
    #[must_use]
    pub const fn exponent(self, base: Base) -> i8 {
        self.0[base.index()]
    }

    /// Whether this is a plain number.
    #[must_use]
    pub fn is_none(self) -> bool {
        self == Self::NONE
    }

    /// The dimension of a product, or `None` when an exponent would overflow.
    #[must_use]
    pub fn times(self, other: Self) -> Option<Self> {
        let mut out = [0i8; 7];
        for (index, slot) in out.iter_mut().enumerate() {
            *slot = self.0[index].checked_add(other.0[index])?;
        }
        Some(Self(out))
    }

    /// The dimension of a quotient, or `None` when an exponent would overflow.
    #[must_use]
    pub fn per(self, other: Self) -> Option<Self> {
        let mut out = [0i8; 7];
        for (index, slot) in out.iter_mut().enumerate() {
            *slot = self.0[index].checked_sub(other.0[index])?;
        }
        Some(Self(out))
    }

    /// Whether temperature appears in the dimension.
    #[must_use]
    pub const fn has_temperature(self) -> bool {
        self.0[Base::Temperature.index()] != 0
    }
}

impl fmt::Display for Dim {
    /// `Money/Weight`, `Weight/Volume`, `Number` for a plain number.
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        let mut numerator = Vec::new();
        let mut denominator = Vec::new();
        for base in BASES {
            let exponent = self.exponent(base);
            let part = |power: i8| {
                if power == 1 {
                    base.name().to_owned()
                } else {
                    format!("{}^{power}", base.name())
                }
            };
            match exponent.cmp(&0) {
                std::cmp::Ordering::Greater => numerator.push(part(exponent)),
                std::cmp::Ordering::Less => denominator.push(part(-exponent)),
                std::cmp::Ordering::Equal => {}
            }
        }
        if numerator.is_empty() && denominator.is_empty() {
            return formatter.write_str("Number");
        }
        let top = if numerator.is_empty() {
            String::from("1")
        } else {
            numerator.join("·")
        };
        if denominator.is_empty() {
            formatter.write_str(&top)
        } else {
            write!(formatter, "{top}/{}", denominator.join("·"))
        }
    }
}

/// A unit that is not built from others: `lb`, `mi`, `$`, `%`.
#[derive(Debug, PartialEq, Eq)]
pub struct Simple {
    /// How the unit is written.
    pub symbol: &'static str,
    /// What it measures; `None` for a dimensionless label (`%`, `pallet`).
    pub base: Option<Base>,
    factor: &'static str,
}

impl Simple {
    /// The exact factor to the base dimension's SI unit (or to 1 for a dimensionless label).
    #[must_use]
    pub fn factor(&self) -> Decimal {
        Decimal::from_str(self.factor).unwrap_or(Decimal::ONE)
    }
}

/// Every simple unit the language has.
pub static SIMPLE_UNITS: &[Simple] = &[
    Simple {
        symbol: "$",
        base: Some(Base::Money),
        factor: "1",
    },
    Simple {
        symbol: "lb",
        base: Some(Base::Weight),
        factor: "0.45359237",
    },
    Simple {
        symbol: "cwt",
        base: Some(Base::Weight),
        factor: "45.359237",
    },
    Simple {
        symbol: "kg",
        base: Some(Base::Weight),
        factor: "1",
    },
    Simple {
        symbol: "t",
        base: Some(Base::Weight),
        factor: "1000",
    },
    Simple {
        symbol: "ton",
        base: Some(Base::Weight),
        factor: "907.18474",
    },
    Simple {
        symbol: "mi",
        base: Some(Base::Distance),
        factor: "1609.344",
    },
    Simple {
        symbol: "km",
        base: Some(Base::Distance),
        factor: "1000",
    },
    Simple {
        symbol: "in",
        base: Some(Base::Length),
        factor: "0.0254",
    },
    Simple {
        symbol: "ft",
        base: Some(Base::Length),
        factor: "0.3048",
    },
    Simple {
        symbol: "m",
        base: Some(Base::Length),
        factor: "1",
    },
    Simple {
        symbol: "ft3",
        base: Some(Base::Volume),
        factor: "0.028316846592",
    },
    Simple {
        symbol: "m3",
        base: Some(Base::Volume),
        factor: "1",
    },
    Simple {
        symbol: "gal",
        base: Some(Base::Volume),
        factor: "0.003785411784",
    },
    Simple {
        symbol: "l",
        base: Some(Base::Volume),
        factor: "0.001",
    },
    Simple {
        symbol: "min",
        base: Some(Base::Duration),
        factor: "60",
    },
    Simple {
        symbol: "h",
        base: Some(Base::Duration),
        factor: "3600",
    },
    Simple {
        symbol: "d",
        base: Some(Base::Duration),
        factor: "86400",
    },
    Simple {
        symbol: "°F",
        base: Some(Base::Temperature),
        factor: "1",
    },
    Simple {
        symbol: "°C",
        base: Some(Base::Temperature),
        factor: "1",
    },
    Simple {
        symbol: "%",
        base: None,
        factor: "0.01",
    },
    Simple {
        symbol: "pallet",
        base: None,
        factor: "1",
    },
    Simple {
        symbol: "piece",
        base: None,
        factor: "1",
    },
    Simple {
        symbol: "stop",
        base: None,
        factor: "1",
    },
];

/// The simple unit written `symbol`.
#[must_use]
pub fn simple(symbol: &str) -> Option<&'static Simple> {
    SIMPLE_UNITS.iter().find(|unit| unit.symbol == symbol)
}

/// Why two quantities could not be combined or converted.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum UnitError {
    /// The dimensions differ (`lb` to `mi`).
    #[error("{from} cannot be converted to {to}")]
    Mismatch {
        /// The unit converted from.
        from: String,
        /// The unit converted to.
        to: String,
    },
    /// A product or quotient of temperatures, which has no meaning.
    #[error("temperatures can only be compared and subtracted")]
    Temperature,
    /// A result past 28 significant digits.
    #[error("the result is too large")]
    Overflow,
    /// A unit text that names no unit.
    #[error("{0:?} is not a unit")]
    Unknown(String),
}

/// A unit, simple or built from simple units: `lb`, `$/mi`, `lb/ft3`, `mi/gal`. The factors keep
/// the order they were written in, so `$/cwt` prints as written.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Unit {
    factors: Vec<(&'static Simple, i8)>,
}

impl Unit {
    /// No unit: a plain number.
    #[must_use]
    pub const fn none() -> Self {
        Self {
            factors: Vec::new(),
        }
    }

    /// One simple unit.
    #[must_use]
    pub fn simple(unit: &'static Simple) -> Self {
        Self {
            factors: vec![(unit, 1)],
        }
    }

    /// Parses a unit as written after a number or in a type: `lb`, `$/mi`, `lb/ft3`, `%`.
    ///
    /// # Errors
    ///
    /// When a part names no unit.
    pub fn parse(text: &str) -> Result<Self, UnitError> {
        let mut factors = Vec::new();
        for (position, part) in text.split('/').enumerate() {
            let part = part.trim();
            let unit = simple(part).ok_or_else(|| UnitError::Unknown(text.to_owned()))?;
            factors.push((unit, if position == 0 { 1 } else { -1 }));
        }
        Ok(Self { factors })
    }

    /// The simple units and their exponents.
    #[must_use]
    pub fn factors(&self) -> &[(&'static Simple, i8)] {
        &self.factors
    }

    /// Whether this is no unit at all.
    #[must_use]
    pub fn is_none(&self) -> bool {
        self.factors.is_empty()
    }

    /// What the unit measures. Dimensionless labels contribute nothing.
    #[must_use]
    pub fn dim(&self) -> Dim {
        let mut dim = Dim::NONE;
        for (unit, exponent) in &self.factors {
            if let Some(base) = unit.base {
                dim.0[base.index()] = dim.0[base.index()].saturating_add(*exponent);
            }
        }
        dim
    }

    /// The unit `.in(target)` converts a value of dimension `dim` to: `target` itself when it
    /// measures `dim` (`.in(cwt)` on a weight, `.in($/cwt)` on a rate per weight), or, on a rate
    /// (money per a quantity), money per `target` when `target` measures the rate's
    /// denominator, so `$0.185/lb .in(cwt)` is in `$/cwt`. `None` when it converts to neither,
    /// and for a bare count, which has no unit to convert to.
    #[must_use]
    pub fn conversion_target(dim: Dim, target: &Self) -> Option<Self> {
        let measures = target.dim();
        if measures.is_none() {
            return None;
        }
        if measures == dim {
            return Some(target.clone());
        }
        let dollar = simple("$")?;
        if dim == Dim::money() || Dim::money().per(measures) != Some(dim) {
            return None;
        }
        let mut factors = vec![(dollar, 1)];
        factors.extend(
            target
                .factors
                .iter()
                .map(|(unit, exponent)| (*unit, exponent.saturating_neg())),
        );
        Some(Self { factors })
    }

    /// The exact factor that converts a value in this unit to SI (dimensionless labels folded
    /// in), as a numerator and a denominator so no division happens until the end.
    fn ratio(&self) -> Result<(Decimal, Decimal), UnitError> {
        let mut numerator = Decimal::ONE;
        let mut denominator = Decimal::ONE;
        for (unit, exponent) in &self.factors {
            let factor = unit.factor();
            for _ in 0..exponent.unsigned_abs() {
                if *exponent > 0 {
                    numerator = numerator.checked_mul(factor).ok_or(UnitError::Overflow)?;
                } else {
                    denominator = denominator.checked_mul(factor).ok_or(UnitError::Overflow)?;
                }
            }
        }
        Ok((numerator, denominator))
    }

    fn temperature(&self) -> Option<&'static Simple> {
        match self.factors.as_slice() {
            [(unit, 1)] if unit.base == Some(Base::Temperature) => Some(unit),
            _ => None,
        }
    }

    /// The same unit with dimensionless labels removed, for display after a product.
    fn without_labels(&self) -> Self {
        Self {
            factors: self
                .factors
                .iter()
                .copied()
                .filter(|(unit, _)| unit.base.is_some())
                .collect(),
        }
    }
}

impl fmt::Display for Unit {
    /// As written: `$/mi`, `lb/ft3`, `lb·ft` for a product.
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        let mut numerator = Vec::new();
        let mut denominator = Vec::new();
        for (unit, exponent) in &self.factors {
            let power = exponent.unsigned_abs();
            let text = if power == 1 {
                unit.symbol.to_owned()
            } else {
                format!("{}^{power}", unit.symbol)
            };
            if *exponent > 0 {
                numerator.push(text);
            } else {
                denominator.push(text);
            }
        }
        let top = numerator.join("·");
        if denominator.is_empty() {
            formatter.write_str(&top)
        } else {
            write!(
                formatter,
                "{}/{}",
                if top.is_empty() { "1" } else { &top },
                denominator.join("/")
            )
        }
    }
}

fn mismatch(from: &Unit, to: &Unit) -> UnitError {
    UnitError::Mismatch {
        from: from.to_string(),
        to: to.to_string(),
    }
}

/// Converts `value` from one unit to another of the same dimension, exactly whenever the result
/// is a terminating decimal.
///
/// # Errors
///
/// When the dimensions differ, or the result overflows.
pub fn convert(value: Decimal, from: &Unit, to: &Unit) -> Result<Decimal, UnitError> {
    if from == to {
        return Ok(value);
    }
    if from.dim() != to.dim() {
        return Err(mismatch(from, to));
    }
    if from.dim().has_temperature() {
        return convert_temperature(value, from, to);
    }
    let (from_numerator, from_denominator) = from.ratio()?;
    let (to_numerator, to_denominator) = to.ratio()?;
    let numerator = from_numerator
        .checked_mul(to_denominator)
        .ok_or(UnitError::Overflow)?;
    let denominator = from_denominator
        .checked_mul(to_numerator)
        .ok_or(UnitError::Overflow)?;
    value
        .checked_mul(numerator)
        .and_then(|scaled| scaled.checked_div(denominator))
        .map(|value| value.normalize())
        .ok_or(UnitError::Overflow)
}

fn convert_temperature(value: Decimal, from: &Unit, to: &Unit) -> Result<Decimal, UnitError> {
    let (Some(source), Some(target)) = (from.temperature(), to.temperature()) else {
        return Err(UnitError::Temperature);
    };
    let thirty_two = Decimal::from(32);
    let five = Decimal::from(5);
    let nine = Decimal::from(9);
    let converted = match (source.symbol, target.symbol) {
        ("°F", "°C") => value
            .checked_sub(thirty_two)
            .and_then(|delta| delta.checked_mul(five))
            .and_then(|scaled| scaled.checked_div(nine)),
        ("°C", "°F") => value
            .checked_mul(nine)
            .and_then(|scaled| scaled.checked_div(five))
            .and_then(|scaled| scaled.checked_add(thirty_two)),
        _ => Some(value),
    };
    converted
        .map(|value| value.normalize())
        .ok_or(UnitError::Overflow)
}

/// A value with its unit, the way the evaluator holds money, rates and quantities.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Quantity {
    /// The amount, in `unit`.
    pub value: Decimal,
    /// The unit it is written in; empty for a plain number.
    pub unit: Unit,
}

impl Quantity {
    /// A quantity.
    #[must_use]
    pub const fn new(value: Decimal, unit: Unit) -> Self {
        Self { value, unit }
    }

    /// The other quantity's value in this one's unit.
    fn aligned(&self, other: &Self) -> Result<Decimal, UnitError> {
        convert(other.value, &other.unit, &self.unit)
    }

    /// `self + other`, in `self`'s unit.
    ///
    /// # Errors
    ///
    /// When the dimensions differ or the sum overflows.
    pub fn add(&self, other: &Self) -> Result<Self, UnitError> {
        let right = self.aligned(other)?;
        let value = self.value.checked_add(right).ok_or(UnitError::Overflow)?;
        Ok(Self::new(value, self.unit.clone()))
    }

    /// `self - other`, in `self`'s unit. A temperature minus a temperature is a difference in
    /// the same unit.
    ///
    /// # Errors
    ///
    /// When the dimensions differ or the difference overflows.
    pub fn sub(&self, other: &Self) -> Result<Self, UnitError> {
        let right = self.aligned(other)?;
        let value = self.value.checked_sub(right).ok_or(UnitError::Overflow)?;
        Ok(Self::new(value, self.unit.clone()))
    }

    /// Orders two quantities of one dimension.
    ///
    /// # Errors
    ///
    /// When the dimensions differ.
    pub fn compare(&self, other: &Self) -> Result<std::cmp::Ordering, UnitError> {
        Ok(self.value.cmp(&self.aligned(other)?))
    }

    /// `self × other`. A unit of `other` whose base also appears with the opposite sign in
    /// `self` is first converted into `self`'s unit for that base and cancelled, so
    /// `$18.50/cwt × 1_200 lb` is `$222` and `$2.35/mi × 812 mi` is `$1,908.20`. Dimensionless
    /// labels fold into the value.
    ///
    /// # Errors
    ///
    /// When temperatures are multiplied, or the product overflows.
    pub fn mul(&self, other: &Self) -> Result<Self, UnitError> {
        combine(self, other, 1)
    }

    /// `self / other`, with the same cancellation as [`Quantity::mul`]: `$450 / 300 mi` is
    /// `$1.5/mi`, and `1_200 lb / 1 cwt` is `12`.
    ///
    /// # Errors
    ///
    /// When temperatures are divided, `other` is zero, or the quotient overflows.
    pub fn div(&self, other: &Self) -> Result<Self, UnitError> {
        combine(self, other, -1)
    }

    /// This quantity in another unit of the same dimension.
    ///
    /// # Errors
    ///
    /// When the dimensions differ or the result overflows.
    pub fn to(&self, unit: &Unit) -> Result<Self, UnitError> {
        Ok(Self::new(
            convert(self.value, &self.unit, unit)?,
            unit.clone(),
        ))
    }
}

/// `left × right` (`sign` 1) or `left / right` (`sign` -1).
fn combine(left: &Quantity, right: &Quantity, sign: i8) -> Result<Quantity, UnitError> {
    if left.unit.dim().has_temperature() || right.unit.dim().has_temperature() {
        return Err(UnitError::Temperature);
    }
    let mut factors: Vec<(&'static Simple, i8)> = left.unit.factors.clone();
    let mut value = right.value;
    for (unit, own) in &right.unit.factors {
        let exponent = own.saturating_mul(sign);
        let Some(base) = unit.base else {
            let factor = unit.factor();
            value = if *own > 0 {
                value.checked_mul(factor)
            } else {
                value.checked_div(factor)
            }
            .ok_or(UnitError::Overflow)?;
            continue;
        };
        if let Some(slot) = factors
            .iter_mut()
            .find(|(existing, _)| existing.base == Some(base) && existing.symbol != unit.symbol)
        {
            let into = Unit::simple(slot.0);
            let from = Unit::simple(unit);
            for _ in 0..own.unsigned_abs() {
                value = if *own > 0 {
                    convert(value, &from, &into)?
                } else {
                    convert(value, &into, &from)?
                };
            }
            slot.1 = slot.1.saturating_add(exponent);
        } else if let Some(slot) = factors
            .iter_mut()
            .find(|(existing, _)| existing.symbol == unit.symbol)
        {
            slot.1 = slot.1.saturating_add(exponent);
        } else {
            factors.push((unit, exponent));
        }
    }
    let mut left_value = left.value;
    for (unit, exponent) in &left.unit.factors {
        if unit.base.is_none() {
            let factor = unit.factor();
            for _ in 0..exponent.unsigned_abs() {
                left_value = if *exponent > 0 {
                    left_value.checked_mul(factor)
                } else {
                    left_value.checked_div(factor)
                }
                .ok_or(UnitError::Overflow)?;
            }
        }
    }
    factors.retain(|(unit, exponent)| *exponent != 0 && unit.base.is_some());
    let product = if sign > 0 {
        left_value.checked_mul(value)
    } else {
        if value.is_zero() {
            return Err(UnitError::Overflow);
        }
        left_value.checked_div(value)
    }
    .ok_or(UnitError::Overflow)?;
    Ok(Quantity::new(
        product.normalize(),
        Unit { factors }.without_labels(),
    ))
}

#[cfg(test)]
mod tests {
    use super::{Base, Dim, Quantity, Unit, UnitError, convert};
    use rust_decimal::Decimal;
    use std::str::FromStr;

    fn d(text: &str) -> Decimal {
        Decimal::from_str(text).expect("a decimal")
    }

    fn q(value: &str, unit: &str) -> Quantity {
        Quantity::new(
            d(value),
            if unit.is_empty() {
                Unit::none()
            } else {
                Unit::parse(unit).expect("a unit")
            },
        )
    }

    #[test]
    fn a_rate_per_hundredweight_prices_pounds_exactly() {
        let charge = q("18.50", "$/cwt")
            .mul(&q("1200", "lb"))
            .expect("a product");
        assert_eq!(charge.value, d("222"));
        assert_eq!(charge.unit.to_string(), "$");
    }

    #[test]
    fn in_converts_a_rate_by_its_denominator_or_its_whole_unit() {
        let unit = |text: &str| Unit::parse(text).expect("a unit");
        let per_weight = unit("$/lb").dim();
        let target = |dim, to: &str| Unit::conversion_target(dim, &unit(to)).map(|u| u.to_string());
        assert_eq!(target(per_weight, "cwt").as_deref(), Some("$/cwt"));
        assert_eq!(target(per_weight, "$/cwt").as_deref(), Some("$/cwt"));
        assert_eq!(target(Dim::of(Base::Weight), "kg").as_deref(), Some("kg"));
        assert_eq!(target(per_weight, "mi"), None);
        assert_eq!(target(Dim::of(Base::Weight), "mi"), None);
        assert_eq!(target(Dim::money(), "lb"), None);
        assert_eq!(target(Dim::NONE, "pallet"), None);
    }

    #[test]
    fn a_rate_per_mile_prices_miles() {
        let charge = q("2.35", "$/mi").mul(&q("812", "mi")).expect("a product");
        assert_eq!(charge.value, d("1908.2"));
        assert_eq!(charge.unit.dim(), Dim::money());
    }

    #[test]
    fn a_rate_per_mile_prices_kilometres_by_converting_them() {
        let charge = q("1", "$/mi").mul(&q("1.609344", "km")).expect("a product");
        assert_eq!(charge.value, d("1"));
    }

    #[test]
    fn money_divided_by_distance_is_a_rate() {
        let rate = q("450", "$").div(&q("300", "mi")).expect("a quotient");
        assert_eq!(rate.value, d("1.5"));
        assert_eq!(rate.unit.to_string(), "$/mi");
        assert_eq!(rate.unit.dim().to_string(), "Money/Distance");
    }

    #[test]
    fn a_weight_over_a_weight_is_a_number() {
        let ratio = q("1200", "lb").div(&q("1", "cwt")).expect("a quotient");
        assert_eq!(ratio.value, d("12"));
        assert!(ratio.unit.is_none());
    }

    #[test]
    fn percents_and_count_labels_fold_into_the_value() {
        let fuel = q("100", "$").mul(&q("18", "%")).expect("a product");
        assert_eq!(
            (fuel.value, fuel.unit.to_string()),
            (d("18"), String::from("$"))
        );
        let stops = q("75", "$/stop").mul(&q("3", "")).expect("a product");
        assert_eq!(
            (stops.value, stops.unit.to_string()),
            (d("225"), String::from("$"))
        );
    }

    #[test]
    fn sums_take_the_left_operands_unit() {
        let total = q("500", "lb").add(&q("2", "cwt")).expect("a sum");
        assert_eq!(
            (total.value, total.unit.to_string()),
            (d("700"), String::from("lb"))
        );
    }

    #[test]
    fn different_dimensions_do_not_add() {
        assert_eq!(
            q("1", "$").add(&q("1", "lb")),
            Err(UnitError::Mismatch {
                from: String::from("lb"),
                to: String::from("$")
            })
        );
    }

    #[test]
    fn distance_and_length_are_different_dimensions() {
        assert!(q("1", "mi").add(&q("1", "ft")).is_err());
    }

    #[test]
    fn conversions_are_exact_when_the_result_terminates() {
        let pounds = Unit::parse("lb").expect("a unit");
        let kilograms = Unit::parse("kg").expect("a unit");
        assert_eq!(convert(d("1"), &pounds, &kilograms), Ok(d("0.45359237")));
        assert_eq!(convert(d("0.45359237"), &kilograms, &pounds), Ok(d("1")));
        let gallons = Unit::parse("gal").expect("a unit");
        let litres = Unit::parse("l").expect("a unit");
        assert_eq!(convert(d("1"), &gallons, &litres), Ok(d("3.785411784")));
    }

    #[test]
    fn densities_and_quotients_parse_and_compare() {
        let density = q("1000", "lb").div(&q("125", "ft3")).expect("a quotient");
        assert_eq!(
            (density.value, density.unit.to_string()),
            (d("8"), String::from("lb/ft3"))
        );
        let scale = Unit::parse("lb/ft3").expect("a unit");
        assert_eq!(density.unit, scale);
        assert_eq!(scale.dim().exponent(Base::Weight), 1);
        assert_eq!(scale.dim().exponent(Base::Volume), -1);
    }

    #[test]
    fn temperatures_compare_and_convert_but_do_not_multiply() {
        let freezing = q("32", "°F");
        assert_eq!(
            freezing
                .to(&Unit::parse("°C").expect("a unit"))
                .map(|t| t.value),
            Ok(d("0"))
        );
        assert_eq!(freezing.mul(&q("2", "")), Err(UnitError::Temperature));
        assert_eq!(
            q("40", "°F")
                .compare(&q("4", "°C"))
                .map(std::cmp::Ordering::is_lt),
            Ok(false)
        );
    }

    #[test]
    fn unknown_units_are_refused() {
        assert_eq!(
            Unit::parse("furlong"),
            Err(UnitError::Unknown(String::from("furlong")))
        );
    }

    #[test]
    fn dividing_by_zero_is_an_error_not_a_panic() {
        assert!(q("1", "$").div(&q("0", "mi")).is_err());
    }
}
