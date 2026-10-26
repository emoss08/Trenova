//! The `tariff.*` editor operations. Each takes the file being edited with its context and a
//! position or range in UTF-16 code units, loads and checks the file (it may be broken), and
//! answers from what is known. `kind` (`tariff`, `library`, `condition`) is optional, as for
//! `tariff.check`. A `doc` on the wire is a [`crate::tariff::docs::Doc`]: the author's words
//! (a markdown string: a doc comment in the file or an organisation library, a table's name, a
//! code's description) or text the module ships, which the hosts translate (a `Message`: the
//! built-ins', the schema's and the standard library's docs).
//!
//! - `tariff.complete` `{source, context, offset}` → `{"range": {from, to}, "items": [{"label",
//!   "kind", "detail", "doc", "insert", "snippet", "sort", "range"?}]}`, the items in `sort`
//!   order; `snippet` inserts use `${1:placeholder}` tab stops and `$0` for the final cursor.
//! - `tariff.hover` `{source, context, offset, values?}` → `null`, or `{"range", "contents":
//!   {"signature", "type", "unit", "doc", "value"?: {"value", "display", "times"},
//!   "constraint"?: Message}}`; `constraint` is an input's `where` clause in words (`from
//!   $0.50/mi to $15/mi`).
//! - `tariff.signature` `{source, context, offset}` → `{"signatures": [{"label", "doc",
//!   "parameters": [{"label": [from, to], "doc"}]}], "active": 0 | null, "activeParameter": n |
//!   null}`.
//! - `tariff.definition` `{source, context, offset}` → `{"targets": [{"module": null | "std.ltl",
//!   "range", "selectionRange"}]}`.
//! - `tariff.references` `{source, context, offset, includeDeclaration?}` → `{"references":
//!   [{"range", "kind": "declaration" | "read" | "write", "module": null | "org.fees"}]}`: the
//!   root file's occurrences in source order, after a declaration in a library (whose range is
//!   in that library's source); `includeDeclaration` (default true) false leaves it out.
//! - `tariff.actions` `{source, context, range: {from, to}}` → `{"actions": [{"id", "title":
//!   Message, "edits": [{"range", "insert"}], "select": {from, to} | null}]}`: the refactorings
//!   for the selection ([`super::actions`]); `select` is in the edited source.
//! - `tariff.rename` `{source, context, offset, newName}` → `{"edits": [{"range", "insert"}]}`
//!   or `{"error": Message}`; without `newName`, `{"range", "placeholder"}` or `{"error"}`.
//! - `tariff.inlays` `{source, context, range: {from, to}, values?}` → `{"hints": [{"offset",
//!   "label", "kind": "type" | "parameter" | "value"}]}`.
//!
//! `values` is the `trace` a `tariff.evaluate` of the same file returned ([`super::values`]).

