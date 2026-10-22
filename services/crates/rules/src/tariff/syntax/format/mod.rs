//! The formatter: one canonical style for every Tariff file, like `gofmt`.
//!
//! - Two-space indentation; one space around binary operators and after `,` and `:`; none
//!   inside `( )` and `[ ]`; records written `{ a: 1 }`.
//! - At most one blank line kept between statements, items, arms, variants and fields; none at
//!   the start or end of a block.
//! - Numbers, units, money, dates and text are written as the author wrote them (digits,
//!   separators and escapes unchanged); a unit is one space after its number.
//! - Comments and doc comments stay where they are: a comment on its own line stays above what
//!   follows it, and a trailing comment stays at the end of its line. A comment the layout has
//!   no place for (inside an expression that does not break) moves to its own line after the
//!   statement, so no comment is ever lost.
//! - A line over [`WIDTH`] columns breaks its lists, arguments, records and long `+`/`-`,
//!   `and`, `or` and `??` chains one item per line, with a trailing comma in lists and before
//!   the operator in chains. `match` arms are not aligned.
//!
//! A file with errors is not formatted: [`format`] returns its errors instead. The result is
//! checked before it is returned: it must parse without errors, to the same tree (compared with
//! [`super::shape`]) and with the same comments.

mod doc;
mod exprs;
mod items;

use super::ast::File;
use super::parse;
use super::shape;
use super::token::TokenKind;
use crate::message;
use crate::message::Message;
use crate::span::Span;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::diagnostic::Severity;
use doc::Doc;

/// The column a formatted line stays within when it can be broken.
pub const WIDTH: usize = 100;

/// Formats `source`, or returns why it cannot be: its errors, or (should formatting ever fail
/// to keep the file's meaning) a `format` diagnostic.
pub fn format(source: &str) -> Result<String, Vec<Diagnostic>> {
    let file = parse(source);
    let errors: Vec<Diagnostic> = file
        .diagnostics
        .iter()
        .filter(|diagnostic| diagnostic.severity == Severity::Error)
        .cloned()
        .collect();
    if !errors.is_empty() {
        return Err(errors);
    }
    let formatted = format_file(source, &file);
    let reparsed = parse(&formatted);
    let comments = |file: &File, text: &str| -> Vec<String> {
        file.tokens
            .iter()
            .filter(|t| matches!(t.kind, TokenKind::Comment | TokenKind::DocComment))
            .map(|t| {
                text.get(t.span.start as usize..t.span.end as usize)
                    .unwrap_or_default()
                    .trim_end()
                    .to_owned()
            })
            .collect()
    };
    let kept = reparsed
        .diagnostics
        .iter()
        .all(|diagnostic| diagnostic.severity != Severity::Error)
        && shape::file(&reparsed) == shape::file(&file)
        && comments(&reparsed, &formatted) == comments(&file, source);
    if !kept {
        return Err(vec![Diagnostic::error(
            "format",
            Span::new(0, 0),
            Message::new(message!(
                "This file can't be formatted without changing it; move the comments inside expressions onto their own lines and try again"
            )),
        )]);
    }
    Ok(formatted)
}

/// One comment of the file, and whether it has been written yet.
struct Comment {
    span: Span,
    text: String,
    taken: bool,
}

/// Builds the layout of a parsed file, placing its comments as it goes.
pub(super) struct Formatter<'a> {
    source: &'a str,
    comments: Vec<Comment>,
    /// The first comment not yet written; every comment before it has been.
    cursor: usize,
    /// Where the last thing written (an element or a comment) ends in the source.
    last_end: u32,
}

fn format_file(source: &str, file: &File) -> String {
    let comments = file
        .tokens
        .iter()
        .filter(|t| matches!(t.kind, TokenKind::Comment | TokenKind::DocComment))
        .map(|t| Comment {
            span: t.span,
            text: source
                .get(t.span.start as usize..t.span.end as usize)
                .unwrap_or_default()
                .trim_end()
                .to_owned(),
            taken: false,
        })
        .collect();
    let mut formatter = Formatter {
        source,
        comments,
        cursor: 0,
        last_end: 0,
    };
    let layout = formatter.file(file);
    doc::render(&layout, WIDTH)
}

