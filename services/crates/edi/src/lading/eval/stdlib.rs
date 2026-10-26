//! The standard library at run time. A function given none where it expects a value gives
//! none, as the checker types it, except those that take none on purpose (`at`, `text`,
//! `isBlank`).

use std::collections::BTreeMap;
use std::rc::Rc;

use regex_lite::RegexBuilder;
use rust_decimal::{Decimal, RoundingStrategy};

use super::interp::{Eval, Frame, Interp};
use super::tree::Tree;
use super::value::{Unit, Value, compare, convert, display, equal, local};
use crate::lading::syntax::ast::Expr;
use crate::lading::types::Dim;
use crate::lading::tz::{ZONES, Zone, days_from_civil, valid_date, weekday};
use crate::lading::values::ZoneValue;

/// One argument: a value, or a lambda's parameter and body.
#[derive(Debug, Clone)]
pub enum Input {
    /// A value.
    Value(Value),
    /// A lambda.
    Lambda(String, Expr),
}

impl Input {
    fn value(&self) -> &Value {
        match self {
            Self::Value(value) => value,
            Self::Lambda(..) => &Value::None,
        }
    }
}

/// Functions that take none on purpose.
fn takes_none(name: &str, index: usize) -> bool {
    matches!((name, index), ("at", 0 | 1) | ("text" | "isBlank", 0))
}

fn text_arg(inputs: &[Input], index: usize) -> Option<String> {
    match inputs.get(index)?.value() {
        Value::None => None,
        Value::Json(json) => Some(match json.as_ref() {
            serde_json::Value::String(text) => text.clone(),
            other => other.to_string(),
        }),
        other => Some(display(other)),
    }
}

fn int_arg(inputs: &[Input], index: usize) -> Option<i64> {
    match inputs.get(index)?.value() {
        Value::Int(value) => Some(*value),
        Value::Decimal(value) => i64::try_from(value.round()).ok(),
        _ => None,
    }
}

fn zone_arg(inputs: &[Input], index: usize) -> Option<ZoneValue> {
    match inputs.get(index)?.value() {
        Value::Zone(zone) => Some(*zone),
        Value::Text(name) => ZoneValue::parse(name),
        _ => None,
    }
}

