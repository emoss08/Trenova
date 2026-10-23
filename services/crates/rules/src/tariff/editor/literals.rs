//! Literals as the editor reads them: which expressions are literals a contract could set, the
//! type each one has (worked out from the literal itself, as the checker types it, so a literal
//! the checker did not reach still has one) and its value, for telling equal literals apart
//! from merely similar ones.
//!
//! A literal is a number with or without a unit, money, a rate, a percent, text without
//! interpolation, a date, date-time or time of day, `true`/`false`, an enum variant (bare or qualified), or
//! the negation of a number, money or percent. `none`, lists and records are not literals here.

use super::names::enums;
use super::session::Session;
use crate::tariff::check::consts::{Folder, Value};
use crate::tariff::syntax::ast::{Expr, ExprKind, UnaryOp};
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::types::Type;
use crate::tariff::units::Unit;

/// Whether an expression is written as a literal (its negation included).
#[must_use]
pub fn is_literal(expr: &Expr) -> bool {
    match &expr.kind {
        ExprKind::Number { .. }
        | ExprKind::Money { .. }
        | ExprKind::Percent(_)
        | ExprKind::Date(_)
        | ExprKind::DateTime(_)
        | ExprKind::Time(_)
        | ExprKind::Bool(_)
        | ExprKind::Path { .. } => true,
        ExprKind::Text(text) => text.literal().is_some(),
        ExprKind::Unary {
            op: UnaryOp::Neg,
            operand,
            ..
        } => matches!(
            operand.kind,
            ExprKind::Number { .. } | ExprKind::Money { .. } | ExprKind::Percent(_)
        ),
        _ => false,
    }
}

/// The type of a literal: `$2.35/mi` is `Money/mi`, `48 ft` is `Length`, `2` is `Int`, `2.5`
/// is `Number`, a variant is its enum. `None` for an expression that is no literal, a unit the
/// language does not have, or a variant of no enum (or of several, written bare).
#[must_use]
pub fn literal_type(session: &Session<'_>, expr: &Expr) -> Option<Type> {
    match &expr.kind {
        ExprKind::Number { integer, unit, .. } => {
            let plain = if *integer { Type::Int } else { Type::Number };
            let Some(unit) = unit else {
                return Some(plain);
            };
            let parsed = Unit::parse(&unit.text()).ok()?;
            Some(if parsed.dim().is_none() {
                plain
            } else {
                Type::quantity_in(&parsed)
            })
        }
        ExprKind::Money { per, .. } => {
            let text = per
                .as_ref()
                .map_or_else(|| String::from("$"), |per| format!("$/{}", per.text()));
            Unit::parse(&text).ok().map(|unit| Type::quantity_in(&unit))
        }
        ExprKind::Percent(_) => Some(Type::Percent),
        ExprKind::Text(text) => text.literal().map(|_| Type::Text),
        ExprKind::Date(_) => Some(Type::Date),
        ExprKind::DateTime(_) => Some(Type::DateTime),
        ExprKind::Time(_) => Some(Type::Time),
        ExprKind::Bool(_) => Some(Type::Bool),
        ExprKind::Path { head, member } => variant_type(session, expr, &head.text, member.as_ref()),
        ExprKind::Unary {
            op: UnaryOp::Neg,
            operand,
            ..
        } if is_literal(expr) => literal_type(session, operand),
        _ => None,
    }
}

/// The enum of a variant written `head` or `head.member`: the checker's, else the one enum
/// the file can name that has it.
fn variant_type(
    session: &Session<'_>,
    expr: &Expr,
    head: &str,
    member: Option<&crate::tariff::syntax::ast::Name>,
) -> Option<Type> {
    if let Some(found) = session
        .analysis
        .module(ModuleId::ROOT)
        .and_then(|analysis| analysis.variants.get(&expr.id))
    {
        return Some(Type::Enum(found.clone()));
    }
    let visible = enums(session);
    let matching: Vec<_> = match member {
        Some(member) => visible
            .iter()
            .filter(|(name, enumeration)| name == head && enumeration.has(&member.text))
            .collect(),
        None => visible
            .iter()
            .filter(|(_, enumeration)| enumeration.has(head))
            .collect(),
    };
    match matching.as_slice() {
        [(_, enumeration)] => Some(Type::Enum(enumeration.clone())),
        _ => None,
    }
}

/// A literal's value, compared by value (`$2.35/mi` equals `$2.350/mi`, `18%` equals
/// `18.0%`) in the unit it is written in, so `1 cwt` and `100 lb` are not the same literal.
#[must_use]
pub fn literal_value(expr: &Expr, ty: &Type) -> Option<Value> {
    let type_of = |_| Some(ty.clone());
    let constant = |_: &Expr| None;
    let folder = Folder {
        constant: &constant,
        type_of: &type_of,
    };
    folder.fold(expr).ok()
}
