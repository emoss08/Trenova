//! A rate agreement rule's lane: its keys and its specificity (`rateagreement.RateAgreementRule`
//! `ApplyLaneKey`, `ReverseLaneKey`, `ComputeSpecificity`).
//!
//! Specificity decides which of two matching rules prices a shipment once priority has not.
//! Geography is scaled by 1024, clear of every condition term (which sum to 1023), so one step of
//! geographic granularity is never outvoted by conditions. The two ends weigh the same. Each
//! condition counts once when the rule names it: a non-empty list, a bound that is set, a flag
//! that is on; naming a tractor type, a trailer type and an equipment class is one statement
//! about equipment.
//!
//! The rule is read in the GraphQL and Go JSON shape, so either host passes its rows as they
//! are; fields the rules do not read are ignored.

use crate::lane::scope::RADIUS_LANE_KEY;
use crate::lane::scope::Scope;
use crate::lane::scope::lane_key;
use crate::lane::scope::weight_of;
use serde::Deserialize;
use serde::Deserializer;
use serde::de::IgnoredAny;
use serde_json::value::RawValue;

/// Lifts the lane term clear of every condition term.
pub const GEOGRAPHY_SCALE: i64 = 1024;

/// A rule that names commodities.
pub const WEIGHT_COMMODITY: i64 = 512;
/// A rule that names freight classes.
pub const WEIGHT_FREIGHT_CLASS: i64 = 256;
/// A rule that names tractor types, trailer types or equipment classes.
pub const WEIGHT_EQUIPMENT_TYPE: i64 = 128;
/// A rule that names service types.
pub const WEIGHT_SERVICE_TYPE: i64 = 64;
/// A rule that names shipment types.
pub const WEIGHT_SHIPMENT_TYPE: i64 = 32;
/// A rule with a weight range.
pub const WEIGHT_WEIGHT_RANGE: i64 = 16;
/// A rule with a distance range.
pub const WEIGHT_DISTANCE_RANGE: i64 = 8;
/// A rule that names service models.
pub const WEIGHT_SERVICE_MODEL: i64 = 4;
/// A hazmat-only rule.
pub const WEIGHT_HAZMAT: i64 = 2;
/// A temperature-controlled-only rule.
pub const WEIGHT_TEMP_CONTROL: i64 = 1;

/// Every condition weight summed.
pub const MAX_CONDITION_SCORE: i64 = WEIGHT_COMMODITY
    + WEIGHT_FREIGHT_CLASS
    + WEIGHT_EQUIPMENT_TYPE
    + WEIGHT_SERVICE_TYPE
    + WEIGHT_SHIPMENT_TYPE
    + WEIGHT_WEIGHT_RANGE
    + WEIGHT_DISTANCE_RANGE
    + WEIGHT_SERVICE_MODEL
    + WEIGHT_HAZMAT
    + WEIGHT_TEMP_CONTROL;

// Conditions summed must stay below one step of geography, or a broad lane naming every condition
// could outrank a narrower one.
const _: () = assert!(MAX_CONDITION_SCORE < GEOGRAPHY_SCALE);

/// Counts a JSON list's entries, reading `null` as empty.
fn list_len<'de, D: Deserializer<'de>>(deserializer: D) -> Result<usize, D::Error> {
    Ok(Option::<Vec<IgnoredAny>>::deserialize(deserializer)?.map_or(0, |list| list.len()))
}

/// Whether a bound is set: `null`, a missing field and a blank string (a form field not filled
/// in) are unset.
fn is_set<'de, D: Deserializer<'de>>(deserializer: D) -> Result<bool, D::Error> {
    let Some(raw) = Option::<&RawValue>::deserialize(deserializer)? else {
        return Ok(false);
    };
    let text = raw.get();
    if text == "null" {
        return Ok(false);
    }
    if text.starts_with('"') {
        let inner = serde_json::from_str::<String>(text).map_err(serde::de::Error::custom)?;
        return Ok(!inner.trim().is_empty());
    }
    Ok(true)
}

