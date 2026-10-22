//! List properties and methods: `len`, `isEmpty`, and `where map count sum min max avg any all
//! first last find sortBy take skip reverse distinct groupBy`.
//!
//! A method's lambda runs through a [`Host`], which evaluates it in the scope it was written
//! in; a two-parameter lambda (`(item, index) => …`) also gets the item's 0-based position.
//! Every item a method visits costs a step and every item it creates counts against the items
//! budget, so each method is bounded by the list it walks.
//!
//! - `min`/`max` with a selector give the smallest or largest selected value (`stops.max(s =>
//!   s.dwell)` is the longest dwell); when the static type says the call gives the list's own
//!   item type, they give the item with that value instead.
//! - `sum` of an empty list is the zero of the call's static type (`$0`, `0 h`, `0`), `0` when
//!   the type is not known. `avg`, `min`, `max`, `first`, `last` and `find` of an empty list are
//!   `none`.
//! - `sortBy` is stable and ascending; `distinct` keeps each value's first occurrence;
//!   `groupBy` gives `{ key, items }` per key in order of first appearance.

use super::budget::Meter;
use super::error::{EvalError, Stop};
use super::operators;
use super::value::Value;
use crate::message::{Arg, Message};
use crate::tariff::types::Type;
use crate::tariff::units::{Base, Dim, Unit};
use rust_decimal::Decimal;
use rust_decimal::prelude::ToPrimitive;
use std::cmp::Ordering;

/// What runs a list method's lambda.
pub trait Host {
    /// Whether the call has a lambda.
    fn has_lambda(&self) -> bool;
    /// The lambda's value for one item and its position; the item itself when there is no
    /// lambda.
    fn apply(&mut self, item: &Value, index: usize) -> Result<Value, Stop>;
    /// The budgets.
    fn meter(&mut self) -> &mut Meter;
}

/// What the static types say about a call, for the cases the values cannot decide.
#[derive(Debug, Clone, Default)]
pub struct Hints {
    /// The zero an empty `sum` gives.
    pub empty_sum: Option<Value>,
    /// Whether `min`/`max` give the item rather than the selected value.
    pub pick_item: bool,
}

/// The zero of a type: `$0`, `0 h`, `0%`, `0`.
#[must_use]
pub fn zero_of(ty: &Type) -> Option<Value> {
    match ty.unwrapped() {
        Type::Int => Some(Value::Int(Decimal::ZERO)),
        Type::Number => Some(Value::Number(Decimal::ZERO)),
        Type::Percent => Some(operators::zero_percent()),
        Type::Quantity(quantity) => {
            let unit = quantity
                .display
                .clone()
                .or_else(|| canonical_unit(quantity.dim))?;
            Some(Value::quantity(Decimal::ZERO, unit))
        }
        _ => None,
    }
}

/// The unit a dimension's zero is shown in when the type names none: `$`, `lb`, `mi`, `ft`,
/// `ft3`, `h`, `°F`, and their quotients.
fn canonical_unit(dim: Dim) -> Option<Unit> {
    let symbol = |base: Base| match base {
        Base::Money => "$",
        Base::Weight => "lb",
        Base::Distance => "mi",
        Base::Length => "ft",
        Base::Volume => "ft3",
        Base::Duration => "h",
        Base::Temperature => "°F",
    };
    let mut top = None;
    let mut bottom = None;
    for base in [
        Base::Money,
        Base::Weight,
        Base::Distance,
        Base::Length,
        Base::Volume,
        Base::Duration,
        Base::Temperature,
    ] {
        match dim.exponent(base) {
            0 => {}
            1 if top.is_none() => top = Some(symbol(base)),
            -1 if bottom.is_none() => bottom = Some(symbol(base)),
            _ => return None,
        }
    }
    let text = match (top, bottom) {
        (Some(top), Some(bottom)) => format!("{top}/{bottom}"),
        (Some(top), None) => top.to_owned(),
        _ => return None,
    };
    Unit::parse(&text).ok()
}

/// `len` or `isEmpty`.
#[must_use]
pub fn property(items: &[Value], name: &str) -> Option<Value> {
    match name {
        "len" => Some(Value::Int(Decimal::from(items.len()))),
        "isEmpty" => Some(Value::Bool(items.is_empty())),
        _ => None,
    }
}

