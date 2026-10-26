//! Reads a whole text: segments, envelopes, the loop tree of every transaction set, and every
//! finding, with the acknowledgment each envelope has earned.
//!
//! This is `x12.read`. One call reads one file however many interchanges it holds, so the host
//! makes one call per file, never one per segment. Each transaction set is read with the guide
//! the caller chose for its set, or the base guide for its set and version; a set no guide covers
//! is not supported, which is what its acknowledgment says.
//!
//! A transaction set with an error is rejected, one with only warnings is accepted with errors,
//! and anything else is accepted. A group is accepted when all its sets are, partially accepted
//! when some are rejected, and rejected when all are or the group itself is in error.

use std::collections::BTreeMap;

use serde::{Deserialize, Serialize};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::ops::RequestError;

use super::element::{CheckOptions, ElementIssue, check_elements};
use super::envelope::{
    GroupSpan, InterchangeSpan, Scanned, Trailer, TransactionSpan, assemble, scan,
};
use super::finding::{Finding, FindingKind, Level, Severity};
use super::guide::builtin;
use super::guide::compiled::CompiledGuide;
use super::guide::model::{ElementDef, ElementType, Guide, SegmentDef, Usage};
use super::separators::{SeparatorSource, Separators, WireSeparators};
use super::structure::walk;

/// The most findings one transaction set reports; past this a single `findingsTruncated` says
/// the rest were left out.
pub const MAX_FINDINGS_PER_TRANSACTION: usize = 1000;

/// What `x12.read` takes.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ReadRequest {
    /// The text to read.
    pub text: String,
    /// The separators for text without an ISA, normally the partner profile's.
    #[serde(default)]
    pub separators: Option<WireSeparators>,
    /// The guide to read each transaction set with; sets not listed use their base guide.
    #[serde(default)]
    pub guides: Vec<GuideChoice>,
    /// Whether to list every segment with its elements, as the inspector needs.
    #[serde(default = "yes")]
    pub include_segments: bool,
    /// Whether the text is a transaction set on its own, which may leave out the ISA and GS.
    #[serde(default)]
    pub fragment: bool,
}

fn yes() -> bool {
    true
}

/// The guide for one transaction set: a partner's guide, or a base guide by ID.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GuideChoice {
    /// The transaction set it is for (ST01).
    pub transaction_set: String,
    /// A guide given in full.
    #[serde(default)]
    pub guide: Option<Guide>,
    /// A base guide by its ID.
    #[serde(default)]
    pub builtin: Option<String>,
}

/// What reading a text found.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Reading {
    /// The separators of the first interchange, or the ones the text was read with.
    pub separators: WireSeparators,
    /// Where they came from.
    pub separator_source: SeparatorSource,
    /// The interchanges in order.
    pub interchanges: Vec<InterchangeReading>,
    /// Every segment, when asked for.
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub segments: Vec<SegmentReading>,
    /// The name and elements of every segment ID the text uses, from the guides it was read
    /// with, when segments are asked for.
    #[serde(skip_serializing_if = "BTreeMap::is_empty")]
    pub definitions: BTreeMap<String, SegmentSummary>,
    /// Findings about the text outside any interchange.
    pub findings: Vec<Finding>,
    /// Counts.
    pub summary: Summary,
}

/// A segment's name and its elements' names, for people reading the text.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SegmentSummary {
    /// The segment's name.
    pub name: String,
    /// Its elements, element 01 first.
    pub elements: Vec<ElementSummary>,
}

/// An element's or component's reference, name and usage.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ElementSummary {
    /// The data element reference number.
    #[serde(rename = "ref")]
    pub reference: String,
    /// Its name.
    pub name: String,
    /// Its data type: `N2` is a number with two implied decimal places.
    #[serde(rename = "type")]
    pub kind: ElementType,
    /// Whether it must appear.
    pub usage: Usage,
    /// Its components, for a composite.
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub components: Vec<ElementSummary>,
}

impl From<&SegmentDef> for SegmentSummary {
    fn from(definition: &SegmentDef) -> Self {
        Self {
            name: definition.name.clone(),
            elements: definition
                .elements
                .iter()
                .map(ElementSummary::from)
                .collect(),
        }
    }
}

impl From<&ElementDef> for ElementSummary {
    fn from(element: &ElementDef) -> Self {
        Self {
            reference: element.reference.clone(),
            name: element.name.clone(),
            kind: element.kind,
            usage: element.usage,
            components: element
                .components
                .as_deref()
                .unwrap_or_default()
                .iter()
                .map(Self::from)
                .collect(),
        }
    }
}

/// One interchange.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct InterchangeReading {
    /// Its position in the file, from 0.
    pub index: usize,
    /// Its first segment's index.
    pub first_segment: usize,
    /// Its last segment's index.
    pub last_segment: usize,
    /// The ISA's values, when there is an ISA.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub header: Option<IsaHeader>,
    /// Its separators.
    pub separators: WireSeparators,
    /// Whether the ISA had its fixed width.
    pub fixed_width: bool,
    /// The IEA, when there is one.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trailer: Option<TrailerReading>,
    /// Its groups.
    pub groups: Vec<GroupReading>,
    /// Findings about the interchange itself.
    pub findings: Vec<Finding>,
    /// What a TA1 for it says.
    pub acknowledgment: InterchangeAcknowledgment,
}

/// The ISA's values, trimmed of their padding. ISA02 and ISA04 are left out, since they can
/// carry a password.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct IsaHeader {
    /// ISA01.
    pub authorization_qualifier: String,
    /// ISA03.
    pub security_qualifier: String,
    /// ISA05.
    pub sender_qualifier: String,
    /// ISA06.
    pub sender_id: String,
    /// ISA07.
    pub receiver_qualifier: String,
    /// ISA08.
    pub receiver_id: String,
    /// ISA09, `YYMMDD`.
    pub date: String,
    /// ISA10, `HHMM`.
    pub time: String,
    /// ISA12.
    pub version: String,
    /// ISA13.
    pub control_number: String,
    /// ISA14: `1` when the sender asks for a TA1.
    pub acknowledgment_requested: String,
    /// ISA15: `P`, `T` or `I`.
    pub usage: String,
}

