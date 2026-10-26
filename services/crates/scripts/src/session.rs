//! One script's run: the SDK, then the script, paused at every host call.
//!
//! A run feeds two snippets to one Monty REPL. The first is the SDK (`sdk.py`), which builds the
//! `trenova` object from the index and inputs the host sent; the second is the script, so it
//! sees `trenova`, `inputs` and `help` as globals. Neither can reach a file, the environment,
//! a module Monty does not ship or the network: the only way out is the SDK's host functions,
//! and each one pauses the run and hands the host a [`Step::Call`].
//!
//! Everything else that would leave the sandbox is answered here: an unknown function raises
//! `NameError`, an unknown import `ModuleNotFoundError`, a file or environment call `OSError`,
//! and `time.sleep` returns at once. The clock is fixed at the instant the host sent, in the
//! organization's zone, so a script reads the same "today" the model was told.

use monty::{MontyRepl, ReplFunctionCall, ReplProgress, ReplStartError};
use monty_types::{
    CompileOptions, DateTimeSource, ExcType, ExtFunctionResult, MontyException, MontyObject,
    NameLookupResult, OsFunctionCall, OsPolicy, PrintWriter, ResourceLimits, ResourceTracker,
    SandboxTimeZone, SleepMode,
};

use crate::output::Tail;
use crate::wire::{Done, EndStatus, HostFunction, Limit, Limits, ScriptError, Start, Step};

/// The SDK every script runs on top of.
pub const SDK_SOURCE: &str = include_str!("sdk.py");

const SDK_FILE: &str = "trenova.py";
const SCRIPT_FILE: &str = "script.py";
const SDK_SNIPPET: &str = "<python-input-0>";
const SCRIPT_SNIPPET: &str = "<python-input-1>";
const BOUND_SCRIPT_SNIPPET: &str = "<python-input-2>";
const BINDINGS_FILE: &str = "datasets.py";
const NO_OUTSIDE: &str = "scripts cannot reach files, the environment or the network";

const READ: &str = "_trenova_read";
const GATHER: &str = "_trenova_gather";
const RESULT: &str = "_trenova_result";
const ROWS: &str = "_trenova_rows";
const SHOW: &str = "_trenova_show";

/// Host calls that page datasets in or show results, beyond the script's reads. The host
/// bounds what they carry; this keeps a script that loops on them from running forever.
const MAX_OTHER_CALLS: usize = 20_000;

fn host_function(name: &str) -> Option<HostFunction> {
    match name {
        READ => Some(HostFunction::Read),
        GATHER => Some(HostFunction::Gather),
        RESULT => Some(HostFunction::Result),
        ROWS => Some(HostFunction::Rows),
        SHOW => Some(HostFunction::Show),
        _ => None,
    }
}

/// A script paused on a host function, waiting for its answer.
#[derive(Debug)]
pub struct Session {
    paused: ReplFunctionCall,
    output: Tail,
    calls: Calls,
    limits: Limits,
    /// The snippets still to feed, in reverse order: the dataset bindings and the script,
    /// while the SDK is still paging datasets in before them.
    pending: Vec<String>,
}

/// The host calls a script has made: its reads, and everything else.
#[derive(Debug, Clone, Copy, Default)]
struct Calls {
    reads: usize,
    other: usize,
}

/// Where a run stands after a step.
#[derive(Debug)]
pub enum Progress {
    /// Paused on a host call; resume the session with its answer.
    Paused(Box<Session>, Step),
    /// Ended.
    Done(Step),
}

impl Progress {
    /// The step to hand the host.
    #[must_use]
    pub fn step(&self) -> &Step {
        match self {
            Self::Paused(_, step) | Self::Done(step) => step,
        }
    }
}

