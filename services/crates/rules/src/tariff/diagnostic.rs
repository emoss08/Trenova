//! Findings about a Tariff file, shared by the parser and the checker.
//!
//! Every finding has a stable kebab-case code the hosts can switch on (`syntax`, `unit-mismatch`,
//! `optional-unhandled`, …), an English [`Message`] each host translates, a byte span and the
//! fixes the editor offers. Spans stay in bytes inside the crate and are converted to UTF-16 at
//! the operation boundary.

use crate::message::Message;
use crate::span::Span;

/// How much a finding matters. Only errors stop a file from saving, previewing and rating.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum Severity {
    /// The file cannot run as written.
    Error,
    /// It runs, but probably not as meant.
    Warning,
    /// Worth knowing.
    Info,
}

impl Severity {
    /// The severity's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Error => "error",
            Self::Warning => "warning",
            Self::Info => "info",
        }
    }
}

/// One replacement a fix makes.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Change {
    /// What it replaces; an empty span inserts.
    pub span: Span,
    /// The text that goes there.
    pub insert: String,
}

/// A change the editor can make to resolve a finding.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Fix {
    /// What the fix does, for its button.
    pub label: Message,
    /// The replacements, which do not overlap.
    pub changes: Vec<Change>,
}

/// One finding.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Diagnostic {
    /// What it is about.
    pub span: Span,
    /// How much it matters.
    pub severity: Severity,
    /// A stable kebab-case code.
    pub code: &'static str,
    /// What is wrong.
    pub message: Message,
    /// The fixes that resolve it.
    pub fixes: Vec<Fix>,
    /// Other places the finding refers to (the arm a range overlaps, the first declaration of a
    /// duplicate), each with what it is.
    pub related: Vec<(Span, Message)>,
}

impl Diagnostic {
    /// An error with no fixes.
    #[must_use]
    pub fn error(code: &'static str, span: Span, message: Message) -> Self {
        Self {
            span,
            severity: Severity::Error,
            code,
            message,
            fixes: Vec::new(),
            related: Vec::new(),
        }
    }

    /// A warning with no fixes.
    #[must_use]
    pub fn warning(code: &'static str, span: Span, message: Message) -> Self {
        Self {
            severity: Severity::Warning,
            ..Self::error(code, span, message)
        }
    }

    /// The same finding with a fix added.
    #[must_use]
    pub fn with_fix(mut self, fix: Fix) -> Self {
        self.fixes.push(fix);
        self
    }

    /// The same finding with a related place added.
    #[must_use]
    pub fn with_related(mut self, span: Span, message: Message) -> Self {
        self.related.push((span, message));
        self
    }
}

/// Names longer than this get no suggestion, so a suggestion never costs more than a bounded
/// comparison per candidate, however long the text.
const LONGEST_SUGGESTED: usize = 64;

/// The edit distance between two names, or `usize::MAX` when their lengths differ by more than
/// three (no suggestion would be close enough).
fn edit_distance(a: &[char], b: &[char]) -> usize {
    if a.len().abs_diff(b.len()) > 3 {
        return usize::MAX;
    }
    let mut previous: Vec<usize> = (0..=b.len()).collect();
    let mut current = vec![0; b.len() + 1];
    for (i, left) in a.iter().enumerate() {
        if let Some(first) = current.first_mut() {
            *first = i + 1;
        }
        for (j, right) in b.iter().enumerate() {
            let cost = usize::from(left != right);
            let above = previous.get(j + 1).copied().unwrap_or(usize::MAX - 1) + 1;
            let before = current.get(j).copied().unwrap_or(usize::MAX - 1) + 1;
            let diagonal = previous.get(j).copied().unwrap_or(usize::MAX - 1) + cost;
            if let Some(cell) = current.get_mut(j + 1) {
                *cell = above.min(before).min(diagonal);
            }
        }
        std::mem::swap(&mut previous, &mut current);
    }
    previous.get(b.len()).copied().unwrap_or(usize::MAX)
}

/// The closest of `candidates` to `name`, for a "did you mean" suggestion: within two edits, or
/// the same but for case. Ties go to the first candidate; `name` itself is never suggested.
#[must_use]
pub fn nearest<'a>(name: &str, candidates: impl IntoIterator<Item = &'a str>) -> Option<&'a str> {
    let lower: Vec<char> = name.to_lowercase().chars().collect();
    if lower.is_empty() || lower.len() > LONGEST_SUGGESTED {
        return None;
    }
    let mut best = None;
    let mut best_distance = 3;
    for candidate in candidates {
        if candidate == name || candidate.chars().count() > LONGEST_SUGGESTED + 3 {
            continue;
        }
        let other: Vec<char> = candidate.to_lowercase().chars().collect();
        let distance = if other == lower {
            0
        } else {
            edit_distance(&lower, &other)
        };
        if distance < best_distance {
            best = Some(candidate);
            best_distance = distance;
        }
    }
    best
}
