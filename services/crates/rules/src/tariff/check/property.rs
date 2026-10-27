//! Properties: a `property` block's `vary`s, `assume`s and monotone `expect`s.
//!
//! A property body is checked like a test body ([`Body::Test`]): its `given`s, `expect`s and
//! other statements follow the test rules. This module adds the property's own:
//!
//! - **Order.** `vary`, `given` and `assume` set up each case before the tariff rates, so they
//!   come before the property's other statements (`given-order`).
//! - **Targets.** `vary target` takes a record path or `input.name`, resolved exactly as a
//!   `given`'s, or a new `camelCase` name bound for the rest of the property with the type of
//!   its values; the name may not shadow anything (`shadow`). A library has no record or
//!   inputs, so it varies names only (`vary-target`). Each target is varied once, and a varied
//!   path is not also given (`duplicate`, `vary-target`, overlapping paths included).
//! - **Values.** A range of an ordered type with both ends written (`range`, `unordered`),
//!   each a compile-time value (`not-constant`) of the target's type (its inner type for an
//!   optional target), not empty (`empty-range`); or a non-empty list literal of the target's
//!   type, whose items may use the test constructors (`vary-values`). `in` may be left out for
//!   a `Bool`, an enum (both also `none` when optional) and an input with a constraint, which
//!   a range must close at both ends (`vary-values`). An explicit range or list item for an
//!   input lies inside its constraint (`vary-range`, naming the end that sticks out). The
//!   values are worked out once, before any case, and a `given` sets one value for every case,
//!   so neither reads a varied name (`vary-read`).
//! - **`assume cond`** is `Bool`; it reads the record, the inputs (by name), the varied names,
//!   functions and constants, but never the bill: `total`, `lines`, `charge(…)` and
//!   `chosen(…)` are refused (`assume-bill`).
//! - **`expect value nondecreasing in target`**: `value` is ordered and not optional
//!   (`unordered`, with a `?? zero` fix), and `target` is varied over a range in this property
//!   (`monotone-target`).
//! - **At least one `vary`** per property (`vary-missing`); a varied name nothing reads is
//!   `unused`. Properties live in tariffs and libraries; one in a condition is `misplaced`.

use super::checker::{Billed, Body, Checker, Globals, Want};
use super::consts::{self, FoldError, Folder, Value as Folded};
use super::decls::{
    Declarer, FoldedConstraint, State, constant_of, constraint_item_type, fold_diagnostic,
};
use super::describe::{quote, quote_span, words, zero};
use super::fixes::replace;
use super::scope::LocalKind;
use super::{HintKind, ModuleAnalysis};
use crate::tariff::constraint::admits;
use crate::tariff::syntax::ast::{
    Block, Expr, ExprKind, FileKind, ItemKind, Name, Property, StmtKind,
};
use crate::tariff::types::Type;
use std::cmp::Ordering;
use std::collections::{BTreeMap, BTreeSet};
use trenova_module_kit::diagnostic::Diagnostic;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

/// An input's folded constraint and where its parts are written, for the messages that quote
/// them.
#[derive(Debug, Clone)]
struct Bounds {
    folded: FoldedConstraint,
    whole: Span,
    start: Option<Span>,
    end: Option<Span>,
}

/// What the checker knows about the property whose body it is checking.
#[derive(Debug, Default)]
pub struct PropertyScope {
    constraints: BTreeMap<String, Bounds>,
    values: BTreeMap<(u32, String), Result<Folded, FoldError>>,
    varied: Vec<Varied>,
    given: Vec<(Vec<String>, Span)>,
    names: Vec<(String, usize)>,
    assuming: bool,
}

/// One `vary` checked so far.
#[derive(Debug, Clone)]
struct Varied {
    path: Vec<String>,
    span: Span,
    range: bool,
}

