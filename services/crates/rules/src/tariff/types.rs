//! Tariff's types: what the checker infers, what the schema declares and what input values are
//! decoded against.
//!
//! A quantity's type is its [`Dim`]ension (`Weight`, `Money/Weight`); the unit it is shown in
//! (`lb`, `$/cwt`) rides along as a display hint and never takes part in equality, so a
//! `Money/cwt` input accepts `$0.185/lb`. Money is the quantity whose dimension is
//! [`Dim::money`]. Count labels fold into the value, so `Money/pallet` is `Money` shown per pallet.
//!
//! Records are nominal when declared (`type Break = { … }`: two declarations with the same
//! fields are different types) and structural when written inline in another type
//! (`temperature: { min: Temperature?, max: Temperature? }`). Enums are nominal.

use super::units::{Base, Dim, Unit};
use std::fmt;
use std::sync::Arc;

/// A type.
#[derive(Debug, Clone)]
pub enum Type {
    /// `true` or `false`.
    Bool,
    /// A whole number.
    Int,
    /// An exact decimal.
    Number,
    /// `18%`.
    Percent,
    /// Unicode text.
    Text,
    /// A calendar date.
    Date,
    /// An instant with its UTC offset.
    DateTime,
    /// Money, a rate or a quantity.
    Quantity(QuantityType),
    /// `List<T>`.
    List(Arc<Type>),
    /// `T?`.
    Optional(Arc<Type>),
    /// A record.
    Record(Arc<RecordType>),
    /// An enum.
    Enum(Arc<EnumType>),
    /// The type of `fail …` and of a `return`'s position: no value, so it fits anywhere.
    Never,
    /// A type the checker could not work out because of an error it already reported. It fits
    /// anywhere, so one mistake is reported once.
    Error,
}

/// A quantity's dimension and the unit its values are shown in.
#[derive(Debug, Clone)]
pub struct QuantityType {
    /// What it measures.
    pub dim: Dim,
    /// The unit an annotation named (`Money/cwt` shows per hundredweight), when one did.
    pub display: Option<Unit>,
}

/// A record type.
#[derive(Debug, Clone)]
pub struct RecordType {
    /// The declared name, qualified by its module (`std.ltl.Break`, `Stop`); `None` for a
    /// record type written inline.
    pub name: Option<String>,
    /// The fields, in declaration order.
    pub fields: Vec<FieldDef>,
}

/// One field of a record type.
#[derive(Debug, Clone)]
pub struct FieldDef {
    /// The field's name.
    pub name: String,
    /// Its type.
    pub ty: Type,
    /// Its doc comment.
    pub doc: Option<String>,
}

/// An enum type.
#[derive(Debug, Clone)]
pub struct EnumType {
    /// The declared name, qualified by its module for a library enum (`org.zones.Zone`); bare
    /// for a schema enum (`StopKind`).
    pub name: String,
    /// The variants, in declaration order, each with its doc comment.
    pub variants: Vec<(String, Option<String>)>,
    /// The enum's doc comment.
    pub doc: Option<String>,
}

impl EnumType {
    /// Whether the enum has a variant.
    #[must_use]
    pub fn has(&self, variant: &str) -> bool {
        self.variants.iter().any(|(name, _)| name == variant)
    }
}

impl RecordType {
    /// The field named `name`.
    #[must_use]
    pub fn field(&self, name: &str) -> Option<&FieldDef> {
        self.fields.iter().find(|field| field.name == name)
    }
}

/// A function's signature, for user functions, library functions and built-in methods.
#[derive(Debug, Clone)]
pub struct FnSig {
    /// The parameters, in order.
    pub params: Vec<ParamSig>,
    /// What it returns.
    pub ret: Type,
}

/// One parameter.
#[derive(Debug, Clone)]
pub struct ParamSig {
    /// Its name, also its name at a call site.
    pub name: String,
    /// Its type.
    pub ty: Type,
    /// Whether it has a default, so it may be left out.
    pub default: bool,
}

impl Type {
    /// Money.
    #[must_use]
    pub const fn money() -> Self {
        Self::Quantity(QuantityType {
            dim: Dim::money(),
            display: None,
        })
    }