impl Formatter<'_> {
    fn slice(&self, span: Span) -> &str {
        self.source
            .get(span.start as usize..span.end as usize)
            .unwrap_or_default()
    }

    /// Whether the source has a blank line between `from` and `to`.
    fn blank_between(&self, from: u32, to: u32) -> bool {
        if to <= from {
            return false;
        }
        self.slice(Span::new(from, to))
            .split('\n')
            .skip(1)
            .collect::<Vec<_>>()
            .split_last()
            .is_some_and(|(_, middle)| middle.iter().any(|line| line.trim().is_empty()))
    }

    /// The comments written before `before`, each on its own line, ending with a newline.
    /// `blank` allows a blank line before the first of them, as the source has one.
    fn leading(&mut self, before: u32, mut blank: bool) -> Vec<Doc> {
        let mut out = Vec::new();
        let last_end = self.last_end;
        let mut previous = last_end;
        let mut index = self.cursor;
        while let Some(comment) = self.comments.get_mut(index) {
            if comment.span.start >= before {
                break;
            }
            index += 1;
            if comment.taken {
                continue;
            }
            comment.taken = true;
            let span = comment.span;
            let text = comment.text.clone();
            if blank && self.blank_between(previous, span.start) {
                out.push(Doc::HardLine);
            }
            out.push(Doc::Comment(text));
            out.push(Doc::HardLine);
            previous = span.end;
            blank = true;
        }
        self.advance();
        self.last_end = previous.max(last_end);
        out
    }

    /// Moves the cursor past the comments already written.
    fn advance(&mut self) {
        while self
            .comments
            .get(self.cursor)
            .is_some_and(|comment| comment.taken)
        {
            self.cursor += 1;
        }
    }

    /// The comment written after `end` on the same line, with nothing but a comma between,
    /// when there is one.
    fn trailing(&mut self, end: u32) -> Option<Doc> {
        let source = self.source;
        let comment = self
            .comments
            .iter_mut()
            .skip(self.cursor)
            .find(|comment| comment.span.start >= end)?;
        if comment.taken {
            return None;
        }
        let between = source
            .get(end as usize..comment.span.start as usize)
            .unwrap_or_default();
        if between
            .chars()
            .any(|c| !(c == ',' || c == ' ' || c == '\t' || c == '\r'))
        {
            return None;
        }
        comment.taken = true;
        let text = format!(" {}", comment.text);
        self.last_end = comment.span.end;
        self.advance();
        Some(Doc::Comment(text))
    }

    /// Every comment not yet written that starts before `before`, each on its own line, for
    /// the end of a block or of the file. Each is preceded by a newline.
    fn closing(&mut self, before: u32) -> Vec<Doc> {
        let mut out = Vec::new();
        let leading = self.leading(before, true);
        for doc in leading {
            if doc == Doc::HardLine {
                continue;
            }
            out.push(Doc::HardLine);
            out.push(doc);
        }
        out
    }

    /// Statement-like elements written one per line: each with its leading comments, a blank
    /// line kept before it when the source has one (never before the first), and its trailing
    /// comment. `blank_first` keeps a blank line before the first too (after the header).
    /// The result starts with a newline.
    fn vertical<T>(
        &mut self,
        elements: &[T],
        blank_first: bool,
        span: impl Fn(&T) -> Span,
        build: impl Fn(&mut Self, &T) -> Doc,
    ) -> Vec<Doc> {
        let mut out = Vec::new();
        for (index, element) in elements.iter().enumerate() {
            let span = span(element);
            out.push(Doc::HardLine);
            let first = index == 0 && !blank_first;
            let comments = self.leading(span.start, !first);
            let had_comments = !comments.is_empty();
            out.extend(comments);
            if (!first || had_comments) && self.blank_between(self.last_end, span.start) {
                out.push(Doc::HardLine);
            }
            out.push(build(self, element));
            self.last_end = self.last_end.max(span.end);
            if let Some(comment) = self.trailing(span.end) {
                out.push(comment);
            }
        }
        out
    }
}