/// Checks every property of a module; returns the inputs and imported names they read.
pub fn properties(
    declarer: &Declarer<'_>,
    state: &mut State,
    globals: &Globals,
    out: &mut ModuleAnalysis,
    billed: &Billed,
    file_lets: &[String],
) -> BTreeSet<String> {
    let module = declarer.module;
    let kind = module
        .file
        .header
        .as_ref()
        .map_or(FileKind::Tariff, |header| header.kind);
    let mut used = BTreeSet::new();
    for item in &module.file.items {
        let ItemKind::Property(property) = &item.kind else {
            continue;
        };
        if kind == FileKind::Condition {
            out.diagnostics.push(Diagnostic::error(
                "misplaced",
                property.name.span,
                Message::new(message!(
                    "A property checks a rule over many ratings, so it goes in a tariff or a library; check a condition with `test` blocks"
                )),
            ));
            continue;
        }
        let scope = PropertyScope {
            constraints: constraints(declarer, state),
            values: state.values.clone(),
            ..PropertyScope::default()
        };
        let ((), _) = declarer.run(state, globals, out, Body::Test, |checker| {
            checker.billed.clone_from(billed);
            checker.file_lets = file_lets.to_vec();
            checker.property = Some(Box::new(scope));
            checker.scopes.push_barrier();
            checker.setup_order(&property.body);
            checker.block(&property.body);
            checker.close_property(property);
            used.extend(std::mem::take(&mut checker.used));
        });
    }
    used
}

/// The module's folded input constraints, with where each is written.
fn constraints(declarer: &Declarer<'_>, state: &State) -> BTreeMap<String, Bounds> {
    let module = declarer.module;
    let mut out = BTreeMap::new();
    for item in &module.file.items {
        let ItemKind::Input(input) = &item.kind else {
            continue;
        };
        let (Some(expr), Some(folded)) = (
            &input.constraint,
            state
                .constraints
                .get(&(module.id.0, input.name.text.clone())),
        ) else {
            continue;
        };
        let (start, end) = match &expr.kind {
            ExprKind::Range { start, end, .. } => (
                start.as_ref().map(|part| part.span),
                end.as_ref().map(|part| part.span),
            ),
            _ => (None, None),
        };
        out.insert(
            input.name.text.clone(),
            Bounds {
                folded: folded.clone(),
                whole: expr.span,
                start,
                end,
            },
        );
    }
    out
}

/// What a `vary`'s values turned out to be.
struct Values {
    /// The type of one value: the target's (its inner type for a range over an optional), or
    /// the values' own for a new name.
    ty: Type,
    /// Whether they are a range.
    range: bool,
}

/// Where a `vary`'s values come from, for the checks that differ by target.
struct Target<'t> {
    /// The target as written: `shipment.weight`, `input.rate`, `w`.
    display: String,
    /// The target's declared type; `None` for a new name.
    declared: Option<Type>,
    /// The input's name, for `input.x`.
    input: Option<&'t str>,
    /// Where the target is written.
    span: Span,
}

