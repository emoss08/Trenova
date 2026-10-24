//! One script's run: the SDK, then the script, paused at every host call.
//!
//! A run feeds two snippets to one Monty REPL. The first is the SDK (`sdk.py`), which builds the
//! `trenova` object from the index and inputs the host sent; the second is the script, so it
//! sees `trenova`, `inputs` and `help` as globals. Neither can reach a file, the environment,
//! a module Monty does not ship or the network: the only way out is the SDK's three host
//! functions, and each one pauses the run and hands the host a [`Step::Call`].
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
const NO_OUTSIDE: &str = "scripts cannot reach files, the environment or the network";

const READ: &str = "_trenova_read";
const GATHER: &str = "_trenova_gather";
const RESULT: &str = "_trenova_result";

fn host_function(name: &str) -> Option<HostFunction> {
    match name {
        READ => Some(HostFunction::Read),
        GATHER => Some(HostFunction::Gather),
        RESULT => Some(HostFunction::Result),
        _ => None,
    }
}

/// A script paused on a host function, waiting for its answer.
#[derive(Debug)]
pub struct Session {
    paused: ReplFunctionCall,
    output: Tail,
    calls: usize,
    limits: Limits,
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
    ];
    let progress = repl.feed_start(SDK_SOURCE, inputs, PrintWriter::Callback(&mut output));
    let repl = match progress {
        Ok(ReplProgress::Complete { repl, .. }) => repl,
        Ok(_) => return failed(output, "the SDK paused before the script started"),
        Err(error) => return failed(output, &format!("the SDK could not start: {}", error.error)),
    };

    let progress = repl.feed_start(
        &start.code,
        Vec::<(String, MontyObject)>::new(),
        PrintWriter::Callback(&mut output),
    );
    drive(progress, output, 0, limits)
}

/// Answers the host function the session is paused on and runs to the next step.
#[must_use]
pub fn resume(session: Session, reply: String) -> Progress {
    let Session {
        paused,
        mut output,
        calls,
        limits,
    } = session;
    let progress = paused.resume(
        MontyObject::string(reply),
        PrintWriter::Callback(&mut output),
    );
    drive(progress, output, calls, limits)
}

fn tracker(limits: &Limits) -> ResourceTracker {
    ResourceTracker::new(ResourceLimits {
        max_feed_duration: Some(std::time::Duration::from_millis(limits.cpu_ms)),
        max_memory: Some(limits.memory_bytes),
        max_recursion_depth: limits.recursion,
        max_suspensions: limits.calls.saturating_mul(4).max(64),
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
    mut calls: usize,
    limits: Limits,
) -> Progress {
    loop {
        progress = match progress {
            Ok(ReplProgress::Complete { .. }) => {
                return finished(output, EndStatus::Ok, None, None);
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
                calls += 1;
                if calls > limits.calls {
                    return finished(output, EndStatus::Limit, Some(Limit::Calls), None);
                }
                let step = Step::Call { function, payload };
                return Progress::Paused(
                    Box::new(Session {
                        paused: call,
                        output,
                        calls,
                        limits,
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
    ScriptError {
        name: error.exc_type().to_string(),
        value: error.message().unwrap_or_default().to_owned(),
        traceback: error
            .to_string()
            .replace(SCRIPT_SNIPPET, SCRIPT_FILE)
            .replace(SDK_SNIPPET, SDK_FILE),
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
