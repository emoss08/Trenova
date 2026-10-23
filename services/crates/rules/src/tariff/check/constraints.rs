//! The rules of an input's `where` constraint, checked in the declaration phase as the
//! constraint is folded (`decls`):
//!
//! - the constraint is a range or a list literal; anything else is an error showing the forms;
//! - a record input, or a list of records, takes no constraint;
//! - a range needs a type with an order (numbers, `Int`, money, quantities, rates, percents,
//!   dates, date-times) and at least one end; every type a list may hold has equality;
//! - a range must not be empty (`min` above `max`, or equal to an excluded `max`), and a list
//!   must hold a value;
//! - the default satisfies the constraint, judged by [`crate::tariff::constraint::breach`] (the
//!   test every caller uses) and worded as a contract value's finding is.
//!
//! Every finding has code `constraint`.

use super::ModuleAnalysis;
use super::consts::{Value, wire};
use super::decls::FoldedConstraint;
use super::describe::{quote, words};
use super::operators::ordered;
use crate::message;
use crate::message::{Arg, Message};
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::eval::value::Value as RunValue;
use crate::tariff::syntax::ast::{Expr, ExprKind};
use crate::tariff::types::Type;
use std::cmp::Ordering;

/// The code of every constraint finding.
const CODE: &str = "constraint";

/// Checks the form of input `name`'s constraint against its type, before its parts are folded;
/// false when the constraint cannot be folded (the reason is reported).
pub fn form(
    out: &mut ModuleAnalysis,
    name: &str,
    ty: &Type,
    item: &Type,
    constraint: &Expr,
) -> bool {
    if holds_records(ty) {
        report(
            out,
            constraint,
            Message::with(
                message!(
                    "{0} is {1}: a record input, or a list of records, takes no `where` constraint"
                ),
                [quote(name), Arg::from(words(ty))],
            ),
        );
        return false;
    }
    match &constraint.kind {
        ExprKind::Range { start, end, .. } => {
            if !ordered(item) {
                report(
                    out,
                    constraint,
                    Message::with(
                        message!(
                            "{0} is {1}, which has no order, so its constraint can't be a range: list the values it may take, as in `where [a, b]`"
                        ),
                        [quote(name), Arg::from(words(item))],
                    ),
                );
                return false;
            }
            if start.is_none() && end.is_none() {
                report(
                    out,
                    constraint,
                    Message::new(message!(
                        "A range constraint needs at least one end, as in `0..`, `..=10` or `0..=10`"
                    )),
                );
                return false;
            }
            true
        }
        ExprKind::List(_) => true,
        _ => {
            report(
                out,
                constraint,
                Message::new(message!(
                    "A `where` constraint is a range or a list of values: `lo..hi` (from `lo` up to but not including `hi`), `lo..=hi` (up to and including `hi`), or `[a, b]`; either end of a range may be left off"
                )),
            );
            false
        }
    }
}

/// Checks a folded constraint admits some value: a range that is not empty, a list that is not;
/// false when it admits none (reported).
pub fn admits_some(
    out: &mut ModuleAnalysis,
    item: &Type,
    constraint: &Expr,
    folded: &FoldedConstraint,
) -> bool {
    match folded {
        FoldedConstraint::OneOf(values) if values.is_empty() => {
            report(
                out,
                constraint,
                Message::new(message!(
                    "This list is empty, so no value can satisfy the constraint"
                )),
            );
            false
        }
        FoldedConstraint::Range {
            min: Some(min),
            max: Some(max),
            max_inclusive,
        } => {
            let (Some(low), Some(high)) = (run_value(min, item), run_value(max, item)) else {
                return true;
            };
            let empty = match low.compare(&high) {
                Ok(Ordering::Greater) => true,
                Ok(Ordering::Equal) => !*max_inclusive,
                _ => false,
            };
            if empty {
                let message = if *max_inclusive {
                    Message::with(
                        message!("This range is empty: its start {0} is above its end {1}"),
                        [Arg::from(low.display()), Arg::from(high.display())],
                    )
                } else {
                    Message::with(
                        message!("This range is empty: nothing is at least {0} and below {1}"),
                        [Arg::from(low.display()), Arg::from(high.display())],
                    )
                };
                report(out, constraint, message);
            }
            !empty
        }
        FoldedConstraint::OneOf(_) | FoldedConstraint::Range { .. } => true,
    }
}

/// Refuses a default the constraint does not admit, at the default, with the words a contract
/// value outside it gets.
pub fn default(
    out: &mut ModuleAnalysis,
    name: &str,
    ty: &Type,
    written: &Expr,
    value: &Value,
    folded: &FoldedConstraint,
) {
    let Some(value) = run_value(value, ty) else {
        return;
    };
    let constraint = folded.to_wire();
    if let Some(breach) = crate::tariff::constraint::breach(&constraint, &value) {
        out.diagnostics.push(Diagnostic::error(
            CODE,
            written.span,
            breach.message(&constraint, name),
        ));
    }
}

/// A folded value as the evaluator holds it, read through the wire codec against its type.
pub fn run_value(value: &Value, ty: &Type) -> Option<RunValue> {
    crate::tariff::wire::decode(&wire(value), ty, "").ok()
}

/// Whether a type is a record, or holds records through lists and optionals.
fn holds_records(ty: &Type) -> bool {
    match ty {
        Type::Record(_) => true,
        Type::List(inner) | Type::Optional(inner) => holds_records(inner),
        _ => false,
    }
}

fn report(out: &mut ModuleAnalysis, constraint: &Expr, message: Message) {
    out.diagnostics
        .push(Diagnostic::error(CODE, constraint.span, message));
}
