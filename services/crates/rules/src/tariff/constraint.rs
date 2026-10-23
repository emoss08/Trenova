//! An input's `where` constraint, folded to wire values, and the one test of a value against it.
//!
//! The checker folds the constraint once ([`crate::tariff::check::InputInfo::constraint`]); the
//! contract-value validation, the evaluator's input resolution, the test runner's `given`s and a
//! property's `vary input.x` all read the same folded form, and all of them decide with
//! [`admits`] (or [`breach`], which also says which value is outside), so they agree on every
//! bound:
//!
//! - a range admits a value from its minimum (always inclusive) up to its maximum (inclusive
//!   only when written `..=`); an open end admits everything on its side;
//! - a list admits a value equal to one of its items;
//! - `none` (an optional input that is not set) is always admitted;
//! - a list value (a `List<T>` input) is admitted when every item is, so a breach names the
//!   item by its index.
//!
//! Values compare exactly and across units with the evaluator's own comparison
//! ([`Value::range_compare`], which orders text too, and [`Value::equals`]): `1 cwt` is inside `0 lb..200 lb`, `$426` equals
//! `$426.00`. A bound is read in the kind of the value it is compared with, through the wire
//! codec, so it keeps the unit it was written in.
//!
//! [`message`] words a breach the same way everywhere: "`rate` must be from $0.50/mi to
//! $15.00/mi; $235.00/mi is outside it", with values written as interpolation writes them.

use crate::message::{Arg, Message};
use crate::tariff::eval::value::{Value, is_percent};
use crate::tariff::types::Type;
use crate::tariff::wire::decode;
use serde_json::Value as Json;
use serde_json::json;
use std::cmp::Ordering;
use std::fmt::Write as _;

/// What values an input accepts, beyond its type.
#[derive(Debug, Clone, PartialEq)]
pub enum Constraint {
    /// `lo..hi`, `lo..=hi`, `lo..`, `..=hi`: the minimum is inclusive, the maximum inclusive
    /// only when written `..=`. An open end is `None`.
    Range {
        /// The smallest value accepted, as a wire value.
        min: Option<Json>,
        /// The bound above, as a wire value.
        max: Option<Json>,
        /// The maximum is accepted too.
        max_inclusive: bool,
    },
    /// `[a, b]`: the value must equal one of these wire values.
    OneOf(Vec<Json>),
}

impl Constraint {
    /// The constraint as `tariff.check` reports it.
    #[must_use]
    pub fn to_json(&self) -> Json {
        match self {
            Self::Range {
                min,
                max,
                max_inclusive,
            } => json!({"range": {"min": min, "max": max, "maxInclusive": max_inclusive}}),
            Self::OneOf(values) => json!({ "oneOf": values }),
        }
    }
}

/// A value outside a constraint.
#[derive(Debug, Clone)]
pub struct Breach {
    /// Where it is inside a list value, outermost index first; empty for the value itself.
    pub at: Vec<usize>,
    /// The value outside the constraint.
    pub value: Value,
}

impl Breach {
    /// The index suffix of the breach: `""`, `"[2]"`, `"[2][0]"`.
    #[must_use]
    pub fn suffix(&self) -> String {
        let mut out = String::new();
        for index in &self.at {
            let _ = write!(out, "[{index}]");
        }
        out
    }

    /// The finding's text, naming `name` (and the item's index, for a list value).
    #[must_use]
    pub fn message(&self, constraint: &Constraint, name: &str) -> Message {
        message(constraint, &format!("{name}{}", self.suffix()), &self.value)
    }
}

/// Whether a constraint admits a value: `none` always; a list when every item is admitted; any
/// other value when it lies in the range or equals one of the listed values.
#[must_use]
pub fn admits(constraint: &Constraint, value: &Value) -> bool {
    breach(constraint, value).is_none()
}

/// The first value outside a constraint, or `None` when the constraint admits the value.
#[must_use]
pub fn breach(constraint: &Constraint, value: &Value) -> Option<Breach> {
    let mut free = || Ok::<(), std::convert::Infallible>(());
    match breach_metered(constraint, value, &mut free) {
        Ok(found) => found,
        Err(never) => match never {},
    }
}

/// [`breach`], calling `tick` once for each value it compares, so a caller running under a
/// budget meters a long list input.
///
/// # Errors
///
/// The first error `tick` returns.
pub fn breach_metered<E>(
    constraint: &Constraint,
    value: &Value,
    tick: &mut impl FnMut() -> Result<(), E>,
) -> Result<Option<Breach>, E> {
    let mut walk = Walk {
        constraint,
        bounds: Read::Pending,
        at: Vec::new(),
    };
    walk.visit(value, tick)
}

/// A walk over a value and its items, reading the bounds once, in the kind of the first value.
struct Walk<'c> {
    constraint: &'c Constraint,
    bounds: Read,
    at: Vec<usize>,
}

