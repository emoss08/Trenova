//! Every property and method a value has, written once as data.
//!
//! The checker types calls from this table, `tariff.catalog` publishes it, and the editor's
//! completion and signature help read it. Each entry names its receiver, its parameters (with
//! their defaults as written), what it returns and a doc, plus the [`Rule`] the checker applies
//! where the result depends on the receiver or a lambda (`sum` of money is money).
//!
//! The list methods agree with the evaluator on these rules:
//!
//! - A lambda takes the item, or two parameters `(item, index)`, the index counted from 0.
//! - `min` and `max` with a lambda give the lambda's smallest or largest value, never the item
//!   that has it, and without one the smallest or largest item; either way optional (`U?`),
//!   `none` for an empty list.
//! - `sum` of an empty list is the zero of its type (`$0`, `0 h`, `0`).
//! - `avg` is not optional: a `Number` for numbers (whole numbers too), else the values' own
//!   kind (`Money`, `Weight`). An empty list stops the rating with an `argument` error.

use crate::message;
use crate::tariff::types::Type;
use crate::tariff::units::Dim;

/// The one property that reads the record (`rating.holidays`), so it is refused where the
/// record can't be read.
pub const HOLIDAY: &str = "holiday";

/// The kinds of value that have properties and methods.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Receiver {
    /// `Int` and `Number`.
    Number,
    /// `Percent`.
    Percent,
    /// `Money`.
    Money,
    /// Quantities other than money: weights, distances, durations, rates, densities.
    Quantity,
    /// `Text`.
    Text,
    /// `Date`.
    Date,
    /// `DateTime`.
    DateTime,
    /// `Time`.
    Time,
    /// The `FreightClass` schema enum.
    FreightClass,
    /// `List<T>`.
    List,
}

impl Receiver {
    /// The receiver as the catalog names it.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Number => "Number",
            Self::Percent => "Percent",
            Self::Money => "Money",
            Self::Quantity => "Quantity",
            Self::Text => "Text",
            Self::Date => "Date",
            Self::DateTime => "DateTime",
            Self::Time => "Time",
            Self::FreightClass => "FreightClass",
            Self::List => "List<T>",
        }
    }

    /// The receiver a value of a type is, when it has properties or methods.
    #[must_use]
    pub fn of(ty: &Type) -> Option<Self> {
        match ty {
            Type::Int | Type::Number => Some(Self::Number),
            Type::Percent => Some(Self::Percent),
            Type::Quantity(quantity) if quantity.dim == Dim::money() => Some(Self::Money),
            Type::Quantity(_) => Some(Self::Quantity),
            Type::Text => Some(Self::Text),
            Type::Date => Some(Self::Date),
            Type::DateTime => Some(Self::DateTime),
            Type::Time => Some(Self::Time),
            Type::Enum(enumeration) if enumeration.name == "FreightClass" => {
                Some(Self::FreightClass)
            }
            Type::List(_) => Some(Self::List),
            _ => None,
        }
    }
}

/// What a parameter takes.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum ParamKind {
    /// An `Int`.
    Int,
    /// `Text`.
    Text,
    /// A value of the receiver's own kind (`min(x)`, `roundUp(to: q)`).
    Same,
    /// A unit written as names: `cwt`, `lb/ft3`.
    Unit,
    /// A lambda from an item to `Bool`.
    Predicate,
    /// A lambda from an item to any value.
    Map,
    /// A lambda from an item to a value that sums (numbers, money, quantities).
    Summable,
    /// A lambda from an item to a value that has an order.
    Ordered,
    /// A lambda from an item to any value, compared for equality.
    Key,
}

/// One parameter of a method.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Param {
    /// Its name, also its name at a call site.
    pub name: &'static str,
    /// Its type as the catalog shows it.
    pub ty: &'static str,
    /// What it takes, for the checker.
    pub kind: ParamKind,
    /// Its default, as written, when it may be left out.
    pub default: Option<&'static str>,
}

