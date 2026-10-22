//! Positions in formula text: byte offsets inside the crate, UTF-16 code units on the wire.
//!
//! Rust slices strings by byte, so every span the lexer and parser produce is a byte range. The
//! editor (CodeMirror) and the server's statement traces count UTF-16 code units, so every
//! position that leaves the crate goes through a [`Utf16Index`], and every position that comes
//! in (a cursor for signature help or hover) goes back through it.

use serde::Serialize;

/// A half-open byte range `[start, end)` into the source it was produced from.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Default, PartialOrd, Ord)]
pub struct Span {
    /// The first byte.
    pub start: u32,
    /// One past the last byte.
    pub end: u32,
}

impl Span {
    /// A span from `start` to `end`.
    #[must_use]
    pub const fn new(start: u32, end: u32) -> Self {
        Self { start, end }
    }

    /// The smallest span covering both.
    #[must_use]
    pub fn cover(self, other: Self) -> Self {
        Self {
            start: self.start.min(other.start),
            end: self.end.max(other.end),
        }
    }
}

/// A range on the wire, in UTF-16 code units: `{"from": 3, "to": 6}`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, serde::Deserialize, Default)]
pub struct Range {
    /// The first code unit.
    pub from: u32,
    /// One past the last code unit.
    pub to: u32,
}

/// Converts between byte offsets and UTF-16 offsets for one source text.
///
/// Built once per request; conversion is a binary search over the non-ASCII characters only, so
/// an ASCII formula costs nothing beyond the scan.
#[derive(Debug, Clone)]
pub struct Utf16Index {
    /// For each non-ASCII character: its byte offset, and how many extra UTF-16 units precede the
    /// byte just after it compared with its byte count (negative drift accumulates).
    marks: Vec<(u32, i64)>,
    len_bytes: u32,
}

impl Utf16Index {
    /// Indexes `source`.
    #[must_use]
    pub fn new(source: &str) -> Self {
        let mut marks = Vec::new();
        let mut drift: i64 = 0;
        for (offset, character) in source.char_indices() {
            let bytes = character.len_utf8();
            if bytes > 1 {
                drift += i64::try_from(character.len_utf16()).unwrap_or(2)
                    - i64::try_from(bytes).unwrap_or(4);
                marks.push((u32::try_from(offset + bytes).unwrap_or(u32::MAX), drift));
            }
        }
        Self {
            marks,
            len_bytes: u32::try_from(source.len()).unwrap_or(u32::MAX),
        }
    }

    /// The UTF-16 offset of a byte offset (clamped to the text).
    #[must_use]
    pub fn to_utf16(&self, byte: u32) -> u32 {
        let byte = byte.min(self.len_bytes);
        let drift = match self.marks.partition_point(|(mark, _)| *mark <= byte) {
            0 => 0,
            index => self.marks[index - 1].1,
        };
        u32::try_from(i64::from(byte) + drift).unwrap_or(0)
    }

    /// The wire range of a byte span.
    #[must_use]
    pub fn range(&self, span: Span) -> Range {
        Range {
            from: self.to_utf16(span.start),
            to: self.to_utf16(span.end),
        }
    }

    /// The byte offset of a UTF-16 offset, rounded down to a character boundary.
    #[must_use]
    pub fn to_byte(&self, source: &str, utf16: u32) -> u32 {
        let mut units = 0u32;
        for (offset, character) in source.char_indices() {
            let width = u32::try_from(character.len_utf16()).unwrap_or(2);
            if units + width > utf16 {
                return u32::try_from(offset).unwrap_or(u32::MAX);
            }
            units += width;
        }
        self.len_bytes
    }
}

#[cfg(test)]
mod tests {
    use super::{Span, Utf16Index};

    #[test]
    fn ascii_offsets_are_unchanged() {
        let index = Utf16Index::new("baseRate * 2");
        assert_eq!(index.to_utf16(5), 5);
        assert_eq!(index.to_byte("baseRate * 2", 5), 5);
    }

    #[test]
    fn non_ascii_characters_count_in_utf16_units() {
        let source = "let tarifa_año = 1; tarifa_año + bar";
        let index = Utf16Index::new(source);
        let bar = u32::try_from(source.find("bar").unwrap_or_default()).unwrap_or_default();
        assert_eq!(index.to_utf16(bar), 33);
        assert_eq!(index.to_byte(source, 33), bar);
    }

    #[test]
    fn astral_characters_are_two_units() {
        let source = "\"𝄞\" + x";
        let index = Utf16Index::new(source);
        let x = u32::try_from(source.find('x').unwrap_or_default()).unwrap_or_default();
        assert_eq!(index.to_utf16(x), 7);
        assert_eq!(index.range(Span::new(0, x)).to, 7);
    }
}
