//! Every example on the Tariff cheat sheet checks clean and passes its own tests.

use serde_json::{Value, json};

const SHEET: &str = include_str!("../../../../docs/engineering/tariff-cheat-sheet.md");

fn call(op: &str, input: &Value) -> Value {
    let request = json!({"op": op, "input": input}).to_string();
    let response = trenova_rules::ops::dispatch(request.as_bytes());
    serde_json::from_slice(&response).unwrap_or(Value::Null)
}

/// Every ```` ```tariff ```` block of the sheet, in order.
fn blocks() -> Vec<&'static str> {
    let mut out = Vec::new();
    let mut rest = SHEET;
    while let Some(start) = rest.find("```tariff\n") {
        let body = &rest[start + "```tariff\n".len()..];
        let Some(end) = body.find("\n```") else { break };
        out.push(&body[..end]);
        rest = &body[end..];
    }
    out
}

/// The charge codes and rate matrices the sheet's examples name.
fn context() -> Value {
    let codes: Vec<Value> = ["FSC", "XDROP", "HAZ", "DET", "LIFT"]
        .iter()
        .map(|code| json!({"code": code, "active": true}))
        .collect();
    json!({
        "codes": codes,
        "tables": [
            {
                "code": "Zones",
                "valueType": "Money/cwt",
                "axes": [{"match": "Exact", "kind": "Zone"}, {"match": "Exact", "kind": "Zone"}],
                "cells": [{"at": [{"key": "A"}, {"key": "B"}], "value": "20"}]
            },
            {
                "code": "Breaks",
                "valueType": "Money/cwt",
                "axes": [{"match": "Range", "kind": "WeightBreak", "unit": "lb"}],
                "cells": [
                    {"at": [{"min": "0", "max": "1000"}], "value": "30"},
                    {"at": [{"min": "1000"}], "value": "25"}
                ]
            }
        ]
    })
}

#[test]
fn every_example_checks_clean_and_passes_its_tests() {
    let blocks = blocks();
    assert!(
        blocks.len() >= 8,
        "the sheet has {} tariff blocks",
        blocks.len()
    );
    let context = context();
    for block in blocks {
        let header = block.lines().next().unwrap_or_default();
        let checked = call(
            "tariff.check",
            &json!({"source": block, "context": context}),
        );
        let diagnostics = checked["ok"]["diagnostics"]
            .as_array()
            .cloned()
            .unwrap_or_default();
        assert!(
            checked["ok"]["ok"] == true && diagnostics.is_empty(),
            "{header}: {checked}"
        );
        if !block.contains("\ntest \"") {
            continue;
        }
        let run = call("tariff.test", &json!({"source": block, "context": context}));
        let tests = run["ok"]["tests"].as_array().cloned().unwrap_or_default();
        assert!(!tests.is_empty(), "{header}: {run}");
        let failed: Vec<String> = tests
            .iter()
            .filter(|test| test["status"] != "passed")
            .map(|test| format!("{header}: {test}"))
            .collect();
        assert!(failed.is_empty(), "{}", failed.join("\n"));
    }
}
