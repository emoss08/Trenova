//! Everything wrong with a matrix that the engine will not complain about.
//!
//! The engine fails quietly: a gap returns no cell and the lane falls through to whatever comes
//! next, an overlap picks the older cell, and neither says a word. The matrix editor lists these
//! instead:
//!
//! - **empty**: a matrix with no axes, or with axes but no cells;
//! - **duplicate**: two cells at the same coordinate (every axis's key or band), reported once
//!   however often it repeats;
//! - for every banded axis, within each *slice* (the cells that agree on every other axis, so a
//!   zone-by-weight matrix checks each zone's weight breaks on their own):
//!   - **gap** below the lowest band when it starts above zero;
//!   - **unbounded** when nothing reaches upward forever: the highest upper bound any band
//!     reaches is where pricing stops;
//!   - **gap** between bands when the next band starts above everything before it reaches;
//!   - **overlap** when the next band starts below what an earlier band reaches.
//! - for every zip-range axis, within each slice, an **overlap** when two postal ranges of the
//!   same length share a code (compared trimmed, upper-cased, character by character in Unicode
//!   order). Gaps between postal ranges are normal and not reported.
//!
//! Bands are swept in order of their lower bound (a missing lower bound first), comparing each
//! with the furthest reach so far, so a wide band that covers two narrower ones is an overlap
//! with both, never a false gap. Bands are half open, so one ending exactly where the next
//! begins is contiguous. The same finding in several slices is reported once, listing them.
//!
//! The messages are English templates the host translates. Numbers are written with English
//! thousands grouping, and an overlap's bands are written as English band labels and also given
//! as bounds, because a message argument cannot itself be a message: a host that translates
//! rebuilds the labels from `bands` with the band label templates here.

use crate::matrix::input::Cell;
use crate::matrix::input::Dimension;
use crate::matrix::input::MatchMode;
use crate::report::fixed::group_thousands;
use rust_decimal::Decimal;
use std::cmp::Ordering;
use std::collections::HashMap;
use std::collections::HashSet;
use trenova_module_kit::message;
use trenova_module_kit::message::Arg;
use trenova_module_kit::message::Message;

/// What kind of problem an issue is.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum IssueKind {
    /// Nothing can be looked up.
    Empty,
    /// Two cells share a coordinate.
    Duplicate,
    /// A range of quantities finds no band.
    Gap,
    /// Two bands claim the same quantity.
    Overlap,
    /// The top band has a ceiling.
    Unbounded,
}

impl IssueKind {
    /// The wire name.
    #[must_use]
    pub const fn name(self) -> &'static str {
        match self {
            Self::Empty => "empty",
            Self::Duplicate => "duplicate",
            Self::Gap => "gap",
            Self::Overlap => "overlap",
            Self::Unbounded => "unbounded",
        }
    }
}

/// A band's bounds.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct Bounds {
    /// The lower bound, inclusive.
    pub min: Option<Decimal>,
    /// The upper bound, exclusive.
    pub max: Option<Decimal>,
}

/// One problem.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Issue {
    /// What kind.
    pub kind: IssueKind,
    /// The banded axis's position, for a band issue.
    pub position: Option<i64>,
    /// What to tell the author.
    pub message: Message,
    /// The two bands of an overlap.
    pub bands: Vec<Bounds>,
    /// The coordinates on the other axes of the slices it occurs in; empty for a matrix with
    /// one axis and for issues that are not about bands.
    pub slices: Vec<String>,
}

/// Writes a number the way the editor shows it: exact, with English thousands grouping.
#[must_use]
pub fn format_number(value: Decimal) -> String {
    group_thousands(&value.normalize().to_string())
}

/// A band's printed name. The upper bound is exclusive, so it reads as where the next band takes
/// over.
#[must_use]
pub fn band_label(bounds: Bounds) -> Message {
    match (bounds.min, bounds.max) {
        (Some(min), Some(max)) => Message::with(
            message!("{0} – under {1}"),
            [Arg::from(format_number(min)), Arg::from(format_number(max))],
        ),
        (Some(min), None) => {
            Message::with(message!("{0} and over"), [Arg::from(format_number(min))])
        }
        (None, Some(max)) => Message::with(message!("under {0}"), [Arg::from(format_number(max))]),
        (None, None) => Message::new(message!("Any")),
    }
}

fn bound_text(value: Option<Decimal>) -> String {
    value
        .map(|value| value.normalize().to_string())
        .unwrap_or_default()
}

/// Where a cell sits along one axis: its key, or its band as `min..max`.
#[must_use]
pub fn bucket_key(cell: &Cell, slot: usize, dimension: &Dimension) -> String {
    match dimension.mode() {
        MatchMode::Exact => return cell.key_at(slot).to_owned(),
        MatchMode::Postal => {
            return format!(
                "{}–{}",
                postal(cell.key_at(slot)),
                postal(cell.key_to_at(slot))
            );
        }
        MatchMode::Range | MatchMode::Other => {}
    }
    let (min, max) = cell.bounds_at(slot);
    format!("{}..{}", bound_text(min), bound_text(max))
}

