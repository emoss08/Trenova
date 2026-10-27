//! Types every expression: names, X12 paths, literals, records, members, indexes, operators,
//! crosswalks, lookups, `match` and `if`. Calls are in `calls`.

use trenova_module_kit::diagnostic::{Change, Diagnostic, Fix, nearest};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::env::Env;
use super::x12::{self as nav, Reference};
use super::{Checker, Decl, Program};
use crate::lading::facts::{Target, Typed};
use crate::lading::schema;
use crate::lading::syntax::ast::{
    BinOp, Direction, Else, Expr, ExprKind, FieldInit, MemberKind, Outcome, OutcomeKind, TextPart,
    UnOp,
};
use crate::lading::types::{Dim, NodeRef, Type};
use crate::x12::guide::compiled::NodeKind;

/// What a unit measures.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum UnitKind {
    /// A measure's dimension.
    Measure(Dim),
    /// A length of time.
    Duration,
}

/// What a unit measures, when it is a unit.
#[must_use]
pub fn unit_kind(unit: &str) -> Option<UnitKind> {
    Some(match unit {
        "lb" | "kg" | "oz" | "t" | "ton" => UnitKind::Measure(Dim::Weight),
        "in" | "ft" | "yd" | "mi" | "cm" | "m" | "km" => UnitKind::Measure(Dim::Length),
        "ft3" | "m3" | "gal" | "l" => UnitKind::Measure(Dim::Volume),
        "°F" | "°C" => UnitKind::Measure(Dim::Temperature),
        "s" | "min" | "h" | "d" => UnitKind::Duration,
        _ => return None,
    })
}

pub(crate) fn mismatch(span: Span, expected: &Type, found: &Type) -> Diagnostic {
    Diagnostic::error(
        "type-mismatch",
        span,
        Message::with(
            message!("Expected {0}, found {1}"),
            [
                Arg::from(expected.to_string()),
                Arg::from(found.to_string()),
            ],
        ),
    )
}

