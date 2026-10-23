//! Searching between cases: shrinking a failure to its simplest case, and narrowing a monotone
//! failure to two neighbouring values.
//!
//! - **Shrinking.** For each axis in turn, the others fixed: on a range, the smallest value
//!   from the range's start that still fails, by halving between the start (which passes, or
//!   it is the answer) and the failing value; on a list, the first item that fails.
//! - **Narrowing.** Two cases one apart on an axis whose results are out of order are narrowed
//!   by halving the gap: the middle case's result is out of order with one side, which becomes
//!   the new pair, until the two are one step apart (or the middle case does not rate).
//!
//! Every search runs at most [`MAX_PROBES`] cases per property, and stops early when the run's
//! step meter is spent, keeping the best case found so far.

use super::{Driver, End, Found, Order, Probe, Record, Unordered};
use crate::tariff::eval::value::Value;
use crate::tariff::syntax::ast::Direction;
use rust_decimal::Decimal;
use std::cmp::Ordering;
use std::collections::BTreeMap;

/// The most cases a property's searches run.
pub const MAX_PROBES: usize = 512;

/// The most halvings one search makes.
const MAX_HALVINGS: usize = 96;

impl Driver<'_, '_, '_, '_> {
    /// Runs one search case, counted against [`MAX_PROBES`]; `None` once they are spent.
    fn probe(&mut self, values: &[Value]) -> Option<Probe> {
        if self.probes >= MAX_PROBES || self.runner.ledger.exhausted().is_some() {
            return None;
        }
        self.probes += 1;
        let probe = self.case(values);
        (!matches!(probe.end, End::Needs)).then_some(probe)
    }

    /// The simplest failing case: each axis in turn moved as low as it still fails.
    pub(super) fn shrink(
        &mut self,
        mut values: Vec<Value>,
        mut probe: Probe,
    ) -> (Vec<Value>, Probe) {
        for index in 0..self.axes.len() {
            let found = if self.axes.get(index).is_some_and(|axis| axis.line.is_some()) {
                self.shrink_range(&values, index)
            } else {
                self.shrink_list(&values, index)
            };
            if let Some((value, failing)) = found {
                if let Some(slot) = values.get_mut(index) {
                    *slot = value;
                }
                probe = failing;
            }
        }
        (values, probe)
    }

    /// The smallest value of a range axis that still fails, by halving from its start.
    fn shrink_range(&mut self, values: &[Value], index: usize) -> Option<(Value, Probe)> {
        let line = self.axes.get(index)?.line.clone()?;
        let mut fail = line.position(values.get(index)?)?;
        if fail <= line.start {
            return None;
        }
        let at = |position: Decimal| -> Option<Vec<Value>> {
            let mut moved = values.to_vec();
            *moved.get_mut(index)? = line.value(position)?;
            Some(moved)
        };
        let start = at(line.start)?;
        let first = self.probe(&start)?;
        if first.failed() {
            return Some((line.value(line.start)?, first));
        }
        let mut pass = line.start;
        let mut best: Option<Probe> = None;
        for _ in 0..MAX_HALVINGS {
            let Some(middle) = line.between(pass, fail) else {
                break;
            };
            let Some(probe) = at(middle).and_then(|moved| self.probe(&moved)) else {
                break;
            };
            if probe.failed() {
                fail = middle;
                best = Some(probe);
            } else {
                pass = middle;
            }
        }
        Some((line.value(fail)?, best?))
    }

    /// The first item of a list axis that still fails.
    fn shrink_list(&mut self, values: &[Value], index: usize) -> Option<(Value, Probe)> {
        let axis = self.axes.get(index)?;
        let current = values.get(index)?;
        let position = axis
            .points
            .iter()
            .position(|point| point.equals(current).unwrap_or(false))?;
        let earlier: Vec<Value> = axis.points.get(..position)?.to_vec();
        for point in earlier {
            let mut moved = values.to_vec();
            *moved.get_mut(index)? = point.clone();
            let probe = self.probe(&moved)?;
            if probe.failed() {
                return Some((point, probe));
            }
        }
        None
    }

    /// The first monotone expectation that does not hold over the passing cases, narrowed.
    pub(super) fn ordered(&mut self, records: &[Record]) -> Found {
        for (number, order) in self.orders().into_iter().enumerate() {
            if let Some(pair) = first_unordered(records, number, order) {
                return self.narrow(number, order, pair);
            }
        }
        Found::Passed(Vec::new())
    }

