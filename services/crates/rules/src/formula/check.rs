//! The checker: static types, diagnostics and the analysis the Studio shows.
//!
//! [`check`] parses a formula and checks it against what the schema and the template know:
//! every name must be a field, one of the template's inputs or a `let` of its own; every call a
//! known function with the right arguments; every operator applied to values of the kind it takes;
//! a field that can be empty guarded; the `let` lines in order and used; and the whole formula
//! coming to an amount (or, for a yes/no rule, to true/false or a number). Each finding carries the
//! span it is about and, where one exists, the one-click fix that resolves it.
//!
//! The browser and the server read the same answer: a formula is accepted iff it has no
//! error-severity diagnostic ([`Analysis::accepted`]).
//!
//! Until the schema has loaded (`schema: None`) no name can be called unknown, so the findings
//! that depend on the record's fields are left out and the analysis says it is `pending`; the
//! functions are compiled into the catalog, so everything about calls is still checked.

mod checker;
mod cursor;
mod knowledge;
mod suggest;
mod types;

pub use cursor::Identifier;
pub use cursor::IdentifierKind;
pub use cursor::IdentifierToken;
pub use cursor::Signature;
pub use cursor::identifier_at;
pub use cursor::identifier_tokens;
pub use cursor::signature_at;
pub use knowledge::RateTable;
pub use knowledge::Schema;
pub use knowledge::SchemaInput;
pub use knowledge::SchemaVariable;
pub use suggest::infer_input_type;
pub use suggest::nearest_known;
pub use types::Type;

use crate::formula::syntax::Program;
use crate::formula::syntax::StatementInfo;
use crate::message::Message;
use crate::span::Span;

/// What a formula must come to.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum Expect {
    /// An amount: a number.
    #[default]
    Amount,
    /// A yes/no rule: true/false, or a number (non-zero is yes).
    Predicate,
}

/// How much a diagnostic matters. Only errors stop a formula from saving.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Severity {
    /// The formula cannot run as written.
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

/// What a diagnostic is about: a stable code the browser and the server can switch on.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum Code {
    /// The text does not follow the grammar, or a pointer is used where it means nothing.
    Syntax,
    /// The text is longer, deeper or larger than a formula may be.
    Limit,
    /// The formula is empty.
    Empty,
    /// A name that is no field, input or `let`.
    UnknownVariable,
    /// A field the record or a list's items do not have.
    UnknownField,
    /// A function or method that does not exist.
    UnknownFunction,
    /// A function formulas must not call.
    DisabledFunction,
    /// A call with too few or too many arguments.
    Arity,
    /// A value of the wrong kind.
    Type,
    /// A field that can be empty, read without a guard.
    Nullable,
    /// A `let` nothing reads.
    LetUnused,
    /// A `let` whose name is already bound.
    LetDuplicate,
    /// A `let` read before the line that defines it.
    LetBeforeDefinition,
    /// A `let` named after a field, an input or a function.
    LetShadows,
    /// A formula that ends on a `let`.
    LetLast,
    /// A formula that comes to the wrong kind of value.
    ResultType,
    /// A value line that is worked out and thrown away.
    Discarded,
    /// A division by a literal zero.
    DivideByZero,
    /// A rate table that does not exist, is not named in quotes, or has the wrong shape.
    RateTable,
}

impl Code {
    /// The code's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Syntax => "syntax",
            Self::Limit => "limit",
            Self::Empty => "empty",
            Self::UnknownVariable => "unknown-variable",
            Self::UnknownField => "unknown-field",
            Self::UnknownFunction => "unknown-function",
            Self::DisabledFunction => "disabled-function",
            Self::Arity => "arity",
            Self::Type => "type",
            Self::Nullable => "nullable",
            Self::LetUnused => "let-unused",
            Self::LetDuplicate => "let-duplicate",
            Self::LetBeforeDefinition => "let-before-definition",
            Self::LetShadows => "let-shadows",
            Self::LetLast => "let-last",
            Self::ResultType => "result-type",
            Self::Discarded => "discarded",
            Self::DivideByZero => "divide-by-zero",
            Self::RateTable => "rate-table",
        }
    }

    /// Whether the finding needs the record's fields, so it is left out until they have loaded.
    #[must_use]
    pub const fn needs_schema(self) -> bool {
        matches!(self, Self::UnknownVariable | Self::UnknownField)
    }
}

