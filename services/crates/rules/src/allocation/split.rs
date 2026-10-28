//! Dividing charges between payers, as `chargeallocation.ResolveShares` and
//! `ResolveOrderChargeShares` do.
//!
//! Each charge is divided on its own:
//!
//! - With no allocations, the whole charge, rounded to cents half to even, goes to the default
//!   payer at 100%.
//! - Rows are ordered by sequence, then by ID; rows without an ID (unsaved, in the browser) keep
//!   their request order among themselves and sort before saved rows of the same sequence,
//!   which is where Go's empty ID sorts.
//! - Every row must use the same method, name a payer, and use `Percent` or `Amount`.
//! - Percent rows must each be in (0, 100] and total 100 at six places. Every share but the last
//!   is `total × percent / 100` rounded half to even to cents; the last is the rounded total
//!   minus the others, so the shares always add up to the rounded charge.
//! - Amount rows are rounded half to even to cents and must add up to the rounded charge. Their
//!   percent is `amount × 100 / total` (shopspring division, rounded to six places), or none
//!   when the charge is zero.
//! - A share is partial when it is not the whole rounded charge.
//!
//! The resolution then sums each payer's shares into freight (freight charges) and accessorial
//! (accessorial and order charges) subtotals. The default payer always leads, even with an empty
//! share; the others follow by ID. A shipment is split when any share is partial or goes to a
//! payer other than the default, or when more than one payer is billed.

use crate::allocation::charge::charge_total;
use crate::allocation::charge::overflow;
use crate::allocation::input::AllocationRow;
use crate::allocation::input::ChargeInput;
use crate::allocation::input::SplitInput;
use crate::report::fixed::div_round;
use crate::report::fixed::round_bank;
use crate::report::fixed::round_half_away;
use crate::report::fixed::string_fixed;
use rust_decimal::Decimal;
use std::collections::BTreeMap;
use trenova_module_kit::message;
use trenova_module_kit::message::Arg;
use trenova_module_kit::message::Message;
use trenova_module_kit::ops::RequestError;

/// Cents: every share is rounded to this many places.
pub const SHARE_PLACES: u32 = 2;

/// The precision percents are compared and stored at.
pub const PERCENT_PLACES: u32 = 6;

const HUNDRED: Decimal = Decimal::ONE_HUNDRED;

/// The split method a row uses.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Method {
    /// A percent of the charge.
    Percent,
    /// A fixed amount of the charge.
    Amount,
}

impl Method {
    /// The method a wire name stands for.
    #[must_use]
    pub fn parse(name: Option<&str>) -> Option<Self> {
        match name {
            Some("Percent") => Some(Self::Percent),
            Some("Amount") => Some(Self::Amount),
            _ => None,
        }
    }

    /// The wire name.
    #[must_use]
    pub const fn name(self) -> &'static str {
        match self {
            Self::Percent => "Percent",
            Self::Amount => "Amount",
        }
    }
}

/// Which payer subtotal a charge's shares join.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Kind {
    /// The shipment's freight charge.
    Freight,
    /// An accessorial charge on the shipment.
    Accessorial,
    /// An order-level charge.
    OrderCharge,
}

impl Kind {
    fn parse(name: Option<&str>) -> Result<Self, RequestError> {
        match name {
            Some("Freight") => Ok(Self::Freight),
            Some("Accessorial") => Ok(Self::Accessorial),
            Some("OrderCharge") => Ok(Self::OrderCharge),
            other => Err(RequestError::invalid_input(format!(
                "a charge's kind must be Freight, Accessorial or OrderCharge, not {other:?}"
            ))),
        }
    }
}

/// One payer's slice of one charge.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Share {
    /// The row's position in the charge's `allocations`; `None` for an unallocated charge.
    pub allocation: Option<usize>,
    /// The row's ID, when it has one.
    pub allocation_id: Option<String>,
    /// The payer.
    pub payer: String,
    /// How the charge was split; `None` when it was billed whole.
    pub method: Option<Method>,
    /// The share, in cents.
    pub amount: Decimal,
    /// The share's percent of the charge; `None` for an amount share of a zero charge.
    pub percent: Option<Decimal>,
    /// Whether the share is less (or more) than the whole charge.
    pub partial: bool,
}

