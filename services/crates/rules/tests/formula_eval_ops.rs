//! `formula.evaluate` end to end through the module's one entry point: the request and response
//! shapes, amounts and predicates, statement traces, lookup receipts, budgets and request errors.

use serde_json::Value as Json;
use serde_json::json;
use trenova_rules::ops::dispatch;

fn call(input: &Json) -> Json {
    let request = json!({"op": "formula.evaluate", "input": input});
    let response = dispatch(request.to_string().as_bytes());
    serde_json::from_slice(&response).expect("the response is JSON")
}

fn ok(input: &Json) -> Json {
    let response = call(input);
    response
        .get("ok")
        .cloned()
        .unwrap_or_else(|| panic!("not ok: {response}"))
}

fn results(input: &Json) -> Vec<Json> {
    ok(input)["results"]
        .as_array()
        .cloned()
        .expect("results are a list")
}

fn only(source: &str, expect: &str, env: &Json) -> Json {
    let mut all = results(&json!({
        "programs": [{"id": "only", "source": source, "expect": expect}],
        "env": env,
    }));
    all.remove(0)
}

fn request_error(input: &Json) -> String {
    let response = call(input);
    assert_eq!(
        response["error"]["code"], "invalid_input",
        "not a request error: {response}"
    );
    response["error"]["detail"]
        .as_str()
        .unwrap_or_default()
        .to_owned()
}

#[test]
fn one_request_prices_the_main_expression_and_every_breakdown_line() {
    let out = results(&json!({
        "programs": [
            {"id": "expression", "source": "linehaul + fuel", "expect": "amount"},
            {"id": "linehaul", "source": "baseRate * miles", "expect": "amount"},
            {"id": "fuel", "source": "miles * 0.1", "expect": "amount"},
        ],
        "env": {"baseRate": 2.5, "miles": 100, "linehaul": 250.00, "fuel": 10},
    }));
    assert_eq!(out.len(), 3);
    assert_eq!(out[0], json!({"id": "expression", "value": "260"}));
    assert_eq!(out[1], json!({"id": "linehaul", "value": "250"}));
    assert_eq!(out[2], json!({"id": "fuel", "value": "10"}));
}

#[test]
fn amounts_are_exact_decimal_strings() {
    assert_eq!(only("0.1 + 0.2", "amount", &json!({}))["value"], "0.3");
    assert_eq!(
        only("round(1.005, 2)", "amount", &json!({}))["value"],
        "1.01"
    );
    assert_eq!(only("x", "amount", &json!({"x": 12.50}))["value"], "12.5");
    assert_eq!(
        only("x * 3", "amount", &json!({"x": "0.1"}))["error"]["code"],
        "type",
        "a JSON string is text, never a number"
    );
}

#[test]
fn an_amount_must_be_a_number() {
    let boolean = only("a > b", "amount", &json!({"a": 2, "b": 1}));
    assert_eq!(boolean["error"]["code"], "booleanAmount");
    assert_eq!(boolean["error"]["from"], 0);
    assert_eq!(boolean["error"]["to"], 5);
    assert!(
        boolean["error"]["message"]
            .as_str()
            .unwrap_or_default()
            .contains("rather than an amount")
    );
    let empty = only("coalesce(a, b)", "amount", &json!({"a": null, "b": null}));
    assert_eq!(empty["error"]["code"], "nullResult");
    assert_eq!(empty["error"]["args"], json!(["nil"]));
    let text = only("x", "amount", &json!({"x": "hello"}));
    assert_eq!(text["error"]["code"], "nonNumericResult");
    assert_eq!(text["error"]["args"], json!(["text"]));
    assert!(text.get("value").is_none());
    let list = only("[1]", "amount", &json!({}));
    assert_eq!(list["error"]["code"], "nonNumericResult");
}

#[test]
fn a_predicate_is_true_false_or_a_number() {
    assert_eq!(
        only("a > b", "predicate", &json!({"a": 2, "b": 1}))["value"],
        true
    );
    assert_eq!(
        only("totalStops - 2", "predicate", &json!({"totalStops": 2}))["value"],
        "0"
    );
    let stubbed = only(
        r#"lookup("anything", 1) + lookup2("grid", "a", 2) == 0"#,
        "predicate",
        &json!({}),
    );
    assert_eq!(
        stubbed["value"], true,
        "a predicate reads every table as zero"
    );
    assert_eq!(
        only("x", "predicate", &json!({"x": "yes"}))["error"]["code"],
        "nonNumericResult"
    );
}

