//! `formula.evaluate`: a batch of formulas against one environment, in one boundary crossing.
//!
//! The server prices a shipment by evaluating the template's main expression and every
//! breakdown line against the same environment and rate tables, so one request carries them
//! all: the environment is read once, the tables are indexed once, and each program runs under
//! its own budget. The shapes are documented in `docs/engineering/rules-engine.md`
//! ("formula.evaluate and the rate table format").
//!
//! A program that does not parse comes back with a `syntax` error and is not run; the host runs
//! `formula.check` first, so that is a caller bug made visible rather than a price. Of the
//! `schema` (the same as `formula.check` takes), only the word operators matter here, since the
//! grammar needs them; the checker has already used the rest.

use super::budget::Limits;
use super::error::Code;
use super::error::EvalError;
use super::interp;
use super::interp::LookupRecord;
use super::interp::LookupSource;
use super::interp::Setup;
use super::tables::Matched;
use super::tables::TableWire;
use super::tables::Tables;
use super::trace::trace_value;
use super::tree::parse_value;
use super::value::Value;
use super::value::format_number;
use crate::formula::check::Schema;
use crate::formula::syntax;
use crate::message::Arg;
use crate::message::Message;
use crate::ops::OpResult;
use crate::ops::RequestError;
use crate::ops::decode;
use crate::ops::encode;
use crate::span::Span;
use crate::span::Utf16Index;
use serde::Deserialize;
use serde::Serialize;
use serde_json::Value as Json;
use serde_json::value::RawValue;
use std::collections::BTreeMap;

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct Request<'a> {
    programs: Vec<ProgramInput>,
    #[serde(default)]
    schema: Option<Schema>,
    #[serde(default, borrow)]
    env: Option<&'a RawValue>,
    #[serde(default)]
    tables: Option<Vec<TableWire>>,
    #[serde(default)]
    trace: TraceInput,
    #[serde(default)]
    budget: BudgetInput,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct ProgramInput {
    id: String,
    source: String,
    expect: Expect,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
enum Expect {
    Amount,
    Predicate,
}

#[derive(Default, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct TraceInput {
    #[serde(default)]
    statements: bool,
    #[serde(default)]
    lookups: bool,
}

#[derive(Default, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct BudgetInput {
    steps: Option<u64>,
    text_bytes: Option<u64>,
    items: Option<u64>,
}

#[derive(Serialize)]
struct Response {
    results: Vec<ProgramOutput>,
}

#[derive(Serialize)]
struct ProgramOutput {
    id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    value: Option<Json>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<ErrorOutput>,
    #[serde(skip_serializing_if = "Option::is_none")]
    statements: Option<Vec<StatementOutput>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    lookups: Option<Vec<LookupOutput>>,
}

#[derive(Serialize)]
struct ErrorOutput {
    code: &'static str,
    message: &'static str,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    args: Vec<Arg>,
    #[serde(skip_serializing_if = "Option::is_none")]
    from: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    to: Option<u32>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    fields: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    guards: Vec<String>,
}

#[derive(Serialize)]
struct StatementOutput {
    index: usize,
    kind: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    name: Option<String>,
    code: String,
    from: u32,
    to: u32,
    #[serde(skip_serializing_if = "Option::is_none")]
    value: Option<Json>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<ErrorOutput>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct LookupOutput {
    function: &'static str,
    table: String,
    keys: Vec<Json>,
    #[serde(skip_serializing_if = "Option::is_none")]
    value: Option<String>,
    #[serde(rename = "match", skip_serializing_if = "Option::is_none")]
    matched: Option<MatchOutput>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<ErrorOutput>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct MatchOutput {
    #[serde(skip_serializing_if = "Option::is_none")]
    matched_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    band_min: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    band_max: Option<String>,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    adjusted: bool,
}

/// Evaluates a batch of formulas against one environment.
pub fn evaluate(input: &RawValue) -> OpResult {
    let request: Request<'_> = decode(input)?;
    let limits = limits(&request.budget)?;
    let env = match request.env {
        Some(raw) => parse_value(raw.get())
            .map_err(|detail| RequestError::invalid_input(format!("env: {detail}")))?,
        None => Value::map(BTreeMap::new()).map_err(|_| RequestError::invalid_input("env"))?,
    };
    if !matches!(env, Value::Map(_)) {
        return Err(RequestError::invalid_input(
            "env must be an object of fields and inputs",
        ));
    }
    let tables = match &request.tables {
        Some(tables) => Some(Tables::build(tables).map_err(RequestError::invalid_input)?),
        None => None,
    };
    let operators: Vec<&str> = request
        .schema
        .iter()
        .flat_map(|schema| schema.operators.iter().map(String::as_str))
        .collect();
    let context = Context {
        env: &env,
        tables: tables.as_ref(),
        limits,
        trace: &request.trace,
        operators: &operators,
    };
    let results = request
        .programs
        .iter()
        .map(|program| run_program(program, &context))
        .collect();
    encode(&Response { results })
}

struct Context<'a> {
    env: &'a Value,
    tables: Option<&'a Tables>,
    limits: Limits,
    trace: &'a TraceInput,
    /// The schema's word operators, which the grammar needs to read the formula.
    operators: &'a [&'a str],
}

fn limits(budget: &BudgetInput) -> Result<Limits, RequestError> {
    let default = Limits::DEFAULT;
    let lowered = |name: &str, asked: Option<u64>, most: u64| match asked {
        None => Ok(most),
        Some(asked) if asked <= most => Ok(asked),
        Some(asked) => Err(RequestError::invalid_input(format!(
            "budget.{name} may only lower the default of {most}, not raise it to {asked}"
        ))),
    };
    Ok(Limits {
        steps: lowered("steps", budget.steps, default.steps)?,
        text_bytes: lowered("textBytes", budget.text_bytes, default.text_bytes)?,
        items: lowered("items", budget.items, default.items)?,
    })
}

