//! The `lading.*` operations: decode the input, run, encode the output with UTF-16 ranges.
//!
//! Every operation that works on a project takes `{project: {files, samples}, context: {org,
//! customFields}}`; the editor's add the file being edited as `path` and an `offset` or
//! `range` in UTF-16 code units of that file. See `docs/engineering/lading.md`, "Module
//! operations".

use std::cell::RefCell;
use std::collections::BTreeMap;
use std::rc::Rc;

use serde::Deserialize;
use serde_json::value::RawValue;
use serde_json::{Value as Json, json};
use trenova_module_kit::diagnostic::WireDiagnostic;
use trenova_module_kit::message;
use trenova_module_kit::message::Message;
use trenova_module_kit::ops::{OpResult, RequestError, decode, encode};
use trenova_module_kit::span::{Range, Span, Utf16Index};

use super::check::{Program, Source, check as check_project};
use super::editor::assist::{actions as offered, inlays as hints};
use super::editor::complete::complete as completions;
use super::editor::hover::{RunValue, hover as hovered};
use super::editor::navigate::{
    Renamed, definition as definitions, references as referenced, rename as renamed,
    signature as signatures,
};
use super::editor::session::Session;
use super::eval::interp::Given;
use super::eval::run::{RunRequest, run as run_map};
use super::flow::{DEFAULT_ACK_WITHIN, FlowEvent, FlowState, route, step};
use super::project::{Context, Project};
use super::replay::{MAX_INPUTS, ReplayRequest, replay as replay_versions};
use super::schema;
use super::starter::{StarterRequest, starter as build_starter};
use super::stdlib;
use super::syntax::ast::{Endpoint, Header, Item};
use super::syntax::format::format as format_source;
use super::syntax::highlight::highlights;
use super::syntax::outline::{folds, outline};
use super::syntax::parser::parse;
use super::testing::run_tests;

/// The most files a project holds.
pub const MAX_FILES: usize = 200;
/// The most samples a project holds.
pub const MAX_SAMPLES: usize = 2_000;

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct Head<'a> {
    #[serde(borrow)]
    project: &'a RawValue,
    #[serde(borrow, default)]
    context: Option<&'a RawValue>,
}

/// How many checked programs an instance keeps.
const CACHED_PROGRAMS: usize = 2;

thread_local! {
    /// Checked programs by the exact bytes of their project and context, most recent first: the
    /// editor and the TMS call the module many times with the same project, and checking it is
    /// most of a call. The key is the bytes themselves, never a hash, so no project is ever
    /// mistaken for another tenant's.
    static PROGRAMS: RefCell<Vec<(String, String, Rc<Program>)>> = const { RefCell::new(Vec::new()) };
}

/// The checked program of an operation's `project` and `context`.
fn program(input: &RawValue) -> Result<Rc<Program>, RequestError> {
    let head: Head<'_> = decode(input)?;
    let project_text = head.project.get();
    let context_text = head.context.map_or("{}", RawValue::get);
    let cached = PROGRAMS.with(|cache| {
        cache
            .borrow()
            .iter()
            .find(|(project, context, _)| project == project_text && context == context_text)
            .map(|(_, _, program)| Rc::clone(program))
    });
    if let Some(program) = cached {
        return Ok(program);
    }
    let project: Project = serde_json::from_str(project_text)
        .map_err(|error| RequestError::invalid_input(format!("project: {error}")))?;
    checked_project(&project)?;
    let context: Context = serde_json::from_str(context_text)
        .map_err(|error| RequestError::invalid_input(format!("context: {error}")))?;
    let program = Rc::new(check_project(&project, &context));
    PROGRAMS.with(|cache| {
        let mut cache = cache.borrow_mut();
        cache.insert(
            0,
            (
                project_text.to_owned(),
                context_text.to_owned(),
                Rc::clone(&program),
            ),
        );
        cache.truncate(CACHED_PROGRAMS);
    });
    Ok(program)
}