/// The type an input declared from a fix starts as.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum InputType {
    /// A number.
    Number,
    /// Text.
    String,
    /// A flag.
    Boolean,
}

impl InputType {
    /// The type's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Number => "Number",
            Self::String => "String",
            Self::Boolean => "Boolean",
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

/// A change the editor can make to resolve a finding, or an input to declare.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Fix {
    /// Replace text.
    Replace {
        /// What the fix does, for its button.
        label: Message,
        /// The replacements, which do not overlap.
        changes: Vec<Change>,
    },
    /// Declare the name as one of the template's inputs.
    Declare {
        /// What the fix does, for its button.
        label: Message,
        /// The input's name.
        name: String,
        /// The type it starts as.
        input_type: InputType,
    },
}

/// One finding.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Diagnostic {
    /// What it is about.
    pub span: Span,
    /// How much it matters.
    pub severity: Severity,
    /// What it is about, as a stable code.
    pub code: Code,
    /// What is wrong.
    pub message: Message,
    /// The name the finding is about, when it is about one.
    pub name: Option<String>,
    /// The fixes that resolve it.
    pub fixes: Vec<Fix>,
}

/// A top-level `let`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Binding {
    /// The bound name.
    pub name: String,
    /// Where the name is written.
    pub name_span: Span,
    /// From `let` to the end of the value.
    pub span: Span,
    /// The value's type.
    pub ty: Type,
    /// How many times the formula reads it.
    pub reads: usize,
}

/// A rate table the formula reads, by a string literal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TableRef {
    /// The table's name.
    pub name: String,
    /// The function that reads it.
    pub function: &'static str,
    /// Where the name's literal is.
    pub span: Span,
}

/// Everything the checker found.
#[derive(Debug, Clone, PartialEq)]
pub struct Analysis {
    /// The parsed formula.
    pub program: Program,
    /// The formula holds nothing but spaces and comments.
    pub empty: bool,
    /// The schema has not loaded, so names were not checked.
    pub pending: bool,
    /// The findings, sorted by where they start.
    pub diagnostics: Vec<Diagnostic>,
    /// The type of the formula's last value.
    pub result_type: Type,
    /// Every field and input the formula reads, inputs first, each in the order it first
    /// appears.
    pub reads: Vec<String>,
    /// The record's fields it reads, by the path it reads them as.
    pub schema_reads: Vec<String>,
    /// The template's inputs it reads.
    pub input_reads: Vec<String>,
    /// The functions it calls, in the order it first calls them.
    pub function_calls: Vec<String>,
    /// Its top-level `let` lines, in order.
    pub bindings: Vec<Binding>,
    /// Its statements, as the rate trace shows them.
    pub statements: Vec<StatementInfo>,
    /// Every rate table it reads by name, in source order.
    pub lookup_tables: Vec<TableRef>,
}

impl Analysis {
    /// Whether the formula may be saved: it has no error-severity diagnostic.
    #[must_use]
    pub fn accepted(&self) -> bool {
        !self
            .diagnostics
            .iter()
            .any(|diagnostic| diagnostic.severity == Severity::Error)
    }
}

/// Checks a formula. `schema` is `None` while the schema is loading; `tables`, when given, are
/// the rate tables the formula may read.
#[must_use]
pub fn check(
    source: &str,
    expect: Expect,
    schema: Option<&Schema>,
    tables: Option<&[RateTable]>,
) -> Analysis {
    checker::run(source, expect, schema, tables)
}
