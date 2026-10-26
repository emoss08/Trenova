//! The editor corpus under `testdata/tariff/editor/`, the operations through dispatch, the work
//! budget on a 2,000-line tariff, and never-panic properties.
//!
//! A corpus file holds `{"cases": [{"name", "source": [lines], "context"?, "input"?,
//! "expect"}]}`. The source marks the cursor with `|` (removed before the request; inlays
//! cases may leave it out; an actions case marks its selection with two, or a cursor with
//! one); `context` replaces keys of the shared context ([`context`]); `input` adds fields to
//! the request (`newName`, `values`, `range`, `includeDeclaration`). `expect` is checked
//! against the response the way the checker corpus checks: objects by the keys written, arrays
//! item by item. Before comparing, every `{from, to}` range in the response gains its `text`
//! (from the module it is in: a reference or definition target in a library reads that
//! library's source), every inlay hint gains `line`: its line with the label written in at its
//! offset as `⟨label⟩`, and every action gains `result`, the source with its edits applied (as
//! lines), its `select` reading that result. An action whose source checks clean must leave a
//! result that checks clean, and one whose source is formatted a formatted result. A completion case's `expect` may instead say
//! `includes` (labels that must be offered), `excludes` (labels that must not), `order` (labels
//! that must sort in this order), `items` (label to the item's expected fields), `range` (the
//! replaced text) and `empty`.

use crate::ops::dispatch;
use crate::tariff::stdlib;
use serde_json::{Map, Value as Json, json};
use std::fmt::Write;

const FILES: &[(&str, &str)] = &[
    (
        "complete",
        include_str!("../../../testdata/tariff/editor/complete.json"),
    ),
    (
        "hover",
        include_str!("../../../testdata/tariff/editor/hover.json"),
    ),
    (
        "signature",
        include_str!("../../../testdata/tariff/editor/signature.json"),
    ),
    (
        "definition",
        include_str!("../../../testdata/tariff/editor/definition.json"),
    ),
    (
        "rename",
        include_str!("../../../testdata/tariff/editor/rename.json"),
    ),
    (
        "inlays",
        include_str!("../../../testdata/tariff/editor/inlays.json"),
    ),
    (
        "references",
        include_str!("../../../testdata/tariff/editor/references.json"),
    ),
    (
        "actions",
        include_str!("../../../testdata/tariff/editor/actions.json"),
    ),
];

/// The context every case starts from: charge codes, three rate tables (one with cells) and
/// two organisation libraries.
fn context() -> Json {
    json!({
        "codes": [
            {"code": "FSC", "description": "Fuel surcharge", "active": true},
            {"code": "DET", "description": "Detention", "active": true},
            {"code": "HAZ", "description": "Hazardous materials", "active": true},
            {"code": "XDROP", "description": "Extra drop", "active": true},
            {"code": "LIFT", "description": "Liftgate", "active": false}
        ],
        "tables": [
            {
                "code": "Zones",
                "name": "Zone rates",
                "axes": [
                    {"kind": "Zip3", "label": "origin zip3", "match": "Exact"},
                    {"kind": "Zip3", "label": "destination zip3", "match": "Exact"}
                ],
                "valueType": "Money/cwt",
                "cells": [
                    {"at": [{"key": "752"}, {"key": "606"}], "value": "18.5"},
                    {"at": [{"key": "752"}, {"key": "303"}], "value": "21"},
                    {"at": [{"key": "100"}, {"key": "606"}], "value": "24.75"}
                ]
            },
            {
                "code": "Breaks",
                "name": "Weight breaks",
                "axes": [{"kind": "WeightBreak", "label": "weight", "match": "Range", "unit": "lb"}],
                "valueType": "Money"
            },
            {
                "code": "ClassRates",
                "axes": [{"kind": "FreightClass", "label": "class", "match": "Exact"}],
                "valueType": "Money/cwt",
                "cells": [
                    {"at": [{"key": "Class50"}], "value": "10"},
                    {"at": [{"key": "Class70"}], "value": "12"}
                ]
            }
        ],
        "libraries": [
            {
                "path": "org.accessorials",
                "version": "v3",
                "source": "library \"org.accessorials\"\n\n/// Accessorial prices the organisation shares.\n\n/// Hazmat handling per loaded mile.\nfn hazmat(miles: Distance) -> Money = $0.10/mi * miles\n\n/// A liftgate at `stops` stops, `fee` each.\nfn liftgate(stops: Int, fee: Money = $75) -> Money = fee * stops\n\n/// A zone of the network.\nenum Zone {\n  /// The home zone.\n  North,\n  South\n}\n\n/// A fee and its floor.\ntype Fee = { amount: Money, floor: Money? }\n\n/// The base fee.\nconst BASE: Money = $25\n\nprivate fn secret() -> Money = $1\n"
            },
            {
                "path": "org.lanes",
                "source": "library \"org.lanes\"\n\n/// Whether a zip3 is in the metro.\nfn metro(zip3: Text) -> Bool = zip3 in [\"606\", \"752\"]\n"
            }
        ]
    })
}

