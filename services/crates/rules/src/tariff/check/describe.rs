//! Types and code in pricing words, for the checker's messages: `money per hundredweight`, `a
//! weight`, `` `shipment.weight` ``. Every phrase is a [`Message`] of its own, so a sentence
//! built from phrases still translates part by part.

use crate::tariff::types::{QuantityType, Type};
use crate::tariff::units::{Base, Dim, Simple, Unit};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

/// The longest snippet of source quoted in a message, in characters.
const LONGEST_QUOTE: usize = 48;

/// Code quoted for a message: `` `rate * miles` ``, shortened with `…` past
/// `LONGEST_QUOTE` characters and kept on one line.
#[must_use]
pub fn quote(code: &str) -> Arg {
    let flat: String = code.split_whitespace().collect::<Vec<_>>().join(" ");
    let short: String = if flat.chars().count() > LONGEST_QUOTE {
        let mut cut: String = flat.chars().take(LONGEST_QUOTE - 1).collect();
        cut.push('…');
        cut
    } else {
        flat
    };
    Arg::Text(format!("`{short}`"))
}

/// The source text a span covers, or nothing when the span is outside it.
#[must_use]
pub fn snippet(source: &str, span: Span) -> &str {
    let start = usize::try_from(span.start).unwrap_or(usize::MAX);
    let end = usize::try_from(span.end).unwrap_or(usize::MAX);
    source.get(start..end).unwrap_or_default()
}

/// The code a span covers, quoted.
#[must_use]
pub fn quote_span(source: &str, span: Span) -> Arg {
    quote(snippet(source, span))
}

/// A type in pricing words: `money`, `money per hundredweight`, `a weight`, `a whole number`,
/// `a weight that may be empty`.
#[must_use]
pub fn words(ty: &Type) -> Message {
    match ty {
        Type::Bool => Message::new(message!("true or false")),
        Type::Int => Message::new(message!("a whole number")),
        Type::Number => Message::new(message!("a number")),
        Type::Percent => Message::new(message!("a percent")),
        Type::Text => Message::new(message!("text")),
        Type::Date => Message::new(message!("a date")),
        Type::DateTime => Message::new(message!("a date and time")),
        Type::Time => Message::new(message!("a time of day")),
        Type::Quantity(quantity) => quantity_words(quantity),
        Type::List(_) => Message::with(message!("a list ({0})"), [quote(&ty.to_string())]),
        Type::Optional(inner) if matches!(**inner, Type::Never) => Message::new(message!("`none`")),
        Type::Optional(inner) => {
            Message::with(message!("{0} that may be empty"), [Arg::from(words(inner))])
        }
        Type::Record(record) => match &record.name {
            Some(_) => Message::with(message!("a record of type {0}"), [quote(&ty.to_string())]),
            None => Message::with(message!("a record ({0})"), [quote(&ty.to_string())]),
        },
        Type::Enum(_) => Message::with(message!("a value of type {0}"), [quote(&ty.to_string())]),
        Type::Never => Message::new(message!("a refusal to rate")),
        Type::Error => Message::new(message!("an unknown value")),
    }
}

/// A type in pricing words with declared records and enums named in full
/// (`org.other.Break`), for telling apart two types whose short names are the same.
#[must_use]
pub fn qualified_words(ty: &Type) -> Message {
    match ty {
        Type::Record(record) => match &record.name {
            Some(name) => Message::with(message!("a record of type {0}"), [quote(name)]),
            None => words(ty),
        },
        Type::Enum(enumeration) => {
            Message::with(message!("a value of type {0}"), [quote(&enumeration.name)])
        }
        Type::Optional(inner) => Message::with(
            message!("{0} that may be empty"),
            [Arg::from(qualified_words(inner))],
        ),
        Type::List(item) => Message::with(
            message!("a list of {0}"),
            [Arg::from(qualified_words(item))],
        ),
        other => words(other),
    }
}

