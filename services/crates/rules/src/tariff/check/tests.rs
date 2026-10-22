//! The checker's corpus (`testdata/tariff/check/*.json`) and its end-to-end tests.
//!
//! A corpus file is `{"cases": [case, …]}`. A case gives `source` (a string or a list of
//! lines), optionally `kind`, `context` (as `tariff.check` takes it) and `values`, and the
//! findings it must produce, all of them, in order of position: `{"code", "at": "snippet"
//! (its first occurrence, or the `nth`), "message": "English text", "severity" (when not
//! the code's usual one), "module" (for a finding in a library), "fixes": [{"label",
//! "result": source after the fix}]}`. A finding without `at` is at the start of the file. A
//! case may also pin `types` (`{"snippet": "Type"}`: the type of the expression written so)
//! and `inputs` (the `inputs` of `tariff.check`'s output).

use super::fixes::apply;
use crate::tariff::context::Context;
use crate::tariff::diagnostic::{Diagnostic, Severity};
use crate::tariff::syntax::program::{ModuleId, load};
use serde_json::{Value as Json, json};

const CORPUS: &[(&str, &str)] = &[
    (
        "types",
        include_str!("../../../testdata/tariff/check/types.json"),
    ),
    (
        "names",
        include_str!("../../../testdata/tariff/check/names.json"),
    ),
    (
        "operators",
        include_str!("../../../testdata/tariff/check/operators.json"),
    ),
    (
        "optionals",
        include_str!("../../../testdata/tariff/check/optionals.json"),
    ),
    (
        "methods",
        include_str!("../../../testdata/tariff/check/methods.json"),
    ),
    (
        "calls",
        include_str!("../../../testdata/tariff/check/calls.json"),
    ),
    (
        "tables",
        include_str!("../../../testdata/tariff/check/tables.json"),
    ),
    (
        "statements",
        include_str!("../../../testdata/tariff/check/statements.json"),
    ),
    (
        "charges",
        include_str!("../../../testdata/tariff/check/charges.json"),
    ),
    (
        "match",
        include_str!("../../../testdata/tariff/check/match.json"),
    ),
    (
        "files",
        include_str!("../../../testdata/tariff/check/files.json"),
    ),
    (
        "tests",
        include_str!("../../../testdata/tariff/check/tests.json"),
    ),
    (
        "modules",
        include_str!("../../../testdata/tariff/check/modules.json"),
    ),
    (
        "inputs",
        include_str!("../../../testdata/tariff/check/inputs.json"),
    ),
];

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

fn kind(value: Option<&Json>) -> Option<crate::tariff::syntax::ast::FileKind> {
    use crate::tariff::syntax::ast::FileKind;
    match value.and_then(Json::as_str) {
        Some("library") => Some(FileKind::Library),
        Some("condition") => Some(FileKind::Condition),
        Some("tariff") => Some(FileKind::Tariff),
        _ => None,
    }
}

/// One finding as the corpus writes it, for comparison and for the failure message.
fn written(source: &str, module: Option<&str>, diagnostic: &Diagnostic) -> Json {
    let start = usize::try_from(diagnostic.span.start).unwrap_or(0);
    let end = usize::try_from(diagnostic.span.end).unwrap_or(0);
    let at = source.get(start..end).unwrap_or_default();
    let nth = if at.is_empty() {
        1
    } else {
        source.get(..start).unwrap_or_default().matches(at).count() + 1
    };
    let mut out = json!({
        "code": diagnostic.code,
        "at": at,
        "message": diagnostic.message.english(),
        "fixes": diagnostic.fixes.iter().map(|fix| json!({
            "label": fix.label.english(),
            "result": apply(source, fix),
        })).collect::<Vec<_>>(),
    });
    if let Some(object) = out.as_object_mut() {
        if nth > 1 {
            object.insert("nth".into(), json!(nth));
        }
        if diagnostic.severity != usual(diagnostic.code) {
            object.insert("severity".into(), json!(diagnostic.severity.as_str()));
        }
        if let Some(module) = module {
            object.insert("module".into(), json!(module));
        }
        if !diagnostic.related.is_empty() {
            let related: Vec<Json> = diagnostic
                .related
                .iter()
                .map(|(span, message)| {
                    let start = usize::try_from(span.start).unwrap_or(0);
                    let end = usize::try_from(span.end).unwrap_or(0);
                    json!({ "at": source.get(start..end).unwrap_or_default(), "line": source.get(..start).unwrap_or_default().matches('\n').count() + 1, "message": message.english() })
                })
                .collect();
            object.insert("related".into(), Json::Array(related));
        }
    }
    out
}

