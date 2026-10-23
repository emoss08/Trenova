//! `match`, as an expression and as a statement: each arm's pattern is typed against the
//! scrutinee and lowered to a [`Pat`], its bindings declared for the arm, its guard checked,
//! and the arms then analysed for coverage by [`super::patterns`].

use super::HintKind;
use super::checker::{Checker, Want};
use super::consts::{date_ordinal, date_time_ordinal};
use super::describe::{quote, quote_span, words};
use super::patterns::{ArmPat, Bound, Domain, Interval, Key, Pat, Space, analyze, listing};
use super::scope::LocalKind;
use crate::message;
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::syntax::ast::{Arm, ArmBody, Expr, ExprKind, Pattern, PatternKind, UnaryOp};
use crate::tariff::types::{EnumType, Type};
use crate::tariff::units::{Unit, convert};
use rust_decimal::Decimal;
use std::sync::Arc;

/// What one match's patterns share while they are typed.
struct Line {
    /// The unit every quantity bound is converted into: the first one written.
    unit: Option<Unit>,
}

impl Checker<'_> {
    /// A `match` expression.
    pub fn match_expr(
        &mut self,
        expr: &Expr,
        scrutinee: &Expr,
        arms: &[Arm],
        expected: Option<&Type>,
    ) -> Type {
        self.match_arms(expr.span, scrutinee, arms, expected, false)
            .unwrap_or(Type::Error)
    }

    /// Checks a match's arms; for an expression, returns the arms' joined type.
    pub fn match_arms(
        &mut self,
        span: Span,
        scrutinee: &Expr,
        arms: &[Arm],
        expected: Option<&Type>,
        statement: bool,
    ) -> Option<Type> {
        let ty = self.expr(scrutinee, None);
        let space = space_of(&ty);
        let key = self.path_key(scrutinee);
        let mut line = Line { unit: None };
        let mut lowered: Vec<ArmPat> = Vec::with_capacity(arms.len());
        let mut joined: Option<Type> = None;
        let mut none_covered = false;
        for arm in arms {
            self.scopes.push(Vec::new());
            let before = self.out.diagnostics.len();
            let pat = self.pattern(&arm.pattern, &ty, none_covered, &mut line);
            let broken = self.out.diagnostics.len() > before;
            let excludes_none = !matches!(pat, Pat::Any | Pat::None) && !or_has_none(&pat);
            if ty.is_optional()
                && (excludes_none || none_covered)
                && let Some(key) = &key
            {
                self.scopes.narrow(key.clone());
            }
            if let Some(guard) = &arm.guard {
                self.expect(guard, &Type::Bool, &Want::Condition);
            }
            match &arm.body {
                ArmBody::Block(block) => self.block(block),
                ArmBody::Expr(body) if statement => {
                    self.statements.push(body.span);
                    self.expr(body, None);
                    self.statements.pop();
                }
                ArmBody::Expr(body) => {
                    let value = self.expr(body, expected.or(joined.as_ref()));
                    joined = Some(self.join_branch(joined, body, &value));
                }
            }
            self.pop_scope();
            if arm.guard.is_none()
                && !broken
                && (matches!(pat, Pat::Any | Pat::None) || or_has_none(&pat) || or_has_any(&pat))
            {
                none_covered = true;
            }
            lowered.push(ArmPat {
                pat,
                guarded: arm.guard.is_some() || broken,
                broken,
            });
        }
        self.coverage(span, scrutinee, arms, &space, &lowered, &ty);
        if statement {
            None
        } else {
            Some(joined.unwrap_or(Type::Error))
        }
    }

    fn coverage(
        &mut self,
        span: Span,
        scrutinee: &Expr,
        arms: &[Arm],
        space: &Space,
        lowered: &[ArmPat],
        ty: &Type,
    ) {
        if matches!(ty, Type::Error) {
            return;
        }
        let coverage = analyze(space, lowered);
        for (index, earlier, what) in &coverage.overlaps {
            let (Some(arm), Some(other)) = (arms.get(*index), arms.get(*earlier)) else {
                continue;
            };
            self.report(
                Diagnostic::error(
                    "match-overlap",
                    arm.pattern.span,
                    Message::with(
                        message!("This arm overlaps an arm above it: {0} matches both"),
                        [Arg::from(what.clone())],
                    ),
                )
                .with_related(
                    other.pattern.span,
                    Message::new(message!("The arm it overlaps")),
                ),
            );
        }
        for index in &coverage.unreachable {
            if let Some(arm) = arms.get(*index) {
                self.report(Diagnostic::warning(
                    "unreachable-arm",
                    arm.pattern.span,
                    Message::new(message!(
                        "Nothing reaches this arm: the arms above it match every value it would"
                    )),
                ));
            }
        }
        if coverage.missing.is_empty() {
            self.exhaustive.insert(span.start);
            return;
        }
        let keyword = Span::new(span.start, span.start.saturating_add(5));
        let at = if self.text(keyword) == "match" {
            keyword
        } else {
            scrutinee.span
        };
        self.report(Diagnostic::error(
            "match-exhaustive",
            at,
            Message::with(
                message!("This match on {0} doesn't cover {1}. Add an arm for each, or a `_` arm for everything else"),
                [quote_span(self.source, scrutinee.span), Arg::from(listing(&coverage.missing, coverage.more))],
            ),
        ));
    }

    /// Types a pattern against the scrutinee's type and lowers it for coverage.
    fn pattern(
        &mut self,
        pattern: &Pattern,
        ty: &Type,
        none_covered: bool,
        line: &mut Line,
    ) -> Pat {
        let inner = ty.unwrapped().clone();
        if matches!(inner, Type::Error) {
            self.bind_all(pattern);
            return Pat::Any;
        }
        match &pattern.kind {
            PatternKind::Wildcard | PatternKind::Missing => Pat::Any,
            PatternKind::None => {
                if !ty.is_optional() {
                    self.pattern_error(pattern, ty, Message::new(message!("`none`")));
                    return Pat::Any;
                }
                Pat::None
            }
            PatternKind::Binding(name) => {
                let bound = if none_covered { inner } else { ty.clone() };
                self.hint(name.span, &bound, HintKind::Pattern);
                self.declare(name, LocalKind::Bound, bound);
                Pat::Any
            }
            PatternKind::Literal(expr) => self.literal_pattern(pattern, expr, &inner, line),
            PatternKind::Range {
                start,
                end,
                inclusive,
            } => self.range_pattern(
                pattern,
                start.as_ref(),
                end.as_ref(),
                *inclusive,
                &inner,
                line,
            ),
            PatternKind::Variant {
                enumeration,
                variant,
            } => self.variant_pattern(pattern, ty, enumeration.as_ref(), variant),
            PatternKind::Record(fields) => self.record_pattern(pattern, ty, fields),
            PatternKind::Or(alternatives) => self.or_pattern(ty, alternatives, none_covered, line),
        }
    }

    fn variant_pattern(
        &mut self,
        pattern: &Pattern,
        ty: &Type,
        enumeration: Option<&crate::tariff::syntax::ast::Name>,
        variant: &crate::tariff::syntax::ast::Name,
    ) -> Pat {
        let inner = ty.unwrapped().clone();
        let Type::Enum(owner) = &inner else {
            self.pattern_error(
                pattern,
                ty,
                Message::with(message!("the variant {0}"), [quote(&variant.text)]),
            );
            return Pat::Any;
        };
        if let Some(written) = enumeration
            && Type::Enum(owner.clone()).to_string() != written.text
        {
            self.report(Diagnostic::error(
                "type-mismatch",
                written.span,
                Message::with(
                    message!("The value matched is {0}, not a value of type {1}"),
                    [Arg::from(words(&inner)), quote(&written.text)],
                ),
            ));
            return Pat::Any;
        }
        if !owner.has(&variant.text) {
            let candidates: Vec<String> = owner
                .variants
                .iter()
                .map(|(name, _)| name.clone())
                .collect();
            let shown = Type::Enum(owner.clone()).to_string();
            let message = Message::with(
                message!("{0} isn't a variant of {1}"),
                [quote(&variant.text), quote(&shown)],
            );
            let diagnostic = Self::did_you_mean(
                "unknown-variant",
                variant.span,
                &variant.text,
                &candidates,
                message.clone(),
                message,
            );
            self.report(diagnostic);
            return Pat::Any;
        }
        self.out
            .pattern_variants
            .insert(pattern.span.start, owner.clone());
        Pat::Variant(variant.text.clone())
    }

    fn record_pattern(
        &mut self,
        pattern: &Pattern,
        ty: &Type,
        fields: &[(crate::tariff::syntax::ast::Name, Pattern)],
    ) -> Pat {
        let inner = ty.unwrapped().clone();
        let Type::Record(record) = &inner else {
            self.pattern_error(pattern, ty, Message::new(message!("a record")));
            return Pat::Any;
        };
        let mut out = Vec::with_capacity(fields.len());
        for (name, sub) in fields {
            if let Some(field) = record.field(&name.text) {
                let field_ty = field.ty.clone();
                let mut field_line = Line { unit: None };
                out.push((
                    name.text.clone(),
                    self.pattern(sub, &field_ty, false, &mut field_line),
                ));
            } else {
                let candidates: Vec<String> =
                    record.fields.iter().map(|f| f.name.clone()).collect();
                let message = Message::with(
                    message!("{0} has no field {1}"),
                    [quote(&inner.to_string()), quote(&name.text)],
                );
                let diagnostic = Self::did_you_mean(
                    "unknown-field",
                    name.span,
                    &name.text,
                    &candidates,
                    message.clone(),
                    message,
                );
                self.report(diagnostic);
            }
        }
        Pat::Record(out)
    }

    fn or_pattern(
        &mut self,
        ty: &Type,
        alternatives: &[Pattern],
        none_covered: bool,
        line: &mut Line,
    ) -> Pat {
        let mut out = Vec::with_capacity(alternatives.len());
        for alternative in alternatives {
            if let PatternKind::Binding(name) = &alternative.kind {
                self.report(Diagnostic::error(
                            "pattern-binding",
                            name.span,
                            Message::new(message!("A name can't be bound in one alternative of `|`; bind the whole value instead")),
                        ));
                out.push(Pat::Any);
                continue;
            }
            out.push(self.pattern(alternative, ty, none_covered, line));
        }
        Pat::Or(out)
    }

    /// Declares the names a pattern binds, as unknown, when its scrutinee did not check.
    fn bind_all(&mut self, pattern: &Pattern) {
        match &pattern.kind {
            PatternKind::Binding(name) => {
                self.declare(name, LocalKind::Bound, Type::Error);
            }
            PatternKind::Record(fields) => {
                for (_, sub) in fields {
                    self.bind_all(sub);
                }
            }
            PatternKind::Literal(expr) => {
                self.expr(expr, None);
            }
            _ => {}
        }
    }

    fn pattern_error(&mut self, pattern: &Pattern, ty: &Type, what: Message) {
        self.report(Diagnostic::error(
            "type-mismatch",
            pattern.span,
            Message::with(
                message!("The value matched is {0}, so it can never be {1}"),
                [Arg::from(words(ty)), Arg::from(what)],
            ),
        ));
    }

    fn literal_pattern(
        &mut self,
        pattern: &Pattern,
        expr: &Expr,
        inner: &Type,
        line: &mut Line,
    ) -> Pat {
        let actual = self.expr(expr, Some(inner));
        if !same_kind(inner, &actual) {
            self.pattern_error(
                pattern,
                inner,
                Message::with(message!("{0}"), [quote_span(self.source, expr.span)]),
            );
            return Pat::Any;
        }
        match &expr.kind {
            ExprKind::Bool(value) => Pat::Bool(*value),
            ExprKind::Text(text) => text.literal().map_or(Pat::Any, Pat::Literal),
            _ => match self.bound(expr, line, true) {
                Some(point) => Pat::Interval(Interval {
                    lo: Some(point.clone()),
                    hi: Some(point),
                }),
                None => Pat::Any,
            },
        }
    }

    fn range_pattern(
        &mut self,
        pattern: &Pattern,
        start: Option<&Expr>,
        end: Option<&Expr>,
        inclusive: bool,
        inner: &Type,
        line: &mut Line,
    ) -> Pat {
        if !super::operators::ranged(inner) {
            for part in [start, end].into_iter().flatten() {
                self.expr(part, None);
            }
            self.unranged_pattern(pattern, inner);
            return Pat::Any;
        }
        if let Type::Enum(enumeration) = inner {
            return self.variant_range(pattern, (start, end), inclusive, enumeration);
        }
        if let (Some(start), Some(end)) = (start, end) {
            self.text_range_ends(start, end);
        }
        let mut bounds = [None, None];
        for (index, part) in [start, end].into_iter().enumerate() {
            let Some(part) = part else {
                continue;
            };
            let actual = self.expr(part, Some(inner));
            if !same_kind(inner, &actual) {
                self.pattern_error(
                    pattern,
                    inner,
                    Message::with(message!("{0}"), [quote_span(self.source, part.span)]),
                );
                return Pat::Any;
            }
            let Some(bound) = self.bound(part, line, index == 0 || inclusive) else {
                return Pat::Any;
            };
            if let Some(slot) = bounds.get_mut(index) {
                *slot = Some(bound);
            }
        }
        let [lo, hi] = bounds;
        if let (Some(lo), Some(hi)) = (&lo, &hi)
            && (lo.value > hi.value || (lo.value == hi.value && !inclusive))
        {
            return self.empty_range(pattern);
        }
        Pat::Interval(Interval { lo, hi })
    }

    /// Warns when a text range's literal ends have different lengths.
    pub fn text_range_ends(&mut self, start: &Expr, end: &Expr) {
        if let Some(diagnostic) = super::operators::text_range(start, end) {
            self.report(diagnostic);
        }
    }

    /// A pattern range on a type that has none: an enum that isn't ordered names the enums that
    /// are; anything else says it can't be a range.
    fn unranged_pattern(&mut self, pattern: &Pattern, inner: &Type) {
        if let Type::Enum(enumeration) = inner {
            let listed: Vec<&str> = enumeration
                .variants
                .iter()
                .take(2)
                .map(|(name, _)| name.as_str())
                .collect();
            self.report(Diagnostic::error(
                "type-mismatch",
                pattern.span,
                Message::with(
                    message!(
                        "The value matched is {0}, which has no order, so it can't be matched by a range: only {1} order their variants. List the variants instead, as in {2}"
                    ),
                    [
                        Arg::from(words(inner)),
                        Arg::from(super::operators::ordered_enums()),
                        quote(&listed.join(" | ")),
                    ],
                ),
            ));
            return;
        }
        self.pattern_error(pattern, inner, Message::new(message!("a range")));
    }

    /// `Mon..=Fri` on an ordered enum: the variants from the start to the end, in declaration
    /// order, as alternatives.
    fn variant_range(
        &mut self,
        pattern: &Pattern,
        (start, end): (Option<&Expr>, Option<&Expr>),
        inclusive: bool,
        enumeration: &Arc<EnumType>,
    ) -> Pat {
        let inner = Type::Enum(enumeration.clone());
        let mut ranks = [Some(0), enumeration.variants.len().checked_sub(1)];
        for (index, part) in [start, end].into_iter().enumerate() {
            let Some(part) = part else {
                continue;
            };
            let actual = self.expr(part, Some(&inner));
            if !same_kind(&inner, &actual) {
                self.pattern_error(
                    pattern,
                    &inner,
                    Message::with(message!("{0}"), [quote_span(self.source, part.span)]),
                );
                return Pat::Any;
            }
            let ExprKind::Path { head, member } = &part.kind else {
                self.pattern_error(
                    pattern,
                    &inner,
                    Message::with(message!("{0}"), [quote_span(self.source, part.span)]),
                );
                return Pat::Any;
            };
            let variant = member.as_ref().unwrap_or(head);
            let Some(rank) = enumeration.rank(&variant.text) else {
                return Pat::Any;
            };
            let rank = if index == 1 && !inclusive {
                rank.checked_sub(1)
            } else {
                Some(rank)
            };
            if let Some(slot) = ranks.get_mut(index) {
                *slot = rank;
            }
        }
        let [Some(low), Some(high)] = ranks else {
            return self.empty_range(pattern);
        };
        if low > high {
            return self.empty_range(pattern);
        }
        Pat::Or(
            enumeration
                .variants
                .get(low..=high)
                .unwrap_or_default()
                .iter()
                .map(|(name, _)| Pat::Variant(name.clone()))
                .collect(),
        )
    }

    fn empty_range(&mut self, pattern: &Pattern) -> Pat {
        self.report(Diagnostic::error(
            "empty-range",
            pattern.span,
            Message::new(message!(
                "This range is empty: its start isn't below its end"
            )),
        ));
        Pat::Any
    }

    /// A literal's position on the match's line, in its first unit.
    fn bound(&self, expr: &Expr, line: &mut Line, inclusive: bool) -> Option<Bound> {
        let value = Self::line_value(expr, line)?;
        Some(Bound {
            value,
            inclusive,
            label: self.text(expr.span).trim().to_owned(),
        })
    }

    fn line_value(expr: &Expr, line: &mut Line) -> Option<Key> {
        if let ExprKind::Text(text) = &expr.kind {
            return text.literal().map(Key::Text);
        }
        Self::number_value(expr, line).map(Key::Number)
    }

    fn number_value(expr: &Expr, line: &mut Line) -> Option<Decimal> {
        match &expr.kind {
            ExprKind::Number { value, unit, .. } => {
                let unit = unit
                    .as_ref()
                    .and_then(|unit| Unit::parse(&unit.text()).ok());
                match unit {
                    Some(unit) if !unit.dim().is_none() => in_line(*value, &unit, line),
                    _ => Some(*value),
                }
            }
            ExprKind::Money { value, per } => {
                let text = per
                    .as_ref()
                    .map_or_else(|| String::from("$"), |per| format!("$/{}", per.text()));
                in_line(*value, &Unit::parse(&text).ok()?, line)
            }
            ExprKind::Percent(value) => Some(*value),
            ExprKind::Date(date) => Some(Decimal::from(date_ordinal(*date))),
            ExprKind::DateTime(at) => Some(Decimal::from(date_time_ordinal(*at))),
            ExprKind::Time(time) => Some(Decimal::from(time.seconds())),
            ExprKind::Unary {
                op: UnaryOp::Neg,
                operand,
                ..
            } => Self::number_value(operand, line).map(|value| -value),
            _ => None,
        }
    }
}

