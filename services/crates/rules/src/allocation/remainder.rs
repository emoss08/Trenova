//! Where a split stands while somebody is still typing it.
//!
//! The split editor shows how much of 100% (percent rows) or of the charge (amount rows) is
//! assigned and what is left, and refuses to save a split the server would refuse. Both answers
//! come from the same rules as `allocation.split`: percents are complete when they total 100 at
//! six places, amounts when their cents add up to the charge's cents.
//!
//! Each row also shows the other half of what it bills: a percent row its cents, an amount row
//! its percent. Once the rows are a valid split those are the split's own shares, so the last
//! row carries the rounding; before then each row is priced as the split prices every row but
//! the last.
//!
//! Two answers let the editor finish a split without arithmetic of its own: the value each row
//! would need to take whatever is left, and an even split of the charge across the rows, to the
//! hundredth of a percent or to the cent with the last row taking the rest.

use crate::allocation::charge::overflow;
use crate::allocation::input::AllocationRow;
use crate::allocation::split::Method;
use crate::allocation::split::PERCENT_PLACES;
use crate::allocation::split::SHARE_PLACES;
use crate::allocation::split::allocate;
use crate::allocation::split::amount_percent;
use crate::allocation::split::checked_sum;
use crate::report::fixed::round_bank;
use crate::report::fixed::round_half_away;
use rust_decimal::Decimal;
use rust_decimal::RoundingStrategy;
use std::collections::HashSet;
use trenova_module_kit::message;
use trenova_module_kit::message::Message;
use trenova_module_kit::ops::RequestError;

const HUNDRED: Decimal = Decimal::ONE_HUNDRED;

/// An even percent split is cut to hundredths, the precision a person types a percent at.
const EVEN_PERCENT_PLACES: u32 = 2;

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

/// What one row bills while its split is being typed.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct RowShare {
    /// The row's cents; `None` for a percent row when the charge total is not known.
    pub amount: Option<Decimal>,
    /// The row's percent of the charge; `None` for an amount row of a zero or unknown charge.
    pub percent: Option<Decimal>,
}

/// What each row bills, by request position (`slots` long, `None` for a `null` row or a row
/// with no positive value yet): the exact `allocation.split` shares once `state` is a complete,
/// savable split, and otherwise each row on its own.
///
/// # Errors
///
/// When a share overflows.
pub fn row_shares(
    rows: &[(usize, &AllocationRow)],
    slots: usize,
    total: Option<Decimal>,
    state: &Remainder,
) -> Result<Vec<Option<RowShare>>, RequestError> {
    let mut shares = vec![None; slots];
    if let Some(total) = total
        && state.complete
        && state.problem.is_none()
        && let Ok(split) = allocate(total, rows, "")?
    {
        for share in split {
            if let Some(index) = share.allocation {
                shares[index] = Some(RowShare {
                    amount: Some(share.amount),
                    percent: share.percent,
                });
            }
        }
        return Ok(shares);
    }
    for (index, row) in rows {
        let method = Method::parse(row.method.as_deref()).unwrap_or(state.method);
        shares[*index] = match method {
            Method::Percent => row
                .percent
                .filter(|percent| *percent > Decimal::ZERO)
                .map(|percent| percent_row_share(percent, total))
                .transpose()?,
            Method::Amount => row
                .amount
                .filter(|amount| *amount > Decimal::ZERO)
                .map(|amount| amount_row_share(amount, total))
                .transpose()?,
        };
    }
    Ok(shares)
}

/// A percent row on its own: its cents as the split prices every row but the last.
fn percent_row_share(percent: Decimal, total: Option<Decimal>) -> Result<RowShare, RequestError> {
    let amount = total
        .map(|total| {
            total
                .checked_mul(percent)
                .and_then(|product| product.checked_div(HUNDRED))
                .map(|share| round_bank(share, SHARE_PLACES))
                .ok_or_else(overflow)
        })
        .transpose()?;
    Ok(RowShare {
        amount,
        percent: Some(percent),
    })
}

