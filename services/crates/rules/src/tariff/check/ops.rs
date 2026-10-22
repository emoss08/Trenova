//! The `tariff.*` operations this area answers.

use crate::ops::{OpResult, RequestError};
use serde_json::value::RawValue;

/// `tariff.check`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn check(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.check is not implemented",
    ))
}

/// `tariff.catalog`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn catalog(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.catalog is not implemented",
    ))
}
