//! Runs a `testdata/<area>.json` corpus through the operation table.
//!
//! A corpus is `{"op": "<default op>", "cases": [...]}`. Each case names itself, may override
//! the op, gives the `input`, and expects either an `output` or an `error` code:
//!
//! - `output` is matched as a pattern against the `ok` value: an object matches when each of its
//!   keys matches (other keys may be present; the string `"$absent"` requires the key to be
//!   missing), an array when it has the same length and each element matches, and anything else
//!   when it is equal. With `"exact": true` the whole value must be equal.
//! - `error` is the request error code (`invalid_input`, ...).
//!
//! The input is sent as the text it is written as, never re-encoded, so `1.005` and a number
//! with 28 significant digits reach the module exactly. The same files are what the Go host
//! and the browser loader run their integrations against.

use serde::Deserialize;
use serde_json::Value;
use serde_json::value::RawValue;
use std::fmt::Write as _;

#[derive(Deserialize)]
struct Corpus<'a> {
    #[serde(default)]
    op: String,
    #[serde(borrow)]
    cases: Vec<Case<'a>>,
}

#[derive(Deserialize)]
struct Case<'a> {
    name: String,
    #[serde(default)]
    op: Option<String>,
    #[serde(borrow, default)]
    input: Option<&'a RawValue>,
    #[serde(default)]
    output: Option<Value>,
    #[serde(default)]
    exact: bool,
    #[serde(default)]
    error: Option<String>,
}

/// Matches `actual` against the pattern `expected`, appending each mismatch to `failures`.
fn matches(expected: &Value, actual: Option<&Value>, path: &str, failures: &mut String) {
    match (expected, actual) {
        (Value::String(marker), None) if marker == "$absent" => {}
        (Value::String(marker), Some(actual)) if marker == "$absent" => {
            let _ = writeln!(failures, "{path}: expected absent, got {actual}");
        }
        (_, None) => {
            let _ = writeln!(failures, "{path}: missing, expected {expected}");
        }
        (Value::Object(fields), Some(Value::Object(actual))) => {
            for (key, value) in fields {
                matches(value, actual.get(key), &format!("{path}.{key}"), failures);
            }
        }
        (Value::Array(items), Some(Value::Array(actual))) => {
            if items.len() != actual.len() {
                let _ = writeln!(
                    failures,
                    "{path}: expected {} items, got {}: {}",
                    items.len(),
                    actual.len(),
                    Value::Array(actual.clone())
                );
                return;
            }
            for (index, (item, got)) in items.iter().zip(actual).enumerate() {
                matches(item, Some(got), &format!("{path}[{index}]"), failures);
            }
        }
        (expected, Some(actual)) => {
            if expected != actual {
                let _ = writeln!(failures, "{path}: expected {expected}, got {actual}");
            }
        }
    }
}

fn check(case: &Case<'_>, response: &Value) -> String {
    let mut found = String::new();
    if let Some(code) = &case.error {
        let got = response
            .get("error")
            .and_then(|error| error.get("code"))
            .and_then(Value::as_str);
        if got != Some(code.as_str()) {
            let _ = writeln!(found, "expected error {code}, got {response}");
        }
        return found;
    }
    let expected = case.output.as_ref().unwrap_or(&Value::Null);
    match response.get("ok") {
        Some(actual) if case.exact => {
            if actual != expected {
                let _ = writeln!(found, "expected exactly {expected}, got {actual}");
            }
        }
        Some(actual) => matches(expected, Some(actual), "ok", &mut found),
        None => {
            let _ = writeln!(found, "expected ok, got {response}");
        }
    }
    found
}

/// Runs every case of a corpus and returns how many ran. Panics with every failure listed.
pub fn run_corpus(corpus: &str) -> usize {
    let corpus: Corpus<'_> = serde_json::from_str(corpus).expect("the corpus is a corpus");
    let mut failures = String::new();
    for case in &corpus.cases {
        let op = case.op.as_deref().unwrap_or(&corpus.op);
        let input = case.input.map_or("null", RawValue::get);
        let request = format!(
            "{{\"op\":{},\"input\":{input}}}",
            serde_json::to_string(op).expect("an op name")
        );
        let response = trenova_rules::ops::dispatch(request.as_bytes());
        let response: Value = serde_json::from_slice(&response).expect("the response is JSON");
        let found = check(case, &response);
        if !found.is_empty() {
            let _ = writeln!(failures, "case {:?} ({op}):\n{found}", case.name);
        }
    }
    assert!(failures.is_empty(), "{failures}");
    corpus.cases.len()
}

#[test]
fn the_runner_matches_patterns() {
    let corpus = r#"{"op": "rules.version", "cases": [
        {"name": "subset", "input": {}, "output": {"operations": ["$skip"], "nope": "$absent"}}
    ]}"#;
    let result = std::panic::catch_unwind(|| run_corpus(corpus));
    assert!(result.is_err(), "an array of the wrong length must fail");
    let corpus = r#"{"op": "rules.version", "cases": [
        {"name": "absent", "input": {}, "output": {"version": "$absent"}}
    ]}"#;
    let result = std::panic::catch_unwind(|| run_corpus(corpus));
    assert!(result.is_err(), "a present key expected absent must fail");
    let corpus = r#"{"op": "nope", "cases": [{"name": "unknown", "error": "unknown_op"}]}"#;
    assert_eq!(run_corpus(corpus), 1);
}
