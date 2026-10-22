//! Charge totals and their split between payers, to the cent.
//!
//! A shipment's freight and accessorials, or an order's charges, can be billed to more than one
//! customer. Each charge is divided by its allocation rows, by percent or by amount, and every
//! payer's invoice is the sum of its shares. The server bills with these rules and the shipment
//! form previews the same invoices while somebody types, so both call this module instead of
//! keeping their own arithmetic.
//!
//! - [`charge`]: what one accessorial line comes to.
//! - [`split`]: dividing charges between payers and summing each payer.
//! - [`remainder`]: what a split editor shows is assigned and left, and whether it can be saved.
//! - [`rebalance`]: amount splits that went stale when their charge changed, converted to
//!   percents.
//!
//! Money is exact decimal throughout and rounded the way the server always has: shares half to
//! even to cents, percents half away from zero to six places.

pub mod charge;
pub mod input;
pub mod ops;
pub mod rebalance;
pub mod remainder;
pub mod split;