fn text(value: &Json) -> String {
    match value {
        Json::Array(lines) => lines
            .iter()
            .filter_map(Json::as_str)
            .collect::<Vec<_>>()
            .join("\n"),
        Json::String(text) => text.clone(),
        _ => String::new(),
    }
}

/// The source without its `|` markers, and the UTF-16 offsets of the first two (the second,
/// for an actions case's selection, as if the first were already removed).
fn selection(source: &str) -> (String, Option<(u32, u32)>) {
    let (once, first) = marked(source);
    let Some(first) = first else {
        return (once, None);
    };
    let (twice, second) = marked(&once);
    (twice, Some((first, second.unwrap_or(first))))
}

/// The source without its `|` marker, and the marker's UTF-16 offset.
fn marked(source: &str) -> (String, Option<u32>) {
    match source.find('|') {
        Some(at) => {
            let mut clean = source.to_owned();
            clean.remove(at);
            let offset = source
                .get(..at)
                .map_or(0, |before| before.encode_utf16().count());
            (clean, u32::try_from(offset).ok())
        }
        None => (source.to_owned(), None),
    }
}

/// The source of the module a definition target names.
fn module_source<'a>(module: Option<&str>, root: &'a str, context: &'a Json) -> String {
    match module {
        None => root.to_owned(),
        Some(path) if path.starts_with("std.") => {
            stdlib::source(path).unwrap_or_default().to_owned()
        }
        Some(path) => context["libraries"]
            .as_array()
            .into_iter()
            .flatten()
            .find(|library| library["path"] == path)
            .and_then(|library| library["source"].as_str())
            .unwrap_or_default()
            .to_owned(),
    }
}

/// The text a UTF-16 range covers.
fn slice16(source: &str, from: u64, to: u64) -> String {
    let units: Vec<u16> = source.encode_utf16().collect();
    let from = usize::try_from(from).unwrap_or(0).min(units.len());
    let to = usize::try_from(to).unwrap_or(0).clamp(from, units.len());
    String::from_utf16_lossy(units.get(from..to).unwrap_or_default())
}

/// Adds `text` to every range of a value, from `source`.
fn texts(value: &mut Json, source: &str) {
    match value {
        Json::Object(object) => {
            if let (Some(from), Some(to)) = (
                object.get("from").and_then(Json::as_u64),
                object.get("to").and_then(Json::as_u64),
            ) {
                object.insert(
                    String::from("text"),
                    Json::String(slice16(source, from, to)),
                );
            }
            for (_, inner) in object.iter_mut() {
                texts(inner, source);
            }
        }
        Json::Array(items) => {
            for item in items {
                texts(item, source);
            }
        }
        _ => {}
    }
}

/// The response with ranges' texts and hints' lines written in.
fn annotate(op: &str, response: &mut Json, source: &str, context: &Json) {
    if op == "definition" {
        for target in response["targets"].as_array_mut().into_iter().flatten() {
            let module = module_source(target["module"].as_str(), source, context);
            texts(target, &module);
        }
        return;
    }
    if op == "references" {
        for reference in response["references"].as_array_mut().into_iter().flatten() {
            let module = module_source(reference["module"].as_str(), source, context);
            texts(reference, &module);
        }
        return;
    }
    if op == "actions" {
        for action in response["actions"].as_array_mut().into_iter().flatten() {
            let result = applied(source, &action["edits"]);
            texts(&mut action["edits"], source);
            texts(&mut action["select"], &result);
            action["result"] = Json::Array(result.split('\n').map(|line| json!(line)).collect());
        }
        return;
    }
    texts(response, source);
    if op == "inlays" {
        let lines: Vec<&str> = source.split('\n').collect();
        for hint in response["hints"].as_array_mut().into_iter().flatten() {
            let offset = hint["offset"].as_u64().unwrap_or(0);
            let before: String = slice16(source, 0, offset);
            let line = before.matches('\n').count();
            let column = before
                .rsplit('\n')
                .next()
                .unwrap_or_default()
                .encode_utf16()
                .count();
            let written = lines.get(line).copied().unwrap_or_default();
            let units: Vec<u16> = written.encode_utf16().collect();
            let (head, tail) = units.split_at(column.min(units.len()));
            let label = hint["label"].as_str().unwrap_or_default();
            hint["line"] = Json::String(format!(
                "{}⟨{label}⟩{}",
                String::from_utf16_lossy(head),
                String::from_utf16_lossy(tail)
            ));
        }
    }
}

