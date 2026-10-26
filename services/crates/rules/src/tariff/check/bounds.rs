//! What an input's `where` bounds decide, judged once a module is checked:
//!
//! - a comparison of the input with a compile-time value (`==`, `!=`, `<`, `<=`, `>`, `>=`,
//!   `in [...]`, `not in [...]`, `in lo..hi`, `not in lo..hi`) whose answer the bounds fix is a
//!   `bounds` warning, "never true" or "always true", quoting the constraint;
//! - a `match` arm on the input whose pattern the bounds rule out is a `bounds` warning;
//! - a test's or property's `given input.x = value` outside the bounds is a `constraint` error,
//!   since no contract can set that value.
//!
//! Against a list constraint, a value outside the list suggests the nearest listed value, with a
//! fix that writes it as the constraint does. Only an input read by its own name is judged: a
//! value read through another binding, or compared with something computed at run time, is not.

use super::checker::Globals;
use super::consts::{FoldError, Folder, Value, compare};
use super::decls::{FoldedConstraint, State, constant_of};
use super::describe::{quote, quote_span, snippet};
use super::fixes::replace;
use super::{ModuleAnalysis, SymbolKind};
use crate::span::Span;
use crate::tariff::diagnostic::{Diagnostic, nearest};
use crate::tariff::syntax::ast::{
    BinaryOp, Expr, ExprKind, ItemKind, Pattern, PatternKind, Stmt, StmtKind,
};
use crate::tariff::syntax::program::Module;
use crate::tariff::syntax::walk::{self, Visit};
use crate::tariff::types::Type;
use std::cmp::Ordering;
use std::collections::BTreeMap;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

/// The code of the warnings.
const CODE: &str = "bounds";

/// An input's bounds, with the constraint as written.
struct Bounds<'m> {
    folded: &'m FoldedConstraint,
    written: &'m Expr,
    ty: Option<&'m Type>,
}

/// Judges every comparison, `match` and `given` of the module against its inputs' bounds.
pub fn check(module: &Module, state: &State, globals: &Globals, out: &mut ModuleAnalysis) {
    let mut bounds = BTreeMap::new();
    for item in &module.file.items {
        let ItemKind::Input(input) = &item.kind else {
            continue;
        };
        let (Some(written), Some(folded)) = (
            &input.constraint,
            state
                .constraints
                .get(&(module.id.0, input.name.text.clone())),
        ) else {
            continue;
        };
        let ty = state
            .symbols
            .iter()
            .find(|symbol| {
                symbol.kind == SymbolKind::Input
                    && symbol.module == Some(module.id)
                    && symbol.name == input.name.text
            })
            .and_then(|symbol| symbol.ty.as_ref());
        if matches!(ty, Some(Type::List(_))) {
            continue;
        }
        bounds.insert(
            input.name.text.clone(),
            Bounds {
                folded,
                written,
                ty,
            },
        );
    }
    if bounds.is_empty() {
        return;
    }
    let reads: BTreeMap<u32, String> = state
        .references
        .iter()
        .filter(|reference| reference.module == module.id)
        .filter_map(|reference| {
            let symbol = state
                .symbols
                .get(usize::try_from(reference.symbol.0).ok()?)?;
            (symbol.kind == SymbolKind::Input
                && symbol.module == Some(module.id)
                && bounds.contains_key(&symbol.name))
            .then(|| (reference.span.start, symbol.name.clone()))
        })
        .collect();
    let mut judge = Judge {
        source: &module.source,
        bounds: &bounds,
        reads: &reads,
        state,
        globals,
        out: &*out,
        found: Vec::new(),
    };
    walk::nodes(&module.file, &mut judge);
    let found = judge.found;
    out.diagnostics.extend(found);
}

/// The walk's state: what it judges against, and what it found.
struct Judge<'a, 'm> {
    source: &'a str,
    bounds: &'a BTreeMap<String, Bounds<'m>>,
    reads: &'a BTreeMap<u32, String>,
    state: &'a State,
    globals: &'a Globals,
    out: &'a ModuleAnalysis,
    found: Vec<Diagnostic>,
}

impl Visit for Judge<'_, '_> {
    fn expr(&mut self, expr: &Expr) {
        match &expr.kind {
            ExprKind::Binary {
                op, left, right, ..
            } => self.comparison(expr.span, *op, left, right),
            ExprKind::Match { scrutinee, arms } => {
                self.arms(scrutinee, arms.iter().map(|arm| &arm.pattern));
            }
            _ => {}
        }
    }

