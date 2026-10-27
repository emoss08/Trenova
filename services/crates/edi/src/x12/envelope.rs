//! Finds a text's segments and the envelopes around them.
//!
//! The scanner reads segment after segment, and each time it meets an ISA it reads that
//! interchange's separators from it, since one file may carry interchanges written with
//! different ones. The assembler then nests the segments: ISA to IEA, GS to GE, ST to SE. It
//! never gives up on a file: a missing trailer closes what it would have closed, a GS outside an
//! ISA or an ST outside a GS gets an envelope of its own, and every such repair is a finding, so
//! the rest of the file is still read and reported on.

use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::finding::{Finding, FindingKind};
use super::lexer::{Lexer, Segment};
use super::separators::{self, SeparatorSource, Separators};

/// One segment as scanned, with the separators it was written with.
#[derive(Debug, Clone, Copy)]
pub struct Scanned<'a> {
    /// The segment.
    pub segment: Segment<'a>,
    /// The separators of the interchange it is in.
    pub separators: Separators,
    /// For an ISA, whether it had its fixed width.
    pub fixed_width: bool,
}

/// Every segment of the text, and the separators the first one was read with.
#[derive(Debug, Clone)]
pub struct Scan<'a> {
    /// The segments in order.
    pub segments: Vec<Scanned<'a>>,
    /// The separators of the first interchange, or the ones used for text without an ISA.
    pub separators: Separators,
    /// Where those came from.
    pub source: SeparatorSource,
}

/// The most segments one text may have; a text with more is read up to here and reported.
pub const MAX_SEGMENTS: usize = 2_000_000;

/// Scans `text`. Text without an ISA is read with `fallback`, or with separators detected from
/// the text when there is none.
#[must_use]
pub fn scan(text: &str, fallback: Option<Separators>) -> Scan<'_> {
    let bytes = text.as_bytes();
    let first = bytes
        .iter()
        .position(|byte| !byte.is_ascii_whitespace())
        .unwrap_or(bytes.len());
    let (mut current, source) = match separators::from_isa(bytes, first) {
        Some(isa) => (isa.separators, SeparatorSource::Isa),
        None => match fallback {
            Some(separators) => (separators, SeparatorSource::Request),
            None => (separators::detect(bytes), SeparatorSource::Detected),
        },
    };
    let initial = current;
    let mut lexer = Lexer::new(text, 0, current);
    let mut segments = Vec::new();
    loop {
        let position = lexer.position();
        let start = position
            + bytes[position..]
                .iter()
                .take_while(|byte| matches!(byte, b'\r' | b'\n' | b'\t' | b' ' | 0x0c))
                .count();
        if start >= bytes.len() || segments.len() >= MAX_SEGMENTS {
            break;
        }
        if let Some(isa) = separators::from_isa(bytes, start) {
            current = isa.separators;
            lexer.set_separators(current);
            let raw = &text[start..isa.terminator];
            segments.push(Scanned {
                segment: Segment {
                    id: "ISA",
                    raw,
                    start,
                    end: start + raw.len(),
                    terminated: true,
                },
                separators: current,
                fixed_width: isa.fixed_width,
            });
            lexer.seek(isa.end);
            continue;
        }
        let Some(segment) = lexer.next() else {
            break;
        };
        segments.push(Scanned {
            segment,
            separators: current,
            fixed_width: true,
        });
    }
    Scan {
        segments,
        separators: initial,
        source,
    }
}

/// A trailer's two values.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Trailer {
    /// The segment's index.
    pub segment: usize,
    /// Its count element (IEA01, GE01, SE01).
    pub count: String,
    /// Its control number element (IEA02, GE02, SE02).
    pub control_number: String,
}

/// One transaction set's place in the text.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TransactionSpan {
    /// The ST's index, or the first segment's when an ST was missing.
    pub first: usize,
    /// The last segment's index: the SE, or the last one read before the transaction ended.
    pub last: usize,
    /// The SE, when there was one.
    pub trailer: Option<Trailer>,
}

/// One functional group's place in the text.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GroupSpan {
    /// The GS's index, or `None` for the envelope made for transaction sets outside a GS.
    pub header: Option<usize>,
    /// The first segment's index.
    pub first: usize,
    /// The last segment's index.
    pub last: usize,
    /// The GE, when there was one.
    pub trailer: Option<Trailer>,
    /// Its transaction sets.
    pub transactions: Vec<TransactionSpan>,
}

