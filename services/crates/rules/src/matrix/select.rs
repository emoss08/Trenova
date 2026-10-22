//! Picking the one cell that prices a lookup (`ratematrix.SelectCell`).
//!
//! The candidates come from the database already narrowed by key and bounds; what is left is
//! what SQL cannot express well:
//!
//! 1. Every declared axis needs a value: at least one key on an exact axis, a quantity on a
//!    banded one. The first axis without one (in the order the dimensions are given) fails the
//!    lookup.
//! 2. A cell matches when, on every exact axis, its key is one of the lookup's keys, and on
//!    every banded axis its band contains the quantity (half open).
//! 3. A place can sit in several zones, so each exact axis's keys are ordered most preferred
//!    first. A match's preference is the sum of the positions of the keys it matched; the
//!    lowest preference wins, then the lowest cell ID (IDs sort by creation, so the older cell),
//!    then the earlier cell in the request (for cells the browser has not saved).

use crate::matrix::band::contains;
use crate::matrix::input::Cell;
use crate::matrix::input::Dimension;
use crate::matrix::input::MAX_DIMENSIONS;
use crate::matrix::input::MatchMode;
use rust_decimal::Decimal;

/// The lookup's value for one axis.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct AxisValue {
    /// The candidate keys of an exact axis, most preferred first.
    pub keys: Vec<String>,
    /// The quantity of a banded axis.
    pub quantity: Option<Decimal>,
}

/// The chosen cell.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Selection {
    /// The cell's position in the request.
    pub index: usize,
    /// The summed positions of the matched keys; zero means every first choice matched.
    pub preference: usize,
    /// Per axis, the key matched or the quantity's text; empty on an undeclared axis.
    pub matched_keys: [String; MAX_DIMENSIONS],
}

/// Why no cell was chosen.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SelectError {
    /// The lookup gave no value for the axis at this slot.
    DimensionValueMissing {
        /// The axis's slot.
        slot: usize,
        /// The axis's position in the request's dimensions.
        dimension: usize,
    },
    /// No cell covers the lookup.
    NoMatchingCell,
}

/// Selects a cell. `dimensions` are `(slot, dimension)` pairs; `cells` keep their request
/// positions.
///
/// # Errors
///
/// When an axis has no value or no cell matches.
pub fn select_cell(
    dimensions: &[(usize, &Dimension)],
    cells: &[(usize, &Cell)],
    values: &[AxisValue; MAX_DIMENSIONS],
) -> Result<Selection, SelectError> {
    for (position, (slot, dimension)) in dimensions.iter().enumerate() {
        let value = &values[*slot];
        let missing = match dimension.mode() {
            MatchMode::Exact => value.keys.is_empty(),
            MatchMode::Range => value.quantity.is_none(),
            MatchMode::Other => false,
        };
        if missing {
            return Err(SelectError::DimensionValueMissing {
                slot: *slot,
                dimension: position,
            });
        }
    }

    let mut best: Option<(Selection, Option<&str>)> = None;
    for (index, cell) in cells {
        let Some(selection) = match_cell(dimensions, *index, cell, values) else {
            continue;
        };
        let id = cell.id.as_deref();
        let better = match &best {
            None => true,
            Some((current, current_id)) => {
                (selection.preference, id.unwrap_or_default())
                    < (current.preference, current_id.unwrap_or_default())
            }
        };
        if better {
            best = Some((selection, id));
        }
    }
    best.map(|(selection, _)| selection)
        .ok_or(SelectError::NoMatchingCell)
}

fn match_cell(
    dimensions: &[(usize, &Dimension)],
    index: usize,
    cell: &Cell,
    values: &[AxisValue; MAX_DIMENSIONS],
) -> Option<Selection> {
    let mut selection = Selection {
        index,
        preference: 0,
        matched_keys: Default::default(),
    };
    for (slot, dimension) in dimensions {
        let value = &values[*slot];
        match dimension.mode() {
            MatchMode::Exact => {
                let key = cell.key_at(*slot);
                let rank = value.keys.iter().position(|candidate| candidate == key)?;
                selection.preference += rank;
                key.clone_into(&mut selection.matched_keys[*slot]);
            }
            MatchMode::Range => {
                let quantity = value.quantity?;
                let (min, max) = cell.bounds_at(*slot);
                if !contains(min, max, quantity) {
                    return None;
                }
                selection.matched_keys[*slot] = quantity.normalize().to_string();
            }
            MatchMode::Other => {}
        }
    }
    Some(selection)
}
