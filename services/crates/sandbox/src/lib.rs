//! A supervised pool of hardened child processes, for running code that
//! parses untrusted input (PDF renderers, image codecs, OCR) where a crash,
//! a hang or an exploit takes down one disposable process and nothing else.
//!
//! The supervisor ([`pool::Pool`]) keeps a fixed number of children, each
//! running one job at a time. A child is the service's own binary started in
//! a child mode: it reads an `Init` frame, loads what it needs, hardens
//! itself ([`harden`]: no-new-privs, landlock, seccomp-bpf; rlimits are set
//! by the supervisor before exec), reports what is in force, and serves
//! jobs over a framed protocol on its standard input ([`frame`]). A job's
//! input travels as a sealed memfd ([`input`]) the child maps read-only.
//!
//! The pool fails closed: a child whose protections fall short of the
//! policy is refused unless the service is explicitly configured to run
//! without them, which is logged at error level each time.
//!
//! Nothing here knows about documents. The service decides what a job's body
//! means; this crate runs it somewhere it can do no harm.

pub mod child;
pub mod frame;
pub mod harden;
pub mod input;
#[cfg(target_os = "linux")]
pub mod pool;
#[cfg(target_os = "linux")]
pub mod seccomp;
pub mod wire;

pub use child::{Child, ChildJob};
pub use harden::{HardeningPolicy, HardeningReport, LandlockStatus};
pub use input::{InputWriter, MappedInput, SealedInput};
#[cfg(target_os = "linux")]
pub use pool::{ChildSpec, Job, JobError, Limits, Outcome, Pool, PoolConfig, SandboxError};