    /// A quantity of one base dimension.
    #[must_use]
    pub const fn of(base: Base) -> Self {
        Self::Quantity(QuantityType {
            dim: Dim::of(base),
            display: None,
        })
    }

    /// A quantity of any dimension.
    #[must_use]
    pub const fn quantity(dim: Dim) -> Self {
        Self::Quantity(QuantityType { dim, display: None })
    }

    /// A quantity shown in a unit: `Money/cwt` is `quantity_in(&Unit::parse("$/cwt"))`.
    #[must_use]
    pub fn quantity_in(unit: &Unit) -> Self {
        Self::Quantity(QuantityType {
            dim: unit.dim(),
            display: Some(unit.clone()),
        })
    }

    /// `List<T>`.
    #[must_use]
    pub fn list(item: Self) -> Self {
        Self::List(Arc::new(item))
    }

    /// `T?`. An optional is never made optional twice.
    #[must_use]
    pub fn optional(inner: Self) -> Self {
        match inner {
            Self::Optional(_) | Self::Never | Self::Error => inner,
            other => Self::Optional(Arc::new(other)),
        }
    }

    /// The type without its `?`.
    #[must_use]
    pub fn unwrapped(&self) -> &Self {
        match self {
            Self::Optional(inner) => inner,
            other => other,
        }
    }

    /// Whether the type is `T?`.
    #[must_use]
    pub const fn is_optional(&self) -> bool {
        matches!(self, Self::Optional(_))
    }

    /// Whether the type is money (any display unit).
    #[must_use]
    pub fn is_money(&self) -> bool {
        matches!(self, Self::Quantity(quantity) if quantity.dim == Dim::money())
    }

    /// Whether the type is a number of either kind.
    #[must_use]
    pub const fn is_numeric(&self) -> bool {
        matches!(self, Self::Int | Self::Number)
    }

    /// Whether a value of type `actual` may be used where `self` is expected: the same type,
    /// `Int` where a `Number` is expected, `T` where a `T?` is, a list whose items may, and
    /// [`Type::Never`] and [`Type::Error`] anywhere.
    #[must_use]
    pub fn accepts(&self, actual: &Self) -> bool {
        match (self, actual) {
            (_, Self::Never | Self::Error) | (Self::Error, _) | (Self::Number, Self::Int) => true,
            (Self::Optional(expected), Self::Optional(actual))
            | (Self::List(expected), Self::List(actual)) => expected.accepts(actual),
            (Self::Optional(expected), actual) => expected.accepts(actual),
            (expected, actual) => expected == actual,
        }
    }

    /// The type a value of either type has when two branches meet (`if … else`, `match` arms,
    /// list items): the wider of the two, or `None` when neither accepts the other.
    #[must_use]
    pub fn join(&self, other: &Self) -> Option<Self> {
        if matches!(self, Self::Never) {
            return Some(other.clone());
        }
        if self.accepts(other) {
            return Some(self.clone());
        }
        if other.accepts(self) {
            return Some(other.clone());
        }
        match (self, other) {
            (Self::Optional(inner), plain) | (plain, Self::Optional(inner)) => {
                inner.join(plain).map(Self::optional)
            }
            _ => None,
        }
    }
}

impl PartialEq for Type {
    fn eq(&self, other: &Self) -> bool {
        match (self, other) {
            (Self::Bool, Self::Bool)
            | (Self::Int, Self::Int)
            | (Self::Number, Self::Number)
            | (Self::Percent, Self::Percent)
            | (Self::Text, Self::Text)
            | (Self::Date, Self::Date)
            | (Self::DateTime, Self::DateTime)
            | (Self::Never, Self::Never)
            | (Self::Error, Self::Error) => true,
            (Self::Quantity(left), Self::Quantity(right)) => left.dim == right.dim,
            (Self::List(left), Self::List(right))
            | (Self::Optional(left), Self::Optional(right)) => left == right,
            (Self::Record(left), Self::Record(right)) => match (&left.name, &right.name) {
                (Some(left), Some(right)) => left == right,
                (None, None) => {
                    left.fields.len() == right.fields.len()
                        && left
                            .fields
                            .iter()
                            .zip(&right.fields)
                            .all(|(left, right)| left.name == right.name && left.ty == right.ty)
                }
                _ => false,
            },
            (Self::Enum(left), Self::Enum(right)) => left.name == right.name,
            _ => false,
        }
    }
}

