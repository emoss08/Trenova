//! The evaluator against the shared conformance corpus (`testdata/conformance.json`), the
//! function catalog, and the WebAssembly stack.

use serde::Deserialize;
use serde_json::Value as Json;
use serde_json::json;
use serde_json::value::RawValue;
use std::collections::BTreeMap;
use trenova_rules::formula::catalog;
use trenova_rules::formula::eval::Limits;
use trenova_rules::formula::eval::interp;
use trenova_rules::formula::eval::interp::LookupSource;
use trenova_rules::formula::eval::interp::Setup;
use trenova_rules::formula::eval::parse_value;
use trenova_rules::formula::syntax;
use trenova_rules::ops::dispatch;

#[derive(Deserialize)]
struct Conformance {
    cases: Vec<ConformanceCase>,
}

#[derive(Deserialize)]
struct ConformanceCase {
    id: String,
    expression: String,
    accepted: bool,
    #[serde(default)]
    value: Option<String>,
}

#[derive(Deserialize)]
struct Evaluation {
    environments: BTreeMap<String, Box<RawValue>>,
}

fn conformance_env() -> Json {
    let corpus: Evaluation =
        serde_json::from_str(include_str!("../testdata/evaluation.json")).expect("corpus");
    let raw = corpus
        .environments
        .get("conformance")
        .expect("the conformance environment");
    serde_json::from_str(raw.get()).expect("environment JSON")
}

/// Every `value` the conformance corpus pins, on an environment equivalent to Go's
/// `conformanceShipment()` plus the corpus inputs, through `formula.evaluate`.
#[test]
fn every_conformance_value_is_what_the_engine_comes_to() {
    let corpus: Conformance =
        serde_json::from_str(include_str!("../testdata/conformance.json")).expect("corpus");
    let env = conformance_env();
    let mut checked = 0;
    let mut failures = Vec::new();
    for case in &corpus.cases {
        let Some(want) = &case.value else {
            continue;
        };
        assert!(case.accepted, "{} pins a value but is refused", case.id);
        let request = json!({
            "op": "formula.evaluate",
            "input": {
                "programs": [{"id": case.id, "source": case.expression, "expect": "amount"}],
                "env": env,
            },
        });
        let response: Json =
            serde_json::from_slice(&dispatch(request.to_string().as_bytes())).expect("JSON");
        let result = &response["ok"]["results"][0];
        if result["value"].as_str() != Some(want.as_str()) {
            failures.push(format!(
                "{} {:?}: want {want}, got {result}",
                case.id, case.expression
            ));
        }
        checked += 1;
    }
    assert!(checked >= 38, "only {checked} values checked");
    assert!(failures.is_empty(), "{}", failures.join("\n"));
}

/// The evaluator implements every function the catalog lists, except the disabled ones.
#[test]
fn every_catalog_function_is_implemented() {
    let missing: Vec<&str> = catalog::FUNCTIONS
        .iter()
        .filter(|function| !function.operator)
        .map(|function| function.name)
        .filter(|name| catalog::disabled(name).is_none() && !interp::implements(name))
        .collect();
    assert!(missing.is_empty(), "not implemented: {missing:?}");
    for disabled in catalog::DISABLED {
        assert!(
            !interp::implements(disabled.name),
            "{} is disabled but callable",
            disabled.name
        );
    }
    for removed in ["toBase64", "fromBase64", "timezone"] {
        assert!(
            !interp::implements(removed),
            "{removed} is not part of the language"
        );
    }
    for method in catalog::DATE_METHODS {
        let source = format!("date(\"2026-01-01\").{method}()");
        assert!(evaluates(&source), "{source}");
    }
    for method in catalog::DURATION_METHODS {
        let source = format!("duration(\"1h\").{method}()");
        assert!(evaluates(&source), "{source}");
    }
}

fn evaluates(source: &str) -> bool {
    let program = syntax::parse(source, &[]);
    let env = parse_value("{}").expect("env");
    interp::evaluate(
        &program,
        Setup {
            env: &env,
            lookups: LookupSource::Unavailable,
            limits: Limits::DEFAULT,
            trace_statements: false,
            record_lookups: false,
        },
    )
    .result
    .is_ok()
}

/// The deepest formula the parser accepts evaluates within a 1 MiB stack, the size a
/// WebAssembly module gets, even unoptimised.
#[test]
fn the_deepest_formulas_fit_the_webassembly_stack() {
    let depth = syntax::MAX_DEPTH - 8;
    let formulas = [
        format!("{}1{}", "(".repeat(depth), ")".repeat(depth)),
        format!("{}1{}", "abs(".repeat(depth), ")".repeat(depth)),
        format!(
            "{}#{}",
            "sum(map([1], ".repeat(depth / 3),
            "))".repeat(depth / 3)
        ),
        format!("{}0", "true ? 1 : ".repeat(depth / 2)),
        format!("{}1{}", "-(".repeat(depth / 2), ")".repeat(depth / 2)),
        format!("{}1{}", "[".repeat(depth / 4), "]".repeat(depth / 4)),
        format!("a?.b{}", ".c".repeat(depth)),
        format!("{}1", "1 + ".repeat(depth)),
        format!("{}1", "1 ** ".repeat(depth)),
        format!(
            "{}1{}",
            "(let x = 1; ".repeat(depth / 2),
            ")".repeat(depth / 2)
        ),
    ];
    for formula in formulas {
        let program = syntax::parse(&formula, &[]);
        assert!(
            program.diagnostics.is_empty(),
            "{formula}: {:?}",
            program
                .diagnostics
                .first()
                .map(|diagnostic| diagnostic.message.english())
        );
        let handle = std::thread::Builder::new()
            .stack_size(1 << 20)
            .spawn(move || {
                let env = parse_value(r#"{"a": null}"#).expect("env");
                interp::evaluate(
                    &program,
                    Setup {
                        env: &env,
                        lookups: LookupSource::Unavailable,
                        limits: Limits::DEFAULT,
                        trace_statements: true,
                        record_lookups: false,
                    },
                )
                .result
                .is_ok()
            })
            .expect("a thread");
        let ran = handle.join().expect("no stack overflow");
        assert!(ran, "{formula}");
    }
}
