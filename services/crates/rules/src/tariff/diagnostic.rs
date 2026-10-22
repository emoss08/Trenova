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
