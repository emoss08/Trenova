//! The types the checker gives every Lading expression.

use std::fmt;

/// The dimension of a measure.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum Dim {
    /// `lb`, `kg`, `oz`, `t`, `ton`
    Weight,
    /// `in`, `ft`, `yd`, `mi`, `cm`, `m`, `km`
    Length,
    /// `ft3`, `m3`, `gal`, `l`
    Volume,
    /// `°F`, `°C`
    Temperature,
}

impl Dim {
    /// The type's name.
    #[must_use]
    pub fn name(self) -> &'static str {
        match self {
            Self::Weight => "Weight",
            Self::Length => "Length",
            Self::Volume => "Volume",
            Self::Temperature => "Temperature",
        }
    }
}

/// Where a segment or loop type points: a guide in the project's table and a node of its
/// structure.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub struct NodeRef {
    /// The guide's index in the checker's table.
    pub guide: usize,
    /// The node's index in that guide's compiled structure.
    pub node: usize,
}

/// A type.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum Type {
    /// Any text.
    Text,
    /// Text from the code list of an element (`S502`).
    Code(String),
    /// A whole number.
    Int,
    /// An exact decimal.
    Decimal,
    /// `true` or `false`.
    Bool,
    /// A calendar date.
    Date,
    /// A time of day.
    Time,
    /// An instant with its zone.
    DateTime,
    /// A time zone.
    Zone,
    /// A length of time.
    Duration,
    /// A percent.
    Percent,
    /// Money in a currency.
    Money,
    /// A measure.
    Measure(Dim),
    /// A list.
    List(Box<Type>),
    /// Text keys to values.
    Map(Box<Type>),
    /// A value or none.
    Optional(Box<Type>),
    /// A segment at a known place in a guide.
    Segment(NodeRef),
    /// A segment by ID, at no particular place (`Segment<G62>`).
    SegmentId(String),
    /// A loop instance at a known place in a guide.
    Loop(NodeRef),
    /// A loop by ID, at no particular place (`Loop<0310>`).
    LoopId(String),
    /// The top of a transaction set read with a guide.
    Transaction(usize),
    /// A record of the schema: a document or one of its records.
    Record(&'static str),
    /// An enum of the schema.
    Enum(&'static str),
    /// A reference to a Trenova record found by a lookup.
    Ref(&'static str),
    /// A record written in place: `{ name: value }`, a CSV row, a composite element.
    Struct(Vec<(String, Type)>),
    /// Any JSON value.
    Json,
    /// A function value: a lambda's parameter and result.
    Lambda(Box<Type>, Box<Type>),
    /// The type of `none`.
    None,
    /// Not known: an error was already reported.
    Any,
}

impl Type {
    /// `T?`, without doubling.
    #[must_use]
    pub fn optional(self) -> Self {
        match self {
            Self::Optional(_) | Self::Any | Self::None => self,
            other => Self::Optional(Box::new(other)),
        }
    }

    /// `[T]`
    #[must_use]
    pub fn list(self) -> Self {
        Self::List(Box::new(self))
    }

    /// The type inside an optional, or the type itself.
    #[must_use]
    pub fn unwrapped(&self) -> &Self {
        match self {
            Self::Optional(inner) => inner,
            other => other,
        }
    }

    /// Whether it may be none.
    #[must_use]
    pub fn is_optional(&self) -> bool {
        matches!(self, Self::Optional(_) | Self::None)
    }

    /// The item type of a list (through an optional).
    #[must_use]
    pub fn item(&self) -> Option<&Self> {
        match self.unwrapped() {
            Self::List(item) => Some(item),
            _ => None,
        }
    }

    /// Whether it is unknown.
    #[must_use]
    pub fn is_any(&self) -> bool {
        matches!(self, Self::Any)
    }

    /// Whether values of it are ordered.
    #[must_use]
    pub fn is_ordered(&self) -> bool {
        matches!(
            self.unwrapped(),
            Self::Int
                | Self::Decimal
                | Self::Date
                | Self::Time
                | Self::DateTime
                | Self::Duration
                | Self::Percent
                | Self::Money
                | Self::Measure(_)
                | Self::Text
                | Self::Code(_)
                | Self::Any
        )
    }

    /// Whether it is a number.
    #[must_use]
    pub fn is_number(&self) -> bool {
        matches!(self.unwrapped(), Self::Int | Self::Decimal | Self::Any)
    }

    /// Whether it is text, coded or not.
    #[must_use]
    pub fn is_text(&self) -> bool {
        matches!(self.unwrapped(), Self::Text | Self::Code(_) | Self::Any)
    }

    /// Whether a value of `self` can go where a `target` is expected.
    #[must_use]
    pub fn fits(&self, target: &Self) -> bool {
        if self == target {
            return true;
        }
        match (self, target) {
            (Self::Optional(inner), Self::Optional(target_inner)) => inner.fits(target_inner),
            (_, Self::Optional(target_inner)) if !matches!(self, Self::None | Self::Any) => {
                self.fits(target_inner)
            }
            (Self::Any, _)
            | (_, Self::Any)
            | (Self::None, Self::Optional(_))
            | (Self::Code(_), Self::Text | Self::Code(_))
            | (Self::Text, Self::Code(_))
            | (Self::Int, Self::Decimal)
            | (Self::Segment(_), Self::SegmentId(_))
            | (Self::Loop(_), Self::LoopId(_)) => true,
            (Self::List(item), Self::List(target_item)) => item.fits(target_item),
            (Self::Map(value), Self::Map(target_value)) => value.fits(target_value),
            (Self::Struct(fields), Self::Struct(target_fields)) => {
                target_fields.iter().all(|(name, ty)| {
                    fields
                        .iter()
                        .find(|(field, _)| field == name)
                        .map_or(ty.is_optional(), |(_, field_ty)| field_ty.fits(ty))
                })
            }
            _ => false,
        }
    }

    /// The common type of two branches (`a ?? b`, `if`, `match`), when they have one.
    #[must_use]
    pub fn join(&self, other: &Self) -> Option<Self> {
        if self.fits(other) {
            return Some(other.clone());
        }
        if other.fits(self) {
            return Some(self.clone());
        }
        match (self, other) {
            (Self::None, other) | (other, Self::None) => Some(other.clone().optional()),
            (Self::Optional(inner), other) | (other, Self::Optional(inner)) => {
                inner.join(other).map(Self::optional)
            }
            _ => None,
        }
    }
}

impl fmt::Display for Type {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Text => f.write_str("Text"),
            Self::Code(element) => write!(f, "Code<{element}>"),
            Self::Int => f.write_str("Int"),
            Self::Decimal => f.write_str("Decimal"),
            Self::Bool => f.write_str("Bool"),
            Self::Date => f.write_str("Date"),
            Self::Time => f.write_str("Time"),
            Self::DateTime => f.write_str("DateTime"),
            Self::Zone => f.write_str("Zone"),
            Self::Duration => f.write_str("Duration"),
            Self::Percent => f.write_str("Percent"),
            Self::Money => f.write_str("Money"),
            Self::Measure(dim) => f.write_str(dim.name()),
            Self::List(item) => write!(f, "[{item}]"),
            Self::Map(value) => write!(f, "[Text: {value}]"),
            Self::Optional(inner) => write!(f, "{inner}?"),
            Self::Segment(_) => f.write_str("Segment"),
            Self::SegmentId(id) => write!(f, "Segment<{id}>"),
            Self::Loop(_) => f.write_str("Loop"),
            Self::LoopId(id) => write!(f, "Loop<{id}>"),
            Self::Transaction(_) => f.write_str("Transaction"),
            Self::Record(name) | Self::Enum(name) | Self::Ref(name) => f.write_str(name),
            Self::Struct(fields) => {
                f.write_str("{ ")?;
                for (index, (name, ty)) in fields.iter().enumerate() {
                    if index > 0 {
                        f.write_str(", ")?;
                    }
                    write!(f, "{name}: {ty}")?;
                }
                f.write_str(" }")
            }
            Self::Json => f.write_str("Json"),
            Self::Lambda(param, result) => write!(f, "({param}) => {result}"),
            Self::None => f.write_str("none"),
            Self::Any => f.write_str("?"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::{Dim, Type};

    #[test]
    fn optionals_accept_their_inner_type_and_none() {
        assert!(Type::Text.fits(&Type::Text.optional()));
        assert!(Type::None.fits(&Type::Text.optional()));
        assert!(!Type::Text.optional().fits(&Type::Text));
        assert!(Type::Code("S502".into()).fits(&Type::Text));
        assert!(Type::Int.fits(&Type::Decimal));
        assert!(!Type::Decimal.fits(&Type::Int));
        assert!(!Type::Decimal.fits(&Type::Measure(Dim::Weight)));
    }

    #[test]
    fn joins_make_optionals() {
        assert_eq!(Type::Text.join(&Type::None), Some(Type::Text.optional()));
        assert_eq!(
            Type::Int.join(&Type::Decimal.optional()),
            Some(Type::Decimal.optional())
        );
        assert_eq!(Type::Text.join(&Type::Bool), None);
    }

    #[test]
    fn display_reads_like_the_language() {
        assert_eq!(
            Type::Code("S502".into()).list().optional().to_string(),
            "[Code<S502>]?"
        );
        assert_eq!(Type::Map(Box::new(Type::Text)).to_string(), "[Text: Text]");
    }
}
