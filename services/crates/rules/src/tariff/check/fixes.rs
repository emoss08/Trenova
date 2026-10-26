//! Building the fixes the checker offers: replacements and insertions on the module's source.

use trenova_module_kit::diagnostic::{Change, Fix};
use trenova_module_kit::message::Message;
use trenova_module_kit::span::Span;

/// A fix that replaces one span with text.
#[must_use]
pub fn replace(label: Message, span: Span, insert: impl Into<String>) -> Fix {
    Fix {
        label,
        changes: vec![Change {
            span,
            insert: insert.into(),
        }],
    }
}

/// A fix made of several changes, sorted by where they start so they never overlap out of
/// order.
#[must_use]
pub fn changes(label: Message, mut list: Vec<Change>) -> Fix {
    list.sort_by_key(|change| (change.span.start, change.span.end));
    Fix {
        label,
        changes: list,
    }
}

/// The whitespace a line starts with, up to the byte offset `at`.
#[must_use]
pub fn indentation(source: &str, at: u32) -> String {
    let at = usize::try_from(at).unwrap_or(usize::MAX).min(source.len());
    let start = source
        .get(..at)
        .and_then(|head| head.rfind('\n').map(|newline| newline + 1))
        .unwrap_or(0);
    source
        .get(start..at)
        .unwrap_or_default()
        .chars()
        .take_while(|character| *character == ' ' || *character == '\t')
        .collect()
}

/// Applies a fix's changes to a source text, for the corpus tests.
#[cfg(test)]
#[must_use]
pub fn apply(source: &str, fix: &Fix) -> String {
    let mut out = String::with_capacity(source.len());
    let mut cursor = 0usize;
    for change in &fix.changes {
        let start = usize::try_from(change.span.start)
            .unwrap_or(usize::MAX)
            .min(source.len());
        let end = usize::try_from(change.span.end)
            .unwrap_or(usize::MAX)
            .min(source.len());
        if start < cursor {
            continue;
        }
        out.push_str(source.get(cursor..start).unwrap_or_default());
        out.push_str(&change.insert);
        cursor = end.max(start);
    }
    out.push_str(source.get(cursor..).unwrap_or_default());
    out
}