/// Calls a standard function.
#[allow(clippy::too_many_lines)]
pub fn call(interp: &mut Interp<'_>, frame: &mut Frame, name: &str, inputs: &[Input]) -> Eval {
    for (index, input) in inputs.iter().enumerate() {
        if let Input::Value(Value::None) = input
            && !takes_none(name, index)
        {
            return Ok(Value::None);
        }
    }
    if let Some(Value::List(items)) = inputs.first().map(Input::value) {
        let items = Rc::clone(items);
        if let Some(result) = list(interp, frame, name, &items, inputs)? {
            return Ok(result);
        }
    }
    let first = inputs.first().map_or(&Value::None, Input::value).clone();
    let text = |index| text_arg(inputs, index);
    Ok(match name {
        "trim" => text(0).map_or(Value::None, |value| Value::text(value.trim())),
        "upper" => text(0).map_or(Value::None, |value| Value::text(value.to_uppercase())),
        "lower" => text(0).map_or(Value::None, |value| Value::text(value.to_lowercase())),
        "titleCase" => text(0).map_or(Value::None, |value| Value::text(title_case(&value))),
        "digits" => text(0).map_or(Value::None, |value| {
            Value::text(
                value
                    .chars()
                    .filter(char::is_ascii_digit)
                    .collect::<String>(),
            )
        }),
        "letters" => text(0).map_or(Value::None, |value| {
            Value::text(
                value
                    .chars()
                    .filter(|c| c.is_alphabetic())
                    .collect::<String>(),
            )
        }),
        "clean" => text(0).map_or(Value::None, |value| Value::text(clean(interp, &value))),
        "take" | "truncate" => match (text(0), int_arg(inputs, 1)) {
            (Some(value), Some(count)) => Value::text(
                value
                    .chars()
                    .take(usize::try_from(count).unwrap_or(0))
                    .collect::<String>(),
            ),
            _ => Value::None,
        },
        "drop" => match (text(0), int_arg(inputs, 1)) {
            (Some(value), Some(count)) => Value::text(
                value
                    .chars()
                    .skip(usize::try_from(count).unwrap_or(0))
                    .collect::<String>(),
            ),
            _ => Value::None,
        },
        "pad" => match (text(0), int_arg(inputs, 1)) {
            (Some(value), Some(width)) => {
                let fill_char = text(2).and_then(|fill| fill.chars().next()).unwrap_or(' ');
                let left = matches!(inputs.get(3).map(Input::value), Some(Value::Bool(true)));
                let width = usize::try_from(width).unwrap_or(0).min(1024);
                let count = value.chars().count();
                let fill: String =
                    std::iter::repeat_n(fill_char, width.saturating_sub(count)).collect();
                Value::text(if left {
                    format!("{fill}{value}")
                } else {
                    format!("{value}{fill}")
                })
            }
            _ => Value::None,
        },
        "replace" => match (text(0), text(1), text(2)) {
            (Some(value), Some(old), Some(new)) if !old.is_empty() => {
                Value::text(value.replace(&old, &new))
            }
            (Some(value), _, _) => Value::text(value),
            _ => Value::None,
        },
        "split" => match (text(0), text(1)) {
            (Some(value), Some(separator)) if !separator.is_empty() => {
                Value::list(value.split(separator.as_str()).map(Value::text).collect())
            }
            _ => Value::None,
        },
        "startsWith" => both(text(0), text(1), |a, b| a.starts_with(b)),
        "endsWith" => both(text(0), text(1), |a, b| a.ends_with(b)),
        "contains" => both(text(0), text(1), |a, b| a.contains(b)),
        "matches" => match (text(0), text(1)) {
            (Some(value), Some(pattern)) => RegexBuilder::new(&format!("^(?:{pattern})$"))
                .size_limit(1 << 20)
                .build()
                .map_or(Value::None, |regex| Value::Bool(regex.is_match(&value))),
            _ => Value::None,
        },
        "length" if inputs.len() == 2 => measure(inputs, Dim::Length),
        "length" => text(0).map_or(Value::None, |value| {
            Value::Int(i64::try_from(value.chars().count()).unwrap_or(0))
        }),
        "isBlank" => Value::Bool(text(0).is_none_or(|value| value.trim().is_empty())),
        "zip5" => text(0).map_or(Value::None, |value| {
            let digits: String = value.chars().filter(char::is_ascii_digit).collect();
            if digits.len() >= 5 {
                Value::text(&digits[..5])
            } else {
                Value::None
            }
        }),
        "zip9" => text(0).map_or(Value::None, |value| {
            let digits: String = value.chars().filter(char::is_ascii_digit).collect();
            if digits.len() == 9 {
                Value::text(format!("{}-{}", &digits[..5], &digits[5..]))
            } else {
                Value::None
            }
        }),
        "stateCode" => text(0)
            .and_then(|value| state_code(&value))
            .map_or(Value::None, Value::text),
        "countryCode" => {
            let iso = int_arg(inputs, 1).unwrap_or(2);
            text(0)
                .and_then(|value| country_code(&value, iso == 3))
                .map_or(Value::None, Value::text)
        }
        "phone" => text(0)
            .and_then(|value| phone(&value))
            .map_or(Value::None, Value::text),
        "checkDigit" => match (text(0), text(1)) {
            (Some(value), Some(scheme)) => Value::Bool(check_digit(&value, &scheme)),
            _ => Value::None,
        },
        "isNumber" => {
            Value::Bool(text(0).is_some_and(|value| value.trim().parse::<Decimal>().is_ok()))
        }
        "isDate" => match (text(0), text(1)) {
            (Some(value), Some(pattern)) => Value::Bool(
                parse_pattern(&value, &pattern).is_some_and(|parts| parts.date().is_some()),
            ),
            _ => Value::None,
        },
        "date" | "time" if matches!(first, Value::DateTime(..)) => {
            let Value::DateTime(instant, zone) = first else {
                return Ok(Value::None);
            };
            let (days, seconds, _) = local(instant, zone);
            if name == "date" {
                Value::Date(days)
            } else {
                Value::Time(seconds)
            }
        }
        "date" => match (text(0), text(1)) {
            (Some(value), Some(pattern)) => parse_pattern(&value, &pattern)
                .and_then(|parts| parts.date())
                .map_or(Value::None, Value::Date),
            _ => Value::None,
        },
        "time" => match (text(0), text(1)) {
            (Some(value), Some(pattern)) => parse_pattern(&value, &pattern)
                .and_then(|parts| parts.time())
                .map_or(Value::None, Value::Time),
            _ => Value::None,
        },
        "dateTime" => match (text(0), text(1), zone_arg(inputs, 2)) {
            (Some(value), Some(pattern), Some(zone)) => parse_pattern(&value, &pattern)
                .and_then(|parts| Some((parts.date()?, parts.time().unwrap_or(0))))
                .map_or(Value::None, |(days, seconds)| {
                    Value::DateTime(zone.instant(days * 86_400 + i64::from(seconds)), zone)
                }),
            _ => Value::None,
        },
        "int" => match &first {
            Value::Int(_) => first.clone(),
            Value::Decimal(value) => i64::try_from(
                value.round_dp_with_strategy(0, RoundingStrategy::MidpointAwayFromZero),
            )
            .map_or(Value::None, Value::Int),
            _ => text(0)
                .and_then(|value| value.trim().parse::<i64>().ok())
                .map_or(Value::None, Value::Int),
        },
        "decimal" => match &first {
            Value::Int(value) => Value::Decimal(Decimal::from(*value)),
            Value::Decimal(_) => first.clone(),
            _ => text(0)
                .and_then(|value| value.trim().parse::<Decimal>().ok())
                .map_or(Value::None, Value::Decimal),
        },
        "text" => text(0).map_or(Value::None, Value::text),
        "round" | "floor" | "ceil" | "abs" => {
            let places = u32::try_from(int_arg(inputs, 1).unwrap_or(0).clamp(0, 12)).unwrap_or(0);
            let apply = |value: Decimal| match name {
                "round" => {
                    value.round_dp_with_strategy(places, RoundingStrategy::MidpointAwayFromZero)
                }
                "floor" => value.floor(),
                "ceil" => value.ceil(),
                _ => value.abs(),
            };
            match first {
                Value::Int(value) => match name {
                    "abs" => Value::Int(value.abs()),
                    _ => Value::Int(value),
                },
                Value::Decimal(value) => Value::Decimal(apply(value)),
                Value::Percent(value) => Value::Percent(apply(value)),
                Value::Money(amount, currency) => Value::Money(apply(amount), currency),
                Value::Measure(amount, unit) => Value::Measure(apply(amount), unit),
                Value::Duration(seconds) if name == "abs" => Value::Duration(seconds.abs()),
                other => other,
            }
        }
        "min" | "max" => {
            let second = inputs.get(1).map_or(&Value::None, Input::value);
            match compare(&first, second) {
                Some(order) if (name == "min") == order.is_le() => first.clone(),
                Some(_) => second.clone(),
                None => Value::None,
            }
        }
        "money" => match (
            inputs.first().map(Input::value).and_then(Value::as_decimal),
            text(1),
        ) {
            (Some(amount), Some(currency)) => Value::Money(amount, Rc::from(currency.trim())),
            _ => Value::None,
        },
        "amount" => match first {
            Value::Money(amount, _) => Value::Decimal(amount),
            _ => Value::None,
        },
        "currency" => match first {
            Value::Money(_, currency) => Value::Text(currency),
            _ => Value::None,
        },
        "weight" => measure(inputs, Dim::Weight),
        "volume" => measure(inputs, Dim::Volume),
        "temperature" => measure(inputs, Dim::Temperature),
        "in" | "value" => match (first, text(1).as_deref().and_then(Unit::named)) {
            (Value::Measure(amount, from), Some(to)) if from.dim == to.dim => {
                let converted = convert(amount, from, to);
                if name == "in" {
                    Value::Measure(converted, to)
                } else {
                    Value::Decimal(converted)
                }
            }
            _ => Value::None,
        },
        "at" => {
            let date = match inputs.first().map(Input::value) {
                Some(Value::Date(days)) => *days,
                _ => return Ok(Value::None),
            };
            let seconds = match inputs.get(1).map(Input::value) {
                Some(Value::Time(seconds)) => *seconds,
                _ => 0,
            };
            match zone_arg(inputs, 2) {
                Some(zone) => {
                    Value::DateTime(zone.instant(date * 86_400 + i64::from(seconds)), zone)
                }
                None => Value::None,
            }
        }
        "zone" => match first {
            Value::DateTime(_, zone) => Value::Zone(zone),
            _ => Value::None,
        },
        "inZone" => match (first, zone_arg(inputs, 1)) {
            (Value::DateTime(instant, _), Some(zone)) => Value::DateTime(instant, zone),
            _ => Value::None,
        },
        "zoneCode" => match first {
            Value::DateTime(instant, zone) => {
                zone_code(instant, zone).map_or(Value::None, Value::text)
            }
            _ => Value::None,
        },
        "zoneFromCode" => text(0)
            .and_then(|code| Zone::from_code(code.trim()))
            .map_or(Value::None, |zone| Value::Zone(ZoneValue::Named(zone))),
        "addDays" => match (first, int_arg(inputs, 1)) {
            (Value::Date(days), Some(count)) => Value::Date(days + count),
            (Value::DateTime(instant, zone), Some(count)) => {
                Value::DateTime(instant + count * 86_400, zone)
            }
            _ => Value::None,
        },
        "addBusinessDays" => match (first, int_arg(inputs, 1)) {
            (Value::Date(days), Some(count)) => Value::Date(business_days(days, count)),
            (Value::DateTime(instant, zone), Some(count)) => {
                let (days, seconds, _) = local(instant, zone);
                let target = business_days(days, count);
                Value::DateTime(zone.instant(target * 86_400 + i64::from(seconds)), zone)
            }
            _ => Value::None,
        },
        "now" => Value::DateTime(interp.given.now, ZoneValue::UTC),
        "refs" | "address" | "party" | "elements" => x12_function(interp, name, &first),
        _ => Value::None,
    })
}

