//! The `tariff.*` operations this area answers: `tariff.evaluate` and `tariff.test`.
//!
//! `tariff.evaluate` rates a batch in one call. Request:
//!
//! ```json
//! {"context": Context, "record": {…} | null, "budget": {"steps": 5000} | null, "trace": false,
//!  "programs": [{"id": "rule-1", "source": "tariff \"…\" …", "kind": "tariff",
//!                "inputs": {"rate": {"$q": "2.35", "u": "$/mi"}}, "record": {…},
//!                "exchange": {"rate": "1.3502", "currency": "CAD"}}]}
//! ```
//!
//! `record` `null` (or absent) rates against the sample record; a program's own `record`
//! overrides the shared one. Programs with the same source and kind share one load. Each
//! program is independent: one program's error never affects another's result. Response:
//!
//! ```json
//! {"results": [{"id": "rule-1", "status": "rated" | "refused" | "error" | "needs",
//!   "bill": {"lines": [{"kind": "charge", "label": "Linehaul", "code": null, "freight": true,
//!                       "amount": {"$q": "1908.2", "u": "$"}, "quantity": …, "rate": …,
//!                       "range": {"from": 0, "to": 40}}],
//!            "notes": [{"text": "…", "range": …}], "freightTotal": …, "accessorialTotal": …,
//!            "total": …, "converted": {"currency": "CAD", "rate": "1.3502", "lines": […],
//!            "total": …}},
//!   "value": true,
//!   "refusal": {"message": "No fuel price on …" | {"message": "…", "args": […]}, "range": …},
//!   "error": {"code": "divideByZero", "message": {"message": "…", "args": […]}, "range": …},
//!   "needs": [{"table": "Zones", "keys": ["752"]}],
//!   "trace": [… ] | null, "lookups": […]}]}
//! ```
//!
//! `bill` is present for a rated tariff, `value` for a condition, `refusal`, `error` and
//! `needs` for those statuses. A refusal's `message` is the author's text, or a `Message` when
//! the crate wrote it (an input with no value).
//!
//! `tariff.test` runs a file's tests: `{"source", "context", "names": […] (optional), "budget"
//! (optional)}` → `{"tests": [{"name", "range", "status": "passed" | "failed" | "error",
//! "failures": [{"range", "message", "values": [{"range", "display"}]}], "bill": … | null,
//! "error": … | null}]}`, plus `"error"` at the top when the file does not load.

use super::NoTypes;
use super::bill::{Converted, Finished, Line};
use super::budget::{BudgetWire, Limits};
use super::error::{Code, EvalError, Refusal, RefusalText};
use super::record::{decode_record, sample};
use super::run::{Outcome, Request, run_program};
use super::tables::{Need, Tables};
use super::tests::{TestRequest, TestResult, run_tests};
use super::trace::range;
use super::value::{Value, format_decimal};
use crate::message::{Arg, Message};
use crate::ops::{OpResult, decode, encode};
use crate::span::Utf16Index;
use crate::tariff::context::Context;
use crate::tariff::diagnostic::Severity;
use crate::tariff::schema::Schema;
use crate::tariff::syntax::ast::FileKind;
use crate::tariff::syntax::program::{Program, load};
use crate::tariff::wire::encode as wire;
use crate::wire::Dec;
use serde::Deserialize;
use serde_json::value::RawValue;
use serde_json::{Map, Value as Json};
use std::collections::BTreeMap;

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct EvaluateInput {
    #[serde(default)]
    context: Context,
    #[serde(default)]
    record: Option<Json>,
    programs: Vec<ProgramInput>,
    #[serde(default)]
    budget: Option<BudgetWire>,
    #[serde(default)]
    trace: bool,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct ProgramInput {
    id: String,
    source: String,
    #[serde(default)]
    kind: Option<Kind>,
    #[serde(default)]
    inputs: BTreeMap<String, Json>,
    #[serde(default)]
    record: Option<Json>,
    #[serde(default)]
    exchange: Option<Exchange>,
}

#[derive(Deserialize, Clone, Copy, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
enum Kind {
    Tariff,
    Condition,
}

impl Kind {
    const fn file(self) -> FileKind {
        match self {
            Self::Tariff => FileKind::Tariff,
            Self::Condition => FileKind::Condition,
        }
    }
}

#[derive(Deserialize, Clone)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct Exchange {
    rate: Dec,
    currency: String,
}