/// The severity a code usually has, so the corpus only spells out the others.
fn usual(code: &str) -> Severity {
    const WARNINGS: &[&str] = &[
        "unused",
        "unused-value",
        "no-charge",
        "needless-fallback",
        "needless-optional-chain",
        "needless-let",
        "unreachable",
        "unreachable-arm",
        "not-billed",
        "doc",
    ];
    if WARNINGS.contains(&code) {
        Severity::Warning
    } else {
        Severity::Error
    }
}

fn normalise(expected: &Json) -> Json {
    let mut out = expected.clone();
    if let Some(object) = out.as_object_mut() {
        object.entry("at").or_insert(json!(""));
        let fixes: Vec<Json> = object
            .get("fixes")
            .and_then(Json::as_array)
            .map(|fixes| {
                fixes
                    .iter()
                    .map(|fix| json!({ "label": fix.get("label").cloned().unwrap_or(Json::Null), "result": text(fix.get("result").unwrap_or(&Json::Null)) }))
                    .collect()
            })
            .unwrap_or_default();
        object.insert("fixes".into(), Json::Array(fixes));
        if object.get("nth") == Some(&json!(1)) {
            object.remove("nth");
        }
    }
    out
}

fn run_case(file: &str, case: &Json) -> Option<String> {
    let name = case.get("name").and_then(Json::as_str).unwrap_or("?");
    let source = text(case.get("source").unwrap_or(&Json::Null));
    let context: Context = case
        .get("context")
        .map(|value| {
            serde_json::from_value(value.clone())
                .unwrap_or_else(|error| panic!("{file}/{name}: bad context: {error}"))
        })
        .unwrap_or_default();
    let program = load(&source, kind(case.get("kind")), &context);
    let analysis = super::check(&program, &context);
    let mut actual: Vec<Json> = Vec::new();
    let mut all: Vec<(ModuleId, &Diagnostic)> =
        program.diagnostics.iter().map(|(m, d)| (*m, d)).collect();
    all.extend(analysis.diagnostics());
    all.sort_by_key(|(module, diagnostic)| {
        (
            module.0 != 0,
            *module,
            diagnostic.span.start,
            diagnostic.span.end,
        )
    });
    for (module, diagnostic) in all {
        let found = program.module(module);
        let module_source = found.map_or("", |m| m.source.as_str());
        let path = found
            .and_then(|m| m.path.as_deref())
            .filter(|_| module.0 != 0);
        actual.push(written(module_source, path, diagnostic));
    }
    let expected: Vec<Json> = case
        .get("diagnostics")
        .and_then(Json::as_array)
        .map(|list| list.iter().map(normalise).collect())
        .unwrap_or_default();
    let mut problems = Vec::new();
    for input in &analysis.inputs {
        let Some(default) = &input.default else {
            continue;
        };
        match crate::tariff::wire::decode(default, &input.ty, &input.name) {
            Ok(value) if crate::tariff::wire::encode(&value) == *default => {}
            Ok(value) => problems.push(format!(
                "the default of {} is {default}, but the codec writes it {}",
                input.name,
                crate::tariff::wire::encode(&value)
            )),
            Err(error) => problems.push(format!(
                "the default of {} does not decode: {}",
                input.name,
                error.message.english()
            )),
        }
    }
    if actual != expected {
        problems.push(format!(
            "diagnostics differ\nexpected: {}\nactual:   {}",
            serde_json::to_string_pretty(&expected).unwrap_or_default(),
            serde_json::to_string_pretty(&actual).unwrap_or_default()
        ));
    }
    if let Some(types) = case.get("types").and_then(Json::as_object) {
        for (snippet, want) in types {
            let got = type_at(&program, &analysis, &source, snippet);
            if got.as_deref() != want.as_str() {
                problems.push(format!("type of `{snippet}`: expected {want}, got {got:?}"));
            }
        }
    }
    if let Some(inputs) = case.get("inputs") {
        let out = super::ops::output_for_tests(&program, &analysis);
        if out.get("inputs") != Some(inputs) {
            problems.push(format!(
                "inputs differ\nexpected: {inputs}\nactual:   {}",
                out.get("inputs").cloned().unwrap_or(Json::Null)
            ));
        }
    }
    (!problems.is_empty()).then(|| format!("{file}/{name}:\n{}", problems.join("\n")))
}

