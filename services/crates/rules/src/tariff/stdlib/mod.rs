//! The standard library: Tariff sources compiled into the module and versioned with it.
//!
//! `use std.<name>` reads these. Every module here must check without a finding and pass its own
//! tests; the crate's tests enforce both. Their doc comments are translated ([`docs`]).

pub mod docs;

/// Every standard library module, by path.
pub const MODULES: &[(&str, &str)] = &[
    ("std.ltl", include_str!("ltl.tariff")),
    ("std.fuel", include_str!("fuel.tariff")),
    ("std.time", include_str!("time.tariff")),
    ("std.accessorials", include_str!("accessorials.tariff")),
];

/// The source of the standard library module at `path`.
#[must_use]
pub fn source(path: &str) -> Option<&'static str> {
    MODULES
        .iter()
        .find(|(module, _)| *module == path)
        .map(|(_, source)| *source)
}