/// `tariff.evaluate`.
///
/// # Errors
///
/// When the input does not have the operation's shape.
pub fn evaluate(input: &RawValue) -> OpResult {
    let input: EvaluateInput = decode(input)?;
    let schema = Schema::new(&input.context.settings);
    let tables = Tables::build(&input.context.tables);
    let limits = input
        .budget
        .map_or(Limits::DEFAULT, |budget| Limits::DEFAULT.lowered(&budget));
    let shared = shared_record(input.record.as_ref(), &input.context, &schema);
    let mut loaded: Vec<(String, Option<Kind>, Program)> = Vec::new();
    let mut results = Vec::with_capacity(input.programs.len());
    for program in &input.programs {
        let found = loaded
            .iter()
            .position(|(source, kind, _)| *source == program.source && *kind == program.kind);
        let index = if let Some(index) = found {
            index
        } else {
            let built = load(
                &program.source,
                program.kind.map(Kind::file),
                &input.context,
            );
            loaded.push((program.source.clone(), program.kind, built));
            loaded.len() - 1
        };
        let Some((_, _, built)) = loaded.get(index) else {
            continue;
        };
        let record = match &program.record {
            Some(json) => decode_record(json, &schema).map_err(|error| record_error(&error)),
            None => shared.clone(),
        };
        results.push(evaluate_one(&Evaluation {
            program,
            loaded: built,
            schema: &schema,
            tables: &tables,
            record,
            limits,
            trace: input.trace,
        }));
    }
    let mut output = Map::new();
    output.insert(String::from("results"), Json::Array(results));
    encode(&Json::Object(output))
}

fn shared_record(
    json: Option<&Json>,
    context: &Context,
    schema: &Schema,
) -> Result<Value, EvalError> {
    match json {
        None | Some(Json::Null) => Ok(sample(&context.settings)),
        Some(json) => decode_record(json, schema).map_err(|error| record_error(&error)),
    }
}

fn record_error(error: &crate::tariff::wire::WireError) -> EvalError {
    EvalError::with(
        Code::Record,
        crate::message!("The record does not fit the schema at {0}: {1}"),
        [
            Arg::from(error.path.clone()),
            Arg::from(error.message.clone()),
        ],
    )
}

/// The first error-severity finding of the loader, as the error a program that does not load
/// gives.
pub fn load_error(program: &Program) -> Option<EvalError> {
    let (module, diagnostic) = program
        .diagnostics
        .iter()
        .find(|(_, diagnostic)| diagnostic.severity == Severity::Error)?;
    let error = EvalError::new(Code::Syntax, diagnostic.message.clone());
    Some(if module.0 == 0 {
        error.at(diagnostic.span)
    } else {
        error
    })
}

struct Evaluation<'e> {
    program: &'e ProgramInput,
    loaded: &'e Program,
    schema: &'e Schema,
    tables: &'e Tables,
    record: Result<Value, EvalError>,
    limits: Limits,
    trace: bool,
}