/// What an AK9 says about a group: rejected when the group itself is in error or every set
/// is, partially accepted when some are rejected, accepted with errors when anything only
/// warns, accepted otherwise.
fn group_acknowledgment(
    transactions: &[TransactionReading],
    findings: &[Finding],
    trailer: Option<&Trailer>,
) -> GroupAcknowledgment {
    let received = transactions.len();
    let accepted = transactions
        .iter()
        .filter(|transaction| transaction.status != TransactionStatus::Rejected)
        .count();
    let with_errors = transactions
        .iter()
        .any(|transaction| transaction.status == TransactionStatus::AcceptedWithErrors);
    let errors: Vec<&'static str> = findings
        .iter()
        .filter(|finding| finding.is_error() && !finding.code.is_empty())
        .map(|finding| finding.code)
        .take(5)
        .collect();
    let group_error = findings.iter().any(Finding::is_error);
    let code = if group_error || (received > 0 && accepted == 0) {
        "R"
    } else if accepted < received {
        "P"
    } else if with_errors
        || findings
            .iter()
            .any(|finding| finding.severity == Severity::Warning)
    {
        "E"
    } else {
        "A"
    };
    GroupAcknowledgment {
        code,
        included: trailer
            .and_then(|trailer| trailer.count.parse().ok())
            .unwrap_or(received),
        received,
        accepted: if group_error { 0 } else { accepted },
        errors,
    }
}

/// What a TA1 says about an interchange.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct InterchangeAcknowledgment {
    /// TA104: `A`, `E` or `R`.
    pub code: &'static str,
    /// TA105: `000` when nothing was wrong, else the first error's note code.
    pub note: &'static str,
}

/// One functional group.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct GroupReading {
    /// Its position in the file, from 0.
    pub index: usize,
    /// Its first segment's index.
    pub first_segment: usize,
    /// Its last segment's index.
    pub last_segment: usize,
    /// The GS's values, when there is a GS.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub header: Option<GsHeader>,
    /// The GE, when there is one.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trailer: Option<TrailerReading>,
    /// Its transaction sets.
    pub transactions: Vec<TransactionReading>,
    /// Findings about the group itself.
    pub findings: Vec<Finding>,
    /// What an AK9 for it says.
    pub acknowledgment: GroupAcknowledgment,
}

/// The GS's values.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct GsHeader {
    /// GS01.
    pub functional_id: String,
    /// GS02.
    pub sender: String,
    /// GS03.
    pub receiver: String,
    /// GS04, `CCYYMMDD`.
    pub date: String,
    /// GS05.
    pub time: String,
    /// GS06.
    pub control_number: String,
    /// GS07.
    pub agency: String,
    /// GS08.
    pub version: String,
}

/// What an AK9 says about a group.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct GroupAcknowledgment {
    /// AK901: `A`, `E`, `P` or `R`.
    pub code: &'static str,
    /// AK902: the sets GE01 says the group holds.
    pub included: usize,
    /// AK903: the sets received.
    pub received: usize,
    /// AK904: the sets accepted.
    pub accepted: usize,
    /// AK905 to AK909: the group's own error codes.
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub errors: Vec<&'static str>,
}

/// One transaction set.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TransactionReading {
    /// Its position in the file, from 0.
    pub index: usize,
    /// Its first segment's index.
    pub first_segment: usize,
    /// Its last segment's index.
    pub last_segment: usize,
    /// ST01.
    pub transaction_set: String,
    /// ST02.
    pub control_number: String,
    /// ST03, when there is one.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub convention: String,
    /// The guide it was read with, empty when none covers it.
    pub guide: String,
    /// The SE, when there is one.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trailer: Option<TrailerReading>,
    /// Whether it is accepted.
    pub status: TransactionStatus,
    /// Every finding about it, its segments and their elements.
    pub findings: Vec<Finding>,
}

/// A trailer's values.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TrailerReading {
    /// The trailer segment's index.
    pub segment: usize,
    /// Its count element.
    pub count: String,
    /// Its control number element.
    pub control_number: String,
}

impl From<&Trailer> for TrailerReading {
    fn from(trailer: &Trailer) -> Self {
        Self {
            segment: trailer.segment,
            count: trailer.count.clone(),
            control_number: trailer.control_number.clone(),
        }
    }
}

/// Whether a transaction set is accepted.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum TransactionStatus {
    /// No finding (AK501 `A`).
    Accepted,
    /// Only warnings (AK501 `E`).
    AcceptedWithErrors,
    /// At least one error (AK501 `R`).
    Rejected,
}

impl TransactionStatus {
    /// The AK501/IK501 code.
    #[must_use]
    pub fn code(self) -> &'static str {
        match self {
            Self::Accepted => "A",
            Self::AcceptedWithErrors => "E",
            Self::Rejected => "R",
        }
    }
}

/// One segment.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SegmentReading {
    /// Its position in the file, from 0.
    pub index: usize,
    /// Its ID.
    pub id: String,
    /// Its first byte.
    pub start: usize,
    /// Just past its last byte, before the terminator.
    pub end: usize,
    /// The interchange it is in.
    pub interchange: usize,
    /// The transaction set it is in.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub transaction: Option<usize>,
    /// Its position in that transaction set, ST being 1.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub position: Option<u32>,
    /// The loops around it, outermost first: `0300/0310`.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub loop_path: String,
    /// Its elements as written, element 01 first.
    pub elements: Vec<String>,
}

/// Counts over the whole reading.
#[derive(Debug, Clone, Copy, Default, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Summary {
    /// Segments read.
    pub segments: usize,
    /// Interchanges.
    pub interchanges: usize,
    /// Functional groups.
    pub groups: usize,
    /// Transaction sets.
    pub transactions: usize,
    /// Transaction sets rejected.
    pub rejected: usize,
    /// Findings that are errors.
    pub errors: usize,
    /// Findings that are warnings.
    pub warnings: usize,
    /// Findings that are only noted.
    pub infos: usize,
}

enum Chosen<'a> {
    Given(&'a CompiledGuide),
    Base(&'static CompiledGuide, bool),
}

impl Chosen<'_> {
    fn guide(&self) -> &CompiledGuide {
        match self {
            Self::Given(guide) | Self::Base(guide, _) => guide,
        }
    }
}

/// What reading a transaction set with its guide gave: the guide used, the group it expects,
/// each segment's loop path and the guide's severity for every finding kind.
#[derive(Default)]
struct GuideOutcome {
    guide_id: String,
    expected_group: Option<String>,
    loop_paths: Vec<String>,
    severities: Vec<(FindingKind, Severity)>,
    definitions: Vec<(String, SegmentSummary)>,
}

