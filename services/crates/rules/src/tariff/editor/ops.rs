//! The `tariff.*` operations this area answers.

use crate::ops::{OpResult, RequestError};
use serde_json::value::RawValue;

/// `tariff.complete`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn complete(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.complete is not implemented",
    ))
}

/// `tariff.hover`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn hover(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.hover is not implemented",
    ))
}

/// `tariff.signature`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn signature(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.signature is not implemented",
    ))
}

/// `tariff.definition`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn definition(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.definition is not implemented",
    ))
}

/// `tariff.rename`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn rename(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.rename is not implemented",
    ))
}

/// `tariff.inlays`.
///
/// # Errors
///
/// Always, until the operation is implemented.
pub fn inlays(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "tariff.inlays is not implemented",
    ))
}