fn evaluate_one(evaluation: &Evaluation<'_>) -> Json {
    let index = evaluation
        .loaded
        .root()
        .map_or_else(|| Utf16Index::new(""), |root| root.utf16.clone());
    let mut result = Map::new();
    result.insert(
        String::from("id"),
        Json::String(evaluation.program.id.clone()),
    );
    let ready = match (load_error(evaluation.loaded), &evaluation.record) {
        (Some(error), _) => Err(error),
        (None, Err(error)) => Err(error.clone()),
        (None, Ok(record)) => Ok(record.clone()),
    };
    let record = match ready {
        Ok(record) => record,
        Err(error) => {
            status(&mut result, "error");
            result.insert(String::from("error"), error_json(&error, &index));
            result.insert(String::from("trace"), Json::Null);
            result.insert(String::from("lookups"), Json::Array(Vec::new()));
            return Json::Object(result);
        }
    };
    let run = run_program(Request {
        program: evaluation.loaded,
        types: &NoTypes,
        schema: evaluation.schema,
        tables: evaluation.tables,
        record,
        inputs: &evaluation.program.inputs,
        limits: evaluation.limits,
        trace: evaluation.trace,
    });
    match &run.outcome {
        Outcome::Rated { bill, clauses } => {
            status(&mut result, "rated");
            let converted = evaluation
                .program
                .exchange
                .as_ref()
                .map(|exchange| (exchange, bill.convert(exchange.rate.get(), clauses)));
            match converted {
                Some((_, Err(error))) => {
                    status(&mut result, "error");
                    result.insert(String::from("error"), error_json(&error, &index));
                }
                Some((exchange, Ok(converted))) => {
                    let mut encoded = bill_json(bill, &index);
                    if let Json::Object(object) = &mut encoded {
                        object.insert(
                            String::from("converted"),
                            converted_json(exchange, &converted),
                        );
                    }
                    result.insert(String::from("bill"), encoded);
                }
                None => {
                    result.insert(String::from("bill"), bill_json(bill, &index));
                }
            }
        }
        Outcome::Condition(holds) => {
            status(&mut result, "rated");
            result.insert(String::from("value"), Json::Bool(*holds));
        }
        Outcome::Refused(refusal) => {
            status(&mut result, "refused");
            result.insert(String::from("refusal"), refusal_json(refusal, &index));
        }
        Outcome::Failed(error) => {
            status(&mut result, "error");
            result.insert(String::from("error"), error_json(error, &index));
        }
        Outcome::Needs(needs) => {
            status(&mut result, "needs");
            result.insert(String::from("needs"), needs_json(needs));
        }
    }
    result.insert(String::from("trace"), run.trace.encode(&index));
    result.insert(String::from("lookups"), run.trace.encode_lookups(&index));
    Json::Object(result)
}

fn status(result: &mut Map<String, Json>, name: &str) {
    result.insert(String::from("status"), Json::String(name.to_owned()));
}

fn object(fields: Vec<(&str, Json)>) -> Json {
    let mut map = Map::new();
    for (name, value) in fields {
        map.insert(name.to_owned(), value);
    }
    Json::Object(map)
}

fn money(amount: rust_decimal::Decimal) -> Json {
    wire(&Value::money(amount))
}

fn message_json(message: &Message) -> Json {
    serde_json::to_value(message).unwrap_or(Json::Null)
}

/// An evaluation error on the wire.
#[must_use]
pub fn error_json(error: &EvalError, index: &Utf16Index) -> Json {
    object(vec![
        ("code", Json::String(error.code.as_str().to_owned())),
        ("message", message_json(&error.message)),
        (
            "range",
            error.span.map_or(Json::Null, |span| range(span, index)),
        ),
    ])
}

/// A refusal on the wire.
#[must_use]
pub fn refusal_json(refusal: &Refusal, index: &Utf16Index) -> Json {
    object(vec![
        (
            "message",
            match &refusal.message {
                RefusalText::Author(text) => Json::String(text.clone()),
                RefusalText::Crate(message) => message_json(message),
            },
        ),
        ("range", range(refusal.span, index)),
    ])
}

fn needs_json(needs: &[Need]) -> Json {
    Json::Array(
        needs
            .iter()
            .map(|need| {
                object(vec![
                    ("table", Json::String(need.table.clone())),
                    (
                        "keys",
                        Json::Array(need.keys.iter().cloned().map(Json::String).collect()),
                    ),
                ])
            })
            .collect(),
    )
}

fn line_json(line: &Line, index: &Utf16Index) -> Json {
    let optional = |value: Option<&Value>| value.map_or(Json::Null, wire);
    object(vec![
        ("kind", Json::String(line.kind.as_str().to_owned())),
        ("label", line.label.clone().map_or(Json::Null, Json::String)),
        ("code", line.code.clone().map_or(Json::Null, Json::String)),
        ("freight", Json::Bool(line.freight())),
        ("amount", money(line.amount)),
        ("quantity", optional(line.quantity.as_ref())),
        ("rate", optional(line.rate.as_ref())),
        (
            "range",
            line.span.map_or(Json::Null, |span| range(span, index)),
        ),
    ])
}

