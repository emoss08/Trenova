//! The built-in functions that take plain values: numbers, text, lists and maps, bits, dates.
//!
//! The predicate functions (`map`, `filter`, `reduce` and the rest) and the rate table lookups
//! live in the interpreter, because they run formula code or read the request's tables.
//!
//! Semantics follow the Go engine (`engine/functions.go`) and expr's builtins, with the
//! contract's changes: one exact number type, so the rounding family, `sum` and `avg` are exact;
//! text counts characters (`len`, `indexOf`, slices); maps are in key order (`keys`, `values`,
//! `toPairs`); aggregates take numbers and lists of numbers flattened one level; `date(text)`
//! takes text only. Every list item and byte of text a function builds is metered.

use super::budget::Meter;
use super::error::Code;
use super::error::EvalError;
use super::error::expected_list;
use super::error::expected_number;
use super::error::expected_text;
use super::error::expected_whole_kind;
use super::error::invalid_argument;
use super::error::kind_phrase;
use super::number;
use super::number::Rounding;
use super::operators::map_key;
use super::operators::too_deep;
use super::time::DurationError;
use super::time::parse_date;
use super::time::parse_duration;
use super::value::Kind;
use super::value::Value;
use super::value::format_number;
use crate::message::Arg;
use regex_lite::Regex;
use regex_lite::RegexBuilder;
use rust_decimal::Decimal;
use std::cmp::Ordering;
use std::collections::BTreeMap;
use std::collections::BTreeSet;
use std::rc::Rc;

/// The most rounding decimals, either way.
const MAX_ROUND_DECIMALS: i64 = 12;
/// How many compiled regular expressions one program keeps.
const REGEX_CACHE: usize = 64;

/// What a built-in may use besides its arguments.
#[derive(Debug)]
pub struct Context<'a> {
    /// The program's budgets.
    pub meter: &'a mut Meter,
    /// Regular expressions compiled so far in this program.
    pub regexes: &'a mut RegexCache,
}

/// Compiled regular expressions, by pattern.
#[derive(Debug, Default)]
pub struct RegexCache {
    compiled: BTreeMap<String, Regex>,
}

impl RegexCache {
    /// Whether `text` matches `pattern` (Go's `regexp` syntax, as far as `regex-lite` reads it).
    /// The step cost is the product of the two lengths, the matcher's own worst case.
    pub fn matches(
        &mut self,
        text: &str,
        pattern: &str,
        meter: &mut Meter,
    ) -> Result<bool, EvalError> {
        meter.steps((text.len() + 1).saturating_mul(pattern.len() + 1))?;
        if let Some(regex) = self.compiled.get(pattern) {
            return Ok(regex.is_match(text));
        }
        let regex = RegexBuilder::new(pattern)
            .size_limit(1 << 20)
            .nest_limit(64)
            .build()
            .map_err(|error| {
                EvalError::with(
                    Code::Regex,
                    crate::message!("error parsing regexp {0}: {1}"),
                    [Arg::from(pattern), Arg::from(error.to_string())],
                )
            })?;
        let matched = regex.is_match(text);
        if self.compiled.len() < REGEX_CACHE {
            self.compiled.insert(pattern.to_owned(), regex);
        }
        Ok(matched)
    }
}

/// Every function this module implements.
pub const NAMES: &[&str] = &[
    "round",
    "roundUp",
    "roundDown",
    "roundHalfEven",
    "roundTo",
    "ceil",
    "floor",
    "abs",
    "min",
    "max",
    "sum",
    "avg",
    "mean",
    "median",
    "coalesce",
    "clamp",
    "pow",
    "sqrt",
    "upper",
    "lower",
    "trim",
    "trimPrefix",
    "trimSuffix",
    "hasPrefix",
    "hasSuffix",
    "indexOf",
    "lastIndexOf",
    "replace",
    "split",
    "splitAfter",
    "join",
    "len",
    "string",
    "float",
    "int",
    "type",
    "first",
    "last",
    "get",
    "take",
    "keys",
    "values",
    "toPairs",
    "fromPairs",
    "reverse",
    "uniq",
    "concat",
    "flatten",
    "sort",
    "bitand",
    "bitor",
    "bitxor",
    "bitnand",
    "bitshl",
    "bitshr",
    "bitushr",
    "bitnot",
    "date",
    "duration",
];

/// Whether `name` is a function this module implements.
#[must_use]
pub fn is_builtin(name: &str) -> bool {
    NAMES.contains(&name)
}

