//! Agent scripts, run in Monty and paused at every read.
//!
//! An agent answers a fan-out question ("every customer past terms, with their last payment")
//! with one short read-only Python script instead of dozens of tool calls. This crate runs that
//! script in [Monty](https://github.com/pydantic/monty), a Python interpreter written in Rust
//! that has no access to files, the environment or the network unless its host grants it. Ours
//! grants nothing: the script reaches data only through the `trenova` SDK, whose every call
//! pauses the run and hands the host a step to answer, so the TMS's broker runs each read as
//! the person, exactly as the same read made by the model would run.
//!
//! The crate compiles to one `wasm32-wasip1` module that only the TMS runs, in wazero, with a
//! fresh instance per script: starting one takes about a millisecond, and the instance's
//! linear memory is the script's whole world. See `docs/engineering/agent-code-mode.md`.

pub mod output;
pub mod session;
pub mod wire;

#[cfg(target_arch = "wasm32")]
pub mod abi;

pub use session::{Progress, Session, resume, start};
pub use wire::{Done, EndStatus, HostFunction, Limit, Limits, Request, ScriptError, Start, Step};
