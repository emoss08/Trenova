//! The `tariff.*` operations this area answers: `tariff.evaluate`, `tariff.test` and
//! `tariff.scaffoldTest`.
//!
//! `tariff.scaffoldTest` is described in [`super::scaffold`].
//!
//! Every operation checks its file first ([`gate`]): a file that does not load or does not
//! check never runs, and its error (code `check`) is the first finding. A file that checks runs
//! with the checker's types and its declared input types.
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
//!                       "limit": null | {"kind": "minimum", "computed": …}, "on": null,
//!                       "range": {"from": 0, "to": 40}}],
//!            "notes": [{"text": "…", "range": …}], "reviews": [{"text": "…", "range": …}],
//!            "freightTotal": …, "accessorialTotal": …,
//!            "total": …, "converted": {"currency": "CAD", "rate": "1.3502", "lines": […],
//!            "total": …}},
//!   "value": true,
//!   "refusal": {"message": "No fuel price on …" | {"message": "…", "args": […]}, "range": …},
//!   "error": {"code": "divideByZero", "message": {"message": "…", "args": […]}, "range": …},
//!   "needs": [{"table": "Zones", "keys": ["752"]}],
//!   "choices": [{"label": "Linehaul", "kind": "best", "range": …, "chosen": "Flat",
//!                "options": [{"label": "Per mile", "range": …, "status": "lost",
//!                             "subtotal": {"$q": "1175", "u": "$"}, "refusal": null}, …]}],
//!   "trace": [… ] | null, "lookups": […]}]}
//! ```
//!
//! `bill` is present for a rated tariff, `value` for a condition, `refusal`, `error` and
//! `needs` for those statuses. A line's `limit` is set when its `minimum:` or `maximum:` changed
//! its amount (`computed` is the amount it replaced, rounded like the line); `on` is `freight`
//! for the adjustment of a total clause `on freight`. `reviews` are the `review` reasons, shaped
//! like `notes`: a bill with any is held for approval. `choices` is every `best` and `worst` the run evaluated, in the
//! order their evaluation began, at most 200: each option's `status` is `chosen`, `lost` or
//! `refused`, with its exact `subtotal` or its `refusal`; `chosen` is `null` when every option
//! refused. A result that never ran, and a condition, has `"choices": []`. A refusal's `message` is the author's text, or a `Message` when
//! the crate wrote it (an input with no value).
//!
//! `tariff.test` runs a file's tests and properties, in source order: `{"source", "context",
//! "names": […] (optional), "budget" (optional)}` → `{"tests": [{"name", "range", "kind":
//! "test" | "property", "status": "passed" | "failed" | "error" | "needs", "failures":
//! [{"range", "message", "values": [{"range", "display", "value"?}]}], "bill": … | null,
//! "error": … | null, "cases": 24, "skipped": 0, "counterexample": … | null}], "needs":
//! [{"table", "keys"}], "coverage": {"regions": [{"range", "kind", "hits"}], "reached": 18,
//! "total": 21} | null}`, plus `"error"` at the top (and `"coverage": null`) when the file does
//! not check. A test has `cases` 1, `skipped` 0 and `counterexample` `null`; a property's
//! `counterexample` is [`super::property::Counterexample`]'s wire form. `needs` is every
//! test's needs, each once: tests read tables under the on-demand protocol `tariff.evaluate`
//! uses, and a test that needed cells the request did not carry reports `needs`; the host loads
//! them and runs the tests again. `coverage` ([`super::coverage`]) is `null` when `names` ran
//! only some tests or a test ended `needs`.