fn failed(error: EvalError) -> Stop {
    Stop::Failed(error)
}

fn expect_bool(value: &Value) -> Result<bool, Stop> {
    match value {
        Value::Bool(flag) => Ok(*flag),
        other => Err(failed(EvalError::internal(Message::with(
            crate::message!("A list method's test must be true or false, not {0}"),
            [Arg::from(other.kind())],
        )))),
    }
}

fn count_arg(name: &str, args: &[Value]) -> Result<usize, Stop> {
    match args.first() {
        Some(Value::Int(count)) if !count.is_sign_negative() => {
            Ok(count.to_usize().unwrap_or(usize::MAX))
        }
        Some(Value::Int(_)) => Ok(0),
        other => Err(failed(EvalError::internal(Message::with(
            crate::message!("{0} needs a whole number, not {1}"),
            [
                Arg::from(name),
                Arg::from(other.map_or_else(|| Value::None.kind(), Value::kind)),
            ],
        )))),
    }
}

fn made(host: &mut dyn Host, items: Vec<Value>) -> Result<Value, Stop> {
    host.meter().items(items.len()).map_err(failed)?;
    Ok(Value::list(items))
}

/// Calls a list method, or `None` when lists have no method by that name.
pub fn call(
    items: &[Value],
    name: &str,
    args: &[Value],
    hints: &Hints,
    host: &mut dyn Host,
) -> Option<Result<Value, Stop>> {
    Some(match name {
        "where" | "count" | "any" | "all" | "first" | "last" | "find" => filter(items, name, host),
        "map" => mapped(items, host).and_then(|values| made(host, values)),
        "sum" => sum(items, hints, host),
        "avg" => average(items, host),
        "min" | "max" => extreme(items, name == "max", hints, host),
        "sortBy" => sort_by(items, host),
        "take" | "skip" => count_arg(name, args).and_then(|count| {
            let kept: Vec<Value> = if name == "take" {
                items.iter().take(count).cloned().collect()
            } else {
                items.iter().skip(count).cloned().collect()
            };
            host.meter().steps(kept.len()).map_err(failed)?;
            made(host, kept)
        }),
        "reverse" => {
            let reversed: Vec<Value> = items.iter().rev().cloned().collect();
            host.meter()
                .steps(reversed.len())
                .map_err(failed)
                .and_then(|()| made(host, reversed))
        }
        "distinct" => distinct(items, host),
        "groupBy" => group_by(items, host),
        _ => return None,
    })
}

fn mapped(items: &[Value], host: &mut dyn Host) -> Result<Vec<Value>, Stop> {
    let mut out = Vec::with_capacity(items.len());
    for (index, item) in items.iter().enumerate() {
        host.meter().step().map_err(failed)?;
        out.push(host.apply(item, index)?);
    }
    Ok(out)
}

fn filter(items: &[Value], name: &str, host: &mut dyn Host) -> Result<Value, Stop> {
    let lambda = host.has_lambda();
    let mut kept = Vec::new();
    let order: Box<dyn Iterator<Item = (usize, &Value)>> = if name == "last" {
        Box::new(items.iter().enumerate().rev())
    } else {
        Box::new(items.iter().enumerate())
    };
    for (index, item) in order {
        host.meter().step().map_err(failed)?;
        let holds = if lambda {
            expect_bool(&host.apply(item, index)?)?
        } else {
            true
        };
        match (name, holds) {
            ("any", true) => return Ok(Value::Bool(true)),
            ("all", false) => return Ok(Value::Bool(false)),
            ("first" | "last" | "find", true) => return Ok(item.clone()),
            ("where" | "count", true) => kept.push(item.clone()),
            _ => {}
        }
    }
    match name {
        "any" => Ok(Value::Bool(false)),
        "all" => Ok(Value::Bool(true)),
        "where" => made(host, kept),
        "count" => Ok(Value::Int(Decimal::from(kept.len()))),
        _ => Ok(Value::None),
    }
}