/// One interchange's place in the text.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct InterchangeSpan {
    /// The ISA's index, or `None` for the envelope made for a text without an ISA.
    pub header: Option<usize>,
    /// The first segment's index.
    pub first: usize,
    /// The last segment's index.
    pub last: usize,
    /// The IEA, when there was one.
    pub trailer: Option<Trailer>,
    /// Its functional groups.
    pub groups: Vec<GroupSpan>,
}

/// The envelopes of a scan.
#[derive(Debug, Clone, Default)]
pub struct Envelopes {
    /// The interchanges in order.
    pub interchanges: Vec<InterchangeSpan>,
    /// What was wrong with the nesting: missing trailers, segments outside an envelope.
    pub findings: Vec<Finding>,
}

struct Assembler<'s, 'a> {
    segments: &'s [Scanned<'a>],
    fragment: bool,
    interchanges: Vec<InterchangeSpan>,
    findings: Vec<Finding>,
    interchange_open: bool,
    group_open: bool,
    transaction_open: bool,
}

/// Nests the segments of a scan in their envelopes. A `fragment` (a transaction set pasted on
/// its own) may leave out the ISA and GS without a finding.
#[must_use]
pub fn assemble(segments: &[Scanned<'_>], fragment: bool) -> Envelopes {
    let mut assembler = Assembler {
        segments,
        fragment,
        interchanges: Vec::new(),
        findings: Vec::new(),
        interchange_open: false,
        group_open: false,
        transaction_open: false,
    };
    for index in 0..segments.len() {
        assembler.take(index);
    }
    let last = segments.len().saturating_sub(1);
    assembler.end_interchange(last, None);
    Envelopes {
        interchanges: assembler.interchanges,
        findings: assembler.findings,
    }
}

impl Assembler<'_, '_> {
    fn segment(&self, index: usize) -> &Segment<'_> {
        &self.segments[index].segment
    }

    fn trailer(&self, index: usize) -> Trailer {
        let scanned = &self.segments[index];
        Trailer {
            segment: index,
            count: scanned
                .segment
                .element(&scanned.separators, 1)
                .trim()
                .to_owned(),
            control_number: scanned
                .segment
                .element(&scanned.separators, 2)
                .trim()
                .to_owned(),
        }
    }

    fn structure_finding(&mut self, index: usize, message: Message) {
        let segment = *self.segment(index);
        self.findings.push(
            Finding::new(FindingKind::InvalidControlStructure, message).at_segment(
                index,
                segment.id,
                (segment.start, segment.end),
            ),
        );
    }

    fn take(&mut self, index: usize) {
        let id = self.segment(index).id;
        match id {
            "ISA" => {
                self.end_interchange(index.saturating_sub(1), None);
                self.interchanges.push(InterchangeSpan {
                    header: Some(index),
                    first: index,
                    last: index,
                    trailer: None,
                    groups: Vec::new(),
                });
                self.interchange_open = true;
            }
            "IEA" => {
                if !self.interchange_open {
                    self.structure_finding(
                        index,
                        Message::new(message!("An IEA closes an interchange that never started")),
                    );
                    return;
                }
                let trailer = self.trailer(index);
                self.end_interchange(index, Some(trailer));
            }
            "GS" => {
                self.ensure_interchange(index);
                self.end_group(index.saturating_sub(1), None);
                if let Some(interchange) = self.interchanges.last_mut() {
                    interchange.groups.push(GroupSpan {
                        header: Some(index),
                        first: index,
                        last: index,
                        trailer: None,
                        transactions: Vec::new(),
                    });
                    interchange.last = index;
                }
                self.group_open = true;
            }
            "GE" => {
                if !self.group_open {
                    self.structure_finding(
                        index,
                        Message::new(message!(
                            "A GE closes a functional group that never started"
                        )),
                    );
                    return;
                }
                let trailer = self.trailer(index);
                self.end_group(index, Some(trailer));
            }
            "ST" => {
                self.ensure_group(index);
                self.end_transaction(index.saturating_sub(1), None);
                if let Some(group) = self
                    .interchanges
                    .last_mut()
                    .and_then(|interchange| interchange.groups.last_mut())
                {
                    group.transactions.push(TransactionSpan {
                        first: index,
                        last: index,
                        trailer: None,
                    });
                    group.last = index;
                }
                self.transaction_open = true;
            }
            "SE" => {
                if !self.transaction_open {
                    self.structure_finding(
                        index,
                        Message::new(message!(
                            "An SE closes a transaction set that never started"
                        )),
                    );
                    return;
                }
                let trailer = self.trailer(index);
                self.end_transaction(index, Some(trailer));
            }
            _ => {
                if !self.transaction_open {
                    self.structure_finding(
                        index,
                        Message::with(
                            message!("{0} is outside any transaction set"),
                            [Arg::from(self.segment(index).id)],
                        ),
                    );
                    return;
                }
                self.extend_transaction(index);
            }
        }
    }

    fn ensure_interchange(&mut self, index: usize) {
        if self.interchange_open {
            return;
        }
        if !self.fragment {
            let id = self.segment(index).id.to_owned();
            self.structure_finding(
                index,
                Message::with(
                    message!("{0} is outside any interchange: the ISA is missing"),
                    [Arg::from(id)],
                ),
            );
        }
        self.interchanges.push(InterchangeSpan {
            header: None,
            first: index,
            last: index,
            trailer: None,
            groups: Vec::new(),
        });
        self.interchange_open = true;
    }

    fn ensure_group(&mut self, index: usize) {
        self.ensure_interchange(index);
        if self.group_open {
            return;
        }
        if !self.fragment {
            let segment = *self.segment(index);
            self.findings.push(
                Finding::new(
                    FindingKind::TransactionNotInGroup,
                    Message::new(message!(
                        "This transaction set is outside any functional group: the GS is missing"
                    )),
                )
                .at_segment(index, segment.id, (segment.start, segment.end)),
            );
        }
        if let Some(interchange) = self.interchanges.last_mut() {
            interchange.groups.push(GroupSpan {
                header: None,
                first: index,
                last: index,
                trailer: None,
                transactions: Vec::new(),
            });
        }
        self.group_open = true;
    }

    fn extend_transaction(&mut self, index: usize) {
        if let Some(interchange) = self.interchanges.last_mut() {
            interchange.last = index;
            if let Some(group) = interchange.groups.last_mut() {
                group.last = index;
                if let Some(transaction) = group.transactions.last_mut() {
                    transaction.last = index;
                }
            }
        }
    }

    fn end_transaction(&mut self, last: usize, trailer: Option<Trailer>) {
        if !self.transaction_open {
            return;
        }
        self.transaction_open = false;
        let closed_by_trailer = trailer.is_some();
        if closed_by_trailer {
            self.extend_transaction(last);
        }
        let Some(transaction) = self
            .interchanges
            .last_mut()
            .and_then(|interchange| interchange.groups.last_mut())
            .and_then(|group| group.transactions.last_mut())
        else {
            return;
        };
        transaction.trailer = trailer;
        if !closed_by_trailer {
            let first = transaction.first;
            let segment = *self.segment(first);
            self.findings.push(
                Finding::new(
                    FindingKind::TransactionTrailerMissing,
                    Message::new(message!("This transaction set has no SE")),
                )
                .at_segment(first, segment.id, (segment.start, segment.end)),
            );
        }
    }

    fn end_group(&mut self, last: usize, trailer: Option<Trailer>) {
        self.end_transaction(last, None);
        if !self.group_open {
            return;
        }
        self.group_open = false;
        let closed_by_trailer = trailer.is_some();
        let Some(interchange) = self.interchanges.last_mut() else {
            return;
        };
        if closed_by_trailer {
            interchange.last = last;
        }
        let Some(group) = interchange.groups.last_mut() else {
            return;
        };
        if closed_by_trailer {
            group.last = last;
        }
        let header = group.header;
        group.trailer = trailer;
        if !closed_by_trailer && let Some(header) = header {
            let segment = *self.segment(header);
            self.findings.push(
                Finding::new(
                    FindingKind::GroupTrailerMissing,
                    Message::new(message!("This functional group has no GE")),
                )
                .at_segment(header, segment.id, (segment.start, segment.end)),
            );
        }
    }

    fn end_interchange(&mut self, last: usize, trailer: Option<Trailer>) {
        self.end_group(last, None);
        if !self.interchange_open {
            return;
        }
        self.interchange_open = false;
        let closed_by_trailer = trailer.is_some();
        let Some(interchange) = self.interchanges.last_mut() else {
            return;
        };
        if closed_by_trailer {
            interchange.last = last;
        }
        let header = interchange.header;
        interchange.trailer = trailer;
        if !closed_by_trailer && let Some(header) = header {
            let segment = *self.segment(header);
            self.findings.push(
                Finding::new(
                    FindingKind::InterchangeTrailerMissing,
                    Message::new(message!(
                        "This interchange has no IEA: the file ended early"
                    )),
                )
                .at_segment(header, segment.id, (segment.start, segment.end)),
            );
        }
    }
}

