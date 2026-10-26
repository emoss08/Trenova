//! Builds 997 and 999 acknowledgments from a reading's groups.
//!
//! An acknowledgment is protocol, not a partner mapping, so it is never a template: it is built
//! here from what reading found. Each group gets its own interchange holding one transaction
//! set:
//!
//! - AK1 names the group (and, in a 999, its version);
//! - an AK2 loop per transaction set, with an AK3 (IK3) loop for each segment in error and its
//!   AK4s (IK4s) inside it, then AK5 (IK5) with `A`, `E` or `R` and the set's own error codes;
//! - AK9 with `A`, `E`, `P` or `R`, the three counts and the group's own error codes.
//!
//! A segment with its own finding (out of sequence, over its maximum use) carries that code in
//! AK304; a segment whose elements are its only problem carries 8. A missing segment gets an AK3
//! of its own naming it. A 997 can only carry the standard's codes, so an implementation code
//! from a partner's narrowing is written as the standard code it falls back to.
//!
//! The group in the request is the group of an `x12.read` reading, as that operation returned
//! it, so the host passes it back without reshaping it.

use std::collections::BTreeMap;

use serde::{Deserialize, Serialize};
use trenova_module_kit::ops::RequestError;

use super::finding::{FindingKind, Level, Severity};
use super::separators::WireSeparators;
use super::write::{Element, WriteRequest, write};

/// Which acknowledgment to build.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
pub enum AckKind {
    /// The functional acknowledgment, 004010.
    #[serde(rename = "997")]
    Functional,
    /// The implementation acknowledgment, 005010X231A1.
    #[serde(rename = "999")]
    Implementation,
}

/// What `x12.acknowledge` takes.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AcknowledgeRequest {
    /// 997 or 999.
    pub kind: AckKind,
    /// The separators to write with.
    #[serde(default)]
    pub separators: Option<WireSeparators>,
    /// One acknowledgment per group.
    pub acknowledgments: Vec<AckInput>,
}

/// One group to acknowledge and the envelope to send it in.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AckInput {
    /// The group as `x12.read` returned it.
    pub group: AckGroup,
    /// The envelope of the acknowledgment itself.
    pub envelope: AckEnvelope,
}

/// The parts of a reading's group an acknowledgment needs.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AckGroup {
    /// The GS, when the group had one.
    #[serde(default)]
    pub header: Option<AckGroupHeader>,
    /// The group's own findings.
    #[serde(default)]
    pub findings: Vec<AckFinding>,
    /// The group's transaction sets.
    #[serde(default)]
    pub transactions: Vec<AckTransaction>,
    /// What the reading decided for AK9.
    pub acknowledgment: AckGroupResult,
}

/// The GS values an AK1 repeats.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AckGroupHeader {
    /// GS01.
    pub functional_id: String,
    /// GS06.
    pub control_number: String,
    /// GS08.
    #[serde(default)]
    pub version: String,
}

/// AK9's values.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AckGroupResult {
    /// AK901.
    pub code: String,
    /// AK902.
    pub included: usize,
    /// AK903.
    pub received: usize,
    /// AK904.
    pub accepted: usize,
    /// AK905 to AK909.
    #[serde(default)]
    pub errors: Vec<String>,
}

/// The parts of a reading's transaction set an acknowledgment needs.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AckTransaction {
    /// ST01.
    pub transaction_set: String,
    /// ST02.
    pub control_number: String,
    /// ST03.
    #[serde(default)]
    pub convention: String,
    /// Accepted, accepted with errors, or rejected.
    pub status: AckStatus,
    /// Its findings.
    #[serde(default)]
    pub findings: Vec<AckFinding>,
}

/// A transaction set's status as a reading writes it.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum AckStatus {
    /// AK501 `A`.
    Accepted,
    /// AK501 `E`.
    AcceptedWithErrors,
    /// AK501 `R`.
    Rejected,
}

impl AckStatus {
    fn code(self) -> &'static str {
        match self {
            Self::Accepted => "A",
            Self::AcceptedWithErrors => "E",
            Self::Rejected => "R",
        }
    }
}

/// The parts of a finding an acknowledgment needs.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AckFinding {
    /// What was found.
    pub kind: FindingKind,
    /// How much it matters.
    pub severity: Severity,
    /// The segment's index in the reading.
    #[serde(default)]
    pub segment: Option<usize>,
    /// The segment's ID.
    #[serde(default)]
    pub segment_id: String,
    /// Its position in the transaction set.
    #[serde(default)]
    pub position: Option<u32>,
    /// Its loop.
    #[serde(default)]
    pub loop_id: String,
    /// The element's position.
    #[serde(default)]
    pub element: Option<u32>,
    /// The component's position.
    #[serde(default)]
    pub component: Option<u32>,
    /// The repetition's position.
    #[serde(default)]
    pub repetition: Option<u32>,
    /// The element's reference number.
    #[serde(default)]
    pub element_ref: String,
    /// A copy of the bad data.
    #[serde(default)]
    pub value: String,
}

