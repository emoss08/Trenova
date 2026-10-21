//! The `lane.*` operations.

use crate::ops::OpResult;
use crate::ops::RequestError;
use serde_json::value::RawValue;

/// Scores rate agreement rules.
pub fn specificity(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "lane.specificity is not built yet",
    ))
}

/// Builds lane keys.
pub fn keys(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input("lane.keys is not built yet"))
}
