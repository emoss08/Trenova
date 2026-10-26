//! The `matrix.*` operations.

use crate::allocation::input::nullable;
use crate::matrix::coverage::find_issues;
use crate::matrix::input::Cell;
use crate::matrix::input::Dimension;
use crate::matrix::input::dimensions as checked_dimensions;
use crate::wire::Dec;
use serde::Deserialize;
use serde::Serialize;
use serde_json::value::RawValue;
use trenova_module_kit::message::Message;
use trenova_module_kit::ops::OpResult;
use trenova_module_kit::ops::decode;
use trenova_module_kit::ops::encode;

#[derive(Deserialize, Default)]
#[serde(default)]
struct CoverageInput {
    #[serde(deserialize_with = "nullable")]
    dimensions: Vec<Option<Dimension>>,
    #[serde(deserialize_with = "nullable")]
    cells: Vec<Option<Cell>>,
}

#[derive(Serialize)]
struct BoundsOut {
    min: Option<Dec>,
    max: Option<Dec>,
}

#[derive(Serialize)]
struct IssueOut {
    kind: &'static str,
    position: Option<i64>,
    message: Message,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    bands: Vec<BoundsOut>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    slices: Vec<String>,
}

#[derive(Serialize)]
struct CoverageOut {
    issues: Vec<IssueOut>,
}

/// `matrix.coverage`: gaps, overlaps, duplicates, a capped top band and empty matrices.
pub fn coverage(input: &RawValue) -> OpResult {
    let input: CoverageInput = decode(input)?;
    let dimensions = checked_dimensions(&input.dimensions)?;
    let cells: Vec<&Cell> = input.cells.iter().flatten().collect();
    let issues = find_issues(&dimensions, &cells)
        .into_iter()
        .map(|issue| IssueOut {
            kind: issue.kind.name(),
            position: issue.position,
            message: issue.message,
            bands: issue
                .bands
                .into_iter()
                .map(|bounds| BoundsOut {
                    min: bounds.min.map(Dec),
                    max: bounds.max.map(Dec),
                })
                .collect(),
            slices: issue.slices,
        })
        .collect();
    encode(&CoverageOut { issues })
}
