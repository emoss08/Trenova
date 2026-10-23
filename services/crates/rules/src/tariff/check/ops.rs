//! The `tariff.*` operations this area answers: `tariff.check` and `tariff.catalog`.
//!
//! `tariff.check` input: `{"source": "…", "kind": "tariff"|"library"|"condition" (optional),
//! "context": Context (optional), "values": {"rule": {name: wire value}, "agreement": {…}}
//! (optional)}`. Output: `{"ok", "header": {"kind","name"} | null, "diagnostics": [...],
//! "inputs": [...], "imports": [{"path","version"}], "tables": [{"code","version"}], "outline":
//! ..., "charges": [{"label","code","range"}]}`. `tables` lists every rate table the program
//! reads (the root or a library it imports), in code order, with the version its shape was sent
//! at (`null` when none), so the host can pin it. Syntax and loader findings come first, then the checker's, each sorted
//! by where it starts; a finding in an imported library carries `"module"` and a range in that
//! library's source. A finding about a contract input value has code `input-value`, a `"path"`
//! and no range.

use super::consts::wire;
use super::inputs::validate;
use super::{Analysis, InputInfo};
use crate::ops::{OpResult, RequestError, decode, encode};
use crate::span::{Span, Utf16Index};
use crate::tariff::context::Context;
use crate::tariff::diagnostic::{Diagnostic, WireDiagnostic};
use crate::tariff::syntax::ast::{FileKind, ItemKind, Stmt, StmtKind};
use crate::tariff::syntax::program::{ModuleId, Program, load};
use serde::Deserialize;
use serde_json::value::RawValue;
use serde_json::{Map, Value as Json, json};

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct CheckInput {
    source: String,
    #[serde(default)]
    kind: Option<String>,
    #[serde(default)]
    context: Context,
    #[serde(default)]
    values: Option<Values>,
}

#[derive(Deserialize, Default)]
#[serde(deny_unknown_fields)]
struct Values {
    #[serde(default)]
    rule: Option<Map<String, Json>>,
    #[serde(default)]
    agreement: Option<Map<String, Json>>,
}

/// The file kind a request names: `tariff`, `library` or `condition`, or none.
///
/// # Errors
///
/// When the text names no kind.
pub fn kind(text: Option<&str>) -> Result<Option<FileKind>, RequestError> {
    match text {
        None => Ok(None),
        Some("tariff") => Ok(Some(FileKind::Tariff)),
        Some("library") => Ok(Some(FileKind::Library)),
        Some("condition") => Ok(Some(FileKind::Condition)),
        Some(other) => Err(RequestError::invalid_input(format!(
            "kind must be tariff, library or condition, not {other:?}"
        ))),
    }
}

/// `tariff.check`.
///
/// # Errors
///
/// When the input does not match the operation's shape.
pub fn check(input: &RawValue) -> OpResult {
    let input: CheckInput = decode(input)?;
    let expected = kind(input.kind.as_deref())?;
    let program = load(&input.source, expected, &input.context);
    let analysis = super::check(&program, &input.context);
    let values = input.values.unwrap_or_default();
    let output = output(
        &program,
        &analysis,
        &input.context,
        values.rule.as_ref(),
        values.agreement.as_ref(),
    );
    encode(&output)
}

fn output(
    program: &Program,
    analysis: &Analysis,
    context: &Context,
    rule: Option<&Map<String, Json>>,
    agreement: Option<&Map<String, Json>>,
) -> Json {
    let mut diagnostics: Vec<Json> = Vec::new();
    for (module, diagnostic) in &program.diagnostics {
        diagnostics.push(wire_diagnostic(program, *module, diagnostic));
    }
    for (module, diagnostic) in analysis.diagnostics() {
        diagnostics.push(wire_diagnostic(program, module, diagnostic));
    }
    let findings = if rule.is_some() || agreement.is_some() {
        validate(&analysis.inputs, rule, agreement)
    } else {
        Vec::new()
    };
    for finding in &findings {
        diagnostics.push(json!({
            "severity": "error",
            "code": "input-value",
            "path": finding.path,
            "message": finding.message,
            "fixes": [],
            "related": [],
        }));
    }
    let root = program.root();
    let utf16 = root.map(|module| &module.utf16);
    let header = root
        .and_then(|module| module.file.header.as_ref())
        .map(|header| json!({ "kind": header.kind.keyword(), "name": header.name.text }));
    json!({
        "ok": analysis.ok && findings.is_empty(),
        "header": header,
        "diagnostics": diagnostics,
        "inputs": analysis.inputs.iter().map(|input| input_json(input, utf16)).collect::<Vec<_>>(),
        "imports": analysis.imports.iter().map(|import| json!({ "path": import.path, "version": import.version })).collect::<Vec<_>>(),
        "tables": tables(analysis, context),
        "outline": outline(program),
        "charges": root.map(charges).unwrap_or_default(),
    })
}

