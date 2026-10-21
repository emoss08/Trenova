//! Evaluation errors: a stable code, a translatable message and where in the formula it happened.
//!
//! A run-time problem never panics and never degrades to a number. It stops the program that hit
//! it and comes back as an [`EvalError`]: a [`Code`] the caller may branch on, a [`Message`] the
//! person reads (the English text of the Go error it replaces, so the translations carry over),
//! the byte span of the node that failed, and for a missing value the fields the formula read and
//! the guards that would keep it pricing.
//!
//! A message that names the kind of a value (`expected a number, got text`) carries the kind as
//! an ICU `select` argument (`nil`, `boolean`, `text`, `number`, `date`, `duration`, `list`,
//! `map`), so the phrase is translated with the rest of the sentence.

use crate::message::Arg;
use crate::message::Message;
use crate::span::Span;

/// What went wrong, as a stable name the caller may branch on.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Code {
    /// A field the formula reads has no value on this record.
    MissingField,
    /// A name that is neither a field, an input, a `let` nor a function.
    UnknownName,
    /// The rate table exists but has no entry, band or cell for the key.
    LookupMiss,
    /// No rate table has the name.
    TableNotFound,
    /// The request carried no rate tables at all.
    LookupUnavailable,
    /// The table exists but has the wrong shape for the call (one axis, two axes, keys or bands).
    TableShape,
    /// A lookup key that the axis cannot read (not numeric, or of an unsupported kind).
    LookupKey,
    /// Division or remainder by zero.
    DivideByZero,
    /// A result that does not fit: a number past 28 significant digits, a date past year 9999,
    /// a duration past 292 years.
    Overflow,
    /// A result that is not a real number (a negative number to a fractional power, zero to a
    /// negative power).
    NonFinite,
    /// An operand or argument of the wrong kind.
    Type,
    /// An argument of the right kind with an unusable value.
    Argument,
    /// An index outside the list or text.
    Index,
    /// A regular expression that does not compile.
    Regex,
    /// The formula used up its step, text or item budget.
    Budget,
    /// An amount that came out as something other than a number.
    NonNumericResult,
    /// An amount that came out as true or false.
    BooleanAmount,
    /// An amount that came out empty.
    NullResult,
}

impl Code {
    /// The code as it appears on the wire.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::MissingField => "missingField",
            Self::UnknownName => "unknownName",
            Self::LookupMiss => "lookupMiss",
            Self::TableNotFound => "tableNotFound",
            Self::LookupUnavailable => "lookupUnavailable",
            Self::TableShape => "tableShape",
            Self::LookupKey => "lookupKey",
            Self::DivideByZero => "divideByZero",
            Self::Overflow => "overflow",
            Self::NonFinite => "nonFinite",
            Self::Type => "type",
            Self::Argument => "argument",
            Self::Index => "index",
            Self::Regex => "regex",
            Self::Budget => "budget",
            Self::NonNumericResult => "nonNumericResult",
            Self::BooleanAmount => "booleanAmount",
            Self::NullResult => "nullResult",
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
    /// The byte span of the node that failed, when one did.
    pub span: Option<Span>,
    /// For a missing value: the field paths the formula read without a value.
    pub fields: Vec<String>,
    /// For a missing value: the guard that keeps the formula pricing, one per field.
    pub guards: Vec<String>,
    /// Which argument of a call was empty, so the interpreter can name the field it came from.
    pub(crate) empty_argument: Option<usize>,
}

impl EvalError {
    /// An error with a code and a message, not yet placed in the source.
    #[must_use]
    pub fn new(code: Code, message: Message) -> Self {
        Self {
            code,
            message,
            span: None,
            fields: Vec::new(),
            guards: Vec::new(),
            empty_argument: None,
        }
    }

    /// An error whose message has positional arguments.
    #[must_use]
    pub fn with(code: Code, template: &'static str, args: impl IntoIterator<Item = Arg>) -> Self {
        Self::new(code, Message::with(template, args))
    }

    /// Places the error at `span` unless it already has a place.
    #[must_use]
    pub fn at(mut self, span: Span) -> Self {
        if self.span.is_none() {
            self.span = Some(span);
        }
        self
    }

    /// Marks the error as caused by the empty argument at `index`.
    #[must_use]
    pub(crate) fn empty_argument(mut self, index: usize) -> Self {
        self.empty_argument = Some(index);
        self
    }

