//! `formula.evaluate`.

use crate::ops::OpResult;
use crate::ops::RequestError;
use serde_json::value::RawValue;

/// Evaluates a batch of formulas against one environment.
pub fn evaluate(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "formula.evaluate is not built yet",
    ))
}
