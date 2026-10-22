//! Wire forms shared by every operation.
//!
//! Decimals are exact on the wire in both directions. An input decimal may be a JSON number or a
//! JSON string (`12.345` or `"12.345"`); either way its text is read as written, never through
//! `f64`. An output decimal is always a JSON string, because the browser's `JSON.parse` would
//! turn a number back into a float.

use rust_decimal::Decimal;
use serde::Deserialize;
use serde::Deserializer;
use serde::Serialize;
use serde::Serializer;
use serde_json::value::RawValue;
use std::str::FromStr;

/// An exact decimal read from a JSON number or string and written as a JSON string.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Default)]
pub struct Dec(pub Decimal);

impl Dec {
    /// The decimal inside.
    #[must_use]
    pub const fn get(self) -> Decimal {
        self.0
    }
}

impl From<Decimal> for Dec {
    fn from(value: Decimal) -> Self {
        Self(value)
    }
}

/// Parses decimal text as JSON writes it: an optional sign, digits, a fraction and an exponent.
///
/// # Errors
///
/// When the text is not a number or does not fit in 28 significant digits.
pub fn parse_decimal(text: &str) -> Result<Decimal, String> {
    let trimmed = text.trim();
    if trimmed.contains(['e', 'E']) {
        Decimal::from_scientific(trimmed)
    } else {
        Decimal::from_str(trimmed)
    }
    .map(|value| value.normalize())
    .map_err(|error| format!("{trimmed:?} is not a decimal number: {error}"))
}

impl<'de> Deserialize<'de> for Dec {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let raw = <&RawValue>::deserialize(deserializer)?;
        let text = raw.get();
        let text = if text.starts_with('"') {
            serde_json::from_str::<&str>(text).map_err(serde::de::Error::custom)?
        } else {
            text
        };
        parse_decimal(text)
            .map(Dec)
            .map_err(serde::de::Error::custom)
    }
}

impl Serialize for Dec {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.collect_str(&self.0.normalize())
    }
}

#[cfg(test)]
mod tests {
    use super::Dec;

    #[test]
    fn reads_numbers_and_strings_exactly() {
        let values: Vec<Dec> =
            serde_json::from_str(r#"[1.005, "2.675", 0.1, "1e-7", 12]"#).unwrap_or_default();
        let text: Vec<String> = values.iter().map(|value| value.0.to_string()).collect();
        assert_eq!(text, ["1.005", "2.675", "0.1", "0.0000001", "12"]);
    }

    #[test]
    fn writes_strings() {
        let json =
            serde_json::to_string(&[Dec("1.50".parse().unwrap_or_default())]).unwrap_or_default();
        assert_eq!(json, r#"["1.5"]"#);
    }
}