fn checked_project(project: &Project) -> Result<(), RequestError> {
    if project.files.len() > MAX_FILES {
        return Err(RequestError::invalid_input(format!(
            "a project holds at most {MAX_FILES} files"
        )));
    }
    if project.samples.len() > MAX_SAMPLES {
        return Err(RequestError::invalid_input(format!(
            "a project holds at most {MAX_SAMPLES} samples"
        )));
    }
    Ok(())
}

fn range(index: &Utf16Index, span: Span) -> Json {
    let range = index.range(span);
    json!({ "from": range.from, "to": range.to })
}

/// The UTF-16 index of every file.
fn indexes(program: &Program) -> Vec<Utf16Index> {
    program
        .units
        .iter()
        .map(|unit| Utf16Index::new(&unit.source))
        .collect()
}

/// `lading.check`.
pub fn check(input: &RawValue) -> OpResult {
    let program = program(input)?;
    encode(&check_json(&program))
}

#[allow(clippy::too_many_lines)]
fn check_json(program: &Program) -> Json {
    let indexes = indexes(program);
    let mut errors = 0;
    let mut warnings = 0;
    let files: Vec<Json> = program
        .units
        .iter()
        .zip(&indexes)
        .map(|(unit, index)| {
            for diagnostic in &unit.diagnostics {
                match diagnostic.severity {
                    trenova_module_kit::diagnostic::Severity::Error => errors += 1,
                    trenova_module_kit::diagnostic::Severity::Warning => warnings += 1,
                    trenova_module_kit::diagnostic::Severity::Info => {}
                }
            }
            json!({
                "path": unit.path,
                "kind": unit.header().kind(),
                "name": unit.header().name().map(|name| name.name.clone()),
                "diagnostics": unit.diagnostics.iter().map(|diagnostic| WireDiagnostic::new(diagnostic, index)).collect::<Vec<_>>(),
                "outline": outline(&unit.source, unit.items(), unit.header()),
                "folds": folds(&unit.source, unit.items()),
            })
        })
        .collect();
    let partner = program.partner.as_ref().map(|partner| {
        let settings: Vec<Json> = partner
            .settings
            .iter()
            .map(|setting| {
                let (doc, default, bounds) = match program.units[setting.unit].items().get(setting.item) {
                    Some(Item::Setting(decl)) => (
                        decl.doc.clone(),
                        decl.default.as_ref().map(|default| text_of(program, setting.unit, default.span)),
                        decl.range.as_ref().map(|(low, high)| {
                            json!([text_of(program, setting.unit, low.span), text_of(program, setting.unit, high.span)])
                        }),
                    ),
                    _ => (None, None, None),
                };
                json!({
                    "name": setting.name,
                    "type": setting.ty.to_string(),
                    "default": default,
                    "range": bounds,
                    "doc": doc,
                    "value": default_value(program, &setting.name),
                })
            })
            .collect();
        let interchange = program.interchange().map(|interchange| {
            json!({
                "them": interchange.them.as_ref().map(|party| json!({ "qualifier": party.qualifier.name, "id": party.id.value })),
                "us": interchange.us.as_ref().map(|party| json!({ "qualifier": party.qualifier.name, "id": party.id.value })),
                "group": interchange.group.as_ref().map(|(theirs, ours)| json!({ "theirs": theirs.value, "ours": ours.value })),
                "version": interchange.version.as_ref().map(|version| version.name.clone()),
                "usage": interchange.usage.as_ref().map(|usage| usage.name.clone()),
                "separators": program.partner_separators().iter().map(char::to_string).collect::<Vec<_>>(),
            })
        });
        json!({
            "name": partner.name,
            "label": partner.label,
            "zone": partner.zone,
            "currency": partner.currency,
            "settings": settings,
            "interchange": interchange,
        })
    });
    let maps: Vec<Json> = program
        .maps
        .iter()
        .map(|(name, info)| {
            let (from, to) = match program.units[info.unit].header() {
                Header::Map { from, to, .. } => (endpoint(from), endpoint(to)),
                _ => (String::new(), String::new()),
            };
            let set = info.guide().map(|guide| {
                program.guides[guide]
                    .compiled
                    .guide()
                    .transaction_set
                    .clone()
            });
            json!({
                "name": name,
                "path": program.units[info.unit].path,
                "direction": if info.inbound() { "inbound" } else { "outbound" },
                "from": from,
                "to": to,
                "document": info.document,
                "set": set,
                "source": match info.source {
                    Source::X12(_) => "x12",
                    Source::Csv(_) => "csv",
                    Source::Json => "json",
                    Source::Document(_) => "document",
                },
            })
        })
        .collect();
    let flows: Vec<Json> = program
        .flows
        .iter()
        .map(|(name, unit)| {
            let clauses: Vec<Json> = program.units[*unit]
                .items()
                .iter()
                .filter_map(|item| match item {
                    Item::Flow(clause) => Some(match clause {
                        super::syntax::ast::FlowDecl::Receive(receive) => json!({
                            "kind": "receive",
                            "set": receive.set.name,
                            "binder": receive.binder.name,
                            "map": receive.map.as_ref().map(|map| map.name.clone()),
                            "rules": receive.rules.as_ref().map(|rules| rules.name.clone()),
                            "acknowledge": receive.acknowledge.as_ref().map_or("997", |ack| ack.name.as_str()),
                        }),
                        super::syntax::ast::FlowDecl::Respond(respond) => json!({
                            "kind": "respond",
                            "set": respond.set.name,
                            "map": respond.map.as_ref().map(|map| map.name.clone()),
                        }),
                        super::syntax::ast::FlowDecl::Send(send) => json!({
                            "kind": "send",
                            "set": send.set.name,
                            "map": send.map.as_ref().map(|map| map.name.clone()),
                            "events": send.events.iter().map(|event| event.name.clone()).collect::<Vec<_>>(),
                            "until": send.until.as_ref().map(|event| event.name.clone()),
                            "batched": send.batch.is_some(),
                        }),
                        super::syntax::ast::FlowDecl::Expect(expect) => json!({
                            "kind": "expect",
                            "set": expect.set.name,
                            "within": expect.within.as_ref().map_or(DEFAULT_ACK_WITHIN, |within| seconds_of(program, *unit, within)),
                        }),
                    }),
                    _ => None,
                })
                .collect();
            json!({ "name": name, "path": program.units[*unit].path, "clauses": clauses })
        })
        .collect();
    let guides: Vec<Json> = program
        .guides
        .iter()
        .map(|entry| {
            json!({
                "name": entry.name,
                "path": entry.unit.map(|unit| program.units[unit].path.clone()),
                "set": entry.compiled.guide().transaction_set,
                "version": entry.compiled.guide().version,
                "base": entry.compiled.guide().derived_from,
            })
        })
        .collect();
    json!({
        "files": files,
        "partner": partner,
        "maps": maps,
        "flows": flows,
        "rules": program.rules.iter().map(|(name, (unit, map))| json!({ "name": name, "path": program.units[*unit].path, "map": map })).collect::<Vec<_>>(),
        "libraries": program.libraries.keys().collect::<Vec<_>>(),
        "guides": guides,
        "errors": errors,
        "warnings": warnings,
    })
}