    /// Narrows an out-of-order pair to neighbouring values.
    fn narrow(&mut self, number: usize, order: Order<'_>, pair: Pair) -> Found {
        let base = self.values_of(&pair.combination);
        let axis = order.axis;
        let points = self
            .axes
            .get(axis)
            .map(|axis| axis.points.clone())
            .unwrap_or_default();
        let (Some(mut low_at), Some(mut high_at)) = (
            points.get(pair.low).cloned(),
            points.get(pair.high).cloned(),
        ) else {
            return Found::Passed(Vec::new());
        };
        let (mut low, mut high) = (pair.low_result, pair.high_result);
        if let Some(line) = self.axes.get(axis).and_then(|axis| axis.line.clone()) {
            for _ in 0..MAX_HALVINGS {
                let (Some(from), Some(to)) = (line.position(&low_at), line.position(&high_at))
                else {
                    break;
                };
                let Some(middle) = line.between(from, to).and_then(|at| line.value(at)) else {
                    break;
                };
                let mut moved = base.clone();
                if let Some(slot) = moved.get_mut(axis) {
                    *slot = middle.clone();
                }
                let Some(probe) = self.probe(&moved) else {
                    break;
                };
                let Some(Some(result)) = matches!(probe.end, End::Passed(_))
                    .then(|| probe.monotone.get(number).cloned().flatten())
                else {
                    break;
                };
                if out_of_order(order.direction, &low, &result) {
                    high_at = middle;
                    high = result;
                } else {
                    low_at = middle;
                    low = result;
                }
            }
        }
        let mut values = base;
        if let Some(slot) = values.get_mut(axis) {
            *slot = low_at.clone();
        }
        let bill = match self.probe(&values).map(|probe| probe.end) {
            Some(End::Passed(bill)) => bill,
            _ => None,
        };
        Found::Unordered(Box::new(Unordered {
            order: number,
            values,
            before: (low_at, low),
            after: (high_at, high),
            bill,
        }))
    }
}

/// A passing case on a group's line: its value index on the axis, its value indexes and its
/// result.
type Placed<'r> = (usize, &'r Vec<usize>, &'r Value);

/// Two cases along one axis, the others fixed, whose results are out of order.
struct Pair {
    combination: Vec<usize>,
    low: usize,
    high: usize,
    low_result: Value,
    high_result: Value,
}

/// The first out-of-order neighbours of monotone expectation `number`: the cases grouped by
/// their other axes (in the order the groups first ran), each group in the axis's order.
fn first_unordered(records: &[Record], number: usize, order: Order<'_>) -> Option<Pair> {
    let mut groups: BTreeMap<Vec<usize>, Vec<Placed<'_>>> = BTreeMap::new();
    let mut first_seen: Vec<Vec<usize>> = Vec::new();
    for (combination, results) in records {
        let Some(Some(result)) = results.get(number) else {
            continue;
        };
        let Some(&position) = combination.get(order.axis) else {
            continue;
        };
        let mut key = combination.clone();
        if let Some(slot) = key.get_mut(order.axis) {
            *slot = usize::MAX;
        }
        if !groups.contains_key(&key) {
            first_seen.push(key.clone());
        }
        groups
            .entry(key)
            .or_default()
            .push((position, combination, result));
    }
    for key in first_seen {
        let Some(mut group) = groups.remove(&key) else {
            continue;
        };
        group.sort_by_key(|(position, _, _)| *position);
        for pair in group.windows(2) {
            let [(low, combination, low_result), (high, _, high_result)] = pair else {
                continue;
            };
            if out_of_order(order.direction, low_result, high_result) {
                return Some(Pair {
                    combination: (*combination).clone(),
                    low: *low,
                    high: *high,
                    low_result: (*low_result).clone(),
                    high_result: (*high_result).clone(),
                });
            }
        }
    }
    None
}

/// Whether `before` (at the lower value) and `after` break the direction.
fn out_of_order(direction: Direction, before: &Value, after: &Value) -> bool {
    matches!(
        (direction, before.compare(after)),
        (Direction::Nondecreasing, Ok(Ordering::Greater))
            | (Direction::Nonincreasing, Ok(Ordering::Less))
    )
}
