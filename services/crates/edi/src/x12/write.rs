//! Writes segments as X12 text.
//!
//! The writer is the one place that knows how text is laid out: separators between elements
//! and components, the terminator and line ending after each segment, trailing empty elements
//! left off, the ISA's elements padded to their fixed widths, and the counts in SE01, GE01 and
//! IEA01 taken from what was written. X12 has no escape, so a value holding a separator has the
//! separator replaced with a space, and the writer says so.

use serde::{Deserialize, Serialize};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::ops::RequestError;

use super::finding::{Finding, FindingKind};
use super::separators::{Separators, WireSeparators};

/// What `x12.write` takes.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WriteRequest {
    /// The separators to write with; empty ones take the conventional `*`, `>`, `^` and `~`.
    #[serde(default)]
    pub separators: Option<WireSeparators>,
    /// The segments: each is its ID followed by its elements.
    pub segments: Vec<Vec<Element>>,
    /// Whether to fill SE01, GE01 and IEA01 with the counts of what was written.
    #[serde(default)]
    pub counts: bool,
}

/// One element to write: text, or the components of a composite.
#[derive(Debug, Clone, PartialEq, Eq, Deserialize)]
#[serde(untagged)]
pub enum Element {
    /// A simple element.
    Text(String),
    /// A composite, its components in order.
    Components(Vec<String>),
}

impl From<&str> for Element {
    fn from(value: &str) -> Self {
        Self::Text(value.to_owned())
    }
}

impl From<String> for Element {
    fn from(value: String) -> Self {
        Self::Text(value)
    }
}

/// What the writer produced.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Written {
    /// The text.
    pub text: String,
    /// How many segments it holds.
    pub segment_count: usize,
    /// What the writer had to change.
    pub findings: Vec<Finding>,
}

/// The ISA's element widths, ISA01 first.
pub const ISA_WIDTHS: [usize; 16] = [2, 10, 2, 10, 2, 15, 2, 15, 6, 4, 1, 5, 9, 1, 1, 1];

/// Writes `request.segments`.
pub fn write(request: &WriteRequest) -> Result<Written, RequestError> {
    let separators = match &request.separators {
        Some(wire) => Separators::from_wire(wire).ok_or_else(|| {
            RequestError::invalid_input(
                "separators must be single ASCII characters that are not letters, digits or spaces, and must differ",
            )
        })?,
        None => Separators::default(),
    };
    let mut segments: Vec<Vec<Element>> = request.segments.clone();
    for (index, segment) in segments.iter().enumerate() {
        let Some(Element::Text(id)) = segment.first() else {
            return Err(RequestError::invalid_input(format!(
                "segment {index} does not start with its ID"
            )));
        };
        let bytes = id.as_bytes();
        let valid = (2..=3).contains(&bytes.len())
            && bytes[0].is_ascii_uppercase()
            && bytes
                .iter()
                .all(|byte| byte.is_ascii_uppercase() || byte.is_ascii_digit());
        if !valid {
            return Err(RequestError::invalid_input(format!(
                "segment {index} has the ID {id:?}, which is not two or three capital letters and digits"
            )));
        }
    }
    if request.counts {
        fill_counts(&mut segments);
    }

    let mut findings = Vec::new();
    let mut text = String::new();
    for (index, segment) in segments.iter().enumerate() {
        let id = match segment.first() {
            Some(Element::Text(id)) => id.as_str(),
            _ => "",
        };
        let mut written: Vec<String> = Vec::with_capacity(segment.len());
        written.push(id.to_owned());
        for (offset, element) in segment[1..].iter().enumerate() {
            let position = offset + 1;
            let isa_separator = id == "ISA" && matches!(position, 11 | 16);
            let value = match element {
                Element::Text(value) if isa_separator => value.clone(),
                Element::Text(value) => {
                    clean(value, separators, index, id, position, &mut findings)
                }
                Element::Components(parts) => parts
                    .iter()
                    .map(|part| clean(part, separators, index, id, position, &mut findings))
                    .collect::<Vec<_>>()
                    .join(&char::from(separators.component).to_string()),
            };
            let value = if id == "ISA" {
                pad_isa(position, &value, separators, index, &mut findings)
            } else {
                value
            };
            written.push(value);
        }
        if id != "ISA" {
            let component = char::from(separators.component);
            while written.len() > 1
                && written
                    .last()
                    .is_some_and(|value| value.chars().all(|character| character == component))
            {
                written.pop();
            }
            if let Some(last) = written.last_mut() {
                let trimmed = last.trim_end_matches(component).len();
                last.truncate(trimmed);
            }
        }
        text.push_str(&written.join(&char::from(separators.element).to_string()));
        text.push(char::from(separators.segment));
        text.push_str(separators.line_ending.as_str());
    }
    Ok(Written {
        text,
        segment_count: segments.len(),
        findings,
    })
}