/// The acknowledgment's own envelope.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AckEnvelope {
    /// ISA05.
    pub sender_qualifier: String,
    /// ISA06.
    pub sender_id: String,
    /// ISA07.
    pub receiver_qualifier: String,
    /// ISA08.
    pub receiver_id: String,
    /// GS02.
    pub application_sender: String,
    /// GS03.
    pub application_receiver: String,
    /// GS04, `CCYYMMDD`; ISA09 is its last six digits.
    pub date: String,
    /// GS05 and ISA10, `HHMM`.
    pub time: String,
    /// ISA13.
    pub interchange_control_number: String,
    /// GS06 and GE02.
    pub group_control_number: String,
    /// ST02 and SE02.
    pub transaction_control_number: String,
    /// ISA15: `P` or `T`.
    pub usage: String,
    /// ISA12; `00401` for a 997 and `00501` for a 999 when empty.
    #[serde(default)]
    pub interchange_version: String,
}

/// One acknowledgment written.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Acknowledgment {
    /// The whole interchange.
    pub text: String,
    /// AK901.
    pub code: String,
    /// Segments in the transaction set, ST to SE.
    pub segment_count: usize,
    /// How many transaction sets it reports on.
    pub transactions: usize,
}

/// What `x12.acknowledge` returns.
#[derive(Debug, Clone, Serialize)]
pub struct Acknowledgments {
    /// One per group, in the order given.
    pub acknowledgments: Vec<Acknowledgment>,
}

/// The most AK4s one AK3 carries.
const MAX_ELEMENT_NOTES: usize = 99;

/// Builds the acknowledgments.
pub fn acknowledge(request: &AcknowledgeRequest) -> Result<Acknowledgments, RequestError> {
    let mut acknowledgments = Vec::with_capacity(request.acknowledgments.len());
    for input in &request.acknowledgments {
        let body = body(request.kind, &input.group);
        let segment_count = body.len();
        let envelope = &input.envelope;
        let version = match (envelope.interchange_version.as_str(), request.kind) {
            ("", AckKind::Functional) => "00401",
            ("", AckKind::Implementation) => "00501",
            (version, _) => version,
        };
        let group_version = match request.kind {
            AckKind::Functional => "004010",
            AckKind::Implementation => "005010X231A1",
        };
        let isa_date = envelope.date.get(2..).unwrap_or(&envelope.date);
        let mut segments = vec![
            texts(&[
                "ISA",
                "00",
                "",
                "00",
                "",
                &envelope.sender_qualifier,
                &envelope.sender_id,
                &envelope.receiver_qualifier,
                &envelope.receiver_id,
                isa_date,
                &envelope.time,
                if version < "00402" { "U" } else { "" },
                version,
                &envelope.interchange_control_number,
                "0",
                &envelope.usage,
                "",
            ]),
            texts(&[
                "GS",
                "FA",
                &envelope.application_sender,
                &envelope.application_receiver,
                &envelope.date,
                &envelope.time,
                &envelope.group_control_number,
                "X",
                group_version,
            ]),
        ];
        let mut st = texts(&[
            "ST",
            match request.kind {
                AckKind::Functional => "997",
                AckKind::Implementation => "999",
            },
            &envelope.transaction_control_number,
        ]);
        if request.kind == AckKind::Implementation {
            st.push(Element::from(group_version));
        }
        segments.push(st);
        segments.extend(body);
        segments.push(texts(&["SE", "", &envelope.transaction_control_number]));
        segments.push(texts(&["GE", "", &envelope.group_control_number]));
        segments.push(texts(&["IEA", "", &envelope.interchange_control_number]));
        let written = write(&WriteRequest {
            separators: request.separators.clone(),
            segments,
            counts: true,
        })?;
        acknowledgments.push(Acknowledgment {
            text: written.text,
            code: input.group.acknowledgment.code.clone(),
            segment_count: segment_count + 2,
            transactions: input.group.transactions.len(),
        });
    }
    Ok(Acknowledgments { acknowledgments })
}

fn texts(values: &[&str]) -> Vec<Element> {
    values.iter().map(|value| Element::from(*value)).collect()
}

fn code(kind: AckKind, finding: FindingKind) -> &'static str {
    match kind {
        AckKind::Functional => finding.code_997(),
        AckKind::Implementation => finding.code(),
    }
}

