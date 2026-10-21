//! The lexer, the parser and the formatter.
//!
//! [`parse`] reads a formula into a [`Program`]: its statements as a tree, every lexical and
//! grammatical problem as a diagnostic, and the tokens and comments. It never fails and never
//! panics, whatever the text; a malformed formula yields a tree with `Missing` nodes and the
//! diagnostics that explain them, so the checker can still report everything else and the editor
//! can underline each problem where it is.
//!
//! The parser bounds its work: a formula longer than [`MAX_SOURCE_UNITS`] UTF-16 code units is
//! not read at all, and one with more than [`MAX_NODES`] nodes or nested more than [`MAX_DEPTH`]
//! levels stops at the limit with a diagnostic. So every tree is at most `MAX_DEPTH` deep, and
//! the checker, the formatter and the evaluator may recurse over it within WebAssembly's stack.

pub mod ast;
mod format;
pub mod lexer;
mod parser;
mod statements;
mod unicode;

pub use ast::Program;
pub use ast::walk;
pub use format::DEFAULT_WIDTH;
pub use format::FormatOutcome;
pub use format::FormatRefusal;
pub use format::format;
pub use statements::StatementInfo;
pub use statements::StatementKindInfo;
pub use statements::split_statements;

use crate::message;
use crate::message::Arg;
use crate::message::Message;
use crate::span::Span;
use ast::SyntaxCode;
use ast::SyntaxDiagnostic;
use ast::Token;
use ast::TokenKind;

/// The longest formula the parser reads, in UTF-16 code units (what the editor counts).
pub const MAX_SOURCE_UNITS: usize = 32_768;

/// The most nodes a formula's tree may have, as the old engine allowed.
pub const MAX_NODES: usize = 1_000;

/// How deep a formula may nest: brackets, calls, operands and `else if` chains together.
pub const MAX_DEPTH: usize = 128;

/// Parses a formula. `word_operators` are the word operators the schema publishes beyond the
/// built-in `startsWith`, `endsWith`, `contains` and `matches`; they parse at comparison
/// precedence.
#[must_use]
pub fn parse(source: &str, word_operators: &[&str]) -> Program {
    let end = u32::try_from(source.len()).unwrap_or(u32::MAX);
    if let Some(cut) = past_limit(source) {
        return Program {
            statements: Vec::new(),
            diagnostics: vec![SyntaxDiagnostic {
                span: Span::new(cut, end),
                code: SyntaxCode::Limit,
                message: Message::with(
                    message!("This formula is too long: a formula holds at most {0} characters"),
                    [Arg::from(MAX_SOURCE_UNITS)],
                ),
                fix: None,
            }],
            tokens: vec![Token {
                kind: TokenKind::Eof,
                span: Span::new(end, end),
            }],
            comments: Vec::new(),
            empty: false,
        };
    }
    let tokenized = lexer::tokenize(source, word_operators);
    if tokenized.tokens.len() == 1 {
        return Program {
            statements: Vec::new(),
            diagnostics: tokenized.diagnostics,
            tokens: tokenized.tokens,
            comments: tokenized.comments,
            empty: true,
        };
    }
    let mut parser = parser::Parser::new(source, &tokenized.tokens, word_operators);
    let statements = parser.parse_program();
    let mut diagnostics = tokenized.diagnostics;
    diagnostics.append(&mut parser.diagnostics);
    diagnostics.sort_by_key(|diagnostic| diagnostic.span.start);
    Program {
        statements,
        diagnostics,
        tokens: tokenized.tokens,
        comments: tokenized.comments,
        empty: false,
    }
}

/// The byte offset where a formula passes [`MAX_SOURCE_UNITS`], when it does.
fn past_limit(source: &str) -> Option<u32> {
    if source.len() <= MAX_SOURCE_UNITS {
        return None;
    }
    let mut units = 0usize;
    for (offset, character) in source.char_indices() {
        units += character.len_utf16();
        if units > MAX_SOURCE_UNITS {
            return Some(u32::try_from(offset).unwrap_or(u32::MAX));
        }
    }
    None
}
