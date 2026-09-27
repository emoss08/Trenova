//! Waiting for a source to say something.
//!
//! Once enabled, a source reports that pages are ready (or that it wants to
//! close) either through a TWAIN 2 callback or, for a TWAIN 1 source, as the
//! answer to `DAT_EVENT` for the application's window messages. The pump owns
//! that loop, and the session tells it how to hand each message to the source.

use core::ffi::c_void;
use std::time::Duration;

/// What the source made of one window message.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Processed {
    /// Not the source's; dispatch it normally.
    NotDsEvent,
    /// The source consumed it, and may have said something (`MSG_NULL` if
    /// not).
    DsEvent(u16),
}

/// What woke the pump.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PumpEvent {
    /// A TWAIN message from the source.
    Twain(u16),
    /// The agent asked to stop.
    Cancel,
    /// Nothing happened in time.
    TimedOut,
}

pub trait EventPump {
    /// The callback a TWAIN 2 source should call, if this pump has one.
    fn callback(&self) -> Option<*mut c_void>;

    /// Runs until the source says something, the agent cancels, or the
    /// timeout passes. `process` hands one window message to the source.
    fn wait(
        &mut self,
        timeout: Option<Duration>,
        process: &mut dyn FnMut(*mut c_void) -> Processed,
    ) -> PumpEvent;
}