struct SegmentMeta {
    interchange: usize,
    transaction: Option<usize>,
    position: Option<u32>,
    loop_path: String,
}

struct Reader<'r, 'a> {
    segments: &'r [Scanned<'a>],
    given: Vec<(String, CompiledGuide)>,
    meta: Vec<SegmentMeta>,
    group_count: usize,
    transaction_count: usize,
    describe: bool,
    definitions: BTreeMap<String, SegmentSummary>,
}

/// Reads `request.text`.
pub fn read(request: &ReadRequest) -> Result<Reading, RequestError> {
    let fallback = match &request.separators {
        Some(wire) => Some(Separators::from_wire(wire).ok_or_else(|| {
            RequestError::invalid_input(
                "separators must be single ASCII characters that are not letters, digits or spaces, and must differ",
            )
        })?),
        None => None,
    };
    let given = compile_choices(&request.guides)?;
    let scanned = scan(&request.text, fallback);
    let envelopes = assemble(&scanned.segments, request.fragment);

    let mut reader = Reader {
        segments: &scanned.segments,
        given,
        meta: (0..scanned.segments.len())
            .map(|_| SegmentMeta {
                interchange: 0,
                transaction: None,
                position: None,
                loop_path: String::new(),
            })
            .collect(),
        group_count: 0,
        transaction_count: 0,
        describe: request.include_segments,
        definitions: BTreeMap::new(),
    };

    let mut nested = envelopes.findings;
    let mut interchanges = Vec::with_capacity(envelopes.interchanges.len());
    let mut control_numbers: Vec<String> = Vec::new();
    for (index, span) in envelopes.interchanges.iter().enumerate() {
        interchanges.push(reader.interchange(index, span, &mut nested, &mut control_numbers));
    }
    let mut file_findings = nested;
    if !request.fragment
        && scanned
            .segments
            .first()
            .is_some_and(|first| first.segment.id != "ISA")
        && interchanges
            .iter()
            .all(|interchange| interchange.header.is_none())
    {
        file_findings.insert(
            0,
            Finding::new(
                FindingKind::InterchangeHeaderMissing,
                Message::new(message!(
                    "The text does not start with an ISA interchange header"
                )),
            ),
        );
    }

    let summary = summarize(scanned.segments.len(), &file_findings, &interchanges);
    let segments = if request.include_segments {
        reader.describe_envelopes();
        reader.segment_readings()
    } else {
        Vec::new()
    };
    Ok(Reading {
        separators: scanned.separators.to_wire(),
        separator_source: scanned.source,
        interchanges,
        segments,
        definitions: reader.definitions,
        findings: file_findings,
        summary,
    })
}

/// Counts the reading's segments, envelopes, rejected sets and findings by severity.
fn summarize(
    segments: usize,
    file_findings: &[Finding],
    interchanges: &[InterchangeReading],
) -> Summary {
    let mut summary = Summary {
        segments,
        interchanges: interchanges.len(),
        ..Summary::default()
    };
    let mut count = |findings: &[Finding]| {
        for finding in findings {
            match finding.severity {
                Severity::Error => summary.errors += 1,
                Severity::Warning => summary.warnings += 1,
                Severity::Info => summary.infos += 1,
            }
        }
    };
    count(file_findings);
    for interchange in interchanges {
        count(&interchange.findings);
        for group in &interchange.groups {
            count(&group.findings);
            for transaction in &group.transactions {
                count(&transaction.findings);
            }
        }
    }
    for interchange in interchanges {
        summary.groups += interchange.groups.len();
        for group in &interchange.groups {
            summary.transactions += group.transactions.len();
            summary.rejected += group
                .transactions
                .iter()
                .filter(|transaction| transaction.status == TransactionStatus::Rejected)
                .count();
        }
    }
    summary
}

fn compile_choices(choices: &[GuideChoice]) -> Result<Vec<(String, CompiledGuide)>, RequestError> {
    let mut given = Vec::new();
    for choice in choices {
        let guide = match (&choice.guide, &choice.builtin) {
            (Some(guide), None) => guide.clone(),
            (None, Some(id)) => builtin::by_id(id)
                .ok_or_else(|| {
                    RequestError::invalid_input(format!("no base guide is named {id:?}"))
                })?
                .guide()
                .clone(),
            _ => {
                return Err(RequestError::invalid_input(
                    "each guide choice names exactly one of guide or builtin",
                ));
            }
        };
        let compiled = CompiledGuide::compile(guide).map_err(|problems| {
            let details: Vec<String> = problems
                .iter()
                .map(|problem| format!("{}: {}", problem.path, problem.message.english()))
                .collect();
            RequestError::invalid_input(format!(
                "the guide for {} is not usable: {}",
                choice.transaction_set,
                details.join("; ")
            ))
        })?;
        given.push((choice.transaction_set.clone(), compiled));
    }
    Ok(given)
}

/// The TA1 note code for an ISA element by position.
fn isa_note(position: u32) -> &'static str {
    match position {
        1 => "010",
        2 => "011",
        3 => "012",
        4 => "013",
        5 => "005",
        6 => "006",
        7 => "007",
        8 => "008",
        9 => "014",
        10 => "015",
        11 => "016",
        12 => "017",
        13 => "018",
        14 => "019",
        15 => "020",
        _ => "027",
    }
}