use super::bill::{Choices, Converted, Finished, Line, OptionOutcome};
use super::budget::{BudgetWire, Limits};
use super::error::{Code, EvalError, Refusal, RefusalText};
use super::record::{decode_record, sample};
use super::run::{Outcome, Request, run_program};
use super::tables::{Need, Tables};
use super::tests::{TestRequest, TestResult, needs_of, run_suite};
use super::trace::range;
use super::value::{Value, format_decimal};
use trenova_module_kit::span::Utf16Index;
use crate::tariff::check::{Analysis, check};
use crate::tariff::context::Context;
use trenova_module_kit::diagnostic::{Diagnostic, Severity};
use crate::tariff::schema::Schema;
use crate::tariff::syntax::ast::FileKind;
use crate::tariff::syntax::program::{ModuleId, Program, load};
use crate::tariff::wire::encode as wire;
use crate::wire::Dec;
use serde::Deserialize;
use serde_json::value::RawValue;
use serde_json::{Map, Value as Json};
use std::collections::BTreeMap;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::ops::{OpResult, decode, encode};

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
    let mut loaded: Vec<Loaded> = Vec::new();
    let mut results = Vec::with_capacity(input.programs.len());
    for program in &input.programs {
        let found = loaded
            .iter()
            .position(|entry| entry.source == program.source && entry.kind == program.kind);
        let index = if let Some(index) = found {
            index
        } else {
            let built = load(
                &program.source,
                program.kind.map(Kind::file),
                &input.context,
            );
            let checked = gate(&built, &input.context);
            loaded.push(Loaded {
                source: program.source.clone(),
                kind: program.kind,
                program: built,
                checked,
            });
            loaded.len() - 1
        };
        let Some(entry) = loaded.get(index) else {
            continue;
        };
        let record = match &program.record {
            Some(json) => decode_record(json, &schema).map_err(|error| record_error(&error)),
            None => shared.clone(),
        };
        results.push(evaluate_one(&Evaluation {
            program,
            loaded: &entry.program,
            checked: &entry.checked,
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
        trenova_module_kit::message!("The record does not fit the schema at {0}: {1}"),
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
    Some(finding_error(*module, diagnostic))
}

/// Checks a loaded program before it runs: a file that does not load or does not check never
/// runs, and one that does runs with the checker's types. The error is the first finding.
pub fn gate(program: &Program, context: &Context) -> Result<Analysis, EvalError> {
    if let Some(error) = load_error(program) {
        return Err(error);
    }
    let analysis = check(program, context);
    if let Some((module, diagnostic)) = analysis
        .diagnostics()
        .find(|(_, diagnostic)| diagnostic.severity == Severity::Error)
    {
        return Err(finding_error(module, diagnostic));
    }
    Ok(analysis)
}

fn finding_error(module: ModuleId, diagnostic: &Diagnostic) -> EvalError {
    let error = EvalError::new(Code::Check, diagnostic.message.clone());
    if module.0 == 0 {
        error.at(diagnostic.span)
    } else {
        error
    }
}

struct Loaded {
    source: String,
    kind: Option<Kind>,
    program: Program,
    checked: Result<Analysis, EvalError>,
}

struct Evaluation<'e> {
    program: &'e ProgramInput,
    loaded: &'e Program,
    checked: &'e Result<Analysis, EvalError>,
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
    let ready = match (evaluation.checked, &evaluation.record) {
        (Err(error), _) | (Ok(_), Err(error)) => Err(error.clone()),
        (Ok(analysis), Ok(record)) => Ok((analysis, record.clone())),
    };
    let (analysis, record) = match ready {
        Ok(ready) => ready,
        Err(error) => {
            status(&mut result, "error");
            result.insert(String::from("error"), error_json(&error, &index));
            result.insert(String::from("choices"), Json::Array(Vec::new()));
            result.insert(String::from("trace"), Json::Null);
            result.insert(String::from("lookups"), Json::Array(Vec::new()));
            return Json::Object(result);
        }
    };
    let run = run_program(Request {
        program: evaluation.loaded,
        types: analysis,
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
    result.insert(String::from("choices"), choices_json(&run.choices, &index));
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

/// Table needs on the wire: `[{"table", "keys"}]`.
#[must_use]
pub fn needs_json(needs: &[Need]) -> Json {
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
            "limit",
            line.limit
                .as_ref()
                .map_or(Json::Null, super::trace::limit_json),
        ),
        (
            "on",
            if line.on_freight {
                Json::String(String::from("freight"))
            } else {
                Json::Null
            },
        ),
        (
            "range",
            line.span.map_or(Json::Null, |span| range(span, index)),
        ),
    ])
}

/// Notes or review reasons on the wire: `[{"text", "range"}]`.
fn notes_json(notes: &[super::bill::Note], index: &Utf16Index) -> Json {
    Json::Array(
        notes
            .iter()
            .map(|note| {
                object(vec![
                    ("text", Json::String(note.text.clone())),
                    ("range", range(note.span, index)),
                ])
            })
            .collect(),
    )
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
        ("notes", notes_json(&bill.notes, index)),
        ("reviews", notes_json(&bill.reviews, index)),
        ("freightTotal", money(bill.freight_total)),
        ("accessorialTotal", money(bill.accessorial_total)),
        ("total", money(bill.total)),
    ])
}

