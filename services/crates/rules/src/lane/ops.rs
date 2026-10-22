//! The `lane.*` operations, each batched over many rules or keys.

use crate::allocation::input::nullable;
use crate::lane::coverage::find_issues;
use crate::lane::parse::Segment;
use crate::lane::parse::parse_key;
use crate::lane::rule::RuleInput;
use crate::lane::scope::Place;
use crate::lane::scope::lane_key_candidates;
use crate::message::Message;
use crate::ops::OpResult;
use crate::ops::decode;
use crate::ops::encode;
use serde::Deserialize;
use serde::Serialize;
use serde_json::value::RawValue;

#[derive(Deserialize)]
struct RulesInput {
    #[serde(default, deserialize_with = "nullable")]
    rules: Vec<RuleInput>,
}

#[derive(Serialize)]
struct ScoresOut {
    scores: Vec<i64>,
}

/// `lane.specificity`: each rule's specificity score.
pub fn specificity(input: &RawValue) -> OpResult {
    let input: RulesInput = decode(input)?;
    encode(&ScoresOut {
        scores: input.rules.iter().map(RuleInput::specificity).collect(),
    })
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct ShipmentLane {
    #[serde(default, deserialize_with = "nullable")]
    origin: Place,
    #[serde(default, deserialize_with = "nullable")]
    destination: Place,
}

#[derive(Deserialize)]
struct KeysInput {
    #[serde(default, deserialize_with = "nullable")]
    rules: Vec<RuleInput>,
    #[serde(default, deserialize_with = "nullable")]
    shipments: Vec<ShipmentLane>,
}

#[derive(Serialize)]
struct RuleKeysOut {
    origin: Option<String>,
    destination: Option<String>,
    key: Option<String>,
    stored: String,
    reverse: Option<String>,
    segments: Vec<Segment>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct ShipmentKeysOut {
    origin_keys: Vec<String>,
    destination_keys: Vec<String>,
    candidates: Vec<String>,
}

#[derive(Serialize)]
struct KeysOut {
    rules: Vec<RuleKeysOut>,
    shipments: Vec<ShipmentKeysOut>,
}

/// `lane.keys`: the keys each rule stores, and the keys each shipment lane can match.
pub fn keys(input: &RawValue) -> OpResult {
    let input: KeysInput = decode(input)?;
    let rules = input
        .rules
        .iter()
        .map(|rule| {
            let keys = rule.lane_keys();
            RuleKeysOut {
                segments: keys.key.as_deref().map(parse_key).unwrap_or_default(),
                origin: keys.origin,
                destination: keys.destination,
                key: keys.key,
                stored: keys.stored,
                reverse: keys.reverse,
            }
        })
        .collect();
    let shipments = input
        .shipments
        .iter()
        .map(|lane| {
            let origin_keys = lane.origin.match_keys();
            let destination_keys = lane.destination.match_keys();
            ShipmentKeysOut {
                candidates: lane_key_candidates(&origin_keys, &destination_keys),
                origin_keys,
                destination_keys,
            }
        })
        .collect();
    encode(&KeysOut { rules, shipments })
}

#[derive(Deserialize)]
struct ParseInput {
    #[serde(default, deserialize_with = "nullable")]
    keys: Vec<String>,
}

#[derive(Serialize)]
struct ParsedOut {
    segments: Vec<Segment>,
}

#[derive(Serialize)]
struct ParseOut {
    keys: Vec<ParsedOut>,
}

/// `lane.parse`: splits stored lane keys into their ends and the record IDs they reference.
pub fn parse(input: &RawValue) -> OpResult {
    let input: ParseInput = decode(input)?;
    encode(&ParseOut {
        keys: input
            .keys
            .iter()
            .map(|key| ParsedOut {
                segments: parse_key(key),
            })
            .collect(),
    })
}

#[derive(Serialize)]
struct IssueOut {
    index: usize,
    kind: &'static str,
    other: usize,
    message: Message,
}

#[derive(Serialize)]
struct CoverageOut {
    issues: Vec<IssueOut>,
}

/// `lane.coverage`: the lanes of one agreement that can never apply.
pub fn coverage(input: &RawValue) -> OpResult {
    let input: RulesInput = decode(input)?;
    encode(&CoverageOut {
        issues: find_issues(&input.rules)
            .into_iter()
            .map(|issue| IssueOut {
                index: issue.index,
                kind: issue.kind.name(),
                other: issue.other,
                message: issue.message,
            })
            .collect(),
    })
}
