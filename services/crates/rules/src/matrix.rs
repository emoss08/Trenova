//! Rate matrix bands: cell selection and coverage.
//!
//! A rate matrix is a tariff sheet: up to four axes (an origin zone, a destination zone, a
//! weight break, a class), each looked up by exact key or by which band a quantity falls in, and
//! one priced cell per coordinate. The engine picks the cell that prices a shipment; the editor
//! draws the sheet and lists its holes before a customer finds them.
//!
//! - [`band`]: the half-open band rule every banded lookup uses.
//! - [`select`]: picking the one cell that prices a lookup.
//! - [`coverage`]: gaps, overlaps, duplicates, a capped top band and empty matrices.
//! - [`input`]: dimensions and cells as the hosts send them.

pub mod band;
pub mod coverage;
pub mod input;
pub mod ops;
pub mod select;