/// An amount row on its own: its cents, and their percent of the charge.
fn amount_row_share(amount: Decimal, total: Option<Decimal>) -> Result<RowShare, RequestError> {
    let cents = round_bank(amount, SHARE_PLACES);
    let percent = match total {
        Some(total) => amount_percent(cents, total)?,
        None => None,
    };
    Ok(RowShare {
        amount: Some(cents),
        percent,
    })
}

/// The value each row would need for the rows to add up, by request position (`slots` long):
/// its own percent plus what is left of 100, or its cents plus the cents left of the charge.
/// `None` for a `null` row, for a result that is not positive (or above 100%), and for amount
/// rows when the total is unknown.
///
/// # Errors
///
/// When a sum overflows.
pub fn fills(
    rows: &[(usize, &AllocationRow)],
    slots: usize,
    total: Option<Decimal>,
    method: Method,
) -> Result<Vec<Option<Decimal>>, RequestError> {
    let mut fills = vec![None; slots];
    match method {
        Method::Percent => {
            let percents: Vec<Decimal> = rows
                .iter()
                .map(|(_, row)| row.percent.unwrap_or_default())
                .collect();
            let left = HUNDRED
                .checked_sub(checked_sum(&percents)?)
                .ok_or_else(overflow)?;
            for ((index, _), percent) in rows.iter().zip(percents) {
                let fill = percent.checked_add(left).ok_or_else(overflow)?;
                fills[*index] = (fill > Decimal::ZERO && fill <= HUNDRED).then_some(fill);
            }
        }
        Method::Amount => {
            let Some(total) = total else {
                return Ok(fills);
            };
            let cents: Vec<Decimal> = rows
                .iter()
                .map(|(_, row)| round_bank(row.amount.unwrap_or_default(), SHARE_PLACES))
                .collect();
            let left = round_bank(total, SHARE_PLACES)
                .checked_sub(checked_sum(&cents)?)
                .ok_or_else(overflow)?;
            for ((index, _), cents) in rows.iter().zip(cents) {
                let fill = cents.checked_add(left).ok_or_else(overflow)?;
                fills[*index] = (fill > Decimal::ZERO).then_some(fill);
            }
        }
    }
    Ok(fills)
}

/// The charge split evenly across the rows, by request position (`slots` long): every row but
/// the last gets the even share cut down to hundredths (percents) or to the cent (amounts), and
/// the last gets the rest, so the values add up exactly. `None` for a `null` row; all `None`
/// when there are no rows, or for amounts when the total is unknown or not positive.
///
/// # Errors
///
/// When a share overflows.
pub fn even_split(
    rows: &[(usize, &AllocationRow)],
    slots: usize,
    total: Option<Decimal>,
    method: Method,
) -> Result<Vec<Option<Decimal>>, RequestError> {
    let mut even = vec![None; slots];
    let (whole, places) = match method {
        Method::Percent => (HUNDRED, EVEN_PERCENT_PLACES),
        Method::Amount => match total.map(|total| round_bank(total, SHARE_PLACES)) {
            Some(cents) if cents > Decimal::ZERO => (cents, SHARE_PLACES),
            _ => return Ok(even),
        },
    };
    let Some(last) = rows.len().checked_sub(1) else {
        return Ok(even);
    };
    let count = Decimal::from(rows.len());
    let each = whole
        .checked_div(count)
        .ok_or_else(overflow)?
        .round_dp_with_strategy(places, RoundingStrategy::ToZero);
    let rest = whole
        .checked_sub(each.checked_mul(Decimal::from(last)).ok_or_else(overflow)?)
        .ok_or_else(overflow)?;
    for (position, (index, _)) in rows.iter().enumerate() {
        even[*index] = Some(if position == last { rest } else { each });
    }
    Ok(even)
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