fn run_program(program: &ProgramInput, context: &Context<'_>) -> ProgramOutput {
    let index = Utf16Index::new(&program.source);
    let parsed = syntax::parse(&program.source, context.operators);
    if let Some(diagnostic) = parsed.diagnostics.first() {
        let error = EvalError::new(Code::Syntax, diagnostic.message.clone()).at(diagnostic.span);
        return ProgramOutput {
            id: program.id.clone(),
            value: None,
            error: Some(error_output(&error, &index)),
            statements: None,
            lookups: None,
        };
    }
    let lookups = match (program.expect, context.tables) {
        (Expect::Predicate, _) => LookupSource::Stub,
        (Expect::Amount, Some(tables)) => LookupSource::Tables(tables),
        (Expect::Amount, None) => LookupSource::Unavailable,
    };
    let evaluation = interp::evaluate(
        &parsed,
        Setup {
            env: context.env,
            lookups,
            limits: context.limits,
            trace_statements: context.trace.statements,
            record_lookups: context.trace.lookups,
        },
    );
    let result_span = parsed
        .statements
        .last()
        .map_or_else(Span::default, |statement| statement.value.span);
    let result = evaluation
        .result
        .and_then(|value| conclude(program.expect, &value).map_err(|error| error.at(result_span)));
    let statements = context.trace.statements.then(|| {
        let last = parsed.statements.len().saturating_sub(1);
        evaluation
            .statements
            .iter()
            .map(|statement| {
                let (value, error) = if statement.index == last && statement.name.is_none() {
                    split(&result, &index)
                } else {
                    match &statement.outcome {
                        Ok(value) => (Some(trace_value(value)), None),
                        Err(error) => (None, Some(error_output(error, &index))),
                    }
                };
                let range = index.range(statement.span);
                StatementOutput {
                    index: statement.index,
                    kind: if statement.name.is_some() {
                        "let"
                    } else {
                        "result"
                    },
                    name: statement.name.clone(),
                    code: program
                        .source
                        .get(statement.span.start as usize..statement.span.end as usize)
                        .unwrap_or_default()
                        .trim()
                        .to_owned(),
                    from: range.from,
                    to: range.to,
                    value,
                    error,
                }
            })
            .collect()
    });
    let lookups = context.trace.lookups.then(|| {
        evaluation
            .lookups
            .iter()
            .map(|record| lookup_output(record, &index))
            .collect()
    });
    let (value, error) = split(&result, &index);
    ProgramOutput {
        id: program.id.clone(),
        value,
        error,
        statements,
        lookups,
    }
}

fn split(
    result: &Result<Json, EvalError>,
    index: &Utf16Index,
) -> (Option<Json>, Option<ErrorOutput>) {
    match result {
        Ok(value) => (Some(value.clone()), None),
        Err(error) => (None, Some(error_output(error, index))),
    }
}

/// What a program's value comes to: an amount is a number (a decimal string on the wire); a
/// predicate is true, false or a number, which the host reads as true when it is not zero.
/// The rules are the Go engine's `toDecimal` and `rejectBooleanAmount`.
fn conclude(expect: Expect, value: &Value) -> Result<Json, EvalError> {
    match (expect, value) {
        (_, Value::Number(number)) => Ok(Json::String(format_number(*number))),
        (Expect::Predicate, Value::Bool(flag)) => Ok(Json::Bool(*flag)),
        (Expect::Amount, Value::Bool(_)) => Err(EvalError::new(
            Code::BooleanAmount,
            Message::new(crate::message!(
                "expression produced true/false rather than an amount; write a conditional such as condition ? amount : 0"
            )),
        )),
        (_, Value::Nil) => Err(EvalError::with(
            Code::NullResult,
            crate::message!(
                "expression must produce a numeric result, got {0, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}}"
            ),
            [Arg::from("nil")],
        )),
        (_, other) => Err(EvalError::with(
            Code::NonNumericResult,
            crate::message!(
                "expression must produce a numeric result, got {0, select, nil {an empty value} boolean {true/false} text {text} number {a number} date {a date} duration {a duration} list {a list} map {a group of fields} other {a value}}"
            ),
            [Arg::from(other.kind_key())],
        )),
    }
}

fn error_output(error: &EvalError, index: &Utf16Index) -> ErrorOutput {
    let range = error.span.map(|span| index.range(span));
    ErrorOutput {
        code: error.code.as_str(),
        message: error.message.message,
        args: error.message.args.clone(),
        from: range.map(|range| range.from),
        to: range.map(|range| range.to),
        fields: error.fields.clone(),
        guards: error.guards.clone(),
    }
}

fn lookup_output(record: &LookupRecord, index: &Utf16Index) -> LookupOutput {
    LookupOutput {
        function: record.function,
        table: record.table.clone(),
        keys: record.keys.iter().map(trace_value).collect(),
        value: record.value.map(format_number),
        matched: record.matched.as_ref().map(match_output),
        error: record
            .error
            .as_ref()
            .map(|error| error_output(error, index)),
    }
}

fn match_output(matched: &Matched) -> MatchOutput {
    MatchOutput {
        matched_key: matched.key.clone(),
        band_min: matched.band_min.map(format_number),
        band_max: matched.band_max.map(format_number),
        adjusted: matched.adjusted,
    }
}
