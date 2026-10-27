//! The four separators an X12 interchange is written with, and where they come from.
//!
//! X12 declares its separators in the ISA segment itself: the element separator is the byte
//! right after `ISA`, ISA16 is the component separator, the byte after ISA16 is the segment
//! terminator, and from version 00402 on ISA11 is the repetition separator. A conforming ISA
//! is exactly 106 bytes, so the separators sit at fixed offsets; a sample copied out of a PDF
//! loses that padding, so the separators are found by counting element separators instead of
//! by offset, and the ISA is reported as squeezed rather than refused.
//!
//! Text without an ISA (a transaction set pasted on its own) takes the separators the caller
//! supplies, normally the partner profile's, or failing that the ones the text itself shows.

use serde::{Deserialize, Serialize};

/// Where a reading's separators came from.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum SeparatorSource {
    /// Read from the interchange's ISA segment.
    Isa,
    /// Supplied by the caller, for text without an ISA.
    Request,
    /// Guessed from the text, for text with neither an ISA nor separators from the caller.
    Detected,
}

/// What follows each segment terminator in the text, when anything does.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub enum LineEnding {
    /// Nothing: segments follow each other directly.
    #[default]
    None,
    /// A line feed.
    Lf,
    /// A carriage return and a line feed.
    CrLf,
    /// A carriage return.
    Cr,
}

impl LineEnding {
    /// The bytes written after each segment terminator.
    #[must_use]
    pub fn as_str(self) -> &'static str {
        match self {
            Self::None => "",
            Self::Lf => "\n",
            Self::CrLf => "\r\n",
            Self::Cr => "\r",
        }
    }
}

/// An interchange's separators.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Separators {
    /// Between the elements of a segment (`*` by convention).
    pub element: u8,
    /// Between the components of a composite element (`>` or `:`).
    pub component: u8,
    /// Between the repetitions of a repeating element, when the version has one (`^`).
    pub repetition: Option<u8>,
    /// After each segment (`~`).
    pub segment: u8,
    /// What follows each terminator in this text.
    pub line_ending: LineEnding,
}

impl Default for Separators {
    fn default() -> Self {
        Self {
            element: b'*',
            component: b'>',
            repetition: Some(b'^'),
            segment: b'~',
            line_ending: LineEnding::None,
        }
    }
}

/// Separators as they cross the wire: one-character strings.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase", default)]
pub struct WireSeparators {
    /// The element separator.
    pub element: String,
    /// The component separator.
    pub component: String,
    /// The repetition separator, empty when the interchange has none.
    pub repetition: String,
    /// The segment terminator.
    pub segment: String,
    /// What follows each terminator.
    pub line_ending: LineEnding,
}

impl Separators {
    /// The separators on the wire.
    #[must_use]
    pub fn to_wire(self) -> WireSeparators {
        WireSeparators {
            element: char::from(self.element).to_string(),
            component: char::from(self.component).to_string(),
            repetition: self
                .repetition
                .map(|byte| char::from(byte).to_string())
                .unwrap_or_default(),
            segment: char::from(self.segment).to_string(),
            line_ending: self.line_ending,
        }
    }

    /// Separators from the wire, falling back to the conventional ones for anything empty.
    /// Returns `None` when a separator is not a single ASCII byte or two of them collide.
    #[must_use]
    pub fn from_wire(wire: &WireSeparators) -> Option<Self> {
        let defaults = Self::default();
        let element = single_byte(&wire.element).ok()?.unwrap_or(defaults.element);
        let component = single_byte(&wire.component)
            .ok()?
            .unwrap_or(defaults.component);
        let segment = single_byte(&wire.segment).ok()?.unwrap_or(defaults.segment);
        let repetition = single_byte(&wire.repetition).ok()?;
        let separators = Self {
            element,
            component,
            repetition,
            segment,
            line_ending: wire.line_ending,
        };
        separators.distinct().then_some(separators)
    }