fn both(a: Option<String>, b: Option<String>, test: impl Fn(&str, &str) -> bool) -> Value {
    match (a, b) {
        (Some(a), Some(b)) => Value::Bool(test(&a, &b)),
        _ => Value::None,
    }
}

fn measure(inputs: &[Input], dim: Dim) -> Value {
    let amount = inputs.first().map(Input::value).and_then(Value::as_decimal);
    let unit = text_arg(inputs, 1).and_then(|code| Unit::from_code(dim, &code));
    match (amount, unit) {
        (Some(amount), Some(unit)) => Value::Measure(amount, unit),
        _ => Value::None,
    }
}

/// The list functions; `None` when `name` is not one.
#[allow(clippy::too_many_lines)]
fn list(
    interp: &mut Interp<'_>,
    frame: &mut Frame,
    name: &str,
    items: &[Value],
    inputs: &[Input],
) -> Eval<Option<Value>> {
    let lambda = match inputs.get(1) {
        Some(Input::Lambda(param, body)) => Some((param.clone(), body.clone())),
        _ => None,
    };
    let apply = |interp: &mut Interp<'_>, frame: &mut Frame, item: &Value| -> Eval {
        match &lambda {
            Some((param, body)) => interp.apply(frame, param, body, item.clone()),
            None => Ok(item.clone()),
        }
    };
    Ok(Some(match name {
        "count" => {
            if lambda.is_some() {
                let mut count = 0i64;
                for item in items {
                    if apply(interp, frame, item)?.truthy() {
                        count += 1;
                    }
                }
                Value::Int(count)
            } else {
                Value::Int(i64::try_from(items.len()).unwrap_or(0))
            }
        }
        "first" => items.first().cloned().unwrap_or(Value::None),
        "last" => items.last().cloned().unwrap_or(Value::None),
        "isEmpty" => Value::Bool(items.is_empty()),
        "find" => {
            for item in items {
                if apply(interp, frame, item)?.truthy() {
                    return Ok(Some(item.clone()));
                }
            }
            Value::None
        }
        "filter" => {
            let mut kept = Vec::new();
            for item in items {
                if apply(interp, frame, item)?.truthy() {
                    kept.push(item.clone());
                }
            }
            Value::list(kept)
        }
        "map" => {
            let mut mapped = Vec::with_capacity(items.len());
            for item in items {
                mapped.push(apply(interp, frame, item)?);
            }
            Value::list(mapped)
        }
        "any" => {
            for item in items {
                if apply(interp, frame, item)?.truthy() {
                    return Ok(Some(Value::Bool(true)));
                }
            }
            Value::Bool(false)
        }
        "all" => {
            for item in items {
                if !apply(interp, frame, item)?.truthy() {
                    return Ok(Some(Value::Bool(false)));
                }
            }
            Value::Bool(true)
        }
        "sum" => {
            let mut total: Option<Value> = None;
            for item in items {
                let value = apply(interp, frame, item)?;
                if value.is_none() {
                    continue;
                }
                total = Some(match total {
                    None => value,
                    Some(sum) => super::interp::arithmetic(
                        crate::lading::syntax::ast::BinOp::Add,
                        &sum,
                        &value,
                    ),
                });
            }
            total.unwrap_or(Value::Int(0))
        }
        "min" | "max" => {
            let mut best: Option<(Value, Value)> = None;
            for item in items {
                let key = apply(interp, frame, item)?;
                if key.is_none() {
                    continue;
                }
                let better = match &best {
                    None => true,
                    Some((best_key, _)) => compare(&key, best_key).is_some_and(|order| {
                        if name == "min" {
                            order.is_lt()
                        } else {
                            order.is_gt()
                        }
                    }),
                };
                if better {
                    best = Some((key, item.clone()));
                }
            }
            best.map_or(Value::None, |(_, item)| item)
        }
        "sortBy" => {
            let mut keyed = Vec::with_capacity(items.len());
            for item in items {
                keyed.push((apply(interp, frame, item)?, item.clone()));
            }
            keyed.sort_by(|(a, _), (b, _)| compare(a, b).unwrap_or(std::cmp::Ordering::Equal));
            Value::list(keyed.into_iter().map(|(_, item)| item).collect())
        }
        "distinct" => {
            let mut kept: Vec<Value> = Vec::new();
            for item in items {
                if !kept.iter().any(|existing| equal(existing, item)) {
                    kept.push(item.clone());
                }
            }
            Value::list(kept)
        }
        "flatten" => Value::list(
            items
                .iter()
                .flat_map(|item| item.items().to_vec())
                .collect(),
        ),
        "get" => int_arg(inputs, 1)
            .and_then(|index| usize::try_from(index).ok())
            .and_then(|index| items.get(index).cloned())
            .unwrap_or(Value::None),
        "take" => {
            let count = usize::try_from(int_arg(inputs, 1).unwrap_or(0)).unwrap_or(0);
            Value::list(items.iter().take(count).cloned().collect())
        }
        "drop" => {
            let count = usize::try_from(int_arg(inputs, 1).unwrap_or(0)).unwrap_or(0);
            Value::list(items.iter().skip(count).cloned().collect())
        }
        "contains" => {
            let wanted = inputs.get(1).map_or(&Value::None, Input::value);
            Value::Bool(items.iter().any(|item| equal(item, wanted)))
        }
        "join" => {
            let separator = text_arg(inputs, 1).unwrap_or_default();
            Value::text(
                items
                    .iter()
                    .filter(|item| !item.is_none())
                    .map(display)
                    .collect::<Vec<_>>()
                    .join(&separator),
            )
        }
        "refs" => {
            let mut refs = BTreeMap::new();
            for item in items {
                if let Value::Segment(tree, index) = item {
                    let tree = Rc::clone(tree);
                    let (qualifier, value) = if tree.segments[*index].id == "N9" {
                        (1, 2)
                    } else {
                        (2, 1)
                    };
                    let key = interp.read_element(&tree, *index, qualifier);
                    let value = interp.read_element(&tree, *index, value);
                    if !key.is_none() && !value.is_none() {
                        refs.entry(display(&key)).or_insert(value);
                    }
                }
            }
            Value::Map(Rc::new(refs))
        }
        _ => return Ok(None),
    }))
}