impl<'a> Reader<'_, 'a> {
    fn scanned(&self, index: usize) -> &Scanned<'a> {
        &self.segments[index]
    }

    fn element(&self, index: usize, position: usize) -> String {
        let scanned = self.scanned(index);
        scanned
            .segment
            .element(&scanned.separators, position)
            .trim()
            .to_owned()
    }

    fn at(&self, finding: Finding, index: usize) -> Finding {
        let segment = &self.scanned(index).segment;
        finding.at_segment(index, segment.id, (segment.start, segment.end))
    }

    fn element_finding(&self, kind: FindingKind, issue: &ElementIssue, index: usize) -> Finding {
        self.at(Finding::new(kind, issue.message.clone()), index)
            .at_element(issue.position, &issue.reference)
            .at_component(issue.component, issue.repetition)
            .with_value(&issue.value)
            .with_span(issue.span)
    }

    fn interchange(
        &mut self,
        index: usize,
        span: &InterchangeSpan,
        nested: &mut Vec<Finding>,
        control_numbers: &mut Vec<String>,
    ) -> InterchangeReading {
        for segment in span.first..=span.last.min(self.segments.len().saturating_sub(1)) {
            if let Some(meta) = self.meta.get_mut(segment) {
                meta.interchange = index;
            }
        }
        let mut findings = take_findings(nested, span.first, span.last, Level::Interchange);
        let (separators, fixed_width) = span.header.map_or(
            (
                self.segments
                    .get(span.first)
                    .map(|scanned| scanned.separators)
                    .unwrap_or_default(),
                true,
            ),
            |header| {
                let scanned = self.scanned(header);
                (scanned.separators, scanned.fixed_width)
            },
        );

        let header = span
            .header
            .map(|header| self.isa_header(header, &mut findings, control_numbers));
        if let (Some(trailer), Some(header)) = (&span.trailer, &header) {
            self.check_iea(trailer, span.groups.len(), header, &mut findings);
        }

        let mut group_numbers: Vec<String> = Vec::new();
        let groups: Vec<GroupReading> = span
            .groups
            .iter()
            .map(|group| self.group(group, nested, &mut group_numbers))
            .collect();

        let acknowledgment = interchange_acknowledgment(&findings);
        InterchangeReading {
            index,
            first_segment: span.first,
            last_segment: span.last,
            header,
            separators: separators.to_wire(),
            fixed_width,
            trailer: span.trailer.as_ref().map(TrailerReading::from),
            groups,
            findings,
            acknowledgment,
        }
    }

    fn isa_header(
        &self,
        header: usize,
        findings: &mut Vec<Finding>,
        control_numbers: &mut Vec<String>,
    ) -> IsaHeader {
        self.check_isa(header, findings);
        let control_number = self.element(header, 13);
        if control_numbers.contains(&control_number) {
            findings.push(self.at(
                Finding::new(
                    FindingKind::DuplicateInterchangeControlNumber,
                    Message::with(
                        message!("Another interchange in this file has the control number {0}"),
                        [Arg::from(control_number.as_str())],
                    ),
                ),
                header,
            ));
        }
        control_numbers.push(control_number.clone());
        IsaHeader {
            authorization_qualifier: self.element(header, 1),
            security_qualifier: self.element(header, 3),
            sender_qualifier: self.element(header, 5),
            sender_id: self.element(header, 6),
            receiver_qualifier: self.element(header, 7),
            receiver_id: self.element(header, 8),
            date: self.element(header, 9),
            time: self.element(header, 10),
            version: self.element(header, 12),
            control_number,
            acknowledgment_requested: self.element(header, 14),
            usage: self.element(header, 15),
        }
    }

    fn check_iea(
        &self,
        trailer: &Trailer,
        groups: usize,
        header: &IsaHeader,
        findings: &mut Vec<Finding>,
    ) {
        if trailer.count != groups.to_string() {
            findings.push(self.at(
                Finding::new(
                    FindingKind::GroupCountMismatch,
                    Message::with(
                        message!("IEA01 says {0} functional groups, but the interchange has {1}"),
                        [Arg::from(trailer.count.as_str()), Arg::from(groups)],
                    ),
                ),
                trailer.segment,
            ));
        }
        if trailer.control_number != header.control_number {
            findings.push(self.at(
                Finding::new(
                    FindingKind::InterchangeControlNumberMismatch,
                    Message::with(
                        message!("IEA02 is {0}, but ISA13 is {1}"),
                        [
                            Arg::from(trailer.control_number.as_str()),
                            Arg::from(header.control_number.as_str()),
                        ],
                    ),
                ),
                trailer.segment,
            ));
        }
    }

    fn check_isa(&self, header: usize, findings: &mut Vec<Finding>) {
        let scanned = *self.scanned(header);
        let separators = scanned.separators;
        if !scanned.fixed_width {
            findings.push(self.at(
                Finding::new(
                    FindingKind::InterchangeHeaderSqueezed,
                    Message::new(message!("The ISA is not 106 characters, so its padding was lost; it was read by its separators")),
                ),
                header,
            ));
        }
        if !separators.distinct() {
            let (kind, message) = if separators.component == separators.element
                || Some(separators.component) == separators.repetition
            {
                (
                    FindingKind::InvalidComponentSeparator,
                    Message::new(message!(
                        "The component separator in ISA16 is the same as another separator"
                    )),
                )
            } else if separators.segment == separators.element
                || Some(separators.segment) == separators.repetition
            {
                (
                    FindingKind::InvalidSegmentTerminator,
                    Message::new(message!(
                        "The segment terminator is the same as another separator"
                    )),
                )
            } else {
                (
                    FindingKind::InvalidElementSeparator,
                    Message::new(message!(
                        "The ISA's separators are not distinct punctuation"
                    )),
                )
            };
            findings.push(self.at(Finding::new(kind, message), header));
        }
        let Some(definition) = builtin::envelope().segments.get("ISA") else {
            return;
        };
        let options = CheckOptions {
            ignore_lengths: !scanned.fixed_width,
        };
        for issue in check_elements(&scanned.segment, definition, &separators, options) {
            let padding_lost = !scanned.fixed_width
                && matches!(issue.position, 2 | 4)
                && issue.kind == FindingKind::MandatoryElementMissing;
            if matches!(issue.position, 11 | 16) || padding_lost {
                continue;
            }
            findings.push(
                self.element_finding(FindingKind::InvalidInterchangeHeader, &issue, header)
                    .with_code(isa_note(issue.position)),
            );
        }
        let version = self.element(header, 12);
        if version.len() == 5 && version.as_str() < "00402" {
            let standards = scanned.segment.element(&separators, 11);
            if standards != "U" && separators.repetition.is_none() {
                findings.push(
                    self.at(
                        Finding::new(
                            FindingKind::InvalidInterchangeHeader,
                            Message::with(
                                message!("ISA11 is {0}; before version 00402 it is the standards identifier U"),
                                [Arg::from(standards)],
                            ),
                        ),
                        header,
                    )
                    .at_element(11, "I10")
                    .with_value(standards)
                    .with_code("016"),
                );
            }
        }
    }

    fn group(
        &mut self,
        span: &GroupSpan,
        nested: &mut Vec<Finding>,
        group_numbers: &mut Vec<String>,
    ) -> GroupReading {
        let index = self.group_count;
        self.group_count += 1;
        let mut findings = take_findings(nested, span.first, span.last, Level::Group);
        let header = span
            .header
            .map(|header| self.gs_header(header, &mut findings, group_numbers));
        if let (Some(trailer), Some(header)) = (&span.trailer, &header) {
            self.check_ge(trailer, span.transactions.len(), header, &mut findings);
        }

        let version = header
            .as_ref()
            .map(|header| header.version.clone())
            .unwrap_or_default();
        let functional_id = header
            .as_ref()
            .map(|header| header.functional_id.clone())
            .unwrap_or_default();
        let mut transaction_numbers: Vec<String> = Vec::new();
        let mut mismatch_reported = false;
        let mut transactions = Vec::with_capacity(span.transactions.len());
        for transaction in &span.transactions {
            let (reading, expected_group) =
                self.transaction(transaction, &version, nested, &mut transaction_numbers);
            if let (Some(expected), Some(gs)) = (expected_group, span.header)
                && !functional_id.is_empty()
                && expected != functional_id
                && !mismatch_reported
            {
                mismatch_reported = true;
                findings.push(self.at(
                    Finding::new(
                        FindingKind::FunctionalGroupMismatch,
                        Message::with(
                            message!("GS01 is {0}, but transaction set {1} belongs in a {2} group"),
                            [
                                Arg::from(functional_id.as_str()),
                                Arg::from(reading.transaction_set.as_str()),
                                Arg::from(expected),
                            ],
                        ),
                    ),
                    gs,
                ));
            }
            transactions.push(reading);
        }

        let acknowledgment = group_acknowledgment(&transactions, &findings, span.trailer.as_ref());
        GroupReading {
            index,
            first_segment: span.first,
            last_segment: span.last,
            header,
            trailer: span.trailer.as_ref().map(TrailerReading::from),
            transactions,
            findings,
            acknowledgment,
        }
    }

    fn gs_header(
        &self,
        header: usize,
        findings: &mut Vec<Finding>,
        group_numbers: &mut Vec<String>,
    ) -> GsHeader {
        let scanned = *self.scanned(header);
        if let Some(definition) = builtin::envelope().segments.get("GS") {
            for issue in check_elements(
                &scanned.segment,
                definition,
                &scanned.separators,
                CheckOptions::default(),
            ) {
                if issue.position == 6 {
                    continue;
                }
                findings.push(self.element_finding(
                    FindingKind::InvalidGroupHeader,
                    &issue,
                    header,
                ));
            }
        }
        let control_number = self.element(header, 6);
        let numeric = (1..=9).contains(&control_number.len())
            && control_number.bytes().all(|byte| byte.is_ascii_digit());
        if !numeric {
            findings.push(
                self.at(
                    Finding::new(
                        FindingKind::InvalidGroupControlNumber,
                        Message::with(
                            message!(
                                "GS06 is {0}, but a group control number is one to nine digits"
                            ),
                            [Arg::from(control_number.as_str())],
                        ),
                    ),
                    header,
                )
                .at_element(6, "28")
                .with_value(&control_number),
            );
        }
        if group_numbers.contains(&control_number) {
            findings.push(self.at(
                Finding::new(
                    FindingKind::DuplicateGroupControlNumber,
                    Message::with(
                        message!("Another functional group in this interchange has the control number {0}"),
                        [Arg::from(control_number.as_str())],
                    ),
                ),
                header,
            ));
        }
        group_numbers.push(control_number.clone());
        GsHeader {
            functional_id: self.element(header, 1),
            sender: self.element(header, 2),
            receiver: self.element(header, 3),
            date: self.element(header, 4),
            time: self.element(header, 5),
            control_number,
            agency: self.element(header, 7),
            version: self.element(header, 8),
        }
    }

    fn check_ge(
        &self,
        trailer: &Trailer,
        transactions: usize,
        header: &GsHeader,
        findings: &mut Vec<Finding>,
    ) {
        if trailer.count != transactions.to_string() {
            findings.push(self.at(
                Finding::new(
                    FindingKind::TransactionCountMismatch,
                    Message::with(
                        message!("GE01 says {0} transaction sets, but the group has {1}"),
                        [Arg::from(trailer.count.as_str()), Arg::from(transactions)],
                    ),
                ),
                trailer.segment,
            ));
        }
        if trailer.control_number != header.control_number {
            findings.push(self.at(
                Finding::new(
                    FindingKind::GroupControlNumberMismatch,
                    Message::with(
                        message!("GE02 is {0}, but GS06 is {1}"),
                        [
                            Arg::from(trailer.control_number.as_str()),
                            Arg::from(header.control_number.as_str()),
                        ],
                    ),
                ),
                trailer.segment,
            ));
        }
    }

    fn choose(&self, set: &str, version: &str) -> Option<Chosen<'_>> {
        if let Some((_, guide)) = self.given.iter().find(|(given_set, _)| given_set == set) {
            return Some(Chosen::Given(guide));
        }
        builtin::for_set(set, version).map(|(guide, fell_back)| Chosen::Base(guide, fell_back))
    }

    fn transaction(
        &mut self,
        span: &TransactionSpan,
        group_version: &str,
        nested: &mut Vec<Finding>,
        numbers: &mut Vec<String>,
    ) -> (TransactionReading, Option<String>) {
        let index = self.transaction_count;
        self.transaction_count += 1;
        let mut findings = take_findings(nested, span.first, span.last, Level::Transaction);
        let first = span.first;
        let is_st = self.scanned(first).segment.id == "ST";
        let set = if is_st {
            self.element(first, 1)
        } else {
            String::new()
        };
        let control_number = if is_st {
            self.element(first, 2)
        } else {
            String::new()
        };
        let convention = if is_st {
            self.element(first, 3)
        } else {
            String::new()
        };

        self.check_st_se(span, &set, &control_number, numbers, &mut findings);

        for (offset, segment) in (span.first..=span.last).enumerate() {
            let meta = &mut self.meta[segment];
            meta.transaction = Some(index);
            meta.position = Some(u32::try_from(offset + 1).unwrap_or(u32::MAX));
        }

        let version = if convention.is_empty() {
            group_version.to_owned()
        } else {
            convention.clone()
        };
        let outcome = self.read_with_guide(span, &set, &version, &convention, &mut findings);
        for (offset, path) in outcome.loop_paths.into_iter().enumerate() {
            self.meta[span.first + offset].loop_path = path;
        }
        for (id, summary) in outcome.definitions {
            self.definitions.entry(id).or_insert(summary);
        }
        for finding in &mut findings {
            if let Some((_, severity)) = outcome
                .severities
                .iter()
                .find(|(kind, _)| *kind == finding.kind)
            {
                finding.severity = *severity;
            }
        }
        if findings.len() > MAX_FINDINGS_PER_TRANSACTION {
            let dropped = findings.len() - MAX_FINDINGS_PER_TRANSACTION;
            findings.truncate(MAX_FINDINGS_PER_TRANSACTION);
            findings.push(self.at(
                Finding::new(
                    FindingKind::FindingsTruncated,
                    Message::with(
                        message!("{0, plural, one {# more finding was} other {# more findings were}} left out"),
                        [Arg::from(dropped)],
                    ),
                ),
                first,
            ));
        }
        let status = if findings.iter().any(Finding::is_error) {
            TransactionStatus::Rejected
        } else if findings
            .iter()
            .any(|finding| finding.severity == Severity::Warning)
        {
            TransactionStatus::AcceptedWithErrors
        } else {
            TransactionStatus::Accepted
        };
        (
            TransactionReading {
                index,
                first_segment: span.first,
                last_segment: span.last,
                transaction_set: set,
                control_number,
                convention,
                guide: outcome.guide_id,
                trailer: span.trailer.as_ref().map(TrailerReading::from),
                status,
                findings,
            },
            outcome.expected_group,
        )
    }

    fn check_st_se(
        &self,
        span: &TransactionSpan,
        set: &str,
        control_number: &str,
        numbers: &mut Vec<String>,
        findings: &mut Vec<Finding>,
    ) {
        let first = span.first;
        let is_st = self.scanned(first).segment.id == "ST";
        if !is_st || set.len() != 3 || !set.bytes().all(|byte| byte.is_ascii_digit()) {
            findings.push(
                self.at(
                    Finding::new(
                        FindingKind::InvalidTransactionSetIdentifier,
                        Message::with(
                            message!(
                                "ST01 is {0}, but a transaction set identifier is three digits"
                            ),
                            [Arg::from(set)],
                        ),
                    )
                    .at_element(1, "143")
                    .with_value(set),
                    first,
                ),
            );
        }
        if !(4..=9).contains(&control_number.len()) {
            findings.push(self.at(
            Finding::new(
                FindingKind::InvalidTransactionControlNumber,
                Message::with(
                    message!("ST02 is {0}, but a transaction set control number is four to nine characters"),
                    [Arg::from(control_number)],
                ),
            )
            .at_element(2, "329")
            .with_value(control_number),
            first,
        ));
        }
        if !control_number.is_empty() && numbers.iter().any(|number| number == control_number) {
            findings.push(self.at(
                Finding::new(
                    FindingKind::DuplicateTransactionControlNumber,
                    Message::with(
                        message!(
                            "Another transaction set in this group has the control number {0}"
                        ),
                        [Arg::from(control_number)],
                    ),
                ),
                first,
            ));
        }
        numbers.push(control_number.to_owned());
        if let Some(trailer) = &span.trailer {
            let actual = span.last - span.first + 1;
            if trailer.count != actual.to_string() {
                findings.push(self.at(
                    Finding::new(
                        FindingKind::SegmentCountMismatch,
                        Message::with(
                            message!("SE01 says {0} segments, but the transaction set has {1}"),
                            [Arg::from(trailer.count.as_str()), Arg::from(actual)],
                        ),
                    ),
                    trailer.segment,
                ));
            }
            if trailer.control_number != control_number {
                findings.push(self.at(
                    Finding::new(
                        FindingKind::TransactionControlNumberMismatch,
                        Message::with(
                            message!("SE02 is {0}, but ST02 is {1}"),
                            [
                                Arg::from(trailer.control_number.as_str()),
                                Arg::from(control_number),
                            ],
                        ),
                    ),
                    trailer.segment,
                ));
            }
        }
    }

    fn read_with_guide(
        &self,
        span: &TransactionSpan,
        set: &str,
        version: &str,
        convention: &str,
        findings: &mut Vec<Finding>,
    ) -> GuideOutcome {
        let first = span.first;
        let mut outcome = GuideOutcome::default();
        match self.choose(set, version) {
            None => findings.push(self.at(
                Finding::new(
                    FindingKind::TransactionSetNotSupported,
                    Message::with(
                        message!("No guide covers transaction set {0}"),
                        [Arg::from(set)],
                    ),
                ),
                first,
            )),
            Some(chosen) => {
                let guide = chosen.guide();
                outcome.guide_id.clone_from(&guide.guide().id);
                outcome.expected_group = Some(guide.guide().functional_group.clone());
                if let (Chosen::Base(_, true), false) = (&chosen, version.is_empty()) {
                    findings.push(self.at(
                        Finding::new(
                            FindingKind::GuideVersionDiffers,
                            Message::with(
                                message!("Read with the {0} guide, since none is for version {1}"),
                                [
                                    Arg::from(guide.guide().version.as_str()),
                                    Arg::from(version),
                                ],
                            ),
                        ),
                        first,
                    ));
                }
                if !convention.is_empty() && convention != guide.guide().version {
                    findings.push(self.at(
                        Finding::new(
                            FindingKind::ConventionReferenceMismatch,
                            Message::with(
                                message!("ST03 is {0}, but the guide is for {1}"),
                                [
                                    Arg::from(convention),
                                    Arg::from(guide.guide().version.as_str()),
                                ],
                            ),
                        ),
                        first,
                    ));
                }
                let (structural, paths) = self.walk_transaction(guide, span);
                findings.extend(structural);
                outcome.loop_paths = paths;
                if self.describe {
                    for index in span.first..=span.last {
                        let id = self.scanned(index).segment.id;
                        let known = self.definitions.contains_key(id)
                            || outcome.definitions.iter().any(|(seen, _)| seen == id);
                        if let (false, Some(definition)) = (known, guide.segment(id)) {
                            outcome
                                .definitions
                                .push((id.to_owned(), SegmentSummary::from(definition)));
                        }
                    }
                }
                outcome.severities = FindingKind::ALL
                    .iter()
                    .map(|kind| (*kind, guide.severity(*kind)))
                    .collect();
            }
        }
        outcome
    }

    fn walk_transaction(
        &self,
        guide: &CompiledGuide,
        span: &TransactionSpan,
    ) -> (Vec<Finding>, Vec<String>) {
        let indexes: Vec<usize> = (span.first..=span.last).collect();
        let ids: Vec<&str> = indexes
            .iter()
            .map(|index| self.segments[*index].segment.id)
            .collect();
        let result = walk(guide, &ids);
        let mut findings = Vec::new();
        for issue in &result.issues {
            let index = indexes[issue.at];
            let position = u32::try_from(issue.at + 1).unwrap_or(u32::MAX);
            let mut finding = self
                .at(Finding::new(issue.kind, issue.message.clone()), index)
                .in_transaction(position, &issue.loop_id);
            finding.segment_id.clone_from(&issue.segment_id);
            findings.push(finding);
        }
        let mut paths = Vec::with_capacity(result.placements.len());
        for (offset, placement) in result.placements.iter().enumerate() {
            let index = indexes[offset];
            paths.push(placement.loop_path.clone());
            let scanned = *self.scanned(index);
            if matches!(scanned.segment.id, "ST" | "SE") {
                continue;
            }
            let Some(definition) = guide.segment(scanned.segment.id) else {
                continue;
            };
            let position = u32::try_from(offset + 1).unwrap_or(u32::MAX);
            for issue in check_elements(
                &scanned.segment,
                definition,
                &scanned.separators,
                CheckOptions::default(),
            ) {
                findings.push(
                    self.element_finding(issue.kind, &issue, index)
                        .in_transaction(position, &placement.loop_id),
                );
            }
        }
        (findings, paths)
    }

    /// Adds the envelope segments' definitions for every envelope segment the text has.
    fn describe_envelopes(&mut self) {
        for scanned in self.segments {
            let id = scanned.segment.id;
            if self.definitions.contains_key(id) {
                continue;
            }
            if let Some(definition) = builtin::envelope().segments.get(id) {
                self.definitions
                    .insert(id.to_owned(), SegmentSummary::from(definition));
            }
        }
    }

    fn segment_readings(&self) -> Vec<SegmentReading> {
        self.segments
            .iter()
            .enumerate()
            .map(|(index, scanned)| {
                let meta = &self.meta[index];
                SegmentReading {
                    index,
                    id: scanned.segment.id.to_owned(),
                    start: scanned.segment.start,
                    end: scanned.segment.end,
                    interchange: meta.interchange,
                    transaction: meta.transaction,
                    position: meta.position,
                    loop_path: meta.loop_path.clone(),
                    elements: scanned
                        .segment
                        .elements(&scanned.separators)
                        .map(str::to_owned)
                        .collect(),
                }
            })
            .collect()
    }
}

