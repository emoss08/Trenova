//! Amount splits that no longer add up, and their conversion to percents
//! (`shipment.FindStaleAmountSplits` and `ConvertAmountSplitsToPercent`).
//!
//! A charge split by amount goes stale when the charge changes after it was split. Percent
//! splits follow the charge and never do. A stale split converts to a percent split that keeps
//! each payer's proportion of what they had agreed to: every row but the last by sequence gets
//! `amount × 100 / sum` (shopspring division, rounded half away from zero to six places) and the
//! last gets what is left of 100, so the percents total exactly 100. A split whose rows sum to
//! zero or less cannot be converted.
//!
//! A split is stale exactly when `allocation.split` would refuse it, so the two never disagree
//! about whether a split adds up.

use crate::allocation::charge::overflow;
use crate::allocation::input::AllocationRow;
use crate::allocation::split::Method;
use crate::allocation::split::PERCENT_PLACES;
use crate::allocation::split::SHARE_PLACES;
use crate::allocation::split::checked_sum;
use crate::allocation::split::ordered;
use crate::report::fixed::div_round;
use crate::report::fixed::round_bank;
use rust_decimal::Decimal;
use trenova_module_kit::ops::RequestError;

const HUNDRED: Decimal = Decimal::ONE_HUNDRED;

/// One charge's state.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Rebalance {
    /// Not split by amount, or split by amounts that add up.
    Current,
    /// Split by amounts that do not add up to the charge.
    Stale {
        /// The rows' amounts summed, exact.
        allocated: Decimal,
        /// The charge.
        charge: Decimal,
        /// The percent each row converts to, in request order (`None` for a `null` row), or
        /// `None` when the rows sum to zero or less.
        percents: Option<Vec<Option<Decimal>>>,
    },
}

/// Judges one charge's rows and converts them when they are stale.
///
/// `rows` are the non-null rows with their request positions; `slots` is the length of the
/// request's `allocations`, `null` entries included.
///
/// # Errors
///
/// When a sum overflows.
pub fn rebalance(
    total: Decimal,
    rows: &[(usize, &AllocationRow)],
    slots: usize,
) -> Result<Rebalance, RequestError> {
    if rows.is_empty()
        || rows
            .iter()
            .any(|(_, row)| Method::parse(row.method.as_deref()) != Some(Method::Amount))
    {
        return Ok(Rebalance::Current);
    }
    let amounts: Vec<Decimal> = rows
        .iter()
        .map(|(_, row)| row.amount.unwrap_or_default())
        .collect();
    let allocated = checked_sum(&amounts)?;
    let cents: Vec<Decimal> = amounts
        .iter()
        .map(|amount| round_bank(*amount, SHARE_PLACES))
        .collect();
    if checked_sum(&cents)? == round_bank(total, SHARE_PLACES) {
        return Ok(Rebalance::Current);
    }
    let percents = if allocated > Decimal::ZERO {
        Some(convert(rows, allocated, slots)?)
    } else {
        None
    };
    Ok(Rebalance::Stale {
        allocated,
        charge: total,
        percents,
    })
}

fn convert(
    rows: &[(usize, &AllocationRow)],
    sum: Decimal,
    slots: usize,
) -> Result<Vec<Option<Decimal>>, RequestError> {
    let sorted = ordered(rows);
    let mut percents = vec![None; slots];
    let mut assigned = Decimal::ZERO;
    let last = sorted.len() - 1;
    for (position, (index, row)) in sorted.iter().enumerate() {
        let percent = if position < last {
            let scaled = row
                .amount
                .unwrap_or_default()
                .checked_mul(HUNDRED)
                .ok_or_else(overflow)?;
            let percent = div_round(scaled, sum)
                .and_then(|quotient| quotient.round(PERCENT_PLACES))
                .ok_or_else(overflow)?;
            assigned = assigned.checked_add(percent).ok_or_else(overflow)?;
            percent
        } else {
            HUNDRED.checked_sub(assigned).ok_or_else(overflow)?
        };
        if let Some(slot) = percents.get_mut(*index) {
            *slot = Some(percent);
        }
    }
    Ok(percents)
}