fn x12_function(interp: &mut Interp<'_>, name: &str, value: &Value) -> Value {
    match (name, value) {
        ("elements", Value::Segment(tree, index)) => {
            let tree = Rc::clone(tree);
            let count = tree.segments[*index].elements.len();
            Value::list(
                (1..=count)
                    .map(|position| {
                        let element = interp.read_element(
                            &tree,
                            *index,
                            u32::try_from(position).unwrap_or(0),
                        );
                        if element.is_none() {
                            Value::text("")
                        } else {
                            Value::text(display(&element))
                        }
                    })
                    .collect(),
            )
        }
        ("address", Value::Loop(tree, instance)) => address(interp, tree, *instance),
        ("party", Value::Loop(tree, instance)) => {
            let tree = Rc::clone(tree);
            let name_value = tree
                .segments_in(*instance, "N1")
                .first()
                .map_or(Value::None, |index| interp.read_element(&tree, *index, 2));
            let mut fields = BTreeMap::new();
            fields.insert("name".to_owned(), name_value);
            fields.insert("address".to_owned(), address(interp, &tree, *instance));
            interp.as_record("Party", fields)
        }
        _ => Value::None,
    }
}

fn address(interp: &mut Interp<'_>, tree: &Rc<Tree>, instance: usize) -> Value {
    let tree = Rc::clone(tree);
    let mut fields = BTreeMap::new();
    if let Some(index) = tree.segments_in(instance, "N3").first() {
        fields.insert("line1".to_owned(), interp.read_element(&tree, *index, 1));
        fields.insert("line2".to_owned(), interp.read_element(&tree, *index, 2));
    }
    if let Some(index) = tree.segments_in(instance, "N4").first() {
        fields.insert("city".to_owned(), interp.read_element(&tree, *index, 1));
        fields.insert("state".to_owned(), interp.read_element(&tree, *index, 2));
        fields.insert("postal".to_owned(), interp.read_element(&tree, *index, 3));
        fields.insert("country".to_owned(), interp.read_element(&tree, *index, 4));
    }
    interp.as_record("Address", fields)
}