/// What a TA1 says about an interchange with these findings: rejected with the first error's
/// note code, accepted with errors when there are only warnings, accepted otherwise.
fn interchange_acknowledgment(findings: &[Finding]) -> InterchangeAcknowledgment {
    match findings.iter().find(|finding| finding.is_error()) {
        Some(finding) => InterchangeAcknowledgment {
            code: "R",
            note: if finding.code.len() == 3 {
                finding.code
            } else {
                "024"
            },
        },
        None if findings
            .iter()
            .any(|finding| finding.severity == Severity::Warning) =>
        {
            InterchangeAcknowledgment {
                code: "E",
                note: "000",
            }
        }
        None => InterchangeAcknowledgment {
            code: "A",
            note: "000",
        },
    }
}

/// Moves the findings at `level` about segments `first..=last` out of `findings`.
fn take_findings(
    findings: &mut Vec<Finding>,
    first: usize,
    last: usize,
    level: Level,
) -> Vec<Finding> {
    let mut taken = Vec::new();
    let mut kept = Vec::with_capacity(findings.len());
    for finding in findings.drain(..) {
        let inside = finding
            .segment
            .is_some_and(|segment| (first..=last).contains(&segment));
        if inside && finding.level == level {
            taken.push(finding);
        } else {
            kept.push(finding);
        }
    }
    *findings = kept;
    taken
}