/// How the checker types a method's result.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Rule {
    /// The receiver's type (joined with `Same` arguments, so `Int.max(Number)` is a `Number`).
    Same,
    /// `Int`.
    Int,
    /// `Number`.
    Number,
    /// `Bool`.
    Bool,
    /// `Text`.
    Text,
    /// `List<Text>`.
    TextList,
    /// `Date`.
    Date,
    /// `Weekday`.
    Weekday,
    /// `Time`.
    Time,
    /// `Int` without `places`, `Number` with it.
    Round,
    /// The receiver converted to the unit argument.
    In,
    /// The list itself: `List<T>`.
    List,
    /// The lambda's values, or the items: `List<U>`.
    Map,
    /// An item, or none: `T?`.
    Item,
    /// The sum of the lambda's values, or of the items: `U`.
    Sum,
    /// The smallest or largest of the lambda's values, or of the items: `U?`.
    Extreme,
    /// The average of the lambda's values, or of the items: `U`, with `Int` averaging to a
    /// `Number`.
    Average,
    /// `List<{ key: K, items: List<T> }>`.
    Group,
}

/// One property or method.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Method {
    /// Its name.
    pub name: &'static str,
    /// What it is a property or method of.
    pub receiver: Receiver,
    /// A property is read without parentheses.
    pub property: bool,
    /// The parameters, in order.
    pub params: &'static [Param],
    /// What it returns, as the catalog shows it.
    pub returns: &'static str,
    /// What it does: an English message template.
    pub doc: &'static str,
    /// How the checker types the result.
    pub rule: Rule,
}

impl Method {
    /// The signature as the catalog and signature help show it: `roundUp(to: Money) -> Money`,
    /// or `len: Int` for a property.
    #[must_use]
    pub fn signature(&self) -> String {
        if self.property {
            return format!("{}: {}", self.name, self.returns);
        }
        let params: Vec<String> = self
            .params
            .iter()
            .map(|param| match param.default {
                Some(default) => format!("{}: {} = {default}", param.name, param.ty),
                None => format!("{}: {}", param.name, param.ty),
            })
            .collect();
        format!("{}({}) -> {}", self.name, params.join(", "), self.returns)
    }

    /// The fewest arguments a call needs.
    #[must_use]
    pub fn required(&self) -> usize {
        self.params
            .iter()
            .filter(|param| param.default.is_none())
            .count()
    }
}

const fn param(name: &'static str, ty: &'static str, kind: ParamKind) -> Param {
    Param {
        name,
        ty,
        kind,
        default: None,
    }
}

const fn optional(
    name: &'static str,
    ty: &'static str,
    kind: ParamKind,
    default: &'static str,
) -> Param {
    Param {
        name,
        ty,
        kind,
        default: Some(default),
    }
}

const fn property(
    name: &'static str,
    receiver: Receiver,
    returns: &'static str,
    rule: Rule,
    doc: &'static str,
) -> Method {
    Method {
        name,
        receiver,
        property: true,
        params: &[],
        returns,
        doc,
        rule,
    }
}

const fn method(
    name: &'static str,
    receiver: Receiver,
    params: &'static [Param],
    returns: &'static str,
    rule: Rule,
    doc: &'static str,
) -> Method {
    Method {
        name,
        receiver,
        property: false,
        params,
        returns,
        doc,
        rule,
    }
}

const PLACES: &[Param] = &[optional("places", "Int", ParamKind::Int, "0")];
const SAME_X: &[Param] = &[param("x", "Self", ParamKind::Same)];
const CLAMP: &[Param] = &[
    param("lo", "Self", ParamKind::Same),
    param("hi", "Self", ParamKind::Same),
];
const TO_MONEY: &[Param] = &[param("to", "Money", ParamKind::Same)];
const TO_SAME: &[Param] = &[param("to", "Self", ParamKind::Same)];
const TEXT_T: &[Param] = &[param("t", "Text", ParamKind::Text)];
const DAYS: &[Param] = &[param("n", "Int", ParamKind::Int)];
const PREDICATE: &[Param] = &[param("f", "T => Bool", ParamKind::Predicate)];
const MAYBE_PREDICATE: &[Param] = &[optional("f", "T => Bool", ParamKind::Predicate, "none")];
const SUMMED: &[Param] = &[optional("f", "T => U", ParamKind::Summable, "the item")];
const ORDERED_BY: &[Param] = &[optional("f", "T => U", ParamKind::Ordered, "the item")];
const COUNT: &[Param] = &[param("n", "Int", ParamKind::Int)];