fn title_case(text: &str) -> String {
    let mut out = String::with_capacity(text.len());
    let mut start = true;
    for c in text.chars() {
        if c.is_alphanumeric() {
            if start {
                out.extend(c.to_uppercase());
            } else {
                out.extend(c.to_lowercase());
            }
            start = false;
        } else {
            out.push(c);
            start = c != '\'';
        }
    }
    out
}

fn clean(interp: &Interp<'_>, text: &str) -> String {
    let separators: Vec<char> = interp.program.partner_separators().into_iter().collect();
    text.chars()
        .map(|c| {
            if c.is_control() || separators.contains(&c) {
                ' '
            } else {
                c
            }
        })
        .filter(|c| c.is_ascii() || c.is_alphanumeric() || *c == ' ')
        .collect::<String>()
        .split_whitespace()
        .collect::<Vec<_>>()
        .join(" ")
}

const STATES: &[(&str, &str)] = &[
    ("AL", "Alabama"),
    ("AK", "Alaska"),
    ("AZ", "Arizona"),
    ("AR", "Arkansas"),
    ("CA", "California"),
    ("CO", "Colorado"),
    ("CT", "Connecticut"),
    ("DE", "Delaware"),
    ("DC", "District of Columbia"),
    ("FL", "Florida"),
    ("GA", "Georgia"),
    ("HI", "Hawaii"),
    ("ID", "Idaho"),
    ("IL", "Illinois"),
    ("IN", "Indiana"),
    ("IA", "Iowa"),
    ("KS", "Kansas"),
    ("KY", "Kentucky"),
    ("LA", "Louisiana"),
    ("ME", "Maine"),
    ("MD", "Maryland"),
    ("MA", "Massachusetts"),
    ("MI", "Michigan"),
    ("MN", "Minnesota"),
    ("MS", "Mississippi"),
    ("MO", "Missouri"),
    ("MT", "Montana"),
    ("NE", "Nebraska"),
    ("NV", "Nevada"),
    ("NH", "New Hampshire"),
    ("NJ", "New Jersey"),
    ("NM", "New Mexico"),
    ("NY", "New York"),
    ("NC", "North Carolina"),
    ("ND", "North Dakota"),
    ("OH", "Ohio"),
    ("OK", "Oklahoma"),
    ("OR", "Oregon"),
    ("PA", "Pennsylvania"),
    ("PR", "Puerto Rico"),
    ("RI", "Rhode Island"),
    ("SC", "South Carolina"),
    ("SD", "South Dakota"),
    ("TN", "Tennessee"),
    ("TX", "Texas"),
    ("UT", "Utah"),
    ("VT", "Vermont"),
    ("VA", "Virginia"),
    ("WA", "Washington"),
    ("WV", "West Virginia"),
    ("WI", "Wisconsin"),
    ("WY", "Wyoming"),
    ("AB", "Alberta"),
    ("BC", "British Columbia"),
    ("MB", "Manitoba"),
    ("NB", "New Brunswick"),
    ("NL", "Newfoundland and Labrador"),
    ("NS", "Nova Scotia"),
    ("NT", "Northwest Territories"),
    ("NU", "Nunavut"),
    ("ON", "Ontario"),
    ("PE", "Prince Edward Island"),
    ("QC", "Quebec"),
    ("SK", "Saskatchewan"),
    ("YT", "Yukon"),
];

