//! The child process: `trenova-imaging child`, started only by the
//! supervisor's sandbox pool.
//!
//! It reads its start-up parameters, loads PDFium and libheif (and with them
//! every font and plugin they read at start), hardens itself, reports in,
//! and then runs one job at a time until told to stop. A job that fails is
//! reported with its typed error and the child carries on; anything worse
//! ends the process, and the supervisor replaces it.

use std::process::ExitCode;

use prost::Message;
use trenova_sandbox::Child;

use crate::engine::error::EngineError;
use crate::engine::{Engine, Sink};
use crate::protocol::{ChildInit, WorkerError, WorkerJob};

/// The largest frame the child reads before it knows its configuration.
const BOOT_FRAME: u32 = 1 << 20;

struct ChildSink<'a> {
    child: &'a mut Child,
}

impl Sink for ChildSink<'_> {
    fn emit(&mut self, message: Vec<u8>) -> Result<(), EngineError> {
        self.child
            .output(&message)
            .map_err(|err| EngineError::internal(format!("send output: {err}")))
    }
}

pub fn run() -> ExitCode {
    let (mut child, policy, init) = match Child::start(BOOT_FRAME) {
        Ok(started) => started,
        Err(err) => {
            report(&format!("start: {err}"));
            return ExitCode::from(2);
        }
    };
    let init = match ChildInit::decode(init.as_slice()) {
        Ok(init) => init,
        Err(err) => {
            report(&format!("init parameters: {err}"));
            return ExitCode::from(2);
        }
    };
    let mut engine = match Engine::new(&init) {
        Ok(engine) => engine,
        Err(err) => {
            report(&format!("load native libraries: {err}"));
            return ExitCode::from(3);
        }
    };
    if let Err(err) = child.harden_and_ready(&policy) {
        report(&format!("harden: {err}"));
        return ExitCode::from(4);
    }

    loop {
        let job = match child.next_job() {
            Ok(Some(job)) => job,
            Ok(None) => return ExitCode::SUCCESS,
            Err(err) => {
                report(&format!("receive job: {err}"));
                return ExitCode::from(5);
            }
        };
        let outcome = match WorkerJob::decode(job.body.as_slice()) {
            Ok(request) => {
                let input = job.input.as_ref().map_or(&[][..], |input| input.as_bytes());
                engine.run(&request, input, &mut ChildSink { child: &mut child })
            }
            Err(err) => Err(EngineError::request(format!("the job is malformed: {err}"))),
        };
        let sent = match outcome {
            Ok(done) => child.done(&done),
            Err(err) => child.failed(&WorkerError::new(err.reason, err.message).encode_to_vec()),
        };
        if let Err(err) = sent {
            report(&format!("send result: {err}"));
            return ExitCode::from(6);
        }
        drop(job);
    }
}

/// One line on stderr, which the supervisor logs. Never document content.
fn report(message: &str) {
    use std::io::Write;
    let _ = writeln!(std::io::stderr(), "imaging child: {message}");
}