#[cfg(test)]
mod tests {
    use super::{ReadRequest, TransactionStatus, read};
    use crate::x12::finding::FindingKind;

    pub(crate) const TENDER: &str = concat!(
        "ISA*00*          *00*          *ZZ*SIMPARTNER     *ZZ*TRENOVA        *260107*1200*^*00401*000000101*0*T*>~\n",
        "GS*SM*SIMPARTNER*TRENOVA*20260107*1200*101*X*004010~\n",
        "ST*204*0001~\n",
        "B2**SIML**SHIP-1**PP~\n",
        "B2A*00~\n",
        "L11*SHIP-1*CR~\n",
        "S5*1*CL~\n",
        "G62*10*20260107*Y*1504~\n",
        "N1*SF*Simulator Shipper*93*SIM-SHIP~\n",
        "N3*100 Simulator Way~\n",
        "N4*Chicago*IL*60601*US~\n",
        "S5*2*CU~\n",
        "G62*68*20260109*Z*1504~\n",
        "N1*ST*Simulator Consignee*93*SIM-CONS~\n",
        "N3*200 Receiver Road~\n",
        "N4*Dallas*TX*75201*US~\n",
        "L3*1000*G***150000~\n",
        "SE*16*0001~\n",
        "GE*1*101~\n",
        "IEA*1*000000101~\n",
    );

