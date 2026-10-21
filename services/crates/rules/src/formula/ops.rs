//! The static formula operations: catalog, check, format, signature help and hover.

use crate::ops::OpResult;
use crate::ops::RequestError;
use serde_json::value::RawValue;

/// Lists every function, built-in and disabled name.
pub fn catalog(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "formula.catalog is not built yet",
    ))
}

/// Checks one formula against a schema and returns its analysis.
pub fn check(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "formula.check is not built yet",
    ))
}

/// Formats one formula.
pub fn format(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "formula.format is not built yet",
    ))
}

/// Finds the call and argument at a cursor.
pub fn signature(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "formula.signature is not built yet",
    ))
}

/// Finds the identifier at a cursor.
pub fn identifier(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "formula.identifier is not built yet",
    ))
}
