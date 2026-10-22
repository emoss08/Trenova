//! Routes one request to the operation it names and wraps the answer.
//!
//! A request is `{"op": "<area>.<name>", "input": {...}}`. The response is `{"ok": <output>}` when
//! the operation ran, whatever it found (a formula with errors is still a successful check), or
//! `{"error": {"code": "...", "detail": "..."}}` when the request itself was unusable: malformed
//! JSON, an unknown operation, or input that does not match the operation's shape. A request
//! error is always a caller bug, never something to show a user, so its detail is plain English
//! for a developer and is not translated.
//!
//! Each area owns its handlers; this table is the only place that names them.

use serde::Deserialize;
use serde::Serialize;
use serde_json::value::RawValue;

/// Why a request could not be run.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct RequestError {
    /// A stable code: `malformed`, `unknown_op` or `invalid_input`.
    pub code: &'static str,
    /// What was wrong, for the developer who sent it.
    pub detail: String,
}

impl RequestError {
    /// The input did not match the operation's shape.
    pub fn invalid_input(detail: impl Into<String>) -> Self {
        Self {
            code: "invalid_input",
            detail: detail.into(),
        }
    }
}

/// What an operation handler returns: its output, already serialised, or a request error.
pub type OpResult = Result<Box<RawValue>, RequestError>;

/// Decodes an operation's input from its raw JSON.
pub fn decode<'a, T: Deserialize<'a>>(input: &'a RawValue) -> Result<T, RequestError> {
    serde_json::from_str(input.get())
        .map_err(|error| RequestError::invalid_input(error.to_string()))
}

/// Serialises an operation's output.
pub fn encode<T: Serialize>(output: &T) -> OpResult {
    serde_json::value::to_raw_value(output).map_err(|error| RequestError {
        code: "invalid_input",
        detail: format!("the output could not be serialised: {error}"),
    })
}

#[derive(Deserialize)]
struct Envelope<'a> {
    op: &'a str,
    #[serde(borrow)]
    input: Option<&'a RawValue>,
}

type Handler = fn(&RawValue) -> OpResult;

/// Every operation the module answers, by name.
const OPERATIONS: &[(&str, Handler)] = &[
    ("rules.version", version),
    ("formula.catalog", crate::formula::ops::catalog),
    ("formula.check", crate::formula::ops::check),
    ("formula.format", crate::formula::ops::format),
    ("formula.signature", crate::formula::ops::signature),
    ("formula.identifier", crate::formula::ops::identifier),
    ("formula.evaluate", crate::formula::eval::ops::evaluate),
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
    ("matrix.select", crate::matrix::ops::select),
    ("matrix.coverage", crate::matrix::ops::coverage),
    ("tariff.check", crate::tariff::check::ops::check),
    ("tariff.evaluate", crate::tariff::eval::ops::evaluate),
    ("tariff.test", crate::tariff::eval::ops::test),
    ("tariff.scaffoldTest", crate::tariff::eval::ops::scaffold_test),
    ("tariff.format", crate::tariff::syntax::ops::format),
    ("tariff.tokens", crate::tariff::syntax::ops::tokens),
    ("tariff.complete", crate::tariff::editor::ops::complete),
    ("tariff.hover", crate::tariff::editor::ops::hover),
    ("tariff.signature", crate::tariff::editor::ops::signature),
    ("tariff.definition", crate::tariff::editor::ops::definition),
    ("tariff.rename", crate::tariff::editor::ops::rename),
    ("tariff.inlays", crate::tariff::editor::ops::inlays),
    ("tariff.catalog", crate::tariff::check::ops::catalog),
];

/// Runs one request and returns the response bytes. Never panics on any input.
#[must_use]
pub fn dispatch(request: &[u8]) -> Vec<u8> {
    let result = run(request);
    let response = match result {
        Ok(output) => format!("{{\"ok\":{}}}", output.get()),
        Err(error) => {
            serde_json::to_string(&ErrorResponse { error: &error }).unwrap_or_else(|_| {
                String::from("{\"error\":{\"code\":\"malformed\",\"detail\":\"\"}}")
            })
        }
    };
    response.into_bytes()
}

#[derive(Serialize)]
struct ErrorResponse<'a> {
    error: &'a RequestError,
}

fn run(request: &[u8]) -> OpResult {
    let text = std::str::from_utf8(request).map_err(|_| RequestError {
        code: "malformed",
        detail: String::from("the request is not UTF-8"),
    })?;
    let envelope: Envelope<'_> = serde_json::from_str(text).map_err(|error| RequestError {
        code: "malformed",
        detail: error.to_string(),
    })?;
    let handler = OPERATIONS
        .iter()
        .find(|(name, _)| *name == envelope.op)
        .map(|(_, handler)| *handler)
        .ok_or_else(|| RequestError {
            code: "unknown_op",
            detail: format!("no operation is named {:?}", envelope.op),
        })?;
    let empty = RawValue::from_string(String::from("{}")).map_err(|error| RequestError {
        code: "malformed",
        detail: error.to_string(),
    })?;
    handler(envelope.input.unwrap_or(&empty))
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct Version {
    version: &'static str,
    operations: Vec<&'static str>,
}

fn version(_: &RawValue) -> OpResult {
    encode(&Version {
        version: env!("CARGO_PKG_VERSION"),
        operations: OPERATIONS.iter().map(|(name, _)| *name).collect(),
    })
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
        assert!(response.contains("\"formula.evaluate\""), "{response}");
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
