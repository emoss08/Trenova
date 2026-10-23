//! The evaluator's corpus: every case in `testdata/tariff/eval/*.json`, run through
//! `crate::ops::dispatch`.
//!
//! A file is a JSON array of cases. Each case has a `name` and one of:
//!
//! - `"expr"`: an expression, run as `let value = <expr>` then `note "{value}"` in a tariff
//!   (after an optional `"prelude"` of lines); `"display"` is the note's text, or `"error"` the
//!   error code, or `"refusal"` the refusal's text.
//! - `"source"`: a whole file for `tariff.evaluate`, with optional `"inputs"`, `"record"`,
//!   `"context"`, `"exchange"`, `"budget"`, `"trace"` and `"kind"`; `"expect"` is matched against
//!   the program's result.
//! - `"tests"`: a file for `tariff.test`, with an optional `"context"`, `"names"` and
//!   `"budget"`; `"expect"` is matched against the response's `tests` (or the whole response
//!   when it is an object).
//! - `"op"` and `"input"`: any operation; `"expect"` is matched against the response.
//!
//! A case may also give `"facts"`: a partial record merged onto the sample record (objects
//! merge key by key; anything else replaces), sent as the program's record.
//!
//! Matching is by inclusion: every key an expected object names must match, arrays must have
//! the same length and match item by item, and anything else must be equal.

use serde_json::{Value, json};
use std::fmt::Write as _;
use std::path::Path;

fn call(op: &str, input: &Value) -> Value {
    let request = json!({"op": op, "input": input}).to_string();
    let response = trenova_rules::ops::dispatch(request.as_bytes());
    serde_json::from_slice(&response).unwrap_or(Value::Null)
}

fn includes(actual: &Value, expected: &Value, path: &str, problems: &mut Vec<String>) {
    match (actual, expected) {
        (Value::Object(actual), Value::Object(expected)) => {
            for (key, wanted) in expected {
                let at = format!("{path}.{key}");
                match actual.get(key) {
                    Some(found) => includes(found, wanted, &at, problems),
                    None if wanted.is_null() => {}
                    None => problems.push(format!("{at}: missing, wanted {wanted}")),
                }
            }
        }
        (Value::Array(actual), Value::Array(expected)) => {
            if actual.len() != expected.len() {
                problems.push(format!(
                    "{path}: {} items, wanted {}: {}",
                    actual.len(),
                    expected.len(),
                    Value::Array(actual.clone())
                ));
                return;
            }
            for (index, (found, wanted)) in actual.iter().zip(expected).enumerate() {
                includes(found, wanted, &format!("{path}[{index}]"), problems);
            }
        }
        _ if actual == expected => {}
        _ => problems.push(format!("{path}: got {actual}, wanted {expected}")),
    }
}

fn merge(base: &mut Value, overlay: &Value) {
    match (base, overlay) {
        (Value::Object(base), Value::Object(overlay)) => {
            for (key, value) in overlay {
                match base.get_mut(key) {
                    Some(slot) => merge(slot, value),
                    None => {
                        base.insert(key.clone(), value.clone());
                    }
                }
            }
        }
        (slot, value) => *slot = value.clone(),
    }
}

fn sample() -> Value {
    let settings = trenova_rules::tariff::schema::Settings::default();
    trenova_rules::tariff::wire::encode(&trenova_rules::tariff::eval::record::sample(&settings))
}

fn program(case: &Value, source: &str) -> Value {
    let mut program = json!({"id": "p", "source": source});
    if let Some(facts) = case.get("facts") {
        let mut record = sample();
        merge(&mut record, facts);
        program["record"] = record;
    }
    for key in ["inputs", "record", "exchange", "kind"] {
        if let Some(value) = case.get(key) {
            program[key] = value.clone();
        }
    }
    let mut input = json!({"programs": [program]});
    for key in ["context", "budget", "trace"] {
        if let Some(value) = case.get(key) {
            input[key] = value.clone();
        }
    }
    input
}

fn expression_source(case: &Value, expr: &str) -> String {
    let mut source = String::from("tariff \"corpus\"\n");
    if let Some(prelude) = case.get("prelude").and_then(Value::as_str) {
        source.push_str(prelude);
        source.push('\n');
    }
    let _ = write!(source, "let value = {expr}\nnote \"{{value}}\"\n");
    source
}

fn run(case: &Value) -> Result<(), String> {
    if let Some(expr) = case.get("expr").and_then(Value::as_str) {
        let source = expression_source(case, expr);
        let response = call("tariff.evaluate", &program(case, &source));
        let result = &response["ok"]["results"][0];
        let mut expected = json!({});
        if let Some(display) = case.get("display") {
            expected = json!({"status": "rated", "bill": {"notes": [{"text": display}]}});
        }
        if let Some(code) = case.get("error") {
            expected = json!({"status": "error", "error": {"code": code}});
        }
        if let Some(text) = case.get("refusal") {
            expected = json!({"status": "refused", "refusal": {"message": text}});
        }
        if let Some(message) = case.get("message") {
            expected["error"]["message"] = message.clone();
        }
        return compare(result, &expected);
    }
    if let Some(source) = case.get("source").and_then(Value::as_str) {
        let response = call("tariff.evaluate", &program(case, source));
        return compare(&response["ok"]["results"][0], &case["expect"]);
    }
    if let Some(source) = case.get("tests").and_then(Value::as_str) {
        let mut input = json!({"source": source});
        for key in ["context", "names", "budget"] {
            if let Some(value) = case.get(key) {
                input[key] = value.clone();
            }
        }
        let response = call("tariff.test", &input);
        let expected = &case["expect"];
        return if expected.is_array() {
            compare(&response["ok"]["tests"], expected)
        } else {
            compare(&response["ok"], expected)
        };
    }
    if let Some(op) = case.get("op").and_then(Value::as_str) {
        let response = call(op, &case["input"]);
        return compare(&response, &case["expect"]);
    }
    Err(String::from("the case has no expr, source, tests or op"))
}

fn compare(actual: &Value, expected: &Value) -> Result<(), String> {
    let mut problems = Vec::new();
    includes(actual, expected, "", &mut problems);
    if problems.is_empty() {
        Ok(())
    } else {
        Err(format!("{}\n  actual: {actual}", problems.join("\n  ")))
    }
}

#[test]
fn the_corpus_passes() {
    let directory = Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata/tariff/eval");
    let mut files: Vec<_> = std::fs::read_dir(&directory)
        .map(|entries| {
            entries
                .filter_map(Result::ok)
                .map(|entry| entry.path())
                .collect()
        })
        .unwrap_or_default();
    files.sort();
    let mut failures = Vec::new();
    let mut count = 0;
    for file in files {
        let text = std::fs::read_to_string(&file).unwrap_or_default();
        let cases: Vec<Value> = match serde_json::from_str(&text) {
            Ok(cases) => cases,
            Err(error) => {
                failures.push(format!("{}: {error}", file.display()));
                continue;
            }
        };
        for case in cases {
            count += 1;
            if let Err(problem) = run(&case) {
                failures.push(format!(
                    "{} / {}: {problem}",
                    file.file_name()
                        .and_then(|name| name.to_str())
                        .unwrap_or_default(),
                    case["name"]
                ));
            }
        }
    }
    assert!(
        failures.is_empty(),
        "{} of {count} cases failed:\n{}",
        failures.len(),
        failures.join("\n\n")
    );
    assert!(
        count >= 250,
        "the corpus has {count} cases; it needs at least 250"
    );
}
