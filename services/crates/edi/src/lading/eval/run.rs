//! `lading.run`: one map (and its rules) on one input. A run that needs lookups the host has
//! not answered says which, and the host runs it again with the answers, so a map makes one
//! batched round trip however many lookups it has.

use std::collections::{BTreeMap, BTreeSet};
use std::rc::Rc;

use serde::{Deserialize, Serialize};
use serde_json::Value as Json;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::csv;
use super::interp::{
    Frame, Given, Halt, Interp, Lineage, Need, Outcome, OutcomeKind2, RUN_STEPS, Unmapped,
};
use super::json::{from_json, to_json};
use super::outbound::RunFinding;
use super::rules::Fixed;
use super::tree::{Raw, Tree};
use super::value::{Reference, Value, local};
use crate::lading::check::{MapInfo, Program, Source};
use crate::lading::syntax::ast::{Endpoint, Header, Item, Stmt};
use crate::lading::types::Type;
use crate::lading::tz::civil_from_days;
use crate::lading::values::{ZoneValue, parse_instant};
use crate::x12::finding::Finding;
use crate::x12::read::{GuideChoice, ReadRequest, TransactionStatus, read};
use crate::x12::write::{Element, WriteRequest, write};

/// What a run reads.
#[derive(Debug, Clone, Deserialize, PartialEq)]
#[serde(rename_all = "camelCase")]
pub enum RunInput {
    /// X12 text, for a map from a guide.
    X12(String),
    /// CSV text.
    Csv(String),
    /// A JSON document.
    Json(Json),
    /// A Trenova document, for a map to X12.
    Document(Json),
}

/// A lookup's answer.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LookupAnswer {
    /// The kind.
    pub kind: String,
    /// The partner's code.
    pub code: String,
    /// The record it maps to; none when it maps to nothing yet.
    #[serde(default)]
    pub found: Option<Found>,
}

/// A record a lookup found.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Found {
    /// Its ID.
    pub id: String,
    /// Its name.
    #[serde(default)]
    pub label: String,
    /// Its time zone, for a location.
    #[serde(default)]
    pub zone: Option<String>,
}

/// The control numbers an outbound run writes.
#[derive(Debug, Clone, Copy, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct Control {
    /// ISA13.
    pub interchange: u64,
    /// GS06.
    pub group: u64,
    /// ST02.
    pub transaction: u64,
}

impl Default for Control {
    fn default() -> Self {
        Self {
            interchange: 1,
            group: 1,
            transaction: 1,
        }
    }
}

/// What a run is asked to do.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RunRequest {
    /// The map.
    pub map: String,
    /// What it reads.
    pub input: RunInput,
    /// Setting values that differ from the partner file's defaults.
    #[serde(default)]
    pub settings: BTreeMap<String, Json>,
    /// Lookup answers.
    #[serde(default)]
    pub lookups: Vec<LookupAnswer>,
    /// The time the run starts, RFC 3339; the epoch when none.
    #[serde(default)]
    pub now: Option<String>,
    /// Control numbers for an outbound run.
    #[serde(default)]
    pub control: Control,
    /// Whether to run the map's rules.
    #[serde(default = "yes")]
    pub rules: bool,
}

fn yes() -> bool {
    true
}

/// One transaction an inbound run read.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TransactionResult {
    /// Its index in the input, from 0.
    pub index: usize,
    /// ST02, for X12.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub control: String,
    /// `accepted`, or `rejected` when validation rejected it and no map ran.
    pub status: &'static str,
    /// What validation found.
    pub findings: Vec<Finding>,
    /// The document, after its rules' fixes.
    pub document: Json,
    /// What the map's elses and the rules decided.
    pub outcomes: Vec<Outcome>,
    /// What the rules changed.
    pub fixes: Vec<Fixed>,
    /// Where each field came from.
    pub lineage: Vec<Lineage>,
    /// Elements present that nothing read: `0300[1].N3[1]02`.
    pub unread: Vec<String>,
}

