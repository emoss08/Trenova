//! An input's `where` constraint, folded to wire values.
//!
//! The checker folds the constraint once ([`crate::tariff::check::InputInfo::constraint`]); the
//! contract-value validation, the evaluator's input resolution, the test runner's `given`s and a
//! property's `vary input.x` all read the same folded form, so they agree on every bound.

use serde_json::Value as Json;
use serde_json::json;

/// What values an input accepts, beyond its type.
#[derive(Debug, Clone, PartialEq)]
pub enum Constraint {
    /// `lo..hi`, `lo..=hi`, `lo..`, `..=hi`: the minimum is inclusive, the maximum inclusive
    /// only when written `..=`. An open end is `None`.
    Range {
        /// The smallest value accepted, as a wire value.
        min: Option<Json>,
        /// The bound above, as a wire value.
        max: Option<Json>,
        /// The maximum is accepted too.
        max_inclusive: bool,
    },
    /// `[a, b]`: the value must equal one of these wire values.
    OneOf(Vec<Json>),
}

impl Constraint {
    /// The constraint as `tariff.check` reports it.
    #[must_use]
    pub fn to_json(&self) -> Json {
        match self {
            Self::Range {
                min,
                max,
                max_inclusive,
            } => json!({"range": {"min": min, "max": max, "maxInclusive": max_inclusive}}),
            Self::OneOf(values) => json!({ "oneOf": values }),
        }
    }
}
