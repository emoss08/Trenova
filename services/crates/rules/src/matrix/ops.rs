//! The `matrix.*` operations.

use crate::allocation::input::nullable;
use crate::allocation::input::optional_decimal;
use crate::matrix::coverage::find_issues;
use crate::matrix::input::Cell;
use crate::matrix::input::Dimension;
use crate::matrix::input::MAX_DIMENSIONS;
use crate::matrix::input::dimensions as checked_dimensions;
use crate::matrix::select::AxisValue;
use crate::matrix::select::SelectError;
use crate::matrix::select::select_cell;
use crate::message;
use crate::message::Arg;
use crate::message::Message;
use crate::ops::OpResult;
use crate::ops::RequestError;
use crate::ops::decode;
use crate::ops::encode;
use crate::wire::Dec;
use rust_decimal::Decimal;
use serde::Deserialize;
use serde::Serialize;
use serde_json::value::RawValue;

#[derive(Deserialize, Default)]
#[serde(default)]
struct ValueInput {
    #[serde(deserialize_with = "nullable")]
    keys: Vec<String>,
    #[serde(deserialize_with = "optional_decimal")]
    quantity: Option<Decimal>,
}

#[derive(Deserialize, Default)]
#[serde(default)]
struct SelectInput {
    #[serde(deserialize_with = "nullable")]
    dimensions: Vec<Option<Dimension>>,
    #[serde(deserialize_with = "nullable")]
    cells: Vec<Option<Cell>>,
    #[serde(deserialize_with = "nullable")]
    values: Vec<Option<ValueInput>>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct CellOut<'a> {
    index: usize,
    #[serde(skip_serializing_if = "Option::is_none")]
    id: Option<&'a str>,
    preference: usize,
    matched_keys: [String; MAX_DIMENSIONS],
}

#[derive(Serialize)]
struct ErrorOut {
    code: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    position: Option<i64>,
    message: Message,
}

#[derive(Serialize)]
struct SelectOut<'a> {
    #[serde(skip_serializing_if = "Option::is_none")]
    cell: Option<CellOut<'a>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<ErrorOut>,
}

/// `matrix.select`: the one cell that prices a lookup, or why there is none.
pub fn select(input: &RawValue) -> OpResult {
    let input: SelectInput = decode(input)?;
    if input.values.len() > MAX_DIMENSIONS {
        return Err(RequestError::invalid_input(
            "a lookup gives at most four values, one per axis position",
        ));
    }
    let dimensions = checked_dimensions(&input.dimensions)?;
    let cells: Vec<(usize, &Cell)> = input
        .cells
        .iter()
        .enumerate()
        .filter_map(|(index, cell)| cell.as_ref().map(|cell| (index, cell)))
        .collect();
    let mut values: [AxisValue; MAX_DIMENSIONS] = Default::default();
    for (slot, value) in input.values.into_iter().enumerate() {
        if let Some(value) = value {
            values[slot] = AxisValue {
                keys: value.keys,
                quantity: value.quantity,
            };
        }
    }
    let out = match select_cell(&dimensions, &cells, &values) {
        Ok(selection) => SelectOut {
            cell: Some(CellOut {
                index: selection.index,
                id: input
                    .cells
                    .get(selection.index)
                    .and_then(Option::as_ref)
                    .and_then(|cell| cell.id.as_deref()),
                preference: selection.preference,
                matched_keys: selection.matched_keys,
            }),
            error: None,
        },
        Err(SelectError::DimensionValueMissing { dimension, .. }) => {
            let (_, axis) = dimensions[dimension];
            SelectOut {
                cell: None,
                error: Some(ErrorOut {
                    code: "dimension_value_missing",
                    position: Some(axis.position),
                    message: Message::with(
                        message!("No value was supplied for the matrix dimension {0}"),
                        [Arg::from(axis.display_label())],
                    ),
                }),
            }
        }
        Err(SelectError::NoMatchingCell) => SelectOut {
            cell: None,
            error: Some(ErrorOut {
                code: "no_matching_cell",
                position: None,
                message: Message::new(message!("No matrix cell matches the supplied values")),
            }),
        },
    };
    encode(&out)
}

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
