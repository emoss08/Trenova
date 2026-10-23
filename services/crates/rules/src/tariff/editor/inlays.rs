//! `tariff.inlays`: what the editor writes between the author's tokens without changing them.
//!
//! - **type** hints: the inferred type after a `let` or `var` name without an annotation, a loop
//!   variable, a lambda parameter and a pattern or `if let` binding (`: Distance`), and the
//!   inferred return type of an `=`-bodied function without `->` (`-> Money`), all from the
//!   checker's hints, and a property's `vary` name (from the values it varies over);
//! - **parameter** hints: the parameter's name before a positional argument whose meaning is
//!   not obvious: any argument of a call with two or more parameters, or a literal, unless the
//!   argument already names the parameter (`detention(dwell, …)`) or is a lambda;
//! - **value** hints, when the last preview's trace is given: each binding's value at the end of
//!   its line (the last value, with `×n` when it ran several times).
//!
//! Only hints inside the requested range are returned, in source order.

use super::complete::callee_of;
use super::cursor::Scopes;
use super::describe::{flat, type_text};
use super::names::parameters;
use super::session::Session;
use super::values::Values;
use super::vary::{varied, vary_type};
use crate::span::Span;
use crate::tariff::check::describe::snippet;
use crate::tariff::check::{Callee, HintKind};
use crate::tariff::syntax::ast::{Expr, ExprKind};
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::types::Type;
use std::collections::BTreeMap;

/// What a hint shows.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum Kind {
    /// An inferred type.
    Type,
    /// A parameter's name.
    Parameter,
    /// A previewed value.
    Value,
}

impl Kind {
    /// The kind's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Type => "type",
            Self::Parameter => "parameter",
            Self::Value => "value",
        }
    }
}

/// One hint.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord)]
pub struct Inlay {
    /// Where it goes, in UTF-16 code units of the root file.
    pub offset: u32,
    /// What it shows.
    pub kind: Kind,
    /// The text.
    pub label: String,
}

/// The hints of the root file inside byte range `range`.
#[must_use]
pub fn inlays(session: &Session<'_>, range: Span, values: &Values) -> Vec<Inlay> {
    let utf16 = &session.root().utf16;
    let mut out = Vec::new();
    types(session, range, &mut out);
    varied_types(session, range, &mut out);
    let scopes = Scopes::of(session.source(), session.file());
    session.tick(scopes.exprs.len());
    for call in &scopes.exprs {
        if call.span.end < range.start || call.span.start > range.end {
            continue;
        }
        if let ExprKind::Call { args, .. } = &call.kind {
            arguments(session, call, args, range, &mut out);
        }
    }
    let mut hints: Vec<Inlay> = out
        .into_iter()
        .map(|(offset, kind, label)| Inlay {
            offset: utf16.to_utf16(offset),
            kind,
            label,
        })
        .collect();
    if !values.is_empty() {
        let from = utf16.to_utf16(range.start);
        let to = utf16.to_utf16(range.end);
        hints.extend(value_hints(session, values, from, to));
    }
    hints.sort();
    hints.dedup();
    hints
}

fn types(session: &Session<'_>, range: Span, out: &mut Vec<(u32, Kind, String)>) {
    let Some(analysis) = session.analysis.module(ModuleId::ROOT) else {
        return;
    };
    session.tick(analysis.hints.len());
    for hint in &analysis.hints {
        if hint.span.end < range.start || hint.span.end > range.end {
            continue;
        }
        if matches!(hint.ty, Type::Error | Type::Never) {
            continue;
        }
        let at = match hint.kind {
            HintKind::Return => after_parameters(session, hint.span.end),
            _ => hint.span.end,
        };
        let label = match hint.kind {
            HintKind::Return => format!("-> {}", type_text(&hint.ty)),
            HintKind::Binding | HintKind::Loop | HintKind::Lambda | HintKind::Pattern => {
                format!(": {}", type_text(&hint.ty))
            }
        };
        out.push((at, Kind::Type, label));
    }
}

/// The type of each `vary` name, after the name: the checker's hints do not reach a property's
/// own names.
fn varied_types(session: &Session<'_>, range: Span, out: &mut Vec<(u32, Kind, String)>) {
    for varied in varied(session.file()) {
        let at = varied.name.span.end;
        if at < range.start || at > range.end {
            continue;
        }
        let Some(ty) = varied
            .values
            .and_then(|values| vary_type(session, values))
            .filter(|ty| !matches!(ty, Type::Error | Type::Never))
        else {
            continue;
        };
        out.push((at, Kind::Type, format!(": {}", type_text(&ty))));
    }
}