/// The fields of a rate agreement rule the lane rules read.
#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize)]
#[serde(default, rename_all = "camelCase")]
pub struct RuleInput {
    /// `Active` or `Inactive`; empty is `Active`, the default a saved rule takes.
    pub status: Option<String>,
    /// `Directional` or `Bidirectional`.
    pub direction: Option<String>,
    /// The origin scope type.
    pub origin_scope_type: Option<String>,
    /// The origin scope value.
    pub origin_scope_value: Option<String>,
    /// The origin city, for a `CityState` origin.
    pub origin_city: Option<String>,
    /// The destination scope type.
    pub destination_scope_type: Option<String>,
    /// The destination scope value.
    pub destination_scope_value: Option<String>,
    /// The destination city, for a `CityState` destination.
    pub destination_city: Option<String>,
    /// Explicit priority; higher wins before specificity is consulted.
    pub priority: Option<i64>,
    /// When the rule takes effect, in epoch seconds.
    pub effective_from: Option<i64>,
    /// When it stops, exclusive; `None` is open-ended.
    pub effective_to: Option<i64>,
    /// Commodity IDs.
    #[serde(deserialize_with = "list_len")]
    pub commodity_ids: usize,
    /// Freight classes.
    #[serde(deserialize_with = "list_len")]
    pub freight_classes: usize,
    /// Tractor type IDs.
    #[serde(deserialize_with = "list_len")]
    pub tractor_type_ids: usize,
    /// Trailer type IDs.
    #[serde(deserialize_with = "list_len")]
    pub trailer_type_ids: usize,
    /// Equipment classes.
    #[serde(deserialize_with = "list_len")]
    pub equipment_classes: usize,
    /// Service type IDs.
    #[serde(deserialize_with = "list_len")]
    pub service_type_ids: usize,
    /// Shipment type IDs.
    #[serde(deserialize_with = "list_len")]
    pub shipment_type_ids: usize,
    /// Service models.
    #[serde(deserialize_with = "list_len")]
    pub service_models: usize,
    /// The weight range's lower bound.
    #[serde(deserialize_with = "is_set")]
    pub min_weight: bool,
    /// The weight range's upper bound.
    #[serde(deserialize_with = "is_set")]
    pub max_weight: bool,
    /// The distance range's lower bound.
    #[serde(deserialize_with = "is_set")]
    pub min_distance: bool,
    /// The distance range's upper bound.
    #[serde(deserialize_with = "is_set")]
    pub max_distance: bool,
    /// Whether the rule applies only to hazmat.
    #[serde(deserialize_with = "nullable_flag")]
    pub hazmat_only: bool,
    /// Whether the rule applies only to temperature-controlled freight.
    #[serde(deserialize_with = "nullable_flag")]
    pub temp_control_only: bool,
}

fn nullable_flag<'de, D: Deserializer<'de>>(deserializer: D) -> Result<bool, D::Error> {
    Ok(Option::<bool>::deserialize(deserializer)?.unwrap_or_default())
}

/// A rule's lane keys.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LaneKeys {
    /// The origin end's key, or `None` when it cannot be keyed.
    pub origin: Option<String>,
    /// The destination end's key.
    pub destination: Option<String>,
    /// `origin>destination`, or `None` when either end cannot be keyed.
    pub key: Option<String>,
    /// What the rule stores: the key, or `RADIUS` for a lane matched geospatially.
    pub stored: String,
    /// The key with the ends swapped, for a bidirectional rule.
    pub reverse: Option<String>,
}

impl RuleInput {
    /// The origin end.
    #[must_use]
    pub fn origin(&self) -> Scope<'_> {
        Scope {
            scope_type: self.origin_scope_type.as_deref(),
            value: self.origin_scope_value.as_deref().unwrap_or_default(),
            city: self.origin_city.as_deref().unwrap_or_default(),
        }
    }

    /// The destination end.
    #[must_use]
    pub fn destination(&self) -> Scope<'_> {
        Scope {
            scope_type: self.destination_scope_type.as_deref(),
            value: self.destination_scope_value.as_deref().unwrap_or_default(),
            city: self.destination_city.as_deref().unwrap_or_default(),
        }
    }

    /// The rule's lane keys.
    #[must_use]
    pub fn lane_keys(&self) -> LaneKeys {
        let origin = self.origin().key();
        let destination = self.destination().key();
        let (key, reverse) = match (&origin, &destination) {
            (Some(origin), Some(destination)) => (
                Some(lane_key(origin, destination)),
                (self.direction.as_deref() == Some("Bidirectional"))
                    .then(|| lane_key(destination, origin)),
            ),
            _ => (None, None),
        };
        LaneKeys {
            stored: key.clone().unwrap_or_else(|| String::from(RADIUS_LANE_KEY)),
            origin,
            destination,
            key,
            reverse,
        }
    }

    /// How narrowly the rule is written.
    #[must_use]
    pub fn specificity(&self) -> i64 {
        let geography = (weight_of(self.origin_scope_type.as_deref())
            + weight_of(self.destination_scope_type.as_deref()))
            * GEOGRAPHY_SCALE;
        let conditions = [
            (self.commodity_ids > 0, WEIGHT_COMMODITY),
            (self.freight_classes > 0, WEIGHT_FREIGHT_CLASS),
            (
                self.tractor_type_ids > 0
                    || self.trailer_type_ids > 0
                    || self.equipment_classes > 0,
                WEIGHT_EQUIPMENT_TYPE,
            ),
            (self.service_type_ids > 0, WEIGHT_SERVICE_TYPE),
            (self.shipment_type_ids > 0, WEIGHT_SHIPMENT_TYPE),
            (self.min_weight || self.max_weight, WEIGHT_WEIGHT_RANGE),
            (
                self.min_distance || self.max_distance,
                WEIGHT_DISTANCE_RANGE,
            ),
            (self.service_models > 0, WEIGHT_SERVICE_MODEL),
            (self.hazmat_only, WEIGHT_HAZMAT),
            (self.temp_control_only, WEIGHT_TEMP_CONTROL),
        ];
        geography
            + conditions
                .iter()
                .filter(|(named, _)| *named)
                .map(|(_, weight)| weight)
                .sum::<i64>()
    }

    /// Whether the rule is active (an empty status is the `Active` default).
    #[must_use]
    pub fn is_active(&self) -> bool {
        matches!(self.status.as_deref(), None | Some("" | "Active"))
    }
}
