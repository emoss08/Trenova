//! Routes one request to the operation it names and wraps the answer.
//!
//! A request is `{"op": "<area>.<name>", "input": {...}}`. The response is `{"ok": <output>}` when
//! the operation ran, whatever it found (a file with errors is still a successful check), or
//! `{"error": {"code": "...", "detail": "..."}}` when the request itself was unusable: malformed
//! JSON, an unknown operation, or input that does not match the operation's shape. A request
//! error is always a caller bug, never something to show a user, so its detail is plain English
//! for a developer and is not translated.
//!
//! The envelope, request errors and dispatch are `trenova_module_kit::ops`; this module holds
//! the rules module's operation table, the only place that names its handlers.

use serde_json::value::RawValue;
use trenova_module_kit::ops::{Handler, Version};
use trenova_module_kit::ops::{OpResult, encode};

/// Every operation the module answers, by name.
const OPERATIONS: &[(&str, Handler)] = &[
    ("rules.version", version),
    ("allocation.split", crate::allocation::ops::split),
    ("allocation.remainder", crate::allocation::ops::remainder),
    (
        "allocation.chargeTotal",
        crate::allocation::ops::charge_total,
    ),
    ("allocation.rebalance", crate::allocation::ops::rebalance),
    ("report.format", crate::report::ops::format),
    ("report.resolve", crate::report::ops::resolve),
    ("lane.specificity", crate::lane::ops::specificity),
    ("lane.keys", crate::lane::ops::keys),
    ("lane.parse", crate::lane::ops::parse),
    ("lane.coverage", crate::lane::ops::coverage),
    ("matrix.coverage", crate::matrix::ops::coverage),
    ("tariff.check", crate::tariff::check::ops::check),
    ("tariff.evaluate", crate::tariff::eval::ops::evaluate),
    ("tariff.test", crate::tariff::eval::ops::test),
    (
        "tariff.scaffoldTest",
        crate::tariff::eval::ops::scaffold_test,
    ),
    ("tariff.format", crate::tariff::syntax::ops::format),
    ("tariff.tokens", crate::tariff::syntax::ops::tokens),
    ("tariff.setName", crate::tariff::syntax::ops::set_name),
    ("tariff.complete", crate::tariff::editor::ops::complete),
    ("tariff.hover", crate::tariff::editor::ops::hover),
    ("tariff.signature", crate::tariff::editor::ops::signature),
    ("tariff.definition", crate::tariff::editor::ops::definition),
    ("tariff.references", crate::tariff::editor::ops::references),
    ("tariff.actions", crate::tariff::editor::ops::actions),
    ("tariff.rename", crate::tariff::editor::ops::rename),
    ("tariff.inlays", crate::tariff::editor::ops::inlays),
    ("tariff.catalog", crate::tariff::check::ops::catalog),
];

/// Runs one request and returns the response bytes. Never panics on any input.
#[must_use]
pub fn dispatch(request: &[u8]) -> Vec<u8> {
    trenova_module_kit::ops::dispatch(request, OPERATIONS)
}

fn version(_: &RawValue) -> OpResult {
    encode(&Version::of(env!("CARGO_PKG_VERSION"), OPERATIONS))
}

#[cfg(test)]
mod tests {
    use super::dispatch;

    fn call(request: &str) -> String {
        String::from_utf8(dispatch(request.as_bytes())).unwrap_or_default()
    }

    #[test]
    fn version_lists_every_operation() {
        let response = call(r#"{"op":"rules.version"}"#);
        assert!(response.starts_with(r#"{"ok":{"version":""#), "{response}");
        assert!(response.contains("\"tariff.evaluate\""), "{response}");
    }

    #[test]
    fn an_unknown_operation_is_a_request_error() {
        assert_eq!(
            call(r#"{"op":"nope"}"#),
            r#"{"error":{"code":"unknown_op","detail":"no operation is named \"nope\""}}"#
        );
    }

    #[test]
    fn malformed_json_is_a_request_error() {
        assert!(call("{").starts_with(r#"{"error":{"code":"malformed""#));
        assert!(call("").starts_with(r#"{"error":{"code":"malformed""#));
    }
}