use ParamKind as K;
use Receiver as R;

/// Every property and method, by receiver.
pub static METHODS: &[Method] = &[
    method(
        "round",
        R::Number,
        PLACES,
        "Int | Number",
        Rule::Round,
        message!(
            "Rounds half away from zero: to a whole `Int`, or to `places` decimals as a `Number`."
        ),
    ),
    method(
        "roundHalfEven",
        R::Number,
        PLACES,
        "Int | Number",
        Rule::Round,
        message!(
            "Rounds half to even (banker's rounding): to a whole `Int`, or to `places` decimals as a `Number`."
        ),
    ),
    method(
        "floor",
        R::Number,
        &[],
        "Int",
        Rule::Int,
        message!("The largest whole number at or below it."),
    ),
    method(
        "ceil",
        R::Number,
        &[],
        "Int",
        Rule::Int,
        message!("The smallest whole number at or above it."),
    ),
    method(
        "abs",
        R::Number,
        &[],
        "Self",
        Rule::Same,
        message!("The value without its sign."),
    ),
    method(
        "min",
        R::Number,
        SAME_X,
        "Self",
        Rule::Same,
        message!("The smaller of it and `x`."),
    ),
    method(
        "max",
        R::Number,
        SAME_X,
        "Self",
        Rule::Same,
        message!("The larger of it and `x`."),
    ),
    method(
        "clamp",
        R::Number,
        CLAMP,
        "Self",
        Rule::Same,
        message!("It, kept between `lo` and `hi`."),
    ),
    method(
        "pow",
        R::Number,
        &[param("n", "Int", K::Int)],
        "Self",
        Rule::Same,
        message!("It raised to the whole power `n`."),
    ),
    method(
        "sqrt",
        R::Number,
        &[],
        "Number",
        Rule::Number,
        message!("Its square root, to 28 significant digits."),
    ),
    property(
        "value",
        R::Percent,
        "Number",
        Rule::Number,
        message!("The bare number of percent: `18%.value` is `18`."),
    ),
    method(
        "round",
        R::Money,
        &[],
        "Money",
        Rule::Same,
        message!("Rounds to the cent, half away from zero."),
    ),
    method(
        "roundHalfEven",
        R::Money,
        &[],
        "Money",
        Rule::Same,
        message!("Rounds to the cent, half to even."),
    ),
    method(
        "roundUp",
        R::Money,
        TO_MONEY,
        "Money",
        Rule::Same,
        message!("Rounds up to the next multiple of `to`: `roundUp(to: $5)`."),
    ),
    method(
        "roundDown",
        R::Money,
        TO_MONEY,
        "Money",
        Rule::Same,
        message!("Rounds down to a multiple of `to`: `roundDown(to: $1)`."),
    ),
    method(
        "abs",
        R::Money,
        &[],
        "Money",
        Rule::Same,
        message!("The amount without its sign."),
    ),
    method(
        "min",
        R::Money,
        &[param("x", "Money", K::Same)],
        "Money",
        Rule::Same,
        message!("The smaller of it and `x`."),
    ),
    method(
        "max",
        R::Money,
        &[param("x", "Money", K::Same)],
        "Money",
        Rule::Same,
        message!("The larger of it and `x`."),
    ),
    method(
        "clamp",
        R::Money,
        &[param("lo", "Money", K::Same), param("hi", "Money", K::Same)],
        "Money",
        Rule::Same,
        message!("The amount, kept between `lo` and `hi`."),
    ),
    property(
        "value",
        R::Quantity,
        "Number",
        Rule::Number,
        message!("The bare number, in the unit it was last written or converted to."),
    ),
    method(
        "in",
        R::Quantity,
        &[param("unit", "unit", K::Unit)],
        "Self",
        Rule::In,
        message!("The same quantity in another unit of its dimension: `shipment.weight.in(cwt)`."),
    ),
    method(
        "round",
        R::Quantity,
        TO_SAME,
        "Self",
        Rule::Same,
        message!("Rounds half away from zero to a multiple of `to`."),
    ),
    method(
        "roundUp",
        R::Quantity,
        TO_SAME,
        "Self",
        Rule::Same,
        message!(
            "Rounds up to the next multiple of `to`: weight breaks bill to the next hundred with `roundUp(to: 100 lb)`."
        ),
    ),
    method(
        "roundDown",
        R::Quantity,
        TO_SAME,
        "Self",
        Rule::Same,
        message!("Rounds down to a multiple of `to`."),
    ),
    method(
        "abs",
        R::Quantity,
        &[],
        "Self",
        Rule::Same,
        message!("The quantity without its sign."),
    ),
    method(
        "min",
        R::Quantity,
        SAME_X,
        "Self",
        Rule::Same,
        message!("The smaller of it and `x`."),
    ),
    method(
        "max",
        R::Quantity,
        SAME_X,
        "Self",
        Rule::Same,
        message!("The larger of it and `x`."),
    ),
    method(
        "clamp",
        R::Quantity,
        CLAMP,
        "Self",
        Rule::Same,
        message!("The quantity, kept between `lo` and `hi`."),
    ),
    property(
        "len",
        R::Text,
        "Int",
        Rule::Int,
        message!("How many characters it has."),
    ),
    property(
        "isEmpty",
        R::Text,
        "Bool",
        Rule::Bool,
        message!("Whether it has no characters."),
    ),
    method(
        "upper",
        R::Text,
        &[],
        "Text",
        Rule::Text,
        message!("The text in upper case."),
    ),
    method(
        "lower",
        R::Text,
        &[],
        "Text",
        Rule::Text,
        message!("The text in lower case."),
    ),
    method(
        "trim",
        R::Text,
        &[],
        "Text",
        Rule::Text,
        message!("The text without spaces at either end."),
    ),
    method(
        "startsWith",
        R::Text,
        TEXT_T,
        "Bool",
        Rule::Bool,
        message!("Whether it starts with `t`."),
    ),
    method(
        "endsWith",
        R::Text,
        TEXT_T,
        "Bool",
        Rule::Bool,
        message!("Whether it ends with `t`."),
    ),
    method(
        "contains",
        R::Text,
        TEXT_T,
        "Bool",
        Rule::Bool,
        message!("Whether `t` appears in it."),
    ),
    method(
        "replace",
        R::Text,
        &[param("a", "Text", K::Text), param("b", "Text", K::Text)],
        "Text",
        Rule::Text,
        message!("The text with every `a` replaced by `b`."),
    ),
    method(
        "split",
        R::Text,
        &[param("sep", "Text", K::Text)],
        "List<Text>",
        Rule::TextList,
        message!("The parts between each `sep`."),
    ),
    method(
        "matches",
        R::Text,
        &[param("pattern", "Text", K::Text)],
        "Bool",
        Rule::Bool,
        message!("Whether the text matches a regular expression, checked in linear time."),
    ),
    property("year", R::Date, "Int", Rule::Int, message!("The year.")),
    property(
        "month",
        R::Date,
        "Int",
        Rule::Int,
        message!("The month, 1 to 12."),
    ),
    property(
        "day",
        R::Date,
        "Int",
        Rule::Int,
        message!("The day of the month."),
    ),
    property(
        "weekday",
        R::Date,
        "Weekday",
        Rule::Weekday,
        message!("The day of the week, `Mon` to `Sun`."),
    ),
    property(
        "weekend",
        R::Date,
        "Bool",
        Rule::Bool,
        message!("Whether it falls on a Saturday or Sunday."),
    ),
    property(
        "holiday",
        R::Date,
        "Bool",
        Rule::Bool,
        message!(
            "Whether it is one of the organisation's holidays (`rating.holidays`). It reads the record, so a function can't use it."
        ),
    ),
    method(
        "addDays",
        R::Date,
        DAYS,
        "Date",
        Rule::Same,
        message!("The date `n` days later (earlier when `n` is negative)."),
    ),
    property(
        "year",
        R::DateTime,
        "Int",
        Rule::Int,
        message!("The year, in the date-time's own offset."),
    ),
    property(
        "month",
        R::DateTime,
        "Int",
        Rule::Int,
        message!("The month, 1 to 12, in the date-time's own offset."),
    ),
    property(
        "day",
        R::DateTime,
        "Int",
        Rule::Int,
        message!("The day of the month, in the date-time's own offset."),
    ),
    property(
        "hour",
        R::DateTime,
        "Int",
        Rule::Int,
        message!("The hour, 0 to 23, in the date-time's own offset."),
    ),
    property(
        "minute",
        R::DateTime,
        "Int",
        Rule::Int,
        message!("The minute."),
    ),
    property(
        "date",
        R::DateTime,
        "Date",
        Rule::Date,
        message!("The calendar date, in the date-time's own offset."),
    ),
    property(
        "time",
        R::DateTime,
        "Time",
        Rule::Time,
        message!(
            "The time of day, in the date-time's own offset: the clock at the stop, as in `stop.arrival?.time`."
        ),
    ),
    property(
        "weekday",
        R::DateTime,
        "Weekday",
        Rule::Weekday,
        message!("The day of the week, in the date-time's own offset."),
    ),
    property(
        "weekend",
        R::DateTime,
        "Bool",
        Rule::Bool,
        message!("Whether it falls on a Saturday or Sunday, in the date-time's own offset."),
    ),
    property(
        "holiday",
        R::DateTime,
        "Bool",
        Rule::Bool,
        message!(
            "Whether its date, in the date-time's own offset, is one of the organisation's holidays (`rating.holidays`). It reads the record, so a function can't use it."
        ),
    ),
    method(
        "addDays",
        R::DateTime,
        DAYS,
        "DateTime",
        Rule::Same,
        message!("The same time `n` days later."),
    ),
    method(
        "addHours",
        R::DateTime,
        DAYS,
        "DateTime",
        Rule::Same,
        message!("The instant `n` hours later."),
    ),
    method(
        "addMinutes",
        R::DateTime,
        DAYS,
        "DateTime",
        Rule::Same,
        message!("The instant `n` minutes later (earlier when `n` is negative)."),
    ),
    property(
        "hour",
        R::Time,
        "Int",
        Rule::Int,
        message!("The hour, 0 to 23."),
    ),
    property(
        "minute",
        R::Time,
        "Int",
        Rule::Int,
        message!("The minute, 0 to 59."),
    ),
    property(
        "number",
        R::FreightClass,
        "Number",
        Rule::Number,
        message!("The class as a number: `Class77_5.number` is `77.5`."),
    ),
    property(
        "len",
        R::List,
        "Int",
        Rule::Int,
        message!("How many items it has."),
    ),
    property(
        "isEmpty",
        R::List,
        "Bool",
        Rule::Bool,
        message!("Whether it has no items."),
    ),
    method(
        "where",
        R::List,
        PREDICATE,
        "List<T>",
        Rule::List,
        message!("The items `f` holds for, in order: `stops.where(s => s.kind == Delivery)`."),
    ),
    method(
        "map",
        R::List,
        &[param("f", "T => U", K::Map)],
        "List<U>",
        Rule::Map,
        message!("Each item's value of `f`, in order."),
    ),
    method(
        "count",
        R::List,
        MAYBE_PREDICATE,
        "Int",
        Rule::Int,
        message!("How many items `f` holds for, or how many there are."),
    ),
    method(
        "sum",
        R::List,
        SUMMED,
        "U",
        Rule::Sum,
        message!(
            "The total of `f` over the items, or of the items: `stops.sum(s => s.dwell ?? 0 h)` is a duration. An empty list sums to zero."
        ),
    ),
    method(
        "min",
        R::List,
        ORDERED_BY,
        "U?",
        Rule::Extreme,
        message!(
            "The smallest value of `f` over the items (the value, not the item: `stops.min(s => s.sequence)` is an `Int?`), or the smallest item; `none` for an empty list."
        ),
    ),
    method(
        "max",
        R::List,
        ORDERED_BY,
        "U?",
        Rule::Extreme,
        message!(
            "The largest value of `f` over the items (the value, not the item: `stops.max(s => s.dwell ?? 0 h)` is a `Duration?`), or the largest item; `none` for an empty list."
        ),
    ),
    method(
        "avg",
        R::List,
        SUMMED,
        "U",
        Rule::Average,
        message!(
            "The average of `f` over the items, or of the items: a `Number` for numbers (whole numbers too), else the values' own kind. An empty list has no average, so it stops the rating with an `argument` error."
        ),
    ),
    method(
        "any",
        R::List,
        PREDICATE,
        "Bool",
        Rule::Bool,
        message!("Whether `f` holds for at least one item."),
    ),
    method(
        "all",
        R::List,
        PREDICATE,
        "Bool",
        Rule::Bool,
        message!("Whether `f` holds for every item (true for an empty list)."),
    ),
    method(
        "first",
        R::List,
        MAYBE_PREDICATE,
        "T?",
        Rule::Item,
        message!("The first item, or the first `f` holds for; `none` when there is none."),
    ),
    method(
        "last",
        R::List,
        MAYBE_PREDICATE,
        "T?",
        Rule::Item,
        message!("The last item, or the last `f` holds for; `none` when there is none."),
    ),
    method(
        "find",
        R::List,
        PREDICATE,
        "T?",
        Rule::Item,
        message!("The first item `f` holds for, or `none`."),
    ),
    method(
        "sortBy",
        R::List,
        &[param("f", "T => K", K::Ordered)],
        "List<T>",
        Rule::List,
        message!(
            "The items ordered by `f`, smallest first; items with equal keys keep their order."
        ),
    ),
    method(
        "take",
        R::List,
        COUNT,
        "List<T>",
        Rule::List,
        message!("The first `n` items."),
    ),
    method(
        "skip",
        R::List,
        COUNT,
        "List<T>",
        Rule::List,
        message!("The items after the first `n`."),
    ),
    method(
        "reverse",
        R::List,
        &[],
        "List<T>",
        Rule::List,
        message!("The items in reverse order."),
    ),
    method(
        "distinct",
        R::List,
        &[],
        "List<T>",
        Rule::List,
        message!("The items with repeats removed, first occurrence kept."),
    ),
    method(
        "groupBy",
        R::List,
        &[param("f", "T => K", K::Key)],
        "List<{ key: K, items: List<T> }>",
        Rule::Group,
        message!("The items grouped by `f`, groups in order of first appearance."),
    ),
];