    fn stmt(&mut self, stmt: &Stmt) {
        match &stmt.kind {
            StmtKind::Match(statement) => self.arms(
                &statement.scrutinee,
                statement.arms.iter().map(|arm| &arm.pattern),
            ),
            StmtKind::Given { target, value, .. } => {
                if let [root, name] = target.as_slice()
                    && root.text == "input"
                {
                    self.given(&name.text, value);
                }
            }
            _ => {}
        }
    }
}

/// What the bounds make of a comparison.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Verdict {
    Never,
    Always,
}

impl Judge<'_, '_> {
    /// The bounded input an expression reads by its own name.
    fn input(&self, expr: &Expr) -> Option<(&str, &Bounds<'_>)> {
        let ExprKind::Name(name) = &expr.kind else {
            return None;
        };
        let input = self.reads.get(&name.span.start)?;
        Some((input.as_str(), self.bounds.get(input)?))
    }

    /// The compile-time value of an expression, when it has one.
    fn fold(&self, expr: &Expr) -> Option<Value> {
        let lookup = |node: &Expr| -> Option<Result<Value, FoldError>> {
            constant_of(self.globals, &self.state.values, node)
        };
        let type_of = |id| self.out.type_of(id).cloned();
        Folder {
            constant: &lookup,
            type_of: &type_of,
        }
        .fold(expr)
        .ok()
    }

    fn comparison(&mut self, span: Span, op: BinaryOp, left: &Expr, right: &Expr) {
        let (input, bounds, other, op) = if let Some((input, bounds)) = self.input(left) {
            (input, bounds, right, op)
        } else if let Some((input, bounds)) = self.input(right) {
            let Some(flipped) = flip(op) else {
                return;
            };
            (input, bounds, left, flipped)
        } else {
            return;
        };
        let verdict = match op {
            BinaryOp::In | BinaryOp::NotIn => {
                if let ExprKind::Range {
                    start,
                    end,
                    inclusive,
                } = &other.kind
                {
                    let Some((start, end)) = self.ends(start.as_deref(), end.as_deref()) else {
                        return;
                    };
                    in_range(
                        bounds.folded,
                        (start.as_ref(), end.as_ref(), *inclusive),
                        op == BinaryOp::In,
                    )
                } else {
                    let Some(Value::List(values)) = self.fold(other) else {
                        return;
                    };
                    membership(bounds.folded, &values, op == BinaryOp::In)
                }
            }
            BinaryOp::Eq
            | BinaryOp::Ne
            | BinaryOp::Lt
            | BinaryOp::Le
            | BinaryOp::Gt
            | BinaryOp::Ge => {
                let Some(value) = self.fold(other) else {
                    return;
                };
                ordered(bounds.folded, op, &value)
            }
            _ => return,
        };
        let Some(verdict) = verdict else {
            return;
        };
        let reason = self.reason(input, bounds);
        let message = match verdict {
            Verdict::Never => message!("This is never true: {0}"),
            Verdict::Always => message!("This is always true: {0}"),
        };
        let suggestion = matches!(op, BinaryOp::Eq | BinaryOp::Ne)
            .then(|| self.suggestion(bounds, other))
            .flatten();
        self.found.push(with_suggestion(
            span,
            Message::with(message, [reason]),
            suggestion,
        ));
    }

    /// A range's ends, folded (`None` for an end left open), or `None` when an end has no
    /// compile-time value.
    fn ends(
        &self,
        start: Option<&Expr>,
        end: Option<&Expr>,
    ) -> Option<(Option<Value>, Option<Value>)> {
        let start = match start {
            Some(start) => Some(self.fold(start)?),
            None => None,
        };
        let end = match end {
            Some(end) => Some(self.fold(end)?),
            None => None,
        };
        Some((start, end))
    }