/// A duration expression's seconds, 0 when it is not a duration.
fn seconds_of(program: &Program, unit: usize, expr: &super::syntax::ast::Expr) -> i64 {
    let given = Given::default();
    let mut interp = super::eval::interp::Interp::new(program, &given, 10_000);
    let mut frame = super::eval::interp::Frame::new(unit);
    match interp.eval(&mut frame, expr) {
        Ok(super::eval::value::Value::Duration(seconds)) => seconds,
        _ => 0,
    }
}

fn default_value(program: &Program, name: &str) -> Json {
    let given = Given::default();
    let interp = super::eval::interp::Interp::new(program, &given, 10_000);
    super::eval::json::to_json(&interp.partner().field(name))
}

fn text_of(program: &Program, unit: usize, span: Span) -> String {
    program.units[unit]
        .source
        .get(span.start as usize..span.end as usize)
        .unwrap_or_default()
        .to_owned()
}

fn endpoint(endpoint: &Endpoint) -> String {
    match endpoint {
        Endpoint::Guide(name) | Endpoint::Document(name) => name.name.clone(),
        Endpoint::Csv { .. } => "csv".to_owned(),
        Endpoint::Json(_) => "json".to_owned(),
        Endpoint::Missing(_) => String::new(),
    }
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct SourceInput {
    source: String,
}

/// `lading.format`.
pub fn format(input: &RawValue) -> OpResult {
    let input: SourceInput = decode(input)?;
    match format_source(&input.source) {
        Ok(text) => encode(&json!({ "text": text })),
        Err(_) => encode(
            &json!({ "error": Message::new(message!("The file has errors, so it is left as it is")) }),
        ),
    }
}

/// `lading.tokens`.
pub fn tokens(input: &RawValue) -> OpResult {
    let input: SourceInput = decode(input)?;
    let parsed = parse(&input.source);
    encode(&json!({ "tokens": highlights(&input.source, &parsed) }))
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct AtInput {
    path: String,
    offset: u32,
    #[serde(default)]
    values: Vec<RunValue>,
    #[serde(default = "yes")]
    include_declaration: bool,
    #[serde(default)]
    new_name: Option<String>,
}

const fn yes() -> bool {
    true
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct RangeInput {
    path: String,
    range: Range,
    #[serde(default)]
    values: Vec<RunValue>,
}

fn open(raw: &RawValue, path: &str) -> Result<Session, RequestError> {
    Session::open(program(raw)?, path)
        .ok_or_else(|| RequestError::invalid_input(format!("the project has no file {path:?}")))
}

/// `lading.complete`.
pub fn complete(raw: &RawValue) -> OpResult {
    let input: AtInput = decode(raw)?;
    let session = open(raw, &input.path)?;
    let offset = session.byte(input.offset);
    let found = completions(&session, offset);
    encode(&json!({ "range": range(&session.index, found.range), "items": found.items }))
}

/// `lading.hover`.
pub fn hover(raw: &RawValue) -> OpResult {
    let input: AtInput = decode(raw)?;
    let session = open(raw, &input.path)?;
    let offset = session.byte(input.offset);
    let Some(found) = hovered(&session, offset, &input.values) else {
        return encode(&Json::Null);
    };
    encode(&json!({
        "range": range(&session.index, found.range),
        "contents": { "signature": found.signature, "type": found.ty, "doc": found.doc, "value": found.value },
    }))
}

/// `lading.signature`.
pub fn signature(raw: &RawValue) -> OpResult {
    let input: AtInput = decode(raw)?;
    let session = open(raw, &input.path)?;
    let offset = session.byte(input.offset);
    let Some(found) = signatures(&session, offset) else {
        return encode(&json!({ "signatures": [], "active": null, "activeParameter": null }));
    };
    let label_index = Utf16Index::new(&found.label);
    encode(&json!({
        "signatures": [{
            "label": found.label,
            "doc": found.doc,
            "parameters": found.parameters.iter().map(|(from, to)| json!({ "label": [label_index.to_utf16(*from), label_index.to_utf16(*to)] })).collect::<Vec<_>>(),
        }],
        "active": 0,
        "activeParameter": found.active,
    }))
}

/// `lading.definition`.
pub fn definition(raw: &RawValue) -> OpResult {
    let input: AtInput = decode(raw)?;
    let session = open(raw, &input.path)?;
    let offset = session.byte(input.offset);
    let indexes = indexes(&session.program);
    let targets: Vec<Json> = definitions(&session, offset)
        .into_iter()
        .map(|location| json!({ "path": session.program.units[location.unit].path, "range": range(&indexes[location.unit], location.span) }))
        .collect();
    encode(&json!({ "targets": targets }))
}

/// `lading.references`.
pub fn references(raw: &RawValue) -> OpResult {
    let input: AtInput = decode(raw)?;
    let session = open(raw, &input.path)?;
    let offset = session.byte(input.offset);
    let indexes = indexes(&session.program);
    let found: Vec<Json> = referenced(&session, offset, input.include_declaration)
        .into_iter()
        .map(|reference| {
            json!({
                "path": session.program.units[reference.unit].path,
                "range": range(&indexes[reference.unit], reference.span),
                "kind": reference.kind,
            })
        })
        .collect();
    encode(&json!({ "references": found }))
}

/// `lading.rename`.
pub fn rename(raw: &RawValue) -> OpResult {
    let input: AtInput = decode(raw)?;
    let session = open(raw, &input.path)?;
    let offset = session.byte(input.offset);
    let indexes = indexes(&session.program);
    match renamed(&session, offset, input.new_name.as_deref()) {
        Renamed::Edits(edits) => encode(&json!({
            "edits": edits.iter().map(|(unit, span, insert)| json!({
                "path": session.program.units[*unit].path,
                "range": range(&indexes[*unit], *span),
                "insert": insert,
            })).collect::<Vec<_>>(),
        })),
        Renamed::Prepared(span, placeholder) => {
            encode(&json!({ "range": range(&session.index, span), "placeholder": placeholder }))
        }
        Renamed::Refused(message) => encode(&json!({ "error": message })),
    }
}

/// `lading.inlays`.
pub fn inlays(raw: &RawValue) -> OpResult {
    let input: RangeInput = decode(raw)?;
    let session = open(raw, &input.path)?;
    let span = Span::new(session.byte(input.range.from), session.byte(input.range.to));
    let found: Vec<Json> = hints(&session, span, &input.values)
        .into_iter()
        .map(|hint| json!({ "offset": session.index.to_utf16(hint.offset), "label": hint.label, "kind": hint.kind }))
        .collect();
    encode(&json!({ "hints": found }))
}

/// `lading.actions`.
pub fn actions(raw: &RawValue) -> OpResult {
    let input: RangeInput = decode(raw)?;
    let session = open(raw, &input.path)?;
    let span = Span::new(session.byte(input.range.from), session.byte(input.range.to));
    let indexes = indexes(&session.program);
    let empty = Utf16Index::new("");
    let found: Vec<Json> = offered(&session, span)
        .into_iter()
        .map(|action| {
            json!({
                "id": action.id,
                "title": action.title,
                "edits": action.edits.iter().map(|edit| {
                    let index = session.program.unit_at(&edit.path).map_or(&empty, |unit| &indexes[unit]);
                    json!({ "path": edit.path, "range": range(index, edit.span), "insert": edit.insert })
                }).collect::<Vec<_>>(),
            })
        })
        .collect();
    encode(&json!({ "actions": found }))
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct RunInput {
    #[serde(flatten)]
    run: RunRequest,
}

/// `lading.run`.
pub fn run(raw: &RawValue) -> OpResult {
    let program = program(raw)?;
    let input: RunInput = decode(raw)?;
    encode(&run_map(&program, &input.run))
}

/// `lading.test`.
pub fn test(raw: &RawValue) -> OpResult {
    let program = program(raw)?;
    let report = run_tests(&program);
    let indexes = indexes(&program);
    let tests: Vec<Json> = report
        .tests
        .iter()
        .map(|test| {
            let index = &indexes[test.unit];
            json!({
                "path": program.units[test.unit].path,
                "name": test.name,
                "kind": test.kind,
                "range": range(index, test.span),
                "passed": test.passed,
                "cases": test.cases,
                "failures": test.failures.iter().map(|failure| json!({ "range": range(index, failure.span), "message": failure.message })).collect::<Vec<_>>(),
                "counterexample": test.counterexample.as_ref().map(|case| case.iter().map(|(name, value)| json!({ "name": name, "value": value })).collect::<Vec<_>>()),
            })
        })
        .collect();
    let coverage: Vec<Json> = report
        .coverage
        .iter()
        .map(|file| {
            let index = &indexes[file.unit];
            json!({
                "path": program.units[file.unit].path,
                "covered": file.covered.iter().map(|span| range(index, *span)).collect::<Vec<_>>(),
                "uncovered": file.uncovered.iter().map(|span| range(index, *span)).collect::<Vec<_>>(),
            })
        })
        .collect();
    let uncovered_arms = report
        .coverage
        .iter()
        .filter(|file| program.units[file.unit].header().kind() == "flow")
        .map(|file| file.uncovered.len())
        .sum::<usize>();
    let errors = program
        .units
        .iter()
        .flat_map(|unit| unit.diagnostics.iter())
        .filter(|diagnostic| diagnostic.severity == trenova_module_kit::diagnostic::Severity::Error)
        .count();
    let failing = report.tests.iter().filter(|test| !test.passed).count();
    encode(&json!({
        "tests": tests,
        "coverage": coverage,
        "unread": report.unread,
        "readiness": {
            "errors": errors,
            "failing": failing,
            "tests": report.tests.len(),
            "uncoveredFlowClauses": uncovered_arms,
            "ready": errors == 0 && failing == 0 && uncovered_arms == 0 && !report.tests.is_empty(),
        },
    }))
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct FlowInput {
    flow: String,
    #[serde(default)]
    state: Option<FlowState>,
    event: FlowEvent,
    #[serde(default)]
    now: Option<String>,
    #[serde(default)]
    settings: BTreeMap<String, Json>,
}

/// `lading.flowStep`.
pub fn flow_step(raw: &RawValue) -> OpResult {
    let program = program(raw)?;
    let input: FlowInput = decode(raw)?;
    if program.has_code_errors() {
        return Err(RequestError::invalid_input("the project has errors"));
    }
    let now = input
        .now
        .as_deref()
        .and_then(super::values::parse_instant)
        .map_or(0, |(instant, _)| instant);
    let request = RunRequest {
        map: String::new(),
        input: super::eval::run::RunInput::Json(Json::Null),
        settings: input.settings.clone(),
        lookups: Vec::new(),
        now: input.now.clone(),
        control: super::eval::run::Control::default(),
        rules: false,
    };
    let given = super::eval::run::given(&program, &request);
    let result = step(
        &program,
        &input.flow,
        input.state.unwrap_or_default(),
        &input.event,
        now,
        &given,
    );
    encode(&result)
}

/// `lading.flowKey`.
pub fn flow_key(raw: &RawValue) -> OpResult {
    let program = program(raw)?;
    let input: FlowInput = decode(raw)?;
    let now = input
        .now
        .as_deref()
        .and_then(super::values::parse_instant)
        .map_or(0, |(instant, _)| instant);
    encode(&json!({ "key": route(&program, &input.flow, &input.event, now) }))
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct ReplayInput {
    before: Project,
    after: Project,
    #[serde(default)]
    context: Context,
    #[serde(flatten)]
    replay: ReplayRequest,
}

/// `lading.replay`.
pub fn replay(input: &RawValue) -> OpResult {
    let input: ReplayInput = decode(input)?;
    checked_project(&input.before)?;
    checked_project(&input.after)?;
    if input.replay.inputs.len() > MAX_INPUTS {
        return Err(RequestError::invalid_input(format!(
            "a replay call takes at most {MAX_INPUTS} inputs"
        )));
    }
    let before = check_project(&input.before, &input.context);
    let after = check_project(&input.after, &input.context);
    encode(&replay_versions(&before, &after, &input.replay))
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct GuideInput {
    name: String,
}

/// `lading.guide`.
pub fn guide(raw: &RawValue) -> OpResult {
    let program = program(raw)?;
    let input: GuideInput = decode(raw)?;
    match program.guide(&input.name) {
        Some(entry) => encode(entry.compiled.guide()),
        None => Err(RequestError::invalid_input(format!(
            "the project has no guide {:?}",
            input.name
        ))),
    }
}

/// `lading.starter`.
pub fn starter(input: &RawValue) -> OpResult {
    let input: StarterRequest = decode(input)?;
    encode(&build_starter(&input))
}

/// `lading.catalog`.
pub fn catalog(_: &RawValue) -> OpResult {
    let record = |record: &schema::Record| {
        json!({
            "name": record.name,
            "doc": Message::new(record.doc),
            "set": record.set,
            "fields": record.fields.iter().map(|field| json!({
                "name": field.name,
                "type": field.spec.ty().to_string(),
                "required": field.required,
                "doc": Message::new(field.doc),
                "maxLength": field.max_length,
            })).collect::<Vec<_>>(),
        })
    };
    encode(&json!({
        "documents": schema::DOCUMENTS.iter().map(record).collect::<Vec<_>>(),
        "records": schema::RECORDS.iter().map(record).collect::<Vec<_>>(),
        "enums": schema::ENUMS.iter().map(|item| json!({
            "name": item.name,
            "doc": Message::new(item.doc),
            "variants": item.variants.iter().map(|(name, doc)| json!({ "name": name, "doc": Message::new(doc) })).collect::<Vec<_>>(),
        })).collect::<Vec<_>>(),
        "lookups": schema::LOOKUPS.iter().map(|item| json!({ "name": item.name, "entity": item.entity, "doc": Message::new(item.doc), "zoned": item.zoned })).collect::<Vec<_>>(),
        "events": schema::EVENTS.iter().map(|item| json!({ "name": item.name, "document": item.document, "doc": Message::new(item.doc) })).collect::<Vec<_>>(),
        "states": schema::STATES.iter().map(|(name, doc)| json!({ "name": name, "doc": Message::new(doc) })).collect::<Vec<_>>(),
        "functions": stdlib::FUNCTIONS.iter().map(|function| json!({ "name": function.name, "signature": function.signature, "doc": Message::new(function.doc) })).collect::<Vec<_>>(),
    }))
}

#[cfg(test)]
mod tests {
    use serde_json::{Value as Json, json};

    use crate::lading::check::corpus::{acme, context};
    use crate::ops::dispatch;

    fn project() -> Json {
        let project = acme();
        let context = context();
        json!({
            "project": {
                "files": project.files.iter().map(|file| json!({ "path": file.path, "source": file.source })).collect::<Vec<_>>(),
                "samples": project.samples.iter().map(|sample| json!({ "path": sample.path, "content": sample.content })).collect::<Vec<_>>(),
            },
            "context": { "org": { "name": context.org.name, "scac": context.org.scac, "zone": context.org.zone } },
        })
    }

    fn call(op: &str, extra: Json) -> Json {
        let mut input = project();
        if let (Some(object), Json::Object(more)) = (input.as_object_mut(), extra) {
            object.extend(more);
        }
        let request = json!({ "op": op, "input": input }).to_string();
        let response: Json = serde_json::from_slice(&dispatch(request.as_bytes())).expect("json");
        assert!(response.get("ok").is_some(), "{op}: {response}");
        response["ok"].clone()
    }

    fn offset_of(path: &str, needle: &str, shift: usize) -> usize {
        let source = acme()
            .files
            .into_iter()
            .find(|file| file.path == path)
            .expect("file")
            .source;
        let at = source.find(needle).expect("needle") + shift;
        source[..at].encode_utf16().count()
    }

    #[test]
    fn check_reports_files_settings_and_maps() {
        let out = call("lading.check", json!({}));
        assert_eq!(out["errors"], 0, "{out}");
        assert_eq!(out["partner"]["settings"][1]["name"], "maxWeight");
        assert_eq!(out["partner"]["settings"][1]["value"]["unit"], "lb");
        assert!(
            out["maps"]
                .as_array()
                .expect("maps")
                .iter()
                .any(|map| map["name"] == "tenderIn" && map["direction"] == "inbound")
        );
    }

    #[test]
    fn completion_after_a_loop_lists_its_segments() {
        let path = "maps/tender-in.lading";
        let offset = offset_of(path, "stop.S502", 5);
        let out = call("lading.complete", json!({ "path": path, "offset": offset }));
        let labels: Vec<&str> = out["items"]
            .as_array()
            .expect("items")
            .iter()
            .filter_map(|item| item["label"].as_str())
            .collect();
        assert!(labels.contains(&"S502"), "{labels:?}");
        assert!(labels.contains(&"0310"), "{labels:?}");
    }

    #[test]
    fn hover_names_an_element() {
        let path = "maps/tender-in.lading";
        let out = call(
            "lading.hover",
            json!({ "path": path, "offset": offset_of(path, "B204", 1) }),
        );
        assert!(
            out["contents"]["signature"]
                .as_str()
                .expect("signature")
                .starts_with("B204"),
            "{out}"
        );
        assert_eq!(out["contents"]["type"], "Text");
    }

    #[test]
    fn definition_reaches_into_a_library() {
        let path = "maps/tender-in.lading";
        let out = call(
            "lading.definition",
            json!({ "path": path, "offset": offset_of(path, "via stopTypes", 6) }),
        );
        assert_eq!(out["targets"][0]["path"], "lib/common.lading", "{out}");
    }

    #[test]
    fn references_and_rename_span_files() {
        let path = "lib/common.lading";
        let offset = offset_of(path, "crosswalk stopTypes", 12);
        let references = call(
            "lading.references",
            json!({ "path": path, "offset": offset }),
        );
        let paths: Vec<&str> = references["references"]
            .as_array()
            .expect("references")
            .iter()
            .filter_map(|item| item["path"].as_str())
            .collect();
        assert!(paths.contains(&"maps/tender-in.lading"), "{paths:?}");
        let renamed = call(
            "lading.rename",
            json!({ "path": path, "offset": offset, "newName": "stopKinds" }),
        );
        assert_eq!(
            renamed["edits"].as_array().expect("edits").len(),
            paths.len(),
            "{renamed}"
        );
        let refused = call(
            "lading.rename",
            json!({ "path": path, "offset": offset, "newName": "Bad Name" }),
        );
        assert!(refused.get("error").is_some());
    }

    #[test]
    fn signature_help_shows_the_active_parameter() {
        let path = "maps/tender-in.lading";
        let out = call(
            "lading.signature",
            json!({ "path": path, "offset": offset_of(path, "window(stop.G62, ", 17) }),
        );
        assert_eq!(out["activeParameter"], 1, "{out}");
        assert!(
            out["signatures"][0]["label"]
                .as_str()
                .expect("label")
                .starts_with("fn window(")
        );
    }

    #[test]
    fn a_literal_can_become_a_setting() {
        let path = "rules/tender-checks.lading";
        let from = offset_of(path, "0 lb", 0);
        let out = call(
            "lading.actions",
            json!({ "path": path, "range": { "from": from, "to": from + 4 } }),
        );
        let ids: Vec<&str> = out["actions"]
            .as_array()
            .expect("actions")
            .iter()
            .filter_map(|item| item["id"].as_str())
            .collect();
        assert!(ids.contains(&"promote-to-setting"), "{out}");
    }

    #[test]
    fn run_test_flow_starter_and_catalog_answer() {
        let sample = acme()
            .sample("samples/two-stops.x12")
            .expect("sample")
            .content
            .clone();
        let run = call(
            "lading.run",
            json!({ "map": "tenderIn", "input": { "x12": sample } }),
        );
        assert!(!run["needs"].as_array().expect("needs").is_empty());
        let test = call("lading.test", json!({}));
        assert_eq!(test["readiness"]["failing"], 0, "{test}");
        let step = call(
            "lading.flowStep",
            json!({ "flow": "tender", "event": { "kind": "planner", "decision": "accept" }, "now": "2026-01-10T00:00:00Z" }),
        );
        assert!(step["actions"].as_array().expect("actions").is_empty());
        let starter = call(
            "lading.starter",
            json!({ "name": "swift", "role": "broker" }),
        );
        assert!(starter["files"].as_array().expect("files").len() > 4);
        let catalog = call("lading.catalog", json!({}));
        assert_eq!(catalog["documents"][0]["name"], "loadTender");
        let tokens = call(
            "lading.tokens",
            json!({ "source": "map a\n  from x12.004010.204\n  to loadTender\n" }),
        );
        assert!(!tokens["tokens"].as_array().expect("tokens").is_empty());
    }
}
