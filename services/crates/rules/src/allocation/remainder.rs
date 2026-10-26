//! Where a split stands while somebody is still typing it.
//!
//! The split editor shows how much of 100% (percent rows) or of the charge (amount rows) is
//! assigned and what is left, and refuses to save a split the server would refuse. Both answers
//! come from the same rules as `allocation.split`: percents are complete when they total 100 at
//! six places, amounts when their cents add up to the charge's cents.

use crate::allocation::charge::overflow;
use crate::allocation::input::AllocationRow;
use crate::allocation::split::Method;
use crate::allocation::split::PERCENT_PLACES;
use crate::allocation::split::SHARE_PLACES;
use crate::allocation::split::checked_sum;
use crate::report::fixed::round_bank;
use crate::report::fixed::round_half_away;
use rust_decimal::Decimal;
use std::collections::HashSet;
use trenova_module_kit::message;
use trenova_module_kit::message::Message;
use trenova_module_kit::ops::RequestError;

const HUNDRED: Decimal = Decimal::ONE_HUNDRED;

/// What an editor shows for one charge's rows.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Remainder {
    /// The method of the first row, `Percent` when there are no rows.
    pub method: Method,
    /// Percent points or currency already assigned, exact.
    pub allocated: Decimal,
    /// What is left to reach 100% or the charge, exact; negative when over. `None` for amount
    /// rows when the charge total is unknown.
    pub remaining: Option<Decimal>,
    /// Whether the rows add up exactly (by the server's rule).
    pub complete: bool,
    /// Whether the rows assign more than there is.
    pub over: bool,
    /// Whether rows use different methods.
    pub mixed_methods: bool,
    /// Whether a payer appears on more than one row.
    pub duplicate_payers: bool,
    /// Whether a row names no payer.
    pub missing_payers: bool,
    /// The first reason the rows cannot be saved, or `None` when they can.
    pub problem: Option<Message>,
}

/// Measures one charge's rows against `total` (`None` when it is not known in this form).
///
/// # Errors
///
/// When a sum overflows.
pub fn measure(rows: &[&AllocationRow], total: Option<Decimal>) -> Result<Remainder, RequestError> {
    let method = match rows
        .first()
        .and_then(|row| Method::parse(row.method.as_deref()))
    {
        Some(Method::Amount) => Method::Amount,
        _ => Method::Percent,
    };
    let mixed_methods = rows
        .iter()
        .any(|row| Method::parse(row.method.as_deref()) != Some(method));
    let mut seen = HashSet::new();
    let duplicate_payers = rows
        .iter()
        .filter_map(|row| row.payer())
        .any(|payer| !seen.insert(payer));
    let missing_payers = rows.iter().any(|row| row.payer().is_none());

    let (allocated, remaining, complete, over, out_of_range) = match method {
        Method::Percent => {
            let percents: Vec<Decimal> = rows
                .iter()
                .map(|row| row.percent.unwrap_or_default())
                .collect();
            let allocated = checked_sum(&percents)?;
            let compared = round_half_away(allocated, PERCENT_PLACES);
            let remaining = HUNDRED.checked_sub(allocated).ok_or_else(overflow)?;
            let out_of_range = percents
                .iter()
                .any(|percent| *percent <= Decimal::ZERO || *percent > HUNDRED);
            (
                allocated,
                Some(remaining),
                !rows.is_empty() && compared == HUNDRED,
                compared > HUNDRED,
                out_of_range,
            )
        }
        Method::Amount => {
            let amounts: Vec<Decimal> = rows
                .iter()
                .map(|row| row.amount.unwrap_or_default())
                .collect();
            let allocated = checked_sum(&amounts)?;
            let cents: Vec<Decimal> = amounts
                .iter()
                .map(|amount| round_bank(*amount, SHARE_PLACES))
                .collect();
            let assigned = checked_sum(&cents)?;
            let target = round_bank(total.unwrap_or_default(), SHARE_PLACES);
            let remaining = match total {
                Some(total) => Some(total.checked_sub(allocated).ok_or_else(overflow)?),
                None => None,
            };
            let out_of_range = amounts.iter().any(|amount| *amount <= Decimal::ZERO);
            (
                allocated,
                remaining,
                total.is_some() && !rows.is_empty() && assigned == target,
                total.is_some() && assigned > target,
                out_of_range,
            )
        }
    };

    let problem = if rows.is_empty() {
        None
    } else if mixed_methods {
        Some(message!(
            "Split every row by percent or every row by amount"
        ))
    } else if duplicate_payers {
        Some(message!("Each payer may appear only once"))
    } else if missing_payers {
        Some(message!("Every row needs a payer"))
    } else if out_of_range {
        Some(match method {
            Method::Percent => message!("Percent must be greater than 0 and at most 100"),
            Method::Amount => message!("Amount must be greater than zero"),
        })
    } else if complete || (method == Method::Amount && total.is_none()) {
        None
    } else {
        Some(match method {
            Method::Percent => message!("Percents must add up to 100"),
            Method::Amount => message!("Amounts must add up to the charge"),
        })
    };

    Ok(Remainder {
        method,
        allocated,
        remaining,
        complete,
        over,
        mixed_methods,
        duplicate_payers,
        missing_payers,
        problem: problem.map(Message::new),
    })
}