/// The rows of a charge in the order the split reads them: by sequence, then by ID, with rows
/// that have no ID first among equals and in request order.
#[must_use]
pub fn ordered<'a>(rows: &[(usize, &'a AllocationRow)]) -> Vec<(usize, &'a AllocationRow)> {
    let mut sorted = rows.to_vec();
    sorted.sort_by(|(_, left), (_, right)| {
        left.sequence
            .unwrap_or_default()
            .cmp(&right.sequence.unwrap_or_default())
            .then_with(|| {
                left.id
                    .as_deref()
                    .unwrap_or_default()
                    .cmp(right.id.as_deref().unwrap_or_default())
            })
    });
    sorted
}

/// Divides one charge among its rows, or explains why it cannot be divided.
///
/// # Errors
///
/// A request error only when an amount overflows; a split that breaks a rule is the inner
/// `Err(Message)`.
pub fn allocate(
    total: Decimal,
    rows: &[(usize, &AllocationRow)],
    default_payer: &str,
) -> Result<Result<Vec<Share>, Message>, RequestError> {
    let rounded = round_bank(total, SHARE_PLACES);
    if rows.is_empty() {
        return Ok(Ok(vec![Share {
            allocation: None,
            allocation_id: None,
            payer: default_payer.to_owned(),
            method: None,
            amount: rounded,
            percent: Some(HUNDRED),
            partial: false,
        }]));
    }

    let sorted = ordered(rows);
    let first = sorted[0].1.method.as_deref();
    if sorted.iter().any(|(_, row)| row.method.as_deref() != first) {
        return Ok(Err(Message::new(message!(
            "A charge must be split either by percent or by amount, not both"
        ))));
    }
    let Some(method) = Method::parse(first) else {
        return Ok(Err(Message::new(message!(
            "Allocation method must be Percent or Amount"
        ))));
    };
    if sorted.iter().any(|(_, row)| row.payer().is_none()) {
        return Ok(Err(Message::new(message!("Bill-to customer is required"))));
    }

    let amounts = match method {
        Method::Percent => match percent_shares(total, rounded, &sorted)? {
            Some(amounts) => amounts,
            None => {
                return Ok(Err(Message::new(message!(
                    "Percent allocations for this charge must total 100"
                ))));
            }
        },
        Method::Amount => {
            let amounts: Vec<Decimal> = sorted
                .iter()
                .map(|(_, row)| round_bank(row.amount.unwrap_or_default(), SHARE_PLACES))
                .collect();
            if checked_sum(&amounts)? != rounded {
                return Ok(Err(Message::with(
                    message!("Amount allocations for this charge must add up to {0}"),
                    [Arg::from(string_fixed(total, SHARE_PLACES))],
                )));
            }
            amounts
        }
    };

    let mut shares = Vec::with_capacity(sorted.len());
    for ((index, row), amount) in sorted.iter().zip(amounts) {
        let percent = match method {
            Method::Percent => row.percent,
            Method::Amount => amount_percent(amount, total)?,
        };
        shares.push(Share {
            allocation: Some(*index),
            allocation_id: row.id.clone(),
            payer: row.payer().unwrap_or_default().to_owned(),
            method: Some(method),
            amount,
            percent,
            partial: amount != rounded,
        });
    }
    Ok(Ok(shares))
}

/// The cents of each percent row, or `None` when the percents are not a valid split
/// (`decimalutils.AllocatePercent`).
fn percent_shares(
    total: Decimal,
    rounded: Decimal,
    sorted: &[(usize, &AllocationRow)],
) -> Result<Option<Vec<Decimal>>, RequestError> {
    let percents: Vec<Decimal> = sorted
        .iter()
        .map(|(_, row)| row.percent.unwrap_or_default())
        .collect();
    if percents
        .iter()
        .any(|percent| *percent <= Decimal::ZERO || *percent > HUNDRED)
    {
        return Ok(None);
    }
    if round_half_away(checked_sum(&percents)?, PERCENT_PLACES) != HUNDRED {
        return Ok(None);
    }
    let last = percents.len() - 1;
    let mut shares = Vec::with_capacity(percents.len());
    let mut allocated = Decimal::ZERO;
    for percent in &percents[..last] {
        let share = total
            .checked_mul(*percent)
            .and_then(|product| product.checked_div(HUNDRED))
            .map(|share| round_bank(share, SHARE_PLACES))
            .ok_or_else(overflow)?;
        allocated = allocated.checked_add(share).ok_or_else(overflow)?;
        shares.push(share);
    }
    shares.push(rounded.checked_sub(allocated).ok_or_else(overflow)?);
    Ok(Some(shares))
}