/// The type the checker recorded for the expression whose source is exactly `snippet`.
fn type_at(
    program: &crate::tariff::syntax::program::Program,
    analysis: &super::Analysis,
    source: &str,
    snippet: &str,
) -> Option<String> {
    let root = program.root()?;
    let mut found = None;
    for item in &root.file.items {
        let mut visit = |expr: &crate::tariff::syntax::ast::Expr| {
            let start = usize::try_from(expr.span.start).unwrap_or(0);
            let end = usize::try_from(expr.span.end).unwrap_or(0);
            let mut written = source.get(start..end).unwrap_or_default();
            for _ in 0..expr.parens {
                written = written
                    .strip_prefix('(')
                    .and_then(|inner| inner.strip_suffix(')'))
                    .unwrap_or(written)
                    .trim();
            }
            if found.is_none() && written == snippet {
                found = analysis
                    .type_of(ModuleId::ROOT, expr.id)
                    .map(ToString::to_string);
            }
        };
        visit_item(item, &mut visit);
    }
    found
}

fn visit_item(
    item: &crate::tariff::syntax::ast::Item,
    visit: &mut impl FnMut(&crate::tariff::syntax::ast::Expr),
) {
    use crate::tariff::syntax::ast::{FnBody, ItemKind};
    match &item.kind {
        ItemKind::Stmt(stmt) => super::visit::stmt(stmt, &mut |_| {}, visit),
        ItemKind::Fn(decl) => match &decl.body {
            FnBody::Expr(value) => super::visit::expr(value, visit),
            FnBody::Block(block) => super::visit::block(block, &mut |_| {}, visit),
        },
        ItemKind::Test(test) => super::visit::block(&test.body, &mut |_| {}, visit),
        ItemKind::Const(decl) => super::visit::expr(&decl.value, visit),
        ItemKind::Input(decl) => {
            if let Some(default) = &decl.default {
                super::visit::expr(default, visit);
            }
        }
        _ => {}
    }
}

#[test]
fn the_corpus_holds() {
    let mut failures = Vec::new();
    let mut count = 0;
    for (file, json) in CORPUS {
        let parsed: Json =
            serde_json::from_str(json).unwrap_or_else(|error| panic!("{file}: {error}"));
        for case in parsed
            .get("cases")
            .and_then(Json::as_array)
            .into_iter()
            .flatten()
        {
            count += 1;
            if let Some(failure) = run_case(file, case) {
                failures.push(failure);
            }
        }
    }
    assert!(
        failures.is_empty(),
        "{} of {count} cases failed:\n\n{}",
        failures.len(),
        failures.join("\n\n")
    );
    assert!(count >= 300, "the corpus has {count} cases");
}

const SPEC: &str = include_str!("../../../../../../docs/engineering/formula-language.md");

/// Every ```` ```tariff ```` block of the reference, in order.
fn spec_blocks() -> Vec<String> {
    let mut out = Vec::new();
    let mut rest = SPEC;
    while let Some(start) = rest.find("```tariff\n") {
        let body = &rest[start + "```tariff\n".len()..];
        let Some(end) = body.find("\n```") else { break };
        out.push(body[..end].to_owned());
        rest = &body[end..];
    }
    out
}