/// The properties and methods of a receiver.
pub fn of(receiver: Receiver) -> impl Iterator<Item = &'static Method> {
    METHODS
        .iter()
        .filter(move |method| method.receiver == receiver)
}

/// The property or method `name` of a value of type `ty`.
#[must_use]
pub fn find(ty: &Type, name: &str) -> Option<&'static Method> {
    let receiver = Receiver::of(ty)?;
    of(receiver).find(|method| method.name == name)
}

#[cfg(test)]
mod tests {
    use super::{METHODS, find};
    use crate::tariff::types::Type;

    #[test]
    fn signatures_read_as_written() {
        let round_up = find(&Type::money(), "roundUp").map(super::Method::signature);
        assert_eq!(round_up.as_deref(), Some("roundUp(to: Money) -> Money"));
        let len = find(&Type::Text, "len").map(super::Method::signature);
        assert_eq!(len.as_deref(), Some("len: Int"));
        assert!(find(&Type::Bool, "len").is_none());
    }

    #[test]
    fn no_receiver_has_a_name_twice() {
        for (index, method) in METHODS.iter().enumerate() {
            assert!(
                !METHODS
                    .iter()
                    .skip(index + 1)
                    .any(|other| other.receiver == method.receiver && other.name == method.name),
                "{} is listed twice",
                method.name
            );
        }
    }
}
