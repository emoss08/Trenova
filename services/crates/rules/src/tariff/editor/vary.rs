//! A property's `vary` statements as the editor reads them: the names they declare and the
//! type each name takes, worked out from the values it varies over (the checker's type of
//! them when it has one, else the literals' own), so hover and inlays have it even where the
//! checker did not reach.

use super::cursor::{statements, vary_name};
use super::literals::literal_type;
use super::session::Session;
use trenova_module_kit::span::Span;
use crate::tariff::syntax::ast::{Expr, ExprKind, File, Name, Stmt, StmtKind};
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::types::Type;

/// One `vary` that declares a name: the statement, the name and its values.
#[derive(Debug, Clone, Copy)]
pub struct Varied<'f> {
    /// The statement.
    pub stmt: &'f Stmt,
    /// The name declared.
    pub name: &'f Name,
    /// `in values`, when written.
    pub values: Option<&'f Expr>,
}

/// Every `vary` of the file that declares a name, in source order.
#[must_use]
pub fn varied(file: &File) -> Vec<Varied<'_>> {
    let mut out = Vec::new();
    statements(file, &mut |stmt| {
        if let StmtKind::Vary { target, values } = &stmt.kind
            && let Some(name) = vary_name(target)
        {
            out.push(Varied {
                stmt,
                name,
                values: values.as_ref(),
            });
        }
    });
    out
}

/// The `vary` that declares the name at `span`.
#[must_use]
pub fn varied_at(file: &File, span: Span) -> Option<Varied<'_>> {
    varied(file)
        .into_iter()
        .find(|varied| varied.name.span == span)
}

/// The type a varied name takes: a range's ends', a list's items'.
#[must_use]
pub fn vary_type(session: &Session<'_>, values: &Expr) -> Option<Type> {
    let checked = |expr: &Expr| {
        session
            .analysis
            .type_of(ModuleId::ROOT, expr.id)
            .filter(|ty| !matches!(ty, Type::Error | Type::Never))
            .cloned()
    };
    let value = |expr: &Expr| checked(expr).or_else(|| literal_type(session, expr));
    match &values.kind {
        ExprKind::Range { start, end, .. } => {
            start.iter().chain(end.iter()).find_map(|part| value(part))
        }
        ExprKind::List(items) => match checked(values) {
            Some(Type::List(item)) => Some((*item).clone()),
            _ => items.iter().find_map(value),
        },
        _ => match checked(values) {
            Some(Type::List(item)) => Some((*item).clone()),
            _ => None,
        },
    }
}
