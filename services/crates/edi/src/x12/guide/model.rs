//! An implementation guide as data: what a transaction set must look like for one partner.
//!
//! A guide is what an EDI specialist reads and what a partner hands over: which segments a
//! transaction set uses, in which loops, how often, and what each element may hold. Trenova keeps
//! it as data so a person can read, narrow and test it in the product, and so the same guide
//! drives reading, validation and (later) mapping.
//!
//! A guide has a **segment dictionary** (each segment's elements in order) and a **structure**
//! (the tree of segment uses and loops, ST first and SE last). A partner's guide names its base
//! in `derivedFrom` and may only narrow it; `x12.checkGuide` reports anything that widens.
//!
//! This is the wire shape of a guide (camelCase JSON). Built-in base guides are written in the
//! compact text form of [`super::text`] and parsed into this.

use std::collections::BTreeMap;

use serde::{Deserialize, Deserializer, Serialize, Serializer};

use crate::x12::finding::{FindingKind, Severity};

/// One implementation guide.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Guide {
    /// A stable identifier: `x12.004010.204` for a base guide, anything unique for a partner's.
    pub id: String,
    /// What people call it ("Motor Carrier Load Tender").
    pub name: String,
    /// Notes for people.
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub description: String,
    /// The transaction set identifier (ST01), `204`.
    pub transaction_set: String,
    /// The functional identifier code its groups carry (GS01), `SM`.
    pub functional_group: String,
    /// The version, release and industry identifier (GS08), `004010` or `005010X231A1`.
    pub version: String,
    /// The guide this one narrows, when it is a partner's.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub derived_from: Option<String>,
    /// Every segment the structure uses, by ID.
    pub segments: BTreeMap<String, SegmentDef>,
    /// The transaction set's structure, ST first and SE last.
    pub structure: Vec<Node>,
    /// Severity overrides by finding kind: a partner that accepts a code outside its list as a
    /// warning says so here.
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub severities: BTreeMap<FindingKind, Severity>,
}

/// One segment's definition.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SegmentDef {
    /// The segment's name ("Name", "Stop Off Details").
    pub name: String,
    /// Its elements, element 01 first.
    pub elements: Vec<ElementDef>,
    /// Its syntax notes: `P0304` (paired), `R0203` (at least one), `E0102` (at most one),
    /// `C0405` (if the first, then all others), `L010203` (if the first, then at least one
    /// other).
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub rules: Vec<String>,
    /// Notes for people.
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub notes: String,
}

/// One element's definition, or one component's within a composite.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ElementDef {
    /// The data element reference number (`98`), or the composite's (`C030`).
    #[serde(rename = "ref")]
    pub reference: String,
    /// The element's name.
    pub name: String,
    /// Its data type.
    #[serde(rename = "type")]
    pub kind: ElementType,
    /// Its minimum length.
    pub min: u32,
    /// Its maximum length.
    pub max: u32,
    /// Whether it must, may, may under a syntax note, or must not appear.
    pub usage: Usage,
    /// How many repetitions it may have; 1 for an element that does not repeat.
    #[serde(default = "one", skip_serializing_if = "is_one")]
    pub repeat: u32,
    /// The codes it may hold, when the guide restricts them.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub codes: Option<Vec<Code>>,
    /// The one value it must hold, when the guide fixes it.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub fixed: Option<String>,
    /// Its components, when it is a composite.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub components: Option<Vec<ElementDef>>,
    /// Notes for people.
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub notes: String,
}

fn one() -> u32 {
    1
}

#[allow(clippy::trivially_copy_pass_by_ref)]
fn is_one(value: &u32) -> bool {
    *value == 1
}

/// One code an element may hold.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Code {
    /// The code as written.
    pub code: String,
    /// What it means.
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub description: String,
}

/// One node of a guide's structure: a segment use or a loop.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(untagged)]
pub enum Node {
    /// A segment at this point of the structure.
    Segment(SegmentUse),
    /// A loop: a group of segments that repeats together, started by its first segment.
    Loop(LoopDef),
}

/// A segment at one point of the structure.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SegmentUse {
    /// The segment ID, defined in the guide's dictionary.
    pub segment: String,
    /// Whether it must, may, or must not appear here.
    pub usage: Usage,
    /// The most times it may appear here in a row.
    pub max: u32,
    /// The fewest times it must appear when it appears; 0 means the standard's (1 when
    /// mandatory).
    #[serde(default, skip_serializing_if = "is_zero")]
    pub min: u32,
    /// Notes for people.
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub notes: String,
}

/// A loop in the structure.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LoopDef {
    /// The loop identifier (`0300`), written in a 999's IK303.
    #[serde(rename = "loop")]
    pub id: String,
    /// What people call it ("Stop Off Details").
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub name: String,
    /// Whether it must, may, or must not appear.
    pub usage: Usage,
    /// The most times it may repeat.
    pub repeat: u32,
    /// The fewest times it must repeat when it appears; 0 means the standard's.
    #[serde(default, skip_serializing_if = "is_zero")]
    pub min: u32,
    /// Its segments and loops; the first is a segment, and it starts each repetition.
    pub children: Vec<Node>,
    /// Notes for people.
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub notes: String,
}

#[allow(clippy::trivially_copy_pass_by_ref)]
fn is_zero(value: &u32) -> bool {
    *value == 0
}