fn state_code(text: &str) -> Option<String> {
    let wanted = text.trim();
    STATES
        .iter()
        .find(|(code, name)| code.eq_ignore_ascii_case(wanted) || name.eq_ignore_ascii_case(wanted))
        .map(|(code, _)| (*code).to_owned())
}

const COUNTRIES: &[(&str, &str, &str)] = &[
    ("US", "USA", "United States"),
    ("CA", "CAN", "Canada"),
    ("MX", "MEX", "Mexico"),
    ("GT", "GTM", "Guatemala"),
    ("PR", "PRI", "Puerto Rico"),
];

fn country_code(text: &str, three: bool) -> Option<String> {
    let wanted = text.trim();
    let alias = matches!(
        wanted.to_uppercase().as_str(),
        "U.S." | "U.S.A." | "UNITED STATES OF AMERICA"
    );
    COUNTRIES
        .iter()
        .find(|(two, long, name)| {
            two.eq_ignore_ascii_case(wanted)
                || long.eq_ignore_ascii_case(wanted)
                || name.eq_ignore_ascii_case(wanted)
                || (alias && *two == "US")
        })
        .map(|(two, long, _)| {
            if three {
                (*long).to_owned()
            } else {
                (*two).to_owned()
            }
        })
}

fn phone(text: &str) -> Option<String> {
    let digits: String = text.chars().filter(char::is_ascii_digit).collect();
    match digits.len() {
        10 => Some(format!("+1{digits}")),
        11 if digits.starts_with('1') => Some(format!("+{digits}")),
        8..=15 if text.trim_start().starts_with('+') => Some(format!("+{digits}")),
        _ => None,
    }
}