/// A source with edits (`[{range: {from, to}, insert}]`, UTF-16, none overlapping) applied.
fn applied(source: &str, edits: &Json) -> String {
    let mut units: Vec<u16> = source.encode_utf16().collect();
    let mut edits: Vec<(usize, usize, String)> = edits
        .as_array()
        .into_iter()
        .flatten()
        .map(|edit| {
            (
                usize::try_from(edit["range"]["from"].as_u64().unwrap_or(0)).unwrap_or(0),
                usize::try_from(edit["range"]["to"].as_u64().unwrap_or(0)).unwrap_or(0),
                edit["insert"].as_str().unwrap_or_default().to_owned(),
            )
        })
        .collect();
    edits.sort_by(|a, b| b.0.cmp(&a.0).then(b.1.cmp(&a.1)));
    for (from, to, insert) in edits {
        let to = to.clamp(from, units.len());
        let from = from.min(to);
        units.splice(from..to, insert.encode_utf16());
    }
    String::from_utf16_lossy(&units)
}

/// Whether a file checks clean: `tariff.check` answers `ok: true`.
fn checks_clean(source: &str, context: &Json) -> bool {
    let request = json!({"op": "tariff.check", "input": {"source": source, "context": context}});
    let response: Json =
        serde_json::from_slice(&dispatch(request.to_string().as_bytes())).unwrap_or(Json::Null);
    response["ok"]["ok"] == Json::Bool(true)
}

/// Whether the formatter leaves a source as it is (its final newline aside).
fn is_formatted(source: &str) -> bool {
    crate::tariff::syntax::format::format(source)
        .is_ok_and(|out| out.trim_end() == source.trim_end())
}

/// Whether an actions case's source offers an action, checks clean and is formatted: the cases
/// that prove promoted files stay clean.
fn proves(response: &Json, source: &str, context: &Json) -> bool {
    response["actions"]
        .as_array()
        .is_some_and(|actions| !actions.is_empty())
        && checks_clean(source, context)
        && is_formatted(source)
}

/// The problems with an actions response's edits, for a source that checks clean: a result
/// that does not, and for a formatted source a result the formatter rewrites (unless a
/// replacement pushed a line past the formatter's width, which the edits leave to it).
fn actions_hold(response: &Json, source: &str, context: &Json, problems: &mut Vec<String>) {
    if !checks_clean(source, context) {
        return;
    }
    let formatted = is_formatted(source);
    for action in response["actions"].as_array().into_iter().flatten() {
        let result = applied(source, &action["edits"]);
        let id = action["id"].as_str().unwrap_or("?");
        if !checks_clean(&result, context) {
            problems.push(format!("{id} leaves a file that does not check: {result}"));
        }
        let narrow = result
            .lines()
            .all(|line| line.chars().count() <= crate::tariff::syntax::format::WIDTH);
        if formatted && narrow && !is_formatted(&result) {
            problems.push(format!(
                "{id} leaves a file the formatter rewrites: {result}"
            ));
        }
    }
}

