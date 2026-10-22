//! The static types the checker works out for every node.

use crate::formula::catalog::Returns;
use crate::message;
use crate::message::Message;

/// A static type.
///
/// `Integer` is a static fact about a number, not a second kind of value: it is a `Number` that
/// is known to be whole, which is what `%`, the bit functions, `..` and `take` need.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Type {
    /// An exact decimal.
    Number,
    /// A number known to be whole.
    Integer,
    /// Text.
    String,
    /// True or false.
    Boolean,
    /// An instant with a UTC offset.
    Date,
    /// A length of time.
    Duration,
    /// A list.
    Array,
    /// A record.
    Object,
    /// Nothing.
    Nil,
    /// Not known statically.
    Any,
}

impl Type {
    /// The type's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Number => "Number",
            Self::Integer => "Integer",
            Self::String => "String",
            Self::Boolean => "Boolean",
            Self::Date => "Date",
            Self::Duration => "Duration",
            Self::Array => "Array",
            Self::Object => "Object",
            Self::Nil => "Nil",
            Self::Any => "Any",
        }
    }

    /// A number, whole or not.
    #[must_use]
    pub const fn is_number(self) -> bool {
        matches!(self, Self::Number | Self::Integer)
    }

    /// Not known, or nothing: compatible with anything.
    #[must_use]
    pub const fn is_open(self) -> bool {
        matches!(self, Self::Any | Self::Nil)
    }

    /// A number or not known statically.
    #[must_use]
    pub const fn is_numeric(self) -> bool {
        matches!(self, Self::Number | Self::Integer | Self::Any)
    }

    /// The kind values of this type compare as: every number is one kind.
    #[must_use]
    pub const fn kind(self) -> Self {
        match self {
            Self::Integer => Self::Number,
            other => other,
        }
    }

    /// Whether two values of these types can be compared for equality.
    #[must_use]
    pub fn compatible(self, other: Self) -> bool {
        self.is_open() || other.is_open() || self.kind() == other.kind()
    }

    /// The type of a value that is one or the other.
    #[must_use]
    pub fn join(self, other: Self) -> Self {
        if self == other {
            return self;
        }
        if self.is_open() {
            return other;
        }
        if other.is_open() {
            return self;
        }
        if self.is_number() && other.is_number() {
            return Self::Number;
        }
        Self::Any
    }

    /// What a value of the type is called in a sentence.
    #[must_use]
    pub fn words(self) -> Message {
        Message::new(match self {
            Self::Number | Self::Integer => message!("a number"),
            Self::String => message!("text"),
            Self::Boolean => message!("true/false"),
            Self::Date => message!("a date"),
            Self::Duration => message!("a duration"),
            Self::Array => message!("a list"),
            Self::Object => message!("a record"),
            Self::Nil => message!("empty"),
            Self::Any => message!("a value"),
        })
    }

    /// The type a schema publishes a field as, by its JSON schema type name (any case), or a
    /// template input's declared type.
    #[must_use]
    pub fn from_schema(raw: &str) -> Self {
        match raw.to_ascii_lowercase().as_str() {
            "number" | "decimal" | "float" | "float64" => Self::Number,
            "integer" | "int" => Self::Integer,
            "string" | "text" => Self::String,
            "boolean" | "bool" => Self::Boolean,
            "date" | "datetime" | "time" | "timestamp" => Self::Date,
            "duration" => Self::Duration,
            "array" | "list" => Self::Array,
            "object" | "map" => Self::Object,
            _ if raw.starts_with("[]") => Self::Array,
            _ => Self::Any,
        }
    }

    /// The type a function's result has.
    #[must_use]
    pub const fn from_returns(returns: Returns) -> Self {
        match returns {
            Returns::Number => Self::Number,
            Returns::Integer => Self::Integer,
            Returns::String => Self::String,
            Returns::Boolean => Self::Boolean,
            Returns::Date => Self::Date,
            Returns::Duration => Self::Duration,
            Returns::Array => Self::Array,
            Returns::Object => Self::Object,
            Returns::Any => Self::Any,
        }
    }

    /// A value that has no fields, no methods and no items.
    #[must_use]
    pub const fn is_scalar(self) -> bool {
        matches!(
            self,
            Self::Number
                | Self::Integer
                | Self::String
                | Self::Boolean
                | Self::Date
                | Self::Duration
        )
    }
}