fn total(values: &[Value]) -> Result<Option<Value>, Stop> {
    let mut iter = values.iter();
    let Some(first) = iter.next() else {
        return Ok(None);
    };
    let mut sum = first.clone();
    for value in iter {
        sum = operators::add(&sum, value).map_err(failed)?;
    }
    Ok(Some(sum))
}

fn sum(items: &[Value], hints: &Hints, host: &mut dyn Host) -> Result<Value, Stop> {
    let values = mapped(items, host)?;
    Ok(total(&values)?
        .unwrap_or_else(|| hints.empty_sum.clone().unwrap_or(Value::Int(Decimal::ZERO))))
}

fn average(items: &[Value], host: &mut dyn Host) -> Result<Value, Stop> {
    let values = mapped(items, host)?;
    match total(&values)? {
        None => Ok(Value::None),
        Some(sum) => {
            operators::div(&sum, &Value::Int(Decimal::from(values.len())), true).map_err(failed)
        }
    }
}

fn extreme(
    items: &[Value],
    greatest: bool,
    hints: &Hints,
    host: &mut dyn Host,
) -> Result<Value, Stop> {
    let keys = mapped(items, host)?;
    let mut best: Option<usize> = None;
    for (index, key) in keys.iter().enumerate() {
        if key.is_none() {
            continue;
        }
        let better = match best.and_then(|at| keys.get(at)) {
            None => true,
            Some(current) => {
                let order = key.compare(current).map_err(failed)?;
                if greatest {
                    order == Ordering::Greater
                } else {
                    order == Ordering::Less
                }
            }
        };
        if better {
            best = Some(index);
        }
    }
    let Some(at) = best else {
        return Ok(Value::None);
    };
    let pool = if hints.pick_item && host.has_lambda() {
        items
    } else {
        &keys
    };
    Ok(pool.get(at).cloned().unwrap_or(Value::None))
}

fn sort_by(items: &[Value], host: &mut dyn Host) -> Result<Value, Stop> {
    let keys = mapped(items, host)?;
    let mut order: Vec<usize> = (0..items.len()).collect();
    let mut problem: Option<EvalError> = None;
    let comparisons = items.len().saturating_mul(usize::BITS as usize);
    host.meter().steps(comparisons).map_err(failed)?;
    order.sort_by(|left, right| match (keys.get(*left), keys.get(*right)) {
        (Some(left), Some(right)) => left.compare(right).unwrap_or_else(|error| {
            problem.get_or_insert(error);
            Ordering::Equal
        }),
        _ => Ordering::Equal,
    });
    if let Some(error) = problem {
        return Err(failed(error));
    }
    let sorted = order
        .into_iter()
        .filter_map(|index| items.get(index).cloned())
        .collect();
    made(host, sorted)
}

fn distinct(items: &[Value], host: &mut dyn Host) -> Result<Value, Stop> {
    let mut kept: Vec<Value> = Vec::new();
    for item in items {
        host.meter().steps(kept.len().max(1)).map_err(failed)?;
        let mut seen = false;
        for existing in &kept {
            if existing.equals(item).map_err(failed)? {
                seen = true;
                break;
            }
        }
        if !seen {
            kept.push(item.clone());
        }
    }
    made(host, kept)
}

fn group_by(items: &[Value], host: &mut dyn Host) -> Result<Value, Stop> {
    let keys = mapped(items, host)?;
    let mut groups: Vec<(Value, Vec<Value>)> = Vec::new();
    for (key, item) in keys.into_iter().zip(items) {
        host.meter().steps(groups.len().max(1)).map_err(failed)?;
        let mut slot = None;
        for (index, (existing, _)) in groups.iter().enumerate() {
            if existing.equals(&key).map_err(failed)? {
                slot = Some(index);
                break;
            }
        }
        match slot.and_then(|index| groups.get_mut(index)) {
            Some((_, members)) => members.push(item.clone()),
            None => groups.push((key, vec![item.clone()])),
        }
    }
    host.meter()
        .items(groups.len().saturating_mul(2))
        .map_err(failed)?;
    Ok(Value::list(
        groups
            .into_iter()
            .map(|(key, members)| {
                Value::record(vec![("key", key), ("items", Value::list(members))])
            })
            .collect(),
    ))
}
