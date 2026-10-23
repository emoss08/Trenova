//! The evaluator's defensive paths, run without the checker.
//!
//! `tariff.evaluate` and `tariff.test` check a file before it runs, so a program that adds
//! money to a weight or reads a name nothing declares never reaches the interpreter through
//! an operation. The interpreter still never guesses: given such a program directly, with no
//! static types ([`NoTypes`]), it stops with the `internal` code (or the run-time code the
//! failure has, such as `budget` for a recursion the checker would refuse). These tests call
//! [`run_program`] and [`run_tests`] directly, skipping the gate, to keep that so.

use trenova_rules::tariff::context::Context;
use trenova_rules::tariff::eval::NoTypes;
use trenova_rules::tariff::eval::budget::Limits;
use trenova_rules::tariff::eval::error::Code;
use trenova_rules::tariff::eval::record::sample;
use trenova_rules::tariff::eval::run::{Outcome, Request, run_program};
use trenova_rules::tariff::eval::tables::Tables;
use trenova_rules::tariff::eval::tests::{Status, TestRequest, run_tests};
use trenova_rules::tariff::syntax::ast::FileKind;
use trenova_rules::tariff::syntax::program::load;

const ZONES: &str = r#"{"tables": [{"code": "Zones", "valueType": "Money/cwt",
    "axes": [{"match": "Exact", "kind": "Zone", "normalization": "Upper"}],
    "cells": [{"at": [{"key": "ZONE_1"}], "value": "18.50"}]}]}"#;

struct Case {
    name: &'static str,
    body: &'static str,
    context: &'static str,
    inputs: &'static str,
    code: Code,
}

const fn case(name: &'static str, body: &'static str, code: Code) -> Case {
    Case {
        name,
        body,
        context: "{}",
        inputs: "{}",
        code,
    }
}

const CASES: &[Case] = &[
    case(
        "a charge of a weight",
        "charge \"Weight\" = 5 lb",
        Code::Internal,
    ),
    case(
        "recursion is stopped by the call depth",
        "fn loop(x: Int) -> Int = loop(x + 1)\nnote \"{loop(0)}\"",
        Code::Budget,
    ),
    case(
        "a self-referential constant",
        "const LOOPED = LOOPED + 1\nnote \"{LOOPED}\"",
        Code::Internal,
    ),
    case("an unknown name", "note \"{nothing}\"", Code::Internal),
    case(
        "calling something that is not a function",
        "note \"{nothing(1)}\"",
        Code::Internal,
    ),
    case(
        "a function that ends without return",
        "fn f(x: Int) -> Int {\n  let y = x\n}\nnote \"{f(1)}\"",
        Code::Internal,
    ),
    case(
        "a missing field",
        "note \"{shipment.nothing}\"",
        Code::Internal,
    ),
    case(
        "reading a field of none without ?.",
        "note \"{shipment.trailerType.code}\"",
        Code::Internal,
    ),
    case(
        "a list method's test that is not a bool",
        "let xs = [3, 1, 4]\nnote \"{xs.where(x => x)}\"",
        Code::Internal,
    ),
    case(
        "an unknown list method",
        "let xs = [3, 1, 4]\nnote \"{xs.shuffle()}\"",
        Code::Internal,
    ),
    case(
        "in with the wrong dimension",
        "note \"{5 lb.in(mi)}\"",
        Code::Conversion,
    ),
    case(
        "temperatures do not multiply",
        "note \"{40 °F * 2}\"",
        Code::Conversion,
    ),
    case(
        "adding money to a weight",
        "note \"{$1 + 1 lb}\"",
        Code::Internal,
    ),
    case("adding text", "note \"{\"a\" + \"b\"}\"", Code::Internal),
    case(
        "ordering text",
        "note \"{\"apple\" < \"banana\"}\"",
        Code::Internal,
    ),
    case(
        "no arm covering a value",
        "let n = 9\nlet label = match n {\n  1 => \"one\"\n}\nnote \"{label}\"",
        Code::Internal,
    ),
    case("assigning a let", "let x = 1\nx = 2", Code::Internal),
    case(
        "a function reading the record",
        "fn peek() -> Text = shipment.proNumber\nnote \"{peek()}\"",
        Code::Internal,
    ),
    case(
        "a function reading a file-level let",
        "let secret = 1\nfn peek() -> Int = secret\nnote \"{peek()}\"",
        Code::Internal,
    ),
    Case {
        name: "band on an exact table",
        body: "note \"{tables.Zones.band(\"A\")}\"",
        context: ZONES,
        inputs: "{}",
        code: Code::Internal,
    },
    Case {
        name: "a table the request did not send",
        body: "note \"{tables.Missing[\"A\"]}\"",
        context: "{}",
        inputs: "{}",
        code: Code::Table,
    },
    Case {
        name: "an input value with no checked type to read it by",
        body: "input rate: Money = $1\ncharge \"A\" = rate",
        context: "{}",
        inputs: r#"{"rate": {"$q": "2", "u": "$"}}"#,
        code: Code::Internal,
    },
];

