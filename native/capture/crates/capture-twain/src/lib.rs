//! TWAIN for the scan helper.
//!
//! [`session`] walks one scan through the TWAIN state machine over a [`Dsm`]:
//! open the manager, list or open a source, negotiate the profile, enable it,
//! and take each page by buffered memory transfer (`TWSX_MEMORY`), with the
//! scanner's patch codes and barcodes when it reads them. [`win`] is the
//! Windows side: `twaindsm.dll`, the hidden window, and the message loop and
//! callback a source reports through.
//!
//! The declarations in [`sys`] are generated from the TWAIN Working Group's
//! `twain.h` (`tools/generate-bindings.sh`).

pub mod capability;
pub mod consts;
pub mod dsm;
pub mod pump;
pub mod session;
pub mod sys;
pub mod text;
#[cfg(windows)]
pub mod win;

#[cfg(test)]
mod fake;
#[cfg(test)]
mod tests;

use capture_protocol::api::RequestFailureCode;

pub use capture_imaging::scan::{ScanEnd, ScanSettings, ScannedPage};
pub use dsm::Dsm;
pub use pump::{EventPump, Processed, PumpEvent};
pub use session::{AppIdentity, Manager, Negotiated, Source, SourceEntry};

pub type TwainResult<T> = Result<T, TwainError>;

#[derive(Debug, thiserror::Error)]
pub enum TwainError {
    #[error("TWAIN is not installed: {0}")]
    Load(String),
    #[error("could not {operation} (return code {rc})")]
    Dsm { operation: &'static str, rc: u16 },
    #[error("the scanner could not {operation} (condition {condition})")]
    Source {
        operation: &'static str,
        condition: u16,
    },
    #[error("could not open {name} (condition {condition})")]
    Open { name: String, condition: u16 },
    #[error("no TWAIN scanner named {0:?} is installed")]
    SourceNotFound(String),
    #[error("the scanner sent {0}, which is not supported")]
    Unsupported(String),
    #[error("a page could not be handed on: {0}")]
    Delivery(String),
}

impl TwainError {
    /// How the failure reads on the request the person is watching.
    pub fn failure_code(&self) -> RequestFailureCode {
        use consts::{TWCC_CHECKDEVICEONLINE, TWCC_DENIED, TWCC_MAXCONNECTIONS};

        match self {
            Self::Load(_) | Self::SourceNotFound(_) => RequestFailureCode::SourceUnavailable,
            Self::Open { condition, .. } | Self::Source { condition, .. } => match *condition {
                TWCC_MAXCONNECTIONS | TWCC_DENIED => RequestFailureCode::SourceBusy,
                TWCC_CHECKDEVICEONLINE => RequestFailureCode::SourceUnavailable,
                _ => RequestFailureCode::DriverError,
            },
            Self::Dsm { .. } | Self::Unsupported(_) => RequestFailureCode::DriverError,
            Self::Delivery(_) => RequestFailureCode::Internal,
        }
    }
}
