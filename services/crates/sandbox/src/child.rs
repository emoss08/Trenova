//! The child's side: read the `Init` frame, set up, harden, report `Ready`,
//! then serve one job at a time until told to stop.
//!
//! The channel is the child's standard input, a Unix socket the supervisor
//! holds the other end of. Standard output is closed (a library that prints
//! must not corrupt the channel) and standard error goes to the supervisor's
//! log, a line at a time; nothing a child writes there may carry document
//! content.

use std::io;
use std::os::fd::{AsFd, FromRawFd, OwnedFd};
use std::os::unix::net::UnixStream;

use crate::frame::{Channel, FrameError, Kind};
use crate::harden::{HardeningPolicy, HardeningReport, harden};
use crate::input::MappedInput;
use crate::wire::{Init, decode_init};

/// The child end of the channel.
#[derive(Debug)]
pub struct Child {
    channel: Channel,
}

/// A job as the child receives it.
#[derive(Debug)]
pub struct ChildJob {
    pub body: Vec<u8>,
    pub input: Option<MappedInput>,
}

#[derive(Debug, thiserror::Error)]
pub enum ChildError {
    #[error(transparent)]
    Frame(#[from] FrameError),
    #[error("the first frame was not Init")]
    NotInit,
    #[error("the init frame is malformed: {0}")]
    BadInit(String),
    #[error("map the job input: {0}")]
    Input(io::Error),
}

impl Child {
    /// Takes the channel from standard input and reads the `Init` frame.
    /// Returns the policy to harden with and the service's own parameters.
    pub fn start(max_frame: u32) -> Result<(Self, HardeningPolicy, Vec<u8>), ChildError> {
        // SAFETY: the supervisor passes the channel as fd 0 and nothing else in
        // this process owns it; taking ownership once at start is the only use.
        let fd = unsafe { OwnedFd::from_raw_fd(0) };
        let stream = UnixStream::from(fd);
        let mut channel = Channel::new(stream, max_frame);
        let frame = channel.recv()?;
        if frame.kind != Kind::Init {
            return Err(ChildError::NotInit);
        }
        let Init { policy, service } = decode_init(&frame.body).map_err(ChildError::BadInit)?;
        Ok((Self { channel }, policy, service))
    }

    /// Hardens the process with `policy` and reports what is in force. Call
    /// once the service has loaded everything it needs from disk.
    pub fn harden_and_ready(
        &mut self,
        policy: &HardeningPolicy,
    ) -> Result<HardeningReport, ChildError> {
        let report = harden(policy);
        let body =
            serde_json::to_vec(&report).map_err(|err| ChildError::BadInit(err.to_string()))?;
        self.channel.send(Kind::Ready, &body, &[])?;
        Ok(report)
    }

    /// The next job, or `None` when the supervisor says to stop or goes away.
    pub fn next_job(&mut self) -> Result<Option<ChildJob>, ChildError> {
        let frame = match self.channel.recv() {
            Ok(frame) => frame,
            Err(FrameError::Closed) => return Ok(None),
            Err(err) => return Err(err.into()),
        };
        match frame.kind {
            Kind::Shutdown => Ok(None),
            Kind::Job => {
                let input = frame
                    .fds
                    .into_iter()
                    .next()
                    .map(MappedInput::map)
                    .transpose()
                    .map_err(ChildError::Input)?;
                Ok(Some(ChildJob {
                    body: frame.body,
                    input,
                }))
            }
            _ => Err(FrameError::Protocol("unexpected frame for a child").into()),
        }
    }

    /// Sends one piece of the current job's output.
    pub fn output(&mut self, body: &[u8]) -> Result<(), ChildError> {
        Ok(self.channel.send(Kind::Output, body, &[])?)
    }

    /// Ends the current job successfully.
    pub fn done(&mut self, body: &[u8]) -> Result<(), ChildError> {
        Ok(self.channel.send(Kind::Done, body, &[])?)
    }

    /// Ends the current job with the service's typed error.
    pub fn failed(&mut self, body: &[u8]) -> Result<(), ChildError> {
        Ok(self.channel.send(Kind::Failed, body, &[])?)
    }

    pub fn channel_fd(&self) -> std::os::fd::BorrowedFd<'_> {
        self.channel.stream().as_fd()
    }
}