use super::actions::actions as offered;
use super::complete::{Completion, complete as completions};
use super::definition::definition as definitions;
use super::hover::hover as hovered;
use super::inlays::inlays as hints;
use super::references::references as referenced;
use super::rename::{prepare, rename as renamed};
use super::session::Session;
use super::signature::signature as signatures;
use super::values::Values;
use crate::span::{Range, Span, Utf16Index};
use crate::tariff::check::ops::kind;
use crate::tariff::context::Context;
use serde::Deserialize;
use serde_json::value::RawValue;
use serde_json::{Value as Json, json};
use trenova_module_kit::ops::{OpResult, decode, encode};

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct AtInput {
    source: String,
    #[serde(default)]
    kind: Option<String>,
    #[serde(default)]
    context: Context,
    offset: u32,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct HoverInput {
    source: String,
    #[serde(default)]
    kind: Option<String>,
    #[serde(default)]
    context: Context,
    offset: u32,
    #[serde(default)]
    values: Option<Json>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct ReferencesInput {
    source: String,
    #[serde(default)]
    kind: Option<String>,
    #[serde(default)]
    context: Context,
    offset: u32,
    #[serde(default = "yes")]
    include_declaration: bool,
}

const fn yes() -> bool {
    true
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct ActionsInput {
    source: String,
    #[serde(default)]
    kind: Option<String>,
    #[serde(default)]
    context: Context,
    range: Range,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct RenameInput {
    source: String,
    #[serde(default)]
    kind: Option<String>,
    #[serde(default)]
    context: Context,
    offset: u32,
    #[serde(default)]
    new_name: Option<String>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct InlaysInput {
    source: String,
    #[serde(default)]
    kind: Option<String>,
    #[serde(default)]
    context: Context,
    range: Range,
    #[serde(default)]
    values: Option<Json>,
}

fn range(index: &Utf16Index, span: Span) -> Json {
    let range = index.range(span);
    json!({ "from": range.from, "to": range.to })
}

/// `tariff.complete`.
///
/// # Errors
///
/// When the input does not match the operation's shape.
pub fn complete(input: &RawValue) -> OpResult {
    let input: AtInput = decode(input)?;
    let session = Session::new(&input.source, kind(input.kind.as_deref())?, &input.context);
    let offset = session.byte(input.offset);
    encode(&completion_json(&session, &completions(&session, offset)))
}

fn completion_json(session: &Session<'_>, completion: &Completion) -> Json {
    let index = &session.root().utf16;
    let mut items: Vec<&super::complete::CompletionItem> = completion.items.iter().collect();
    items.sort_by(|a, b| a.sort.cmp(&b.sort).then_with(|| a.label.cmp(&b.label)));
    json!({
        "range": range(index, completion.range),
        "items": items.iter().map(|item| {
            let mut out = json!({
                "label": item.label,
                "kind": item.kind.as_str(),
                "detail": item.detail,
                "doc": item.doc,
                "insert": item.insert,
                "snippet": item.snippet,
                "sort": item.sort,
            });
            if let (Some(span), Some(object)) = (item.range, out.as_object_mut()) {
                object.insert(String::from("range"), range(index, span));
            }
            out
        }).collect::<Vec<_>>(),
    })
}

/// `tariff.hover`.
///
/// # Errors
///
/// When the input does not match the operation's shape.
pub fn hover(input: &RawValue) -> OpResult {
    let input: HoverInput = decode(input)?;
    let session = Session::new(&input.source, kind(input.kind.as_deref())?, &input.context);
    let offset = session.byte(input.offset);
    let values = Values::read(input.values.as_ref());
    let Some(found) = hovered(&session, offset, &values) else {
        return encode(&Json::Null);
    };
    let index = &session.root().utf16;
    let mut contents = json!({
        "signature": found.signature,
        "type": found.ty,
        "unit": found.unit,
        "doc": found.doc,
    });
    if let (Some(value), Some(object)) = (found.value, contents.as_object_mut()) {
        object.insert(String::from("value"), value);
    }
    if let (Some(constraint), Some(object)) = (found.constraint, contents.as_object_mut()) {
        object.insert(String::from("constraint"), json!(constraint));
    }
    encode(&json!({ "range": range(index, found.range), "contents": contents }))
}

/// `tariff.signature`.
///
/// # Errors
///
/// When the input does not match the operation's shape.
pub fn signature(input: &RawValue) -> OpResult {
    let input: AtInput = decode(input)?;
    let session = Session::new(&input.source, kind(input.kind.as_deref())?, &input.context);
    let offset = session.byte(input.offset);
    let help = signatures(&session, offset);
    let found: Vec<Json> = help
        .signatures
        .iter()
        .map(|help| {
            json!({
                "label": help.signature.label,
                "doc": help.doc,
                "parameters": help.signature.params.iter().zip(help.params.iter().chain(std::iter::repeat(&None))).map(|(param, param_doc)| json!({
                    "label": [param.range.0, param.range.1],
                    "doc": param_doc,
                })).collect::<Vec<_>>(),
            })
        })
        .collect();
    let active = (!found.is_empty()).then_some(0);
    encode(&json!({
        "signatures": found,
        "active": active,
        "activeParameter": help.active_parameter,
    }))
}

/// `tariff.definition`.
///
/// # Errors
///
/// When the input does not match the operation's shape.
pub fn definition(input: &RawValue) -> OpResult {
    let input: AtInput = decode(input)?;
    let session = Session::new(&input.source, kind(input.kind.as_deref())?, &input.context);
    let offset = session.byte(input.offset);
    let targets: Vec<Json> = definitions(&session, offset)
        .iter()
        .map(|target| {
            let index = session.utf16(target.module);
            json!({
                "module": session.module_path(target.module),
                "range": range(index, target.range),
                "selectionRange": range(index, target.selection),
            })
        })
        .collect();
    encode(&json!({ "targets": targets }))
}

/// `tariff.references`.
///
/// # Errors
///
/// When the input does not match the operation's shape.
pub fn references(input: &RawValue) -> OpResult {
    let input: ReferencesInput = decode(input)?;
    let session = Session::new(&input.source, kind(input.kind.as_deref())?, &input.context);
    let offset = session.byte(input.offset);
    let found: Vec<Json> = referenced(&session, offset, input.include_declaration)
        .iter()
        .map(|occurrence| {
            json!({
                "range": range(session.utf16(occurrence.module), occurrence.span),
                "kind": occurrence.kind.as_str(),
                "module": session.module_path(occurrence.module),
            })
        })
        .collect();
    encode(&json!({ "references": found }))
}

/// `tariff.actions`.
///
/// # Errors
///
/// When the input does not match the operation's shape.
pub fn actions(input: &RawValue) -> OpResult {
    let input: ActionsInput = decode(input)?;
    let session = Session::new(&input.source, kind(input.kind.as_deref())?, &input.context);
    let from = session.byte(input.range.from.min(input.range.to));
    let to = session.byte(input.range.to.max(input.range.from));
    let index = &session.root().utf16;
    let found: Vec<Json> = offered(&session, Span::new(from, to))
        .iter()
        .map(|action| {
            json!({
                "id": action.id,
                "title": action.title,
                "edits": action.edits.iter().map(|edit| json!({
                    "range": range(index, edit.span),
                    "insert": edit.insert,
                })).collect::<Vec<_>>(),
                "select": action.select.map(|select| json!({"from": select.from, "to": select.to})),
            })
        })
        .collect();
    encode(&json!({ "actions": found }))
}

/// `tariff.rename`.
///
/// # Errors
///
/// When the input does not match the operation's shape.
pub fn rename(input: &RawValue) -> OpResult {
    let input: RenameInput = decode(input)?;
    let session = Session::new(&input.source, kind(input.kind.as_deref())?, &input.context);
    let offset = session.byte(input.offset);
    let index = &session.root().utf16;
    let Some(new_name) = &input.new_name else {
        return encode(&match prepare(&session, offset) {
            Ok(prepared) => json!({
                "range": range(index, prepared.range),
                "placeholder": prepared.placeholder,
            }),
            Err(message) => json!({ "error": message }),
        });
    };
    encode(&match renamed(&session, offset, new_name) {
        Ok(edits) => json!({
            "edits": edits.iter().map(|edit| json!({
                "range": range(index, edit.span),
                "insert": edit.insert,
            })).collect::<Vec<_>>(),
        }),
        Err(message) => json!({ "error": message }),
    })
}

/// `tariff.inlays`.
///
/// # Errors
///
/// When the input does not match the operation's shape.
pub fn inlays(input: &RawValue) -> OpResult {
    let input: InlaysInput = decode(input)?;
    let session = Session::new(&input.source, kind(input.kind.as_deref())?, &input.context);
    let from = session.byte(input.range.from);
    let to = session.byte(input.range.to.max(input.range.from));
    let values = Values::read(input.values.as_ref());
    let found = hints(&session, Span::new(from, to), &values);
    encode(&json!({
        "hints": found.iter().map(|hint| json!({
            "offset": hint.offset,
            "label": hint.label,
            "kind": hint.kind.as_str(),
        })).collect::<Vec<_>>(),
    }))
}

/// The work an operation's session did, for the benchmark tests: the load and check (one step
/// per token and node) and every node and candidate the feature visited.
#[cfg(test)]
pub fn steps_for_tests(op: &str, input: &Json) -> u64 {
    let source = input["source"].as_str().unwrap_or_default();
    let context: Context = serde_json::from_value(input["context"].clone()).unwrap_or_default();
    let session = Session::new(source, None, &context);
    let offset = session.byte(u32::try_from(input["offset"].as_u64().unwrap_or(0)).unwrap_or(0));
    let values = Values::read(input.get("values"));
    match op {
        "complete" => {
            let _ = completions(&session, offset);
        }
        "hover" => {
            let _ = hovered(&session, offset, &values);
        }
        "signature" => {
            let _ = signatures(&session, offset);
        }
        "definition" => {
            let _ = definitions(&session, offset);
        }
        "references" => {
            let _ = referenced(&session, offset, true);
        }
        "actions" => {
            let _ = offered(&session, Span::new(offset, offset));
        }
        "rename" => {
            let _ = renamed(
                &session,
                offset,
                input["newName"].as_str().unwrap_or("renamed"),
            );
        }
        _ => {
            let end = u32::try_from(source.len()).unwrap_or(u32::MAX);
            let _ = hints(&session, Span::new(0, end), &values);
        }
    }
    session.steps()
}