    fn arms<'p>(&mut self, scrutinee: &Expr, patterns: impl Iterator<Item = &'p Pattern>) {
        let Some((input, bounds)) = self.input(scrutinee) else {
            return;
        };
        let mut found = Vec::new();
        for pattern in patterns {
            if self.never_matches(bounds, pattern) != Some(true) {
                continue;
            }
            let reason = self.reason(input, bounds);
            let suggestion = match &pattern.kind {
                PatternKind::Literal(literal) => self.suggestion(bounds, literal),
                PatternKind::Variant { variant, .. } => {
                    self.variant_suggestion(bounds, variant.span)
                }
                _ => None,
            };
            found.push(with_suggestion(
                pattern.span,
                Message::with(message!("This arm is never matched: {0}"), [reason]),
                suggestion,
            ));
        }
        self.found.extend(found);
    }

    /// Whether the bounds rule a pattern out: `None` when that can't be told.
    fn never_matches(&self, bounds: &Bounds<'_>, pattern: &Pattern) -> Option<bool> {
        match &pattern.kind {
            PatternKind::Literal(literal) => {
                let value = self.fold(literal)?;
                Some(!admits(bounds.folded, &value)?)
            }
            PatternKind::Variant { variant, .. } => match bounds.folded {
                FoldedConstraint::OneOf(values) => Some(
                    !values
                        .iter()
                        .any(|value| matches!(value, Value::Enum(name) if *name == variant.text)),
                ),
                FoldedConstraint::Range { .. } => {
                    Some(!admits(bounds.folded, &Value::Enum(variant.text.clone()))?)
                }
            },
            PatternKind::Range {
                start,
                end,
                inclusive,
            } => {
                let start = match start {
                    Some(start) => Some(self.fold(start)?),
                    None => None,
                };
                let end = match end {
                    Some(end) => Some(self.fold(end)?),
                    None => None,
                };
                disjoint(bounds.folded, start.as_ref(), end.as_ref(), *inclusive)
            }
            PatternKind::Or(alternatives) => {
                let mut every = true;
                for alternative in alternatives {
                    every &= self.never_matches(bounds, alternative)?;
                }
                Some(every)
            }
            PatternKind::Wildcard
            | PatternKind::None
            | PatternKind::Binding(_)
            | PatternKind::Record(_)
            | PatternKind::Missing => None,
        }
    }

    fn given(&mut self, input: &str, value: &Expr) {
        let Some(bounds) = self.bounds.get(input) else {
            return;
        };
        let (Some(folded), Some(ty)) = (self.fold(value), bounds.ty) else {
            return;
        };
        let item = ty.unwrapped();
        let Some(run) = super::constraints::run_value(&folded, item) else {
            return;
        };
        let constraint = bounds.folded.to_wire();
        let Some(breach) = crate::tariff::constraint::breach(&constraint, &run) else {
            return;
        };
        let mut diagnostic = Diagnostic::error(
            "constraint",
            value.span,
            Message::with(
                message!("{0}, so no contract can set it and this test proves nothing"),
                [Arg::from(breach.message(&constraint, input))],
            ),
        );
        if let Some((text, span)) = self.suggestion(bounds, value) {
            diagnostic = diagnostic.with_fix(replace(
                Message::with(message!("Change to {0}"), [quote(&text)]),
                span,
                text,
            ));
        }
        self.found.push(diagnostic);
    }

    /// Why the bounds decide: "`service` is only ever one of `[…]`", or "`rate` is always
    /// within `…`".
    fn reason(&self, input: &str, bounds: &Bounds<'_>) -> Arg {
        let written = quote_span(self.source, bounds.written.span);
        let message = match bounds.folded {
            FoldedConstraint::OneOf(_) => message!("{0} is only ever one of {1}"),
            FoldedConstraint::Range { .. } => message!("{0} is always within {1}"),
        };
        Arg::from(Message::with(message, [quote(input), written]))
    }

    /// The listed value nearest a literal outside a list constraint, as the constraint writes
    /// it, with the literal's span.
    fn suggestion(&self, bounds: &Bounds<'_>, literal: &Expr) -> Option<(String, Span)> {
        let ExprKind::List(items) = &bounds.written.kind else {
            return None;
        };
        let value = self.fold(literal)?;
        if admits(bounds.folded, &value)? {
            return None;
        }
        let written = match &value {
            Value::Text(text) => text.clone(),
            Value::Enum(name) => name.clone(),
            _ => return None,
        };
        let options: Vec<(String, &Expr)> = items
            .iter()
            .filter_map(|item| match self.fold(item)? {
                Value::Text(text) | Value::Enum(text) => Some((text, item)),
                _ => None,
            })
            .collect();
        let near = nearest(&written, options.iter().map(|(text, _)| text.as_str()))?;
        let (_, item) = options.iter().find(|(text, _)| text == near)?;
        Some((snippet(self.source, item.span).to_owned(), literal.span))
    }

    /// The listed variant nearest a variant pattern the list leaves out.
    fn variant_suggestion(&self, bounds: &Bounds<'_>, span: Span) -> Option<(String, Span)> {
        let FoldedConstraint::OneOf(values) = bounds.folded else {
            return None;
        };
        let written = snippet(self.source, span);
        let names: Vec<&str> = values
            .iter()
            .filter_map(|value| match value {
                Value::Enum(name) => Some(name.as_str()),
                _ => None,
            })
            .collect();
        nearest(written, names).map(|near| (near.to_owned(), span))
    }
}

