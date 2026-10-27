//! The Studio's language features, computed in the module so the browser has no language logic of
//! its own: completion, hover, signature help, go to definition, references, rename, inlay hints
//! and refactorings.
//!
//! Every operation reads the file being typed, which is often broken: it loads the file with the
//! libraries it imports ([`crate::tariff::syntax::program::load`]), checks it
//! ([`crate::tariff::check::check`]) and answers from what the parser recovered and the checker
//! worked out. No feature resolves a name or types an expression itself: it reads the
//! [`Analysis`](crate::tariff::check::Analysis) (types and expected types per node, symbols,
//! references, resolved callees and variants, inferred binding types) and the method table
//! ([`crate::tariff::check::methods`]).
//!
//! - [`session`]: one loaded and checked file, with the indexes every feature shares (symbols by
//!   declaration, references by position) and the work meter.
//! - [`cursor`]: where a position is in a possibly broken file: the token, the nodes around it,
//!   the kind of body it is in, and the local names visible there.
//! - [`complete`], [`hover`], [`signature`], [`definition`], [`references`], [`rename`],
//!   [`inlays`], [`actions`]: one feature each; [`names`] holds the candidates completion offers
//!   where a value or a statement goes.
//! - [`occurrences`]: what a name names and every place it is written, shared by references and
//!   rename (and the scope resolution of names the checker did not reach).
//! - [`choices`], [`vary`], [`literals`]: the file's `best`/`worst` labels and `chosen` calls,
//!   a property's `vary` names and their types, and literals' types and values, read from the
//!   syntax tree.
//! - [`describe`]: declarations and signatures written as Tariff code, and the one-line docs of
//!   keywords and units.
//! - [`values`]: the last preview's trace, read back for hover and inlay values.
//! - [`ops`]: the `tariff.*` operations.
//!
//! Each operation does one pass over the file beyond the shared load and check, so it answers a
//! 2,000-line tariff in a few milliseconds; the work meter counts it, and the tests hold it to a
//! small fraction of the evaluator's step budget.

pub mod actions;
pub mod choices;
pub mod complete;
pub mod cursor;
pub mod definition;
pub mod describe;
pub mod hover;
pub mod inlays;
pub mod literals;
pub mod names;
pub mod occurrences;
pub mod ops;
pub mod references;
pub mod rename;
pub mod session;
pub mod signature;
pub mod values;
pub mod vary;

#[cfg(test)]
mod tests;