fn check_digit(text: &str, scheme: &str) -> bool {
    let digits: Vec<u32> = text.chars().filter_map(|c| c.to_digit(10)).collect();
    if digits.len() < 2 || digits.len() != text.trim().chars().count() {
        return false;
    }
    match scheme {
        "mod10" => {
            let sum: u32 = digits
                .iter()
                .rev()
                .enumerate()
                .map(|(index, digit)| {
                    if index % 2 == 1 {
                        let doubled = digit * 2;
                        if doubled > 9 { doubled - 9 } else { doubled }
                    } else {
                        *digit
                    }
                })
                .sum();
            sum.is_multiple_of(10)
        }
        "mod7" => {
            let (last, body) = digits.split_last().unwrap_or((&0, &[]));
            let number = body
                .iter()
                .fold(0u64, |acc, digit| (acc * 10 + u64::from(*digit)) % 7);
            number == u64::from(*last)
        }
        _ => false,
    }
}

/// What a date or time pattern read.
#[derive(Debug, Default)]
pub(crate) struct Parts {
    year: Option<i64>,
    month: Option<u32>,
    day: Option<u32>,
    hour: Option<u32>,
    minute: Option<u32>,
    second: Option<u32>,
}

impl Parts {
    pub(crate) fn date(&self) -> Option<i64> {
        let (year, month, day) = (self.year?, self.month?, self.day?);
        valid_date(year, month, day).then(|| days_from_civil(year, month, day))
    }