/// A finding, with its "did you mean" fix when there is one.
fn with_suggestion(span: Span, message: Message, suggestion: Option<(String, Span)>) -> Diagnostic {
    match suggestion {
        Some((text, at)) => Diagnostic::warning(
            CODE,
            span,
            Message::with(
                message!("{0}. Did you mean {1}?"),
                [Arg::from(message), quote(&text)],
            ),
        )
        .with_fix(replace(
            Message::with(message!("Change to {0}"), [quote(&text)]),
            at,
            text,
        )),
        None => Diagnostic::warning(CODE, span, message),
    }
}

/// The comparison seen from the other side: `5 < x` is `x > 5`.
fn flip(op: BinaryOp) -> Option<BinaryOp> {
    Some(match op {
        BinaryOp::Eq => BinaryOp::Eq,
        BinaryOp::Ne => BinaryOp::Ne,
        BinaryOp::Lt => BinaryOp::Gt,
        BinaryOp::Le => BinaryOp::Ge,
        BinaryOp::Gt => BinaryOp::Lt,
        BinaryOp::Ge => BinaryOp::Le,
        _ => return None,
    })
}

/// Whether two values are equal: by order where they have one (`$2.35` is `$2.350`), else by
/// what they are (text, variants, `true`).
fn equal(a: &Value, b: &Value) -> bool {
    compare(a, b).map_or_else(|| a == b, |order| order == Ordering::Equal)
}

/// Whether the constraint admits a value: `None` when the two can't be compared.
fn admits(folded: &FoldedConstraint, value: &Value) -> Option<bool> {
    match folded {
        FoldedConstraint::OneOf(values) => Some(values.iter().any(|listed| equal(listed, value))),
        FoldedConstraint::Range {
            min,
            max,
            max_inclusive,
        } => {
            if let Some(min) = min
                && compare(value, min)? == Ordering::Less
            {
                return Some(false);
            }
            if let Some(max) = max {
                let order = compare(value, max)?;
                if order == Ordering::Greater || (order == Ordering::Equal && !max_inclusive) {
                    return Some(false);
                }
            }
            Some(true)
        }
    }
}

/// Whether `x op value` holds for one value of the input.
fn holds(op: BinaryOp, x: &Value, value: &Value) -> Option<bool> {
    match op {
        BinaryOp::Eq => return Some(equal(x, value)),
        BinaryOp::Ne => return Some(!equal(x, value)),
        _ => {}
    }
    let order = compare(x, value)?;
    Some(match op {
        BinaryOp::Lt => order == Ordering::Less,
        BinaryOp::Le => order != Ordering::Greater,
        BinaryOp::Gt => order == Ordering::Greater,
        BinaryOp::Ge => order != Ordering::Less,
        _ => return None,
    })
}

/// What the bounds make of `x op value`: `None` when the answer depends on the value.
fn ordered(folded: &FoldedConstraint, op: BinaryOp, value: &Value) -> Option<Verdict> {
    match folded {
        FoldedConstraint::OneOf(values) => {
            let mut every = true;
            let mut none = true;
            for listed in values {
                let holds = holds(op, listed, value)?;
                every &= holds;
                none &= !holds;
            }
            verdict(every, none)
        }
        FoldedConstraint::Range {
            min,
            max,
            max_inclusive,
        } => {
            let at_min = match min {
                Some(min) => Some(compare(min, value)?),
                None => None,
            };
            let at_max = match max {
                Some(max) => Some(compare(max, value)?),
                None => None,
            };
            let below = |order: Option<Ordering>, strict: bool| {
                order.is_some_and(|order| {
                    order == Ordering::Less || (!strict && order == Ordering::Equal)
                })
            };
            let above = |order: Option<Ordering>, strict: bool| {
                order.is_some_and(|order| {
                    order == Ordering::Greater || (!strict && order == Ordering::Equal)
                })
            };
            let (always, never) = match op {
                BinaryOp::Gt => (above(at_min, true), below(at_max, false)),
                BinaryOp::Ge => (above(at_min, false), below(at_max, *max_inclusive)),
                BinaryOp::Lt => (below(at_max, *max_inclusive), above(at_min, false)),
                BinaryOp::Le => (below(at_max, false), above(at_min, true)),
                BinaryOp::Eq => (false, !admits(folded, value)?),
                BinaryOp::Ne => (!admits(folded, value)?, false),
                _ => return None,
            };
            verdict(always, never)
        }
    }
}