/// The choices a run evaluated on the wire: `[{"label", "kind", "range", "chosen",
/// "options": [{"label", "range", "status", "subtotal", "refusal"}]}]`.
#[must_use]
pub fn choices_json(choices: &Choices, index: &Utf16Index) -> Json {
    Json::Array(
        choices
            .records()
            .map(|choice| {
                let options = choice
                    .options
                    .iter()
                    .enumerate()
                    .map(|(at, option)| {
                        let (status, subtotal, refusal) = match &option.outcome {
                            OptionOutcome::Billed(amount) => (
                                if choice.chosen == Some(at) {
                                    "chosen"
                                } else {
                                    "lost"
                                },
                                money(*amount),
                                Json::Null,
                            ),
                            OptionOutcome::Refused(refusal) => {
                                ("refused", Json::Null, refusal_json(refusal, index))
                            }
                        };
                        object(vec![
                            ("label", Json::String(option.label.clone())),
                            ("range", range(option.span, index)),
                            ("status", Json::String(status.to_owned())),
                            ("subtotal", subtotal),
                            ("refusal", refusal),
                        ])
                    })
                    .collect();
                object(vec![
                    ("label", Json::String(choice.label.clone())),
                    ("kind", Json::String(choice.kind.keyword().to_owned())),
                    ("range", range(choice.span, index)),
                    (
                        "chosen",
                        choice
                            .chosen_label()
                            .map_or(Json::Null, |label| Json::String(label.to_owned())),
                    ),
                    ("options", Json::Array(options)),
                ])
            })
            .collect(),
    )
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
    let analysis = match gate(&program, &input.context) {
        Ok(analysis) => analysis,
        Err(error) => {
            output.insert(String::from("tests"), Json::Array(Vec::new()));
            output.insert(String::from("needs"), Json::Array(Vec::new()));
            output.insert(String::from("coverage"), Json::Null);
            output.insert(String::from("error"), error_json(&error, &index));
            return encode(&Json::Object(output));
        }
    };
    let tables = Tables::build(&input.context.tables);
    let suite = run_suite(&TestRequest {
        program: &program,
        types: &analysis,
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
            suite
                .results
                .iter()
                .map(|result| test_json(result, &index))
                .collect(),
        ),
    );
    output.insert(String::from("needs"), needs_json(&needs_of(&suite.results)));
    output.insert(
        String::from("coverage"),
        suite
            .coverage
            .as_ref()
            .map_or(Json::Null, |hits| hits.to_json(&index)),
    );
    encode(&Json::Object(output))
}

fn test_json(result: &TestResult, index: &Utf16Index) -> Json {
    object(vec![
        ("name", Json::String(result.name.clone())),
        ("range", range(result.span, index)),
        ("kind", Json::String(result.kind.as_str().to_owned())),
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
                                        .map(|(span, display, value)| {
                                            let mut fields = vec![
                                                ("range", range(*span, index)),
                                                ("display", Json::String(display.clone())),
                                            ];
                                            if let Some(value) = value {
                                                fields.push(("value", value.clone()));
                                            }
                                            object(fields)
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
        ("cases", Json::from(result.cases)),
        ("skipped", Json::from(result.skipped)),
        (
            "counterexample",
            result
                .counterexample
                .as_ref()
                .map_or(Json::Null, |counterexample| counterexample.to_json(index)),
        ),
    ])
}

/// `tariff.scaffoldTest`: writes a `test` block from a real shipment's facts.
///
/// # Errors
///
/// When the input does not have the operation's shape.
pub fn scaffold_test(input: &RawValue) -> OpResult {
    super::scaffold::op(input)
}