/// What an outbound run wrote.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct WriteResult {
    /// The interchange.
    pub text: String,
    /// The transaction set alone, ST to SE.
    pub transaction: String,
    /// The document as written, after its rules' fixes.
    pub document: Json,
    /// What was wrong with the values written.
    pub findings: Vec<RunFinding>,
    /// What reading the interchange back found.
    pub validation: Vec<Finding>,
    /// What the rules decided; a reject or hold keeps the message from being sent.
    pub outcomes: Vec<Outcome>,
    /// What the rules changed.
    pub fixes: Vec<Fixed>,
    /// Where each element came from.
    pub lineage: Vec<Lineage>,
}

/// What a run produced.
#[derive(Debug, Clone, Serialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct RunResult {
    /// The transactions read, for an inbound map.
    pub transactions: Vec<TransactionResult>,
    /// What was written, for an outbound map.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub output: Option<WriteResult>,
    /// Lookups to answer before the result is final.
    pub needs: Vec<Need>,
    /// Codes that map to nothing yet.
    pub unmapped: Vec<Unmapped>,
    /// Why the run could not finish.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<Message>,
    /// Statements that ran, by file path and byte offset.
    #[serde(skip)]
    pub coverage: BTreeSet<(usize, u32)>,
    /// Evaluation steps used.
    pub steps: u64,
}

/// What a run is given, from a request.
#[must_use]
pub fn given(program: &Program, request: &RunRequest) -> Given {
    let mut settings = BTreeMap::new();
    if let Some(partner) = &program.partner {
        for (name, value) in &request.settings {
            if let Some(setting) = partner
                .settings
                .iter()
                .find(|setting| &setting.name == name)
                && let Ok(value) = from_json(program, value, &setting.ty, name)
            {
                settings.insert(name.clone(), value);
            }
        }
    }
    let lookups = request
        .lookups
        .iter()
        .map(|answer| {
            let found = answer.found.as_ref().and_then(|found| {
                let kind = crate::lading::schema::lookup(&answer.kind)?.name;
                Some(Reference {
                    kind,
                    id: found.id.clone(),
                    code: answer.code.clone(),
                    label: found.label.clone(),
                    zone: found.zone.as_deref().and_then(ZoneValue::parse),
                })
            });
            ((answer.kind.clone(), answer.code.clone()), found)
        })
        .collect();
    let now = request
        .now
        .as_deref()
        .and_then(parse_instant)
        .map_or(0, |(instant, _)| instant);
    Given {
        settings,
        lookups,
        now,
        stand_in: false,
    }
}

/// Runs a map on an input.
#[must_use]
pub fn run(program: &Program, request: &RunRequest) -> RunResult {
    let given = given(program, request);
    let Some(info) = program.maps.get(&request.map) else {
        return failed(Message::with(
            message!("There is no map named {0}"),
            [Arg::from(request.map.as_str())],
        ));
    };
    if program.has_code_errors() {
        return failed(Message::new(message!(
            "The project has errors; fix them before running a map"
        )));
    }
    let mut result = match (&info.source, &request.input) {
        (Source::X12(guide), RunInput::X12(text)) => {
            run_x12(program, &given, request, info, *guide, text)
        }
        (Source::Csv(_), RunInput::Csv(text)) => run_rows(program, &given, request, info, text),
        (Source::Json, RunInput::Json(json)) => {
            let input = Value::json(json);
            run_document_input(program, &given, request, info, ("input", input))
        }
        (Source::Document(document), RunInput::Document(json)) => {
            run_outbound(program, &given, request, info, document, json)
        }
        _ => failed(Message::new(message!(
            "The input is not what this map reads"
        ))),
    };
    result
        .unmapped
        .dedup_by(|a, b| a.kind == b.kind && a.code == b.code);
    result
}

fn failed(message: Message) -> RunResult {
    RunResult {
        error: Some(message),
        ..RunResult::default()
    }
}

fn budget() -> Message {
    Message::new(message!(
        "The run used more than its budget of steps; it may loop over far more data than expected"
    ))
}

pub fn statements(program: &Program, unit: usize) -> Vec<&Stmt> {
    program.units[unit]
        .items()
        .iter()
        .filter_map(|item| match item {
            Item::Stmt(stmt) => Some(stmt),
            _ => None,
        })
        .collect()
}