/// Calls the built-in `name`, or returns `None` when this module has no such function.
pub fn call(
    name: &str,
    args: &[Value],
    context: &mut Context<'_>,
) -> Option<Result<Value, EvalError>> {
    let result = match name {
        "round" => rounding(name, args, Rounding::HalfAwayFromZero),
        "roundUp" => rounding(name, args, Rounding::Up),
        "roundDown" => rounding(name, args, Rounding::Down),
        "roundHalfEven" => rounding(name, args, Rounding::HalfEven),
        "roundTo" => round_to(args),
        "ceil" => unary_number(name, args, |value| Ok(value.ceil())),
        "floor" => unary_number(name, args, |value| Ok(value.floor())),
        "abs" => unary_number(name, args, |value| Ok(value.abs())),
        "sqrt" => unary_number(name, args, number::sqrt),
        "min" => extremum(name, args, Ordering::Less, context),
        "max" => extremum(name, args, Ordering::Greater, context),
        "sum" => arity(name, args, 1, usize::MAX)
            .and_then(|()| total(args, context))
            .map(|(sum, _)| Value::Number(sum)),
        "avg" | "mean" => arity(name, args, 1, usize::MAX).and_then(|()| average(args, context)),
        "median" => arity(name, args, 1, usize::MAX).and_then(|()| median(args, context)),
        "coalesce" => arity(name, args, 1, usize::MAX).map(|()| {
            args.iter()
                .find(|value| !value.is_nil())
                .cloned()
                .unwrap_or(Value::Nil)
        }),
        "clamp" => clamp(args),
        "pow" => arity(name, args, 2, 2).and_then(|()| {
            number::pow(number_arg(args, 0)?, number_arg(args, 1)?).map(Value::Number)
        }),
        "upper" => map_text(name, args, context, str::to_uppercase),
        "lower" => map_text(name, args, context, str::to_lowercase),
        "trim" => trim(args, context),
        "trimPrefix" => trim_affix(name, args, context, true),
        "trimSuffix" => trim_affix(name, args, context, false),
        "hasPrefix" => {
            text_pair(name, args).map(|(text, part)| Value::Bool(text.starts_with(part)))
        }
        "hasSuffix" => text_pair(name, args).map(|(text, part)| Value::Bool(text.ends_with(part))),
        "indexOf" => text_pair(name, args).map(|(text, part)| {
            Value::int(text.find(part).map_or(-1, |at| char_count(&text[..at])))
        }),
        "lastIndexOf" => text_pair(name, args).map(|(text, part)| {
            Value::int(text.rfind(part).map_or(-1, |at| char_count(&text[..at])))
        }),
        "replace" => replace(args, context),
        "split" => split(name, args, context, false),
        "splitAfter" => split(name, args, context, true),
        "join" => join(args, context),
        "len" => length(args),
        "string" => arity(name, args, 1, 1).and_then(|()| {
            let text = args[0].display();
            context.meter.text(text.len())?;
            Ok(Value::Text(Rc::from(text)))
        }),
        "float" => convert(name, args, false),
        "int" => convert(name, args, true),
        "type" => arity(name, args, 1, 1).map(|()| Value::text(args[0].kind().type_name())),
        "first" => edge(name, args, true),
        "last" => edge(name, args, false),
        "get" => get(args, context),
        "take" => take(args, context),
        "keys" | "values" | "toPairs" => entries(name, args, context),
        "fromPairs" => from_pairs(args, context),
        "reverse" => arity(name, args, 1, 1)
            .and_then(|()| list_arg(name, args, 0))
            .and_then(|items| {
                let reversed: Vec<Value> = items.iter().rev().cloned().collect();
                built_list(reversed, context)
            }),
        "uniq" => uniq(args, context),
        "concat" => concat(args, context),
        "flatten" => flatten(args, context),
        "sort" => sort(args, context),
        "bitand" | "bitor" | "bitxor" | "bitnand" | "bitshl" | "bitshr" | "bitushr" => {
            bits(name, args)
        }
        "bitnot" => arity(name, args, 1, 1)
            .and_then(|()| whole_arg(args, 0).map(|value| Value::int(!value))),
        "date" => date(args),
        "duration" => duration(args),
        _ => return None,
    };
    Some(result)
}

fn char_count(text: &str) -> i64 {
    i64::try_from(text.chars().count()).unwrap_or(i64::MAX)
}

/// Fails unless the call has between `min` and `max` arguments.
pub fn arity(name: &str, args: &[Value], min: usize, max: usize) -> Result<(), EvalError> {
    if (min..=max).contains(&args.len()) {
        return Ok(());
    }
    let expected = if min == max {
        min.to_string()
    } else if max == usize::MAX {
        format!("at least {min}")
    } else {
        format!("{min} to {max}")
    };
    Err(EvalError::with(
        Code::Argument,
        crate::message!("invalid number of arguments for {0} (expected {1}, got {2})"),
        [
            Arg::from(name),
            Arg::from(expected),
            Arg::from(args.len().to_string()),
        ],
    ))
}

fn empty_at(error: EvalError, args: &[Value], index: usize) -> EvalError {
    if args.get(index).is_some_and(Value::is_nil) {
        error.empty_argument(index)
    } else {
        error
    }
}

/// The number at `index`.
pub fn number_arg(args: &[Value], index: usize) -> Result<Decimal, EvalError> {
    match args.get(index) {
        Some(Value::Number(value)) => Ok(*value),
        Some(other) => Err(empty_at(expected_number(other.kind_key()), args, index)),
        None => Err(expected_number("nil")),
    }
}

/// The whole number at `index`, as an `i64`.
pub fn whole_arg(args: &[Value], index: usize) -> Result<i64, EvalError> {
    match args.get(index) {
        Some(Value::Number(value)) => number::to_i64(*value),
        Some(other) => Err(empty_at(expected_whole_kind(other.kind_key()), args, index)),
        None => Err(expected_whole_kind("nil")),
    }
}