/// A finished bill on the wire.
#[must_use]
pub fn bill_json(bill: &Finished, index: &Utf16Index) -> Json {
    object(vec![
        (
            "lines",
            Json::Array(
                bill.lines
                    .iter()
                    .map(|line| line_json(line, index))
                    .collect(),
            ),
        ),
        (
            "notes",
            Json::Array(
                bill.notes
                    .iter()
                    .map(|note| {
                        object(vec![
                            ("text", Json::String(note.text.clone())),
                            ("range", range(note.span, index)),
                        ])
                    })
                    .collect(),
            ),
        ),
        ("freightTotal", money(bill.freight_total)),
        ("accessorialTotal", money(bill.accessorial_total)),
        ("total", money(bill.total)),
    ])
}

fn converted_json(exchange: &Exchange, converted: &Converted) -> Json {
    object(vec![
        ("currency", Json::String(exchange.currency.clone())),
        ("rate", Json::String(format_decimal(exchange.rate.get()))),
        (
            "lines",
            Json::Array(
                converted
                    .lines
                    .iter()
                    .map(|amount| money(*amount))
                    .collect(),
            ),
        ),
        ("total", money(converted.total)),
    ])
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct TestInput {
    source: String,
    #[serde(default)]
    context: Context,
    #[serde(default)]
    names: Option<Vec<String>>,
    #[serde(default)]
    budget: Option<BudgetWire>,
}

/// `tariff.test`.
///
/// # Errors
///
/// When the input does not have the operation's shape.
pub fn test(input: &RawValue) -> OpResult {
    let input: TestInput = decode(input)?;
    let program = load(&input.source, None, &input.context);
    let index = program
        .root()
        .map_or_else(|| Utf16Index::new(""), |root| root.utf16.clone());
    let mut output = Map::new();
    if let Some(error) = load_error(&program) {
        output.insert(String::from("tests"), Json::Array(Vec::new()));
        output.insert(String::from("error"), error_json(&error, &index));
        return encode(&Json::Object(output));
    }
    let schema = Schema::new(&input.context.settings);
    let tables = Tables::build(&input.context.tables);
    let results = run_tests(&TestRequest {
        program: &program,
        types: &NoTypes,
        schema: &schema,
        settings: &input.context.settings,
        tables: &tables,
        limits: input
            .budget
            .map_or(Limits::DEFAULT, |budget| Limits::DEFAULT.lowered(&budget)),
        names: input.names.as_deref(),
    });
    output.insert(
        String::from("tests"),
        Json::Array(
            results
                .iter()
                .map(|result| test_json(result, &index))
                .collect(),
        ),
    );
    encode(&Json::Object(output))
}

fn test_json(result: &TestResult, index: &Utf16Index) -> Json {
    object(vec![
        ("name", Json::String(result.name.clone())),
        ("range", range(result.span, index)),
        ("status", Json::String(result.status.as_str().to_owned())),
        (
            "failures",
            Json::Array(
                result
                    .failures
                    .iter()
                    .map(|failure| {
                        object(vec![
                            ("range", range(failure.span, index)),
                            ("message", message_json(&failure.message)),
                            (
                                "values",
                                Json::Array(
                                    failure
                                        .values
                                        .iter()
                                        .map(|(span, display)| {
                                            object(vec![
                                                ("range", range(*span, index)),
                                                ("display", Json::String(display.clone())),
                                            ])
                                        })
                                        .collect(),
                                ),
                            ),
                        ])
                    })
                    .collect(),
            ),
        ),
        (
            "bill",
            result
                .bill
                .as_ref()
                .map_or(Json::Null, |bill| bill_json(bill, index)),
        ),
        (
            "error",
            result
                .error
                .as_ref()
                .map_or(Json::Null, |error| error_json(error, index)),
        ),
    ])
}