/// The rules files of a map, by file.
pub fn rules_of(program: &Program, map: &str) -> Vec<usize> {
    program
        .rules
        .values()
        .filter(|(_, for_map)| for_map == map)
        .map(|(unit, _)| *unit)
        .collect()
}

/// Fills a document and runs its rules.
pub fn fill_value(
    interp: &mut Interp<'_>,
    map: &str,
    rules: bool,
    info: &MapInfo,
    mut frame: Frame,
) -> Result<(Value, Vec<Fixed>), Halt> {
    let program = interp.program;
    let Some(document) = info.document else {
        return Ok((Value::None, Vec::new()));
    };
    let stmts = statements(program, info.unit);
    let mut value = interp.fill_document(&mut frame, &stmts, document)?;
    let mut fixes = Vec::new();
    if rules {
        for unit in rules_of(program, map) {
            value = interp.run_rules(unit, value, &mut fixes)?;
        }
    }
    Ok((value, fixes))
}

fn fill(
    interp: &mut Interp<'_>,
    request: &RunRequest,
    info: &MapInfo,
    frame: Frame,
) -> Result<(Json, Vec<Fixed>), Halt> {
    fill_value(interp, &request.map, request.rules, info, frame)
        .map(|(value, fixes)| (to_json(&value), fixes))
}

/// One transaction set read from X12 with a guide.
#[derive(Debug)]
pub struct ReadTransaction {
    /// ST02.
    pub control: String,
    /// Whether validation rejected it.
    pub rejected: bool,
    /// What validation found.
    pub findings: Vec<Finding>,
    /// Its segments in loop instances, when it was not rejected.
    pub tree: Option<Rc<Tree>>,
}

/// Reads the transaction sets of a guide's set from X12 text.
pub fn read_x12(
    program: &Program,
    guide: usize,
    text: &str,
) -> Result<Vec<ReadTransaction>, Message> {
    let compiled = &program.guides[guide].compiled;
    let set = compiled.guide().transaction_set.clone();
    let reading = read(&ReadRequest {
        text: text.to_owned(),
        separators: None,
        guides: vec![GuideChoice {
            transaction_set: set.clone(),
            guide: Some(compiled.guide().clone()),
            builtin: None,
        }],
        include_segments: true,
        fragment: false,
    })
    .map_err(|problem| {
        Message::with(
            message!("The X12 does not read: {0}"),
            [Arg::from(problem.detail)],
        )
    })?;
    let component = reading.separators.component.chars().next().unwrap_or('>');
    let repetition = reading.separators.repetition.chars().next();
    let mut found = Vec::new();
    for interchange in &reading.interchanges {
        for group in &interchange.groups {
            for transaction in &group.transactions {
                if transaction.transaction_set != set {
                    continue;
                }
                let rejected = transaction.status == TransactionStatus::Rejected;
                let tree = (!rejected).then(|| {
                    let last = transaction
                        .last_segment
                        .min(reading.segments.len().saturating_sub(1));
                    let raw: Vec<Raw> = reading.segments[transaction.first_segment..=last]
                        .iter()
                        .map(|segment| Raw {
                            id: segment.id.clone(),
                            elements: segment.elements.clone(),
                            position: segment.position.unwrap_or(0),
                        })
                        .collect();
                    Rc::new(Tree::build(
                        guide,
                        compiled.clone(),
                        raw,
                        component,
                        repetition,
                    ))
                });
                found.push(ReadTransaction {
                    control: transaction.control_number.clone(),
                    rejected,
                    findings: transaction.findings.clone(),
                    tree,
                });
            }
        }
    }
    if found.is_empty() {
        return Err(Message::with(
            message!("The X12 holds no {0} transaction set"),
            [Arg::from(set.as_str())],
        ));
    }
    Ok(found)
}

/// The elements present in a transaction that nothing read.
#[must_use]
pub fn unread(tree: &Tree) -> Vec<(usize, u32)> {
    let read = tree.read.borrow();
    tree.present()
        .into_iter()
        .filter(|key| !read.contains(key))
        .collect()
}