/// Whether `actual` holds everything `expected` says.
fn holds(actual: &Json, expected: &Json, path: &str, problems: &mut Vec<String>) {
    match (actual, expected) {
        (Json::Object(actual), Json::Object(expected)) => {
            for (key, value) in expected {
                match actual.get(key) {
                    Some(found) => holds(found, value, &format!("{path}.{key}"), problems),
                    None if value.is_null() => {}
                    None => problems.push(format!("{path}.{key} is missing")),
                }
            }
        }
        (Json::Array(actual), Json::Array(expected)) => {
            if actual.len() != expected.len() {
                problems.push(format!(
                    "{path} has {} items, not {}: {}",
                    actual.len(),
                    expected.len(),
                    Json::Array(actual.clone())
                ));
                return;
            }
            for (index, (found, value)) in actual.iter().zip(expected).enumerate() {
                holds(found, value, &format!("{path}[{index}]"), problems);
            }
        }
        _ if actual == expected => {}
        _ => problems.push(format!("{path} is {actual}, not {expected}")),
    }
}

/// The checks a completion case's `expect` makes.
fn completion_holds(response: &Json, expected: &Json, problems: &mut Vec<String>) {
    let items = response["items"].as_array().cloned().unwrap_or_default();
    let labels: Vec<&str> = items
        .iter()
        .filter_map(|item| item["label"].as_str())
        .collect();
    for label in expected["includes"]
        .as_array()
        .into_iter()
        .flatten()
        .filter_map(Json::as_str)
    {
        if !labels.contains(&label) {
            problems.push(format!("{label} is not offered"));
        }
    }
    for label in expected["excludes"]
        .as_array()
        .into_iter()
        .flatten()
        .filter_map(Json::as_str)
    {
        if labels.contains(&label) {
            problems.push(format!("{label} is offered"));
        }
    }
    let order: Vec<&str> = expected["order"]
        .as_array()
        .into_iter()
        .flatten()
        .filter_map(Json::as_str)
        .collect();
    let positions: Vec<Option<usize>> = order
        .iter()
        .map(|label| labels.iter().position(|found| found == label))
        .collect();
    if positions.windows(2).any(|pair| match pair {
        [Some(a), Some(b)] => a > b,
        _ => true,
    }) && order.len() > 1
    {
        problems.push(format!("{order:?} are not offered in this order"));
    }
    if let Some(Json::Object(expected_items)) = expected.get("items") {
        for (label, fields) in expected_items {
            match items.iter().find(|item| item["label"] == label.as_str()) {
                Some(item) => holds(item, fields, &format!("items[{label}]"), problems),
                None => problems.push(format!("{label} is not offered")),
            }
        }
    }
    if let Some(range) = expected.get("range") {
        holds(&response["range"]["text"], range, "range", problems);
    }
    if expected["empty"] == Json::Bool(true) && !labels.is_empty() {
        problems.push(format!("items are offered: {labels:?}"));
    }
}

/// A case's request answered: the annotated response, the source sent and the context.
///
/// # Errors
///
/// When the request fails.
fn respond(op: &str, case: &Json) -> Result<(Json, String, Json), String> {
    let (source, marks) = if op == "actions" {
        selection(&text(&case["source"]))
    } else {
        let (source, offset) = marked(&text(&case["source"]));
        (source, offset.map(|offset| (offset, offset)))
    };
    let offset = marks.map(|(first, _)| first);
    let mut context = context();
    if let (Some(base), Some(Json::Object(overrides))) =
        (context.as_object_mut(), case.get("context"))
    {
        for (key, value) in overrides {
            base.insert(key.clone(), value.clone());
        }
    }
    let mut input = Map::new();
    input.insert(String::from("source"), Json::String(source.clone()));
    input.insert(String::from("context"), context.clone());
    if op == "inlays" {
        let end = source.encode_utf16().count();
        input.insert(String::from("range"), json!({"from": 0, "to": end}));
    } else if op == "actions" {
        let (from, to) = marks.unwrap_or((0, 0));
        input.insert(String::from("range"), json!({"from": from, "to": to}));
    } else {
        input.insert(String::from("offset"), json!(offset.unwrap_or(0)));
    }
    if let Some(Json::Object(extra)) = case.get("input") {
        for (key, value) in extra {
            input.insert(key.clone(), value.clone());
        }
    }
    let request = json!({ "op": format!("tariff.{op}"), "input": input });
    let response: Json =
        serde_json::from_slice(&dispatch(request.to_string().as_bytes())).unwrap_or(Json::Null);
    let Some(mut ok) = response.get("ok").cloned() else {
        return Err(format!("the request failed: {response}"));
    };
    annotate(op, &mut ok, &source, &context);
    Ok((ok, source, context))
}

