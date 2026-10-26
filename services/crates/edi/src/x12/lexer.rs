//! Splits X12 text into segments, elements, repetitions and components without copying.
//!
//! A [`Segment`] borrows its slice of the input and knows its byte span, so the inspector can
//! highlight it and a finding can point at it. Elements are split from the segment on demand,
//! and repetitions and components from an element on demand; nothing is allocated for a value
//! until it goes on the wire. Whitespace between segments (the line breaks many partners add
//! after each terminator, or a PDF's wrapping) is skipped, not read as data.

use super::separators::Separators;

/// One segment of the text.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Segment<'a> {
    /// The segment ID: the text before the first element separator.
    pub id: &'a str,
    /// The segment as written, without its terminator.
    pub raw: &'a str,
    /// The byte offset of the segment's first byte.
    pub start: usize,
    /// The byte offset just past the segment's last byte, before the terminator.
    pub end: usize,
    /// Whether a terminator followed the segment; `false` only for text cut short.
    pub terminated: bool,
}

impl<'a> Segment<'a> {
    /// The segment's elements, ID excluded: element 1 first. An element written as empty is
    /// an empty string.
    pub fn elements(&self, separators: &Separators) -> impl Iterator<Item = &'a str> + use<'a> {
        let raw = self.raw;
        let separator = char::from(separators.element);
        let mut parts = raw.split(separator);
        parts.next();
        parts
    }

    /// The element at a one-based position, or an empty string past the last one.
    #[must_use]
    pub fn element(&self, separators: &Separators, position: usize) -> &'a str {
        if position == 0 {
            return "";
        }
        self.elements(separators).nth(position - 1).unwrap_or("")
    }

    /// How many elements the segment has as written, trailing empty ones included.
    #[must_use]
    pub fn element_count(&self, separators: &Separators) -> usize {
        self.elements(separators).count()
    }

    /// The byte span of the element at a one-based position, or the segment's end past the last
    /// one.
    #[must_use]
    pub fn element_span(&self, separators: &Separators, position: usize) -> (usize, usize) {
        let mut offset = self.start + self.id.len();
        for (index, element) in self.elements(separators).enumerate() {
            offset += 1;
            if index + 1 == position {
                return (offset, offset + element.len());
            }
            offset += element.len();
        }
        (self.end, self.end)
    }

    /// Whether the ID is two or three uppercase letters and digits starting with a letter, the
    /// only shape X12 segment IDs take.
    #[must_use]
    pub fn has_valid_id(&self) -> bool {
        let bytes = self.id.as_bytes();
        (2..=3).contains(&bytes.len())
            && bytes[0].is_ascii_uppercase()
            && bytes
                .iter()
                .all(|byte| byte.is_ascii_uppercase() || byte.is_ascii_digit())
    }
}

/// Splits an element into its repetitions; an element without the repetition separator is one
/// repetition.
pub fn repetitions<'a>(element: &'a str, separators: &Separators) -> Vec<&'a str> {
    match separators.repetition {
        Some(byte) => element.split(char::from(byte)).collect(),
        None => vec![element],
    }
}

/// Splits one repetition of an element into its components; a simple element is one
/// component.
pub fn components<'a>(value: &'a str, separators: &Separators) -> Vec<&'a str> {
    value.split(char::from(separators.component)).collect()
}

/// Iterates over the segments of `text` from a byte offset.
#[derive(Debug, Clone)]
pub struct Lexer<'a> {
    text: &'a str,
    position: usize,
    separators: Separators,
}

impl<'a> Lexer<'a> {
    /// A lexer over `text` starting at byte `start`.
    #[must_use]
    pub fn new(text: &'a str, start: usize, separators: Separators) -> Self {
        Self {
            text,
            position: start.min(text.len()),
            separators,
        }
    }

    /// The byte offset the next segment will be read from.
    #[must_use]
    pub fn position(&self) -> usize {
        self.position
    }

    /// Changes the separators for the segments still to come, as a new ISA does.
    pub fn set_separators(&mut self, separators: Separators) {
        self.separators = separators;
    }