#[test]
fn a_missing_value_names_the_field_where_it_is_read() {
    let out = only(
        "weight * rate",
        "amount",
        &json!({"weight": null, "rate": 2.5}),
    );
    let error = &out["error"];
    assert_eq!(error["code"], "missingField");
    assert_eq!(error["fields"], json!(["weight"]));
    assert_eq!(error["guards"], json!(["coalesce(weight, 0)"]));
    assert_eq!(error["from"], 0);
    assert_eq!(error["to"], 6);
    assert_eq!(
        error["message"],
        "no value for {0} on this record; write {1} so the formula still prices when it is missing"
    );
    assert_eq!(error["args"], json!(["weight", "coalesce(weight, 0)"]));
}

#[test]
fn error_ranges_count_utf16_units() {
    let out = only(
        "let tarifa_año = 1; tarifa_año + weight",
        "amount",
        &json!({"weight": null}),
    );
    assert_eq!(out["error"]["from"], 33);
    assert_eq!(out["error"]["to"], 39);
}

#[test]
fn statement_traces_evaluate_every_let_in_one_pass() {
    let out = results(&json!({
        "programs": [{
            "id": "t",
            "source": "let base = rate * miles; let label = \"x;y\"; let fast = miles > 100; base * 1.5",
            "expect": "amount",
        }],
        "env": {"rate": 2.5, "miles": 200},
        "trace": {"statements": true},
    }));
    let statements = out[0]["statements"].as_array().cloned().expect("traced");
    assert_eq!(statements.len(), 4);
    assert_eq!(
        statements[0],
        json!({"index": 0, "kind": "let", "name": "base", "code": "let base = rate * miles", "from": 0, "to": 23, "value": "500"})
    );
    assert_eq!(statements[1]["value"], "x;y");
    assert_eq!(statements[2]["value"], true);
    assert_eq!(
        statements[3],
        json!({"index": 3, "kind": "result", "code": "base * 1.5", "from": 68, "to": 78, "value": "750"})
    );
    assert_eq!(out[0]["value"], "750");
}

#[test]
fn a_let_that_divides_by_zero_fails_the_formula_where_float64_carried_infinity() {
    let out = results(&json!({
        "programs": [{
            "id": "t",
            "source": "let zero = 0.0; let ratio = 1 / zero; let fine = 4; fine * 2",
            "expect": "amount",
        }],
        "trace": {"statements": true},
    }));
    let statements = out[0]["statements"].as_array().cloned().expect("traced");
    assert_eq!(statements[0]["value"], "0");
    assert_eq!(statements[1]["error"]["code"], "divideByZero");
    assert!(statements[1].get("value").is_none());
    assert_eq!(
        statements[2]["value"], "4",
        "the trace goes on past a failed let"
    );
    assert_eq!(statements[3]["error"]["code"], "divideByZero");
    assert_eq!(out[0]["error"]["code"], "divideByZero");
    assert_eq!(out[0]["error"]["from"], 28);
    assert_eq!(out[0]["error"]["to"], 36);
}

#[test]
fn a_result_that_is_true_or_false_is_reported_in_the_trace() {
    let out = results(&json!({
        "programs": [{"id": "t", "source": "let ok = 3; ok > 2", "expect": "amount"}],
        "trace": {"statements": true},
    }));
    let statements = out[0]["statements"].as_array().cloned().expect("traced");
    assert_eq!(statements[0]["value"], "3");
    assert_eq!(statements[1]["error"]["code"], "booleanAmount");
}

#[test]
fn statement_positions_are_utf16() {
    let source = "let tag = \"🚚 é\"; // ✓ comment\nlet miles = 10; miles * 2;";
    let out = results(&json!({
        "programs": [{"id": "t", "source": source, "expect": "amount"}],
        "trace": {"statements": true},
    }));
    let statements = out[0]["statements"].as_array().cloned().expect("traced");
    let units: Vec<u16> = source.encode_utf16().collect();
    for statement in &statements {
        let from = usize::try_from(statement["from"].as_u64().unwrap_or_default()).unwrap_or(0);
        let to = usize::try_from(statement["to"].as_u64().unwrap_or_default()).unwrap_or(0);
        let text = String::from_utf16(&units[from..to]).expect("utf-16");
        assert_eq!(text, statement["code"].as_str().unwrap_or_default());
    }
    assert_eq!(statements[0]["code"], "let tag = \"🚚 é\"");
    assert_eq!(statements[0]["value"], "🚚 é");
    assert_eq!(statements[2]["value"], "20");
}