/// The segments between ST and SE.
fn body(kind: AckKind, group: &AckGroup) -> Vec<Vec<Element>> {
    let mut segments = Vec::new();
    let header = group.header.as_ref();
    let mut ak1 = texts(&[
        "AK1",
        header.map_or("", |header| header.functional_id.as_str()),
        header.map_or("", |header| header.control_number.as_str()),
    ]);
    if kind == AckKind::Implementation {
        ak1.push(Element::from(
            header.map_or("", |header| header.version.as_str()),
        ));
    }
    segments.push(ak1);

    for transaction in &group.transactions {
        let mut ak2 = texts(&[
            "AK2",
            &transaction.transaction_set,
            &transaction.control_number,
        ]);
        if kind == AckKind::Implementation && !transaction.convention.is_empty() {
            ak2.push(Element::from(transaction.convention.as_str()));
        }
        segments.push(ak2);
        segments.extend(segment_notes(kind, &transaction.findings));

        let mut codes: Vec<&'static str> = transaction
            .findings
            .iter()
            .filter(|finding| finding.kind.level() == Level::Transaction && reported(finding))
            .map(|finding| code(kind, finding.kind))
            .filter(|code| !code.is_empty())
            .collect();
        let segment_errors = transaction.findings.iter().any(|finding| {
            matches!(finding.kind.level(), Level::Segment | Level::Element)
                && finding.severity == Severity::Error
        });
        if segment_errors {
            codes.push("5");
        }
        codes.dedup();
        let mut trailer = vec![
            Element::from(if kind == AckKind::Implementation {
                "IK5"
            } else {
                "AK5"
            }),
            Element::from(transaction.status.code()),
        ];
        trailer.extend(codes.into_iter().take(5).map(Element::from));
        segments.push(trailer);
    }

    let result = &group.acknowledgment;
    let mut ak9 = vec![
        Element::from("AK9"),
        Element::from(result.code.as_str()),
        Element::from(result.included.to_string()),
        Element::from(result.received.to_string()),
        Element::from(result.accepted.to_string()),
    ];
    ak9.extend(
        result
            .errors
            .iter()
            .take(5)
            .map(|code| Element::from(code.as_str())),
    );
    segments.push(ak9);
    segments
}

fn reported(finding: &AckFinding) -> bool {
    finding.severity != Severity::Info
}

/// The AK3/AK4 (IK3/IK4) loops for one transaction set's findings, in segment order.
fn segment_notes(kind: AckKind, findings: &[AckFinding]) -> Vec<Vec<Element>> {
    struct Note<'f> {
        segment_id: &'f str,
        position: u32,
        loop_id: &'f str,
        code: &'static str,
        elements: Vec<&'f AckFinding>,
    }
    let mut notes: BTreeMap<(u32, usize), Note<'_>> = BTreeMap::new();
    let mut order = 0usize;
    for finding in findings.iter().filter(|finding| reported(finding)) {
        let Some(position) = finding.position else {
            continue;
        };
        match finding.kind.level() {
            Level::Segment => {
                let missing = finding.kind == FindingKind::MandatorySegmentMissing;
                let key = if missing {
                    order += 1;
                    (position, order)
                } else {
                    (position, 0)
                };
                let note = notes.entry(key).or_insert(Note {
                    segment_id: &finding.segment_id,
                    position,
                    loop_id: &finding.loop_id,
                    code: "",
                    elements: Vec::new(),
                });
                if note.code.is_empty() || note.code == "8" {
                    note.code = code(kind, finding.kind);
                }
            }
            Level::Element => {
                let note = notes.entry((position, 0)).or_insert(Note {
                    segment_id: &finding.segment_id,
                    position,
                    loop_id: &finding.loop_id,
                    code: "8",
                    elements: Vec::new(),
                });
                note.elements.push(finding);
            }
            _ => {}
        }
    }

    let (segment_tag, element_tag) = match kind {
        AckKind::Functional => ("AK3", "AK4"),
        AckKind::Implementation => ("IK3", "IK4"),
    };
    let mut segments = Vec::new();
    for note in notes.values() {
        segments.push(texts(&[
            segment_tag,
            note.segment_id,
            &note.position.to_string(),
            note.loop_id,
            note.code,
        ]));
        for finding in note.elements.iter().take(MAX_ELEMENT_NOTES) {
            let mut position = vec![finding.element.unwrap_or(0).to_string()];
            if let Some(component) = finding.component {
                position.push(component.to_string());
            }
            if kind == AckKind::Implementation
                && let Some(repetition) = finding.repetition
            {
                if finding.component.is_none() {
                    position.push(String::new());
                }
                position.push(repetition.to_string());
            }
            let element_code = code(kind, finding.kind);
            let mut ak4 = vec![
                Element::from(element_tag),
                if position.len() == 1 {
                    Element::Text(position.remove(0))
                } else {
                    Element::Components(position)
                },
                Element::from(finding.element_ref.as_str()),
                Element::from(element_code),
            ];
            if !finding.value.is_empty() {
                ak4.push(Element::from(finding.value.as_str()));
            }
            segments.push(ak4);
        }
    }
    segments
}

