//! The wire shapes the `allocation.*` operations read.
//!
//! The browser sends what a form holds while somebody types, so an empty string where a decimal
//! goes means "not filled in yet", the same as `null` or a missing field; anything else must be
//! a decimal. Field names follow the GraphQL charge allocation and additional charge types, so
//! either host can pass its rows as they are, and a `null` list or string reads as empty, as
//! GraphQL writes an unset one.
//!
//! [`optional_decimal`] and [`nullable`] are the wire readers every area of the crate uses for
//! form and GraphQL input.

use crate::wire::Dec;
use crate::wire::parse_decimal;
use rust_decimal::Decimal;
use serde::Deserialize;
use serde::Deserializer;
use serde_json::value::RawValue;

/// Reads an optional decimal where `null`, a missing field and a blank string all mean absent.
///
/// # Errors
///
/// When the value is neither absent nor a decimal number or string.
pub fn optional_decimal<'de, D: Deserializer<'de>>(
    deserializer: D,
) -> Result<Option<Decimal>, D::Error> {
    let raw = Option::<&RawValue>::deserialize(deserializer)?;
    let Some(raw) = raw else {
        return Ok(None);
    };
    let text = raw.get();
    if text == "null" {
        return Ok(None);
    }
    if text.starts_with('"') {
        let inner = serde_json::from_str::<String>(text).map_err(serde::de::Error::custom)?;
        if inner.trim().is_empty() {
            return Ok(None);
        }
        return parse_decimal(&inner)
            .map(Some)
            .map_err(serde::de::Error::custom);
    }
    parse_decimal(text)
        .map(Some)
        .map_err(serde::de::Error::custom)
}

/// Reads a value where `null` means its default: an empty list or string.
///
/// # Errors
///
/// When the value is neither `null` nor a `T`.
pub fn nullable<'de, D, T>(deserializer: D) -> Result<T, D::Error>
where
    D: Deserializer<'de>,
    T: Deserialize<'de> + Default,
{
    Ok(Option::<T>::deserialize(deserializer)?.unwrap_or_default())
}

/// One allocation row: one payer's share of one charge.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AllocationRow {
    /// The saved row's ID; absent on a row the browser has not saved.
    #[serde(default)]
    pub id: Option<String>,
    /// The payer.
    #[serde(default)]
    pub bill_to_customer_id: Option<String>,
    /// `Percent` or `Amount`.
    #[serde(default)]
    pub method: Option<String>,
    /// The percent of the charge, on a percent row.
    #[serde(default, deserialize_with = "optional_decimal")]
    pub percent: Option<Decimal>,
    /// The amount of the charge, on an amount row.
    #[serde(default, deserialize_with = "optional_decimal")]
    pub amount: Option<Decimal>,
    /// The row's position; the last row by sequence absorbs a percent split's rounding.
    #[serde(default)]
    pub sequence: Option<i64>,
}

impl AllocationRow {
    /// The payer, when one is named.
    #[must_use]
    pub fn payer(&self) -> Option<&str> {
        self.bill_to_customer_id
            .as_deref()
            .filter(|payer| !payer.is_empty())
    }
}

/// One charge: its total, given or computed from its accessorial line, and its allocations.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ChargeInput {
    /// `Freight`, `Accessorial` or `OrderCharge`; decides which payer subtotal a share joins.
    #[serde(default)]
    pub kind: Option<String>,
    /// The charge total, when the caller already knows it (freight, an order charge).
    #[serde(default, deserialize_with = "optional_decimal")]
    pub total: Option<Decimal>,
    /// The accessorial method (`Flat`, `PerUnit`, `Percentage`) when the total is computed.
    #[serde(default)]
    pub method: Option<String>,
    /// The accessorial amount: a price, a price per unit, or a percent of the base.
    #[serde(default, deserialize_with = "optional_decimal")]
    pub amount: Option<Decimal>,
    /// The accessorial unit count.
    #[serde(default)]
    pub unit: Option<i64>,
    /// The rows splitting this charge; `null` entries are skipped.
    #[serde(default, deserialize_with = "nullable")]
    pub allocations: Vec<Option<AllocationRow>>,
}

impl ChargeInput {
    /// The allocation rows with their positions in the request, skipping `null` entries.
    pub fn rows(&self) -> impl Iterator<Item = (usize, &AllocationRow)> {
        self.allocations
            .iter()
            .enumerate()
            .filter_map(|(index, row)| row.as_ref().map(|row| (index, row)))
    }
}

/// A batch of charges sharing one percentage base (a shipment's freight).
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ChargesInput {
    /// What a `Percentage` charge is a percent of; zero when absent.
    #[serde(default, deserialize_with = "optional_decimal")]
    pub base: Option<Decimal>,
    /// The charges.
    #[serde(default, deserialize_with = "nullable")]
    pub charges: Vec<ChargeInput>,
}

/// The input of `allocation.split`.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SplitInput {
    /// The payer a charge with no allocations goes to: the shipment's bill-to, or its customer.
    #[serde(default, deserialize_with = "nullable")]
    pub default_payer_id: String,
    /// What a `Percentage` charge is a percent of; zero when absent.
    #[serde(default, deserialize_with = "optional_decimal")]
    pub base: Option<Decimal>,
    /// The charges, freight first by convention.
    #[serde(default, deserialize_with = "nullable")]
    pub charges: Vec<ChargeInput>,
}

/// An output decimal.
#[must_use]
pub fn dec(value: Decimal) -> Dec {
    Dec(value)
}
