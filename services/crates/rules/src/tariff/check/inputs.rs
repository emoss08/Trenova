//! A file's inputs as the hosts see them: name, doc, type (display text and a structured
//! descriptor for building forms), whether a value is required, and the default as a wire
//! value; and the validation of the input values a rate agreement rule and its agreement set.
//!
//! Descriptors: `{"kind":"money"}`, `{"kind":"quantity","dimension":"Money/Distance","unit":
//! "$/mi"}`, `{"kind":"list","item":…}`, `{"kind":"optional","inner":…}`, `{"kind":"record",
//! "name":"Break","fields":[{"name","doc","type"}]}`, `{"kind":"enum","name","variants":[…]}`,
//! and `int`, `number`, `percent`, `text`, `bool`, `date`, `dateTime`.
//!
//! Values are checked against the reference's "Wire values" exactly: `{"$n":"1200"}` for `Int`
//! and `Number`, `{"$q":"2.35","u":"$/mi"}` for money, rates, quantities and percents (`"u":
//! "%"`), `{"$date":"…"}` for dates and date-times, a variant's name as a string, `null` for
//! `none`, arrays and objects for lists and records.

use super::consts::wire;
use super::decls::State;
use super::describe::{quote, unit_text, words};
use super::{InputInfo, ModuleAnalysis};
use crate::message;
use crate::message::{Arg, Message};
use crate::tariff::diagnostic::nearest;
use crate::tariff::syntax::ast::ItemKind;
use crate::tariff::syntax::program::{Decl, Module};
use crate::tariff::types::Type;
use crate::tariff::units::{Dim, Unit};
use rust_decimal::Decimal;
use serde_json::{Map, Value as Json, json};
use std::str::FromStr;

/// The root's inputs, in declaration order.
#[must_use]
pub fn describe(root: &Module, state: &State, _: &[ModuleAnalysis]) -> Vec<InputInfo> {
    let mut out: Vec<InputInfo> = Vec::new();
    for (name, symbol) in &root.symbols {
        let Decl::Input(item) = symbol.decl else {
            continue;
        };
        let Some(item) = root.file.items.get(item) else {
            continue;
        };
        let ItemKind::Input(input) = &item.kind else {
            continue;
        };
        let ty = state
            .symbols
            .iter()
            .find(|known| known.module == Some(root.id) && known.span == Some(symbol.span))
            .and_then(|known| known.ty.clone())
            .unwrap_or(Type::Error);
        out.push(InputInfo {
            name: name.clone(),
            doc: item.docs.as_ref().map(|doc| doc.text.clone()),
            descriptor: descriptor(&ty),
            required: input.default.is_none(),
            default: state.defaults.get(&(root.id.0, name.clone())).map(wire),
            ty,
            span: symbol.span,
        });
    }
    out.sort_by_key(|input| input.span.start);
    out
}

/// A type as a form descriptor.
#[must_use]
pub fn descriptor(ty: &Type) -> Json {
    match ty {
        Type::Bool => json!({ "kind": "bool" }),
        Type::Int => json!({ "kind": "int" }),
        Type::Number => json!({ "kind": "number" }),
        Type::Percent => json!({ "kind": "percent" }),
        Type::Text => json!({ "kind": "text" }),
        Type::Date => json!({ "kind": "date" }),
        Type::DateTime => json!({ "kind": "dateTime" }),
        Type::Quantity(quantity) if quantity.dim == Dim::money() => json!({ "kind": "money" }),
        Type::Quantity(quantity) => json!({
            "kind": "quantity",
            "dimension": quantity.dim.to_string(),
            "unit": unit_text(quantity),
        }),
        Type::List(item) => json!({ "kind": "list", "item": descriptor(item) }),
        Type::Optional(inner) => json!({ "kind": "optional", "inner": descriptor(inner) }),
        Type::Record(record) => json!({
            "kind": "record",
            "name": record.name.as_ref().map(|_| ty.to_string()),
            "fields": record.fields.iter().map(|field| json!({
                "name": field.name,
                "doc": field.doc,
                "type": descriptor(&field.ty),
            })).collect::<Vec<_>>(),
        }),
        Type::Enum(enumeration) => json!({
            "kind": "enum",
            "name": ty.to_string(),
            "variants": enumeration.variants.iter().map(|(name, _)| name.clone()).collect::<Vec<_>>(),
        }),
        Type::Never | Type::Error => json!({ "kind": "unknown" }),
    }
}