    pub(crate) fn time(&self) -> Option<u32> {
        let (hour, minute) = (self.hour?, self.minute.unwrap_or(0));
        let second = self.second.unwrap_or(0);
        (hour < 24 && minute < 60 && second < 60).then_some(hour * 3600 + minute * 60 + second)
    }
}

/// Reads `text` with a pattern of `yyyy`, `yy`, `MM`, `dd`, `HH`, `mm` and `ss`.
pub(crate) fn parse_pattern(text: &str, pattern: &str) -> Option<Parts> {
    let text = text.trim();
    let mut parts = Parts::default();
    let mut rest = text;
    let mut pattern_rest = pattern;
    let tokens = [
        ("yyyy", 4),
        ("yy", 2),
        ("MM", 2),
        ("dd", 2),
        ("HH", 2),
        ("mm", 2),
        ("ss", 2),
    ];
    while !pattern_rest.is_empty() {
        if let Some((token, width)) = tokens
            .iter()
            .find(|(token, _)| pattern_rest.starts_with(token))
        {
            let digits = rest.get(..*width)?;
            if !digits.bytes().all(|byte| byte.is_ascii_digit()) {
                return None;
            }
            let number: u32 = digits.parse().ok()?;
            match *token {
                "yyyy" => parts.year = Some(i64::from(number)),
                "yy" => parts.year = Some(2000 + i64::from(number)),
                "MM" => parts.month = Some(number),
                "dd" => parts.day = Some(number),
                "HH" => parts.hour = Some(number),
                "mm" => parts.minute = Some(number),
                _ => parts.second = Some(number),
            }
            rest = &rest[*width..];
            pattern_rest = &pattern_rest[token.len()..];
        } else {
            let mut chars = pattern_rest.chars();
            let literal = chars.next()?;
            if !rest.starts_with(literal) {
                return None;
            }
            rest = &rest[literal.len_utf8()..];
            pattern_rest = chars.as_str();
        }
    }
    rest.is_empty().then_some(parts)
}

/// The X12 time code of the zone an instant is in.
fn zone_code(instant: i64, zone: ZoneValue) -> Option<String> {
    match zone {
        ZoneValue::Named(zone) => Some(zone.code(instant, false).to_owned()),
        ZoneValue::Fixed(offset) => ZONES
            .iter()
            .find(|candidate| candidate.standard == offset)
            .or_else(|| {
                ZONES
                    .iter()
                    .find(|candidate| candidate.daylight && candidate.standard + 60 == offset)
            })
            .map(|candidate| candidate.codes.0.to_owned()),
    }
}

fn business_days(start: i64, count: i64) -> i64 {
    let mut day = start;
    let step = if count < 0 { -1 } else { 1 };
    let mut left = count.abs().min(100_000);
    while left > 0 {
        day += step;
        if !matches!(weekday(day), 0 | 6) {
            left -= 1;
        }
    }
    day
}

#[cfg(test)]
mod tests {
    use super::{check_digit, country_code, parse_pattern, phone, state_code, title_case};

    #[test]
    fn text_helpers_behave() {
        assert_eq!(title_case("DALLAS FORT-WORTH"), "Dallas Fort-Worth");
        assert_eq!(title_case("o'hare"), "O'hare");
        assert_eq!(state_code("texas").as_deref(), Some("TX"));
        assert_eq!(state_code("qc").as_deref(), Some("QC"));
        assert_eq!(country_code("Canada", true).as_deref(), Some("CAN"));
        assert_eq!(phone("(555) 123-4567").as_deref(), Some("+15551234567"));
        assert!(check_digit("79927398713", "mod10"));
        assert!(!check_digit("79927398710", "mod10"));
    }

    #[test]
    fn patterns_read_dates_and_times() {
        let parts = parse_pattern("01/10/2026 14:30", "MM/dd/yyyy HH:mm").expect("parts");
        assert_eq!(parts.date(), Some(20_463));
        assert_eq!(parts.time(), Some(52_200));
        assert!(parse_pattern("1/10/2026", "MM/dd/yyyy").is_none());
    }
}