    /// Whether no two separators are the same byte and none is a letter or digit.
    #[must_use]
    pub fn distinct(&self) -> bool {
        let mut bytes = vec![self.element, self.component, self.segment];
        bytes.extend(self.repetition);
        let unique = bytes
            .iter()
            .enumerate()
            .all(|(index, byte)| !bytes[..index].contains(byte));
        unique && bytes.iter().all(|byte| is_separator_byte(*byte))
    }
}

/// A one-character separator from the wire: `Ok(None)` when empty, `Err` when it is not one
/// ASCII byte.
fn single_byte(text: &str) -> Result<Option<u8>, NotOneByte> {
    match text.as_bytes() {
        [] => Ok(None),
        [byte] if byte.is_ascii() => Ok(Some(*byte)),
        _ => Err(NotOneByte),
    }
}

#[derive(Debug)]
struct NotOneByte;

/// Whether `byte` may serve as a separator: printable or control ASCII that is not a letter,
/// a digit or a space.
#[must_use]
pub fn is_separator_byte(byte: u8) -> bool {
    byte.is_ascii() && !byte.is_ascii_alphanumeric() && byte != b' '
}

/// The ISA's fixed length in bytes, terminator included.
pub const ISA_LENGTH: usize = 106;

/// What reading the separators from an ISA found.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct IsaSeparators {
    /// The separators.
    pub separators: Separators,
    /// Whether the ISA had its fixed width; `false` for a squeezed sample.
    pub fixed_width: bool,
    /// The byte offset of the ISA's terminator.
    pub terminator: usize,
    /// The byte offset just past the ISA's terminator and line ending.
    pub end: usize,
}

/// Reads the separators from the ISA starting at `start`, or `None` when the text there is not
/// an ISA that declares them.
#[must_use]
pub fn from_isa(text: &[u8], start: usize) -> Option<IsaSeparators> {
    let isa = text.get(start..)?;
    if !isa.starts_with(b"ISA") {
        return None;
    }
    let element = *isa.get(3)?;
    if !is_separator_byte(element) || element == b'\r' || element == b'\n' {
        return None;
    }
    let mut seen = 0usize;
    let mut sixteenth = None;
    let mut eleventh = None;
    for (offset, byte) in isa.iter().enumerate().skip(3) {
        if *byte == element {
            seen += 1;
            if seen == 11 {
                eleventh = Some(offset);
            }
            if seen == 16 {
                sixteenth = Some(offset);
                break;
            }
        }
        if offset > 4096 {
            return None;
        }
    }
    let sixteenth = sixteenth?;
    let component = *isa.get(sixteenth + 1)?;
    let segment = *isa.get(sixteenth + 2)?;
    if !is_separator_byte(component) || !is_separator_byte(segment) {
        return None;
    }
    let (line_ending, ending_len) = line_ending_after(isa, sixteenth + 3, segment);
    let repetition = eleventh.and_then(|offset| {
        let value = isa.get(offset + 1..)?;
        let byte = *value.first()?;
        let is_single = value.get(1) == Some(&element);
        (is_single && is_separator_byte(byte)).then_some(byte)
    });
    let separators = Separators {
        element,
        component,
        repetition,
        segment,
        line_ending,
    };
    Some(IsaSeparators {
        separators,
        fixed_width: sixteenth + 3 == ISA_LENGTH,
        terminator: start + sixteenth + 2,
        end: start + sixteenth + 3 + ending_len,
    })
}

fn line_ending_after(text: &[u8], offset: usize, segment: u8) -> (LineEnding, usize) {
    if segment == b'\n' || segment == b'\r' {
        return (LineEnding::None, 0);
    }
    match (text.get(offset), text.get(offset + 1)) {
        (Some(b'\r'), Some(b'\n')) => (LineEnding::CrLf, 2),
        (Some(b'\n'), _) => (LineEnding::Lf, 1),
        (Some(b'\r'), _) => (LineEnding::Cr, 1),
        _ => (LineEnding::None, 0),
    }
}

