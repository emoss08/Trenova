//! The chassis every Trenova Rust service is built on, so each one starts,
//! configures, logs, measures, serves, sheds load and stops the same way.
//!
//! - [`config`]: one typed configuration from an optional TOML file and the
//!   environment, decoded strictly and validated before anything starts.
//! - [`telemetry`]: JSON logs, OTLP traces and the W3C trace context Go's
//!   `otelgrpc` sends, so a span started in the TMS continues here.
//! - [`metrics`]: a Prometheus registry and the `/metrics` listener.
//! - [`grpc`]: the tonic server: mutual TLS, `grpc.health.v1`, message and
//!   stream limits, keepalives, a trace span per call. No reflection.
//! - [`shutdown`]: SIGTERM and SIGINT, and draining in-flight work within a
//!   grace period.
//! - [`admission`]: a per-process budget of cost and memory, two priority
//!   classes with a reserve for interactive work, and the retry-after a
//!   refused caller is told.
//!
//! Nothing here logs a request's content; the services built on it handle
//! tenant documents and must not either.

pub mod admission;
pub mod config;
pub mod grpc;
pub mod metrics;
pub mod shutdown;
pub mod telemetry;

pub use admission::{Admission, AdmissionConfig, Cost, Permit, Priority, Rejection};
pub use config::{ConfigError, Validate, Validation};
pub use shutdown::Shutdown;