fn run_x12(
    program: &Program,
    given: &Given,
    request: &RunRequest,
    info: &MapInfo,
    guide: usize,
    text: &str,
) -> RunResult {
    let transactions = match read_x12(program, guide, text) {
        Ok(transactions) => transactions,
        Err(message) => return failed(message),
    };
    let mut result = RunResult::default();
    for (index, transaction) in transactions.into_iter().enumerate() {
        let mut entry = TransactionResult {
            index,
            control: transaction.control,
            status: if transaction.rejected {
                "rejected"
            } else {
                "accepted"
            },
            findings: transaction.findings,
            document: Json::Null,
            outcomes: Vec::new(),
            fixes: Vec::new(),
            lineage: Vec::new(),
            unread: Vec::new(),
        };
        let Some(tree) = transaction.tree else {
            result.transactions.push(entry);
            continue;
        };
        let mut interp = Interp::new(program, given, RUN_STEPS);
        let mut frame = Frame::new(info.unit);
        frame.root = Some(Rc::clone(&tree));
        match fill(&mut interp, request, info, frame) {
            Ok((document, fixes)) => {
                entry.document = document;
                entry.fixes = fixes;
            }
            Err(Halt::Budget) => result.error = Some(budget()),
        }
        entry.unread = unread(&tree)
            .into_iter()
            .map(|(segment, position)| format!("{}{position:02}", tree.segments[segment].path))
            .collect();
        absorb(&mut result, &mut entry, interp);
        result.transactions.push(entry);
        if result.error.is_some() {
            break;
        }
    }
    result
}

fn absorb(result: &mut RunResult, entry: &mut TransactionResult, interp: Interp<'_>) {
    result.steps += interp.steps();
    result.needs.extend(interp.needs.iter().cloned());
    result.needs.sort();
    result.needs.dedup();
    result.unmapped.extend(interp.unmapped);
    result.coverage.extend(interp.coverage);
    entry.outcomes = interp.outcomes;
    entry.lineage = interp.lineage;
}

fn run_rows(
    program: &Program,
    given: &Given,
    request: &RunRequest,
    info: &MapInfo,
    text: &str,
) -> RunResult {
    let columns: Vec<(crate::lading::syntax::ast::CsvColumn, Type)> =
        match program.units[info.unit].header() {
            Header::Map {
                from: Endpoint::Csv { columns, .. },
                ..
            } => match &info.source {
                Source::Csv(types) => columns
                    .iter()
                    .cloned()
                    .zip(types.iter().map(|(_, ty)| ty.clone()))
                    .collect(),
                _ => Vec::new(),
            },
            _ => Vec::new(),
        };
    let zone = program
        .partner
        .as_ref()
        .and_then(|partner| partner.zone.as_deref())
        .and_then(ZoneValue::parse)
        .unwrap_or(ZoneValue::UTC);
    let (rows, problems) = csv::rows(text, &columns, zone);
    if rows.len() > csv::MAX_ROWS {
        return failed(Message::with(
            message!("A file holds at most {0} rows"),
            [Arg::from(csv::MAX_ROWS)],
        ));
    }
    let mut result = run_document_input(program, given, request, info, ("rows", Value::list(rows)));
    if let Some(first) = problems.into_iter().next()
        && result.error.is_none()
    {
        result.error = Some(first);
    }
    result
}

/// Runs an inbound map over CSV rows or JSON: one document.
fn run_document_input(
    program: &Program,
    given: &Given,
    request: &RunRequest,
    info: &MapInfo,
    input: (&str, Value),
) -> RunResult {
    let mut result = RunResult::default();
    let mut interp = Interp::new(program, given, RUN_STEPS);
    let mut frame = Frame::new(info.unit);
    frame.bind(input.0, input.1);
    let mut entry = TransactionResult {
        index: 0,
        control: String::new(),
        status: "accepted",
        findings: Vec::new(),
        document: Json::Null,
        outcomes: Vec::new(),
        fixes: Vec::new(),
        lineage: Vec::new(),
        unread: Vec::new(),
    };
    match fill(&mut interp, request, info, frame) {
        Ok((document, fixes)) => {
            entry.document = document;
            entry.fixes = fixes;
        }
        Err(Halt::Budget) => result.error = Some(budget()),
    }
    absorb(&mut result, &mut entry, interp);
    result.transactions.push(entry);
    result
}

