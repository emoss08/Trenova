//! Rate matrix bands and coverage.
//!
//! A rate matrix is a tariff sheet: up to four axes (an origin zone, a destination zone, a
//! weight break, a class), each looked up by exact key or by which band a quantity falls in, and
//! one priced cell per coordinate. A tariff reads a matrix as a rate table
//! (`crate::tariff::eval::tables`); the editor draws the sheet and lists its holes before a
//! customer finds them.
//!
//! - [`band`]: the half-open band rule every banded lookup uses, a tariff's table lookups
//!   included.
//! - [`coverage`]: gaps, overlaps, duplicates, a capped top band and empty matrices.
//! - [`input`]: dimensions and cells as the hosts send them.

pub mod band;
pub mod coverage;
pub mod input;
pub mod ops;