#[test]
fn trace_values_follow_the_go_trace() {
    let out = results(&json!({
        "programs": [{
            "id": "t",
            "source": "let stops = [{city: \"Dallas\", at: pickup, gone: nil}]; let wait = duration(\"90m\"); 1",
            "expect": "amount",
        }],
        "env": {"pickup": {"$date": "2026-01-02T03:04:05-05:00"}},
        "trace": {"statements": true},
    }));
    let statements = out[0]["statements"].as_array().cloned().expect("traced");
    assert_eq!(
        statements[0]["value"],
        json!([{"at": "2026-01-02T08:04:05Z", "city": "Dallas"}])
    );
    assert_eq!(statements[1]["value"], "1h30m0s");
}

fn tables() -> Json {
    json!([
        {"name": "fsc", "axes": [{"match": "Exact"}], "cells": [{"at": [{"key": "DIESEL"}], "value": "0.35"}]},
        {"name": "miles", "axes": [{"match": "Range", "kind": "Distance", "overflow": "ClampToTopBand"}], "cells": [
            {"at": [{"min": "0", "max": "500"}], "value": "2.5"},
            {"at": [{"min": "500", "max": "1000"}], "value": 2.0},
        ]},
        {"name": "class_rates", "axes": [{"match": "Exact", "normalization": "Upper"}, {"match": "Range", "kind": "WeightBreak"}], "cells": [
            {"at": [{"key": "zone_5"}, {"min": 0, "max": 5000}], "value": "18.50"},
            {"at": [{"key": "zone_5"}, {"min": 5000}], "value": "12.25"},
        ]},
    ])
}

#[test]
fn lookups_are_recorded_for_the_receipt() {
    let out = results(&json!({
        "programs": [{
            "id": "expression",
            "source": "lookup(\"fsc\", \"DIESEL\") * lookup(\"miles\", 1200) + lookup2(\"class_rates\", \"Zone_5\", 8500) + lookupOr(\"fsc\", \"GAS\", 0)",
            "expect": "amount",
        }],
        "tables": tables(),
        "trace": {"lookups": true},
    }));
    assert_eq!(out[0]["value"], "12.95");
    let lookups = out[0]["lookups"].as_array().cloned().expect("recorded");
    assert_eq!(
        lookups[0],
        json!({"function": "lookup", "table": "fsc", "keys": ["DIESEL"], "value": "0.35", "match": {"matchedKey": "DIESEL"}})
    );
    assert_eq!(
        lookups[1],
        json!({"function": "lookup", "table": "miles", "keys": ["1200"], "value": "2", "match": {"bandMin": "500", "bandMax": "1000", "adjusted": true}})
    );
    assert_eq!(
        lookups[2],
        json!({"function": "lookup2", "table": "class_rates", "keys": ["Zone_5", "8500"], "value": "12.25", "match": {"matchedKey": "ZONE_5 × 5000+"}})
    );
    assert_eq!(lookups[3]["function"], "lookupOr");
    assert_eq!(lookups[3]["error"]["code"], "lookupMiss");
    assert_eq!(lookups[3]["error"]["args"], json!(["fsc", "GAS"]));
    assert!(lookups[3].get("value").is_none());
}

#[test]
fn without_tables_lookups_are_unavailable_and_an_empty_set_has_none() {
    let unavailable = only("lookup(\"fsc\", \"DIESEL\")", "amount", &json!({}));
    assert_eq!(unavailable["error"]["code"], "lookupUnavailable");
    let none = results(&json!({
        "programs": [{"id": "x", "source": "lookup(\"fsc\", \"DIESEL\")", "expect": "amount"}],
        "tables": [],
    }));
    assert_eq!(none[0]["error"]["code"], "tableNotFound");
}

#[test]
fn a_two_axis_lookup_scans_many_bands_and_overlaps() {
    let mut cells = Vec::new();
    for band in 0..50 {
        for zone in 0..40 {
            cells.push(json!({
                "at": [{"min": band * 100, "max": (band + 1) * 100}, {"key": format!("Z{zone:02}")}],
                "value": band * 1000 + zone,
            }));
        }
    }
    cells.push(json!({"at": [{"min": 4000}, {"key": "HEAVY"}], "value": 999_999}));
    let grid = json!([{
        "name": "big_grid",
        "axes": [{"match": "Range", "kind": "WeightBreak"}, {"match": "Exact", "kind": "Zone"}],
        "cells": cells,
    }]);
    let run = |source: &str| {
        results(&json!({
            "programs": [{"id": "x", "source": source, "expect": "amount"}],
            "tables": grid,
        }))
        .remove(0)
    };
    assert_eq!(
        run("lookup2(\"big_grid\", 1250, \"Z07\")")["value"],
        "12007"
    );
    assert_eq!(
        run("lookup2(\"big_grid\", 4999, \"HEAVY\")")["value"],
        "999999"
    );
    assert_eq!(
        run("lookup2(\"big_grid\", 4999, \"Z00\")")["value"],
        "49000"
    );
    assert_eq!(
        run("lookup2(\"big_grid\", 5000, \"Z00\")")["error"]["code"],
        "lookupMiss"
    );
}