impl Checker<'_> {
    /// Types an expression, recording what the editor needs.
    pub(crate) fn expr(&mut self, env: &mut Env, expr: &Expr, expected: Option<&Type>) -> Type {
        let ty = self.expr_inner(env, expr, expected);
        if matches!(
            expr.kind,
            ExprKind::Name(_)
                | ExprKind::Member { .. }
                | ExprKind::Call { .. }
                | ExprKind::X12(_)
                | ExprKind::Digits(_)
                | ExprKind::Index { .. }
                | ExprKind::Via { .. }
                | ExprKind::Lookup { .. }
                | ExprKind::Type(_)
        ) {
            self.facts.types.push(Typed {
                unit: env.unit,
                span: expr.span,
                ty: ty.clone(),
            });
        }
        ty
    }

    /// Types an expression that must fit `expected`, reporting when it does not.
    pub(crate) fn expect_type(&mut self, env: &mut Env, expr: &Expr, expected: &Type) -> Type {
        let ty = self.expr(env, expr, Some(expected));
        if !ty.fits(expected) {
            let mut diagnostic = mismatch(expr.span, expected, &ty);
            if ty.is_optional() && ty.unwrapped().fits(expected) {
                diagnostic = Diagnostic::error(
                    "optional-unhandled",
                    expr.span,
                    Message::with(
                        message!("This may be none, and {0} is expected: give a default with ??"),
                        [Arg::from(expected.to_string())],
                    ),
                );
            }
            self.report(env.unit, diagnostic);
        }
        ty
    }

    /// Types a condition.
    pub(crate) fn condition(&mut self, env: &mut Env, expr: &Expr) {
        let ty = self.expr(env, expr, Some(&Type::Bool));
        if !matches!(ty.unwrapped(), Type::Bool | Type::Any) {
            self.report(env.unit, mismatch(expr.span, &Type::Bool, &ty));
        }
    }

    /// Types an outcome's message and its `at` or `code`.
    pub(crate) fn outcome(&mut self, env: &mut Env, outcome: &Outcome, inbound: bool) {
        let ty = self.expr(env, &outcome.message, Some(&Type::Text));
        if !matches!(ty.unwrapped(), Type::Text | Type::Code(_) | Type::Any) {
            self.report(env.unit, mismatch(outcome.message.span, &Type::Text, &ty));
        }
        match &outcome.kind {
            OutcomeKind::Reject(Some(at)) => {
                self.expr(env, at, None);
                if !matches!(
                    at.kind,
                    ExprKind::Member { .. } | ExprKind::Name(_) | ExprKind::X12(_)
                ) {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "reject-at",
                            at.span,
                            Message::new(message!(
                                "reject at names a field or an element, such as stop.address.postal"
                            )),
                        ),
                    );
                }
            }
            OutcomeKind::Decline(code) => {
                if !inbound {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "decline-outbound",
                            outcome.span,
                            Message::new(message!(
                                "Only a received message can be declined: use hold or reject here"
                            )),
                        ),
                    );
                }
                if let Some(code) = code {
                    self.expect_type(env, code, &Type::Text);
                }
            }
            _ => {}
        }
    }

    /// Types an `else`, returning the type it makes the value.
    pub(crate) fn else_clause(
        &mut self,
        env: &mut Env,
        value: Type,
        otherwise: Option<&Else>,
        inbound: bool,
    ) -> Type {
        match otherwise {
            None => value,
            Some(Else::Value(default)) => {
                let fallback = self.expr(env, default, Some(value.unwrapped()));
                if !value.is_optional() && !value.is_any() {
                    self.report(
                        env.unit,
                        Diagnostic::warning(
                            "else-never-used",
                            default.span,
                            Message::new(message!(
                                "The value is never none, so this else is never used"
                            )),
                        ),
                    );
                }
                value.join(&fallback).map_or_else(
                    || {
                        self.report(
                            env.unit,
                            mismatch(default.span, value.unwrapped(), &fallback),
                        );
                        Type::Any
                    },
                    |joined| {
                        if fallback.is_optional() {
                            joined
                        } else {
                            joined.unwrapped().clone()
                        }
                    },
                )
            }
            Some(Else::Outcome(outcome)) => {
                self.outcome(env, outcome, inbound);
                if matches!(outcome.kind, OutcomeKind::Warn | OutcomeKind::Hold) {
                    value
                } else {
                    value.unwrapped().clone()
                }
            }
        }
    }

    #[allow(clippy::too_many_lines)]
    fn expr_inner(&mut self, env: &mut Env, expr: &Expr, expected: Option<&Type>) -> Type {
        match &expr.kind {
            ExprKind::Error => Type::Any,
            ExprKind::Name(name) => self.name(env, name, expr.span),
            ExprKind::Type(variant) => self.variant(env, variant, expr.span, expected),
            ExprKind::X12(text) => self.x12_root(env, text, expr.span),
            ExprKind::Digits(text) => self.digits(env, text, expr.span),
            ExprKind::Number(text) => {
                if text.contains('.') {
                    Type::Decimal
                } else {
                    Type::Int
                }
            }
            ExprKind::Quantity(_, unit) => match unit_kind(unit) {
                Some(UnitKind::Measure(dim)) => Type::Measure(dim),
                Some(UnitKind::Duration) => Type::Duration,
                None => Type::Any,
            },
            ExprKind::Percent(_) => Type::Percent,
            ExprKind::Money(_) => Type::Money,
            ExprKind::Text(parts) => {
                for part in parts {
                    if let TextPart::Expr(inner) = part {
                        self.expr(env, inner, None);
                    }
                }
                Type::Text
            }
            ExprKind::Raw(_) => Type::Text,
            ExprKind::Date(text) => self.literal_check(env, expr.span, text, Type::Date),
            ExprKind::Time(text) => self.literal_check(env, expr.span, text, Type::Time),
            ExprKind::Instant(text) => self.literal_check(env, expr.span, text, Type::DateTime),
            ExprKind::Bool(_) => Type::Bool,
            ExprKind::None => Type::None,
            ExprKind::List(items) => {
                let item_expected = expected.and_then(Type::item).cloned();
                let mut joined: Option<Type> = None;
                for item in items {
                    let ty = self.expr(env, item, item_expected.as_ref());
                    joined = Some(match joined {
                        None => ty,
                        Some(previous) => previous.join(&ty).unwrap_or_else(|| {
                            self.report(env.unit, mismatch(item.span, &previous, &ty));
                            previous
                        }),
                    });
                }
                joined.or(item_expected).unwrap_or(Type::Any).list()
            }
            ExprKind::Record { ty, fields } => {
                self.record(env, ty.as_ref(), fields, expr.span, expected)
            }
            ExprKind::Member {
                target,
                member,
                kind,
            } => {
                let target_ty = self.expr(env, target, None);
                self.member(env, &target_ty, member.name.as_str(), member.span, *kind)
            }
            ExprKind::Index { target, index } => {
                let target_ty = self.expr(env, target, None);
                self.index(env, &target_ty, index)
            }
            ExprKind::Call { callee, args } => self.call(env, callee, args, expr.span, expected),
            ExprKind::Unary(op, inner) => self.unary(env, *op, inner),
            ExprKind::Binary(op, lhs, rhs) => self.binary(env, *op, lhs, rhs, expr.span),
            ExprKind::Via {
                value,
                crosswalk,
                back,
            } => {
                let value_ty = self.expr(env, value, None);
                self.via(env, &value_ty, value.span, crosswalk, *back)
            }
            ExprKind::Exists(inner) => {
                let ty = self.expr(env, inner, None);
                if !ty.is_optional() && !ty.is_any() && !matches!(ty, Type::List(_)) {
                    self.report(
                        env.unit,
                        Diagnostic::warning(
                            "exists-always",
                            inner.span,
                            Message::new(message!("This is never none, so exists is always true")),
                        ),
                    );
                }
                Type::Bool
            }
            ExprKind::Lookup { kind, code, learn } => {
                let code_ty = self.expr(env, code, Some(&Type::Text));
                if !code_ty.is_text() && !code_ty.is_any() {
                    self.report(env.unit, mismatch(code.span, &Type::Text, &code_ty));
                }
                if let Some(learn) = learn {
                    self.expr(env, learn, None);
                }
                if let Some(found) = schema::lookup(&kind.name) {
                    self.occurrence(
                        env.unit,
                        kind.span,
                        Target::Type(found.name.to_owned()),
                        false,
                        false,
                    );
                    Type::Ref(found.name).optional()
                } else {
                    let mut diagnostic = Diagnostic::error(
                        "unknown-lookup",
                        kind.span,
                        Message::with(
                            message!("{0} is not a lookup kind"),
                            [Arg::from(kind.name.as_str())],
                        ),
                    );
                    if let Some(close) =
                        nearest(&kind.name, schema::LOOKUPS.iter().map(|item| item.name))
                    {
                        diagnostic = diagnostic.with_fix(super::replace_fix(kind.span, close));
                    }
                    self.report(env.unit, diagnostic);
                    Type::Any
                }
            }
            ExprKind::Match {
                value,
                arms,
                otherwise,
            } => {
                let value_ty = self.expr(env, value, None);
                let mut result: Option<Type> = None;
                let mut covered = Vec::new();
                for arm in arms {
                    for pattern in &arm.patterns {
                        let pattern_ty = self.expr(env, pattern, Some(value_ty.unwrapped()));
                        if value_ty.unwrapped().join(&pattern_ty).is_none() {
                            self.report(
                                env.unit,
                                mismatch(pattern.span, value_ty.unwrapped(), &pattern_ty),
                            );
                        }
                        if let ExprKind::Type(name) = &pattern.kind {
                            covered.push(name.clone());
                        }
                    }
                    let ty = self.expr(env, &arm.value, expected);
                    result = Some(self.join_branch(env, result, ty, arm.value.span));
                }
                let exhaustive = match value_ty.unwrapped() {
                    Type::Enum(name) => schema::enumeration(name).is_some_and(|item| {
                        item.variants
                            .iter()
                            .all(|(variant, _)| covered.iter().any(|c| c == variant))
                    }),
                    Type::Bool => true,
                    _ => false,
                };
                match otherwise {
                    Some(otherwise) => {
                        let ty = self.expr(env, otherwise, expected);
                        self.join_branch(env, result, ty, otherwise.span)
                    }
                    None if exhaustive && !value_ty.is_optional() => result.unwrap_or(Type::Any),
                    None => result.unwrap_or(Type::Any).optional(),
                }
            }
            ExprKind::If {
                cond,
                then,
                otherwise,
            } => {
                self.condition(env, cond);
                let then_ty = self.expr(env, then, expected);
                let else_ty = self.expr(env, otherwise, expected);
                self.join_branch(env, Some(then_ty), else_ty, otherwise.span)
            }
            ExprKind::Lambda { .. } => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "lambda-here",
                        expr.span,
                        Message::new(message!(
                            "A function like x => ... is only an argument, as in stops.count(s => s.type == Pickup)"
                        )),
                    ),
                );
                Type::Any
            }
            ExprKind::Each { .. } => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "each-here",
                        expr.span,
                        Message::new(message!(
                            "each builds a list of records for a field: stops <- each 0300 as stop"
                        )),
                    ),
                );
                Type::Any
            }
        }
    }

    fn join_branch(&mut self, env: &Env, previous: Option<Type>, ty: Type, span: Span) -> Type {
        match previous {
            None => ty,
            Some(previous) => previous.join(&ty).unwrap_or_else(|| {
                self.report(env.unit, mismatch(span, &previous, &ty));
                previous
            }),
        }
    }

    fn literal_check(&mut self, env: &Env, span: Span, text: &str, ty: Type) -> Type {
        let valid = match ty {
            Type::Date => crate::lading::values::parse_date(text).is_some(),
            Type::Time => crate::lading::values::parse_time(text).is_some(),
            _ => crate::lading::values::parse_instant(text).is_some(),
        };
        if !valid {
            self.report(
                env.unit,
                Diagnostic::error(
                    "bad-literal",
                    span,
                    Message::with(
                        message!("{0} is not a valid {1}"),
                        [Arg::from(text), Arg::from(ty.to_string())],
                    ),
                ),
            );
        }
        ty
    }

    #[allow(clippy::too_many_lines)]
    fn name(&mut self, env: &mut Env, name: &str, span: Span) -> Type {
        if let Some(local) = env.lookup(name) {
            let ty = local.ty.clone();
            let declared = local.span;
            self.occurrence(
                env.unit,
                span,
                Target::Local {
                    unit: env.unit,
                    span: declared,
                },
                false,
                false,
            );
            return ty;
        }
        let program: &Program = self.program;
        match name {
            "partner" => {
                self.occurrence(
                    env.unit,
                    span,
                    Target::Builtin("partner".into()),
                    false,
                    false,
                );
                return program.partner_type();
            }
            "org" => {
                self.occurrence(env.unit, span, Target::Builtin("org".into()), false, false);
                return Program::org_type();
            }
            _ => {}
        }
        if let Some(document) = env.document.and_then(schema::document)
            && let Some(field) = document.field(name)
        {
            self.occurrence(
                env.unit,
                span,
                Target::Field {
                    record: document.name,
                    field: field.name,
                },
                false,
                false,
            );
            return program.read_type(field);
        }
        if let Some((unit, Decl::Const(index))) = program.decl(env.unit, name) {
            self.occurrence(
                env.unit,
                span,
                Target::Decl {
                    unit,
                    name: name.to_owned(),
                },
                false,
                false,
            );
            return self.const_type(unit, index);
        }
        if let Some((unit, decl)) = program.decl(env.unit, name) {
            self.occurrence(
                env.unit,
                span,
                Target::Decl {
                    unit,
                    name: name.to_owned(),
                },
                false,
                false,
            );
            let what = match decl {
                Decl::Fn(_) => Message::with(
                    message!("{0} is a function: call it as {0}(...)"),
                    [Arg::from(name)],
                ),
                _ => Message::with(
                    message!("{0} is a crosswalk: use it as value via {0}"),
                    [Arg::from(name)],
                ),
            };
            self.report(env.unit, Diagnostic::error("not-a-value", span, what));
            return Type::Any;
        }
        let mut candidates: Vec<String> =
            env.visible().into_iter().map(|(name, _, _)| name).collect();
        candidates.push("partner".into());
        candidates.push("org".into());
        if let Some(document) = env.document.and_then(schema::document) {
            candidates.extend(document.fields.iter().map(|field| field.name.to_owned()));
        }
        let mut diagnostic = Diagnostic::error(
            "unknown-name",
            span,
            Message::with(
                message!("There is nothing named {0} here"),
                [Arg::from(name)],
            ),
        );
        if let Some(close) = nearest(name, candidates.iter().map(String::as_str)) {
            diagnostic = diagnostic.with_fix(super::replace_fix(span, close));
        }
        self.report(env.unit, diagnostic);
        Type::Any
    }

    pub(crate) fn const_type(&mut self, unit: usize, index: usize) -> Type {
        let program: &Program = self.program;
        let Some(crate::lading::syntax::ast::Item::Const(decl)) =
            program.units[unit].items().get(index)
        else {
            return Type::Any;
        };
        let key = (unit, decl.name.name.clone());
        if let Some(ty) = self.const_types.get(&key) {
            return ty.clone();
        }
        self.const_types.insert(key.clone(), Type::Any);
        let mut env = Env::new(unit);
        let ty = self.expr(&mut env, &decl.value, None);
        self.const_types.insert(key, ty.clone());
        ty
    }

    fn variant(&mut self, env: &Env, variant: &str, span: Span, expected: Option<&Type>) -> Type {
        let owner = match expected.map(Type::unwrapped) {
            Some(Type::Enum(name)) => schema::enumeration(name).filter(|item| {
                item.variants
                    .iter()
                    .any(|(candidate, _)| *candidate == variant)
            }),
            _ => None,
        }
        .or_else(|| schema::enum_of_variant(variant));
        if let Some(owner) = owner {
            let name = owner
                .variants
                .iter()
                .find(|(candidate, _)| *candidate == variant)
                .map_or("", |(name, _)| *name);
            self.occurrence(
                env.unit,
                span,
                Target::Variant {
                    owner: owner.name,
                    name,
                },
                false,
                false,
            );
            return Type::Enum(owner.name);
        }
        let candidates: Vec<&str> = match expected.map(Type::unwrapped) {
            Some(Type::Enum(name)) => schema::enumeration(name)
                .map(|item| item.variants.iter().map(|(variant, _)| *variant).collect())
                .unwrap_or_default(),
            _ => schema::ENUMS
                .iter()
                .flat_map(|item| item.variants.iter().map(|(variant, _)| *variant))
                .collect(),
        };
        let mut diagnostic = Diagnostic::error(
            "unknown-variant",
            span,
            Message::with(
                message!("{0} is not a value Trenova knows"),
                [Arg::from(variant)],
            ),
        );
        if let Some(close) = nearest(variant, candidates) {
            diagnostic = diagnostic.with_fix(super::replace_fix(span, close));
        }
        self.report(env.unit, diagnostic);
        Type::Any
    }

    /// A bare X12 reference: at the transaction's top level.
    fn x12_root(&mut self, env: &mut Env, text: &str, span: Span) -> Type {
        if let Some(Type::Loop(reference)) = env.filter {
            return self.x12_child(
                env,
                reference.guide,
                Some(reference.node),
                text,
                span,
                false,
            );
        }
        let Some(guide) = env.x12_root else {
            self.report(
                env.unit,
                Diagnostic::error(
                    "x12-unavailable",
                    span,
                    Message::with(
                        message!("{0} reads X12, and only a map from a guide reads X12"),
                        [Arg::from(text)],
                    ),
                ),
            );
            return Type::Any;
        };
        self.x12_child(env, guide, None, text, span, false)
    }

    /// A segment or element reference under a node (`None` for the top).
    fn x12_child(
        &mut self,
        env: &mut Env,
        guide_index: usize,
        node: Option<usize>,
        text: &str,
        span: Span,
        optional_parent: bool,
    ) -> Type {
        let program: &Program = self.program;
        let compiled = &program.guides[guide_index].compiled;
        let (segment, position) = match nav::reference(text) {
            Some(Reference::Segment(segment)) => (segment, None),
            Some(Reference::Element(segment, position)) => (segment, Some(position)),
            None => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "x12-unknown",
                        span,
                        Message::with(
                            message!("{0} is not a segment or element reference"),
                            [Arg::from(text)],
                        ),
                    ),
                );
                return Type::Any;
            }
        };
        let Some(child) = nav::child_segment(compiled, node, segment) else {
            self.x12_not_here(env, guide_index, node, segment, span);
            return Type::Any;
        };
        let reference = NodeRef {
            guide: guide_index,
            node: child,
        };
        if compiled.node(child).usage == crate::x12::guide::model::Usage::NotUsed {
            self.report(
                env.unit,
                Diagnostic::warning(
                    "x12-unused",
                    span,
                    Message::with(
                        message!("The partner's guide does not use {0}, so it is always none"),
                        [Arg::from(segment)],
                    ),
                ),
            );
        }
        let segment_ty = nav::child_type(compiled, guide_index, child);
        let ty = match position {
            None => {
                self.occurrence(env.unit, span, Target::Segment(reference), false, false);
                segment_ty
            }
            Some(position) => {
                self.occurrence(
                    env.unit,
                    span,
                    Target::Element {
                        node: reference,
                        position,
                    },
                    false,
                    false,
                );
                if matches!(segment_ty, Type::List(_)) {
                    let first = nav::read_element(compiled, child, position).unwrap_or(Type::Any);
                    self.report(
                        env.unit,
                        Diagnostic::warning(
                            "x12-first-of-many",
                            span,
                            Message::with(
                                message!("{0} may repeat; this reads the first. Pick one with {0}[01 == \"...\"]"),
                                [Arg::from(segment)],
                            ),
                        ),
                    );
                    first.optional()
                } else {
                    match nav::read_element(compiled, child, position) {
                        Some(element) if segment_ty.is_optional() => element.optional(),
                        Some(element) => element,
                        None => {
                            self.no_element(env, segment, position, span);
                            Type::Any
                        }
                    }
                }
            }
        };
        if optional_parent { ty.optional() } else { ty }
    }

    fn no_element(&mut self, env: &Env, segment: &str, position: u32, span: Span) {
        self.report(
            env.unit,
            Diagnostic::error(
                "x12-unknown",
                span,
                Message::with(
                    message!("{0} has no element {1}"),
                    [Arg::from(segment), Arg::from(format!("{position:02}"))],
                ),
            ),
        );
    }

    fn x12_not_here(
        &mut self,
        env: &Env,
        guide_index: usize,
        node: Option<usize>,
        segment: &str,
        span: Span,
    ) {
        let compiled = &self.program.guides[guide_index].compiled;
        if node.is_none() {
            let binder =
                env.visible()
                    .into_iter()
                    .rev()
                    .find_map(|(name, ty, _)| match ty.unwrapped() {
                        Type::Loop(reference)
                            if reference.guide == guide_index
                                && nav::child_segment(compiled, Some(reference.node), segment)
                                    .is_some() =>
                        {
                            Some(name)
                        }
                        _ => None,
                    });
            if let Some(binder) = binder {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "x12-unqualified",
                        span,
                        Message::with(
                            message!(
                                "{0} here reads the top of the transaction; the item's is {1}.{0}"
                            ),
                            [Arg::from(segment), Arg::from(binder.as_str())],
                        ),
                    )
                    .with_fix(Fix {
                        label: Message::with(
                            message!("Read {0}.{1}"),
                            [Arg::from(binder.as_str()), Arg::from(segment)],
                        ),
                        changes: vec![Change {
                            span: Span::new(span.start, span.start),
                            insert: format!("{binder}."),
                        }],
                    }),
                );
                return;
            }
        }
        let message = match nav::find_segment_anywhere(compiled, segment) {
            Some(path) if !path.is_empty() => Message::with(
                message!(
                    "{0} is in loop {1}, not here: go through the loop, as in each {1} as item, then item.{0}"
                ),
                [Arg::from(segment), Arg::from(path.join("."))],
            ),
            Some(_) => Message::with(
                message!("{0} is at the top of the transaction, not in this loop"),
                [Arg::from(segment)],
            ),
            None => Message::with(
                message!("The guide has no {0} segment"),
                [Arg::from(segment)],
            ),
        };
        let candidates: Vec<String> = nav::children(compiled, node)
            .iter()
            .map(|index| match &compiled.node(*index).kind {
                NodeKind::Segment(id) | NodeKind::Loop { id, .. } => id.clone(),
            })
            .collect();
        let mut diagnostic = Diagnostic::error("x12-unknown", span, message);
        if let Some(close) = nearest(segment, candidates.iter().map(String::as_str)) {
            diagnostic = diagnostic.with_fix(super::replace_fix(span, close));
        }
        self.report(env.unit, diagnostic);
    }

    fn digits(&mut self, env: &mut Env, text: &str, span: Span) -> Type {
        if let Some(Type::Loop(reference)) = env.filter {
            return self.loop_child(
                env,
                reference.guide,
                Some(reference.node),
                text,
                span,
                false,
            );
        }
        if let Some(filter) = env.filter.clone() {
            let Ok(position) = text.parse::<u32>() else {
                return Type::Any;
            };
            return self.element_of(env, &filter, position, span);
        }
        let Some(guide) = env.x12_root else {
            self.report(
                env.unit,
                Diagnostic::error(
                    "x12-unavailable",
                    span,
                    Message::with(
                        message!("{0} reads X12, and only a map from a guide reads X12"),
                        [Arg::from(text)],
                    ),
                ),
            );
            return Type::Any;
        };
        self.loop_child(env, guide, None, text, span, false)
    }

    fn loop_child(
        &mut self,
        env: &Env,
        guide_index: usize,
        node: Option<usize>,
        id: &str,
        span: Span,
        optional_parent: bool,
    ) -> Type {
        let compiled = &self.program.guides[guide_index].compiled;
        let Some(child) = nav::child_loop(compiled, node, id) else {
            let candidates: Vec<String> = nav::children(compiled, node)
                .iter()
                .filter_map(|index| nav::loop_id(compiled, *index).map(str::to_owned))
                .collect();
            let mut diagnostic = Diagnostic::error(
                "x12-unknown",
                span,
                Message::with(message!("There is no loop {0} here"), [Arg::from(id)]),
            );
            if let Some(close) = nearest(id, candidates.iter().map(String::as_str)) {
                diagnostic = diagnostic.with_fix(super::replace_fix(span, close));
            }
            self.report(env.unit, diagnostic);
            return Type::Any;
        };
        let reference = NodeRef {
            guide: guide_index,
            node: child,
        };
        self.occurrence(env.unit, span, Target::Loop(reference), false, false);
        let ty = nav::child_type(compiled, guide_index, child);
        if optional_parent { ty.optional() } else { ty }
    }

    /// An element of a segment value by position.
    fn element_of(&mut self, env: &Env, segment: &Type, position: u32, span: Span) -> Type {
        let optional = segment.is_optional();
        let ty = match segment.unwrapped() {
            Type::Segment(reference) => {
                let compiled = &self.program.guides[reference.guide].compiled;
                let id = nav::segment_id(compiled, reference.node).to_owned();
                if let Some(ty) = nav::read_element(compiled, reference.node, position) {
                    self.occurrence(
                        env.unit,
                        span,
                        Target::Element {
                            node: *reference,
                            position,
                        },
                        false,
                        false,
                    );
                    ty
                } else {
                    self.no_element(env, &id, position, span);
                    Type::Any
                }
            }
            Type::SegmentId(id) => self.element_by_id(env, id, position, span),
            Type::Struct(fields) => {
                let key = format!("{position:02}");
                fields.iter().find(|(name, _)| *name == key).map_or_else(
                    || {
                        self.report(
                            env.unit,
                            Diagnostic::error(
                                "x12-unknown",
                                span,
                                Message::with(
                                    message!("There is no component {0}"),
                                    [Arg::from(key.clone())],
                                ),
                            ),
                        );
                        Type::Any
                    },
                    |(_, ty)| ty.clone(),
                )
            }
            Type::Any => Type::Any,
            other => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "not-a-segment",
                        span,
                        Message::with(
                            message!("Element positions read segments, and this is {0}"),
                            [Arg::from(other.to_string())],
                        ),
                    ),
                );
                Type::Any
            }
        };
        if optional { ty.optional() } else { ty }
    }

    /// An element of a segment known only by ID: typed from the first guide that has it.
    fn element_by_id(&mut self, env: &Env, id: &str, position: u32, span: Span) -> Type {
        let program: &Program = self.program;
        let guide = env.x12_root.or_else(|| {
            program
                .guides
                .iter()
                .position(|entry| entry.compiled.segment(id).is_some())
        });
        let found = guide.and_then(|guide| {
            let compiled = &program.guides[guide].compiled;
            let definition = compiled.definition(nav::first_use(compiled, id), id)?;
            let element = definition.elements.get(position.checked_sub(1)? as usize)?;
            Some(nav::element_type(id, position, element).optional())
        });
        if let Some(ty) = found {
            return ty;
        }
        if crate::x12::guide::builtin::is_known_segment(id) {
            return Type::Text.optional();
        }
        self.no_element(env, id, position, span);
        Type::Any
    }

    #[allow(clippy::too_many_lines)]
    fn record(
        &mut self,
        env: &mut Env,
        ty: Option<&crate::lading::syntax::ast::Ident>,
        fields: &[FieldInit],
        span: Span,
        expected: Option<&Type>,
    ) -> Type {
        let record_name = match ty {
            Some(ident) => {
                if let Some(record) = schema::record(&ident.name) {
                    self.occurrence(
                        env.unit,
                        ident.span,
                        Target::Type(record.name.to_owned()),
                        false,
                        false,
                    );
                    Some(record)
                } else {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "unknown-type",
                            ident.span,
                            Message::with(
                                message!("There is no record named {0}"),
                                [Arg::from(ident.name.as_str())],
                            ),
                        ),
                    );
                    return Type::Any;
                }
            }
            None => match expected.map(Type::unwrapped) {
                Some(Type::Record(name)) => schema::record(name),
                _ => None,
            },
        };
        let Some(record) = record_name else {
            let mut typed = Vec::new();
            let expected_fields = match expected.map(Type::unwrapped) {
                Some(Type::Struct(fields)) => fields.clone(),
                _ => Vec::new(),
            };
            for field in fields {
                let field_expected = expected_fields
                    .iter()
                    .find(|(name, _)| *name == field.name.name)
                    .map(|(_, ty)| ty.clone());
                let ty = self.expr(env, &field.value, field_expected.as_ref());
                let ty = self.else_clause(env, ty, field.otherwise.as_ref(), true);
                typed.push((field.name.name.clone(), ty));
            }
            return Type::Struct(typed);
        };
        let mut optional = false;
        let mut seen = Vec::new();
        for field in fields {
            let Some(definition) = record.field(&field.name.name) else {
                let mut diagnostic = Diagnostic::error(
                    "unknown-field",
                    field.name.span,
                    Message::with(
                        message!("{0} has no field {1}"),
                        [Arg::from(record.name), Arg::from(field.name.name.as_str())],
                    ),
                );
                if let Some(close) = nearest(
                    &field.name.name,
                    record.fields.iter().map(|field| field.name),
                ) {
                    diagnostic = diagnostic.with_fix(super::replace_fix(field.name.span, close));
                }
                self.report(env.unit, diagnostic);
                self.expr(env, &field.value, None);
                continue;
            };
            self.occurrence(
                env.unit,
                field.name.span,
                Target::Field {
                    record: record.name,
                    field: definition.name,
                },
                false,
                true,
            );
            seen.push(definition.name);
            let field_ty = self.program.field_type(definition);
            let ty = self.expr(env, &field.value, Some(&field_ty));
            let ty = self.else_clause(env, ty, field.otherwise.as_ref(), true);
            if !ty.unwrapped().fits(&field_ty) && !ty.is_any() {
                self.report(env.unit, mismatch(field.value.span, &field_ty, &ty));
            }
            if definition.required && ty.is_optional() {
                optional = true;
            }
        }
        let missing: Vec<&str> = record
            .fields
            .iter()
            .filter(|field| field.required && !seen.contains(&field.name))
            .map(|field| field.name)
            .collect();
        if !missing.is_empty() {
            self.report(
                env.unit,
                Diagnostic::error(
                    "field-unassigned",
                    span,
                    Message::with(
                        message!("{0} needs {1}"),
                        [Arg::from(record.name), Arg::from(missing.join(", "))],
                    ),
                ),
            );
        }
        let ty = Type::Record(record.name);
        if optional { ty.optional() } else { ty }
    }

    #[allow(clippy::too_many_lines)]
    pub(crate) fn member(
        &mut self,
        env: &mut Env,
        target: &Type,
        name: &str,
        span: Span,
        kind: MemberKind,
    ) -> Type {
        let optional = target.is_optional();
        let inner = target.unwrapped().clone();
        let wrap = |ty: Type| if optional { ty.optional() } else { ty };
        match (&inner, kind) {
            (Type::Any, _) => Type::Any,
            (Type::Transaction(guide), MemberKind::X12) => {
                self.x12_child(env, *guide, None, name, span, optional)
            }
            (Type::Transaction(guide), MemberKind::Digits) => {
                self.loop_child(env, *guide, None, name, span, optional)
            }
            (Type::Loop(reference), MemberKind::X12) => self.x12_child(
                env,
                reference.guide,
                Some(reference.node),
                name,
                span,
                optional,
            ),
            (Type::Loop(reference), MemberKind::Digits) => self.loop_child(
                env,
                reference.guide,
                Some(reference.node),
                name,
                span,
                optional,
            ),
            (Type::LoopId(_), MemberKind::X12 | MemberKind::Digits) => Type::Any.optional(),
            (Type::Segment(_) | Type::SegmentId(_) | Type::Struct(_), MemberKind::Digits) => {
                let Ok(position) = name.parse::<u32>() else {
                    return Type::Any;
                };
                self.element_of(env, target, position, span)
            }
            (Type::List(item), MemberKind::X12 | MemberKind::Digits)
                if matches!(item.as_ref(), Type::Segment(_) | Type::Loop(_)) =>
            {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "x12-many",
                        span,
                        Message::new(message!(
                            "This repeats: pick one with [01 == \"...\"], .first, or go through each"
                        )),
                    ),
                );
                Type::Any
            }
            (Type::Record(record), MemberKind::Name) => {
                let Some(owner) = schema::record(record) else {
                    return Type::Any;
                };
                if let Some(field) = owner.field(name) {
                    self.occurrence(
                        env.unit,
                        span,
                        Target::Field {
                            record: owner.name,
                            field: field.name,
                        },
                        false,
                        false,
                    );
                    return wrap(self.program.read_type(field));
                }
                {
                    let names: Vec<&str> = owner.fields.iter().map(|field| field.name).collect();
                    self.property(env, target, name, span, &names)
                }
            }
            (Type::Ref(kind_name), MemberKind::Name) => {
                if let Some((field, ty, _)) = schema::ref_fields(kind_name)
                    .into_iter()
                    .find(|(field, _, _)| *field == name)
                {
                    let _ = field;
                    return wrap(ty);
                }
                let names: Vec<&str> = schema::ref_fields(kind_name)
                    .into_iter()
                    .map(|(field, _, _)| field)
                    .collect();
                self.property(env, target, name, span, &names)
            }
            (Type::Struct(fields), MemberKind::Name) => {
                if let Some((_, ty)) = fields.iter().find(|(field, _)| field == name) {
                    if let Some(builtin) = self.builtin_member(target, name) {
                        self.occurrence(env.unit, span, builtin, false, false);
                    }
                    return wrap(ty.clone());
                }
                let names: Vec<&str> = fields.iter().map(|(field, _)| field.as_str()).collect();
                let names_owned: Vec<String> =
                    names.iter().map(|name| (*name).to_owned()).collect();
                self.property_owned(env, target, name, span, &names_owned)
            }
            (Type::Json, MemberKind::Name) => Type::Json.optional(),
            (_, MemberKind::Name) => self.property(env, target, name, span, &[]),
            _ => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "unknown-member",
                        span,
                        Message::with(
                            message!("{0} has no {1}"),
                            [Arg::from(inner.to_string()), Arg::from(name)],
                        ),
                    ),
                );
                Type::Any
            }
        }
    }

    fn builtin_member(&self, target: &Type, name: &str) -> Option<Target> {
        let partner = self.program.partner_type();
        if target.unwrapped() == &partner {
            if self
                .program
                .partner
                .as_ref()
                .is_some_and(|partner| partner.settings.iter().any(|setting| setting.name == name))
            {
                return Some(Target::Setting(name.to_owned()));
            }
            return Some(Target::Builtin(format!("partner.{name}")));
        }
        if target.unwrapped() == &Program::org_type() {
            return Some(Target::Builtin(format!("org.{name}")));
        }
        None
    }

    fn property(
        &mut self,
        env: &mut Env,
        target: &Type,
        name: &str,
        span: Span,
        fields: &[&str],
    ) -> Type {
        let owned: Vec<String> = fields.iter().map(|field| (*field).to_owned()).collect();
        self.property_owned(env, target, name, span, &owned)
    }

    /// A zero-argument standard function used as a property (`stops.count`, `name.length`).
    fn property_owned(
        &mut self,
        env: &mut Env,
        target: &Type,
        name: &str,
        span: Span,
        fields: &[String],
    ) -> Type {
        if let Some(ty) = self.stdlib_call(env, name, Some((target.clone(), span)), &[], span, true)
        {
            self.occurrence(
                env.unit,
                span,
                Target::Stdlib(
                    crate::lading::stdlib::named(name)
                        .next()
                        .map_or("", |f| f.name),
                ),
                false,
                false,
            );
            return ty;
        }
        let mut diagnostic = Diagnostic::error(
            "unknown-member",
            span,
            Message::with(
                message!("{0} has no {1}"),
                [Arg::from(target.unwrapped().to_string()), Arg::from(name)],
            ),
        );
        if let Some(close) = nearest(name, fields.iter().map(String::as_str)) {
            diagnostic = diagnostic.with_fix(super::replace_fix(span, close));
        }
        self.report(env.unit, diagnostic);
        Type::Any
    }

    fn index(&mut self, env: &mut Env, target: &Type, index: &Expr) -> Type {
        match target.unwrapped().clone() {
            Type::List(item) => {
                let saved = env.filter.take();
                if matches!(
                    item.as_ref(),
                    Type::Segment(_) | Type::SegmentId(_) | Type::Loop(_)
                ) {
                    env.filter = Some(item.as_ref().clone());
                }
                let index_ty = self.expr(env, index, None);
                env.filter = saved;
                match index_ty.unwrapped() {
                    Type::Bool | Type::Int | Type::Any => item.as_ref().clone().optional(),
                    other => {
                        self.report(
                            env.unit,
                            Diagnostic::error(
                                "bad-index",
                                index.span,
                                Message::with(
                                    message!("A list takes a condition or a position, not {0}"),
                                    [Arg::from(other.to_string())],
                                ),
                            ),
                        );
                        Type::Any
                    }
                }
            }
            Type::Map(value) => {
                self.expect_type(env, index, &Type::Text);
                value.as_ref().clone().optional()
            }
            Type::Segment(_) | Type::SegmentId(_) => {
                let saved = env.filter.replace(target.unwrapped().clone());
                self.condition(env, index);
                env.filter = saved;
                target.clone().optional()
            }
            Type::Any => {
                self.expr(env, index, None);
                Type::Any
            }
            other => {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "bad-index",
                        index.span,
                        Message::with(
                            message!("{0} cannot be indexed"),
                            [Arg::from(other.to_string())],
                        ),
                    ),
                );
                Type::Any
            }
        }
    }

    fn unary(&mut self, env: &mut Env, op: UnOp, inner: &Expr) -> Type {
        let ty = self.expr(env, inner, None);
        match op {
            UnOp::Not => {
                if !matches!(ty.unwrapped(), Type::Bool | Type::Any) {
                    self.report(env.unit, mismatch(inner.span, &Type::Bool, &ty));
                }
                Type::Bool
            }
            UnOp::Neg => {
                if !matches!(
                    ty,
                    Type::Int
                        | Type::Decimal
                        | Type::Money
                        | Type::Measure(_)
                        | Type::Duration
                        | Type::Percent
                        | Type::Any
                ) {
                    self.report(env.unit, mismatch(inner.span, &Type::Decimal, &ty));
                }
                ty
            }
        }
    }

    #[allow(clippy::too_many_lines)]
    fn binary(&mut self, env: &mut Env, op: BinOp, lhs: &Expr, rhs: &Expr, span: Span) -> Type {
        match op {
            BinOp::Default => {
                let left = self.expr(env, lhs, None);
                let right = self.expr(env, rhs, Some(left.unwrapped()));
                if !left.is_optional() && !left.is_any() {
                    self.report(
                        env.unit,
                        Diagnostic::warning(
                            "default-never-used",
                            rhs.span,
                            Message::new(message!(
                                "The left side is never none, so the default is never used"
                            )),
                        ),
                    );
                }
                match left.unwrapped().join(&right) {
                    Some(joined) if right.is_optional() => joined.optional(),
                    Some(joined) => joined.unwrapped().clone(),
                    None => {
                        self.report(env.unit, mismatch(rhs.span, left.unwrapped(), &right));
                        Type::Any
                    }
                }
            }
            BinOp::And | BinOp::Or => {
                self.condition(env, lhs);
                self.condition(env, rhs);
                Type::Bool
            }
            BinOp::Eq | BinOp::Ne => {
                let left = self.expr(env, lhs, None);
                let right = self.expr(env, rhs, Some(left.unwrapped()));
                if left.unwrapped().join(right.unwrapped()).is_none() {
                    self.report(env.unit, mismatch(rhs.span, left.unwrapped(), &right));
                }
                self.code_literal(env, &left, rhs);
                self.code_literal(env, &right, lhs);
                Type::Bool
            }
            BinOp::Lt | BinOp::Le | BinOp::Gt | BinOp::Ge => {
                let left = self.expr(env, lhs, None);
                let right = self.expr(env, rhs, Some(left.unwrapped()));
                let comparable = left.unwrapped().join(right.unwrapped()).is_some()
                    || matches!((left.unwrapped(), right.unwrapped()), (Type::Measure(a), Type::Measure(b)) if a == b);
                if !comparable || !left.is_ordered() {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "not-comparable",
                            span,
                            Message::with(
                                message!("{0} and {1} cannot be ordered"),
                                [Arg::from(left.to_string()), Arg::from(right.to_string())],
                            ),
                        ),
                    );
                }
                Type::Bool
            }
            BinOp::In => {
                let left = self.expr(env, lhs, None);
                let right = self.expr(env, rhs, Some(&left.clone().list()));
                match right.unwrapped() {
                    Type::List(item) => {
                        if left.unwrapped().join(item).is_none() {
                            self.report(env.unit, mismatch(lhs.span, item, &left));
                        }
                        if let ExprKind::List(items) = &rhs.kind {
                            for item in items {
                                self.code_literal(env, &left, item);
                            }
                        }
                    }
                    Type::Any => {}
                    other => self.report(
                        env.unit,
                        Diagnostic::error(
                            "not-a-list",
                            rhs.span,
                            Message::with(
                                message!("in needs a list, and this is {0}"),
                                [Arg::from(other.to_string())],
                            ),
                        ),
                    ),
                }
                Type::Bool
            }
            BinOp::Add | BinOp::Sub | BinOp::Mul | BinOp::Div => {
                let left = self.expr(env, lhs, None);
                let right = self.expr(env, rhs, None);
                for (side, ty) in [(lhs, &left), (rhs, &right)] {
                    if ty.is_optional() {
                        self.report(
                            env.unit,
                            Diagnostic::error(
                                "optional-unhandled",
                                side.span,
                                Message::new(message!(
                                    "This may be none: give a default with ?? before calculating"
                                )),
                            ),
                        );
                    }
                }
                arithmetic(op, left.unwrapped(), right.unwrapped()).unwrap_or_else(|| {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "bad-arithmetic",
                            span,
                            Message::with(
                                message!("{0} {1} {2} is not a calculation Lading knows"),
                                [
                                    Arg::from(left.unwrapped().to_string()),
                                    Arg::from(op.symbol()),
                                    Arg::from(right.unwrapped().to_string()),
                                ],
                            ),
                        ),
                    );
                    Type::Any
                })
            }
        }
    }

    /// Warns when a text literal compared with a coded element is not one of its codes.
    pub(crate) fn code_literal(&mut self, env: &Env, ty: &Type, literal: &Expr) {
        let (Type::Code(element), ExprKind::Text(parts)) = (ty.unwrapped(), &literal.kind) else {
            return;
        };
        let [TextPart::Literal(value)] = parts.as_slice() else {
            return;
        };
        let Some(codes) = self.codes_of(env, element) else {
            return;
        };
        if codes.iter().any(|(code, _)| code == value) {
            return;
        }
        let mut diagnostic = Diagnostic::warning(
            "code-not-in-list",
            literal.span,
            Message::with(
                message!("{0} is not one of {1}'s codes in the guide"),
                [Arg::from(value.as_str()), Arg::from(element.as_str())],
            ),
        );
        let names: Vec<&str> = codes.iter().map(|(code, _)| code.as_str()).collect();
        if let Some(close) = nearest(value, names) {
            diagnostic =
                diagnostic.with_fix(super::replace_fix(literal.span, &format!("\"{close}\"")));
        }
        self.report(env.unit, diagnostic);
    }

    /// The codes of an element reference (`S502`) under the guide in context.
    pub(crate) fn codes_of(&self, env: &Env, element: &str) -> Option<Vec<(String, String)>> {
        let Some(Reference::Element(segment, position)) = nav::reference(element) else {
            return None;
        };
        let guides = &self.program.guides;
        let guide = env
            .x12_root
            .or(env.write_guide)
            .or(env.node.map(|node| node.guide))
            .or_else(|| {
                guides
                    .iter()
                    .position(|entry| entry.unit.is_some() && entry.compiled.uses_segment(segment))
                    .or_else(|| {
                        guides
                            .iter()
                            .position(|entry| entry.compiled.uses_segment(segment))
                    })
            })?;
        nav::codes(&guides[guide].compiled, segment, position)
    }

    #[allow(clippy::too_many_lines)]
    fn via(
        &mut self,
        env: &Env,
        value: &Type,
        value_span: Span,
        crosswalk: &crate::lading::syntax::ast::Ident,
        back: bool,
    ) -> Type {
        let program: &Program = self.program;
        let Some((unit, Decl::Crosswalk(index))) = program.decl(env.unit, &crosswalk.name) else {
            let names: Vec<String> = program
                .crosswalks
                .keys()
                .filter(|(unit, _)| {
                    *unit == env.unit
                        || program.units[env.unit]
                            .header()
                            .uses()
                            .iter()
                            .any(|library| program.libraries.get(&library.name) == Some(unit))
                })
                .map(|(_, name)| name.clone())
                .collect();
            let mut diagnostic = Diagnostic::error(
                "unknown-crosswalk",
                crosswalk.span,
                Message::with(
                    message!("There is no crosswalk named {0}"),
                    [Arg::from(crosswalk.name.as_str())],
                ),
            );
            if let Some(close) = nearest(&crosswalk.name, names.iter().map(String::as_str)) {
                diagnostic = diagnostic.with_fix(super::replace_fix(crosswalk.span, close));
            }
            self.report(env.unit, diagnostic);
            return Type::Any;
        };
        self.occurrence(
            env.unit,
            crosswalk.span,
            Target::Decl {
                unit,
                name: crosswalk.name.clone(),
            },
            false,
            false,
        );
        let Some((left, right)) = program
            .crosswalks
            .get(&(unit, crosswalk.name.clone()))
            .cloned()
        else {
            return Type::Any;
        };
        let Some(crate::lading::syntax::ast::Item::Crosswalk(decl)) =
            program.units[unit].items().get(index)
        else {
            return Type::Any;
        };
        let forward = if back {
            false
        } else if value.unwrapped().fits(&left) {
            true
        } else if value.unwrapped().fits(&right) {
            false
        } else {
            if !value.is_any() {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "crosswalk-type",
                        value_span,
                        Message::with(
                            message!("{0} pairs {1} with {2}, and this is {3}"),
                            [
                                Arg::from(crosswalk.name.as_str()),
                                Arg::from(left.to_string()),
                                Arg::from(right.to_string()),
                                Arg::from(value.to_string()),
                            ],
                        ),
                    ),
                );
            }
            return Type::Any;
        };
        let has_else = decl.otherwise.iter().any(|(direction, _)| {
            matches!(
                (direction, forward),
                (Direction::Forward, true) | (Direction::Backward, false)
            )
        });
        let (from, to) = if forward {
            (&left, &right)
        } else {
            (&right, &left)
        };
        // What the crosswalk covers in this direction.
        let covered: Vec<String> = decl
            .pairs
            .iter()
            .filter(|pair| match pair.direction {
                Direction::Both => true,
                Direction::Forward => forward,
                Direction::Backward => !forward,
            })
            .flat_map(|pair| {
                if forward {
                    pair.left.iter()
                } else {
                    pair.right.iter()
                }
            })
            .filter_map(literal_text)
            .collect();
        if !has_else {
            if let Type::Enum(name) = from.unwrapped()
                && let Some(item) = schema::enumeration(name)
            {
                let missing: Vec<&str> = item
                    .variants
                    .iter()
                    .map(|(variant, _)| *variant)
                    .filter(|variant| !covered.iter().any(|c| c == variant))
                    .collect();
                if !missing.is_empty() {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "crosswalk-incomplete",
                            crosswalk.span,
                            Message::with(
                                message!("{0} has nothing for {1}: add pairs or an else"),
                                [
                                    Arg::from(crosswalk.name.as_str()),
                                    Arg::from(missing.join(", ")),
                                ],
                            ),
                        ),
                    );
                }
            }
            if let Type::Code(element) = value.unwrapped()
                && let Some(codes) = self.codes_of(env, element)
            {
                let missing: Vec<String> = codes
                    .iter()
                    .map(|(code, _)| code.clone())
                    .filter(|code| !covered.contains(code))
                    .collect();
                if !missing.is_empty() && forward == value.unwrapped().fits(&left) {
                    let mut diagnostic = Diagnostic::warning(
                        "crosswalk-misses-code",
                        crosswalk.span,
                        Message::with(
                            message!("The guide allows {0} in {1}, and {2} does not read it"),
                            [
                                Arg::from(missing.join(", ")),
                                Arg::from(element.as_str()),
                                Arg::from(crosswalk.name.as_str()),
                            ],
                        ),
                    );
                    if unit == env.unit
                        && let Some(first) = decl.pairs.first()
                        && let Some(placeholder) = first.right.first()
                    {
                        let placeholder_text = &program.units[unit].source
                            [placeholder.span.start as usize..placeholder.span.end as usize];
                        let insert_at = first.span.start;
                        let lines = missing.iter().fold(String::new(), |mut lines, code| {
                            use std::fmt::Write as _;
                            let _ = write!(lines, "\"{code}\" -> {placeholder_text}\n  ");
                            lines
                        });
                        diagnostic = diagnostic.with_fix(Fix {
                            label: Message::new(message!("Add the missing codes")),
                            changes: vec![Change {
                                span: Span::new(insert_at, insert_at),
                                insert: lines,
                            }],
                        });
                    }
                    self.report(env.unit, diagnostic);
                }
            }
        }
        let _ = from;
        if has_else && !value.is_optional() {
            to.unwrapped().clone()
        } else {
            to.unwrapped().clone().optional()
        }
    }
}