/// Whether something must appear.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum Usage {
    /// It must appear (X12 `M`).
    Mandatory,
    /// It may appear (X12 `O`).
    Optional,
    /// It may appear, and a syntax note decides when it must (X12 `X`).
    Conditional,
    /// The partner does not use it, so it must not appear.
    NotUsed,
}

impl Usage {
    /// Reads an X12 usage letter: `M`, `O`, `X` or `N`.
    #[must_use]
    pub fn from_letter(letter: &str) -> Option<Self> {
        match letter {
            "M" => Some(Self::Mandatory),
            "O" => Some(Self::Optional),
            "X" => Some(Self::Conditional),
            "N" => Some(Self::NotUsed),
            _ => None,
        }
    }

    /// How strongly it requires presence, for comparing a partner's narrowing with its base:
    /// not used and mandatory are both narrower than optional.
    #[must_use]
    pub fn is_narrowing_of(self, base: Self) -> bool {
        match base {
            Self::Optional | Self::Conditional => true,
            Self::Mandatory => self == Self::Mandatory,
            Self::NotUsed => self == Self::NotUsed,
        }
    }
}

/// An element's data type.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum ElementType {
    /// `AN`: any characters of the character set.
    Alphanumeric,
    /// `ID`: a code.
    Identifier,
    /// `DT`: a date, `CCYYMMDD` or `YYMMDD`.
    Date,
    /// `TM`: a time, `HHMM`, `HHMMSS` or `HHMMSSd..d`.
    Time,
    /// `N0` to `N9`: a whole number with that many implied decimal places.
    Numeric(u8),
    /// `R`: a decimal number with an explicit point.
    Decimal,
    /// `B`: binary data.
    Binary,
    /// A composite: its components carry the types.
    Composite,
}

impl ElementType {
    /// Reads an X12 type code (`AN`, `ID`, `DT`, `TM`, `N0`..`N9`, `R`, `B`) or `composite`.
    #[must_use]
    pub fn from_code(code: &str) -> Option<Self> {
        match code {
            "AN" => Some(Self::Alphanumeric),
            "ID" => Some(Self::Identifier),
            "DT" => Some(Self::Date),
            "TM" => Some(Self::Time),
            "R" => Some(Self::Decimal),
            "B" => Some(Self::Binary),
            "composite" => Some(Self::Composite),
            "N" => Some(Self::Numeric(0)),
            _ => {
                let digits = code.strip_prefix('N')?;
                let [digit] = digits.as_bytes() else {
                    return None;
                };
                digit.is_ascii_digit().then(|| Self::Numeric(digit - b'0'))
            }
        }
    }

    /// The X12 type code.
    #[must_use]
    pub fn code(self) -> String {
        match self {
            Self::Alphanumeric => String::from("AN"),
            Self::Identifier => String::from("ID"),
            Self::Date => String::from("DT"),
            Self::Time => String::from("TM"),
            Self::Numeric(places) => format!("N{places}"),
            Self::Decimal => String::from("R"),
            Self::Binary => String::from("B"),
            Self::Composite => String::from("composite"),
        }
    }
}

impl Serialize for ElementType {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.serialize_str(&self.code())
    }
}

impl<'de> Deserialize<'de> for ElementType {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let code = String::deserialize(deserializer)?;
        Self::from_code(&code)
            .ok_or_else(|| serde::de::Error::custom(format!("{code:?} is not an X12 data type")))
    }
}

impl Guide {
    /// The X12 version without an implementation convention suffix: `005010` for
    /// `005010X231A1`.
    #[must_use]
    pub fn base_version(&self) -> &str {
        self.version.get(..6).unwrap_or(&self.version)
    }
}

#[cfg(test)]
mod tests {
    use super::{ElementType, Guide, Node, Usage};

    #[test]
    fn element_types_read_and_write_their_x12_codes() {
        for code in ["AN", "ID", "DT", "TM", "N0", "N2", "R", "B", "composite"] {
            let kind = ElementType::from_code(code).expect(code);
            assert_eq!(kind.code(), code);
        }
        assert_eq!(ElementType::from_code("N"), Some(ElementType::Numeric(0)));
        assert_eq!(ElementType::from_code("N10"), None);
        assert_eq!(ElementType::from_code("XX"), None);
    }

    #[test]
    fn a_partner_may_only_narrow_usage() {
        assert!(Usage::Mandatory.is_narrowing_of(Usage::Optional));
        assert!(Usage::NotUsed.is_narrowing_of(Usage::Optional));
        assert!(!Usage::Optional.is_narrowing_of(Usage::Mandatory));
        assert!(!Usage::NotUsed.is_narrowing_of(Usage::Mandatory));
    }

    #[test]
    fn guides_round_trip_as_json() {
        let json = r#"{"id":"p","name":"P","transactionSet":"990","functionalGroup":"GF","version":"004010","segments":{"ST":{"name":"Transaction Set Header","elements":[{"ref":"143","name":"Transaction Set Identifier Code","type":"ID","min":3,"max":3,"usage":"mandatory"}]}},"structure":[{"segment":"ST","usage":"mandatory","max":1},{"loop":"0100","usage":"optional","repeat":5,"children":[{"segment":"ST","usage":"mandatory","max":1}]}]}"#;
        let guide: Guide = serde_json::from_str(json).expect("a guide");
        assert!(matches!(guide.structure[1], Node::Loop(_)));
        assert_eq!(serde_json::to_string(&guide).expect("json"), json);
    }
}