/// What percent of the charge an amount share is: shopspring's `amount × 100 / total`, rounded
/// to six places.
fn amount_percent(amount: Decimal, total: Decimal) -> Result<Option<Decimal>, RequestError> {
    if total.is_zero() {
        return Ok(None);
    }
    let scaled = amount.checked_mul(HUNDRED).ok_or_else(overflow)?;
    div_round(scaled, total)
        .and_then(|quotient| quotient.round(PERCENT_PLACES))
        .map(Some)
        .ok_or_else(overflow)
}

/// Adds decimals, failing on overflow.
///
/// # Errors
///
/// When the sum overflows.
pub fn checked_sum(values: &[Decimal]) -> Result<Decimal, RequestError> {
    values.iter().try_fold(Decimal::ZERO, |sum, value| {
        sum.checked_add(*value).ok_or_else(overflow)
    })
}

/// One charge's outcome.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChargeSplit {
    /// The charge total, exact and unrounded.
    pub total: Decimal,
    /// The shares, or why the charge cannot be split.
    pub shares: Result<Vec<Share>, Message>,
}

/// Everything one payer is billed for.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PayerShare {
    /// The payer.
    pub payer: String,
    /// The freight shares.
    pub freight: Decimal,
    /// The accessorial and order charge shares.
    pub accessorial: Decimal,
    /// Both.
    pub total: Decimal,
}

/// The outcome of a whole split.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Resolution {
    /// Each charge's split, in request order.
    pub charges: Vec<ChargeSplit>,
    /// The payers, default first and the others by ID; `None` when any charge failed.
    pub payers: Option<Vec<PayerShare>>,
    /// Whether billing departs from one invoice to the default payer; `None` with `payers`.
    pub is_split: Option<bool>,
}

/// Splits every charge and sums the payers.
///
/// # Errors
///
/// When a charge's kind is missing or unknown, or an amount overflows.
pub fn resolve(input: &SplitInput) -> Result<Resolution, RequestError> {
    let base = input.base.unwrap_or_default();
    let default_payer = input.default_payer_id.as_str();
    let mut charges = Vec::with_capacity(input.charges.len());
    let mut payers: BTreeMap<String, PayerShare> = BTreeMap::new();
    payers.insert(default_payer.to_owned(), empty_share(default_payer));
    let mut split = false;
    let mut failed = false;

    for charge in &input.charges {
        let kind = Kind::parse(charge.kind.as_deref())?;
        let total = charge_total(charge, base)?;
        let shares = allocate(total, &rows_of(charge), default_payer)?;
        if let Ok(shares) = &shares {
            for share in shares {
                if share.partial || share.payer != default_payer {
                    split = true;
                }
                let payer = payers
                    .entry(share.payer.clone())
                    .or_insert_with(|| empty_share(&share.payer));
                add(payer, kind, share.amount)?;
            }
        } else {
            failed = true;
        }
        charges.push(ChargeSplit { total, shares });
    }

    if failed {
        return Ok(Resolution {
            charges,
            payers: None,
            is_split: None,
        });
    }
    let lead = payers
        .remove(default_payer)
        .unwrap_or_else(|| empty_share(default_payer));
    let mut ordered = Vec::with_capacity(payers.len() + 1);
    ordered.push(lead);
    ordered.extend(payers.into_values());
    let is_split = split || ordered.len() > 1;
    Ok(Resolution {
        charges,
        payers: Some(ordered),
        is_split: Some(is_split),
    })
}

/// A charge's rows with their request positions.
#[must_use]
pub fn rows_of(charge: &ChargeInput) -> Vec<(usize, &AllocationRow)> {
    charge.rows().collect()
}

fn empty_share(payer: &str) -> PayerShare {
    PayerShare {
        payer: payer.to_owned(),
        freight: Decimal::ZERO,
        accessorial: Decimal::ZERO,
        total: Decimal::ZERO,
    }
}

fn add(payer: &mut PayerShare, kind: Kind, amount: Decimal) -> Result<(), RequestError> {
    match kind {
        Kind::Freight => payer.freight = payer.freight.checked_add(amount).ok_or_else(overflow)?,
        Kind::Accessorial | Kind::OrderCharge => {
            payer.accessorial = payer.accessorial.checked_add(amount).ok_or_else(overflow)?;
        }
    }
    payer.total = payer.total.checked_add(amount).ok_or_else(overflow)?;
    Ok(())
}