impl Checker<'_> {
    /// Whether the code being checked reads inputs by name: a file's top level, and a
    /// property's `assume`.
    #[must_use]
    pub fn sees_inputs(&self) -> bool {
        matches!(self.body, Body::Top) || self.property.as_ref().is_some_and(|p| p.assuming)
    }

    /// Reports a `vary`, `given` or `assume` written after the property's other statements.
    fn setup_order(&mut self, body: &Block) {
        let mut first: Option<Span> = None;
        let mut expected = false;
        for stmt in &body.stmts {
            let word = match &stmt.kind {
                StmtKind::Vary { .. } => "vary",
                StmtKind::Given { .. } => "given",
                StmtKind::Assume(_) => "assume",
                other => {
                    first.get_or_insert(stmt.span);
                    expected |= matches!(other, StmtKind::Expect(_));
                    continue;
                }
            };
            let Some(first) = first else {
                continue;
            };
            if word == "given" && expected {
                continue;
            }
            self.report(
                Diagnostic::error(
                    "given-order",
                    stmt.span,
                    Message::with(
                        message!(
                            "{0} comes before the property's other statements: `vary`, `given` and `assume` set up each case before the tariff rates"
                        ),
                        [quote(word)],
                    ),
                )
                .with_related(first, Message::new(message!("The first other statement"))),
            );
        }
    }

    /// The end of a property's check: it needs a `vary`, and every varied name is read.
    fn close_property(&mut self, property: &Property) {
        let Some(scope) = self.property.take() else {
            return;
        };
        if scope.varied.is_empty() {
            self.report(Diagnostic::error(
                "vary-missing",
                property.name.span,
                Message::new(message!(
                    "A property needs at least one `vary`: say what changes from case to case, as in `vary shipment.weight in 1 lb..40_000 lb`"
                )),
            ));
        }
        for (name, id) in &scope.names {
            if let Some(local) = self.scopes.get(*id)
                && local.reads == 0
            {
                let span = local.span;
                self.report(Diagnostic::warning(
                    "unused",
                    span,
                    Message::with(message!("{0} is never used"), [quote(name)]),
                ));
            }
        }
    }

    /// `vary target in values`.
    pub(super) fn vary(&mut self, target: &[Name], values: Option<&Expr>) {
        let (Some(head), Some(last), true) =
            (target.first(), target.last(), self.property.is_some())
        else {
            self.skip_values(values);
            return;
        };
        if head.text.is_empty() {
            self.skip_values(values);
            return;
        }
        let path: Vec<String> = target.iter().map(|part| part.text.clone()).collect();
        let span = head.span.cover(last.span);
        let new_name = target.len() == 1
            && head.text != "input"
            && head.text != "tables"
            && self.schema.root(&head.text).is_none();
        if head.text == "tables" {
            self.report(Diagnostic::error(
                "vary-target",
                span,
                Message::new(message!(
                    "A rate table can't vary: give the cells a property reads, as in `given tables.Zones[\"A\", \"B\"] = $20/cwt`, and vary the record or inputs that look them up"
                )),
            ));
            self.skip_values(values);
            return;
        }
        if !new_name && self.kind == FileKind::Library {
            self.report(Diagnostic::error(
                "vary-target",
                span,
                Message::new(message!(
                    "A library has no record or inputs to vary: vary a new name, as in `vary x in 1..10`, and pass it to the functions under test"
                )),
            ));
            self.skip_values(values);
            return;
        }
        let range = values.is_some_and(|values| matches!(values.kind, ExprKind::Range { .. }));
        let declared = if new_name {
            None
        } else if let Some(ty) = self.given_target(target) {
            Some(ty)
        } else {
            self.skip_values(values);
            self.varied(path, span, range);
            return;
        };
        self.vary_conflicts(&path, span);
        let input = (head.text == "input")
            .then(|| target.get(1).map(|name| name.text.as_str()))
            .flatten();
        let target_info = Target {
            display: path.join("."),
            declared,
            input,
            span,
        };
        let found = self.vary_values(&target_info, values);
        if new_name {
            let ty = found.as_ref().map_or(Type::Error, |found| found.ty.clone());
            self.vary_name(head, ty);
        }
        let range = found.map_or(range, |found| found.range);
        self.varied(path, span, range);
    }

    /// Records a varied target, so later statements check against it.
    fn varied(&mut self, path: Vec<String>, span: Span, range: bool) {
        if let Some(scope) = self.property.as_mut() {
            scope.varied.push(Varied { path, span, range });
        }
    }

    /// Checks a `vary`'s values when its target is not usable: a range's ends, or the
    /// expression.
    fn skip_values(&mut self, values: Option<&Expr>) {
        let Some(values) = values else {
            return;
        };
        if let ExprKind::Range { start, end, .. } = &values.kind {
            self.record(values, Type::Error);
            for part in [start, end].into_iter().flatten() {
                self.expr(part, None);
            }
        } else {
            self.expr(values, None);
        }
    }

    /// A new name a `vary` binds: it may not shadow an input or the bill's `total`, `lines`
    /// and `reviews`, besides what any binding may not shadow.
    fn vary_name(&mut self, name: &Name, ty: Type) {
        let taken = if matches!(
            self.globals.values.get(&name.text),
            Some(super::checker::Global::Input { .. })
        ) {
            Some(Message::new(message!("an input")))
        } else if self.kind == FileKind::Tariff
            && super::catalog::TEST_NAMES.contains(&name.text.as_str())
        {
            Some(Message::new(message!(
                "what a property reads from the bill"
            )))
        } else {
            None
        };
        if let Some(what) = taken {
            self.report(Diagnostic::error(
                "shadow",
                name.span,
                Message::with(
                    message!("{0} is already {1}; pick another name"),
                    [quote(&name.text), Arg::from(what)],
                ),
            ));
        }
        self.hint(name.span, &ty, HintKind::Binding);
        if let Some(id) = self.declare(name, LocalKind::Vary, ty)
            && let Some(scope) = self.property.as_mut()
        {
            scope.names.push((name.text.clone(), id));
        }
    }

    /// Reports a path varied twice, or varied and given, overlapping paths included.
    fn vary_conflicts(&mut self, path: &[String], span: Span) {
        let Some(scope) = self.property.as_ref() else {
            return;
        };
        let varied = scope
            .varied
            .iter()
            .find(|other| overlaps(&other.path, path))
            .map(|other| (other.path.join("."), other.span));
        let given = scope
            .given
            .iter()
            .find(|(other, _)| overlaps(other, path))
            .map(|(other, span)| (other.join("."), *span));
        let shown = path.join(".");
        if let Some((other, at)) = varied {
            let message = if other == shown {
                Message::with(
                    message!("{0} is varied twice: each value is varied once per property"),
                    [quote(&shown)],
                )
            } else {
                Message::with(
                    message!("{0} and {1} overlap: vary one of them"),
                    [quote(&shown), quote(&other)],
                )
            };
            self.report(
                Diagnostic::error("duplicate", span, message)
                    .with_related(at, Message::new(message!("It is varied here"))),
            );
        }
        if let Some((other, at)) = given {
            self.report(
                Diagnostic::error(
                    "vary-target",
                    span,
                    Message::with(
                        message!(
                            "{0} is set by a `given` in this property, so it can't also vary: remove one of them"
                        ),
                        [quote(&other)],
                    ),
                )
                .with_related(at, Message::new(message!("It is given here"))),
            );
        }
    }

    /// A `given` in a property: it may not set a varied path, nor read a varied name.
    pub(super) fn property_given(&mut self, target: &[Name], keys: Option<&[Expr]>, value: &Expr) {
        let Some(scope) = self.property.as_ref() else {
            return;
        };
        let (Some(head), Some(last)) = (target.first(), target.last()) else {
            return;
        };
        let path: Vec<String> = target.iter().map(|part| part.text.clone()).collect();
        let span = head.span.cover(last.span);
        let varied = scope
            .varied
            .iter()
            .find(|other| overlaps(&other.path, &path))
            .map(|other| (other.path.join("."), other.span));
        if let Some((other, at)) = varied {
            self.report(
                Diagnostic::error(
                    "vary-target",
                    span,
                    Message::with(
                        message!(
                            "{0} varies in this property, so a `given` can't also set it: remove one of them"
                        ),
                        [quote(&other)],
                    ),
                )
                .with_related(at, Message::new(message!("It is varied here"))),
            );
        }
        for values in [keys.unwrap_or_default(), std::slice::from_ref(value)] {
            self.varied_reads(
                values,
                message!(
                    "A `given` sets one value for every case, so it can't read {0}, which varies: vary the field itself instead"
                ),
            );
        }
        if let Some(scope) = self.property.as_mut() {
            scope.given.push((path, span));
        }
    }

    /// Reports each read of a varied name in values worked out before any case.
    fn varied_reads(&mut self, values: &[Expr], template: &'static str) {
        let Some(scope) = self.property.as_ref() else {
            return;
        };
        let names: Vec<String> = scope.names.iter().map(|(name, _)| name.clone()).collect();
        let mut found: Vec<(Span, String)> = Vec::new();
        for value in values {
            super::visit::expr(value, &mut |node| {
                if let ExprKind::Name(name) = &node.kind
                    && names.contains(&name.text)
                {
                    found.push((name.span, name.text.clone()));
                }
            });
        }
        for (span, name) in found {
            self.report(Diagnostic::error(
                "vary-read",
                span,
                Message::with(template, [quote(&name)]),
            ));
        }
    }

    /// The values a `vary` takes, checked against its target.
    fn vary_values(&mut self, target: &Target<'_>, values: Option<&Expr>) -> Option<Values> {
        let Some(values) = values else {
            return self.implicit_values(target);
        };
        match &values.kind {
            ExprKind::Range {
                start,
                end,
                inclusive,
            } => {
                self.record(values, Type::Error);
                let (Some(start), Some(end)) = (start, end) else {
                    let inner = target.declared.as_ref().map(|ty| ty.unwrapped().clone());
                    for part in [start, end].into_iter().flatten() {
                        self.expr(part, inner.as_ref());
                    }
                    self.report(Diagnostic::error(
                        "range",
                        values.span,
                        Message::new(message!(
                            "A `vary` range needs both ends, as in `1 lb..40_000 lb`"
                        )),
                    ));
                    return None;
                };
                self.vary_range(target, values, (start, end), *inclusive)
            }
            ExprKind::List(items) => self.vary_list(target, values, items),
            _ => {
                self.expr(values, target.declared.as_ref());
                self.report(Diagnostic::error(
                    "vary-values",
                    values.span,
                    Message::new(message!(
                        "A `vary` takes a range, as in `1 lb..40_000 lb`, or a list of values, as in `[Standard, Expedited]`"
                    )),
                ));
                None
            }
        }
    }

    /// `vary target in lo..hi`.
    fn vary_range(
        &mut self,
        target: &Target<'_>,
        values: &Expr,
        (start, end): (&Expr, &Expr),
        inclusive: bool,
    ) -> Option<Values> {
        let before = self.out.diagnostics.len();
        let item = if let Some(declared) = &target.declared {
            let item = declared.unwrapped().clone();
            if super::operators::ordered(&item) {
                let want = Want::Given {
                    path: target.display.clone(),
                };
                self.expect(start, &item, &want);
                self.expect(end, &item, &want);
            } else {
                self.expr(start, None);
                self.expr(end, None);
            }
            item
        } else {
            let first = self.operand(start);
            let second = self.operand(end);
            if let Some(joined) = first.join(&second) {
                joined
            } else {
                self.mismatch(end, &second, &first, &Want::Value);
                first
            }
        };
        if !super::operators::ordered(&item) {
            let template = if matches!(item, Type::Text) {
                message!(
                    "{0} is {1}, which has no next value to step to, so it can't be varied over a range: list its values instead, as in {2}"
                )
            } else {
                message!(
                    "{0} is {1}, which has no order, so it can't be varied over a range: list its values instead, as in {2}"
                )
            };
            self.report(Diagnostic::error(
                "unordered",
                values.span,
                Message::with(
                    template,
                    [
                        quote(&target.display),
                        Arg::from(words(&item)),
                        quote(&format!("vary {} in [a, b]", target.display)),
                    ],
                ),
            ));
            return None;
        }
        if self.out.diagnostics.len() > before {
            return Some(Values {
                ty: item,
                range: true,
            });
        }
        let (Some(lo), Some(hi)) = (self.range_end(start), self.range_end(end)) else {
            return Some(Values {
                ty: item,
                range: true,
            });
        };
        let order = consts::compare(&lo, &hi);
        if order == Some(Ordering::Greater) || (order == Some(Ordering::Equal) && !inclusive) {
            self.report(Diagnostic::error(
                "empty-range",
                values.span,
                Message::new(message!(
                    "This range is empty: its start isn't below its end"
                )),
            ));
        } else if let Some(input) = target.input {
            self.range_inside(input, values, (start, &lo), (end, &hi), inclusive);
        }
        Some(Values {
            ty: item,
            range: true,
        })
    }

    /// One end of a `vary` range, folded: a compile-time value, or the reason it isn't.
    fn range_end(&mut self, end: &Expr) -> Option<Folded> {
        match self.fold_value(end) {
            Ok(value) => Some(value),
            Err(FoldError::NotConstant(span)) => {
                self.report(Diagnostic::error(
                    "not-constant",
                    span,
                    Message::new(message!(
                        "A `vary` range's ends are worked out before any case: write each as a literal or a constant"
                    )),
                ));
                None
            }
            Err(error) => {
                if let Some(diagnostic) = fold_diagnostic(error) {
                    self.report(diagnostic);
                }
                None
            }
        }
    }

    /// Folds a compile-time value written in the property.
    fn fold_value(&self, expr: &Expr) -> Result<Folded, FoldError> {
        let Some(scope) = self.property.as_ref() else {
            return Err(FoldError::Upstream);
        };
        let globals = self.globals;
        let lookup = |node: &Expr| constant_of(globals, &scope.values, node);
        let type_of = |id| self.recorded(id);
        Folder {
            constant: &lookup,
            type_of: &type_of,
        }
        .fold(expr)
    }

    /// Reports the end of an input's `vary` range that lies outside its constraint.
    fn range_inside(
        &mut self,
        input: &str,
        values: &Expr,
        (start, lo): (&Expr, &Folded),
        (end, hi): (&Expr, &Folded),
        inclusive: bool,
    ) {
        let Some(bounds) = self
            .property
            .as_ref()
            .and_then(|scope| scope.constraints.get(input))
            .cloned()
        else {
            return;
        };
        let (min, max, max_inclusive) = match &bounds.folded {
            FoldedConstraint::OneOf(_) => {
                self.report(Diagnostic::error(
                    "vary-range",
                    values.span,
                    Message::with(
                        message!(
                            "The constraint on {0} lists the values it takes, {1}, so vary it over a list of them, or leave out `in` to take them all"
                        ),
                        [quote(input), quote_span(self.source, bounds.whole)],
                    ),
                ));
                return;
            }
            FoldedConstraint::Range {
                min,
                max,
                max_inclusive,
            } => (min, max, *max_inclusive),
        };
        if let (Some(min), Some(at)) = (min, bounds.start)
            && consts::compare(lo, min) == Some(Ordering::Less)
        {
            self.report(Diagnostic::error(
                "vary-range",
                start.span,
                Message::with(
                    message!(
                        "The range starts at {0}, below {1}, where the constraint on {2} starts"
                    ),
                    [
                        quote_span(self.source, start.span),
                        quote_span(self.source, at),
                        quote(input),
                    ],
                ),
            ));
        }
        if let (Some(max), Some(at)) = (max, bounds.end) {
            let template = match consts::compare(hi, max) {
                Some(Ordering::Greater) => Some(message!(
                    "The range reaches {0}, past {1}, where the constraint on {2} ends"
                )),
                Some(Ordering::Equal) if inclusive && !max_inclusive => Some(message!(
                    "The range reaches {0}, but the constraint on {2} takes only values below {1}"
                )),
                _ => None,
            };
            if let Some(template) = template {
                self.report(Diagnostic::error(
                    "vary-range",
                    end.span,
                    Message::with(
                        template,
                        [
                            quote_span(self.source, end.span),
                            quote_span(self.source, at),
                            quote(input),
                        ],
                    ),
                ));
            }
        }
    }

    /// `vary target in [a, b]`.
    fn vary_list(&mut self, target: &Target<'_>, values: &Expr, items: &[Expr]) -> Option<Values> {
        let Some((first, rest)) = items.split_first() else {
            self.record(values, Type::Error);
            self.report(Diagnostic::error(
                "vary-values",
                values.span,
                Message::new(message!("A `vary` list needs at least one value")),
            ));
            return None;
        };
        let ty = if let Some(declared) = &target.declared {
            let want = Want::Given {
                path: target.display.clone(),
            };
            for item in items {
                self.expect(item, declared, &want);
            }
            declared.clone()
        } else {
            let mut ty = self.expr(first, None);
            if unknown(&ty) {
                self.report(Diagnostic::error(
                    "annotation-needed",
                    first.span,
                    Message::with(
                        message!(
                            "{0} doesn't say what {1} holds: start the list with a value of the type it takes"
                        ),
                        [quote_span(self.source, first.span), quote(&target.display)],
                    ),
                ));
                ty = Type::Error;
            }
            for item in rest {
                let found = self.expr(item, Some(&ty));
                match ty.join(&found) {
                    Some(joined) => ty = joined,
                    None => self.mismatch(item, &found, &ty, &Want::Item),
                }
            }
            ty
        };
        self.record(values, Type::list(ty.clone()));
        self.varied_reads(
            items,
            message!(
                "The values a `vary` takes are worked out once, before any case, so they can't read {0}, which varies"
            ),
        );
        if let (Some(input), Some(declared)) = (target.input, &target.declared) {
            self.items_inside(input, declared, items);
        }
        Some(Values { ty, range: false })
    }

    /// Reports each item of an input's `vary` list that its constraint refuses.
    fn items_inside(&mut self, input: &str, declared: &Type, items: &[Expr]) {
        let Some(bounds) = self
            .property
            .as_ref()
            .and_then(|scope| scope.constraints.get(input))
            .cloned()
        else {
            return;
        };
        for item in items {
            let Ok(value) = self.fold_value(item) else {
                continue;
            };
            let constraint = bounds.folded.to_wire();
            let admitted = crate::tariff::wire::decode(&consts::wire(&value), declared, input)
                .is_ok_and(|value| admits(&constraint, &value));
            if !admitted {
                self.report(Diagnostic::error(
                    "vary-range",
                    item.span,
                    Message::with(
                        message!("{0} is outside the constraint on {1}, {2}"),
                        [
                            quote_span(self.source, item.span),
                            quote(input),
                            quote_span(self.source, bounds.whole),
                        ],
                    ),
                ));
            }
        }
    }

    /// The values of a `vary` written without `in`: both `Bool`s, every variant (and `none`
    /// for an optional), or an input's constraint.
    fn implicit_values(&mut self, target: &Target<'_>) -> Option<Values> {
        let Some(declared) = target.declared.clone() else {
            self.report(Diagnostic::error(
                "vary-values",
                target.span,
                Message::with(
                    message!(
                        "Say which values {0} takes, as in {1}: a new name has no type of its own"
                    ),
                    [
                        quote(&target.display),
                        quote(&format!("vary {} in 1..10", target.display)),
                    ],
                ),
            ));
            return None;
        };
        let bounds = target.input.and_then(|input| {
            self.property
                .as_ref()
                .and_then(|scope| scope.constraints.get(input))
                .cloned()
        });
        if let Some(bounds) = bounds {
            return self.constraint_values(target, &declared, &bounds);
        }
        if matches!(declared.unwrapped(), Type::Bool | Type::Enum(_)) {
            return Some(Values {
                ty: declared,
                range: false,
            });
        }
        self.report(Diagnostic::error(
            "vary-values",
            target.span,
            Message::with(
                message!(
                    "Say which values {0} takes, as in {1}: only a `Bool`, an enum or an input with a constraint can leave out `in`"
                ),
                [
                    quote(&target.display),
                    quote(&format!("vary {} in lo..hi", target.display)),
                ],
            ),
        ));
        None
    }

    /// An input varied over its constraint.
    fn constraint_values(
        &mut self,
        target: &Target<'_>,
        declared: &Type,
        bounds: &Bounds,
    ) -> Option<Values> {
        if matches!(declared, Type::List(_)) {
            self.report(Diagnostic::error(
                "vary-values",
                target.span,
                Message::with(
                    message!(
                        "The constraint on {0} bounds each item of the list, not the list: say which lists it takes, as in {1}"
                    ),
                    [
                        quote(target.input.unwrap_or_default()),
                        quote(&format!("vary {} in [[…], […]]", target.display)),
                    ],
                ),
            ));
            return None;
        }
        match &bounds.folded {
            FoldedConstraint::OneOf(_) => Some(Values {
                ty: declared.clone(),
                range: false,
            }),
            FoldedConstraint::Range { .. }
                if matches!(constraint_item_type(declared), Type::Text) =>
            {
                self.report(
                    Diagnostic::error(
                        "vary-values",
                        target.span,
                        Message::with(
                            message!(
                                "The constraint on {0}, {1}, is a text range, which has no next value to step to, so say which values it takes: {2}"
                            ),
                            [
                                quote(target.input.unwrap_or_default()),
                                quote_span(self.source, bounds.whole),
                                quote(&format!("vary {} in [a, b]", target.display)),
                            ],
                        ),
                    )
                    .with_related(
                        bounds.whole,
                        Message::new(message!("The constraint is written here")),
                    ),
                );
                None
            }
            FoldedConstraint::Range {
                min: Some(_),
                max: Some(_),
                ..
            } => Some(Values {
                ty: constraint_item_type(declared),
                range: true,
            }),
            FoldedConstraint::Range { .. } => {
                self.report(
                    Diagnostic::error(
                        "vary-values",
                        target.span,
                        Message::with(
                            message!(
                                "The constraint on {0}, {1}, is open at one end, so give it a range: {2}"
                            ),
                            [
                                quote(target.input.unwrap_or_default()),
                                quote_span(self.source, bounds.whole),
                                quote(&format!("vary {} in lo..hi", target.display)),
                            ],
                        ),
                    )
                    .with_related(
                        bounds.whole,
                        Message::new(message!("The constraint is written here")),
                    ),
                );
                None
            }
        }
    }

    /// `assume cond`.
    pub(super) fn assume(&mut self, condition: &Expr) {
        if self.kind == FileKind::Tariff && self.property.is_some() {
            self.assume_reads_bill(condition);
        }
        if let Some(scope) = self.property.as_mut() {
            scope.assuming = true;
        }
        self.expect(condition, &Type::Bool, &Want::Condition);
        if let Some(scope) = self.property.as_mut() {
            scope.assuming = false;
        }
    }

    /// Reports each read of the bill in an `assume`.
    fn assume_reads_bill(&mut self, condition: &Expr) {
        let mut found: Vec<Span> = Vec::new();
        super::visit::expr(condition, &mut |node| match &node.kind {
            ExprKind::Name(name)
                if super::catalog::TEST_NAMES.contains(&name.text.as_str())
                    && self.scopes.find(&name.text).is_none()
                    && !self.globals.values.contains_key(&name.text) =>
            {
                found.push(node.span);
            }
            ExprKind::Call { callee, .. } => {
                if let ExprKind::Name(name) = &callee.kind
                    && matches!(name.text.as_str(), "charge" | "chosen")
                    && self.scopes.find(&name.text).is_none()
                    && !self.globals.values.contains_key(&name.text)
                {
                    found.push(node.span);
                }
            }
            _ => {}
        });
        for span in found {
            self.report(Diagnostic::error(
                "assume-bill",
                span,
                Message::with(
                    message!(
                        "`assume` runs before the tariff rates, so it can't read the bill ({0}): narrow the cases by the record, the inputs and the varied values"
                    ),
                    [quote_span(self.source, span)],
                ),
            ));
        }
    }

    /// `expect value nondecreasing in target` (or `nonincreasing`).
    pub(super) fn monotone(&mut self, value: &Expr, target: &[Name]) {
        let ty = self.expr(value, None);
        if self.property.is_none() {
            return;
        }
        self.monotone_value(value, &ty);
        let (Some(head), Some(last)) = (target.first(), target.last()) else {
            return;
        };
        if head.text.is_empty() {
            return;
        }
        let path: Vec<String> = target.iter().map(|part| part.text.clone()).collect();
        let span = head.span.cover(last.span);
        if let [name] = target
            && let Some(id) = self.scopes.find(&name.text)
        {
            self.scopes.read(id);
            if let Some(symbol) = self.scopes.get(id).map(|local| local.symbol) {
                self.reference(name.span, symbol);
            }
        }
        let shown = path.join(".");
        let varied = self
            .property
            .as_ref()
            .and_then(|scope| scope.varied.iter().find(|other| other.path == path))
            .map(|other| other.range);
        let message = match varied {
            Some(true) => return,
            Some(false) => Message::with(
                message!(
                    "{0} is varied over a list, which has no order: vary it over a range to compare along it"
                ),
                [quote(&shown)],
            ),
            None => Message::with(
                message!(
                    "{0} isn't varied in this property: vary it over a range first, as in {1}"
                ),
                [quote(&shown), quote(&format!("vary {shown} in lo..hi"))],
            ),
        };
        self.report(Diagnostic::error("monotone-target", span, message));
    }

    /// A monotone expectation's value has an order and is never empty.
    fn monotone_value(&mut self, value: &Expr, ty: &Type) {
        match ty {
            Type::Error | Type::Never => {}
            Type::Optional(inner) if !matches!(**inner, Type::Never) => {
                let code = self.text(value.span).to_owned();
                let mut diagnostic = Diagnostic::error(
                    "unordered",
                    value.span,
                    Message::with(
                        message!(
                            "{0} may be empty, and an empty value has no place in an order: give it a fallback"
                        ),
                        [quote(&code)],
                    ),
                );
                if let Some(fallback) = zero(inner).filter(|_| super::operators::ordered(inner)) {
                    let written = format!("{code} ?? {fallback}");
                    diagnostic = diagnostic.with_fix(replace(
                        Message::with(message!("Fall back to {0}"), [quote(&written)]),
                        value.span,
                        written,
                    ));
                }
                self.report(diagnostic);
            }
            other if !super::operators::ordered(other) => {
                self.report(Diagnostic::error(
                    "unordered",
                    value.span,
                    Message::with(
                        message!(
                            "{0} is {1}, which has no order, so it can't rise or fall: compare a number, money, a quantity, a percent or a date"
                        ),
                        [quote_span(self.source, value.span), Arg::from(words(other))],
                    ),
                ));
            }
            _ => {}
        }
    }
}

/// Whether one path is the other or lies inside it.
fn overlaps(a: &[String], b: &[String]) -> bool {
    let shorter = a.len().min(b.len());
    a.get(..shorter) == b.get(..shorter)
}

/// Whether a type is one an empty literal leaves unknown: `none`, `[]`.
fn unknown(ty: &Type) -> bool {
    match ty {
        Type::Never => true,
        Type::Optional(inner) | Type::List(inner) => matches!(**inner, Type::Never),
        _ => false,
    }
}
