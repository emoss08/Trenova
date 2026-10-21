//! The `matrix.*` operations.

use crate::ops::OpResult;
use crate::ops::RequestError;
use serde_json::value::RawValue;

/// Selects the cell a query falls in.
pub fn select(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "matrix.select is not built yet",
    ))
}

/// Finds gaps, overlaps and duplicates.
pub fn coverage(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "matrix.coverage is not built yet",
    ))
}
