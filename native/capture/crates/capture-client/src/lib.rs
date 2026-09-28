//! The Trenova Capture agent's side of the server.
//!
//! [`api::Api`] holds the device credential and makes every call, refreshing
//! the access token as it goes. [`pairing`] pairs the computer with a
//! person. [`stream`] keeps the device stream open and turns it into
//! "fetch your requests". [`spool`] keeps every page on disk, encrypted,
//! until [`uploader`] has it on the server and sealed.
//!
//! Nothing here is Windows-specific: the credential store and the page
//! encryption are traits the agent fills with Credential Manager and DPAPI,
//! which is what lets all of it be tested against a mock server.

pub mod api;
pub mod backoff;
pub mod credentials;
pub mod error;
pub mod pairing;
pub mod server;
pub mod spool;
pub mod sse;
pub mod stream;
pub mod uploader;

pub use api::{AgentInfo, Api, PageMarkers, PairingPoll};
pub use credentials::{Credential, MemoryStore, SecretStore};
pub use error::ApiError;
pub use server::{Server, ServerUrlError};
pub use spool::{Protector, Spool, SpoolError, SpoolSummary, SpooledBatch};
