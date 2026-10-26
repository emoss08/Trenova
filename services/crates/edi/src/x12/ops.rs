//! The X12 operations' handlers: decode the input, run, encode the output.

use serde::{Deserialize, Serialize};
use serde_json::value::RawValue;
use trenova_module_kit::ops::{OpResult, RequestError, decode, encode};

use super::ack::{AcknowledgeRequest, acknowledge as build_acknowledgments};
use super::guide::builtin;
use super::guide::check::{GuideProblem, check_guide as check};
use super::guide::model::Guide;
use super::read::{ReadRequest, read as read_text};
use super::write::{WriteRequest, write as write_segments};

/// `x12.read`: reads a text against its guides.
pub fn read(input: &RawValue) -> OpResult {
    let request: ReadRequest = decode(input)?;
    encode(&read_text(&request)?)
}

/// `x12.acknowledge`: writes the 997 or 999 for each group given.
pub fn acknowledge(input: &RawValue) -> OpResult {
    let request: AcknowledgeRequest = decode(input)?;
    encode(&build_acknowledgments(&request)?)
}

/// `x12.write`: writes segments as text.
pub fn write(input: &RawValue) -> OpResult {
    let request: WriteRequest = decode(input)?;
    encode(&write_segments(&request)?)
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct GuideSummary<'a> {
    id: &'a str,
    name: &'a str,
    transaction_set: &'a str,
    functional_group: &'a str,
    version: &'a str,
}

#[derive(Serialize)]
struct GuideList<'a> {
    guides: Vec<GuideSummary<'a>>,
}

/// `x12.guides`: lists the base guides.
pub fn guides(_: &RawValue) -> OpResult {
    encode(&GuideList {
        guides: builtin::all()
            .iter()
            .map(|compiled| {
                let guide = compiled.guide();
                GuideSummary {
                    id: &guide.id,
                    name: &guide.name,
                    transaction_set: &guide.transaction_set,
                    functional_group: &guide.functional_group,
                    version: &guide.version,
                }
            })
            .collect(),
    })
}

#[derive(Deserialize)]
struct GuideRequest {
    id: String,
}

/// `x12.guide`: one base guide in full.
pub fn guide(input: &RawValue) -> OpResult {
    let request: GuideRequest = decode(input)?;
    let compiled = builtin::by_id(&request.id).ok_or_else(|| {
        RequestError::invalid_input(format!("no base guide is named {:?}", request.id))
    })?;
    encode(compiled.guide())
}

#[derive(Deserialize)]
struct CheckRequest {
    guide: Guide,
}

#[derive(Serialize)]
struct CheckResult {
    problems: Vec<GuideProblem>,
}

/// `x12.checkGuide`: checks a guide, and when it names a base guide in `derivedFrom`, that it
/// only narrows it.
pub fn check_guide(input: &RawValue) -> OpResult {
    let request: CheckRequest = decode(input)?;
    let base = match request.guide.derived_from.as_deref() {
        Some(id) => Some(
            builtin::by_id(id)
                .ok_or_else(|| {
                    RequestError::invalid_input(format!("no base guide is named {id:?}"))
                })?
                .guide(),
        ),
        None => None,
    };
    encode(&CheckResult {
        problems: check(&request.guide, base),
    })
}

#[cfg(test)]
mod tests {
    use crate::ops::dispatch;

    fn call(request: &str) -> serde_json::Value {
        let response = dispatch(request.as_bytes());
        serde_json::from_slice(&response).unwrap_or_default()
    }

    #[test]
    fn lists_and_returns_base_guides() {
        let listed = call(r#"{"op":"x12.guides"}"#);
        let ids: Vec<&str> = listed["ok"]["guides"]
            .as_array()
            .map(|guides| {
                guides
                    .iter()
                    .filter_map(|guide| guide["id"].as_str())
                    .collect()
            })
            .unwrap_or_default();
        assert_eq!(
            ids,
            [
                "x12.004010.204",
                "x12.004010.210",
                "x12.004010.214",
                "x12.004010.990",
                "x12.004010.997",
                "x12.005010.999"
            ]
        );
        let guide = call(r#"{"op":"x12.guide","input":{"id":"x12.004010.990"}}"#);
        assert_eq!(guide["ok"]["transactionSet"], "990");
        assert_eq!(guide["ok"]["segments"]["B1"]["elements"][0]["ref"], "140");
        let missing = call(r#"{"op":"x12.guide","input":{"id":"nope"}}"#);
        assert_eq!(missing["error"]["code"], "invalid_input");
    }

    #[test]
    fn checks_a_partner_guide_against_its_base() {
        let base = call(r#"{"op":"x12.guide","input":{"id":"x12.004010.990"}}"#);
        let mut guide = base["ok"].clone();
        guide["derivedFrom"] = serde_json::Value::from("x12.004010.990");
        guide["segments"]["B1"]["elements"][1]["max"] = serde_json::Value::from(99);
        let request = serde_json::json!({"op": "x12.checkGuide", "input": {"guide": guide}});
        let checked = call(&request.to_string());
        let problems = checked["ok"]["problems"]
            .as_array()
            .cloned()
            .unwrap_or_default();
        assert_eq!(problems.len(), 1, "{checked}");
        assert_eq!(problems[0]["path"], "segments/B1/02");
        assert_eq!(problems[0]["kind"], "widening");
    }

    #[test]
    fn reads_over_the_wire() {
        let text = "ST*990*0001~B1*SIML*SHIP-1*20260107*A~SE*3*0001~";
        let request =
            serde_json::json!({"op": "x12.read", "input": {"text": text, "fragment": true}});
        let reading = call(&request.to_string());
        let transaction = &reading["ok"]["interchanges"][0]["groups"][0]["transactions"][0];
        assert_eq!(transaction["status"], "accepted", "{reading}");
        assert_eq!(reading["ok"]["segments"][1]["elements"][1], "SHIP-1");
    }
}