/// Runs one case; the problems found, with the response, when it fails.
fn run_case(op: &str, case: &Json) -> Option<String> {
    let (ok, source, context) = match respond(op, case) {
        Ok(answered) => answered,
        Err(problem) => return Some(problem),
    };
    let expected = case.get("expect").unwrap_or(&Json::Null);
    let mut problems = Vec::new();
    if op == "actions" {
        actions_hold(&ok, &source, &context, &mut problems);
    }
    if op == "complete"
        && expected.is_object()
        && !expected.get("items").is_some_and(Json::is_array)
    {
        completion_holds(&ok, expected, &mut problems);
    } else {
        holds(&ok, expected, "", &mut problems);
    }
    (!problems.is_empty()).then(|| {
        let shown = if op == "complete" {
            let labels: Vec<String> = ok["items"]
                .as_array()
                .into_iter()
                .flatten()
                .map(|item| {
                    format!(
                        "{} ({})",
                        item["label"].as_str().unwrap_or_default(),
                        item["kind"].as_str().unwrap_or_default()
                    )
                })
                .collect();
            format!("range {} items {}", ok["range"], labels.join(", "))
        } else {
            ok.to_string()
        };
        format!("{}\n  response: {shown}", problems.join("\n  "))
    })
}

#[test]
fn promoted_files_check_clean_in_most_cases() {
    let cases: Json =
        serde_json::from_str(include_str!("../../../testdata/tariff/editor/actions.json"))
            .unwrap_or(Json::Null);
    let proving = cases["cases"]
        .as_array()
        .into_iter()
        .flatten()
        .filter(|case| {
            respond("actions", case)
                .is_ok_and(|(ok, source, context)| proves(&ok, &source, &context))
        })
        .count();
    assert!(
        proving >= 25,
        "only {proving} actions cases start from a clean, formatted file"
    );
}

#[test]
fn the_corpus_holds() {
    let mut failures = Vec::new();
    let mut count = 0;
    for (op, file) in FILES {
        let cases: Json =
            serde_json::from_str(file).unwrap_or_else(|error| panic!("{op}.json: {error}"));
        for case in cases["cases"].as_array().into_iter().flatten() {
            count += 1;
            let name = case["name"].as_str().unwrap_or("?");
            if let Some(problem) = run_case(op, case) {
                failures.push(format!("{op}: {name}\n  {problem}"));
            }
        }
    }
    assert!(
        failures.is_empty(),
        "{} failing:\n\n{}",
        failures.len(),
        failures.join("\n\n")
    );
    assert!(count >= 150, "the corpus has {count} cases");
}

/// A tariff of at least 2,000 lines: functions, inputs, bindings, branches, loops, charges and
/// tests, every one checking.
fn generated() -> String {
    let mut out = String::from(
        "tariff \"Generated\"\n\nuse std.ltl.{ deficit, Break }\nuse std.fuel\n\n/// Paid per mile.\ninput rate: Money/mi = $2.35/mi\ninput breaks: List<Break> = []\n\n",
    );
    for index in 0..100 {
        let _ = write!(
            out,
            "/// Helper {index}.\nfn helper{index}(a: Money, b: Int = 1) -> Money {{\n  let scaled = a * b\n  return scaled.max($0)\n}}\n\n"
        );
    }
    for index in 0..150 {
        let helper = index % 100;
        let _ = write!(
            out,
            "let miles{index} = shipment.distance\nlet weight{index} = shipment.weight ?? 0 lb\nif weight{index} > 100 lb {{\n  charge \"Heavy {index}\" = $0.01/lb * weight{index}\n}}\nfor s in shipment.stops where s.kind == Delivery {{\n  let dwell = s.dwell ?? 0 h\n  note \"Stop {{s.sequence}} waited {{dwell}}\"\n}}\ncharge \"Line {index}\" (quantity: miles{index}, rate: rate)\ncharge \"Fee {index}\" = helper{helper}($5, b: 2)\n\n"
        );
    }
    out.push_str("minimum $100\nround to cents\n\n");
    for index in 0..20 {
        let _ = write!(
            out,
            "test \"generated {index}\" {{\n  given shipment.distance = {index} mi\n  expect total >= $100\n}}\n\n"
        );
    }
    out
}

