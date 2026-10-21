//! Run-time values.
//!
//! There is one number type, an exact `rust_decimal::Decimal`; text is shared `Rc<str>`; lists
//! and maps are shared behind `Rc`, so passing the environment, a field or a list to a function
//! is a reference-count bump, never a copy. The Go engine deep-copied the whole environment for
//! every evaluation (`maputils.CloneDeep`); nothing here does.
//!
//! Maps keep their keys sorted (`BTreeMap`), so `keys`, `values`, `toPairs`, printing and traces
//! never depend on hash order.
//!
//! Every list and map records how deeply it nests, and none may nest deeper than [`MAX_DEPTH`].
//! That one invariant bounds every recursive walk over a value (equality, ordering, printing,
//! traces, encoding) without a stack check in each: the WebAssembly stack is 1 MiB and a formula
//! like `reduce(1..1000, [#acc], [])` would otherwise build a value a thousand levels deep.

use super::time::Date;
use super::time::format_duration;
use rust_decimal::Decimal;
use std::cmp::Ordering;
use std::collections::BTreeMap;
use std::rc::Rc;

/// How deeply lists and maps may nest, in the environment or built by a formula.
pub const MAX_DEPTH: u8 = 32;

/// One value.
#[derive(Debug, Clone)]
pub enum Value {
    /// No value.
    Nil,
    /// True or false.
    Bool(bool),
    /// An exact decimal.
    Number(Decimal),
    /// Unicode text.
    Text(Rc<str>),
    /// An instant with the offset it is shown in.
    Date(Date),
    /// A signed count of nanoseconds.
    Duration(i64),
    /// An ordered list.
    List(Rc<List>),
    /// Fields by name, in key order.
    Map(Rc<Map>),
}

/// The items of a list and how deeply it nests.
#[derive(Debug)]
pub struct List {
    items: Vec<Value>,
    depth: u8,
}

/// The entries of a map, sorted by key, and how deeply it nests.
#[derive(Debug)]
pub struct Map {
    entries: BTreeMap<Rc<str>, Value>,
    depth: u8,
}

/// A value nested deeper than [`MAX_DEPTH`].
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct TooDeep;

impl List {
    /// The items in order.
    #[must_use]
    pub fn items(&self) -> &[Value] {
        &self.items
    }
}

impl Map {
    /// The entries in key order.
    #[must_use]
    pub fn entries(&self) -> &BTreeMap<Rc<str>, Value> {
        &self.entries
    }

    /// The value under `key`, if the map has one.
    #[must_use]
    pub fn get(&self, key: &str) -> Option<&Value> {
        self.entries.get(key)
    }
}

/// The kind of a value, as messages and `type()` name it.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Kind {
    /// No value.
    Nil,
    /// True or false.
    Boolean,
    /// Text.
    Text,
    /// A number.
    Number,
    /// A date.
    Date,
    /// A duration.
    Duration,
    /// A list.
    List,
    /// A map.
    Map,
}

impl Kind {
    /// The `select` key messages use for the kind.
    #[must_use]
    pub const fn key(self) -> &'static str {
        match self {
            Self::Nil => "nil",
            Self::Boolean => "boolean",
            Self::Text => "text",
            Self::Number => "number",
            Self::Date => "date",
            Self::Duration => "duration",
            Self::List => "list",
            Self::Map => "map",
        }
    }

    /// What `type(value)` returns.
    #[must_use]
    pub const fn type_name(self) -> &'static str {
        match self {
            Self::Nil => "nil",
            Self::Boolean => "bool",
            Self::Text => "string",
            Self::Number => "number",
            Self::Date => "date",
            Self::Duration => "duration",
            Self::List => "array",
            Self::Map => "map",
        }
    }
}

impl Value {
    /// Text from a string slice.
    #[must_use]
    pub fn text(text: &str) -> Self {
        Self::Text(Rc::from(text))
    }

    /// A number from an integer.
    #[must_use]
    pub fn int(value: i64) -> Self {
        Self::Number(Decimal::from(value))
    }

    /// A list, or [`TooDeep`] when it would nest past [`MAX_DEPTH`].
    pub fn list(items: Vec<Self>) -> Result<Self, TooDeep> {
        let depth = items.iter().map(Self::depth).max().unwrap_or(0) + 1;
        if depth > MAX_DEPTH {
            return Err(TooDeep);
        }
        Ok(Self::List(Rc::new(List { items, depth })))
    }

    /// A map, or [`TooDeep`] when it would nest past [`MAX_DEPTH`].
    pub fn map(entries: BTreeMap<Rc<str>, Self>) -> Result<Self, TooDeep> {
        let depth = entries.values().map(Self::depth).max().unwrap_or(0) + 1;
        if depth > MAX_DEPTH {
            return Err(TooDeep);
        }
        Ok(Self::Map(Rc::new(Map { entries, depth })))
    }

    /// How deeply the value nests: 0 for a scalar.
    #[must_use]
    pub fn depth(&self) -> u8 {
        match self {
            Self::List(list) => list.depth,
            Self::Map(map) => map.depth,
            _ => 0,
        }
    }

    /// The value's kind.
    #[must_use]
    pub const fn kind(&self) -> Kind {
        match self {
            Self::Nil => Kind::Nil,
            Self::Bool(_) => Kind::Boolean,
            Self::Number(_) => Kind::Number,
            Self::Text(_) => Kind::Text,
            Self::Date(_) => Kind::Date,
            Self::Duration(_) => Kind::Duration,
            Self::List(_) => Kind::List,
            Self::Map(_) => Kind::Map,
        }
    }

