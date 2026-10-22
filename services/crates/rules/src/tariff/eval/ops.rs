//! The `tariff.*` operations this area answers.

use crate::ops::{OpResult, RequestError};
use serde_json::value::RawValue;

/// `tariff.evaluate`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn evaluate(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.evaluate is not implemented",
    ))
}

/// `tariff.test`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn test(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.test is not implemented",
    ))
}