    /// A missing value: the formula read `fields` and they were empty.
    #[must_use]
    pub fn missing(fields: Vec<String>, guards: Vec<String>) -> Self {
        let message = Message::with(
            crate::message!(
                "no value for {0} on this record; write {1} so the formula still prices when it is missing"
            ),
            [Arg::from(fields.join(", ")), Arg::from(guards.join(", "))],
        );
        Self {
            code: Code::MissingField,
            message,
            span: None,
            fields,
            guards,
            empty_argument: None,
        }
    }

    /// Whether the error must stop the whole program even inside a `let` whose value is never
    /// used: a spent budget is never deferred.
    #[must_use]
    pub fn is_fatal(&self) -> bool {
        self.code == Code::Budget
    }
}

/// An operand or argument of the wrong kind: `expected a number, got text`.
#[must_use]
pub fn expected_number(kind: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!(
            "expected a number, got {0, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}}"
        ),
        [Arg::from(kind)],
    )
}

/// An argument that must be a whole number and is of another kind.
#[must_use]
pub fn expected_whole_kind(kind: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!(
            "expected a whole number, got {0, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}}"
        ),
        [Arg::from(kind)],
    )
}

/// A number that must be whole and has a fraction, or does not fit a 64-bit integer.
#[must_use]
pub fn expected_whole_value(shown: String) -> EvalError {
    EvalError::with(
        Code::Argument,
        crate::message!("expected a whole number, got {0}"),
        [Arg::from(shown)],
    )
}

/// An argument that must be text.
#[must_use]
pub fn expected_text(kind: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!(
            "expected text, got {0, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}}"
        ),
        [Arg::from(kind)],
    )
}

/// A condition or predicate that must be true or false.
#[must_use]
pub fn expected_boolean(kind: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!(
            "expected true/false, got {0, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}}"
        ),
        [Arg::from(kind)],
    )
}

/// An argument that must be a list.
#[must_use]
pub fn expected_list(kind: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!(
            "expected a list, got {0, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}}"
        ),
        [Arg::from(kind)],
    )
}

/// An operator applied to operands it is not defined on: `"a" * 2`.
#[must_use]
pub fn invalid_operation(operator: &str, left: &'static str, right: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!(
            "invalid operation: {0, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}} {1} {2, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}}"
        ),
        [Arg::from(left), Arg::from(operator), Arg::from(right)],
    )
}

/// A unary operator applied to an operand it is not defined on: `-"a"`.
#[must_use]
pub fn invalid_unary(operator: &str, operand: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!(
            "invalid operation: {0} {1, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}}"
        ),
        [Arg::from(operator), Arg::from(operand)],
    )
}

/// A comparison between values that have no order, or between different kinds.
#[must_use]
pub fn cannot_compare(left: &'static str, right: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!(
            "cannot compare {0, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}} with {1, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}}"
        ),
        [Arg::from(left), Arg::from(right)],
    )
}

/// A function given an argument of a kind it does not take: `len(5)`.
#[must_use]
pub fn invalid_argument(function: &str, kind: &'static str) -> EvalError {
    EvalError::with(
        Code::Type,
        crate::message!(
            "invalid argument for {0} (type {1, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}})"
        ),
        [Arg::from(function), Arg::from(kind)],
    )
}

/// A number past what 28 significant digits hold.
#[must_use]
pub fn number_overflow() -> EvalError {
    EvalError::new(
        Code::Overflow,
        Message::new(crate::message!(
            "the result is too large for a number (28 significant digits)"
        )),
    )
}

/// Division or remainder by zero.
#[must_use]
pub fn divide_by_zero() -> EvalError {
    EvalError::new(
        Code::DivideByZero,
        Message::new(crate::message!("division by zero")),
    )
}

/// A result that is not a real number.
#[must_use]
pub fn non_finite() -> EvalError {
    EvalError::new(
        Code::NonFinite,
        Message::new(crate::message!("expression produced a non-finite number")),
    )
}

#[cfg(test)]
mod tests {
    use super::{Code, EvalError, expected_number};

    #[test]
    fn a_missing_value_names_its_fields_and_guards() {
        let error = EvalError::missing(
            vec![String::from("weight")],
            vec![String::from("coalesce(weight, 0)")],
        );
        assert_eq!(error.code, Code::MissingField);
        assert_eq!(
            error.message.english(),
            "no value for weight on this record; write coalesce(weight, 0) so the formula still prices when it is missing"
        );
    }

    #[test]
    fn kinds_travel_as_select_arguments() {
        let error = expected_number("text");
        assert!(error.message.message.contains("select"));
        assert_eq!(error.message.args.len(), 1);
        assert_eq!(error.code.as_str(), "type");
    }
}