    fn request(text: &str) -> ReadRequest {
        ReadRequest {
            text: text.to_owned(),
            separators: None,
            guides: Vec::new(),
            include_segments: true,
            fragment: false,
        }
    }

    #[test]
    fn reads_a_clean_tender_and_accepts_it() {
        let reading = read(&request(TENDER)).expect("a reading");
        assert_eq!(
            reading.summary.errors, 0,
            "{:#?}",
            reading.interchanges[0].groups[0].transactions[0].findings
        );
        let interchange = &reading.interchanges[0];
        assert_eq!(interchange.acknowledgment.code, "A");
        assert_eq!(
            interchange
                .header
                .as_ref()
                .map(|header| header.sender_id.as_str()),
            Some("SIMPARTNER")
        );
        let group = &interchange.groups[0];
        assert_eq!(group.acknowledgment.code, "A");
        assert_eq!(
            (group.acknowledgment.received, group.acknowledgment.accepted),
            (1, 1)
        );
        let transaction = &group.transactions[0];
        assert_eq!(transaction.status, TransactionStatus::Accepted);
        assert_eq!(transaction.guide, "x12.004010.204");
        let n1 = reading
            .segments
            .iter()
            .find(|segment| segment.id == "N1")
            .expect("an N1");
        assert_eq!(n1.loop_path, "0300/0310");
        assert_eq!(n1.position, Some(7));
        assert_eq!(n1.elements[1], "Simulator Shipper");
        assert_eq!(reading.definitions["N1"].name, "Name");
        assert_eq!(
            reading.definitions["N1"].elements[0].name,
            "Entity Identifier Code"
        );
        assert_eq!(reading.definitions["ISA"].elements[5].reference, "I06");
    }