/// Starts a script: runs the SDK, then the script up to its first host call or its end.
#[must_use]
pub fn start(start: &Start) -> Progress {
    let limits = start.limits;
    let mut output = Tail::new(limits.output_chars);
    let repl = MontyRepl::new(SDK_FILE, tracker(&limits), CompileOptions::default())
        .with_os_policy(os_policy(start));

    let inputs = vec![
        (
            "_TRENOVA_INDEX".to_owned(),
            MontyObject::string(start.index.clone()),
        ),
        (
            "_TRENOVA_INPUTS".to_owned(),
            MontyObject::string(start.inputs.clone()),
        ),
        (
            "_TRENOVA_DATASETS".to_owned(),
            MontyObject::string(start.datasets.clone()),
        ),
    ];
    let progress = repl.feed_start(SDK_SOURCE, inputs, PrintWriter::Callback(&mut output));
    if let Err(error) = &progress {
        return failed(output, &format!("the SDK could not start: {}", error.error));
    }
    let bindings = match dataset_bindings(&start.datasets) {
        Ok(bindings) => bindings,
        Err(why) => return failed(output, &why),
    };
    // The SDK pauses before the script only to page a dataset in; the bindings and the script
    // are fed when the SDK's snippet completes, however many pages that took.
    let mut pending = vec![start.code.clone()];
    if !bindings.is_empty() {
        pending.push(bindings);
    }
    drive(progress, output, Calls::default(), limits, pending)
}

/// Names a dataset may take: a Python name that is not one the SDK defines.
const RESERVED_NAMES: &[&str] = &[
    "trenova",
    "inputs",
    "help",
    "json",
    "datetime",
    "show_figure",
    "show_table",
    "show_chart",
    "mean",
    "median",
    "percentile",
    "stdev",
    "group_by",
    "Refused",
];

/// The snippet that binds each dataset's rows to its name, as the script reads them.
fn dataset_bindings(datasets: &str) -> Result<String, String> {
    #[derive(serde::Deserialize)]
    struct Named {
        name: String,
    }
    let named: Vec<Named> = serde_json::from_str(datasets)
        .map_err(|error| format!("the datasets are malformed: {error}"))?;
    let mut bindings = String::new();
    for dataset in named {
        let valid = dataset.name.len() <= 40
            && dataset.name.starts_with(|c: char| c.is_ascii_lowercase())
            && dataset
                .name
                .chars()
                .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '_')
            && !RESERVED_NAMES.contains(&dataset.name.as_str());
        if !valid {
            return Err(format!("{:?} cannot name a dataset", dataset.name));
        }
        bindings.push_str(&dataset.name);
        bindings.push_str(" = _DATASET_ROWS[\"");
        bindings.push_str(&dataset.name);
        bindings.push_str("\"]\n");
    }
    Ok(bindings)
}

/// Answers the host function the session is paused on and runs to the next step.
#[must_use]
pub fn resume(session: Session, reply: String) -> Progress {
    let Session {
        paused,
        mut output,
        calls,
        limits,
        pending,
    } = session;
    let progress = paused.resume(
        MontyObject::string(reply),
        PrintWriter::Callback(&mut output),
    );
    drive(progress, output, calls, limits, pending)
}

fn tracker(limits: &Limits) -> ResourceTracker {
    ResourceTracker::new(ResourceLimits {
        max_feed_duration: Some(std::time::Duration::from_millis(limits.cpu_ms)),
        max_memory: Some(limits.memory_bytes),
        max_recursion_depth: limits.recursion,
        max_suspensions: limits
            .calls
            .saturating_mul(4)
            .max(64)
            .saturating_add(MAX_OTHER_CALLS),
        ..ResourceLimits::default()
    })
}

fn os_policy(start: &Start) -> OsPolicy {
    OsPolicy {
        datetime: DateTimeSource::Fixed {
            unix_seconds: start.clock.unix_seconds,
            microsecond: 0,
        },
        timezone: SandboxTimeZone::Fixed {
            offset_seconds: start.clock.offset_seconds,
            name: start.clock.zone.clone(),
        },
        sleep: SleepMode::Zero,
        ..OsPolicy::default()
    }
}