#[cfg(test)]
mod tests {
    use super::{AckEnvelope, AckInput, AckKind, AcknowledgeRequest, acknowledge};
    use crate::x12::read::{ReadRequest, read};

    const TENDER: &str = concat!(
        "ISA*00*          *00*          *ZZ*SIMPARTNER     *ZZ*TRENOVA        *260107*1200*^*00401*000000101*0*T*>~",
        "GS*SM*SIMPARTNER*TRENOVA*20260107*1200*101*X*004010~",
        "ST*204*0001~B2**SIML**SHIP-1**PP~B2A*00~S5*1*CL~N1**Shipper~N4*Chicago*IL*60601~SE*7*0001~",
        "ST*204*0002~B2**SIML**SHIP-2**PP~B2A*00~S5*1*CL~SE*5*0002~",
        "GE*2*101~IEA*1*000000101~",
    );

    fn acknowledgment(kind: AckKind, text: &str) -> String {
        let reading = read(&ReadRequest {
            text: text.to_owned(),
            separators: None,
            guides: Vec::new(),
            include_segments: false,
            fragment: false,
        })
        .expect("a reading");
        let group = serde_json::to_value(&reading.interchanges[0].groups[0]).expect("json");
        let request = AcknowledgeRequest {
            kind,
            separators: None,
            acknowledgments: vec![AckInput {
                group: serde_json::from_value(group).expect("a group"),
                envelope: AckEnvelope {
                    sender_qualifier: String::from("ZZ"),
                    sender_id: String::from("TRENOVA"),
                    receiver_qualifier: String::from("ZZ"),
                    receiver_id: String::from("SIMPARTNER"),
                    application_sender: String::from("TRENOVA"),
                    application_receiver: String::from("SIMPARTNER"),
                    date: String::from("20260107"),
                    time: String::from("1201"),
                    interchange_control_number: String::from("000000555"),
                    group_control_number: String::from("555"),
                    transaction_control_number: String::from("0001"),
                    usage: String::from("T"),
                    interchange_version: String::new(),
                },
            }],
        };
        let mut written = acknowledge(&request).expect("acknowledgments");
        written.acknowledgments.remove(0).text
    }

    #[test]
    fn a_997_nests_each_error_under_its_transaction_set() {
        let text = acknowledgment(AckKind::Functional, TENDER);
        let body: Vec<&str> = text.split('~').skip(3).collect();
        assert_eq!(
            body[..8],
            [
                "AK1*SM*101",
                "AK2*204*0001",
                "AK3*N1*5*0310*8",
                "AK4*1*98*1",
                "AK5*R*5",
                "AK2*204*0002",
                "AK5*A",
                "AK9*P*2*2*1",
            ]
        );
        assert!(text.contains("SE*10*0001~"), "{text}");
        let reading = read(&ReadRequest {
            text,
            separators: None,
            guides: Vec::new(),
            include_segments: false,
            fragment: false,
        })
        .expect("a reading of the 997");
        assert_eq!(
            reading.summary.errors, 0,
            "{:?}",
            reading.interchanges[0].groups[0].transactions[0].findings
        );
    }

    #[test]
    fn a_999_uses_its_own_segments_and_version() {
        let text = acknowledgment(AckKind::Implementation, TENDER);
        assert!(text.contains("*00501*000000555*0*T*>~"), "{text}");
        assert!(
            text.contains("GS*FA*TRENOVA*SIMPARTNER*20260107*1201*555*X*005010X231A1~"),
            "{text}"
        );
        assert!(text.contains("ST*999*0001*005010X231A1~AK1*SM*101*004010~AK2*204*0001~IK3*N1*5*0310*8~IK4*1*98*1~IK5*R*5~"), "{text}");
        let reading = read(&ReadRequest {
            text,
            separators: None,
            guides: Vec::new(),
            include_segments: false,
            fragment: false,
        })
        .expect("a reading of the 999");
        assert_eq!(
            reading.summary.errors, 0,
            "{:?}",
            reading.interchanges[0].groups[0].transactions[0].findings
        );
    }

    #[test]
    fn a_missing_segment_gets_an_ak3_of_its_own() {
        let text = TENDER.replace("B2A*00~S5*1*CL~SE*5*0002", "S5*1*CL~SE*4*0002");
        let ack = acknowledgment(AckKind::Functional, &text);
        assert!(ack.contains("AK2*204*0002~AK3*B2A*3**3~AK5*R*5~"), "{ack}");
    }
}