/// The text at `index`.
pub fn text_arg(args: &[Value], index: usize) -> Result<&str, EvalError> {
    match args.get(index) {
        Some(Value::Text(text)) => Ok(text),
        Some(other) => Err(empty_at(expected_text(other.kind_key()), args, index)),
        None => Err(expected_text("nil")),
    }
}

/// The list at `index`.
pub fn list_arg<'a>(name: &str, args: &'a [Value], index: usize) -> Result<&'a [Value], EvalError> {
    if index == 0 {
        arity(name, args, 1, usize::MAX)?;
    }
    match args.get(index) {
        Some(Value::List(list)) => Ok(list.items()),
        Some(other) => Err(empty_at(expected_list(other.kind_key()), args, index)),
        None => Err(expected_list("nil")),
    }
}

/// A list the function built, metered by its items.
fn built_list(items: Vec<Value>, context: &mut Context<'_>) -> Result<Value, EvalError> {
    context.meter.items(items.len())?;
    Value::list(items).map_err(|_| too_deep())
}

fn built_text(text: String, context: &mut Context<'_>) -> Result<Value, EvalError> {
    context.meter.text(text.len())?;
    Ok(Value::Text(Rc::from(text)))
}

fn rounding(name: &str, args: &[Value], rounding: Rounding) -> Result<Value, EvalError> {
    arity(name, args, 1, 2)?;
    let value = number_arg(args, 0)?;
    let places = if args.len() > 1 {
        whole_arg(args, 1)?
    } else {
        0
    };
    if !(-MAX_ROUND_DECIMALS..=MAX_ROUND_DECIMALS).contains(&places) {
        return Err(EvalError::with(
            Code::Argument,
            crate::message!("round decimals must be between -12 and 12, got {0}"),
            [Arg::from(places.to_string())],
        ));
    }
    number::round(value, places, rounding).map(Value::Number)
}

fn round_to(args: &[Value]) -> Result<Value, EvalError> {
    arity("roundTo", args, 2, 2)?;
    let value = number_arg(args, 0)?;
    let step = number_arg(args, 1)?;
    if step <= Decimal::ZERO {
        return Err(EvalError::with(
            Code::Argument,
            crate::message!("roundTo increment must be a positive number, got {0}"),
            [Arg::from(format_number(step))],
        ));
    }
    let multiples = number::div(value, step)?
        .round_dp_with_strategy(0, rust_decimal::RoundingStrategy::MidpointAwayFromZero);
    number::mul(multiples, step).map(Value::Number)
}

fn unary_number(
    name: &str,
    args: &[Value],
    apply: impl Fn(Decimal) -> Result<Decimal, EvalError>,
) -> Result<Value, EvalError> {
    arity(name, args, 1, 1)?;
    apply(number_arg(args, 0)?).map(Value::Number)
}

/// The numbers of an aggregate's arguments, lists flattened one level.
fn numbers(args: &[Value], context: &mut Context<'_>) -> Result<Vec<Decimal>, EvalError> {
    let mut values = Vec::with_capacity(args.len());
    for (index, arg) in args.iter().enumerate() {
        match arg {
            Value::Number(value) => values.push(*value),
            Value::List(list) => {
                context.meter.steps(list.items().len())?;
                for item in list.items() {
                    match item {
                        Value::Number(value) => values.push(*value),
                        other => return Err(expected_number(other.kind_key())),
                    }
                }
            }
            other => {
                return Err(empty_at(expected_number(other.kind_key()), args, index));
            }
        }
    }
    Ok(values)
}

fn extremum(
    name: &str,
    args: &[Value],
    better: Ordering,
    context: &mut Context<'_>,
) -> Result<Value, EvalError> {
    let values = numbers(args, context)?;
    let mut iter = values.into_iter();
    let Some(mut best) = iter.next() else {
        return Err(EvalError::with(
            Code::Argument,
            crate::message!("{0} requires at least one argument"),
            [Arg::from(name)],
        ));
    };
    for candidate in iter {
        if candidate.cmp(&best) == better {
            best = candidate;
        }
    }
    Ok(Value::Number(best))
}

fn total(args: &[Value], context: &mut Context<'_>) -> Result<(Decimal, usize), EvalError> {
    let values = numbers(args, context)?;
    let mut sum = Decimal::ZERO;
    for value in &values {
        sum = number::add(sum, *value)?;
    }
    Ok((sum, values.len()))
}

fn average(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    let (sum, count) = total(args, context)?;
    if count == 0 {
        return Ok(Value::Number(Decimal::ZERO));
    }
    number::div(sum, Decimal::from(count)).map(Value::Number)
}

fn median(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    let mut values = numbers(args, context)?;
    if values.is_empty() {
        return Ok(Value::Number(Decimal::ZERO));
    }
    context.meter.steps(sort_cost(values.len()))?;
    values.sort();
    let middle = values.len() / 2;
    if values.len() % 2 == 1 {
        return Ok(Value::Number(values[middle]));
    }
    let sum = number::add(values[middle - 1], values[middle])?;
    number::div(sum, Decimal::TWO).map(Value::Number)
}