    #[test]
    fn rejects_a_tender_with_errors_and_says_where() {
        let text = TENDER.replace("N1*SF*", "N1**").replace("SE*16*", "SE*15*");
        let reading = read(&request(&text)).expect("a reading");
        let transaction = &reading.interchanges[0].groups[0].transactions[0];
        assert_eq!(transaction.status, TransactionStatus::Rejected);
        let kinds: Vec<_> = transaction
            .findings
            .iter()
            .map(|finding| finding.kind)
            .collect();
        assert!(
            kinds.contains(&FindingKind::MandatoryElementMissing),
            "{kinds:?}"
        );
        assert!(
            kinds.contains(&FindingKind::SegmentCountMismatch),
            "{kinds:?}"
        );
        let missing = transaction
            .findings
            .iter()
            .find(|finding| finding.kind == FindingKind::MandatoryElementMissing)
            .expect("the missing element");
        assert_eq!(missing.position, Some(7));
        assert_eq!(missing.loop_id, "0310");
        assert_eq!(missing.element, Some(1));
        assert_eq!(missing.element_ref, "98");
        assert_eq!(reading.interchanges[0].groups[0].acknowledgment.code, "R");
    }

    #[test]
    fn a_set_no_guide_covers_is_not_supported() {
        let text = TENDER.replace("ST*204*0001", "ST*856*0001");
        let reading = read(&request(&text)).expect("a reading");
        let transaction = &reading.interchanges[0].groups[0].transactions[0];
        assert_eq!(
            transaction.findings[0].kind,
            FindingKind::TransactionSetNotSupported
        );
        assert_eq!(transaction.findings[0].code, "1");
        assert!(transaction.guide.is_empty());
    }

    #[test]
    fn checks_the_envelope_counts_and_control_numbers() {
        let text = TENDER
            .replace("GE*1*101", "GE*2*102")
            .replace("IEA*1*000000101", "IEA*2*000000999");
        let reading = read(&request(&text)).expect("a reading");
        let interchange = &reading.interchanges[0];
        let kinds: Vec<_> = interchange
            .findings
            .iter()
            .map(|finding| finding.kind)
            .collect();
        assert_eq!(
            kinds,
            [
                FindingKind::GroupCountMismatch,
                FindingKind::InterchangeControlNumberMismatch
            ]
        );
        assert_eq!(interchange.acknowledgment.code, "R");
        assert_eq!(interchange.acknowledgment.note, "021");
        let group_kinds: Vec<_> = interchange.groups[0]
            .findings
            .iter()
            .map(|finding| finding.kind)
            .collect();
        assert_eq!(
            group_kinds,
            [
                FindingKind::TransactionCountMismatch,
                FindingKind::GroupControlNumberMismatch
            ]
        );
        assert_eq!(interchange.groups[0].acknowledgment.errors, ["5", "4"]);
    }

    #[test]
    fn reports_a_squeezed_isa_as_a_warning_only() {
        let text = TENDER
            .replace("SIMPARTNER     ", "SIMPARTNER")
            .replace("TRENOVA        ", "TRENOVA")
            .replacen("          ", "", 2);
        let reading = read(&request(&text)).expect("a reading");
        let interchange = &reading.interchanges[0];
        assert!(!interchange.fixed_width);
        assert_eq!(interchange.findings.len(), 1, "{:?}", interchange.findings);
        assert_eq!(
            interchange.findings[0].kind,
            FindingKind::InterchangeHeaderSqueezed
        );
        assert_eq!(interchange.acknowledgment.code, "E");
    }

    #[test]
    fn a_guide_may_lower_a_findings_severity() {
        let text = TENDER.replace("L3*1000*G***150000", "L3*1000*G***1500.00");
        let reading = read(&request(&text)).expect("a reading");
        let transaction = &reading.interchanges[0].groups[0].transactions[0];
        assert_eq!(transaction.status, TransactionStatus::AcceptedWithErrors);
        assert_eq!(reading.interchanges[0].groups[0].acknowledgment.code, "E");

        let mut guide = crate::x12::guide::builtin::by_id("x12.004010.204")
            .expect("the 204")
            .guide()
            .clone();
        guide.severities.insert(
            FindingKind::SegmentCountMismatch,
            crate::x12::finding::Severity::Warning,
        );
        let mut req = request(&TENDER.replace("SE*16*", "SE*99*"));
        req.guides.push(super::GuideChoice {
            transaction_set: String::from("204"),
            guide: Some(guide),
            builtin: None,
        });
        let reading = read(&req).expect("a reading");
        let transaction = &reading.interchanges[0].groups[0].transactions[0];
        assert_eq!(transaction.status, TransactionStatus::AcceptedWithErrors);
        assert_eq!(
            transaction.findings[0].kind,
            FindingKind::SegmentCountMismatch
        );
    }

    #[test]
    fn reads_a_fragment_with_the_callers_separators() {
        let mut req = request("ST|990|0001\nB1|SIML|SHIP-1|20260107|A\nSE|3|0001\n");
        req.fragment = true;
        req.separators = Some(crate::x12::separators::WireSeparators {
            element: String::from("|"),
            segment: String::from("\n"),
            ..crate::x12::separators::WireSeparators::default()
        });
        let reading = read(&req).expect("a reading");
        assert!(reading.findings.is_empty(), "{:?}", reading.findings);
        let transaction = &reading.interchanges[0].groups[0].transactions[0];
        assert_eq!(
            transaction.status,
            TransactionStatus::Accepted,
            "{:?}",
            transaction.findings
        );
    }

    #[test]
    fn refuses_unusable_requests() {
        let mut req = request(TENDER);
        req.separators = Some(crate::x12::separators::WireSeparators {
            element: String::from("A"),
            ..crate::x12::separators::WireSeparators::default()
        });
        assert!(read(&req).is_err());
        let mut req = request(TENDER);
        req.guides.push(super::GuideChoice {
            transaction_set: String::from("204"),
            guide: None,
            builtin: Some(String::from("nope")),
        });
        assert!(read(&req).is_err());
    }
}