#[allow(clippy::too_many_lines)]
fn run_outbound(
    program: &Program,
    given: &Given,
    request: &RunRequest,
    info: &MapInfo,
    document: &'static str,
    json: &Json,
) -> RunResult {
    let Some(guide) = info.target_guide else {
        return failed(Message::new(message!("The map writes no guide")));
    };
    let value = match from_json(program, json, &Type::Record(document), "") {
        Ok(value) => value,
        Err(problem) => {
            return failed(Message::with(
                message!("The document does not read: {0}"),
                [Arg::from(problem)],
            ));
        }
    };
    let mut interp = Interp::new(program, given, RUN_STEPS);
    let mut fixes = Vec::new();
    let mut value = value;
    if request.rules {
        for unit in rules_of(program, &request.map) {
            match interp.run_rules(unit, value.clone(), &mut fixes) {
                Ok(after) => value = after,
                Err(Halt::Budget) => return failed(budget()),
            }
        }
    }
    let mut frame = Frame::new(info.unit);
    frame.document = Some(value.clone());
    let stmts = statements(program, info.unit);
    let output = match interp.write_document(&mut frame, guide, &stmts) {
        Ok(output) => output,
        Err(Halt::Budget) => return failed(budget()),
    };
    let body = interp.lay_out(guide, &output);
    let (text, transaction) = envelope(program, guide, given, request.control, body);
    let compiled = &program.guides[guide].compiled;
    let validation = read(&ReadRequest {
        text: text.clone(),
        separators: None,
        guides: vec![GuideChoice {
            transaction_set: compiled.guide().transaction_set.clone(),
            guide: Some(compiled.guide().clone()),
            builtin: None,
        }],
        include_segments: false,
        fragment: false,
    })
    .map(|reading| {
        reading
            .interchanges
            .iter()
            .flat_map(|interchange| interchange.groups.iter())
            .flat_map(|group| group.transactions.iter())
            .flat_map(|transaction| transaction.findings.iter().cloned())
            .collect()
    })
    .unwrap_or_default();
    let mut result = RunResult {
        steps: interp.steps(),
        needs: interp.needs.iter().cloned().collect(),
        unmapped: interp.unmapped.clone(),
        coverage: interp.coverage.clone(),
        ..RunResult::default()
    };
    result.output = Some(WriteResult {
        text,
        transaction,
        document: to_json(&value),
        findings: output.findings,
        validation,
        outcomes: interp.outcomes,
        fixes,
        lineage: interp.lineage,
    });
    result
}

/// Whether an outbound result may be sent: no reject or hold, no finding that is an error.
#[must_use]
pub fn sendable(output: &WriteResult) -> bool {
    !output
        .outcomes
        .iter()
        .any(|outcome| matches!(outcome.kind, OutcomeKind2::Reject | OutcomeKind2::Hold))
        && !output
            .findings
            .iter()
            .any(|finding| finding.severity == "error")
        && !output
            .validation
            .iter()
            .any(|finding| finding.severity == crate::x12::finding::Severity::Error)
}

