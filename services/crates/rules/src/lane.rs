//! Rate agreement lane keys and specificity.
//!
//! Every rate rule names a scope for its origin and one for its destination: a location, a
//! postal code or prefix, a city, a zone, a state, a country, a radius or anywhere. The rule
//! stores one lane key built from them, a shipment produces every key a rule could have stored
//! to reach it, and the most specific matching rule prices the shipment. The lane editor
//! predicts all of this as somebody types, so it must build the same keys and scores as the
//! engine to the character.
//!
//! - [`scope`]: scope types, weights, key building and text normalisation; a shipment end's
//!   candidate keys.
//! - [`rule`]: a rule's lane keys and its specificity score.
//! - [`parse`]: reading a stored key back into its ends and the record IDs it references.
//! - [`coverage`]: lanes that can never apply.

pub mod coverage;
pub mod ops;
pub mod parse;
pub mod rule;
pub mod scope;