/// A problem with a contract's input value, at a path such as `rule.rate` or
/// `agreement.breaks[2].rate`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ValueFinding {
    /// Where the value is.
    pub path: String,
    /// What is wrong.
    pub message: Message,
}

/// Checks the values a rule and an agreement set against the inputs: unknown names, values
/// that do not fit, and required inputs neither sets.
#[must_use]
pub fn validate(
    inputs: &[InputInfo],
    rule: Option<&Map<String, Json>>,
    agreement: Option<&Map<String, Json>>,
) -> Vec<ValueFinding> {
    let mut out = Vec::new();
    for (scope, values) in [("rule", rule), ("agreement", agreement)] {
        let Some(values) = values else { continue };
        for (name, value) in values {
            let path = format!("{scope}.{name}");
            if let Some(input) = inputs.iter().find(|input| input.name == *name) {
                check_value(&input.ty, value, &path, &mut out);
            } else {
                let suggestion = nearest(name, inputs.iter().map(|input| input.name.as_str()));
                out.push(ValueFinding {
                    path,
                    message: match suggestion {
                        Some(close) => Message::with(
                            message!("{0} isn't an input of this tariff. Did you mean {1}?"),
                            [quote(name), quote(close)],
                        ),
                        None => Message::with(
                            message!("{0} isn't an input of this tariff"),
                            [quote(name)],
                        ),
                    },
                });
            }
        }
    }
    for input in inputs.iter().filter(|input| input.required) {
        let set = |values: Option<&Map<String, Json>>| {
            values.is_some_and(|values| values.contains_key(&input.name))
        };
        if !set(rule) && !set(agreement) {
            out.push(ValueFinding {
                path: format!("rule.{}", input.name),
                message: Message::with(
                    message!("{0} has no default, so the rule or the agreement must set it"),
                    [quote(&input.name)],
                ),
            });
        }
    }
    out
}

fn finding(out: &mut Vec<ValueFinding>, path: &str, message: Message) {
    out.push(ValueFinding {
        path: path.to_owned(),
        message,
    });
}

fn wrong(out: &mut Vec<ValueFinding>, path: &str, ty: &Type) {
    finding(
        out,
        path,
        Message::with(message!("This value must be {0}"), [Arg::from(words(ty))]),
    );
}

/// The only key of a tagged object, when it has exactly the keys given.
fn tagged<'a>(value: &'a Json, keys: &[&str]) -> Option<&'a Map<String, Json>> {
    let object = value.as_object()?;
    (object.len() == keys.len() && keys.iter().all(|key| object.contains_key(*key)))
        .then_some(object)
}

fn decimal(value: Option<&Json>) -> Option<Decimal> {
    let text = value?.as_str()?;
    let valid = !text.is_empty() && text.trim() == text && !text.contains(['e', 'E', '_', '+']);
    if !valid {
        return None;
    }
    Decimal::from_str(text).ok()
}

