//! What reading and validating a file can find, in the standard's own terms.
//!
//! Every finding has a stable kind, the level it applies to and the X12 code an acknowledgment
//! carries for it: a TA1 note code for the interchange, an AK905 code for a functional group, an
//! AK502 or IK502 code for a transaction set, an AK304 or IK304 code for a segment, and an AK403
//! or IK403 code for an element. A 997 can only carry the standard's codes, so a finding that
//! exists only in a 999 (an `I` code, raised by a partner's narrowing) has a standard code it
//! falls back to in a 997.

use serde::{Deserialize, Serialize};
use trenova_module_kit::message::Message;

/// How much a finding matters to the file it is in.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum Severity {
    /// It rejects what it is in: the transaction set, group or interchange.
    Error,
    /// It is reported but accepts: an acknowledgment says "accepted with errors" for it.
    Warning,
    /// It is only noted.
    Info,
}

/// Which part of the file a finding is about.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum Level {
    /// The text as a whole, outside any interchange.
    File,
    /// One interchange (ISA to IEA).
    Interchange,
    /// One functional group (GS to GE).
    Group,
    /// One transaction set (ST to SE).
    Transaction,
    /// One segment.
    Segment,
    /// One element of a segment.
    Element,
}

macro_rules! finding_kinds {
    ($($(#[$doc:meta])* $kind:ident => $level:ident, $code:literal, $code997:literal, $severity:ident;)*) => {
        /// Every kind of finding, by stable name.
        #[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Serialize, Deserialize)]
        #[serde(rename_all = "camelCase")]
        pub enum FindingKind {
            $($(#[$doc])* $kind,)*
        }

        impl FindingKind {
            /// Every kind, in declaration order.
            pub const ALL: &[Self] = &[$(Self::$kind,)*];

            /// The level the finding applies to.
            #[must_use]
            pub fn level(self) -> Level {
                match self {
                    $(Self::$kind => Level::$level,)*
                }
            }

            /// The X12 code a 999 (or a TA1, or an AK9) carries for it; empty when there is
            /// none.
            #[must_use]
            pub fn code(self) -> &'static str {
                match self {
                    $(Self::$kind => $code,)*
                }
            }

            /// The X12 code a 997 carries for it: the standard code an implementation code falls
            /// back to; empty when there is none.
            #[must_use]
            pub fn code_997(self) -> &'static str {
                match self {
                    $(Self::$kind => $code997,)*
                }
            }

            /// Its severity when the guide does not override it.
            #[must_use]
            pub fn default_severity(self) -> Severity {
                match self {
                    $(Self::$kind => Severity::$severity,)*
                }
            }
        }
    };
}

finding_kinds! {
    /// The text has no ISA where an interchange was expected.
    InterchangeHeaderMissing => File, "", "", Error;
    /// Segments outside any interchange, or a GS or ST outside its envelope (TA1 022).
    InvalidControlStructure => Interchange, "022", "022", Error;
    /// The ISA's separators collide or are letters, digits or spaces (TA1 026).
    InvalidElementSeparator => Interchange, "026", "026", Error;
    /// The component separator collides with another (TA1 027).
    InvalidComponentSeparator => Interchange, "027", "027", Error;
    /// The segment terminator collides with another separator (TA1 004).
    InvalidSegmentTerminator => Interchange, "004", "004", Error;
    /// The ISA's padding was lost, as in a sample copied out of a PDF; it was read by counting
    /// separators.
    InterchangeHeaderSqueezed => Interchange, "", "", Warning;
    /// An ISA element is not what its position allows; the code is the TA1 note code for that
    /// element.
    InvalidInterchangeHeader => Interchange, "", "", Error;
    /// The IEA is missing: the interchange ended early (TA1 023).
    InterchangeTrailerMissing => Interchange, "023", "023", Error;
    /// ISA13 and IEA02 differ (TA1 001).
    InterchangeControlNumberMismatch => Interchange, "001", "001", Error;
    /// IEA01 does not count the groups (TA1 021).
    GroupCountMismatch => Interchange, "021", "021", Error;
    /// Two interchanges in one file share a control number (TA1 025).
    DuplicateInterchangeControlNumber => Interchange, "025", "025", Error;

    /// The GE is missing (AK905 3).
    GroupTrailerMissing => Group, "3", "3", Error;
    /// GS06 and GE02 differ (AK905 4).
    GroupControlNumberMismatch => Group, "4", "4", Error;
    /// GE01 does not count the transaction sets (AK905 5).
    TransactionCountMismatch => Group, "5", "5", Error;
    /// GS06 is not a number of one to nine digits (AK905 6).
    InvalidGroupControlNumber => Group, "6", "6", Error;
    /// Two groups in one interchange share a control number (AK905 19).
    DuplicateGroupControlNumber => Group, "19", "19", Error;
    /// A GS element is not what its position allows.
    InvalidGroupHeader => Group, "", "", Warning;
    /// GS01 is not the functional group the transaction set belongs to.
    FunctionalGroupMismatch => Group, "", "", Warning;

    /// No guide covers the transaction set (AK502 1).
    TransactionSetNotSupported => Transaction, "1", "1", Error;
    /// The SE is missing (AK502 2).
    TransactionTrailerMissing => Transaction, "2", "2", Error;
    /// ST02 and SE02 differ (AK502 3).
    TransactionControlNumberMismatch => Transaction, "3", "3", Error;
    /// SE01 does not count the segments (AK502 4).
    SegmentCountMismatch => Transaction, "4", "4", Error;
    /// ST01 is missing or not a transaction set identifier (AK502 6).
    InvalidTransactionSetIdentifier => Transaction, "6", "6", Error;
    /// ST02 is missing or not four to nine characters (AK502 7).
    InvalidTransactionControlNumber => Transaction, "7", "7", Error;
    /// An ST outside any GS (AK502 18).
    TransactionNotInGroup => Transaction, "18", "18", Error;
    /// Two transaction sets in one group share a control number (AK502 23).
    DuplicateTransactionControlNumber => Transaction, "23", "23", Error;
    /// ST03 names a different implementation convention than the guide's (IK502 19).
    ConventionReferenceMismatch => Transaction, "19", "", Warning;
    /// The guide is for another version than the group's GS08.
    GuideVersionDiffers => Transaction, "", "", Info;

    /// The segment ID is not one X12 defines (AK304 1).
    UnrecognizedSegment => Segment, "1", "1", Error;
    /// The segment cannot appear where it is (AK304 2).
    UnexpectedSegment => Segment, "2", "2", Error;
    /// A mandatory segment is missing (AK304 3).
    MandatorySegmentMissing => Segment, "3", "3", Error;
    /// A loop repeats more often than allowed (AK304 4).
    LoopOverMaximum => Segment, "4", "4", Error;
    /// A segment appears more often than allowed (AK304 5).
    SegmentOverMaximumUse => Segment, "5", "5", Error;
    /// The segment is not part of this transaction set (AK304 6).
    SegmentNotInSet => Segment, "6", "6", Error;
    /// The segment belongs to this transaction set, but not here (AK304 7).
    SegmentOutOfSequence => Segment, "7", "7", Error;
    /// The partner does not use the segment, but it is present (IK304 I4).
    NotUsedSegmentPresent => Segment, "I4", "2", Error;
    /// A loop repeats fewer times than the partner requires (IK304 I7).
    LoopUnderMinimum => Segment, "I7", "3", Error;
    /// A segment appears fewer times than the partner requires (IK304 I8).
    SegmentUnderMinimumUse => Segment, "I8", "3", Error;

    /// A mandatory element is missing (AK403 1).
    MandatoryElementMissing => Element, "1", "1", Error;
    /// A syntax note requires an element that is missing (AK403 2).
    ConditionalElementMissing => Element, "2", "2", Error;
    /// The segment has more elements than defined (AK403 3).
    TooManyElements => Element, "3", "3", Error;
    /// The element is shorter than its minimum (AK403 4).
    ElementTooShort => Element, "4", "4", Error;
    /// The element is longer than its maximum (AK403 5).
    ElementTooLong => Element, "5", "5", Error;
    /// The element holds a character its type or the character set does not allow (AK403 6).
    InvalidCharacter => Element, "6", "6", Error;
    /// A number with implied decimal places (`N2`) is written with a decimal point, which the
    /// standard does not allow but many partners send (AK403 6).
    ExplicitDecimalInImpliedNumber => Element, "6", "6", Warning;
    /// The element holds a code that is not in its list (AK403 7).
    InvalidCode => Element, "7", "7", Error;
    /// The element is not a valid date (AK403 8).
    InvalidDate => Element, "8", "8", Error;
    /// The element is not a valid time (AK403 9).
    InvalidTime => Element, "9", "9", Error;
    /// A syntax note allows at most one of these elements (AK403 10).
    ExclusionViolated => Element, "10", "10", Error;
    /// The element repeats more often than allowed (AK403 12).
    TooManyRepetitions => Element, "12", "12", Error;
    /// The composite has more components than defined, or a simple element has components
    /// (AK403 13).
    TooManyComponents => Element, "13", "13", Error;
    /// The partner does not use the element, but it is present (IK403 I10).
    NotUsedElementPresent => Element, "I10", "10", Error;
    /// The element does not hold the value the partner fixes (IK403 I12).
    FixedValueMismatch => Element, "I12", "7", Error;
    /// A value to be written held a separator, which was written as a space.
    ValueContainsSeparator => Element, "", "", Warning;
    /// A transaction set has so many findings that the rest were not reported.
    FindingsTruncated => Transaction, "", "", Info;
}

/// One finding.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Finding {
    /// What was found.
    pub kind: FindingKind,
    /// How much it matters, after the guide's overrides.
    pub severity: Severity,
    /// The level it applies to.
    pub level: Level,
    /// The X12 code a 999 or TA1 carries for it; empty when there is none.
    #[serde(skip_serializing_if = "str::is_empty")]
    pub code: &'static str,
    /// The index of the segment it is about in the reading's segment list.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub segment: Option<usize>,
    /// The ID of the segment it is about.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub segment_id: String,
    /// The segment's position in its transaction set, ST being 1.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub position: Option<u32>,
    /// The innermost loop the segment is in.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub loop_id: String,
    /// The element's position in the segment, one-based.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub element: Option<u32>,
    /// The component's position in the composite, one-based.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub component: Option<u32>,
    /// The repetition's position, one-based, for a repeating element.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub repetition: Option<u32>,
    /// The data element reference number.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub element_ref: String,
    /// A copy of the bad data, at most 99 characters, as an AK404 carries it.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub value: String,
    /// The byte span the finding points at.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub span: Option<(usize, usize)>,
    /// What it means, for people.
    pub message: Message,
}

/// The longest copy of bad data an acknowledgment carries (AK404 is 1 to 99 characters).
pub const MAX_VALUE_COPY: usize = 99;

impl Finding {
    /// A finding of `kind` with only its message; the location is added with the builders.
    #[must_use]
    pub fn new(kind: FindingKind, message: Message) -> Self {
        Self {
            kind,
            severity: kind.default_severity(),
            level: kind.level(),
            code: kind.code(),
            segment: None,
            segment_id: String::new(),
            position: None,
            loop_id: String::new(),
            element: None,
            component: None,
            repetition: None,
            element_ref: String::new(),
            value: String::new(),
            span: None,
            message,
        }
    }

    /// Sets the TA1 or other code when the kind's own depends on where it was found.
    #[must_use]
    pub fn with_code(mut self, code: &'static str) -> Self {
        self.code = code;
        self
    }

    /// Points the finding at a segment.
    #[must_use]
    pub fn at_segment(mut self, index: usize, id: &str, span: (usize, usize)) -> Self {
        self.segment = Some(index);
        id.clone_into(&mut self.segment_id);
        self.span = Some(span);
        self
    }

    /// Records the segment's position in its transaction set and its loop.
    #[must_use]
    pub fn in_transaction(mut self, position: u32, loop_id: &str) -> Self {
        self.position = Some(position);
        loop_id.clone_into(&mut self.loop_id);
        self
    }

    /// Points the finding at an element.
    #[must_use]
    pub fn at_element(mut self, position: u32, reference: &str) -> Self {
        self.element = Some(position);
        reference.clone_into(&mut self.element_ref);
        self
    }

    /// Points the finding at a component and repetition within the element.
    #[must_use]
    pub fn at_component(mut self, component: Option<u32>, repetition: Option<u32>) -> Self {
        self.component = component;
        self.repetition = repetition;
        self
    }

    /// Records a copy of the bad data, cut to what an acknowledgment carries.
    #[must_use]
    pub fn with_value(mut self, value: &str) -> Self {
        self.value = value.chars().take(MAX_VALUE_COPY).collect();
        self
    }

    /// Narrows the span to part of the segment.
    #[must_use]
    pub fn with_span(mut self, span: (usize, usize)) -> Self {
        self.span = Some(span);
        self
    }

    /// Whether it rejects what it is in.
    #[must_use]
    pub fn is_error(&self) -> bool {
        self.severity == Severity::Error
    }
}

#[cfg(test)]
mod tests {
    use super::{FindingKind, Level, Severity};

    #[test]
    fn every_segment_and_element_finding_has_a_997_code() {
        for kind in FindingKind::ALL {
            let written_only = *kind == FindingKind::ValueContainsSeparator;
            if matches!(kind.level(), Level::Segment | Level::Element) && !written_only {
                assert!(!kind.code().is_empty(), "{kind:?}");
                assert!(!kind.code_997().is_empty(), "{kind:?}");
                assert!(
                    kind.code_997().bytes().all(|byte| byte.is_ascii_digit()),
                    "{kind:?} falls back to a standard code in a 997"
                );
            }
        }
    }

    #[test]
    fn kinds_have_stable_wire_names() {
        assert_eq!(
            serde_json::to_string(&FindingKind::SegmentOutOfSequence).unwrap_or_default(),
            "\"segmentOutOfSequence\""
        );
        assert_eq!(FindingKind::InvalidCode.default_severity(), Severity::Error);
        assert_eq!(FindingKind::NotUsedSegmentPresent.code(), "I4");
    }
}