    /// The `select` key of the value's kind, for messages.
    #[must_use]
    pub const fn kind_key(&self) -> &'static str {
        self.kind().key()
    }

    /// Whether the value is nil.
    #[must_use]
    pub const fn is_nil(&self) -> bool {
        matches!(self, Self::Nil)
    }

    /// Structural equality: numbers by value (`2.50` equals `2.5`), dates by instant, lists item
    /// by item, maps entry by entry. Values of different kinds are never equal; nil equals only
    /// nil.
    #[must_use]
    pub fn same(&self, other: &Self) -> bool {
        match (self, other) {
            (Self::Nil, Self::Nil) => true,
            (Self::Bool(left), Self::Bool(right)) => left == right,
            (Self::Number(left), Self::Number(right)) => left == right,
            (Self::Text(left), Self::Text(right)) => left == right,
            (Self::Date(left), Self::Date(right)) => left.cmp_instant(right) == Ordering::Equal,
            (Self::Duration(left), Self::Duration(right)) => left == right,
            (Self::List(left), Self::List(right)) => {
                left.items.len() == right.items.len()
                    && left
                        .items
                        .iter()
                        .zip(&right.items)
                        .all(|(left, right)| left.same(right))
            }
            (Self::Map(left), Self::Map(right)) => {
                left.entries.len() == right.entries.len()
                    && left.entries.iter().zip(&right.entries).all(
                        |((left_key, left), (right_key, right))| {
                            left_key == right_key && left.same(right)
                        },
                    )
            }
            _ => false,
        }
    }

    /// The order of two values of the same orderable kind (numbers, text by code point, dates by
    /// instant, durations), or `None` when they have none.
    #[must_use]
    pub fn compare(&self, other: &Self) -> Option<Ordering> {
        match (self, other) {
            (Self::Number(left), Self::Number(right)) => Some(left.cmp(right)),
            (Self::Text(left), Self::Text(right)) => Some(left.cmp(right)),
            (Self::Date(left), Self::Date(right)) => Some(left.cmp_instant(right)),
            (Self::Duration(left), Self::Duration(right)) => Some(left.cmp(right)),
            _ => None,
        }
    }

    /// The value as `string(value)` writes it: numbers in plain decimal notation without
    /// trailing zeros, dates in RFC 3339 with their offset, durations as Go writes them
    /// (`1h30m0s`), lists as `[1 2 3]` and maps as `map[a:1 b:2]` in key order, nil as `nil`.
    #[must_use]
    pub fn display(&self) -> String {
        let mut out = String::new();
        self.write_display(&mut out);
        out
    }

    fn write_display(&self, out: &mut String) {
        match self {
            Self::Nil => out.push_str("nil"),
            Self::Bool(value) => out.push_str(if *value { "true" } else { "false" }),
            Self::Number(value) => out.push_str(&format_number(*value)),
            Self::Text(text) => out.push_str(text),
            Self::Date(date) => out.push_str(&date.rfc3339()),
            Self::Duration(nanos) => out.push_str(&format_duration(*nanos)),
            Self::List(list) => {
                out.push('[');
                for (index, item) in list.items.iter().enumerate() {
                    if index > 0 {
                        out.push(' ');
                    }
                    item.write_display(out);
                }
                out.push(']');
            }
            Self::Map(map) => {
                out.push_str("map[");
                for (index, (key, item)) in map.entries.iter().enumerate() {
                    if index > 0 {
                        out.push(' ');
                    }
                    out.push_str(key);
                    out.push(':');
                    item.write_display(out);
                }
                out.push(']');
            }
        }
    }
}

impl PartialEq for Value {
    /// [`Value::same`].
    fn eq(&self, other: &Self) -> bool {
        self.same(other)
    }
}

/// A decimal in plain notation without trailing zeros; zero is always `0`, never `-0`.
#[must_use]
pub fn format_number(value: Decimal) -> String {
    if value.is_zero() {
        return String::from("0");
    }
    value.normalize().to_string()
}

#[cfg(test)]
mod tests {
    use super::{MAX_DEPTH, TooDeep, Value, format_number};
    use rust_decimal::Decimal;
    use std::collections::BTreeMap;
    use std::rc::Rc;

    fn number(text: &str) -> Value {
        Value::Number(text.parse::<Decimal>().expect("decimal"))
    }

    #[test]
    fn numbers_compare_by_value() {
        assert!(number("2.50").same(&number("2.5")));
        assert!(!number("2.5").same(&Value::text("2.5")));
        assert!(Value::Nil.same(&Value::Nil));
        assert!(!Value::Nil.same(&number("0")));
    }

    #[test]
    fn nesting_is_bounded() {
        let mut value = Value::Nil;
        for _ in 0..MAX_DEPTH {
            value = Value::list(vec![value]).expect("within the bound");
        }
        assert_eq!(Value::list(vec![value]).err(), Some(TooDeep));
    }

    #[test]
    fn prints_like_string() {
        let mut entries = BTreeMap::new();
        entries.insert(Rc::from("b"), number("2.50"));
        entries.insert(Rc::from("a"), Value::text("x"));
        let map = Value::map(entries).expect("shallow");
        let list =
            Value::list(vec![number("1"), Value::Bool(true), map, Value::Nil]).expect("shallow");
        assert_eq!(list.display(), "[1 true map[a:x b:2.5] nil]");
        assert_eq!(format_number("-0.000".parse().expect("decimal")), "0");
        assert_eq!(format_number(Decimal::new(100_000, 2)), "1000");
    }
}