#[test]
fn every_operation_answers_a_two_thousand_line_tariff_within_a_small_budget() {
    let source = generated();
    assert!(
        source.lines().count() >= 2_000,
        "{} lines",
        source.lines().count()
    );
    let context = context();
    let program = crate::tariff::syntax::program::load(
        &source,
        None,
        &serde_json::from_value(context.clone()).unwrap_or_default(),
    );
    assert!(
        program.diagnostics.is_empty(),
        "{:?}",
        program.diagnostics.first()
    );
    let at = |needle: &str, after: usize| {
        let byte = source.find(needle).unwrap_or(0) + after;
        u32::try_from(
            source
                .get(..byte)
                .unwrap_or_default()
                .encode_utf16()
                .count(),
        )
        .unwrap_or(0)
    };
    let budget = crate::tariff::eval::budget::Limits::DEFAULT.steps / 10;
    let probes = [
        ("complete", at("weight149 > 100 lb", 0)),
        (
            "complete",
            at(
                "shipment.stops where s.kind == Delivery {\n  let dwell = s.dwell ?? 0 h\n  note \"Stop {s.sequence} waited {dwell}\"\n}\ncharge \"Line 149\"",
                9,
            ),
        ),
        ("hover", at("miles149 = ", 2)),
        ("signature", at("helper49($5, b: 2)\n\nlet miles150", 9)),
        ("definition", at("weight149 > 100 lb", 2)),
        ("rename", at("let miles0 ", 6)),
        ("rename", at("fn helper7(", 5)),
        ("references", at("let miles0 ", 6)),
        ("references", at("input rate", 7)),
        ("actions", at("$0.01/lb * weight149", 2)),
        ("inlays", 0),
    ];
    for (op, offset) in probes {
        let input =
            json!({"source": source, "context": context, "offset": offset, "newName": "renamed"});
        let steps = super::ops::steps_for_tests(op, &input);
        assert!(
            steps < budget,
            "{op} at {offset} took {steps} steps, over {budget}"
        );
        let request = json!({"op": format!("tariff.{op}"), "input": if op == "inlays" {
            json!({"source": source, "context": context, "range": {"from": 0, "to": offset + 50_000}})
        } else if op == "actions" {
            json!({"source": source, "context": context, "range": {"from": offset, "to": offset}})
        } else {
            json!({"source": source, "context": context, "offset": offset})
        }});
        let response: Json =
            serde_json::from_slice(&dispatch(request.to_string().as_bytes())).unwrap_or(Json::Null);
        assert!(response.get("ok").is_some(), "{op}: {response}");
    }
}

const SPEC: &str = include_str!("../../../../../../docs/engineering/formula-language.md");

/// The reference's first example, whole.
fn first_example() -> String {
    let start = SPEC
        .find("```tariff\n")
        .map_or(0, |at| at + "```tariff\n".len());
    let body = SPEC.get(start..).unwrap_or_default();
    body.get(..body.find("\n```").unwrap_or(0))
        .unwrap_or_default()
        .to_owned()
}

fn features_at_every_offset(source: &str) {
    use super::session::Session;
    use super::values::Values;
    let context: crate::tariff::context::Context =
        serde_json::from_value(context()).unwrap_or_default();
    let session = Session::new(source, None, &context);
    let values = Values::default();
    let end = u32::try_from(source.len()).unwrap_or(0);
    for offset in 0..=end {
        if !source.is_char_boundary(offset as usize) {
            continue;
        }
        let _ = super::complete::complete(&session, offset);
        let _ = super::hover::hover(&session, offset, &values);
        let _ = super::signature::signature(&session, offset);
        let _ = super::definition::definition(&session, offset);
        let _ = super::rename::prepare(&session, offset);
        let _ = super::rename::rename(&session, offset, "renamed");
        let _ = super::references::references(&session, offset, true);
        let _ = super::actions::actions(&session, trenova_module_kit::span::Span::new(offset, offset));
        let _ = super::actions::actions(
            &session,
            trenova_module_kit::span::Span::new(offset, offset.saturating_add(5).min(end)),
        );
    }
    let _ = super::inlays::inlays(&session, trenova_module_kit::span::Span::new(0, end), &values);
}