fn sort_cost(count: usize) -> usize {
    count.saturating_mul(usize::try_from(count.max(2).ilog2()).unwrap_or(64) + 1)
}

fn clamp(args: &[Value]) -> Result<Value, EvalError> {
    arity("clamp", args, 3, 3)?;
    let value = number_arg(args, 0)?;
    let low = number_arg(args, 1)?;
    let high = number_arg(args, 2)?;
    Ok(Value::Number(if value < low {
        low
    } else if value > high {
        high
    } else {
        value
    }))
}

fn map_text(
    name: &str,
    args: &[Value],
    context: &mut Context<'_>,
    apply: fn(&str) -> String,
) -> Result<Value, EvalError> {
    arity(name, args, 1, 1)?;
    let text = text_arg(args, 0)?;
    built_text(apply(text), context)
}

fn trim(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity("trim", args, 1, 2)?;
    let text = text_arg(args, 0)?;
    let trimmed = if args.len() == 2 {
        let cutset = text_arg(args, 1)?;
        text.trim_matches(|character| cutset.contains(character))
    } else {
        text.trim()
    };
    built_text(trimmed.to_owned(), context)
}

fn trim_affix(
    name: &str,
    args: &[Value],
    context: &mut Context<'_>,
    prefix: bool,
) -> Result<Value, EvalError> {
    arity(name, args, 1, 2)?;
    let text = text_arg(args, 0)?;
    let affix = if args.len() == 2 {
        text_arg(args, 1)?
    } else {
        " "
    };
    let trimmed = if prefix {
        text.strip_prefix(affix).unwrap_or(text)
    } else {
        text.strip_suffix(affix).unwrap_or(text)
    };
    built_text(trimmed.to_owned(), context)
}

fn text_pair<'a>(name: &str, args: &'a [Value]) -> Result<(&'a str, &'a str), EvalError> {
    arity(name, args, 2, 2)?;
    Ok((text_arg(args, 0)?, text_arg(args, 1)?))
}

/// How many non-overlapping times `part` occurs; an empty `part` occurs before every character
/// and at the end, as Go's `strings.Count`.
fn occurrences(text: &str, part: &str) -> usize {
    if part.is_empty() {
        return text.chars().count() + 1;
    }
    text.matches(part).count()
}

fn replace(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity("replace", args, 3, 4)?;
    let text = text_arg(args, 0)?;
    let old = text_arg(args, 1)?;
    let new = text_arg(args, 2)?;
    let limit = if args.len() == 4 {
        whole_arg(args, 3)?
    } else {
        -1
    };
    let found = occurrences(text, old);
    let count = if limit < 0 {
        found
    } else {
        found.min(usize::try_from(limit).unwrap_or(usize::MAX))
    };
    if old == new || count == 0 {
        return built_text(text.to_owned(), context);
    }
    let size = (text.len() + count.saturating_mul(new.len())).saturating_sub(count * old.len());
    context.meter.text_would_fit(size)?;
    let mut out = String::with_capacity(size);
    let mut start = 0;
    for done in 0..count {
        let at = if old.is_empty() {
            if done == 0 {
                start
            } else {
                start + text[start..].chars().next().map_or(0, char::len_utf8)
            }
        } else {
            start + text[start..].find(old).unwrap_or(text.len() - start)
        };
        out.push_str(&text[start..at]);
        out.push_str(new);
        start = at + old.len();
    }
    out.push_str(&text[start.min(text.len())..]);
    built_text(out, context)
}

fn split(
    name: &str,
    args: &[Value],
    context: &mut Context<'_>,
    keep_separator: bool,
) -> Result<Value, EvalError> {
    arity(name, args, 2, 3)?;
    let text = text_arg(args, 0)?;
    let separator = text_arg(args, 1)?;
    let limit = if args.len() == 3 {
        whole_arg(args, 2)?
    } else {
        -1
    };
    if limit == 0 {
        return built_list(Vec::new(), context);
    }
    let most = if limit < 0 {
        usize::MAX
    } else {
        usize::try_from(limit).unwrap_or(usize::MAX)
    };
    let mut pieces: Vec<&str> = Vec::new();
    if separator.is_empty() {
        let mut rest = text;
        while let Some(character) = rest.chars().next() {
            if pieces.len() + 1 == most {
                pieces.push(rest);
                break;
            }
            let width = character.len_utf8();
            pieces.push(&rest[..width]);
            rest = &rest[width..];
        }
    } else {
        let mut rest = text;
        while pieces.len() + 1 < most {
            let Some(at) = rest.find(separator) else {
                break;
            };
            let cut = if keep_separator {
                at + separator.len()
            } else {
                at
            };
            pieces.push(&rest[..cut]);
            rest = &rest[at + separator.len()..];
        }
        pieces.push(rest);
    }
    context.meter.steps(pieces.len())?;
    context
        .meter
        .text(pieces.iter().map(|piece| piece.len()).sum())?;
    let items = pieces.into_iter().map(Value::text).collect();
    built_list(items, context)
}