fn drive(
    mut progress: Result<ReplProgress, Box<ReplStartError>>,
    mut output: Tail,
    mut calls: Calls,
    limits: Limits,
    mut pending: Vec<String>,
) -> Progress {
    loop {
        progress = match progress {
            Ok(ReplProgress::Complete { repl, .. }) => {
                let Some(code) = pending.pop() else {
                    return finished(output, EndStatus::Ok, None, None);
                };
                repl.feed_start(
                    &code,
                    Vec::<(String, MontyObject)>::new(),
                    PrintWriter::Callback(&mut output),
                )
            }
            Ok(ReplProgress::FunctionCall(call)) => {
                let Some(function) = host_function(&call.function_name) else {
                    let name = call.function_name.clone();
                    let print = PrintWriter::Callback(&mut output);
                    progress = call.resume(ExtFunctionResult::NotFound(name), print);
                    continue;
                };
                let payload = match single_string(&call) {
                    Ok(payload) => payload,
                    Err(message) => {
                        let error = MontyException::new(ExcType::TypeError, Some(message));
                        progress = call.resume(error, PrintWriter::Callback(&mut output));
                        continue;
                    }
                };
                if function.is_read() {
                    calls.reads += 1;
                    if calls.reads > limits.calls {
                        return finished(output, EndStatus::Limit, Some(Limit::Calls), None);
                    }
                } else {
                    calls.other += 1;
                    if calls.other > MAX_OTHER_CALLS {
                        return finished(output, EndStatus::Limit, Some(Limit::Calls), None);
                    }
                }
                let step = Step::Call { function, payload };
                return Progress::Paused(
                    Box::new(Session {
                        paused: call,
                        output,
                        calls,
                        limits,
                        pending,
                    }),
                    step,
                );
            }
            Ok(ReplProgress::NameLookup(lookup)) => {
                let answer = if host_function(&lookup.name).is_some() {
                    NameLookupResult::Value(MontyObject::function(lookup.name.clone(), None))
                } else {
                    NameLookupResult::Undefined
                };
                lookup.resume(answer, PrintWriter::Callback(&mut output))
            }
            Ok(ReplProgress::OsCall(call)) => {
                let answer = match &call.function_call {
                    OsFunctionCall::Import(name) => ExtFunctionResult::NotFound(name.clone()),
                    _ => MontyException::new(ExcType::OSError, Some(NO_OUTSIDE.to_owned())).into(),
                };
                call.resume(answer, PrintWriter::Callback(&mut output))
            }
            Ok(ReplProgress::ResolveFutures(futures)) => {
                let error = MontyException::new(
                    ExcType::RuntimeError,
                    Some("a script's reads are not awaited; call them, or use gather()".to_owned()),
                );
                futures.abort(error, PrintWriter::Callback(&mut output))
            }
            Err(error) => return ended(output, &error.error),
        };
    }
}

fn single_string(call: &ReplFunctionCall) -> Result<String, String> {
    let mut args = call.args.args();
    let (Some(only), None, 0) = (args.next(), args.next(), call.args.kwargs().len()) else {
        return Err(format!("{}() takes one string", call.function_name));
    };
    only.as_str()
        .map(str::to_owned)
        .ok_or_else(|| format!("{}() takes one string", call.function_name))
}

fn ended(output: Tail, error: &MontyException) -> Progress {
    let message = error.message().unwrap_or_default();
    let limit = match error.exc_type() {
        ExcType::MemoryError if message.contains("memory limit exceeded") => Some(Limit::Memory),
        ExcType::TimeoutError if message.contains("time limit exceeded") => Some(Limit::Cpu),
        _ => None,
    };
    let status = if limit.is_some() {
        EndStatus::Limit
    } else {
        EndStatus::Raised
    };
    finished(output, status, limit, Some(script_error(error)))
}

fn script_error(error: &MontyException) -> ScriptError {
    let traceback = error.to_string();
    // With datasets bound, the bindings are the second snippet and the script the third.
    let traceback = if traceback.contains(BOUND_SCRIPT_SNIPPET) {
        traceback
            .replace(BOUND_SCRIPT_SNIPPET, SCRIPT_FILE)
            .replace(SCRIPT_SNIPPET, BINDINGS_FILE)
    } else {
        traceback.replace(SCRIPT_SNIPPET, SCRIPT_FILE)
    };
    ScriptError {
        name: error.exc_type().to_string(),
        value: error.message().unwrap_or_default().to_owned(),
        traceback: traceback.replace(SDK_SNIPPET, SDK_FILE),
    }
}

fn failed(output: Tail, why: &str) -> Progress {
    let error = ScriptError {
        name: "Failed".to_owned(),
        value: why.to_owned(),
        traceback: String::new(),
    };
    finished(output, EndStatus::Failed, None, Some(error))
}

fn finished(
    output: Tail,
    status: EndStatus,
    limit: Option<Limit>,
    error: Option<ScriptError>,
) -> Progress {
    let (output, output_truncated) = output.finish();
    Progress::Done(Step::Done(Done {
        status,
        limit,
        error,
        output,
        output_truncated,
    }))
}