/// Where the `)` closing the parameter list that starts after `from` ends.
fn after_parameters(session: &Session<'_>, from: u32) -> u32 {
    let tokens = &session.file().tokens;
    let mut depth = 0usize;
    for token in tokens
        .iter()
        .skip(tokens.partition_point(|token| token.span.start < from))
    {
        match snippet(session.source(), token.span) {
            "(" => depth += 1,
            ")" => {
                depth = depth.saturating_sub(1);
                if depth == 0 {
                    return token.span.end;
                }
            }
            _ => {}
        }
    }
    from
}

/// The parameter names before a call's positional arguments.
fn arguments(
    session: &Session<'_>,
    call: &Expr,
    args: &[crate::tariff::syntax::ast::Arg],
    range: Span,
    out: &mut Vec<(u32, Kind, String)>,
) {
    if args.iter().all(|arg| arg.name.is_some()) {
        return;
    }
    let Some(callee) = callee_of(session, call) else {
        return;
    };
    let names: Vec<String> = match &callee {
        Callee::Function { .. } | Callee::Constructor(_) => parameters(session, call, &callee)
            .into_iter()
            .map(|param| param.name)
            .collect(),
        Callee::Method(method) => method
            .params
            .iter()
            .map(|param| param.name.to_owned())
            .collect(),
        Callee::Conversion(_) | Callee::ChargeLookup | Callee::Table { .. } => return,
    };
    for (arg, name) in args.iter().take_while(|arg| arg.name.is_none()).zip(&names) {
        if arg.value.span.start < range.start || arg.value.span.start > range.end {
            continue;
        }
        if !needs_name(session, &arg.value, name, names.len()) {
            continue;
        }
        out.push((arg.value.span.start, Kind::Parameter, format!("{name}:")));
    }
}

/// Whether a positional argument's meaning is not obvious without its parameter's name.
fn needs_name(session: &Session<'_>, value: &Expr, name: &str, count: usize) -> bool {
    if matches!(value.kind, ExprKind::Lambda { .. } | ExprKind::Missing) {
        return false;
    }
    let written = flat(snippet(session.source(), value.span)).to_lowercase();
    let last = written
        .rsplit(|c: char| !(c.is_ascii_alphanumeric() || c == '_'))
        .find(|part| !part.is_empty())
        .unwrap_or_default();
    let lowered = name.to_lowercase();
    if last == lowered || last.ends_with(&lowered) {
        return false;
    }
    let literal = matches!(
        value.kind,
        ExprKind::Number { .. }
            | ExprKind::Money { .. }
            | ExprKind::Percent(_)
            | ExprKind::Text(_)
            | ExprKind::Bool(_)
            | ExprKind::None
            | ExprKind::Date(_)
            | ExprKind::DateTime(_)
    );
    count >= 2 || literal
}

/// Each binding's last previewed value, at the end of the line it is bound on.
fn value_hints(session: &Session<'_>, values: &Values, from: u32, to: u32) -> Vec<Inlay> {
    let utf16 = &session.root().utf16;
    let source = session.source();
    let line_ends: Vec<u32> = source
        .match_indices('\n')
        .map(|(index, _)| utf16.to_utf16(u32::try_from(index).unwrap_or(u32::MAX)))
        .chain(std::iter::once(
            utf16.to_utf16(u32::try_from(source.len()).unwrap_or(u32::MAX)),
        ))
        .collect();
    session.tick(line_ends.len());
    let mut last: BTreeMap<(u32, String), (String, usize)> = BTreeMap::new();
    for traced in values.events() {
        if traced.assign || traced.range.from < from || traced.range.from > to {
            continue;
        }
        let entry = last
            .entry((traced.range.from, traced.name.clone()))
            .or_insert_with(|| (String::new(), 0));
        entry.0.clone_from(&traced.display);
        entry.1 += 1;
    }
    last.into_iter()
        .filter_map(|((start, _), (display, times))| {
            let line = line_ends.partition_point(|end| *end < start);
            let offset = *line_ends.get(line)?;
            let label = if times > 1 {
                format!("{display} ×{times}")
            } else {
                display
            };
            Some(Inlay {
                offset,
                kind: Kind::Value,
                label,
            })
        })
        .collect()
}
