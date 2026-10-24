//! The WebAssembly boundary: three exports, JSON in and out, one script per instance.
//!
//! The host writes a [`Request`] into memory from [`scripts_alloc`], calls [`scripts_call`]
//! with its address and length, and gets back one `u64`: the [`Step`]'s address in the high 32
//! bits and its length in the low 32. It reads the step and hands both buffers back with
//! [`scripts_free`]. The first request starts the script; every later one resumes it, until a
//! step says it is done. The host makes a fresh instance for every script and drops it after,
//! so no state outlives a run.
//!
//! Memory is counted by `monty-alloc`: the interpreter raises `MemoryError` at the script's
//! limit, and past a margin above it the module exits with `OOM_EXIT_CODE`, which the host
//! reads as the memory limit.

use std::cell::RefCell;

use monty_types::{OOM_EXIT_CODE, memory_limit_with_headroom};

use crate::session::{self, Progress, Session};
use crate::wire::{Done, EndStatus, Request, ScriptError, Step};

#[global_allocator]
static ALLOCATOR: monty_alloc::LimitedAllocator = monty_alloc::LimitedAllocator;

thread_local! {
    static RUN: RefCell<Run> = const { RefCell::new(Run::Unstarted) };
}

#[derive(Debug)]
enum Run {
    Unstarted,
    Paused(Box<Session>),
    Ended,
}

/// The exit code the module ends with when a script passes its hard memory ceiling.
#[unsafe(no_mangle)]
pub extern "C" fn scripts_oom_exit_code() -> i32 {
    OOM_EXIT_CODE
}

/// Allocates `len` bytes inside the module for the host to write a request into.
///
/// Returns a null pointer when `len` is zero.
#[unsafe(no_mangle)]
pub extern "C" fn scripts_alloc(len: u32) -> *mut u8 {
    if len == 0 {
        return std::ptr::null_mut();
    }
    let mut buffer = vec![0u8; len as usize].into_boxed_slice();
    let pointer = buffer.as_mut_ptr();
    std::mem::forget(buffer);
    pointer
}

/// Returns a buffer from [`scripts_alloc`] or a step from [`scripts_call`] to the module.
///
/// # Safety
///
/// `pointer` and `len` must describe exactly one buffer this module handed out and the host has
/// not freed yet. A null pointer or zero length is ignored.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn scripts_free(pointer: *mut u8, len: u32) {
    if pointer.is_null() || len == 0 {
        return;
    }
    // SAFETY: the caller promises `pointer` and `len` came from one live boxed slice this module
    // leaked in `scripts_alloc` or `scripts_call`, so rebuilding and dropping it frees it once.
    unsafe {
        drop(Box::from_raw(std::ptr::slice_from_raw_parts_mut(
            pointer,
            len as usize,
        )));
    }
}

/// Runs one request and returns the step's address and length packed into a `u64`.
///
/// # Safety
///
/// `pointer` and `len` must describe initialised memory the host owns for the duration of the
/// call (normally a buffer from [`scripts_alloc`]).
#[unsafe(no_mangle)]
pub unsafe extern "C" fn scripts_call(pointer: *const u8, len: u32) -> u64 {
    let request: &[u8] = if pointer.is_null() || len == 0 {
        &[]
    } else {
        // SAFETY: the caller promises `pointer..pointer + len` is initialised and stays valid
        // while this call runs.
        unsafe { std::slice::from_raw_parts(pointer, len as usize) }
    };
    let step = RUN.with_borrow_mut(|run| advance(run, request));
    let encoded = serde_json::to_vec(&step).unwrap_or_else(|_| b"{}".to_vec());
    let response = encoded.into_boxed_slice();
    let response_len = u32::try_from(response.len()).unwrap_or(u32::MAX);
    let response_pointer = Box::into_raw(response).cast::<u8>();
    let address = u32::try_from(response_pointer as usize).unwrap_or(0);
    (u64::from(address) << 32) | u64::from(response_len)
}

fn advance(run: &mut Run, request: &[u8]) -> Step {
    let request = match serde_json::from_slice::<Request>(request) {
        Ok(request) => request,
        Err(error) => return refused(&format!("the request is not valid: {error}")),
    };
    let progress = match (std::mem::replace(run, Run::Ended), request) {
        (Run::Unstarted, Request::Start(start)) => {
            let ceiling = memory_limit_with_headroom(Some(start.limits.memory_bytes), false);
            if monty_alloc::set_hard_limit(ceiling).is_err() {
                return refused("the module's allocator is not installed");
            }
            session::start(&start)
        }
        (Run::Paused(session), Request::Resume { reply }) => session::resume(*session, reply),
        (Run::Unstarted, Request::Resume { .. }) => return refused("no script has started"),
        (_, Request::Start(_)) => return refused("a script has already started"),
        (Run::Ended, Request::Resume { .. }) => return refused("the script has ended"),
    };
    match progress {
        Progress::Paused(session, step) => {
            *run = Run::Paused(session);
            step
        }
        Progress::Done(step) => step,
    }
}

fn refused(why: &str) -> Step {
    Step::Done(Done {
        status: EndStatus::Failed,
        limit: None,
        error: Some(ScriptError {
            name: "Failed".to_owned(),
            value: why.to_owned(),
            traceback: String::new(),
        }),
        output: String::new(),
        output_truncated: false,
    })
}
