//! What can go wrong talking to the server, sorted by what the agent does
//! about it.

use std::time::Duration;

use capture_protocol::api::ProblemDetail;

#[derive(Debug, thiserror::Error)]
pub enum ApiError {
    /// No connection, a reset, DNS, TLS. Try again later.
    #[error("could not reach the Trenova server: {0}")]
    Network(String),
    /// The device has no credential.
    #[error("this computer is not signed in")]
    NotSignedIn,
    /// The credential was revoked or has lapsed; the person must pair again.
    #[error("this computer was signed out of Trenova")]
    SignedOut,
    /// The organization needs a newer companion.
    #[error("Trenova Capture {minimum_version} or later is required")]
    Outdated { minimum_version: String },
    /// The organization turned capture off. Pages stay on this computer
    /// until it is turned back on.
    #[error("scanning into Trenova is turned off for your organization")]
    CaptureDisabled,
    /// Refused for the person: a permission was removed.
    #[error("{}", .0.summary())]
    Denied(Box<ProblemDetail>),
    #[error("{}", .0.summary())]
    NotFound(Box<ProblemDetail>),
    #[error("{}", .0.summary())]
    Conflict(Box<ProblemDetail>),
    #[error("the upload is larger than the server accepts")]
    TooLarge,
    #[error("{}", .0.summary())]
    Invalid(Box<ProblemDetail>),
    #[error("the server asked to slow down")]
    RateLimited { retry_after: Option<Duration> },
    #[error("the server had a problem ({status})")]
    Server { status: u16 },
    #[error("unexpected answer from the server ({status})")]
    Unexpected { status: u16 },
    #[error("unreadable answer from the server: {0}")]
    Decode(String),
}

impl ApiError {
    /// Whether the same request may succeed if simply tried again later.
    pub fn is_retryable(&self) -> bool {
        matches!(
            self,
            Self::Network(_) | Self::RateLimited { .. } | Self::Server { .. }
        )
    }

    /// Whether nothing more can be sent until the person or the
    /// organization changes something (signs in, updates, turns capture on).
    pub fn blocks_everything(&self) -> bool {
        matches!(
            self,
            Self::NotSignedIn
                | Self::SignedOut
                | Self::Outdated { .. }
                | Self::CaptureDisabled
                | Self::Denied(_)
        )
    }

    /// How long the server asked to wait, when it said.
    pub fn retry_after(&self) -> Option<Duration> {
        match self {
            Self::RateLimited { retry_after } => *retry_after,
            _ => None,
        }
    }
}

impl From<reqwest::Error> for ApiError {
    fn from(err: reqwest::Error) -> Self {
        if err.is_decode() {
            return Self::Decode(err.to_string());
        }
        // Nothing in a reqwest error carries a credential: headers are not
        // part of its message.
        Self::Network(err.to_string())
    }
}