/// Guesses separators for text with no ISA: the element separator is the byte after the first
/// segment's two or three character ID, and the terminator is `~` when the text has one, else a
/// line feed.
#[must_use]
pub fn detect(text: &[u8]) -> Separators {
    let mut separators = Separators::default();
    let start = text
        .iter()
        .position(|byte| !byte.is_ascii_whitespace())
        .unwrap_or(0);
    let id_len = text[start..]
        .iter()
        .take_while(|byte| byte.is_ascii_alphanumeric())
        .count();
    if (2..=3).contains(&id_len)
        && let Some(byte) = text.get(start + id_len)
        && is_separator_byte(*byte)
        && *byte != b'\n'
        && *byte != b'\r'
    {
        separators.element = *byte;
    }
    if !text.contains(&b'~') && text.contains(&b'\n') {
        separators.segment = b'\n';
    } else if text.windows(2).any(|pair| pair == b"~\n") {
        separators.line_ending = if text.windows(3).any(|triple| triple == b"~\r\n") {
            LineEnding::CrLf
        } else {
            LineEnding::Lf
        };
    }
    separators
}

#[cfg(test)]
mod tests {
    use super::{LineEnding, Separators, WireSeparators, detect, from_isa};

    const ISA: &str = "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        *260107*1200*^*00401*000000104*0*P*>~";

    #[test]
    fn reads_a_fixed_width_isa() {
        assert_eq!(ISA.len(), 106);
        let found = from_isa(ISA.as_bytes(), 0);
        let found = found.expect("an ISA");
        assert!(found.fixed_width);
        assert_eq!(found.end, 106);
        assert_eq!(
            found.separators,
            Separators {
                element: b'*',
                component: b'>',
                repetition: Some(b'^'),
                segment: b'~',
                line_ending: LineEnding::None,
            }
        );
    }

    #[test]
    fn reads_a_squeezed_isa_and_its_line_ending() {
        let text = "ISA*00**00**ZZ*PARTNER*ZZ*TRENOVA*260107*1200*U*00401*104*0*P*:~\r\nGS*QM~";
        let found = from_isa(text.as_bytes(), 0).expect("an ISA");
        assert!(!found.fixed_width);
        assert_eq!(found.separators.component, b':');
        assert_eq!(found.separators.repetition, None, "U is not a separator");
        assert_eq!(found.separators.line_ending, LineEnding::CrLf);
        assert_eq!(&text[found.end..], "GS*QM~");
    }

    #[test]
    fn refuses_text_that_is_not_an_isa() {
        assert!(from_isa(b"GS*QM*A*B~", 0).is_none());
        assert!(from_isa(b"ISA*00*", 0).is_none());
        assert!(from_isa(b"ISAA00", 0).is_none());
    }

    #[test]
    fn detects_separators_without_an_isa() {
        let found = detect(b"ST|214|0001\nB10|PRO\nSE|3|0001\n");
        assert_eq!(found.element, b'|');
        assert_eq!(found.segment, b'\n');
        let found = detect(b"ST*204*0001~\nB2**X~\n");
        assert_eq!(found.segment, b'~');
        assert_eq!(found.line_ending, LineEnding::Lf);
    }

    #[test]
    fn separators_round_trip_the_wire_and_refuse_collisions() {
        let separators = Separators::default();
        assert_eq!(
            Separators::from_wire(&separators.to_wire()),
            Some(separators)
        );
        let colliding = WireSeparators {
            element: String::from("*"),
            component: String::from("*"),
            ..WireSeparators::default()
        };
        assert_eq!(Separators::from_wire(&colliding), None);
        let letter = WireSeparators {
            element: String::from("A"),
            ..WireSeparators::default()
        };
        assert_eq!(Separators::from_wire(&letter), None);
    }
}
