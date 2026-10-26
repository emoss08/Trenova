//! Routes one request to the operation it names and wraps the answer.
//!
//! A request is `{"op": "<area>.<name>", "input": {...}}`. The response is `{"ok": <output>}` when
//! the operation ran, whatever it found (a file with errors is still a successful check), or
//! `{"error": {"code": "...", "detail": "..."}}` when the request itself was unusable: malformed
//! JSON, an unknown operation, or input that does not match the operation's shape. A request
//! error is always a caller bug, never something to show a user, so its detail is plain English
//! for a developer and is not translated.
//!
//! Each module owns its operation table and passes it to [`dispatch`]; the table is the only
//! place that names its handlers.

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

/// An operation handler: the raw JSON input in, the serialised output out.
pub type Handler = fn(&RawValue) -> OpResult;

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

/// Runs one request against `operations` and returns the response bytes. Never panics on any
/// input.
#[must_use]
pub fn dispatch(request: &[u8], operations: &[(&str, Handler)]) -> Vec<u8> {
    let result = run(request, operations);
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

fn run(request: &[u8], operations: &[(&str, Handler)]) -> OpResult {
    let text = std::str::from_utf8(request).map_err(|_| RequestError {
        code: "malformed",
        detail: String::from("the request is not UTF-8"),
    })?;
    let envelope: Envelope<'_> = serde_json::from_str(text).map_err(|error| RequestError {
        code: "malformed",
        detail: error.to_string(),
    })?;
    let handler = operations
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

/// What a module's `<area>.version` operation answers: its crate version and every operation
/// it runs.
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Version {
    /// The module crate's version.
    pub version: &'static str,
    /// Every operation in the table, in table order.
    pub operations: Vec<&'static str>,
}

impl Version {
    /// The version answer for a module's crate version and operation table.
    #[must_use]
    pub fn of(version: &'static str, operations: &[(&'static str, Handler)]) -> Self {
        Self {
            version,
            operations: operations.iter().map(|(name, _)| *name).collect(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::{Handler, OpResult, Version, dispatch, encode};
    use serde_json::value::RawValue;

    fn echo(input: &RawValue) -> OpResult {
        encode(&serde_json::from_str::<serde_json::Value>(input.get()).unwrap_or_default())
    }

    fn version(_: &RawValue) -> OpResult {
        encode(&Version::of("1.2.3", TABLE))
    }

    const TABLE: &[(&str, Handler)] = &[("test.version", version), ("test.echo", echo)];

    fn call(request: &str) -> String {
        String::from_utf8(dispatch(request.as_bytes(), TABLE)).unwrap_or_default()
    }

    #[test]
    fn runs_the_named_operation_and_wraps_its_output() {
        assert_eq!(
            call(r#"{"op":"test.echo","input":{"a":1}}"#),
            r#"{"ok":{"a":1}}"#
        );
        assert_eq!(call(r#"{"op":"test.echo"}"#), r#"{"ok":{}}"#);
    }

    #[test]
    fn version_lists_every_operation() {
        assert_eq!(
            call(r#"{"op":"test.version"}"#),
            r#"{"ok":{"version":"1.2.3","operations":["test.version","test.echo"]}}"#
        );
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
        assert!(
            String::from_utf8(dispatch(&[0xff], TABLE))
                .unwrap_or_default()
                .starts_with(r#"{"error":{"code":"malformed""#)
        );
    }
}