    /// Moves the read position, as reading an ISA's fixed layout does.
    pub fn seek(&mut self, position: usize) {
        self.position = position.min(self.text.len());
    }
}

impl<'a> Iterator for Lexer<'a> {
    type Item = Segment<'a>;

    fn next(&mut self) -> Option<Self::Item> {
        let bytes = self.text.as_bytes();
        while self.position < bytes.len() && is_gap(bytes[self.position]) {
            self.position += 1;
        }
        if self.position >= bytes.len() {
            return None;
        }
        let start = self.position;
        let terminator = bytes[start..]
            .iter()
            .position(|byte| *byte == self.separators.segment)
            .map(|offset| start + offset);
        let (mut end, terminated, next) = match terminator {
            Some(at) => (at, true, at + 1),
            None => (bytes.len(), false, bytes.len()),
        };
        while end > start && matches!(bytes[end - 1], b'\r' | b'\n') {
            end -= 1;
        }
        if !terminated {
            while end > start && bytes[end - 1].is_ascii_whitespace() {
                end -= 1;
            }
        }
        self.position = next;
        let raw = &self.text[start..end];
        let id_end = raw
            .find(char::from(self.separators.element))
            .unwrap_or(raw.len());
        Some(Segment {
            id: &raw[..id_end],
            raw,
            start,
            end,
            terminated,
        })
    }
}

fn is_gap(byte: u8) -> bool {
    matches!(byte, b'\r' | b'\n' | b'\t' | b' ' | 0x0c)
}

#[cfg(test)]
mod tests {
    use super::{Lexer, components, repetitions};
    use crate::x12::separators::Separators;

    #[test]
    fn splits_segments_and_skips_line_breaks_between_them() {
        let text = "ST*204*0001~\r\nB2**SCAC**SHIP**PP~\nSE*3*0001~";
        let segments: Vec<_> = Lexer::new(text, 0, Separators::default()).collect();
        let ids: Vec<_> = segments.iter().map(|segment| segment.id).collect();
        assert_eq!(ids, ["ST", "B2", "SE"]);
        assert_eq!(segments[1].start, 14);
        assert_eq!(
            &text[segments[1].start..segments[1].end],
            "B2**SCAC**SHIP**PP"
        );
        assert!(segments.iter().all(|segment| segment.terminated));
    }

    #[test]
    fn reads_elements_by_position_and_their_spans() {
        let separators = Separators::default();
        let segment = Lexer::new("N1*SH*ACME DC*93*A1~", 0, separators)
            .next()
            .expect("a segment");
        assert_eq!(segment.element(&separators, 2), "ACME DC");
        assert_eq!(segment.element(&separators, 9), "");
        assert_eq!(segment.element_count(&separators), 4);
        assert_eq!(segment.element_span(&separators, 2), (6, 13));
        assert!(segment.has_valid_id());
    }

    #[test]
    fn keeps_text_cut_short_as_an_unterminated_segment() {
        let segments: Vec<_> = Lexer::new("ST*204*1~B2**X  \n", 0, Separators::default()).collect();
        assert_eq!(segments.len(), 2);
        assert!(!segments[1].terminated);
        assert_eq!(segments[1].raw, "B2**X");
    }

    #[test]
    fn splits_repetitions_and_components() {
        let separators = Separators::default();
        assert_eq!(repetitions("A^B", &separators), ["A", "B"]);
        assert_eq!(components("1>2>", &separators), ["1", "2", ""]);
        let without = Separators {
            repetition: None,
            ..separators
        };
        assert_eq!(repetitions("A^B", &without), ["A^B"]);
    }

    #[test]
    fn a_segment_id_must_look_like_one() {
        let separators = Separators::default();
        for (text, valid) in [
            ("N1*A~", true),
            ("ISA*A~", true),
            ("n1*A~", false),
            ("1N*A~", false),
            ("ABCD*A~", false),
        ] {
            let segment = Lexer::new(text, 0, separators).next().expect("a segment");
            assert_eq!(segment.has_valid_id(), valid, "{text}");
        }
    }
}