fn quantity_words(quantity: &QuantityType) -> Message {
    let dim = quantity.dim;
    if dim == Dim::money() {
        return Message::new(message!("money"));
    }
    if let Some(base) = single(dim) {
        return base_words(base);
    }
    if dim.exponent(Base::Money) == 1 {
        let per = quantity
            .display
            .as_ref()
            .and_then(denominator)
            .map(unit_noun)
            .or_else(|| {
                Dim::money().per(dim).and_then(single).map(|base| {
                    Message::with(message!("unit of {0}"), [Arg::from(base_noun(base))])
                })
            });
        if let Some(per) = per {
            return Message::with(message!("money per {0}"), [Arg::from(per)]);
        }
    }
    if dim == density() {
        return Message::new(message!("a density"));
    }
    if let Some((top, bottom)) = quotient(dim) {
        return Message::with(
            message!("{0} per {1}"),
            [Arg::from(base_noun(top)), Arg::from(base_noun(bottom))],
        );
    }
    Message::with(
        message!("a quantity measured in {0}"),
        [Arg::Text(dim.to_string())],
    )
}

/// `Weight/Volume`.
#[must_use]
pub fn density() -> Dim {
    Dim::of(Base::Weight)
        .per(Dim::of(Base::Volume))
        .unwrap_or(Dim::NONE)
}

/// The base a dimension is made of when it is one base to the first power.
#[must_use]
pub fn single(dim: Dim) -> Option<Base> {
    let mut found = None;
    for base in BASES {
        match dim.exponent(base) {
            0 => {}
            1 if found.is_none() => found = Some(base),
            _ => return None,
        }
    }
    found
}

/// The two bases of a dimension that is one base per another.
#[must_use]
pub fn quotient(dim: Dim) -> Option<(Base, Base)> {
    let mut top = None;
    let mut bottom = None;
    for base in BASES {
        match dim.exponent(base) {
            0 => {}
            1 if top.is_none() => top = Some(base),
            -1 if bottom.is_none() => bottom = Some(base),
            _ => return None,
        }
    }
    top.zip(bottom)
}

/// Every base dimension, in `Base`'s order.
pub const BASES: [Base; 7] = [
    Base::Money,
    Base::Weight,
    Base::Distance,
    Base::Length,
    Base::Volume,
    Base::Duration,
    Base::Temperature,
];

/// The unit a quantity's display unit is per: `cwt` for `$/cwt`.
fn denominator(unit: &Unit) -> Option<&'static Simple> {
    let below: Vec<&'static Simple> = unit
        .factors()
        .iter()
        .filter(|(_, exponent)| *exponent < 0)
        .map(|(simple, _)| *simple)
        .collect();
    match below.as_slice() {
        [one] => Some(*one),
        _ => None,
    }
}

/// `a weight`, `a distance`.
#[must_use]
pub fn base_words(base: Base) -> Message {
    Message::new(match base {
        Base::Money => message!("money"),
        Base::Weight => message!("a weight"),
        Base::Distance => message!("a distance"),
        Base::Length => message!("a length"),
        Base::Volume => message!("a volume"),
        Base::Duration => message!("a duration"),
        Base::Temperature => message!("a temperature"),
    })
}

/// `weight`, `distance`: the dimension as a bare noun.
#[must_use]
pub fn base_noun(base: Base) -> Message {
    Message::new(match base {
        Base::Money => message!("money"),
        Base::Weight => message!("weight"),
        Base::Distance => message!("distance"),
        Base::Length => message!("length"),
        Base::Volume => message!("volume"),
        Base::Duration => message!("time"),
        Base::Temperature => message!("temperature"),
    })
}

/// A unit's name in words, singular: `hundredweight`, `mile`, `pallet`.
#[must_use]
pub fn unit_noun(unit: &Simple) -> Message {
    Message::new(match unit.symbol {
        "$" => message!("dollar"),
        "lb" => message!("pound"),
        "cwt" => message!("hundredweight"),
        "kg" => message!("kilogram"),
        "t" => message!("metric tonne"),
        "ton" => message!("short ton"),
        "mi" => message!("mile"),
        "km" => message!("kilometre"),
        "in" => message!("inch"),
        "ft" => message!("foot"),
        "m" => message!("metre"),
        "ft3" => message!("cubic foot"),
        "m3" => message!("cubic metre"),
        "gal" => message!("gallon"),
        "l" => message!("litre"),
        "min" => message!("minute"),
        "h" => message!("hour"),
        "d" => message!("day"),
        "°F" => message!("degree Fahrenheit"),
        "°C" => message!("degree Celsius"),
        "%" => message!("percent"),
        "pallet" => message!("pallet"),
        "piece" => message!("piece"),
        _ => message!("stop"),
    })
}

