//! The wire shapes of a rate matrix: its dimensions and its cells, as GraphQL and the Go JSON
//! write them.
//!
//! A cell's four key and bound pairs are positional: pair zero belongs to the dimension at
//! position zero. Exact axes fill the key and leave the bounds empty; range axes the reverse;
//! a zip-range axis (`kind` `ZipRange`, `matchMode` `Range`) holds its range's first postal
//! code in the key and its last in `dNKeyTo`.

use crate::allocation::input::nullable;
use crate::allocation::input::optional_decimal;
use crate::ops::RequestError;
use rust_decimal::Decimal;
use serde::Deserialize;

/// How many axes a matrix can declare.
pub const MAX_DIMENSIONS: usize = 4;

/// How an axis is looked up.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MatchMode {
    /// By equality with one of the lookup's keys.
    Exact,
    /// By which band the lookup's quantity falls in.
    Range,
    /// By which inclusive postal-code range the key falls in: a `ZipRange` axis.
    Postal,
    /// Neither; the axis constrains nothing, as in Go.
    Other,
}

/// One axis of a matrix.
#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize)]
#[serde(default, rename_all = "camelCase")]
pub struct Dimension {
    /// The pair of cell columns the axis reads, 0 to 3.
    pub position: i64,
    /// `Exact` or `Range`.
    #[serde(deserialize_with = "nullable")]
    pub match_mode: String,
    /// What the axis varies by (`Zone`, `WeightBreak`, ...).
    #[serde(deserialize_with = "nullable")]
    pub kind: String,
    /// The name the author gave the axis.
    #[serde(deserialize_with = "nullable")]
    pub label: String,
}

impl Dimension {
    /// How the axis is looked up.
    #[must_use]
    pub fn mode(&self) -> MatchMode {
        match self.match_mode.as_str() {
            "Exact" => MatchMode::Exact,
            "Range" if self.kind == crate::tariff::schema::ZIP_RANGE => MatchMode::Postal,
            "Range" => MatchMode::Range,
            _ => MatchMode::Other,
        }
    }

    /// The position as an index into a cell's pairs.
    ///
    /// # Errors
    ///
    /// When the position is outside 0..=3, which the server never stores.
    pub fn slot(&self) -> Result<usize, RequestError> {
        usize::try_from(self.position)
            .ok()
            .filter(|slot| *slot < MAX_DIMENSIONS)
            .ok_or_else(|| {
                RequestError::invalid_input(format!(
                    "a dimension's position must be 0 to 3, not {}",
                    self.position
                ))
            })
    }

    /// What messages call the axis: the label the author typed, or the English name of its
    /// kind. A host that translates passes the translated name as the label.
    #[must_use]
    pub fn display_label(&self) -> String {
        let typed = self.label.trim();
        if !typed.is_empty() {
            return typed.to_owned();
        }
        let name = match self.kind.as_str() {
            "Zone" => "Zone",
            "Zip3" => "Postal prefix",
            "Zip5" => "Postal code",
            "State" => "State",
            "Country" => "Country",
            "WeightBreak" => "Weight break",
            "Distance" => "Distance",
            "PieceCount" => "Pieces",
            "LinearFeet" => "Linear feet",
            "FreightClass" => "Freight class",
            "EquipmentType" => "Equipment",
            "ServiceType" => "Service",
            "Custom" => "Custom",
            "Quantity" => "Quantity",
            "ZipRange" => "Postal range",
            other => other,
        };
        name.to_owned()
    }
}

/// One priced intersection of a matrix's axes.
#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize)]
#[serde(default, rename_all = "camelCase")]
pub struct Cell {
    /// The cell's ID; absent on a cell the browser has not saved.
    pub id: Option<String>,
    /// The exact key on axis 0.
    pub d0_key: Option<String>,
    /// The exact key on axis 1.
    pub d1_key: Option<String>,
    /// The exact key on axis 2.
    pub d2_key: Option<String>,
    /// The exact key on axis 3.
    pub d3_key: Option<String>,
    /// The last postal code of the range on axis 0, when it is a zip-range axis (the first is
    /// `d0_key`).
    pub d0_key_to: Option<String>,
    /// The last postal code of the range on axis 1.
    pub d1_key_to: Option<String>,
    /// The last postal code of the range on axis 2.
    pub d2_key_to: Option<String>,
    /// The last postal code of the range on axis 3.
    pub d3_key_to: Option<String>,
    /// The band's lower bound on axis 0.
    #[serde(deserialize_with = "optional_decimal")]
    pub d0_min: Option<Decimal>,
    /// The band's upper bound on axis 0, exclusive.
    #[serde(deserialize_with = "optional_decimal")]
    pub d0_max: Option<Decimal>,
    /// The band's lower bound on axis 1.
    #[serde(deserialize_with = "optional_decimal")]
    pub d1_min: Option<Decimal>,
    /// The band's upper bound on axis 1, exclusive.
    #[serde(deserialize_with = "optional_decimal")]
    pub d1_max: Option<Decimal>,
    /// The band's lower bound on axis 2.
    #[serde(deserialize_with = "optional_decimal")]
    pub d2_min: Option<Decimal>,
    /// The band's upper bound on axis 2, exclusive.
    #[serde(deserialize_with = "optional_decimal")]
    pub d2_max: Option<Decimal>,
    /// The band's lower bound on axis 3.
    #[serde(deserialize_with = "optional_decimal")]
    pub d3_min: Option<Decimal>,
    /// The band's upper bound on axis 3, exclusive.
    #[serde(deserialize_with = "optional_decimal")]
    pub d3_max: Option<Decimal>,
}

impl Cell {
    /// The exact key on an axis (`KeyAt`); empty when unset.
    #[must_use]
    pub fn key_at(&self, slot: usize) -> &str {
        let key = match slot {
            0 => &self.d0_key,
            1 => &self.d1_key,
            2 => &self.d2_key,
            3 => &self.d3_key,
            _ => &None,
        };
        key.as_deref().unwrap_or_default()
    }

    /// The last postal code of the range on a zip-range axis (`KeyToAt`); empty when unset.
    #[must_use]
    pub fn key_to_at(&self, slot: usize) -> &str {
        let key = match slot {
            0 => &self.d0_key_to,
            1 => &self.d1_key_to,
            2 => &self.d2_key_to,
            3 => &self.d3_key_to,
            _ => &None,
        };
        key.as_deref().unwrap_or_default()
    }

    /// The band on an axis (`BoundsAt`).
    #[must_use]
    pub fn bounds_at(&self, slot: usize) -> (Option<Decimal>, Option<Decimal>) {
        match slot {
            0 => (self.d0_min, self.d0_max),
            1 => (self.d1_min, self.d1_max),
            2 => (self.d2_min, self.d2_max),
            3 => (self.d3_min, self.d3_max),
            _ => (None, None),
        }
    }
}

/// The dimensions of a request, `null` entries skipped, with their slots checked.
///
/// # Errors
///
/// When a position is outside 0..=3.
pub fn dimensions(raw: &[Option<Dimension>]) -> Result<Vec<(usize, &Dimension)>, RequestError> {
    raw.iter()
        .flatten()
        .map(|dimension| dimension.slot().map(|slot| (slot, dimension)))
        .collect()
}
