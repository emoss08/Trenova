//! The JSON the host and the module exchange, one request and one step at a time.
//!
//! A run is a conversation. The host sends [`Request::Start`] with the script, the reads its
//! agent holds, its inputs, the clock and the limits; the module answers with a [`Step`]. A
//! [`Step::Call`] asks the host to run one of its functions and send back the answer
//! as [`Request::Resume`]; [`Step::Done`] ends the run. Payloads and replies stay JSON text the
//! script's own SDK encodes and decodes, so the module never interprets a read's arguments or
//! its rows: the host's broker decides everything about a read.

use serde::{Deserialize, Serialize};

/// What the host asks of the module.
#[derive(Debug, Deserialize)]
#[serde(tag = "op", rename_all = "camelCase", deny_unknown_fields)]
pub enum Request {
    /// Start a script. Only valid as an instance's first request.
    Start(Box<Start>),
    /// Answer the host function the last [`Step::Call`] named, with the JSON text it returns.
    Resume {
        /// The function's return value, as JSON text.
        reply: String,
    },
}

/// One script and everything it runs with.
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Start {
    /// The Python source.
    pub code: String,
    /// The tools the script may call, as JSON text: `[{name, description, parameters, writes}]`.
    pub index: String,
    /// The script's inputs, as JSON text of an object, read as `trenova.inputs`.
    pub inputs: String,
    /// The datasets the script reads, as JSON text: `[{name, label, rows, columns: [{name, kind}]}]`.
    /// Their rows are not here: the SDK pages them in with [`HostFunction::Rows`].
    #[serde(default = "no_datasets")]
    pub datasets: String,
    /// The instant `date.today()` and `datetime.now()` read, in the organization's zone.
    pub clock: Clock,
    /// What the run may spend.
    pub limits: Limits,
}

fn no_datasets() -> String {
    "[]".to_owned()
}

/// A fixed clock: every read of the time inside the script sees this instant.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Clock {
    /// Seconds since the Unix epoch.
    pub unix_seconds: i64,
    /// The organization's offset from UTC at that instant.
    pub offset_seconds: i32,
    /// The zone's name, reported by `tzname` and `%Z`.
    #[serde(default)]
    pub zone: Option<String>,
}

/// The module's own limits. Reads, rows, bytes and wall time are the host's.
#[derive(Debug, Clone, Copy, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Limits {
    /// Printed output kept, counted in characters from the end.
    pub output_chars: usize,
    /// Interpreter memory, in bytes, past which the script raises `MemoryError`.
    pub memory_bytes: usize,
    /// Interpreter time, in milliseconds, not counting time spent waiting on the host.
    pub cpu_ms: u64,
    /// Python call depth.
    pub recursion: usize,
    /// Reads the script may make in all: [`HostFunction::Read`], [`HostFunction::Gather`] and
    /// [`HostFunction::Result`] calls. Paging a dataset in and showing a result are the host's
    /// to bound, and do not count.
    pub calls: usize,
}

/// What the module answers with.
#[derive(Debug, Serialize)]
#[serde(tag = "step", rename_all = "camelCase")]
pub enum Step {
    /// The script is paused on a host function.
    Call {
        /// Which function.
        function: HostFunction,
        /// Its one argument, JSON text the SDK encoded.
        payload: String,
    },
    /// The script ended.
    Done(Done),
}

/// The things a script can ask of the host.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum HostFunction {
    /// One read: `{"tool", "args"}`, answered with `{"status", "data"?, "message"?}`.
    Read,
    /// Several reads at once: `[{"tool", "args"}]`, answered with a list of replies in order.
    Gather,
    /// The script's one structured result, answered with `{"status"}`.
    Result,
    /// A page of a dataset's rows: `{"dataset", "offset"}`, answered with `{"rows", "done"}`.
    Rows,
    /// A figure, table or chart for the person: `{"type", ...}`, answered with `{"status"}`.
    Show,
}

impl HostFunction {
    /// Whether a call spends the script's reads.
    #[must_use]
    pub const fn is_read(self) -> bool {
        matches!(self, Self::Read | Self::Gather | Self::Result)
    }
}

/// How a script ended.
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Done {
    /// Whether it finished, raised, hit a limit or could not run.
    pub status: EndStatus,
    /// The limit that ended it.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub limit: Option<Limit>,
    /// The exception it raised, or why it could not run.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<ScriptError>,
    /// What it printed, its end kept.
    pub output: String,
    /// Whether the start of the output was dropped.
    pub output_truncated: bool,
}

/// The ways a script ends.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum EndStatus {
    /// It ran to its end.
    Ok,
    /// It raised an exception it did not catch.
    Raised,
    /// A limit stopped it.
    Limit,
    /// It could not run: the request was malformed or the interpreter failed.
    Failed,
}

/// The module's limits, named as the host reports them.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum Limit {
    /// [`Limits::memory_bytes`].
    Memory,
    /// [`Limits::cpu_ms`].
    Cpu,
    /// [`Limits::calls`].
    Calls,
}

/// An exception, as Python would print it.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ScriptError {
    /// The exception's type, such as `KeyError`.
    pub name: String,
    /// Its message.
    pub value: String,
    /// The traceback, with the script's frames named `script.py`.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub traceback: String,
}
