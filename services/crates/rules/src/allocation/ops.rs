//! The `allocation.*` operations. Each takes every charge of one shipment, order or form in one
//! call; see `docs/engineering/rules-engine.md` for the wire shapes.

use crate::allocation::charge::charge_total as total_of;
use crate::allocation::charge::overflow;
use crate::allocation::input::ChargesInput;
use crate::allocation::input::SplitInput;
use crate::allocation::input::dec;
use crate::allocation::rebalance::Rebalance;
use crate::allocation::rebalance::rebalance as rebalance_charge;
use crate::allocation::remainder::measure;
use crate::allocation::split::Method;
use crate::allocation::split::resolve;
use crate::allocation::split::rows_of;
use crate::wire::Dec;
use rust_decimal::Decimal;
use serde::Serialize;
use serde_json::value::RawValue;
use trenova_module_kit::message::Message;
use trenova_module_kit::ops::OpResult;
use trenova_module_kit::ops::decode;
use trenova_module_kit::ops::encode;

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct ShareOut<'a> {
    #[serde(skip_serializing_if = "Option::is_none")]
    allocation: Option<usize>,
    #[serde(skip_serializing_if = "Option::is_none")]
    allocation_id: Option<&'a str>,
    bill_to_customer_id: &'a str,
    method: Option<&'static str>,
    amount: Dec,
    percent: Option<Dec>,
    partial: bool,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct ChargeOut<'a> {
    total: Dec,
    #[serde(skip_serializing_if = "Option::is_none")]
    shares: Option<Vec<ShareOut<'a>>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<&'a Message>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct PayerOut<'a> {
    payer_id: &'a str,
    freight_amount: Dec,
    accessorial_amount: Dec,
    total_amount: Dec,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct SplitOut<'a> {
    charges: Vec<ChargeOut<'a>>,
    payers: Option<Vec<PayerOut<'a>>>,
    is_split: Option<bool>,
}

/// `allocation.split`: divides every charge between its payers and sums each payer.
pub fn split(input: &RawValue) -> OpResult {
    let input: SplitInput = decode(input)?;
    let resolution = resolve(&input)?;
    let charges = resolution
        .charges
        .iter()
        .map(|charge| match &charge.shares {
            Ok(shares) => ChargeOut {
                total: dec(charge.total),
                shares: Some(
                    shares
                        .iter()
                        .map(|share| ShareOut {
                            allocation: share.allocation,
                            allocation_id: share.allocation_id.as_deref(),
                            bill_to_customer_id: &share.payer,
                            method: share.method.map(Method::name),
                            amount: dec(share.amount),
                            percent: share.percent.map(dec),
                            partial: share.partial,
                        })
                        .collect(),
                ),
                error: None,
            },
            Err(message) => ChargeOut {
                total: dec(charge.total),
                shares: None,
                error: Some(message),
            },
        })
        .collect();
    let payers = resolution.payers.as_ref().map(|payers| {
        payers
            .iter()
            .map(|payer| PayerOut {
                payer_id: &payer.payer,
                freight_amount: dec(payer.freight),
                accessorial_amount: dec(payer.accessorial),
                total_amount: dec(payer.total),
            })
            .collect()
    });
    encode(&SplitOut {
        charges,
        payers,
        is_split: resolution.is_split,
    })
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct RemainderOut {
    method: &'static str,
    allocated: Dec,
    remaining: Option<Dec>,
    complete: bool,
    over: bool,
    mixed_methods: bool,
    duplicate_payers: bool,
    missing_payers: bool,
    problem: Option<Message>,
    valid: bool,
}

#[derive(Serialize)]
struct RemaindersOut {
    charges: Vec<RemainderOut>,
}

/// `allocation.remainder`: what each charge's split editor shows is assigned and left.
pub fn remainder(input: &RawValue) -> OpResult {
    let input: ChargesInput = decode(input)?;
    let base = input.base.unwrap_or_default();
    let mut charges = Vec::with_capacity(input.charges.len());
    for charge in &input.charges {
        let total = match (charge.total, charge.method.as_deref()) {
            (Some(total), _) => Some(total),
            (None, Some(_)) => Some(total_of(charge, base)?),
            (None, None) => None,
        };
        let rows: Vec<_> = charge.rows().map(|(_, row)| row).collect();
        let state = measure(&rows, total)?;
        charges.push(RemainderOut {
            method: state.method.name(),
            allocated: dec(state.allocated),
            remaining: state.remaining.map(dec),
            complete: state.complete,
            over: state.over,
            mixed_methods: state.mixed_methods,
            duplicate_payers: state.duplicate_payers,
            missing_payers: state.missing_payers,
            valid: state.problem.is_none(),
            problem: state.problem,
        });
    }
    encode(&RemaindersOut { charges })
}

#[derive(Serialize)]
struct TotalsOut {
    totals: Vec<Dec>,
    sum: Dec,
}

/// `allocation.chargeTotal`: what each charge comes to, and their sum.
pub fn charge_total(input: &RawValue) -> OpResult {
    let input: ChargesInput = decode(input)?;
    let base = input.base.unwrap_or_default();
    let mut totals = Vec::with_capacity(input.charges.len());
    let mut sum = Decimal::ZERO;
    for charge in &input.charges {
        let total = total_of(charge, base)?;
        sum = sum.checked_add(total).ok_or_else(overflow)?;
        totals.push(dec(total));
    }
    encode(&TotalsOut {
        totals,
        sum: dec(sum),
    })
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct RebalanceOut {
    stale: bool,
    allocated_total: Option<Dec>,
    charge_total: Option<Dec>,
    percents: Option<Vec<Option<Dec>>>,
}

#[derive(Serialize)]
struct RebalancesOut {
    charges: Vec<RebalanceOut>,
    converted: usize,
}

/// `allocation.rebalance`: finds amount splits that no longer add up and converts them to
/// percents that keep each payer's proportion.
pub fn rebalance(input: &RawValue) -> OpResult {
    let input: ChargesInput = decode(input)?;
    let base = input.base.unwrap_or_default();
    let mut charges = Vec::with_capacity(input.charges.len());
    let mut converted = 0;
    for charge in &input.charges {
        let total = total_of(charge, base)?;
        let state = rebalance_charge(total, &rows_of(charge), charge.allocations.len())?;
        charges.push(match state {
            Rebalance::Current => RebalanceOut {
                stale: false,
                allocated_total: None,
                charge_total: None,
                percents: None,
            },
            Rebalance::Stale {
                allocated,
                charge,
                percents,
            } => {
                if percents.is_some() {
                    converted += 1;
                }
                RebalanceOut {
                    stale: true,
                    allocated_total: Some(dec(allocated)),
                    charge_total: Some(dec(charge)),
                    percents: percents.map(|percents| {
                        percents
                            .into_iter()
                            .map(|percent| percent.map(dec))
                            .collect()
                    }),
                }
            }
        });
    }
    encode(&RebalancesOut { charges, converted })
}
