//! What one charge comes to.
//!
//! An accessorial line is priced by its method, as `shipment.AdditionalCharge.Total` does:
//!
//! - `Flat`: the amount times the unit count, at least once (`amount × max(unit, 1)`);
//! - `PerUnit`: the amount times the unit count, and nothing when there are no units;
//! - `Percentage`: that percent of the base, the shipment's freight (`base × amount / 100`);
//! - anything else: nothing.
//!
//! Freight and order charges arrive with their total already known.

use crate::allocation::input::ChargeInput;
use rust_decimal::Decimal;
use trenova_module_kit::ops::RequestError;

const HUNDRED: Decimal = Decimal::ONE_HUNDRED;

/// The total of one accessorial line.
///
/// # Errors
///
/// When the product does not fit in 28 significant digits, which no stored charge reaches.
pub fn line_total(
    method: Option<&str>,
    amount: Decimal,
    unit: i64,
    base: Decimal,
) -> Result<Decimal, RequestError> {
    let total = match method {
        Some("Flat") => amount.checked_mul(Decimal::from(unit.max(1))),
        Some("PerUnit") if unit < 1 => Some(Decimal::ZERO),
        Some("PerUnit") => amount.checked_mul(Decimal::from(unit)),
        Some("Percentage") => amount
            .checked_div(HUNDRED)
            .and_then(|share| base.checked_mul(share)),
        _ => Some(Decimal::ZERO),
    };
    total.ok_or_else(overflow)
}

/// A charge's total: the one given, or its line priced against `base`.
///
/// # Errors
///
/// When the line's product overflows.
pub fn charge_total(charge: &ChargeInput, base: Decimal) -> Result<Decimal, RequestError> {
    match charge.total {
        Some(total) => Ok(total),
        None => line_total(
            charge.method.as_deref(),
            charge.amount.unwrap_or_default(),
            charge.unit.unwrap_or_default(),
            base,
        ),
    }
}

/// The request error for arithmetic past 28 significant digits.
#[must_use]
pub fn overflow() -> RequestError {
    RequestError::invalid_input("a charge amount overflows 28 significant digits")
}