/// The unit a value of a dimension is written in when nothing says otherwise: pounds, miles,
/// feet, cubic feet, hours and °F, the units the domain stores.
#[must_use]
pub fn default_unit(base: Base) -> &'static str {
    match base {
        Base::Money => "$",
        Base::Weight => "lb",
        Base::Distance => "mi",
        Base::Length => "ft",
        Base::Volume => "ft3",
        Base::Duration => "h",
        Base::Temperature => "°F",
    }
}

/// The unit a quantity type's values are written in: its display unit, else one built from
/// each base's default unit (`$/lb` for `Money/Weight`). `None` for a dimension no unit text
/// can spell (a product of two bases).
#[must_use]
pub fn unit_text(quantity: &QuantityType) -> Option<String> {
    if let Some(unit) = &quantity.display
        && !unit.is_none()
    {
        return Some(unit.to_string());
    }
    let dim = quantity.dim;
    if let Some(base) = single(dim) {
        return Some(default_unit(base).to_owned());
    }
    quotient(dim).map(|(top, bottom)| format!("{}/{}", default_unit(top), default_unit(bottom)))
}

/// A zero of a type, as source text an author would write for a fallback: `0 lb`, `$0`, `""`,
/// `false`, `[]`. `None` for a type with no natural zero (dates, enums, records).
#[must_use]
pub fn zero(ty: &Type) -> Option<String> {
    match ty {
        Type::Bool => Some(String::from("false")),
        Type::Int | Type::Number => Some(String::from("0")),
        Type::Percent => Some(String::from("0%")),
        Type::Text => Some(String::from("\"\"")),
        Type::List(_) => Some(String::from("[]")),
        Type::Quantity(quantity) => {
            if quantity.dim == Dim::money() {
                return Some(String::from("$0"));
            }
            let unit = unit_text(quantity)?;
            match unit.strip_prefix("$/") {
                Some(per) => Some(format!("$0/{per}")),
                None => Some(format!("0 {unit}")),
            }
        }
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::{words, zero};
    use crate::tariff::types::Type;
    use crate::tariff::units::{Base, Unit};

    fn unit(text: &str) -> Unit {
        Unit::parse(text).unwrap_or_default()
    }

    #[test]
    fn rates_read_as_money_per_unit() {
        let rate = Type::quantity_in(&unit("$/cwt"));
        assert_eq!(words(&rate).english(), "money per hundredweight");
        assert_eq!(words(&Type::of(Base::Weight)).english(), "a weight");
        assert_eq!(
            words(&Type::optional(Type::of(Base::Weight))).english(),
            "a weight that may be empty"
        );
        assert_eq!(
            words(&Type::quantity(
                crate::tariff::units::Dim::money()
                    .per(crate::tariff::units::Dim::of(Base::Distance))
                    .unwrap_or_default()
            ))
            .english(),
            "money per unit of distance"
        );
        assert_eq!(
            words(&Type::quantity_in(&unit("lb/ft3"))).english(),
            "a density"
        );
    }

    #[test]
    fn zeros_carry_the_unit() {
        assert_eq!(
            zero(&Type::quantity_in(&unit("lb"))).as_deref(),
            Some("0 lb")
        );
        assert_eq!(zero(&Type::of(Base::Duration)).as_deref(), Some("0 h"));
        assert_eq!(zero(&Type::money()).as_deref(), Some("$0"));
        assert_eq!(
            zero(&Type::quantity_in(&unit("$/mi"))).as_deref(),
            Some("$0/mi")
        );
        assert_eq!(zero(&Type::Date), None);
    }
}