fn join(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity("join", args, 1, 2)?;
    let items = list_arg("join", args, 0)?;
    let glue = if args.len() == 2 {
        text_arg(args, 1)?
    } else {
        ""
    };
    context.meter.steps(items.len())?;
    let mut size = glue.len().saturating_mul(items.len().saturating_sub(1));
    for item in items {
        match item {
            Value::Text(text) => size = size.saturating_add(text.len()),
            other => return Err(expected_text(other.kind_key())),
        }
    }
    context.meter.text_would_fit(size)?;
    let mut out = String::with_capacity(size);
    for (index, item) in items.iter().enumerate() {
        if index > 0 {
            out.push_str(glue);
        }
        if let Value::Text(text) = item {
            out.push_str(text);
        }
    }
    built_text(out, context)
}

fn length(args: &[Value]) -> Result<Value, EvalError> {
    arity("len", args, 1, 1)?;
    let count = match &args[0] {
        Value::Text(text) => text.chars().count(),
        Value::List(list) => list.items().len(),
        Value::Map(map) => map.entries().len(),
        other => return Err(empty_at(invalid_argument("len", other.kind_key()), args, 0)),
    };
    Ok(Value::int(i64::try_from(count).unwrap_or(i64::MAX)))
}

fn convert(name: &str, args: &[Value], whole: bool) -> Result<Value, EvalError> {
    arity(name, args, 1, 1)?;
    let number = match &args[0] {
        Value::Number(value) => *value,
        Value::Text(text) => {
            let parsed = if text.trim() == &**text && !text.is_empty() {
                crate::wire::parse_decimal(text).ok()
            } else {
                None
            };
            parsed.ok_or_else(|| conversion_error(name, (**text).to_owned()))?
        }
        Value::Nil => {
            return Err(conversion_error(name, String::from("nil")).empty_argument(0));
        }
        other => return Err(conversion_error(name, other.kind().type_name().to_owned())),
    };
    Ok(Value::Number(if whole { number.trunc() } else { number }))
}

fn conversion_error(name: &str, shown: String) -> EvalError {
    EvalError::with(
        Code::Argument,
        crate::message!("invalid operation: {0}({1})"),
        [Arg::from(name), Arg::from(shown)],
    )
}

fn edge(name: &str, args: &[Value], first: bool) -> Result<Value, EvalError> {
    arity(name, args, 1, 1)?;
    match &args[0] {
        Value::List(list) => Ok(if first {
            list.items().first()
        } else {
            list.items().last()
        }
        .cloned()
        .unwrap_or(Value::Nil)),
        Value::Nil => Ok(Value::Nil),
        other => Err(invalid_argument(name, other.kind_key())),
    }
}

fn get(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity("get", args, 2, 2)?;
    let key = &args[1];
    match &args[0] {
        Value::List(list) => {
            let at = whole_arg(args, 1)?;
            Ok(resolve(at, list.items().len()).map_or(Value::Nil, |at| list.items()[at].clone()))
        }
        Value::Text(text) => {
            let at = whole_arg(args, 1)?;
            let length = text.chars().count();
            match resolve(at, length).and_then(|at| text.chars().nth(at)) {
                Some(character) => built_text(character.to_string(), context),
                None => Ok(Value::Nil),
            }
        }
        Value::Map(map) => {
            if key.is_nil() {
                return Ok(Value::Nil);
            }
            let key = map_key(key, 1)?;
            Ok(map.get(&key).cloned().unwrap_or(Value::Nil))
        }
        _ => Ok(Value::Nil),
    }
}

fn resolve(index: i64, length: usize) -> Option<usize> {
    let length_signed = i64::try_from(length).ok()?;
    let at = if index < 0 {
        length_signed.checked_add(index)?
    } else {
        index
    };
    usize::try_from(at).ok().filter(|at| *at < length)
}

fn take(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity("take", args, 2, 2)?;
    let items = list_arg("take", args, 0)?;
    let count = whole_arg(args, 1)?;
    let Ok(count) = usize::try_from(count) else {
        return Err(EvalError::with(
            Code::Argument,
            crate::message!("cannot take {0} elements"),
            [Arg::from(count.to_string())],
        ));
    };
    built_list(items[..count.min(items.len())].to_vec(), context)
}

fn entries(name: &str, args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity(name, args, 1, 1)?;
    let Value::Map(map) = &args[0] else {
        return Err(empty_at(
            invalid_argument(name, args[0].kind_key()),
            args,
            0,
        ));
    };
    context.meter.steps(map.entries().len())?;
    let mut items = Vec::with_capacity(map.entries().len());
    for (key, value) in map.entries() {
        items.push(match name {
            "keys" => Value::Text(Rc::clone(key)),
            "values" => value.clone(),
            _ => {
                context.meter.items(2)?;
                Value::list(vec![Value::Text(Rc::clone(key)), value.clone()])
                    .map_err(|_| too_deep())?
            }
        });
    }
    built_list(items, context)
}

