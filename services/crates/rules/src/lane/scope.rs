//! Scopes: the ways one end of a lane names a place, their weights and their keys
//! (`rategeo`).
//!
//! A rule stores one key per lane, `origin>destination`, and a shipment produces every key a
//! rule could have stored to reach it, so matching is one set-membership probe on an index. The
//! two sides must build keys identically or a rule silently stops matching, which is why both
//! live here. The text rules are Go's byte for byte: postal codes keep the part before a dash
//! (not at the start) and are trimmed and upper-cased, cities are folded to single spaces and
//! upper-cased, countries are trimmed and upper-cased. Upper-casing is Unicode's simple
//! (one-to-one) mapping, which is what Go's `strings.ToUpper` applies.

use crate::allocation::input::nullable;
use serde::Deserialize;

/// What an unconditional end stores, with no prefix.
pub const KEY_ANY: &str = "ANY";

/// Joins an origin key to a destination key.
pub const LANE_KEY_SEPARATOR: char = '>';

/// What a lane matched by radius stores: no shipment generates it.
pub const RADIUS_LANE_KEY: &str = "RADIUS";

const ZIP3_LENGTH: usize = 3;

/// The granularity a scope names a place at.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum ScopeType {
    /// Anywhere.
    Any,
    /// A country, by ISO code.
    Country,
    /// A state, by ID.
    State,
    /// A rate zone, by ID.
    Zone,
    /// A circle around a point; matched geospatially, never by key.
    Radius,
    /// A city within a state.
    CityState,
    /// A three-character postal prefix.
    Zip3,
    /// A postal code.
    Zip5,
    /// One location, by ID.
    Location,
}

impl ScopeType {
    /// The type a wire name stands for. An empty or missing name is `Any`, the default a rule
    /// takes before it is keyed; an unknown one is `None`.
    #[must_use]
    pub fn parse(name: Option<&str>) -> Option<Self> {
        Some(match name.unwrap_or_default() {
            "" | "Any" => Self::Any,
            "Country" => Self::Country,
            "State" => Self::State,
            "Zone" => Self::Zone,
            "Radius" => Self::Radius,
            "CityState" => Self::CityState,
            "Zip3" => Self::Zip3,
            "Zip5" => Self::Zip5,
            "Location" => Self::Location,
            _ => return None,
        })
    }

    /// The wire name.
    #[must_use]
    pub const fn name(self) -> &'static str {
        match self {
            Self::Any => "Any",
            Self::Country => "Country",
            Self::State => "State",
            Self::Zone => "Zone",
            Self::Radius => "Radius",
            Self::CityState => "CityState",
            Self::Zip3 => "Zip3",
            Self::Zip5 => "Zip5",
            Self::Location => "Location",
        }
    }

    /// How narrowly the scope reads. Fixed constants, so a rule's rank never moves because a
    /// zone gained a postal code; radius sits between zone and city on purpose.
    #[must_use]
    pub const fn weight(self) -> i64 {
        match self {
            Self::Any => 0,
            Self::Country => 300,
            Self::State => 500,
            Self::Zone => 600,
            Self::Radius => 650,
            Self::CityState => 700,
            Self::Zip3 => 800,
            Self::Zip5 => 900,
            Self::Location => 1000,
        }
    }
}

/// The weight of a wire scope type name; an unknown name weighs nothing, as in Go.
#[must_use]
pub fn weight_of(name: Option<&str>) -> i64 {
    ScopeType::parse(name).map_or(0, ScopeType::weight)
}

/// One end of a lane.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Scope<'a> {
    /// The scope type's wire name.
    pub scope_type: Option<&'a str>,
    /// The location, postal code, zone, state or country it names.
    pub value: &'a str,
    /// The city, for a `CityState` scope (whose value is the state).
    pub city: &'a str,
}

impl Scope<'_> {
    /// The key a rule stores for this end, or `None` when the end cannot be keyed: a radius, an
    /// unknown type, a missing value, or a postal prefix shorter than three characters.
    #[must_use]
    pub fn key(&self) -> Option<String> {
        let value = self.value;
        let present = !value.is_empty();
        match ScopeType::parse(self.scope_type)? {
            ScopeType::Any => Some(String::from(KEY_ANY)),
            ScopeType::Location => present.then(|| format!("LOC:{value}")),
            ScopeType::Zip5 => present.then(|| format!("Z5:{}", normalize_postal(value))),
            ScopeType::Zip3 => {
                let postal = normalize_postal(value);
                (postal.len() >= ZIP3_LENGTH).then(|| format!("Z3:{}", zip3(&postal)))
            }
            ScopeType::CityState => {
                (present && !self.city.is_empty()).then(|| city_state_key(value, self.city))
            }
            ScopeType::Zone => present.then(|| format!("ZN:{value}")),
            ScopeType::State => present.then(|| format!("ST:{value}")),
            ScopeType::Country => present.then(|| format!("CT:{}", normalize_country(value))),
            ScopeType::Radius => None,
        }
    }
}

