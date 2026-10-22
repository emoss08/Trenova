//! Tone rules: conditional emphasis on numeric cells (`reportfmt.Rule`, `ToneFor`).
//!
//! A rule names a comparison, a threshold and a tone (`positive`, `warning`, `negative`,
//! `neutral`). Rules are read top to bottom and the first match wins. Only numeric cells are
//! toned. A percent column stores a fraction and shows it times 100, so its values are compared
//! on the shown scale: a threshold of 12 means 12%. Comparisons are exact decimals, so a rule on
//! `eq 0.3` matches `0.3`.

use crate::wire::Dec;
use rust_decimal::Decimal;
use serde::Deserialize;
use serde::Serialize;

/// The most rules a column may carry.
pub const MAX_RULES: usize = 8;

/// One tone rule.
#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(default, rename_all = "camelCase")]
pub struct Rule {
    /// `gt`, `gte`, `lt`, `lte`, `eq` or `between` (inclusive).
    pub op: String,
    /// The threshold, or the lower bound of `between`.
    pub value: Dec,
    /// The upper bound of `between`.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub upper: Option<Dec>,
    /// The tone a match earns.
    pub tone: String,
}

impl Rule {
    fn matches(&self, value: Decimal) -> bool {
        let threshold = self.value.get();
        match self.op.as_str() {
            "gt" => value > threshold,
            "gte" => value >= threshold,
            "lt" => value < threshold,
            "lte" => value <= threshold,
            "eq" => value == threshold,
            "between" => {
                value >= threshold && value <= self.upper.map(Dec::get).unwrap_or_default()
            }
            _ => false,
        }
    }
}

/// The tone a numeric value earns, or `None` when no rule matches.
#[must_use]
pub fn tone_for(rules: &[Rule], value: Decimal, percent: bool) -> Option<&str> {
    if rules.is_empty() {
        return None;
    }
    let value = if percent {
        value.checked_mul(Decimal::ONE_HUNDRED)?
    } else {
        value
    };
    rules
        .iter()
        .find(|rule| rule.matches(value))
        .map(|rule| rule.tone.as_str())
        .filter(|tone| !tone.is_empty())
}