/// A tariff with every construct the reference added last: input constraints, choices nested
/// in a loop and an option, properties with every kind of `vary`, and `chosen` in a test.
const NEW_SYNTAX: &str = "tariff \"Choices\"\n\n/// Paid per loaded mile.\ninput rate: Money/mi = $2.35/mi where $0.50/mi..=$15/mi\ninput flat: Money = $450 where [$400, $450]\ninput service: Service = Standard\n\nenum Service { Standard, Expedited }\n\nbest \"Linehaul\" {\n  option \"Per mile\" {\n    charge \"Linehaul\" (quantity: shipment.distance, rate: rate)\n  }\n  option \"Flat\" {\n    let fee = flat\n    charge \"Flat rate\" = fee\n  }\n}\n\nfor stop in shipment.stops {\n  worst \"Stop {stop.sequence}\" {\n    option \"Fee\" {\n      charge \"Stop fee\" = $5\n    }\n    option \"None\" {\n      fail \"no\"\n    }\n  }\n}\n\nproperty \"heavier never bills less\" {\n  vary shipment.weight in 1 lb..40_000 lb\n  vary input.rate\n  vary s in [Standard, Expedited]\n  assume s == Standard\n  given fuel.price = $3.20/gal\n  expect total nondecreasing in shipment.weight\n  expect total >= $0\n}\n\ntest \"flat wins\" {\n  expect chosen(\"Linehaul\") == \"Flat\"\n}\n";

#[test]
fn every_feature_answers_at_every_offset_of_the_new_syntax() {
    features_at_every_offset(NEW_SYNTAX);
    let half = NEW_SYNTAX.len() / 2;
    features_at_every_offset(NEW_SYNTAX.get(..half).unwrap_or_default());
}

#[test]
fn every_feature_answers_at_every_offset_of_the_first_example() {
    let example = first_example();
    assert!(example.starts_with("tariff \"Dry van TL\""), "{example}");
    features_at_every_offset(&example);
    let half = example.len() / 2;
    let cut: String = example.get(..half).unwrap_or_default().to_owned();
    features_at_every_offset(&cut);
}

#[test]
fn the_first_example_completes_after_its_dots() {
    let example = first_example();
    let context: crate::tariff::context::Context =
        serde_json::from_value(context()).unwrap_or_default();
    let session = super::session::Session::new(&example, None, &context);
    let at = u32::try_from(example.find("shipment.distance").unwrap_or(0) + "shipment.".len())
        .unwrap_or(0);
    let found = super::complete::complete(&session, at);
    assert!(found.items.iter().any(|item| item.label == "distance"));
}

#[test]
fn malformed_requests_are_refused() {
    for request in [
        json!({"op": "tariff.complete", "input": {"source": "x"}}),
        json!({"op": "tariff.hover", "input": {"source": "x", "offset": -1}}),
        json!({"op": "tariff.inlays", "input": {"source": "x", "offset": 1}}),
        json!({"op": "tariff.rename", "input": {"source": "x", "offset": 0, "extra": true}}),
        json!({"op": "tariff.signature", "input": {"source": "x", "offset": 0, "kind": "nope"}}),
        json!({"op": "tariff.definition", "input": {"offset": 0}}),
        json!({"op": "tariff.references", "input": {"source": "x", "offset": 0, "includeDeclaration": "yes"}}),
        json!({"op": "tariff.references", "input": {"source": "x"}}),
        json!({"op": "tariff.actions", "input": {"source": "x", "offset": 0}}),
        json!({"op": "tariff.actions", "input": {"source": "x", "range": {"from": 0, "to": 1}, "newName": "a"}}),
    ] {
        let response: Json =
            serde_json::from_slice(&dispatch(request.to_string().as_bytes())).unwrap_or(Json::Null);
        assert_eq!(response["error"]["code"], "invalid_input", "{request}");
    }
}

#[test]
fn offsets_past_the_end_are_clamped() {
    let request = json!({"op": "tariff.complete", "input": {"source": "tariff \"t\"\nlet a = shipment.", "offset": 9_999}});
    let response: Json =
        serde_json::from_slice(&dispatch(request.to_string().as_bytes())).unwrap_or(Json::Null);
    let labels: Vec<&str> = response["ok"]["items"]
        .as_array()
        .into_iter()
        .flatten()
        .filter_map(|item| item["label"].as_str())
        .collect();
    assert!(labels.contains(&"weight"), "{response}");
}

const OPS: &[&str] = &[
    "complete",
    "hover",
    "signature",
    "definition",
    "rename",
    "inlays",
    "references",
    "actions",
];

