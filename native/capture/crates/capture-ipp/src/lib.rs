//! The IPP/2.0 subset the Trenova virtual printer speaks.
//!
//! Windows prints to the "Trenova" printer through its inbox IPP Class
//! Driver, which talks IPP to the capture service on the loopback interface.
//! [`codec`] is the wire encoding (RFC 8010), [`value`] the attribute types,
//! and [`printer`] the printer itself: the operations the class driver uses,
//! over a [`printer::JobHandler`] that decides what happens to each document.
//! HTTP is the service's concern; this crate takes and returns IPP bodies.

pub mod codec;
pub mod printer;
pub mod value;

pub use printer::{DocumentFormat, IncomingJob, JobHandler, Printer, PrinterConfig, Refusal};