/// Every rate table the program reads, by code, with the version the context sent it at (`null`
/// when it sent none): `[{"code", "version"}]`.
fn tables(analysis: &Analysis, context: &Context) -> Vec<Json> {
    analysis
        .tables()
        .into_iter()
        .map(|code| {
            let version = context
                .tables
                .iter()
                .find(|shape| shape.code == code)
                .and_then(|shape| shape.version);
            json!({ "code": code, "version": version })
        })
        .collect()
}

fn range(utf16: Option<&Utf16Index>, span: Span) -> Json {
    let range = utf16.map(|index| index.range(span)).unwrap_or_default();
    json!({ "from": range.from, "to": range.to })
}

fn input_json(input: &InputInfo, utf16: Option<&Utf16Index>) -> Json {
    json!({
        "name": input.name,
        "doc": input.doc,
        "type": input.ty.to_string(),
        "descriptor": input.descriptor,
        "required": input.required,
        "default": input.default,
        "constraint": input.constraint.as_ref().map_or(Json::Null, crate::tariff::constraint::Constraint::to_json),
        "range": range(utf16, input.span),
    })
}

/// A finding on the wire, in the shared form ([`WireDiagnostic`]) with UTF-16 ranges in its
/// module's source, plus the module's path when it is not the root.
fn wire_diagnostic(program: &Program, module: ModuleId, diagnostic: &Diagnostic) -> Json {
    let found = program.module(module);
    let fallback = Utf16Index::new("");
    let index = found.map_or(&fallback, |module| &module.utf16);
    let mut out =
        serde_json::to_value(WireDiagnostic::new(diagnostic, index)).unwrap_or(Json::Null);
    if let (Some(path), Some(object)) = (
        found.and_then(|module| module.path.clone()),
        out.as_object_mut(),
    ) {
        object.insert(String::from("module"), Json::String(path));
    }
    out
}

/// The root's outline and folding ranges: `{"items": [{"kind", "name", "code", "depth",
/// "range", "selection"}], "folding": [{"range", "kind"}]}`.
fn outline(program: &Program) -> Json {
    let Some(root) = program.root() else {
        return Json::Null;
    };
    let found = crate::tariff::syntax::outline::outline(&root.source, &root.file);
    let utf16 = Some(&root.utf16);
    json!({
        "items": found.items.iter().map(|item| json!({
            "kind": item.kind.as_str(),
            "name": item.name,
            "code": item.code,
            "depth": item.depth,
            "range": range(utf16, item.span),
            "selection": range(utf16, item.selection),
        })).collect::<Vec<_>>(),
        "folding": found.folding.iter().map(|(span, kind)| json!({
            "range": range(utf16, *span),
            "kind": kind.as_str(),
        })).collect::<Vec<_>>(),
    })
}

/// Every charge statement of the root, nested ones included, with its label as written and
/// its code.
fn charges(module: &crate::tariff::syntax::program::Module) -> Vec<Json> {
    let mut out = Vec::new();
    for item in &module.file.items {
        if let ItemKind::Stmt(stmt) = &item.kind {
            collect(stmt, module, &mut out);
        }
    }
    out
}

fn collect(stmt: &Stmt, module: &crate::tariff::syntax::program::Module, out: &mut Vec<Json>) {
    super::visit::stmt(
        stmt,
        &mut |inner| {
            if let StmtKind::Charge(charge) = &inner.kind {
                out.push(json!({
                    "label": super::program::label_text(&module.source, &charge.label),
                    "code": charge.code.as_ref().map(|code| code.text.clone()),
                    "range": range(Some(&module.utf16), inner.span),
                }));
            }
        },
        &mut |_| {},
    );
}

/// `tariff.catalog`.
///
/// # Errors
///
/// When the input does not match the operation's shape.
pub fn catalog(input: &RawValue) -> OpResult {
    let _: Map<String, Json> = decode(input)?;
    encode(&super::catalog::catalog())
}

/// A default's wire value, for callers that hold a folded value.
#[must_use]
pub fn default_wire(value: &super::consts::Value) -> Json {
    wire(value)
}

/// The output of `tariff.check` for a loaded and checked program, for the corpus tests.
#[cfg(test)]
pub fn output_for_tests(program: &Program, analysis: &Analysis, context: &Context) -> Json {
    output(program, analysis, context, None, None)
}
