//! Report value formatting: numbers, money, percentages, durations, bands and tones.
//!
//! A report column carries a display contract (style, decimals, grouping, currency, negative
//! style, notation, prefix and suffix, boolean, duration and date styles, empty-value text, tone
//! rules and a band), and every place a report is shown (the builder preview, the result grid,
//! CSV, PDF and the XLSX text fallback) renders its values through that contract. This module is
//! the one renderer, so a column reads the same everywhere and a value is rounded the same way
//! on the server and in the browser.
//!
//! - [`display`]: the contract and how it is resolved from a column's type, hint and overrides.
//! - [`format`]: rendering one value.
//! - [`band`]: banded columns, whose value is a range's lower bound.
//! - [`rules`]: tone rules.
//! - [`fixed`]: exact rounding and division, shared with charge allocation.
//!
//! Dates are not formatted here: a value in a date column comes back as epoch seconds for the
//! host to write with its own time zone support (Go's time layouts, the browser's `Intl`).
//! Excel number formats stay in Go (`reportfmt.ExcelFormat`), since nothing in the browser
//! needs them.

pub mod band;
pub mod display;
pub mod fixed;
pub mod format;
pub mod ops;
pub mod rules;