fn clean(
    value: &str,
    separators: Separators,
    index: usize,
    id: &str,
    position: usize,
    findings: &mut Vec<Finding>,
) -> String {
    let mut reserved = vec![separators.element, separators.segment, separators.component];
    reserved.extend(separators.repetition);
    let had = value
        .bytes()
        .any(|byte| reserved.contains(&byte) || byte == b'\r' || byte == b'\n');
    if !had {
        return value.to_owned();
    }
    findings.push(written_finding(
        FindingKind::ValueContainsSeparator,
        Message::with(
            message!("{0}{1} held a separator, which was written as a space"),
            [Arg::from(id), Arg::from(format!("{position:02}"))],
        ),
        index,
        id,
        position,
    ));
    value
        .chars()
        .map(|character| {
            let reserved_char = u8::try_from(character)
                .is_ok_and(|byte| reserved.contains(&byte) || byte == b'\r' || byte == b'\n');
            if reserved_char { ' ' } else { character }
        })
        .collect()
}

fn pad_isa(
    position: usize,
    value: &str,
    separators: Separators,
    index: usize,
    findings: &mut Vec<Finding>,
) -> String {
    let Some(width) = position
        .checked_sub(1)
        .and_then(|offset| ISA_WIDTHS.get(offset))
    else {
        return value.to_owned();
    };
    match position {
        11 => {
            if let Some(repetition) = separators.repetition
                && value.is_empty()
            {
                return char::from(repetition).to_string();
            }
            return value.chars().take(1).collect();
        }
        16 => return char::from(separators.component).to_string(),
        _ => {}
    }
    let length = value.chars().count();
    if length > *width {
        findings.push(written_finding(
            FindingKind::ElementTooLong,
            Message::with(
                message!(
                    "ISA{0} is {1} characters, longer than its fixed width of {2}; it was cut"
                ),
                [
                    Arg::from(format!("{position:02}")),
                    Arg::from(length),
                    Arg::from(*width),
                ],
            ),
            index,
            "ISA",
            position,
        ));
        return value.chars().take(*width).collect();
    }
    if position == 13 {
        format!("{value:0>width$}", width = *width)
    } else {
        format!("{value:<width$}", width = *width)
    }
}

/// A finding about segment `index` of the written list; it has no span, since the text it
/// would point into is the output.
fn written_finding(
    kind: FindingKind,
    message: Message,
    index: usize,
    id: &str,
    position: usize,
) -> Finding {
    let mut finding = Finding::new(kind, message)
        .at_segment(index, id, (0, 0))
        .at_element(u32::try_from(position).unwrap_or(u32::MAX), "");
    finding.span = None;
    finding
}