#[cfg(test)]
mod tests {
    use super::{assemble, scan};
    use crate::x12::finding::FindingKind;
    use crate::x12::separators::SeparatorSource;

    const FILE: &str = concat!(
        "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        *260107*1200*^*00401*000000104*0*P*>~\n",
        "GS*QM*PARTNER*TRENOVA*20260107*1200*104*X*004010~\n",
        "ST*214*0001~B10*PRO*SHIP*SCAC~SE*3*0001~\n",
        "ST*214*0002~B10*PRO*SHIP*SCAC~SE*3*0002~\n",
        "GE*2*104~IEA*1*000000104~\n",
    );

    #[test]
    fn nests_interchanges_groups_and_transactions() {
        let scanned = scan(FILE, None);
        assert_eq!(scanned.source, SeparatorSource::Isa);
        assert_eq!(scanned.segments.len(), 10);
        assert_eq!(scanned.segments[0].segment.id, "ISA");
        assert_eq!(scanned.segments[1].segment.start, 107);
        let envelopes = assemble(&scanned.segments, false);
        assert!(envelopes.findings.is_empty(), "{:?}", envelopes.findings);
        let interchange = &envelopes.interchanges[0];
        assert_eq!((interchange.first, interchange.last), (0, 9));
        assert_eq!(
            interchange
                .trailer
                .as_ref()
                .map(|trailer| trailer.count.as_str()),
            Some("1")
        );
        let group = &interchange.groups[0];
        assert_eq!((group.first, group.last), (1, 8));
        assert_eq!(group.transactions.len(), 2);
        assert_eq!(
            (group.transactions[1].first, group.transactions[1].last),
            (5, 7)
        );
    }