fn from_pairs(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity("fromPairs", args, 1, 1)?;
    let pairs = list_arg("fromPairs", args, 0)?;
    context.meter.steps(pairs.len())?;
    let mut entries: BTreeMap<Rc<str>, Value> = BTreeMap::new();
    for pair in pairs {
        let Value::List(pair_list) = pair else {
            return Err(EvalError::with(
                Code::Argument,
                crate::message!("invalid pair {0}"),
                [Arg::from(pair.display())],
            ));
        };
        let [key, value] = pair_list.items() else {
            return Err(EvalError::with(
                Code::Argument,
                crate::message!("invalid pair length {0}"),
                [Arg::from(pair.display())],
            ));
        };
        entries.insert(Rc::from(map_key(key, 0)?), value.clone());
    }
    context.meter.items(entries.len())?;
    Value::map(entries).map_err(|_| too_deep())
}

/// A key `uniq` can order, so it runs in `n log n` rather than comparing every pair.
#[derive(Debug, PartialEq, Eq, PartialOrd, Ord)]
enum Fingerprint {
    Nil,
    Bool(bool),
    Number(Decimal),
    Text(Rc<str>),
    Instant(i128),
    Duration(i64),
}

fn fingerprint(value: &Value) -> Option<Fingerprint> {
    Some(match value {
        Value::Nil => Fingerprint::Nil,
        Value::Bool(flag) => Fingerprint::Bool(*flag),
        Value::Number(number) => Fingerprint::Number(number.normalize()),
        Value::Text(text) => Fingerprint::Text(Rc::clone(text)),
        Value::Date(date) => Fingerprint::Instant(date.instant_nanos()),
        Value::Duration(nanos) => Fingerprint::Duration(*nanos),
        Value::List(_) | Value::Map(_) => return None,
    })
}

fn uniq(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity("uniq", args, 1, 1)?;
    let items = list_arg("uniq", args, 0)?;
    context.meter.steps(items.len())?;
    let mut seen: BTreeSet<Fingerprint> = BTreeSet::new();
    let mut compound: Vec<&Value> = Vec::new();
    let mut kept = Vec::new();
    for item in items {
        let fresh = if let Some(key) = fingerprint(item) {
            seen.insert(key)
        } else {
            context.meter.steps(compound.len())?;
            let repeated = compound.iter().any(|earlier| earlier.same(item));
            if !repeated {
                compound.push(item);
            }
            !repeated
        };
        if fresh {
            kept.push(item.clone());
        }
    }
    built_list(kept, context)
}

fn concat(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity("concat", args, 1, usize::MAX)?;
    let mut items = Vec::new();
    for index in 0..args.len() {
        let list = list_arg("concat", args, index)?;
        context.meter.steps(list.len())?;
        items.extend_from_slice(list);
    }
    built_list(items, context)
}

fn flatten(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity("flatten", args, 1, 1)?;
    let items = list_arg("flatten", args, 0)?;
    let mut out = Vec::new();
    flatten_into(items, &mut out, context)?;
    built_list(out, context)
}

fn flatten_into(
    items: &[Value],
    out: &mut Vec<Value>,
    context: &mut Context<'_>,
) -> Result<(), EvalError> {
    context.meter.steps(items.len())?;
    for item in items {
        match item {
            Value::List(list) => flatten_into(list.items(), out, context)?,
            other => out.push(other.clone()),
        }
    }
    Ok(())
}

/// Fails unless every value is of one orderable kind, so sorting has a total order.
pub fn sortable(values: &[Value]) -> Result<(), EvalError> {
    let Some(first) = values.first() else {
        return Ok(());
    };
    let kind = first.kind();
    if !matches!(
        kind,
        Kind::Number | Kind::Text | Kind::Date | Kind::Duration
    ) {
        return Err(super::error::cannot_compare(
            first.kind_key(),
            first.kind_key(),
        ));
    }
    if let Some(other) = values.iter().find(|value| value.kind() != kind) {
        return Err(super::error::cannot_compare(
            first.kind_key(),
            other.kind_key(),
        ));
    }
    Ok(())
}

/// Reads `sort`'s and `sortBy`'s order argument: `"asc"` or `"desc"`.
pub fn sort_order(value: Option<&Value>) -> Result<bool, EvalError> {
    match value {
        None => Ok(false),
        Some(Value::Text(order)) if &**order == "asc" => Ok(false),
        Some(Value::Text(order)) if &**order == "desc" => Ok(true),
        Some(Value::Text(order)) => Err(EvalError::with(
            Code::Argument,
            crate::message!("invalid order {0}, expected asc or desc"),
            [Arg::from(order.to_string())],
        )),
        Some(other) => Err(EvalError::with(
            Code::Type,
            crate::message!("sort order argument must be a string (got {0})"),
            [kind_phrase(other.kind_key())],
        )),
    }
}

/// Sorts `items` by `keys` (one per item), stably, ascending or descending.
pub fn sort_by_keys(
    items: &[Value],
    keys: &[Value],
    descending: bool,
    context: &mut Context<'_>,
) -> Result<Value, EvalError> {
    sortable(keys)?;
    context.meter.steps(sort_cost(items.len()))?;
    let mut order: Vec<usize> = (0..items.len()).collect();
    order.sort_by(|left, right| {
        let ordering = keys[*left]
            .compare(&keys[*right])
            .unwrap_or(Ordering::Equal);
        if descending {
            ordering.reverse()
        } else {
            ordering
        }
    });
    built_list(
        order
            .into_iter()
            .map(|index| items[index].clone())
            .collect(),
        context,
    )
}