fn context(json: &str) -> Context {
    serde_json::from_str(json).unwrap_or_else(|error| panic!("a context: {error}"))
}

fn outcome(case: &Case) -> Outcome {
    let context = context(case.context);
    let source = format!("tariff \"unchecked\"\n{}\n", case.body);
    let program = load(&source, Some(FileKind::Tariff), &context);
    assert!(
        program.diagnostics.is_empty(),
        "{}: the source does not load: {:?}",
        case.name,
        program.diagnostics
    );
    let tables = Tables::build(&context.tables);
    let inputs = serde_json::from_str(case.inputs).unwrap_or_default();
    run_program(Request {
        program: &program,
        types: &NoTypes,
        tables: &tables,
        record: sample(&context.settings),
        inputs: &inputs,
        limits: Limits::DEFAULT,
        trace: false,
    })
    .outcome
}

#[test]
fn what_the_checker_rules_out_stops_the_evaluator_with_its_code() {
    let mut problems = Vec::new();
    for case in CASES {
        match outcome(case) {
            Outcome::Failed(error) if error.code == case.code => {}
            other => problems.push(format!(
                "{}: wanted {}, got {other:?}",
                case.name,
                case.code.as_str()
            )),
        }
    }
    assert!(problems.is_empty(), "{}", problems.join("\n"));
}

#[test]
fn an_internal_error_names_the_kinds_of_value() {
    let case = case("true plus one", "let a = true + 1", Code::Internal);
    let Outcome::Failed(error) = outcome(&case) else {
        panic!("the program did not fail");
    };
    assert_eq!(error.code, Code::Internal);
    let wire = serde_json::to_value(&error.message).unwrap_or_default();
    assert_eq!(
        wire,
        serde_json::json!({
            "message": "{0} cannot be applied to {1} and {2}",
            "args": ["+", {"message": "true or false"}, {"message": "a whole number"}],
        })
    );
}

#[test]
fn a_given_to_a_field_that_does_not_exist_stops_the_test() {
    let context = Context::default();
    let source = "tariff \"s\"\ncharge \"x\" = $1\ntest \"bad\" {\n  given shipment.nothing = 1\n  expect true\n}\n";
    let program = load(source, Some(FileKind::Tariff), &context);
    let tables = Tables::build(&context.tables);
    let results = run_tests(&TestRequest {
        program: &program,
        types: &NoTypes,
        settings: &context.settings,
        tables: &tables,
        limits: Limits::DEFAULT,
        names: None,
    });
    let [result] = results.as_slice() else {
        panic!("wanted one test, got {results:?}");
    };
    assert_eq!(result.status, Status::Error);
    assert_eq!(
        result.error.as_ref().map(|error| error.code),
        Some(Code::Internal)
    );
}
