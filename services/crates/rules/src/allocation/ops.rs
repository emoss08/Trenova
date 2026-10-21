//! The `allocation.*` operations.

use crate::ops::OpResult;
use crate::ops::RequestError;
use serde_json::value::RawValue;

/// Splits charges between payers.
pub fn split(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "allocation.split is not built yet",
    ))
}

/// Measures what is left to allocate.
pub fn remainder(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "allocation.remainder is not built yet",
    ))
}

/// Totals additional charges.
pub fn charge_total(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "allocation.chargeTotal is not built yet",
    ))
}

/// Converts stale amount splits to percentages.
pub fn rebalance(_: &RawValue) -> OpResult {
    Err(RequestError::invalid_input(
        "allocation.rebalance is not built yet",
    ))
}