impl fmt::Display for Type {
    /// The type as an author writes it: `Money/cwt`, `Weight?`, `List<Stop>`, `{ min: Weight }`.
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Bool => formatter.write_str("Bool"),
            Self::Int => formatter.write_str("Int"),
            Self::Number => formatter.write_str("Number"),
            Self::Percent => formatter.write_str("Percent"),
            Self::Text => formatter.write_str("Text"),
            Self::Date => formatter.write_str("Date"),
            Self::DateTime => formatter.write_str("DateTime"),
            Self::Quantity(quantity) => write_quantity(quantity, formatter),
            Self::List(item) => write!(formatter, "List<{item}>"),
            Self::Optional(inner) => write!(formatter, "{inner}?"),
            Self::Record(record) => {
                if let Some(name) = &record.name {
                    return formatter.write_str(name.rsplit('.').next().unwrap_or(name));
                }
                formatter.write_str("{ ")?;
                for (index, field) in record.fields.iter().enumerate() {
                    if index > 0 {
                        formatter.write_str(", ")?;
                    }
                    write!(formatter, "{}: {}", field.name, field.ty)?;
                }
                formatter.write_str(" }")
            }
            Self::Enum(enumeration) => formatter.write_str(
                enumeration
                    .name
                    .rsplit('.')
                    .next()
                    .unwrap_or(&enumeration.name),
            ),
            Self::Never => formatter.write_str("Never"),
            Self::Error => formatter.write_str("?"),
        }
    }
}

/// `Money`, `Money/cwt` (a money rate shown per unit), `Weight`, `Weight/Volume`.
fn write_quantity(quantity: &QuantityType, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
    let money_rate = quantity.dim.exponent(Base::Money) == 1;
    if let (true, Some(unit)) = (money_rate, &quantity.display) {
        let denominator: Vec<String> = unit
            .factors()
            .iter()
            .filter(|(_, exponent)| *exponent < 0)
            .map(|(simple, _)| simple.symbol.to_owned())
            .collect();
        if !denominator.is_empty() {
            return write!(formatter, "Money/{}", denominator.join("/"));
        }
    }
    write!(formatter, "{}", quantity.dim)
}

#[cfg(test)]
mod tests {
    use super::Type;
    use crate::tariff::units::{Base, Unit};

    fn unit(text: &str) -> Unit {
        Unit::parse(text).expect("a unit")
    }

    #[test]
    fn display_units_do_not_change_the_type() {
        let per_cwt = Type::quantity_in(&unit("$/cwt"));
        let per_lb = Type::quantity_in(&unit("$/lb"));
        assert_eq!(per_cwt, per_lb);
        assert_eq!(per_cwt.to_string(), "Money/cwt");
        assert_ne!(per_cwt, Type::quantity_in(&unit("$/mi")));
    }

    #[test]
    fn quantities_print_their_dimension() {
        assert_eq!(Type::money().to_string(), "Money");
        assert_eq!(Type::of(Base::Weight).to_string(), "Weight");
        assert_eq!(
            Type::quantity_in(&unit("lb/ft3")).to_string(),
            "Weight/Volume"
        );
        assert_eq!(
            Type::quantity_in(&unit("$/pallet")).to_string(),
            "Money/pallet"
        );
        assert_eq!(Type::quantity_in(&unit("$/pallet")), Type::money());
    }

    #[test]
    fn subtyping_is_int_to_number_and_t_to_optional() {
        assert!(Type::Number.accepts(&Type::Int));
        assert!(!Type::Int.accepts(&Type::Number));
        assert!(Type::optional(Type::money()).accepts(&Type::money()));
        assert!(!Type::money().accepts(&Type::optional(Type::money())));
        assert!(Type::list(Type::Number).accepts(&Type::list(Type::Int)));
        assert!(Type::money().accepts(&Type::Never));
    }

    #[test]
    fn joins_widen_branches() {
        assert_eq!(Type::Int.join(&Type::Number), Some(Type::Number));
        assert_eq!(
            Type::optional(Type::Int).join(&Type::Number),
            Some(Type::optional(Type::Number))
        );
        assert_eq!(Type::Text.join(&Type::Int), None);
    }
}