/// The text of a literal crosswalk value: a text or a variant.
fn literal_text(expr: &Expr) -> Option<String> {
    match &expr.kind {
        ExprKind::Text(parts) => match parts.as_slice() {
            [TextPart::Literal(value)] => Some(value.clone()),
            [] => Some(String::new()),
            _ => None,
        },
        ExprKind::Type(name) => Some(name.clone()),
        ExprKind::Number(value) => Some(value.clone()),
        _ => None,
    }
}

/// The result of an arithmetic operation, when it has one.
#[must_use]
pub fn arithmetic(op: BinOp, left: &Type, right: &Type) -> Option<Type> {
    use Type::{Any, DateTime, Decimal, Duration, Int, Measure, Money, Percent, Text};
    Some(match (op, left, right) {
        (_, Any, _) | (_, _, Any) => Any,
        (BinOp::Add, Text | Type::Code(_), Text | Type::Code(_)) => Text,
        (BinOp::Add | BinOp::Sub | BinOp::Mul, Int, Int) => Int,
        (BinOp::Div, Measure(a), Measure(b)) if a == b => Decimal,
        (_, Int | Decimal, Int | Decimal)
        | (BinOp::Div, Money, Money)
        | (BinOp::Mul, Percent, Int | Decimal)
        | (BinOp::Mul, Int | Decimal, Percent) => Decimal,
        (BinOp::Add | BinOp::Sub, Money, Money)
        | (BinOp::Mul | BinOp::Div, Money, Int | Decimal)
        | (BinOp::Mul, Int | Decimal | Percent, Money)
        | (BinOp::Mul, Money, Percent) => Money,
        (BinOp::Add | BinOp::Sub, Measure(a), Measure(b)) if a == b => Measure(*a),
        (BinOp::Mul | BinOp::Div, Measure(dim), Int | Decimal)
        | (BinOp::Mul, Int | Decimal, Measure(dim)) => Measure(*dim),
        (BinOp::Add | BinOp::Sub, DateTime, Duration) | (BinOp::Add, Duration, DateTime) => {
            DateTime
        }
        (BinOp::Sub, DateTime, DateTime)
        | (BinOp::Add | BinOp::Sub, Duration, Duration)
        | (BinOp::Mul | BinOp::Div, Duration, Int | Decimal)
        | (BinOp::Mul, Int | Decimal, Duration) => Duration,
        (BinOp::Add | BinOp::Sub, Type::Date, Duration) => Type::Date,
        (BinOp::Add | BinOp::Sub, Percent, Percent) => Percent,
        _ => return None,
    })
}