/// Writes the envelope around a transaction's segments: the interchange and the transaction
/// set alone.
#[allow(clippy::too_many_lines)]
pub fn envelope(
    program: &Program,
    guide: usize,
    given: &Given,
    control: Control,
    body: Vec<Vec<Element>>,
) -> (String, String) {
    let compiled = &program.guides[guide].compiled;
    let model = compiled.guide();
    let separators = program.partner_separators();
    let wire = crate::x12::separators::WireSeparators {
        element: separators[0].to_string(),
        segment: separators[1].to_string(),
        component: separators[2].to_string(),
        repetition: separators[3].to_string(),
        line_ending: crate::x12::separators::LineEnding::None,
    };
    let interchange = program.interchange();
    let party = |them: bool| {
        interchange
            .and_then(|interchange| {
                if them {
                    interchange.them.as_ref()
                } else {
                    interchange.us.as_ref()
                }
            })
            .map_or(("ZZ".to_owned(), String::new()), |party| {
                (party.qualifier.name.clone(), party.id.value.clone())
            })
    };
    let (us_qualifier, us_id) = party(false);
    let (them_qualifier, them_id) = party(true);
    let (theirs, ours) = interchange
        .and_then(|interchange| interchange.group.as_ref())
        .map_or((them_id.clone(), us_id.clone()), |(theirs, ours)| {
            (theirs.value.clone(), ours.value.clone())
        });
    let usage = interchange
        .and_then(|interchange| interchange.usage.as_ref())
        .map_or(
            "T",
            |usage| if usage.name == "production" { "P" } else { "T" },
        );
    let zone = program
        .partner
        .as_ref()
        .and_then(|partner| partner.zone.as_deref())
        .and_then(ZoneValue::parse)
        .unwrap_or(ZoneValue::UTC);
    let (days, seconds, _) = local(given.now, zone);
    let (year, month, day) = civil_from_days(days);
    let five = model.version.starts_with("005");
    let text_element = |value: String| Element::Text(value);
    let isa = vec![
        text_element("ISA".into()),
        text_element("00".into()),
        text_element(String::new()),
        text_element("00".into()),
        text_element(String::new()),
        text_element(us_qualifier),
        text_element(us_id),
        text_element(them_qualifier),
        text_element(them_id),
        text_element(format!("{:02}{month:02}{day:02}", year.rem_euclid(100))),
        text_element(format!("{:02}{:02}", seconds / 3600, seconds / 60 % 60)),
        text_element(if five {
            separators[3].to_string()
        } else {
            "U".into()
        }),
        text_element(if five { "00501".into() } else { "00401".into() }),
        text_element(format!("{:09}", control.interchange % 1_000_000_000)),
        text_element("0".into()),
        text_element(usage.into()),
        text_element(separators[2].to_string()),
    ];
    let gs = vec![
        text_element("GS".into()),
        text_element(model.functional_group.clone()),
        text_element(ours),
        text_element(theirs),
        text_element(format!("{year:04}{month:02}{day:02}")),
        text_element(format!("{:02}{:02}", seconds / 3600, seconds / 60 % 60)),
        text_element(control.group.to_string()),
        text_element("X".into()),
        text_element(model.version.clone()),
    ];
    let st_control = format!("{:04}", control.transaction % 1_000_000_000);
    let st = vec![
        text_element("ST".into()),
        text_element(model.transaction_set.clone()),
        text_element(st_control.clone()),
    ];
    let se = vec![
        text_element("SE".into()),
        text_element(String::new()),
        text_element(st_control),
    ];
    let ge = vec![
        text_element("GE".into()),
        text_element(String::new()),
        text_element(control.group.to_string()),
    ];
    let iea = vec![
        text_element("IEA".into()),
        text_element(String::new()),
        text_element(format!("{:09}", control.interchange % 1_000_000_000)),
    ];
    let mut transaction_segments = vec![st];
    transaction_segments.extend(body);
    transaction_segments.push(se);
    let mut all = vec![isa, gs];
    all.extend(transaction_segments.iter().cloned());
    all.push(ge);
    all.push(iea);
    let text = write(&WriteRequest {
        separators: Some(wire.clone()),
        segments: all,
        counts: true,
    })
    .map(|written| written.text)
    .unwrap_or_default();
    let transaction = write(&WriteRequest {
        separators: Some(wire),
        segments: transaction_segments,
        counts: true,
    })
    .map(|written| written.text)
    .unwrap_or_default();
    (text, transaction)
}

#[cfg(test)]
mod tests {
    use serde_json::json;

    use super::{LookupAnswer, RunInput, RunRequest, run};
    use crate::lading::check::check;
    use crate::lading::check::corpus::{acme, context};

    fn request(map: &str, input: RunInput) -> RunRequest {
        RunRequest {
            map: map.into(),
            input,
            settings: std::collections::BTreeMap::default(),
            lookups: Vec::new(),
            now: Some("2026-01-09T18:00:00Z".into()),
            control: super::Control::default(),
            rules: true,
        }
    }