fn fill_counts(segments: &mut [Vec<Element>]) {
    let id = |segment: &[Element]| match segment.first() {
        Some(Element::Text(id)) => id.clone(),
        _ => String::new(),
    };
    let mut transaction_start = None;
    let mut transactions = 0usize;
    let mut groups = 0usize;
    for (index, segment) in segments.iter_mut().enumerate() {
        match id(segment).as_str() {
            "ISA" => groups = 0,
            "GS" => {
                groups += 1;
                transactions = 0;
            }
            "ST" => {
                transactions += 1;
                transaction_start = Some(index);
            }
            "SE" => {
                if let Some(start) = transaction_start.take() {
                    set_element(segment, 1, (index - start + 1).to_string());
                }
            }
            "GE" => set_element(segment, 1, transactions.to_string()),
            "IEA" => set_element(segment, 1, groups.to_string()),
            _ => {}
        }
    }
}

fn set_element(segment: &mut Vec<Element>, position: usize, value: String) {
    while segment.len() <= position {
        segment.push(Element::Text(String::new()));
    }
    segment[position] = Element::Text(value);
}

#[cfg(test)]
mod tests {
    use super::{Element, WriteRequest, write};
    use crate::x12::finding::FindingKind;
    use crate::x12::read::{ReadRequest, read};

    fn segment(parts: &[&str]) -> Vec<Element> {
        parts.iter().map(|part| Element::from(*part)).collect()
    }

    #[test]
    fn writes_a_padded_isa_counted_trailers_and_trims_empty_elements() {
        let request = WriteRequest {
            separators: None,
            segments: vec![
                segment(&[
                    "ISA", "00", "", "00", "", "ZZ", "TRENOVA", "ZZ", "PARTNER", "260107", "1200",
                    "", "00401", "7", "0", "T", "",
                ]),
                segment(&[
                    "GS", "FA", "TRENOVA", "PARTNER", "20260107", "1200", "7", "X", "004010",
                ]),
                segment(&["ST", "997", "0001"]),
                segment(&["AK1", "SM", "101"]),
                segment(&["AK2", "204", "0001"]),
                segment(&["AK5", "A", "", ""]),
                segment(&["AK9", "A", "1", "1", "1"]),
                segment(&["SE", "", "0001"]),
                segment(&["GE", "", "7"]),
                segment(&["IEA", "", "000000007"]),
            ],
            counts: true,
        };
        let written = write(&request).expect("text");
        assert!(written.findings.is_empty(), "{:?}", written.findings);
        let lines: Vec<&str> = written.text.split('~').collect();
        assert_eq!(lines[0].len(), 105);
        assert_eq!(
            lines[0],
            "ISA*00*          *00*          *ZZ*TRENOVA        *ZZ*PARTNER        *260107*1200*^*00401*000000007*0*T*>"
        );
        assert_eq!(lines[5], "AK5*A");
        assert_eq!(lines[7], "SE*6*0001");
        assert_eq!(lines[8], "GE*1*7");
        assert_eq!(lines[9], "IEA*1*000000007");

        let reading = read(&ReadRequest {
            text: written.text,
            separators: None,
            guides: Vec::new(),
            include_segments: false,
            fragment: false,
        })
        .expect("a reading");
        assert_eq!(
            reading.summary.errors, 0,
            "{:?}",
            reading.interchanges[0].groups[0].transactions[0].findings
        );
    }

    #[test]
    fn replaces_separators_inside_values_and_writes_composites() {
        let request = WriteRequest {
            separators: None,
            segments: vec![
                segment(&["N1", "SH", "ACME*DC~1"]),
                vec![
                    Element::from("AK4"),
                    Element::Components(vec![String::from("2"), String::from("1")]),
                    Element::from("98"),
                ],
            ],
            counts: false,
        };
        let written = write(&request).expect("text");
        assert_eq!(written.text, "N1*SH*ACME DC 1~AK4*2>1*98~");
        assert_eq!(written.findings.len(), 1);
        assert_eq!(
            written.findings[0].kind,
            FindingKind::ValueContainsSeparator
        );
    }

    #[test]
    fn refuses_a_segment_without_a_proper_id() {
        let request = WriteRequest {
            separators: None,
            segments: vec![segment(&["n1", "SH"])],
            counts: false,
        };
        assert!(write(&request).is_err());
    }
}