    #[test]
    fn closes_what_a_missing_trailer_would_have_closed() {
        let text = FILE
            .replace("SE*3*0001~", "")
            .replace("GE*2*104~IEA*1*000000104~\n", "");
        let scanned = scan(&text, None);
        let envelopes = assemble(&scanned.segments, false);
        let kinds: Vec<_> = envelopes
            .findings
            .iter()
            .map(|finding| finding.kind)
            .collect();
        assert_eq!(
            kinds,
            [
                FindingKind::TransactionTrailerMissing,
                FindingKind::GroupTrailerMissing,
                FindingKind::InterchangeTrailerMissing
            ]
        );
        assert_eq!(envelopes.interchanges[0].groups[0].transactions.len(), 2);
    }

    #[test]
    fn a_fragment_needs_no_envelope_and_a_file_does() {
        let text = "ST*990*0001~B1*SCAC*SHIP**A~SE*3*0001~";
        let scanned = scan(text, None);
        assert_eq!(scanned.source, SeparatorSource::Detected);
        assert!(assemble(&scanned.segments, true).findings.is_empty());
        let kinds: Vec<_> = assemble(&scanned.segments, false)
            .findings
            .iter()
            .map(|finding| finding.kind)
            .collect();
        assert_eq!(
            kinds,
            [
                FindingKind::InvalidControlStructure,
                FindingKind::TransactionNotInGroup
            ]
        );
    }

    #[test]
    fn reads_each_interchange_with_its_own_separators() {
        let second = "ISA|00|          |00|          |ZZ|A              |ZZ|B              |260107|1200|U|00401|000000105|0|P|:\nGS|QM|A|B|20260107|1200|105|X|004010\nGE|0|105\nIEA|1|000000105\n";
        let text = format!("{FILE}{second}");
        let scanned = scan(&text, None);
        assert_eq!(scanned.segments.len(), 14);
        assert_eq!(scanned.segments[11].segment.id, "GS");
        assert_eq!(scanned.segments[11].separators.element, b'|');
        let envelopes = assemble(&scanned.segments, false);
        assert_eq!(envelopes.interchanges.len(), 2);
        assert!(envelopes.findings.is_empty(), "{:?}", envelopes.findings);
    }

    #[test]
    fn a_segment_outside_a_transaction_is_reported() {
        let text = FILE.replace("GE*2*104~", "N1*SH*X~GE*2*104~");
        let scanned = scan(&text, None);
        let envelopes = assemble(&scanned.segments, false);
        assert_eq!(envelopes.findings.len(), 1);
        assert_eq!(
            envelopes.findings[0].kind,
            FindingKind::InvalidControlStructure
        );
        assert_eq!(envelopes.findings[0].segment_id, "N1");
    }
}