/// What the bounds make of `x in values` (`not in` when `inside` is false).
fn membership(folded: &FoldedConstraint, values: &[Value], inside: bool) -> Option<Verdict> {
    let mut any = false;
    for value in values {
        any |= admits(folded, value)?;
    }
    let every = match folded {
        FoldedConstraint::OneOf(listed) => listed
            .iter()
            .all(|one| values.iter().any(|value| equal(one, value))),
        FoldedConstraint::Range { .. } => false,
    };
    if inside {
        verdict(every, !any)
    } else {
        verdict(!any, every)
    }
}

/// What the bounds make of `x in start..end` (`not in` when `inside` is false): never when the
/// range misses every value the bounds admit, always when it holds all of them.
fn in_range(
    folded: &FoldedConstraint,
    (start, end, inclusive): (Option<&Value>, Option<&Value>, bool),
    inside: bool,
) -> Option<Verdict> {
    let never = disjoint(folded, start, end, inclusive)?;
    let every = contained(folded, start, end, inclusive)?;
    if inside {
        verdict(every, never)
    } else {
        verdict(never, every)
    }
}

/// Whether every value the bounds admit lies in `start..end`: `None` when that can't be told.
fn contained(
    folded: &FoldedConstraint,
    start: Option<&Value>,
    end: Option<&Value>,
    inclusive: bool,
) -> Option<bool> {
    let within = |value: &Value| -> Option<bool> {
        if let Some(start) = start
            && compare(value, start)? == Ordering::Less
        {
            return Some(false);
        }
        if let Some(end) = end {
            let order = compare(value, end)?;
            if order == Ordering::Greater || (order == Ordering::Equal && !inclusive) {
                return Some(false);
            }
        }
        Some(true)
    };
    match folded {
        FoldedConstraint::OneOf(values) => {
            let mut every = true;
            for value in values {
                every &= within(value)?;
            }
            Some(every)
        }
        FoldedConstraint::Range {
            min,
            max,
            max_inclusive,
        } => {
            let low = match (start, min) {
                (None, _) => true,
                (Some(_), None) => false,
                (Some(start), Some(min)) => compare(start, min)? != Ordering::Greater,
            };
            let high = match (end, max) {
                (None, _) => true,
                (Some(_), None) => false,
                (Some(end), Some(max)) => match compare(max, end)? {
                    Ordering::Less => true,
                    Ordering::Equal => inclusive || !max_inclusive,
                    Ordering::Greater => false,
                },
            };
            Some(low && high)
        }
    }
}

/// Whether a range pattern misses every value the bounds admit: `None` when that can't be told.
fn disjoint(
    folded: &FoldedConstraint,
    start: Option<&Value>,
    end: Option<&Value>,
    inclusive: bool,
) -> Option<bool> {
    let inside = |value: &Value| -> Option<bool> {
        if let Some(start) = start
            && compare(value, start)? == Ordering::Less
        {
            return Some(false);
        }
        if let Some(end) = end {
            let order = compare(value, end)?;
            if order == Ordering::Greater || (order == Ordering::Equal && !inclusive) {
                return Some(false);
            }
        }
        Some(true)
    };
    match folded {
        FoldedConstraint::OneOf(values) => {
            let mut none = true;
            for value in values {
                none &= !inside(value)?;
            }
            Some(none)
        }
        FoldedConstraint::Range {
            min,
            max,
            max_inclusive,
        } => {
            if let (Some(end), Some(min)) = (end, min) {
                let order = compare(end, min)?;
                if order == Ordering::Less || (order == Ordering::Equal && !inclusive) {
                    return Some(true);
                }
            }
            if let (Some(start), Some(max)) = (start, max) {
                let order = compare(start, max)?;
                if order == Ordering::Greater || (order == Ordering::Equal && !max_inclusive) {
                    return Some(true);
                }
            }
            Some(false)
        }
    }
}

fn verdict(always: bool, never: bool) -> Option<Verdict> {
    match (always, never) {
        (true, false) => Some(Verdict::Always),
        (false, true) => Some(Verdict::Never),
        _ => None,
    }
}
