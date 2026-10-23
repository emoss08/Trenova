//! The file's choices (`best` and `worst`), read from the syntax tree: their labels for
//! completion, hover and definition, and the `chosen("…")` calls of tests that name them.
//!
//! A choice is named by its label, which is text: `chosen("Linehaul")` in a test reads the
//! option the last evaluation of the choice labelled `Linehaul` kept. The checker owns whether
//! a label is unique and a `chosen` names a real one; the editor only needs to find them.

use super::cursor::{Scopes, statements};
use super::session::Session;
use crate::span::Span;
use crate::tariff::syntax::ast::{ChoiceOption, Choose, Expr, ExprKind, File, StmtKind};
use crate::tariff::syntax::program::ModuleId;

/// The test function that reads a choice's outcome.
pub const CHOSEN: &str = "chosen";

/// What `chosen` gives, for completion and signature help.
pub const CHOSEN_DOC: &str = crate::message!(
    "The label of the option the last evaluation of the `best` or `worst` with this label kept, or `none` when it was not reached or every option refused."
);

/// Every choice of a file, in source order, nested ones included.
#[must_use]
pub fn choices(file: &File) -> Vec<&Choose> {
    let mut out = Vec::new();
    statements(file, &mut |stmt| {
        let written = match &stmt.kind {
            StmtKind::Choose(choose) => Some(choose),
            StmtKind::Let(binding) | StmtKind::Var(binding) => valued(&binding.value),
            StmtKind::Assign { value, .. } => valued(value),
            _ => None,
        };
        if let Some(choose) = written
            && !choose.label.text.is_empty()
        {
            out.push(choose);
        }
    });
    out
}

/// The choice a binding's or assignment's value is, when it is one.
fn valued(value: &Expr) -> Option<&Choose> {
    match &value.kind {
        ExprKind::Choose(choose) => Some(choose),
        _ => None,
    }
}

/// The first choice of a file with a label.
#[must_use]
pub fn choice_named<'f>(file: &'f File, label: &str) -> Option<&'f Choose> {
    choices(file)
        .into_iter()
        .find(|choose| choose.label.text == label)
}

/// A label the position is on.
#[derive(Debug, Clone, Copy)]
pub enum Label<'f> {
    /// A choice's own label.
    Choice(&'f Choose),
    /// One of its options' labels.
    OptionOf(&'f Choose, &'f ChoiceOption),
}

/// The choice or option label written at byte `offset`.
#[must_use]
pub fn label_at(file: &File, offset: u32) -> Option<Label<'_>> {
    let holds = |span: Span| span.start <= offset && offset <= span.end && span.start < span.end;
    choices(file).into_iter().find_map(|choose| {
        if holds(choose.label.span) {
            return Some(Label::Choice(choose));
        }
        choose
            .options
            .iter()
            .find(|option| holds(option.label.span))
            .map(|option| Label::OptionOf(choose, option))
    })
}

/// Whether a call is a test's `chosen(…)`: written `chosen`, with no function of that name in
/// scope.
#[must_use]
pub fn is_chosen(session: &Session<'_>, call: &Expr) -> bool {
    matches!(&call.kind, ExprKind::Call { callee, .. }
        if matches!(&callee.kind, ExprKind::Name(name) if name.text == CHOSEN)
            && session.global(ModuleId::ROOT, CHOSEN).is_none())
}

/// The label a `chosen("…")` call's text at byte `offset` names, with the text's span.
#[must_use]
pub fn chosen_at(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    offset: u32,
) -> Option<(Span, String)> {
    scopes.exprs.iter().find_map(|expr| {
        let ExprKind::Call { args, .. } = &expr.kind else {
            return None;
        };
        if !is_chosen(session, expr) {
            return None;
        }
        let argument = args.first()?;
        let ExprKind::Text(text) = &argument.value.kind else {
            return None;
        };
        let span = argument.value.span;
        (span.start <= offset && offset <= span.end)
            .then(|| text.literal().map(|label| (span, label)))
            .flatten()
    })
}