fn sort(args: &[Value], context: &mut Context<'_>) -> Result<Value, EvalError> {
    arity("sort", args, 1, 2)?;
    let items = list_arg("sort", args, 0)?;
    let descending = sort_order(args.get(1))?;
    sort_by_keys(items, items, descending, context)
}

fn bits(name: &str, args: &[Value]) -> Result<Value, EvalError> {
    arity(name, args, 2, 2)?;
    let left = whole_arg(args, 0)?;
    let right = whole_arg(args, 1)?;
    let shift = || {
        if right < 0 {
            return Err(EvalError::with(
                Code::Argument,
                crate::message!("invalid operation: negative shift count {0} (type int)"),
                [Arg::from(right.to_string())],
            ));
        }
        Ok(u32::try_from(right).unwrap_or(u32::MAX))
    };
    let result = match name {
        "bitand" => left & right,
        "bitor" => left | right,
        "bitxor" => left ^ right,
        "bitnand" => left & !right,
        "bitshl" => left.checked_shl(shift()?).unwrap_or(0),
        "bitshr" => left
            .checked_shr(shift()?)
            .unwrap_or(if left < 0 { -1 } else { 0 }),
        _ => {
            let unsigned = u64::from_ne_bytes(left.to_ne_bytes());
            let shifted = unsigned.checked_shr(shift()?).unwrap_or(0);
            i64::from_ne_bytes(shifted.to_ne_bytes())
        }
    };
    Ok(Value::int(result))
}

fn date(args: &[Value]) -> Result<Value, EvalError> {
    arity("date", args, 1, 1)?;
    let text = text_arg(args, 0)?;
    parse_date(text).map(Value::Date).ok_or_else(|| {
        EvalError::with(
            Code::Argument,
            crate::message!("invalid date {0}"),
            [Arg::from(text)],
        )
    })
}

fn duration(args: &[Value]) -> Result<Value, EvalError> {
    arity("duration", args, 1, 1)?;
    let text = text_arg(args, 0)?;
    parse_duration(text).map(Value::Duration).map_err(|error| {
        let args = [Arg::from(text)];
        match error {
            DurationError::Invalid | DurationError::Overflow => EvalError::with(
                Code::Argument,
                crate::message!("time: invalid duration \"{0}\""),
                args,
            ),
            DurationError::MissingUnit => EvalError::with(
                Code::Argument,
                crate::message!("time: missing unit in duration \"{0}\""),
                args,
            ),
            DurationError::UnknownUnit => EvalError::with(
                Code::Argument,
                crate::message!("time: unknown unit in duration \"{0}\""),
                args,
            ),
        }
    })
}

#[cfg(test)]
mod tests {
    use super::{Context, RegexCache, call};
    use crate::formula::eval::budget::{Limits, Meter};
    use crate::formula::eval::error::Code;
    use crate::formula::eval::tree::parse_value;
    use crate::formula::eval::value::Value;

    fn run(name: &str, args: &str) -> Result<Value, Code> {
        let mut meter = Meter::new(Limits::DEFAULT);
        let mut regexes = RegexCache::default();
        let mut context = Context {
            meter: &mut meter,
            regexes: &mut regexes,
        };
        let Value::List(list) = parse_value(args).expect("arguments") else {
            panic!("arguments are a list");
        };
        call(name, list.items(), &mut context)
            .expect("known function")
            .map_err(|error| error.code)
    }

    fn check(name: &str, args: &str, want: &str) {
        let got = run(name, args).unwrap_or_else(|code| panic!("{name}{args}: {code:?}"));
        let want = parse_value(want).expect("expected value");
        assert!(got.same(&want), "{name}{args}: got {}", got.display());
    }