/// The bounds a walk compares with, read when the first value is met.
enum Read {
    /// No value met yet.
    Pending,
    /// Read in the kind of the first value.
    Ready(Bounds),
    /// A bound does not read as that kind, so the constraint admits everything.
    Unreadable,
}

impl Walk<'_> {
    fn visit<E>(
        &mut self,
        value: &Value,
        tick: &mut impl FnMut() -> Result<(), E>,
    ) -> Result<Option<Breach>, E> {
        match value {
            Value::None => Ok(None),
            Value::List(items) => {
                for (index, item) in items.iter().enumerate() {
                    self.at.push(index);
                    let found = self.visit(item, tick)?;
                    self.at.pop();
                    if found.is_some() {
                        return Ok(found);
                    }
                }
                Ok(None)
            }
            scalar => {
                tick()?;
                if matches!(self.bounds, Read::Pending) {
                    self.bounds =
                        Bounds::read(self.constraint, scalar).map_or(Read::Unreadable, Read::Ready);
                }
                let inside = match &self.bounds {
                    Read::Ready(bounds) => bounds.admit(scalar),
                    Read::Pending | Read::Unreadable => true,
                };
                Ok((!inside).then(|| Breach {
                    at: self.at.clone(),
                    value: scalar.clone(),
                }))
            }
        }
    }
}

/// A constraint's bounds as run-time values.
enum Bounds {
    Range {
        min: Option<Value>,
        max: Option<Value>,
        max_inclusive: bool,
    },
    OneOf(Vec<Value>),
}

impl Bounds {
    /// The bounds read in the kind of `like`, or `None` when one does not read as that kind
    /// (which the checker rules out: every bound is checked against the input's type).
    fn read(constraint: &Constraint, like: &Value) -> Option<Self> {
        Some(match constraint {
            Constraint::Range {
                min,
                max,
                max_inclusive,
            } => Self::Range {
                min: match min {
                    Some(json) => Some(bound(json, like)?),
                    None => None,
                },
                max: match max {
                    Some(json) => Some(bound(json, like)?),
                    None => None,
                },
                max_inclusive: *max_inclusive,
            },
            Constraint::OneOf(values) => Self::OneOf(
                values
                    .iter()
                    .map(|json| bound(json, like))
                    .collect::<Option<Vec<_>>>()?,
            ),
        })
    }

    fn admit(&self, value: &Value) -> bool {
        match self {
            Self::Range {
                min,
                max,
                max_inclusive,
            } => {
                let above_min = min.as_ref().is_none_or(|min| {
                    matches!(
                        value.range_compare(min),
                        Ok(Ordering::Greater | Ordering::Equal)
                    )
                });
                let below_max = max
                    .as_ref()
                    .is_none_or(|max| match value.range_compare(max) {
                        Ok(Ordering::Less) => true,
                        Ok(Ordering::Equal) => *max_inclusive,
                        _ => false,
                    });
                above_min && below_max
            }
            Self::OneOf(values) => values
                .iter()
                .any(|allowed| value.equals(allowed).unwrap_or(false)),
        }
    }
}

/// A bound's wire value read as a value of the same kind as `like`.
fn bound(json: &Json, like: &Value) -> Option<Value> {
    if let Value::Enum(_) = like {
        return json.as_str().map(Value::variant);
    }
    decode(json, &type_like(like)?, "").ok()
}

/// The type a bound compared with `value` is read as: numbers as `Number` (an `Int` compares
/// with it), a percent as `Percent`, a quantity in its own dimension.
fn type_like(value: &Value) -> Option<Type> {
    Some(match value {
        Value::Bool(_) => Type::Bool,
        Value::Int(_) | Value::Number(_) => Type::Number,
        Value::Quantity(quantity) if is_percent(quantity) => Type::Percent,
        Value::Quantity(quantity) => Type::quantity_in(&quantity.unit),
        Value::Text(_) => Type::Text,
        Value::Date(_) => Type::Date,
        Value::DateTime(_) => Type::DateTime,
        Value::Time(_) => Type::Time,
        Value::None | Value::List(_) | Value::Record(_) | Value::Enum(_) | Value::Table(_) => {
            return None;
        }
    })
}

