//! The `tariff.*` operations this area answers.

use crate::ops::{OpResult, RequestError};
use serde_json::value::RawValue;

/// `tariff.format`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn format(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.format is not implemented",
    ))
}

/// `tariff.tokens`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn tokens(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.tokens is not implemented",
    ))
}