fn coordinate(cell: &Cell, dimensions: &[(usize, &Dimension)], skip: Option<usize>) -> String {
    dimensions
        .iter()
        .enumerate()
        .filter(|(position, _)| Some(*position) != skip)
        .map(|(_, (slot, dimension))| bucket_key(cell, *slot, dimension))
        .collect::<Vec<_>>()
        .join(" · ")
}

/// Finds every issue.
#[must_use]
pub fn find_issues(dimensions: &[(usize, &Dimension)], cells: &[&Cell]) -> Vec<Issue> {
    if dimensions.is_empty() {
        return vec![global(message!(
            "This matrix has no axes, so nothing can ever look a rate up in it."
        ))];
    }
    if cells.is_empty() {
        return vec![global(message!(
            "This matrix has no rates yet, so every lane pointing at it prices nothing."
        ))];
    }
    let mut issues = duplicates(dimensions, cells);
    for (position, (slot, dimension)) in dimensions.iter().enumerate() {
        match dimension.mode() {
            MatchMode::Range => {
                issues.extend(band_issues(dimensions, cells, position, *slot, dimension));
            }
            MatchMode::Postal => {
                issues.extend(postal_issues(dimensions, cells, position, *slot, dimension));
            }
            MatchMode::Exact | MatchMode::Other => {}
        }
    }
    issues
}

/// A postal code as a zip-range axis compares it: trimmed and upper-cased.
fn postal(text: &str) -> String {
    text.trim().to_uppercase()
}

/// Overlapping postal ranges on a zip-range axis, within each slice: two ranges of the same
/// length that share a code, compared character by character in Unicode order.
fn postal_issues(
    dimensions: &[(usize, &Dimension)],
    cells: &[&Cell],
    position: usize,
    slot: usize,
    dimension: &Dimension,
) -> Vec<Issue> {
    let mut slice_order: Vec<String> = Vec::new();
    let mut slices: HashMap<String, Vec<(String, String)>> = HashMap::new();
    for cell in cells {
        let at = coordinate(cell, dimensions, Some(position));
        let range = (postal(cell.key_at(slot)), postal(cell.key_to_at(slot)));
        if range.0.is_empty() || range.0.chars().count() != range.1.chars().count() {
            continue;
        }
        let ranges = slices.entry(at.clone()).or_insert_with(|| {
            slice_order.push(at);
            Vec::new()
        });
        if !ranges.contains(&range) {
            ranges.push(range);
        }
    }

    let label = dimension.display_label();
    let multi = dimensions.len() > 1;
    let mut merged: Vec<Issue> = Vec::new();
    for at in &slice_order {
        let Some(ranges) = slices.get_mut(at) else {
            continue;
        };
        ranges.sort_by(|left, right| {
            left.0
                .chars()
                .count()
                .cmp(&right.0.chars().count())
                .then_with(|| left.cmp(right))
        });
        let mut reach: Option<&(String, String)> = None;
        for range in ranges.iter() {
            let overlaps = reach.filter(|earlier| {
                earlier.0.chars().count() == range.0.chars().count() && range.0 <= earlier.1
            });
            if let Some(earlier) = overlaps {
                let message = Message::with(
                    message!(
                        "{0} ranges {1} and {2} overlap, so the same postal code matches both."
                    ),
                    [
                        Arg::from(label.as_str()),
                        Arg::from(format!("{}–{}", earlier.0, earlier.1)),
                        Arg::from(format!("{}–{}", range.0, range.1)),
                    ],
                );
                let known = merged.iter_mut().find(|known| known.message == message);
                if let Some(known) = known {
                    if multi {
                        known.slices.push(at.clone());
                    }
                } else {
                    merged.push(Issue {
                        kind: IssueKind::Overlap,
                        position: Some(dimension.position),
                        message,
                        bands: Vec::new(),
                        slices: if multi { vec![at.clone()] } else { Vec::new() },
                    });
                }
            }
            let further = reach.is_none_or(|earlier| {
                earlier.0.chars().count() != range.0.chars().count() || range.1 > earlier.1
            });
            if further {
                reach = Some(range);
            }
        }
    }
    merged
}

fn global(template: &'static str) -> Issue {
    Issue {
        kind: IssueKind::Empty,
        position: None,
        message: Message::new(template),
        bands: Vec::new(),
        slices: Vec::new(),
    }
}