/// A city within a state: `CS:<state>|<CITY>`.
#[must_use]
pub fn city_state_key(state: &str, city: &str) -> String {
    format!("CS:{state}|{}", normalize_city(city))
}

/// Everything known about one end of a shipment's lane (`rategeo.Place`).
#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize)]
#[serde(default, rename_all = "camelCase")]
pub struct Place {
    /// The stop's location.
    #[serde(deserialize_with = "nullable")]
    pub location_id: String,
    /// Its postal code.
    #[serde(deserialize_with = "nullable")]
    pub postal_code: String,
    /// Its city.
    #[serde(deserialize_with = "nullable")]
    pub city: String,
    /// Its state.
    #[serde(deserialize_with = "nullable")]
    pub state_id: String,
    /// Its country's ISO 3166 alpha-3 code.
    #[serde(rename = "countryIso3")]
    #[serde(deserialize_with = "nullable")]
    pub country_iso3: String,
    /// The rate zones it belongs to.
    #[serde(deserialize_with = "nullable")]
    pub zone_ids: Vec<String>,
}

impl Place {
    /// Every key a rule could have stored to match this place, most specific first, always
    /// ending with `ANY` (`MatchKeys`).
    #[must_use]
    pub fn match_keys(&self) -> Vec<String> {
        let mut keys = Vec::with_capacity(7 + self.zone_ids.len());
        if !self.location_id.is_empty() {
            keys.push(format!("LOC:{}", self.location_id));
        }
        let postal = normalize_postal(&self.postal_code);
        if !postal.is_empty() {
            keys.push(format!("Z5:{postal}"));
            if postal.len() >= ZIP3_LENGTH {
                keys.push(format!("Z3:{}", zip3(&postal)));
            }
        }
        if !self.state_id.is_empty() && !self.city.is_empty() {
            keys.push(city_state_key(&self.state_id, &self.city));
        }
        for zone in self.zone_ids.iter().filter(|zone| !zone.is_empty()) {
            keys.push(format!("ZN:{zone}"));
        }
        if !self.state_id.is_empty() {
            keys.push(format!("ST:{}", self.state_id));
        }
        if !self.country_iso3.is_empty() {
            keys.push(format!("CT:{}", normalize_country(&self.country_iso3)));
        }
        keys.push(String::from(KEY_ANY));
        keys
    }
}

/// Joins two end keys into a lane key.
#[must_use]
pub fn lane_key(origin: &str, destination: &str) -> String {
    format!("{origin}{LANE_KEY_SEPARATOR}{destination}")
}

/// Every lane key that could match an origin and a destination: the cross product, origin
/// major.
#[must_use]
pub fn lane_key_candidates(origins: &[String], destinations: &[String]) -> Vec<String> {
    origins
        .iter()
        .flat_map(|origin| {
            destinations
                .iter()
                .map(move |destination| lane_key(origin, destination))
        })
        .collect()
}

/// Upper-cases with Unicode's one-to-one mapping, keeping a character whose upper case is more
/// than one character, as Go's `strings.ToUpper` does.
#[must_use]
pub fn simple_upper(text: &str) -> String {
    text.chars()
        .map(|character| {
            let mut upper = character.to_uppercase();
            match (upper.next(), upper.next()) {
                (Some(single), None) => single,
                _ => character,
            }
        })
        .collect()
}

/// A postal code as rates key it: trimmed, the part before a dash (unless the dash leads), and
/// upper-cased.
#[must_use]
pub fn normalize_postal(value: &str) -> String {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        return String::new();
    }
    let leading = match trimmed.find('-') {
        Some(dash) if dash > 0 => &trimmed[..dash],
        _ => trimmed,
    };
    simple_upper(leading)
}

/// The first three bytes of a normalised postal code, cut back to a character boundary (Go cuts
/// bytes; a postal code is ASCII in practice).
fn zip3(postal: &str) -> &str {
    let mut end = ZIP3_LENGTH.min(postal.len());
    while !postal.is_char_boundary(end) {
        end -= 1;
    }
    &postal[..end]
}

fn normalize_city(value: &str) -> String {
    simple_upper(&value.split_whitespace().collect::<Vec<_>>().join(" "))
}

fn normalize_country(value: &str) -> String {
    simple_upper(value.trim())
}