/// Why `value` is outside `constraint`, naming `subject` (an input, or an item of one, as
/// `breaks[2]`), with every value written as interpolation writes it:
///
/// - "{0} must be from {1} to {2}; {3} is outside it" (`lo..=hi`);
/// - "{0} must be at least {1} and below {2}; {3} is outside it" (`lo..hi`);
/// - "{0} must be at least {1}; {2} is below it" (`lo..`);
/// - "{0} must be below {1}; {2} isn't" (`..hi`) and "{0} must be at most {1}; {2} is above
///   it" (`..=hi`);
/// - "{0} must be one of {1}; {2} isn't" (a list).
#[must_use]
pub fn message(constraint: &Constraint, subject: &str, value: &Value) -> Message {
    let subject = Arg::from(format!("`{subject}`"));
    let shown = Arg::from(value.display());
    let bounds = Bounds::read(constraint, value);
    let text = |json: &Option<Json>, read: Option<&Value>| -> Arg {
        Arg::from(read.map_or_else(
            || json.as_ref().map(ToString::to_string).unwrap_or_default(),
            Value::display,
        ))
    };
    match (constraint, &bounds) {
        (Constraint::OneOf(values), bounds) => {
            let listed = match bounds {
                Some(Bounds::OneOf(read)) => Value::list(read.clone()).display(),
                _ => Json::Array(values.clone()).to_string(),
            };
            Message::with(
                crate::message!("{0} must be one of {1}; {2} isn't"),
                [subject, Arg::from(listed), shown],
            )
        }
        (
            Constraint::Range {
                min,
                max,
                max_inclusive,
            },
            bounds,
        ) => {
            let (low, high) = match bounds {
                Some(Bounds::Range { min, max, .. }) => (min.as_ref(), max.as_ref()),
                _ => (None, None),
            };
            let low = min.as_ref().map(|_| text(min, low));
            let high = max.as_ref().map(|_| text(max, high));
            range_message(subject, low, high, *max_inclusive, shown)
        }
    }
}

fn range_message(
    subject: Arg,
    low: Option<Arg>,
    high: Option<Arg>,
    inclusive: bool,
    shown: Arg,
) -> Message {
    match (low, high) {
        (Some(low), Some(high)) if inclusive => Message::with(
            crate::message!("{0} must be from {1} to {2}; {3} is outside it"),
            [subject, low, high, shown],
        ),
        (Some(low), Some(high)) => Message::with(
            crate::message!("{0} must be at least {1} and below {2}; {3} is outside it"),
            [subject, low, high, shown],
        ),
        (Some(low), None) => Message::with(
            crate::message!("{0} must be at least {1}; {2} is below it"),
            [subject, low, shown],
        ),
        (None, Some(high)) if inclusive => Message::with(
            crate::message!("{0} must be at most {1}; {2} is above it"),
            [subject, high, shown],
        ),
        (None, Some(high)) => Message::with(
            crate::message!("{0} must be below {1}; {2} isn't"),
            [subject, high, shown],
        ),
        (None, None) => Message::with(
            crate::message!("{0} is outside its constraint: {1}"),
            [subject, shown],
        ),
    }
}

#[cfg(test)]
mod tests {
    use super::{Constraint, admits, breach};
    use crate::tariff::eval::value::Value;
    use crate::tariff::units::Unit;
    use rust_decimal::Decimal;
    use serde_json::json;
    use std::str::FromStr;

    fn q(value: &str, unit: &str) -> Value {
        Value::quantity(
            Decimal::from_str(value).expect("a decimal"),
            Unit::parse(unit).expect("a unit"),
        )
    }

    fn range(
        min: Option<serde_json::Value>,
        max: Option<serde_json::Value>,
        inclusive: bool,
    ) -> Constraint {
        Constraint::Range {
            min,
            max,
            max_inclusive: inclusive,
        }
    }

    #[test]
    fn ranges_compare_across_units_and_scales() {
        let weights = range(
            Some(json!({"$q": "0", "u": "lb"})),
            Some(json!({"$q": "200", "u": "lb"})),
            false,
        );
        assert!(admits(&weights, &q("1", "cwt")));
        assert!(admits(&weights, &q("0", "kg")));
        assert!(!admits(&weights, &q("2", "cwt")));
        let inclusive = range(None, Some(json!({"$q": "15", "u": "$/mi"})), true);
        assert!(admits(&inclusive, &q("15.00", "$/mi")));
        assert!(!admits(&inclusive, &q("15.01", "$/mi")));
    }

    #[test]
    fn none_passes_and_lists_name_the_item_outside() {
        let counts = range(Some(json!({"$n": "0"})), Some(json!({"$n": "10"})), true);
        assert!(admits(&counts, &Value::None));
        let items = Value::list(vec![Value::int(1), Value::int(10), Value::int(11)]);
        let found = breach(&counts, &items).expect("a breach");
        assert_eq!(found.at, vec![2]);
        assert_eq!(found.suffix(), "[2]");
        assert_eq!(
            found.message(&counts, "drops").english(),
            "`drops[2]` must be from 0 to 10; 11 is outside it"
        );
    }

    #[test]
    fn a_list_admits_its_items_only() {
        let services = Constraint::OneOf(vec![json!("Standard"), json!("Guaranteed")]);
        assert!(admits(&services, &Value::variant("Guaranteed")));
        let found = breach(&services, &Value::variant("Expedited")).expect("a breach");
        assert_eq!(
            found.message(&services, "service").english(),
            "`service` must be one of [Standard, Guaranteed]; Expedited isn't"
        );
    }
}
