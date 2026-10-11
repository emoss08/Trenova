//! WIA 2.0, for scanners with no TWAIN driver.
//!
//! Every WIA scanner is reachable through `IWiaDevMgr2` with no vendor
//! software, which is why it is the fallback. It reads feeder and duplex
//! through `WIA_IPS_DOCUMENT_HANDLING_SELECT` and transfers each page as a
//! BMP stream through `IWiaTransfer`, the one format every WIA driver must
//! produce. Patch codes and the driver's own dialog are TWAIN features
//! (design §5.7); asked for here they are recorded as refused and the scan
//! goes on without them, while cover sheets still work because the server
//! reads those.
//!
//! On anything but Windows this crate is empty; the helper only builds it in.

#[cfg_attr(not(windows), allow(dead_code))]
mod allowed;
#[cfg(windows)]
mod props;
#[cfg(windows)]
mod scanner;

#[cfg(windows)]
pub use scanner::{scan, sources};

use capture_protocol::api::RequestFailureCode;

pub type WiaResult<T> = Result<T, WiaError>;

#[derive(Debug, thiserror::Error)]
pub enum WiaError {
    #[error("no WIA scanner named {0:?} is installed")]
    SourceNotFound(String),
    #[error("the scanner is offline")]
    Offline,
    #[error("the scanner is busy")]
    Busy,
    #[error("the scanner has no flatbed or feeder to scan from")]
    NoScanSurface,
    #[error("WIA failed to {operation}: {message}")]
    Com {
        operation: &'static str,
        message: String,
    },
    #[error("a page could not be read: {0}")]
    Image(String),
    #[error("a page could not be handed on: {0}")]
    Delivery(String),
}

impl WiaError {
    /// How the failure reads on the request the person is watching.
    pub fn failure_code(&self) -> RequestFailureCode {
        match self {
            Self::SourceNotFound(_) | Self::Offline | Self::NoScanSurface => {
                RequestFailureCode::SourceUnavailable
            }
            Self::Busy => RequestFailureCode::SourceBusy,
            Self::Com { .. } | Self::Image(_) => RequestFailureCode::DriverError,
            Self::Delivery(_) => RequestFailureCode::Internal,
        }
    }
}
