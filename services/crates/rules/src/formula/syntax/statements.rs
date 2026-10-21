//! A formula's statements as the rate trace shows them: each `let` line and each value, with its
//! code and where it is.
//!
//! The trace evaluates and labels each statement on its own, so a statement's span runs from its
//! first token to the last token of its value: the `;` that ends it, the spaces and the comments
//! around it are not part of it. Unlike the old engine's splitter, this works on a formula that
//! does not parse too (each statement the tolerant parser found is listed), and a value line in the
//! middle of the formula is its own statement rather than part of the last.

use super::ast::Program;
use super::ast::StatementKind;
use crate::span::Span;

/// What kind of statement a [`StatementInfo`] is.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum StatementKindInfo {
    /// `let name = value`.
    Let,
    /// A value; the last is the formula's result.
    Result,
}

/// One statement of a formula.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StatementInfo {
    /// Its position among the statements, from 0.
    pub index: usize,
    /// A `let` or a value.
    pub kind: StatementKindInfo,
    /// The bound name of a `let`; `None` for a value or a `let` whose name is not written yet.
    pub name: Option<String>,
    /// Its text, from its first token to the last token of its value.
    pub code: String,
    /// Where that text is.
    pub span: Span,
}

/// Lists a parsed formula's statements.
#[must_use]
pub fn split_statements(program: &Program, source: &str) -> Vec<StatementInfo> {
    program
        .statements
        .iter()
        .enumerate()
        .map(|(index, statement)| {
            let code = source
                .get(statement.span.start as usize..statement.span.end as usize)
                .unwrap_or_default()
                .trim()
                .to_owned();
            let (kind, name) = match &statement.kind {
                StatementKind::Let(name) => (
                    StatementKindInfo::Let,
                    (!name.text.is_empty()).then(|| name.text.clone()),
                ),
                StatementKind::Result => (StatementKindInfo::Result, None),
            };
            StatementInfo {
                index,
                kind,
                name,
                code,
                span: statement.span,
            }
        })
        .collect()
}