/// Checks one wire value against a type.
pub fn check_value(ty: &Type, value: &Json, path: &str, out: &mut Vec<ValueFinding>) {
    match ty {
        Type::Error | Type::Never => {}
        Type::Optional(inner) => {
            if !value.is_null() {
                check_value(inner, value, path, out);
            }
        }
        _ if value.is_null() => finding(
            out,
            path,
            Message::new(message!("This value can't be empty")),
        ),
        Type::Bool
        | Type::Text
        | Type::Int
        | Type::Number
        | Type::Percent
        | Type::Quantity(_)
        | Type::Date
        | Type::DateTime => {
            check_scalar(ty, value, path, out);
        }
        Type::Enum(enumeration) => match value.as_str() {
            Some(variant) if enumeration.has(variant) => {}
            _ => finding(
                out,
                path,
                Message::with(
                    message!("This value must be one of {0}"),
                    [Arg::Text(
                        enumeration
                            .variants
                            .iter()
                            .map(|(name, _)| name.as_str())
                            .collect::<Vec<_>>()
                            .join(", "),
                    )],
                ),
            ),
        },
        Type::List(item) => match value.as_array() {
            Some(items) => {
                for (index, entry) in items.iter().enumerate() {
                    check_value(item, entry, &format!("{path}[{index}]"), out);
                }
            }
            None => wrong(out, path, ty),
        },
        Type::Record(record) => {
            let Some(object) = value.as_object() else {
                return wrong(out, path, ty);
            };
            for field in &record.fields {
                match object.get(&field.name) {
                    Some(entry) => {
                        check_value(&field.ty, entry, &format!("{path}.{}", field.name), out);
                    }
                    None if field.ty.is_optional() => {}
                    None => finding(
                        out,
                        path,
                        Message::with(message!("The field {0} is missing"), [quote(&field.name)]),
                    ),
                }
            }
            for key in object.keys() {
                if record.field(key).is_none() {
                    finding(
                        out,
                        &format!("{path}.{key}"),
                        Message::with(
                            message!("{0} has no field {1}"),
                            [quote(&ty.to_string()), quote(key)],
                        ),
                    );
                }
            }
        }
    }
}

/// Checks a value of a type that is not a list, record, enum or optional.
fn check_scalar(ty: &Type, value: &Json, path: &str, out: &mut Vec<ValueFinding>) {
    match ty {
        Type::Bool => {
            if !value.is_boolean() {
                wrong(out, path, ty);
            }
        }
        Type::Text => {
            if !value.is_string() {
                wrong(out, path, ty);
            }
        }
        Type::Int | Type::Number => match tagged(value, &["$n"])
            .and_then(|object| decimal(object.get("$n")))
        {
            None => wrong(out, path, ty),
            Some(number) if matches!(ty, Type::Int) && number.fract() != Decimal::ZERO => finding(
                out,
                path,
                Message::with(
                    message!("{0} isn't a whole number"),
                    [Arg::Text(number.to_string())],
                ),
            ),
            Some(_) => {}
        },
        Type::Percent => {
            let ok = tagged(value, &["$q", "u"]).is_some_and(|object| {
                decimal(object.get("$q")).is_some()
                    && object.get("u").and_then(Json::as_str) == Some("%")
            });
            if !ok {
                wrong(out, path, ty);
            }
        }
        Type::Quantity(quantity) => {
            let Some(object) =
                tagged(value, &["$q", "u"]).filter(|object| decimal(object.get("$q")).is_some())
            else {
                return wrong(out, path, ty);
            };
            let unit = object.get("u").and_then(Json::as_str).unwrap_or_default();
            match Unit::parse(unit) {
                Ok(parsed) if parsed.dim() == quantity.dim && unit != "%" => {}
                _ => finding(
                    out,
                    path,
                    Message::with(
                        message!("{0} isn't a unit of {1}"),
                        [quote(unit), Arg::from(words(ty))],
                    ),
                ),
            }
        }
        Type::Date | Type::DateTime => {
            let text = tagged(value, &["$date"])
                .and_then(|object| object.get("$date"))
                .and_then(Json::as_str);
            let ok = text.is_some_and(|text| {
                if matches!(ty, Type::Date) {
                    valid_date(text)
                } else {
                    valid_date_time(text)
                }
            });
            if !ok {
                wrong(out, path, ty);
            }
        }
        _ => {}
    }
}

