//! The EDI module's operation table, the only place that names its handlers.
//!
//! The envelope, request errors and dispatch are `trenova_module_kit::ops`. Operations:
//!
//! | Operation | Does |
//! |---|---|
//! | `edi.version` | the crate version and every operation |
//! | `x12.read` | reads a text: separators, segments, envelopes, loop trees, findings |
//! | `x12.acknowledge` | writes the 997 or 999 for each group of a reading |
//! | `x12.write` | writes segments as text with a profile's separators |
//! | `x12.guides` | lists the base guides |
//! | `x12.guide` | one base guide, as JSON |
//! | `x12.checkGuide` | checks a guide, and a partner's against its base |
//! | `lading.*` | Lading projects: check, format, tokens, the editor, run, test, flow steps, replay, guides, starters and the catalog ([`crate::lading::ops`]) |

use serde_json::value::RawValue;
use trenova_module_kit::ops::{Handler, OpResult, Version, encode};

const OPERATIONS: &[(&str, Handler)] = &[
    ("edi.version", version),
    ("x12.read", crate::x12::ops::read),
    ("x12.acknowledge", crate::x12::ops::acknowledge),
    ("x12.write", crate::x12::ops::write),
    ("x12.guides", crate::x12::ops::guides),
    ("x12.guide", crate::x12::ops::guide),
    ("x12.checkGuide", crate::x12::ops::check_guide),
    ("lading.check", crate::lading::ops::check),
    ("lading.format", crate::lading::ops::format),
    ("lading.tokens", crate::lading::ops::tokens),
    ("lading.complete", crate::lading::ops::complete),
    ("lading.hover", crate::lading::ops::hover),
    ("lading.signature", crate::lading::ops::signature),
    ("lading.definition", crate::lading::ops::definition),
    ("lading.references", crate::lading::ops::references),
    ("lading.rename", crate::lading::ops::rename),
    ("lading.inlays", crate::lading::ops::inlays),
    ("lading.actions", crate::lading::ops::actions),
    ("lading.run", crate::lading::ops::run),
    ("lading.test", crate::lading::ops::test),
    ("lading.flowStep", crate::lading::ops::flow_step),
    ("lading.flowKey", crate::lading::ops::flow_key),
    ("lading.replay", crate::lading::ops::replay),
    ("lading.guide", crate::lading::ops::guide),
    ("lading.starter", crate::lading::ops::starter),
    ("lading.catalog", crate::lading::ops::catalog),
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

    #[test]
    fn version_lists_every_operation() {
        let response = String::from_utf8(dispatch(br#"{"op":"edi.version"}"#)).unwrap_or_default();
        assert!(response.starts_with(r#"{"ok":{"version":""#), "{response}");
        for op in ["x12.read", "x12.acknowledge", "x12.write", "x12.checkGuide"] {
            assert!(response.contains(op), "{response}");
        }
    }
}
