//! Why a program stopped: a refusal (the author's words) or an evaluation error (the crate's).
//!
//! A run-time problem never panics and never degrades to a number. It stops the program that hit
//! it and comes back as an [`EvalError`]: a [`Code`] the host may branch on, a [`Message`] the
//! person reads and the byte span of the node that failed. A refusal (`fail`, `require … else`,
//! `?? fail`) is not an error: it is the tariff declining to rate, with the author's
//! interpolated text and where it is written.
//!
//! Spans are always in the program's root file: when an error or refusal leaves a call into
//! another module, the interpreter moves it onto the call, so the host can underline it in the
//! source the author is looking at.

use trenova_module_kit::span::Span;
use trenova_module_kit::message::{Arg, Message};

/// What went wrong, as a stable name the host may branch on.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Code {
    /// The file does not check: a syntax error, an unknown import, the wrong kind of file, a
    /// type error. The error is the first finding; `tariff.check` reports them all.
    Check,
    /// The program did something its static types rule out: an operation on values of the
    /// wrong kind, a name nothing declares, a `match` no arm covers. The checker in front of
    /// the evaluator makes this unreachable for a file that checked clean.
    Internal,
    /// The program used up a budget: steps, text, list items or call depth.
    Budget,
    /// An input value the host sent does not fit the input's declared type, or names no input.
    Input,
    /// The record the host sent does not fit the schema.
    Record,
    /// Division or remainder by zero.
    DivideByZero,
    /// A result past 28 significant digits, or a date past year 9999.
    Overflow,
    /// A unit conversion between dimensions, or one that does not fit.
    Conversion,
    /// An argument of the right kind with an unusable value (a rounding increment of zero, a
    /// negative square root).
    Argument,
    /// A regular expression that does not compile or is too large.
    Regex,
    /// A rate table the request did not carry, or one whose cells do not fit its shape.
    Table,
    /// A coded charge with a negative amount.
    NegativeCharge,
    /// A bill whose freight lines or total come out below zero.
    NegativeTotal,
}

impl Code {
    /// The code as it appears on the wire.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Check => "check",
            Self::Internal => "internal",
            Self::Budget => "budget",
            Self::Input => "input",
            Self::Record => "record",
            Self::DivideByZero => "divideByZero",
            Self::Overflow => "overflow",
            Self::Conversion => "conversion",
            Self::Argument => "argument",
            Self::Regex => "regex",
            Self::Table => "table",
            Self::NegativeCharge => "negativeCharge",
            Self::NegativeTotal => "negativeTotal",
        }
    }
}

/// One run-time failure.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EvalError {
    /// The stable code.
    pub code: Code,
    /// What the person reads.
    pub message: Message,
    /// The byte span in the root file of the node that failed, once it is known.
    pub span: Option<Span>,
}

impl EvalError {
    /// An error not yet placed in the source.
    #[must_use]
    pub const fn new(code: Code, message: Message) -> Self {
        Self {
            code,
            message,
            span: None,
        }
    }

    /// An error whose message has positional arguments.
    #[must_use]
    pub fn with(code: Code, template: &'static str, args: impl IntoIterator<Item = Arg>) -> Self {
        Self::new(code, Message::with(template, args))
    }

    /// An internal error: the program did something its types rule out.
    #[must_use]
    pub fn internal(message: Message) -> Self {
        Self::new(Code::Internal, message)
    }

    /// The same error placed at `span`, unless it already has a place.
    #[must_use]
    pub fn at(mut self, span: Span) -> Self {
        if self.span.is_none() {
            self.span = Some(span);
        }
        self
    }

    /// A result that does not fit in 28 significant digits.
    #[must_use]
    pub fn overflow() -> Self {
        Self::new(
            Code::Overflow,
            Message::new(trenova_module_kit::message!(
                "The result is too large: it does not fit in 28 significant digits"
            )),
        )
    }

    /// Division or remainder by zero.
    #[must_use]
    pub fn divide_by_zero() -> Self {
        Self::new(
            Code::DivideByZero,
            Message::new(trenova_module_kit::message!("Division by zero")),
        )
    }
}

/// The program refused to rate: the author wrote `fail "…"`, `require … else "…"` or
/// `?? fail "…"`, or an input nobody set has no default.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Refusal {
    /// Why.
    pub message: RefusalText,
    /// Where the refusal is written (or the input declared), in the root file.
    pub span: Span,
}

/// A refusal's text.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum RefusalText {
    /// The author's words, interpolated: shown as written, never translated by the crate.
    Author(String),
    /// A refusal the crate words: an input with no value.
    Crate(Message),
}

impl RefusalText {
    /// The text a test's `expect fails "…"` compares with.
    #[must_use]
    pub fn text(&self) -> String {
        match self {
            Self::Author(text) => text.clone(),
            Self::Crate(message) => message.english(),
        }
    }
}

/// Why a program stopped before its end.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Stop {
    /// The program refused to rate.
    Refused(Refusal),
    /// The program could not be evaluated.
    Failed(EvalError),
}

impl From<EvalError> for Stop {
    fn from(error: EvalError) -> Self {
        Self::Failed(error)
    }
}

impl Stop {
    /// The same stop placed at `span` when it has no place yet (errors only; a refusal always
    /// has its own).
    #[must_use]
    pub fn at(self, span: Span) -> Self {
        match self {
            Self::Failed(error) => Self::Failed(error.at(span)),
            refused @ Self::Refused(_) => refused,
        }
    }

    /// The same stop moved onto `span`: what an error or refusal raised inside another module
    /// becomes at the call that reached it.
    #[must_use]
    pub fn moved_to(self, span: Span) -> Self {
        match self {
            Self::Failed(mut error) => {
                error.span = Some(span);
                Self::Failed(error)
            }
            Self::Refused(mut refusal) => {
                refusal.span = span;
                Self::Refused(refusal)
            }
        }
    }
}