#[test]
fn a_program_that_does_not_parse_is_reported_and_the_rest_still_run() {
    let out = results(&json!({
        "programs": [
            {"id": "broken", "source": "baseRate * ", "expect": "amount"},
            {"id": "fine", "source": "2 * 2", "expect": "amount"},
        ],
    }));
    assert_eq!(out[0]["error"]["code"], "syntax");
    assert!(out[0]["error"].get("from").is_some());
    assert_eq!(out[1]["value"], "4");
}

#[test]
fn budgets_may_be_lowered_but_not_raised() {
    let lowered = results(&json!({
        "programs": [{"id": "x", "source": "len(1..50)", "expect": "amount"}],
        "budget": {"items": 10},
    }));
    assert_eq!(lowered[0]["error"]["code"], "budget");
    let steps = results(&json!({
        "programs": [{"id": "x", "source": "reduce(1..100, #acc + #, 0)", "expect": "amount"}],
        "budget": {"steps": 50},
    }));
    assert_eq!(steps[0]["error"]["code"], "budget");
    let text = results(&json!({
        "programs": [{"id": "x", "source": "len(\"ab\" + \"cd\")", "expect": "amount"}],
        "budget": {"textBytes": 3},
    }));
    assert_eq!(text[0]["error"]["code"], "budget");
    assert_eq!(text[0]["error"]["args"], json!([3]));
    let detail = request_error(&json!({
        "programs": [],
        "budget": {"steps": 10_000_000_u64},
    }));
    assert!(detail.contains("may only lower"), "{detail}");
}

#[test]
fn each_program_runs_under_its_own_budget() {
    let out = results(&json!({
        "programs": [
            {"id": "a", "source": "len(1..8)", "expect": "amount"},
            {"id": "b", "source": "len(1..8)", "expect": "amount"},
        ],
        "budget": {"items": 10},
    }));
    assert_eq!(out[0]["value"], "8");
    assert_eq!(out[1]["value"], "8");
}

#[test]
fn malformed_requests_are_request_errors() {
    assert!(request_error(&json!({})).contains("programs"));
    assert!(request_error(&json!({"programs": [], "env": [1]})).contains("object"));
    assert!(request_error(&json!({"programs": [], "env": {"$when": "x"}})).contains("reserved"));
    assert!(
        request_error(&json!({"programs": [], "env": {"a": {"$date": "tomorrow"}}}))
            .contains("RFC 3339")
    );
    assert!(request_error(&json!({"programs": [], "extra": 1})).contains("extra"));
    assert!(
        request_error(&json!({"programs": [{"id": "x", "source": "1", "expect": "price"}]}))
            .contains("price")
    );
    let three_axes = json!([{"name": "t", "axes": [{"match": "Exact"}, {"match": "Exact"}, {"match": "Exact"}], "cells": []}]);
    assert!(request_error(&json!({"programs": [], "tables": three_axes})).contains("3 axes"));
    let twice = json!([
        {"name": "t", "axes": [{"match": "Exact"}], "cells": []},
        {"name": "t", "axes": [{"match": "Exact"}], "cells": []},
    ]);
    assert!(request_error(&json!({"programs": [], "tables": twice})).contains("twice"));
    let short_cell = json!([{"name": "t", "axes": [{"match": "Exact"}, {"match": "Range"}], "cells": [{"at": [{"key": "a"}], "value": 1}]}]);
    assert!(request_error(&json!({"programs": [], "tables": short_cell})).contains("position"));
    let bad_mode = json!([{"name": "t", "axes": [{"match": "Fuzzy"}], "cells": []}]);
    assert!(request_error(&json!({"programs": [], "tables": bad_mode})).contains("Fuzzy"));
}

#[test]
fn the_schema_is_accepted_and_left_to_the_checker() {
    let out = results(&json!({
        "programs": [{"id": "x", "source": "weight * 2", "expect": "amount"}],
        "schema": {"fields": [{"name": "weight", "type": "Number"}]},
        "env": {"weight": 3},
    }));
    assert_eq!(out[0]["value"], "6");
}

#[test]
fn dates_and_durations_arrive_tagged() {
    let out = only(
        "(dropoff - pickup).Hours() + wait.Minutes()",
        "amount",
        &json!({
            "pickup": {"$date": "2026-01-02T08:00:00-05:00"},
            "dropoff": {"$date": "2026-01-02T15:30:00Z"},
            "wait": {"$duration": "1800000000000"},
        }),
    );
    assert_eq!(out["value"], "32.5");
}
