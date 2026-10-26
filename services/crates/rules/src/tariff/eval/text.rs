//! Text properties and methods: `len`, `isEmpty`, `upper`, `lower`, `trim`, `startsWith`,
//! `endsWith`, `contains`, `replace`, `split`, `matches`, and indexing by character.
//!
//! Length and indexing count characters (Unicode scalar values), never bytes. Every byte of text
//! a method builds is charged to the text budget, and every item `split` creates to the items
//! budget. `matches` uses `regex-lite`, whose matching is linear in the text, and refuses a
//! pattern whose compiled form passes a size limit, so no pattern can make matching slow.

use super::budget::Meter;
use super::error::{Code, EvalError};
use super::value::Value;
use regex_lite::{Regex, RegexBuilder};
use rust_decimal::Decimal;
use rust_decimal::prelude::ToPrimitive;
use trenova_module_kit::message::{Arg, Message};

/// The most a compiled pattern may take, in bytes.
const REGEX_SIZE_LIMIT: usize = 64 * 1024;
/// How deeply a pattern may nest groups and repetitions.
const REGEX_NEST_LIMIT: u32 = 32;

/// A text property: `len` or `isEmpty`.
#[must_use]
pub fn property(text: &str, name: &str) -> Option<Value> {
    match name {
        "len" => Some(Value::Int(Decimal::from(text.chars().count()))),
        "isEmpty" => Some(Value::Bool(text.is_empty())),
        _ => None,
    }
}

fn built(meter: &mut Meter, text: &str) -> Result<Value, EvalError> {
    meter.text(text.len())?;
    Ok(Value::text(text))
}

fn text_arg<'a>(args: &'a [Value], index: usize, method: &str) -> Result<&'a str, EvalError> {
    match args.get(index) {
        Some(Value::Text(text)) => Ok(text),
        other => Err(EvalError::internal(Message::with(
            trenova_module_kit::message!("{0} needs text, not {1}"),
            [
                Arg::from(method),
                Arg::from(other.map_or_else(|| Value::None.kind(), Value::kind)),
            ],
        ))),
    }
}

/// Calls a text method, or `None` when `name` is not one. `regex` is `matches`'s pattern,
/// compiled by the caller (which caches what it compiles).
///
/// # Errors
///
/// When an argument is unusable or a budget runs out.
pub fn call(
    text: &str,
    name: &str,
    args: &[Value],
    meter: &mut Meter,
    regex: Option<&Regex>,
) -> Option<Result<Value, EvalError>> {
    Some(match name {
        "upper" => built(meter, &text.to_uppercase()),
        "lower" => built(meter, &text.to_lowercase()),
        "trim" => built(meter, text.trim()),
        "startsWith" => text_arg(args, 0, name).map(|prefix| Value::Bool(text.starts_with(prefix))),
        "endsWith" => text_arg(args, 0, name).map(|suffix| Value::Bool(text.ends_with(suffix))),
        "contains" => text_arg(args, 0, name).map(|part| Value::Bool(text.contains(part))),
        "replace" => replace(text, args, meter),
        "split" => split(text, args, meter),
        "matches" => text_arg(args, 0, name).and_then(|_| {
            regex
                .map(|regex| Value::Bool(regex.is_match(text)))
                .ok_or_else(|| {
                    EvalError::internal(Message::new(trenova_module_kit::message!(
                        "matches was called without its pattern"
                    )))
                })
        }),
        _ => return None,
    })
}

fn replace(text: &str, args: &[Value], meter: &mut Meter) -> Result<Value, EvalError> {
    let from = text_arg(args, 0, "replace")?;
    let to = text_arg(args, 1, "replace")?;
    if from.is_empty() {
        return Err(EvalError::new(
            Code::Argument,
            Message::new(trenova_module_kit::message!(
                "replace needs some text to look for, not \"\""
            )),
        ));
    }
    let count = text.matches(from).count();
    let size = text
        .len()
        .saturating_sub(count.saturating_mul(from.len()))
        .saturating_add(count.saturating_mul(to.len()));
    meter.text(size)?;
    Ok(Value::text(&text.replace(from, to)))
}

fn split(text: &str, args: &[Value], meter: &mut Meter) -> Result<Value, EvalError> {
    let separator = text_arg(args, 0, "split")?;
    if separator.is_empty() {
        return Err(EvalError::new(
            Code::Argument,
            Message::new(trenova_module_kit::message!(
                "split needs a separator, not \"\""
            )),
        ));
    }
    let count = text.matches(separator).count().saturating_add(1);
    meter.items_would_fit(u64::try_from(count).unwrap_or(u64::MAX))?;
    meter.items(count)?;
    meter.text(text.len())?;
    Ok(Value::list(
        text.split(separator).map(Value::text).collect(),
    ))
}

/// Compiles a pattern for `matches`.
///
/// # Errors
///
/// When the pattern does not compile or passes the size limit.
pub fn compile(pattern: &str) -> Result<Regex, EvalError> {
    RegexBuilder::new(pattern)
        .size_limit(REGEX_SIZE_LIMIT)
        .nest_limit(REGEX_NEST_LIMIT)
        .build()
        .map_err(|_| {
            EvalError::with(
                Code::Regex,
                trenova_module_kit::message!("\"{0}\" is not a pattern matches can use"),
                [Arg::from(pattern)],
            )
        })
}

/// `text[i]`: the character at a 0-based index, or `None` outside the text.
#[must_use]
pub fn index(text: &str, at: Decimal) -> Option<Value> {
    let position = at.to_usize()?;
    if !at.fract().is_zero() {
        return None;
    }
    text.chars()
        .nth(position)
        .map(|character| Value::text(character.encode_utf8(&mut [0; 4])))
}

#[cfg(test)]
mod tests {
    use super::{call, compile, index, property};
    use crate::tariff::eval::budget::{Limits, Meter};
    use crate::tariff::eval::value::Value;
    use rust_decimal::Decimal;

    fn run(text: &str, name: &str, args: &[Value]) -> String {
        let mut meter = Meter::new(Limits::DEFAULT);
        let regex = match args.first() {
            Some(Value::Text(pattern)) if name == "matches" => compile(pattern).ok(),
            _ => None,
        };
        if name == "matches" && regex.is_none() {
            return String::from("regex");
        }
        call(text, name, args, &mut meter, regex.as_ref()).map_or_else(
            || String::from("unknown"),
            |result| result.map_or_else(|error| error.code.as_str().to_owned(), |v| v.display()),
        )
    }

    #[test]
    fn methods_work_on_characters() {
        assert_eq!(
            property("café", "len").map(|v| v.display()).as_deref(),
            Some("4")
        );
        assert_eq!(run(" Atl ", "trim", &[]), "Atl");
        assert_eq!(run("atl", "upper", &[]), "ATL");
        assert_eq!(run("a-b-c", "split", &[Value::text("-")]), "[a, b, c]");
        assert_eq!(
            run("a-b", "replace", &[Value::text("-"), Value::text("+")]),
            "a+b"
        );
        assert_eq!(
            run("75201", "matches", &[Value::text("^75[0-9]{3}$")]),
            "true"
        );
        assert_eq!(run("x", "matches", &[Value::text("(")]), "regex");
        assert_eq!(run("x", "split", &[Value::text("")]), "argument");
        assert_eq!(
            index("café", Decimal::from(3))
                .map(|v| v.display())
                .as_deref(),
            Some("é")
        );
        assert!(index("café", Decimal::from(4)).is_none());
    }
}
