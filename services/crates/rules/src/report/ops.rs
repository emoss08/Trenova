//! The `report.*` operations.

use crate::ops::OpResult;
use crate::ops::RequestError;
use serde_json::value::RawValue;

/// Formats columns of report values.
pub fn format(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "report.format is not built yet",
    ))
}