    #[test]
    fn text_counts_characters() {
        check("indexOf", r#"["é-x", "-"]"#, "1");
        check("lastIndexOf", r#"["a-b-c", "-"]"#, "3");
        check("indexOf", r#"["abc", "z"]"#, "-1");
        check("lastIndexOf", r#"["abc", ""]"#, "3");
        check("len", r#"["🚚é"]"#, "2");
        check("split", r#"["a-b-c", "-"]"#, r#"["a", "b", "c"]"#);
        check("split", r#"["a-b-c", "-", 2]"#, r#"["a", "b-c"]"#);
        check("split", r#"["ab", ""]"#, r#"["a", "b"]"#);
        check("split", r#"["a-b", "-", 0]"#, "[]");
        check("splitAfter", r#"["a-b", "-"]"#, r#"["a-", "b"]"#);
        check("replace", r#"["a-b-c", "-", ""]"#, r#""abc""#);
        check("replace", r#"["a-b-c", "-", "+", 1]"#, r#""a+b-c""#);
        check("replace", r#"["ab", "", "."]"#, r#"".a.b.""#);
        check("trim", r#"["  x  "]"#, r#""x""#);
        check("trim", r#"["--x-", "-"]"#, r#""x""#);
        check("trimPrefix", r#"["C-ACME", "C-"]"#, r#""ACME""#);
        check("trimSuffix", r#"["75201-0000", "-0000"]"#, r#""75201""#);
        check("upper", r#"["tx"]"#, r#""TX""#);
        check("join", r#"[["a", "b"], "-"]"#, r#""a-b""#);
        check("string", "[12.50]", r#""12.5""#);
    }

    #[test]
    fn conversions() {
        check("int", r#"["12"]"#, "12");
        check("int", "[-3.9]", "-3");
        check("float", r#"["1.25"]"#, "1.25");
        check("type", "[1]", r#""number""#);
        assert_eq!(run("int", r#"["abc"]"#), Err(Code::Argument));
        assert_eq!(run("float", "[true]"), Err(Code::Argument));
    }

    #[test]
    fn lists_and_maps() {
        check("first", "[[]]", "null");
        check("last", "[[1, 2]]", "2");
        check("get", "[[1, 2], -1]", "2");
        check("get", "[[1, 2], 5]", "null");
        check("get", r#"[{"a": 1}, "b"]"#, "null");
        check("take", "[[1, 2, 3], 2]", "[1, 2]");
        check("take", "[[1], 5]", "[1]");
        check("keys", r#"[{"b": 1, "a": 2}]"#, r#"["a", "b"]"#);
        check("values", r#"[{"b": 1, "a": 2}]"#, "[2, 1]");
        check("toPairs", r#"[{"a": 1}]"#, r#"[["a", 1]]"#);
        check(
            "fromPairs",
            r#"[[["a", 1], ["b", 2]]]"#,
            r#"{"a": 1, "b": 2}"#,
        );
        check(
            "uniq",
            r#"[[1, 1.0, "1", [1], [1], null, null]]"#,
            r#"[1, "1", [1], null]"#,
        );
        check("concat", "[[1], [2, 3]]", "[1, 2, 3]");
        check("flatten", "[[1, [2, [3]]]]", "[1, 2, 3]");
        check("sort", "[[3, 1, 2]]", "[1, 2, 3]");
        check("sort", r#"[["b", "a"], "desc"]"#, r#"["b", "a"]"#);
        check("reverse", "[[1, 2]]", "[2, 1]");
        assert_eq!(run("sort", r#"[[1, "a"]]"#), Err(Code::Type));
        assert_eq!(run("sort", r#"[[1], "up"]"#), Err(Code::Argument));
        assert_eq!(run("take", "[[1], -1]"), Err(Code::Argument));
    }

    #[test]
    fn numbers() {
        check("round", "[2.675, 2]", "2.68");
        check("roundTo", "[10.13, 0.25]", "10.25");
        check("min", "[5, 3, 8]", "3");
        check("max", "[[1, 9], 3]", "9");
        check("sum", "[[0.1, 0.2], 0.3]", "0.6");
        check("avg", "[[1, 2], [3, 4]]", "2.5");
        check("avg", "[[]]", "0");
        assert_eq!(run("avg", "[]"), Err(Code::Argument));
        check("median", "[3, 1, 2, 4]", "2.5");
        check("clamp", "[15, 0, 10]", "10");
        check("coalesce", "[null, null, 3]", "3");
        assert_eq!(run("min", "[]"), Err(Code::Argument));
        assert_eq!(run("round", "[1, 13]"), Err(Code::Argument));
        assert_eq!(run("round", "[1, 1.5]"), Err(Code::Argument));
        assert_eq!(run("roundTo", "[10, 0]"), Err(Code::Argument));
        assert_eq!(run("sum", r#"[1, "two"]"#), Err(Code::Type));
    }

    #[test]
    fn bits_follow_go() {
        check("bitand", "[12, 10]", "8");
        check("bitor", "[12, 10]", "14");
        check("bitxor", "[12, 10]", "6");
        check("bitnand", "[12, 10]", "4");
        check("bitshl", "[1, 3]", "8");
        check("bitshr", "[-16, 2]", "-4");
        check("bitushr", "[-1, 60]", "15");
        check("bitshl", "[1, 64]", "0");
        check("bitnot", "[1]", "-2");
        assert_eq!(run("bitshl", "[1, -1]"), Err(Code::Argument));
        assert_eq!(run("bitand", "[1.5, 1]"), Err(Code::Argument));
    }

    #[test]
    fn dates_and_durations() {
        check(
            "date",
            r#"["2026-01-02"]"#,
            r#"{"$date": "2026-01-02T00:00:00Z"}"#,
        );
        check(
            "duration",
            r#"["1h30m"]"#,
            r#"{"$duration": "5400000000000"}"#,
        );
        assert_eq!(run("date", r#"["01/02/2026"]"#), Err(Code::Argument));
        assert_eq!(run("duration", r#"["1d"]"#), Err(Code::Argument));
    }

    #[test]
    fn regexes_compile_once_and_report_errors() {
        let mut meter = Meter::new(Limits::DEFAULT);
        let mut cache = RegexCache::default();
        assert_eq!(cache.matches("75201", "^7[0-9]{4}$", &mut meter), Ok(true));
        assert_eq!(cache.matches("85201", "^7[0-9]{4}$", &mut meter), Ok(false));
        assert_eq!(
            cache
                .matches("x", "(", &mut meter)
                .map_err(|error| error.code),
            Err(Code::Regex)
        );
    }
}