fn digits(text: &str, count: usize) -> Option<u32> {
    (text.len() == count && text.bytes().all(|byte| byte.is_ascii_digit()))
        .then(|| text.parse().ok())?
}

/// `2026-10-22`, a real calendar date.
fn valid_date(text: &str) -> bool {
    let mut parts = text.split('-');
    let (Some(year), Some(month), Some(day), None) =
        (parts.next(), parts.next(), parts.next(), parts.next())
    else {
        return false;
    };
    let (Some(year), Some(month), Some(day)) = (digits(year, 4), digits(month, 2), digits(day, 2))
    else {
        return false;
    };
    let leap = year % 4 == 0 && (year % 100 != 0 || year % 400 == 0);
    let last = match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if leap => 29,
        2 => 28,
        _ => return false,
    };
    (1..=last).contains(&day)
}

/// `2026-10-22T08:00:00-05:00`, `…T08:00Z`: RFC 3339 with an offset.
fn valid_date_time(text: &str) -> bool {
    let Some((date, rest)) = text.split_once('T') else {
        return false;
    };
    if !valid_date(date) {
        return false;
    }
    let (clock, offset) = if let Some(clock) = rest.strip_suffix('Z') {
        (clock, None)
    } else if let Some(at) = rest.rfind(['+', '-']) {
        (rest.get(..at).unwrap_or_default(), rest.get(at + 1..))
    } else {
        return false;
    };
    let fields: Vec<&str> = clock.split(':').collect();
    let clock_ok = match fields.as_slice() {
        [hour, minute] => {
            digits(hour, 2).is_some_and(|h| h < 24) && digits(minute, 2).is_some_and(|m| m < 60)
        }
        [hour, minute, second] => {
            digits(hour, 2).is_some_and(|h| h < 24)
                && digits(minute, 2).is_some_and(|m| m < 60)
                && digits(second, 2).is_some_and(|s| s < 60)
        }
        _ => false,
    };
    let offset_ok = offset.is_none_or(|offset| {
        let mut parts = offset.split(':');
        matches!(
            (parts.next().and_then(|h| digits(h, 2)), parts.next().and_then(|m| digits(m, 2)), parts.next()),
            (Some(hours), Some(minutes), None) if hours <= 18 && minutes < 60
        )
    });
    clock_ok && offset_ok
}

#[cfg(test)]
mod tests {
    use super::{check_value, valid_date, valid_date_time};
    use crate::tariff::types::Type;
    use crate::tariff::units::Unit;
    use serde_json::json;

    fn findings(ty: &Type, value: &serde_json::Value) -> Vec<String> {
        let mut out = Vec::new();
        check_value(ty, value, "rule.x", &mut out);
        out.into_iter()
            .map(|finding| format!("{}: {}", finding.path, finding.message.english()))
            .collect()
    }

    #[test]
    fn quantities_check_their_dimension() {
        let rate = Type::quantity_in(&Unit::parse("$/mi").unwrap_or_default());
        assert!(findings(&rate, &json!({"$q": "2.35", "u": "$/km"})).is_empty());
        assert_eq!(
            findings(&rate, &json!({"$q": "2.35", "u": "$/cwt"})),
            ["rule.x: `$/cwt` isn't a unit of money per mile"]
        );
        assert_eq!(
            findings(&Type::Int, &json!({"$n": "1.5"})),
            ["rule.x: 1.5 isn't a whole number"]
        );
        assert_eq!(
            findings(&Type::Int, &json!(3)),
            ["rule.x: This value must be a whole number"]
        );
    }

    #[test]
    fn dates_are_validated() {
        assert!(valid_date("2026-02-28"));
        assert!(!valid_date("2026-02-30"));
        assert!(valid_date_time("2026-10-22T08:00:00-05:00"));
        assert!(valid_date_time("2026-10-22T08:00Z"));
        assert!(!valid_date_time("2026-10-22T25:00Z"));
    }
}
