//! Banded report columns (`reportfmt.Band`).
//!
//! A banded dimension groups a numeric column into ranges, and the query groups each row on its
//! range's lower bound. So the value a banded cell carries is a lower bound, never a number a
//! reader would recognise, and it is shown as the range it stands for: "$0.00 – $1,000.00",
//! "4:00+". A band is either fixed-width (every range the same size, unbounded both ways) or an
//! ascending list of boundaries whose last range is open-ended, and a value below the first
//! boundary is grouped into the first range, as the query does.
//!
//! The bounds are exact decimals, so `0.1 + 0.2` is `0.3` here, where Go and the browser added
//! floats.

use crate::allocation::input::nullable;
use crate::wire::Dec;
use rust_decimal::Decimal;
use serde::Deserialize;
use serde::Serialize;

/// The most boundaries an explicit band may list.
pub const MAX_BAND_EDGES: usize = 24;

/// The ranges a banded column groups into.
#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(default, rename_all = "camelCase")]
pub struct Band {
    /// The width of every range, for a fixed-width band.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<Dec>,
    /// The ascending boundaries, for an explicit band.
    #[serde(skip_serializing_if = "Vec::is_empty")]
    #[serde(deserialize_with = "nullable")]
    pub edges: Vec<Dec>,
}

/// The range a grouped lower bound stands for.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Bounds {
    /// The range's lower bound.
    pub lower: Decimal,
    /// The range's upper bound, or `None` for the open-ended top range.
    pub upper: Option<Decimal>,
}

impl Band {
    fn width(&self) -> Decimal {
        self.width.map(Dec::get).unwrap_or_default()
    }

    /// Whether no band was authored at all (`IsEmpty`). A malformed band is not empty.
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.width().is_zero() && self.edges.is_empty()
    }

    /// Whether the band validates and can be rendered (`IsUsable`).
    #[must_use]
    pub fn is_usable(&self) -> bool {
        if self.is_empty() {
            return false;
        }
        let width = self.width();
        if !width.is_zero() {
            return self.edges.is_empty() && width > Decimal::ZERO;
        }
        self.edges.len() >= 2
            && self.edges.len() <= MAX_BAND_EDGES
            && self.edges.windows(2).all(|pair| pair[1] > pair[0])
    }

    /// The range `value` stands for, on a usable band (`Bounds`).
    #[must_use]
    pub fn bounds(&self, value: Decimal) -> Bounds {
        let width = self.width();
        if width > Decimal::ZERO {
            return Bounds {
                lower: value,
                upper: value.checked_add(width),
            };
        }
        let mut index = 0;
        for (position, edge) in self.edges.iter().enumerate() {
            if value >= edge.get() {
                index = position;
            }
        }
        let Some(lower) = self.edges.get(index) else {
            return Bounds {
                lower: value,
                upper: None,
            };
        };
        Bounds {
            lower: lower.get(),
            upper: self.edges.get(index + 1).map(|edge| edge.get()),
        }
    }
}