/// The libraries and codes the reference's examples use. `std.fuel` and
/// `std.accessorials` are supplied as organisation libraries with the same declarations, since
/// the standard library's sources are written elsewhere.
fn spec_context() -> Context {
    serde_json::from_value(json!({
        "codes": [
            {"code": "FSC", "active": true},
            {"code": "XDROP", "active": true},
            {"code": "HAZ", "active": true},
            {"code": "DET", "active": true},
        ],
        "libraries": [
            {"path": "org.accessorials", "source": "library \"org.accessorials\"\n\n/// Hazmat handling per loaded mile.\nfn hazmat(miles: Distance) -> Money = $0.10/mi * miles\n"},
            {"path": "org.fuel", "source": "library \"org.fuel\"\n\n/// Whole steps of `step` above `peg`, each worth `perStep`.\nfn stepPercent(price: Money/gal, peg: Money/gal, step: Money/gal, perStep: Percent) -> Percent {\n  let steps = ((price - peg) / step).floor().max(0)\n  return perStep * steps\n}\n"},
            {"path": "org.detention", "source": "library \"org.detention\"\n\n/// Time past `free`, rounded up to `increment`, at `rate`.\nfn detention(dwell: Duration, free: Duration = 2 h, rate: Money/h, increment: Duration = 15 min) -> Money {\n  let billable = (dwell - free).max(0 h).roundUp(to: increment)\n  return rate * billable\n}\n"},
        ],
    }))
    .unwrap_or_default()
}

/// Each block as a whole file: fragments get the header and the bindings they assume.
fn spec_file(index: usize, block: &str) -> String {
    match index {
        0 => block
            .replace(
                "use std.fuel.{ stepPercent }",
                "use org.fuel.{ stepPercent }",
            )
            .replace(
                "use std.accessorials.{ detention }",
                "use org.detention.{ detention }",
            ),
        1 => format!("tariff \"t\"\n{block}\ncharge \"x\" = perCwt * (shipment.weight ?? 0 lb)\n"),
        2 => format!(
            "tariff \"t\"\n{}\ncharge \"x\" = $1 * (deliveries.len + palletCount) + $1/lb * (heaviest ?? 0 lb)\n",
            block.replace("c.pallets", "c.pieces")
        ),
        3 => format!("library \"t\"\n{block}\n"),
        4 => {
            let mut lines: Vec<&str> = block.lines().collect();
            let last = lines.pop().unwrap_or_default();
            format!(
                "tariff \"t\"\ninput rate: Money/mi = $2/mi\ninput fuelPercent: Percent = 10%\nlet miles = shipment.distance\nlet linehaul = rate * miles\n{}\nfor stop in shipment.stops {{\n  let billable = stop.dwell ?? 0 h\n  {last}\n}}\n",
                lines.join("\n")
            )
        }
        6 => format!(
            "tariff \"t\"\nrequire let w = shipment.weight else \"This rate needs the shipment's weight\"\ncharge \"Linehaul\" = $1/lb * w\n{block}\n"
        ),
        _ => block.to_owned(),
    }
}

#[test]
fn the_references_examples_check_clean() {
    let blocks = spec_blocks();
    assert_eq!(blocks.len(), 7, "the reference's tariff blocks changed");
    let context = spec_context();
    for (index, block) in blocks.iter().enumerate() {
        let source = spec_file(index, block);
        let program = load(&source, None, &context);
        let analysis = super::check(&program, &context);
        let findings: Vec<Json> = program
            .diagnostics
            .iter()
            .map(|(m, d)| (*m, d))
            .chain(analysis.diagnostics())
            .map(|(module, diagnostic)| {
                written(
                    program.module(module).map_or("", |m| m.source.as_str()),
                    None,
                    diagnostic,
                )
            })
            .collect();
        assert!(
            findings.is_empty(),
            "block {index}:\n{source}\n{findings:#?}"
        );
        assert!(analysis.ok, "block {index}");
    }
}

#[test]
fn every_standard_library_module_checks_without_a_finding() {
    let context = Context::default();
    for (path, source) in crate::tariff::stdlib::MODULES {
        let program = load(
            source,
            Some(crate::tariff::syntax::ast::FileKind::Library),
            &context,
        );
        let analysis = super::check(&program, &context);
        let findings: Vec<String> = program
            .diagnostics
            .iter()
            .map(|(m, d)| (*m, d))
            .chain(analysis.diagnostics())
            .map(|(_, diagnostic)| format!("{}: {}", diagnostic.code, diagnostic.message.english()))
            .collect();
        assert!(findings.is_empty(), "{path}: {findings:#?}");
    }
}