    #[test]
    fn a_tender_becomes_a_load_tender() {
        let project = acme();
        let program = check(&project, &context());
        let sample = project
            .sample("samples/two-stops.x12")
            .expect("sample")
            .content
            .clone();
        let first = run(
            &program,
            &request("tenderIn", RunInput::X12(sample.clone())),
        );
        assert!(first.error.is_none(), "{:?}", first.error);
        let codes: Vec<(&str, &str)> = first
            .needs
            .iter()
            .map(|need| (need.kind.as_str(), need.code.as_str()))
            .collect();
        assert!(codes.contains(&("Customer", "ACME-BT")), "{codes:?}");
        assert!(codes.contains(&("Location", "STORE-7")), "{codes:?}");
        let mut again = request("tenderIn", RunInput::X12(sample));
        again.lookups = first
            .needs
            .iter()
            .map(|need| LookupAnswer {
                kind: need.kind.clone(),
                code: need.code.clone(),
                found: (need.code != "STORE-7").then(|| super::Found {
                    id: format!("id-{}", need.code),
                    label: need.code.clone(),
                    zone: Some("America/Chicago".into()),
                }),
            })
            .collect();
        let second = run(&program, &again);
        assert!(second.needs.is_empty(), "{:?}", second.needs);
        assert_eq!(second.unmapped.len(), 1);
        let transaction = &second.transactions[0];
        let document = &transaction.document;
        assert_eq!(document["key"], "L-1001", "{document:#}");
        assert_eq!(document["purpose"], "Original");
        assert_eq!(document["bol"], "BOL-55");
        assert_eq!(document["refs"]["PO"], "PO-9");
        assert_eq!(document["customer"]["id"], "id-ACME-BT");
        assert_eq!(document["weight"]["value"], "42000");
        assert_eq!(document["total"]["amount"], "1850.00");
        let stops = document["stops"].as_array().expect("stops");
        assert_eq!(stops.len(), 2);
        assert_eq!(stops[0]["type"], "Pickup");
        assert_eq!(stops[0]["name"], "Acme Dc");
        assert_eq!(stops[0]["address"]["city"], "DALLAS");
        assert_eq!(
            stops[0]["window"]["start"]["at"],
            "2026-01-10T08:00:00-06:00"
        );
        assert_eq!(
            stops[1]["address"]["postal"], "60601",
            "the nine-digit zip is fixed"
        );
        assert_eq!(transaction.fixes.len(), 1);
        let city = transaction
            .lineage
            .iter()
            .find(|entry| entry.path == "stops[0].name")
            .expect("lineage");
        assert_eq!(city.sources[0].path, "0300[1].0310.N102");
        assert!(
            city.steps.contains(&"titleCase".to_owned()),
            "{:?}",
            city.steps
        );
        assert!(
            transaction.outcomes.is_empty(),
            "{:?}",
            transaction.outcomes
        );
    }

    #[test]
    fn a_status_writes_a_214() {
        let program = check(&acme(), &context());
        let document = json!({
            "key": "L-1001",
            "proNumber": "PRO-1",
            "events": [{ "kind": "ArrivedDelivery", "at": "2026-01-10T08:30[America/Chicago]", "city": "Chicago", "state": "IL" }]
        });
        let result = run(
            &program,
            &request("statusOut", RunInput::Document(document)),
        );
        assert!(result.error.is_none(), "{:?}", result.error);
        let output = result.output.expect("output");
        assert!(
            output
                .transaction
                .contains("AT7*X1*NS***20260110*0830*CT~MS1*Chicago*IL~"),
            "{}",
            output.transaction
        );
        assert!(
            output
                .transaction
                .starts_with("ST*214*0001~B10*PRO-1*L-1001*TRNV~LX*1~"),
            "{}",
            output.transaction
        );
        assert!(output.text.starts_with("ISA*00*"), "{}", output.text);
        assert!(output.validation.is_empty(), "{:?}", output.validation);
        assert!(output.findings.is_empty(), "{:?}", output.findings);
    }
}