fn in_line(value: Decimal, unit: &Unit, line: &mut Line) -> Option<Decimal> {
    if let Some(first) = &line.unit {
        convert(value, unit, first).ok()
    } else {
        line.unit = Some(unit.clone());
        Some(value)
    }
}

/// Whether a pattern literal's type fits the value matched: the same type, an `Int` for a
/// `Number`, or a quantity of the same dimension.
fn same_kind(expected: &Type, actual: &Type) -> bool {
    matches!(actual, Type::Error)
        || expected.accepts(actual)
        || expected.unwrapped().accepts(actual)
}

fn or_has_none(pat: &Pat) -> bool {
    match pat {
        Pat::None => true,
        Pat::Or(alternatives) => alternatives.iter().any(or_has_none),
        _ => false,
    }
}

fn or_has_any(pat: &Pat) -> bool {
    match pat {
        Pat::Any => true,
        Pat::Or(alternatives) => alternatives.iter().any(or_has_any),
        _ => false,
    }
}

/// The space of values a type has, for coverage.
#[must_use]
pub fn space_of(ty: &Type) -> Space {
    match ty {
        Type::Optional(inner) => Space::Optional(Box::new(space_of(inner))),
        Type::Bool => Space::Bool,
        Type::Enum(enumeration) => Space::Enum(
            enumeration
                .variants
                .iter()
                .map(|(name, _)| name.clone())
                .collect(),
        ),
        Type::Int | Type::Date => Space::Line {
            discrete: true,
            domain: None,
        },
        Type::Time => Space::Line {
            discrete: true,
            domain: Some(Domain {
                min: Decimal::ZERO,
                max: Decimal::from(86_399),
            }),
        },
        Type::Number | Type::Percent | Type::Quantity(_) | Type::DateTime => Space::Line {
            discrete: false,
            domain: None,
        },
        Type::Record(record) => Space::Record(
            record
                .fields
                .iter()
                .map(|field| (field.name.clone(), space_of(&field.ty)))
                .collect(),
        ),
        Type::Text => Space::Text,
        Type::List(_) => Space::Open,
        Type::Never | Type::Error => Space::Unknown,
    }
}
