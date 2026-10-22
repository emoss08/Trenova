//! Reading a stored lane key back into its ends.
//!
//! A key is shown to people with every record ID replaced by the record's name, so a caller has
//! to know which parts of a key are IDs. `LOC:`, `ZN:`, `ST:` and `CS:` carry an ID (a
//! `CS:` key's ID is the state before the `|`); postal, country and `ANY` ends are literal and
//! are their own display. Keys that arrive already assembled (import diffs, simulation rows)
//! and keys built by `lane.keys` are read the same way.

use crate::lane::scope::LANE_KEY_SEPARATOR;
use serde::Serialize;

/// One end of a lane key.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct Segment {
    /// The end as stored.
    pub text: String,
    /// The scope type whose record the end references: `Location`, `Zone`, `State` or
    /// `CityState`; absent on a literal end.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scope: Option<&'static str>,
    /// The referenced record's ID.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub id: Option<String>,
    /// What precedes the ID (`CS:`).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub prefix: Option<String>,
    /// What follows the ID (`|DALLAS`).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub suffix: Option<String>,
}

/// Splits a lane key at `>` and reads each end.
#[must_use]
pub fn parse_key(key: &str) -> Vec<Segment> {
    key.split(LANE_KEY_SEPARATOR).map(parse_segment).collect()
}

fn parse_segment(segment: &str) -> Segment {
    let literal = || Segment {
        text: segment.to_owned(),
        ..Segment::default()
    };
    let Some((prefix, rest)) = segment.split_once(':') else {
        return literal();
    };
    let scope = match prefix {
        "LOC" => "Location",
        "ZN" => "Zone",
        "ST" => "State",
        "CS" => "CityState",
        _ => return literal(),
    };
    let (id, suffix) = if scope == "CityState" {
        match rest.find('|') {
            Some(bar) => (&rest[..bar], &rest[bar..]),
            None => (rest, ""),
        }
    } else {
        (rest, "")
    };
    if id.is_empty() {
        return literal();
    }
    Segment {
        text: segment.to_owned(),
        scope: Some(scope),
        id: Some(id.to_owned()),
        prefix: Some(format!("{prefix}:")),
        suffix: Some(suffix.to_owned()),
    }
}