fn duplicates(dimensions: &[(usize, &Dimension)], cells: &[&Cell]) -> Vec<Issue> {
    let mut seen = HashSet::new();
    let mut reported = HashSet::new();
    let mut issues = Vec::new();
    for cell in cells {
        let at = coordinate(cell, dimensions, None);
        if !seen.insert(at.clone()) && reported.insert(at.clone()) {
            issues.push(Issue {
                kind: IssueKind::Duplicate,
                position: None,
                message: Message::with(
                    message!(
                        "Two rates sit at {0}. The engine will take the one entered first and never mention the other."
                    ),
                    [Arg::from(at)],
                ),
                bands: Vec::new(),
                slices: Vec::new(),
            });
        }
    }
    issues
}

fn compare_min(left: Option<Decimal>, right: Option<Decimal>) -> Ordering {
    match (left, right) {
        (None, None) => Ordering::Equal,
        (None, Some(_)) => Ordering::Less,
        (Some(_), None) => Ordering::Greater,
        (Some(left), Some(right)) => left.cmp(&right),
    }
}

/// Whether reach `left` (an upper bound, `None` for unbounded) is beyond `right`.
fn reaches_further(left: Option<Decimal>, right: Option<Decimal>) -> bool {
    match (left, right) {
        (None, Some(_)) => true,
        (Some(left), Some(right)) => left > right,
        (_, None) => false,
    }
}

fn band_issues(
    dimensions: &[(usize, &Dimension)],
    cells: &[&Cell],
    position: usize,
    slot: usize,
    dimension: &Dimension,
) -> Vec<Issue> {
    let mut slice_order: Vec<String> = Vec::new();
    let mut slices: HashMap<String, Vec<Bounds>> = HashMap::new();
    for cell in cells {
        let at = coordinate(cell, dimensions, Some(position));
        let (min, max) = cell.bounds_at(slot);
        let bounds = Bounds { min, max };
        let bands = slices.entry(at.clone()).or_insert_with(|| {
            slice_order.push(at);
            Vec::new()
        });
        if !bands.contains(&bounds) {
            bands.push(bounds);
        }
    }

    let label = dimension.display_label();
    let axis = dimension.position;
    let mut merged: Vec<Issue> = Vec::new();
    let multi = dimensions.len() > 1;
    for at in &slice_order {
        let Some(bands) = slices.get_mut(at) else {
            continue;
        };
        bands.sort_by(|left, right| compare_min(left.min, right.min));
        for issue in sweep(bands, &label, axis) {
            let known = merged
                .iter_mut()
                .find(|known| known.kind == issue.kind && known.message == issue.message);
            if let Some(known) = known {
                if multi {
                    known.slices.push(at.clone());
                }
            } else {
                let mut issue = issue;
                if multi {
                    issue.slices.push(at.clone());
                }
                merged.push(issue);
            }
        }
    }
    merged
}

fn sweep(bands: &[Bounds], label: &str, axis: i64) -> Vec<Issue> {
    let issue = |kind, message, bands| Issue {
        kind,
        position: Some(axis),
        message,
        bands,
        slices: Vec::new(),
    };
    let mut issues = Vec::new();
    let Some(lowest) = bands.first() else {
        return issues;
    };
    if let Some(min) = lowest.min.filter(|min| *min > Decimal::ZERO) {
        issues.push(issue(
            IssueKind::Gap,
            Message::with(
                message!("{0} has no band below {1}, so anything smaller finds no rate."),
                [Arg::from(label), Arg::from(format_number(min))],
            ),
            Vec::new(),
        ));
    }

    let mut between = Vec::new();
    let mut reach = *lowest;
    for band in &bands[1..] {
        match (reach.max, band.min) {
            (Some(end), Some(start)) if start > end => between.push(issue(
                IssueKind::Gap,
                Message::with(
                    message!("{0} jumps from {1} to {2} with nothing in between."),
                    [
                        Arg::from(label),
                        Arg::from(format_number(end)),
                        Arg::from(format_number(start)),
                    ],
                ),
                Vec::new(),
            )),
            (Some(end), Some(start)) if start == end => {}
            _ => between.push(issue(
                IssueKind::Overlap,
                Message::with(
                    message!("{0} bands {1} and {2} overlap, so the same shipment matches both."),
                    [
                        Arg::from(label),
                        Arg::from(band_label(reach)),
                        Arg::from(band_label(*band)),
                    ],
                ),
                vec![reach, *band],
            )),
        }
        if reaches_further(band.max, reach.max) {
            reach = *band;
        }
    }

    if let Some(end) = reach.max {
        issues.push(issue(
            IssueKind::Unbounded,
            Message::with(
                message!(
                    "{0} stops at {1}. Anything at or above that finds no rate — give the top band no upper bound to catch it."
                ),
                [Arg::from(label), Arg::from(format_number(end))],
            ),
            Vec::new(),
        ));
    }
    issues.extend(between);
    issues
}