proptest::proptest! {
    #![proptest_config(proptest::prelude::ProptestConfig::with_cases(128))]

    #[test]
    fn no_operation_panics_on_arbitrary_text(source in "\\PC{0,120}", offset in 0u32..140, op in 0usize..8) {
        let name = OPS.get(op).copied().unwrap_or("complete");
        let input = json!({"source": source, "offset": offset, "newName": "x", "range": {"from": offset, "to": offset + 40}, "values": [{"kind": "bind", "name": "a", "display": "1", "range": {"from": 0, "to": 3}}]});
        let mut input = input;
        if let Some(object) = input.as_object_mut() {
            match name {
                "inlays" => { object.remove("offset"); object.remove("newName"); }
                "actions" => { object.remove("offset"); object.remove("newName"); object.remove("values"); }
                "rename" => { object.remove("range"); object.remove("values"); }
                "hover" => { object.remove("range"); object.remove("newName"); }
                _ => { object.remove("range"); object.remove("newName"); object.remove("values"); }
            }
        }
        let request = json!({"op": format!("tariff.{name}"), "input": input});
        let response: Json = serde_json::from_slice(&dispatch(request.to_string().as_bytes())).unwrap_or(Json::Null);
        proptest::prop_assert!(response.get("ok").is_some(), "{}", response);
    }

    #[test]
    fn no_feature_panics_on_a_cut_example(start in 0usize..1800, length in 0usize..80) {
        let example = first_example();
        let start = start.min(example.len());
        let end = (start + length).min(example.len());
        if let (Some(head), Some(tail)) = (example.get(..start), example.get(end..)) {
            let cut = format!("{head}{tail}");
            let context: crate::tariff::context::Context = serde_json::from_value(context()).unwrap_or_default();
            let session = super::session::Session::new(&cut, None, &context);
            let at = u32::try_from(start).unwrap_or(0);
            let _ = super::complete::complete(&session, at);
            let _ = super::hover::hover(&session, at, &super::values::Values::default());
            let _ = super::signature::signature(&session, at);
            let _ = super::definition::definition(&session, at);
            let _ = super::rename::rename(&session, at, "renamed");
            let _ = super::references::references(&session, at, true);
            let _ = super::actions::actions(&session, trenova_module_kit::span::Span::new(at, at));
        }
    }
}

#[test]
fn the_check_carries_the_outline_and_folding_ranges() {
    let source = "tariff \"t\"\n\nuse std.fuel\nuse std.ltl.{ deficit }\n\n// Rates for the\n// north lane.\ninput rate: Money/mi = $1/mi\n\nfn fee(a: Money) -> Money {\n  return a\n}\n\ncharge \"Line\" = rate * shipment.distance + fee($1) + $0 * fuel.perMile($1/gal, peg: $1/gal, mpg: 6) * 1 mi\n\ntest \"bills\" {\n  expect total > $0\n}\n";
    let request = json!({"op": "tariff.check", "input": {"source": source}});
    let response: Json =
        serde_json::from_slice(&dispatch(request.to_string().as_bytes())).unwrap_or(Json::Null);
    let outline = &response["ok"]["outline"];
    let kinds: Vec<&str> = outline["items"]
        .as_array()
        .into_iter()
        .flatten()
        .filter_map(|item| item["kind"].as_str())
        .collect();
    assert_eq!(
        kinds,
        ["header", "input", "function", "charge", "test"],
        "{outline}"
    );
    let mut folding: Vec<Json> = outline["folding"].as_array().cloned().unwrap_or_default();
    for fold in &mut folding {
        texts(fold, source);
    }
    let folded: Vec<(String, String)> = folding
        .iter()
        .map(|fold| {
            (
                fold["kind"].as_str().unwrap_or_default().to_owned(),
                fold["range"]["text"]
                    .as_str()
                    .unwrap_or_default()
                    .to_owned(),
            )
        })
        .collect();
    assert!(
        folded.contains(&(
            String::from("imports"),
            String::from("use std.fuel\nuse std.ltl.{ deficit }")
        )),
        "{folded:?}"
    );
    assert!(
        folded.contains(&(
            String::from("comment"),
            String::from("// Rates for the\n// north lane.")
        )),
        "{folded:?}"
    );
    assert!(
        folded
            .iter()
            .any(|(kind, text)| kind == "block" && text.starts_with("{\n  return a")),
        "{folded:?}"
    );
}
