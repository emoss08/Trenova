//! The standard library's own tests pass under the test runner.

use serde_json::{Value, json};

fn call(op: &str, input: &Value) -> Value {
    let request = json!({"op": op, "input": input}).to_string();
    let response = trenova_rules::ops::dispatch(request.as_bytes());
    serde_json::from_slice(&response).unwrap_or(Value::Null)
}

#[test]
fn every_standard_library_module_passes_its_tests() {
    for (path, source) in trenova_rules::tariff::stdlib::MODULES {
        let response = call("tariff.test", &json!({"source": source}));
        let tests = response["ok"]["tests"]
            .as_array()
            .cloned()
            .unwrap_or_default();
        assert!(!tests.is_empty(), "{path} has no tests: {response}");
        let failed: Vec<String> = tests
            .iter()
            .filter(|test| test["status"] != "passed")
            .map(|test| format!("{path}: {test}"))
            .collect();
        assert!(failed.is_empty(), "{}", failed.join("\n"));
    }
}