/// Whether `actual` holds everything `expected` says: objects by the keys written, arrays item
/// by item and of the same length, everything else equal.
fn holds(actual: &Json, expected: &Json, path: &str, problems: &mut Vec<String>) {
    match (actual, expected) {
        (Json::Object(actual), Json::Object(expected)) => {
            for (key, value) in expected {
                match actual.get(key) {
                    Some(found) => holds(found, value, &format!("{path}.{key}"), problems),
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

#[test]
fn the_operations_answer_through_dispatch() {
    let cases: Json = serde_json::from_str(include_str!("../../../testdata/tariff/check/ops.json"))
        .unwrap_or_else(|error| panic!("ops.json: {error}"));
    let mut failures = Vec::new();
    for case in cases
        .get("cases")
        .and_then(Json::as_array)
        .into_iter()
        .flatten()
    {
        let name = case.get("name").and_then(Json::as_str).unwrap_or("?");
        let mut request = case.get("request").cloned().unwrap_or(Json::Null);
        if let Some(input) = request.get_mut("input").and_then(Json::as_object_mut)
            && let Some(source) = input
                .get("source")
                .filter(|source| source.is_array())
                .cloned()
        {
            input.insert("source".into(), Json::String(text(&source)));
        }
        let response = crate::ops::dispatch(request.to_string().as_bytes());
        let response: Json = serde_json::from_slice(&response).unwrap_or(Json::Null);
        let mut problems = Vec::new();
        holds(
            &response,
            case.get("response").unwrap_or(&Json::Null),
            "",
            &mut problems,
        );
        if !problems.is_empty() {
            failures.push(format!(
                "{name}:\n  {}\n  response: {response}",
                problems.join("\n  ")
            ));
        }
    }
    assert!(failures.is_empty(), "{}", failures.join("\n\n"));
}

#[test]
fn the_analysis_resolves_symbols_references_hints_and_callees() {
    let context: Context = serde_json::from_value(json!({
        "libraries": [{"path": "org.fees", "source": "library \"org.fees\"\n\n/// Fees.\n\n/// The fee.\nfn fee(amount: Money = $5) -> Money = amount\n"}]
    }))
    .unwrap_or_default();
    let source = "tariff \"t\"\nuse org.fees.{ fee }\nlet miles = shipment.distance\nlet n = shipment.stops.count(s => s.kind == Delivery)\ncharge \"x\" = $1/mi * miles + fee() + $1 * n\ntest \"it bills\" {\n  expect total > $0\n}";
    let program = load(source, None, &context);
    let analysis = super::check(&program, &context);
    assert!(
        analysis.ok,
        "{:?}",
        analysis.diagnostics().collect::<Vec<_>>()
    );
    let miles = analysis
        .symbols
        .iter()
        .position(|symbol| symbol.name == "miles" && symbol.kind == super::SymbolKind::Let)
        .map(|index| super::SymbolId(u32::try_from(index).unwrap_or(0)));
    let uses: Vec<_> = analysis
        .references
        .iter()
        .filter(|reference| Some(reference.symbol) == miles)
        .collect();
    assert_eq!(uses.len(), 1);
    let fee = analysis
        .symbols
        .iter()
        .find(|symbol| symbol.name == "fee" && symbol.kind == super::SymbolKind::Function)
        .cloned();
    assert_eq!(
        fee.as_ref()
            .and_then(|symbol| symbol.doc.clone())
            .as_deref(),
        Some("The fee.")
    );
    assert_ne!(fee.and_then(|symbol| symbol.module), Some(ModuleId::ROOT));
    assert!(
        analysis
            .symbols
            .iter()
            .any(|symbol| symbol.kind == super::SymbolKind::Test && symbol.name == "it bills")
    );
    let root: Vec<(String, String)> = analysis
        .module(ModuleId::ROOT)
        .map(|module| {
            module
                .hints
                .iter()
                .map(|hint| {
                    (
                        super::describe::snippet(source, hint.span).to_owned(),
                        hint.ty.to_string(),
                    )
                })
                .collect()
        })
        .unwrap_or_default();
    assert!(
        root.contains(&(String::from("miles"), String::from("Distance"))),
        "{root:?}"
    );
    assert!(
        root.contains(&(String::from("s"), String::from("Stop"))),
        "{root:?}"
    );
    let callees: Vec<String> = analysis
        .module(ModuleId::ROOT)
        .map(|module| {
            module
                .callees
                .values()
                .map(|callee| format!("{callee:?}"))
                .collect()
        })
        .unwrap_or_default();
    assert!(
        callees
            .iter()
            .any(|callee| callee.contains("name: \"fee\"")),
        "{callees:?}"
    );
    assert!(
        callees.iter().any(|callee| callee.contains("count")),
        "{callees:?}"
    );
    let variants = analysis
        .module(ModuleId::ROOT)
        .map_or(0, |module| module.variants.len());
    assert_eq!(variants, 1);
}

const TOKENS: &[&str] = &[
    "tariff \"t\"\n",
    "library \"l\"\n",
    "condition \"c\"\n",
    "let ",
    "var ",
    "x",
    "y",
    " = ",
    "charge \"c\"",
    " as FSC",
    " (quantity: ",
    "rate: ",
    ")",
    "(",
    " = ",
    "$5",
    "$2/mi",
    "5 lb",
    "1_200",
    "2.5",
    "18%",
    "@2026-10-22",
    "shipment",
    ".",
    "?.",
    "weight",
    "stops",
    "distance",
    " ?? ",
    " fail \"no\"",
    "\n",
    " + ",
    " * ",
    " / ",
    " - ",
    " == ",
    " != ",
    " < ",
    " and ",
    " or ",
    "not ",
    "none",
    "true",
    "[",
    "]",
    ", ",
    "{",
    "}",
    "if ",
    "else ",
    "match ",
    " => ",
    "..",
    "..=",
    "_",
    "for ",
    " in ",
    " where ",
    "fn f(a: Weight) -> Money",
    "Money/cwt",
    "Weight?",
    "List<Stop>",
    "s => s.kind",
    "Delivery",
    "New",
    "return ",
    "require let w = shipment.weight else \"x\"",
    "minimum $5",
    "round to cents",
    "test \"t\" {",
    "given shipment.weight = 1 lb",
    "expect total > $0",
    "tables.Zones[",
    "x.in(cwt)",
    "\"{x}\"",
    "use std.fuel",
    "const K: Money = $1",
    "type T = { a: Int }",
    "enum E { A, B }",
    "input r: Money/mi = $1/mi",
];

proptest::proptest! {
    #![proptest_config(proptest::prelude::ProptestConfig::with_cases(256))]

    #[test]
    fn checking_arbitrary_text_never_panics(source in "\\PC{0,160}") {
        let context = Context::default();
        let program = load(&source, None, &context);
        let _ = super::check(&program, &context);
    }

    #[test]
    fn checking_token_soup_never_panics(picks in proptest::collection::vec(0..TOKENS.len(), 0..80)) {
        let source: String = picks.iter().filter_map(|index| TOKENS.get(*index)).copied().collect();
        let context = spec_context();
        let program = load(&source, None, &context);
        let _ = super::check(&program, &context);
    }

    #[test]
    fn checking_a_cut_example_never_panics(start in 0usize..2000, length in 0usize..60) {
        let blocks = spec_blocks();
        let source = spec_file(0, blocks.first().map_or("", String::as_str));
        let start = start.min(source.len());
        let end = (start + length).min(source.len());
        let (Some(head), Some(tail)) = (source.get(..start), source.get(end..)) else {
            return Ok(());
        };
        let cut = format!("{head}{tail}");
        let context = spec_context();
        let program = load(&cut, None, &context);
        let _ = super::check(&program, &context);
        let request = json!({"op": "tariff.check", "input": {"source": cut, "context": {"libraries": []}}});
        let _ = crate::ops::dispatch(request.to_string().as_bytes());
    }
}
