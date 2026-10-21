//! The engine's one error type: an [`ErrorReason`] from the contract and a
//! message safe to log, which never quotes document content.

use trenova_proto::imaging::v1::ErrorReason;

use crate::protocol::WorkerError;

#[derive(Clone, Debug, PartialEq, Eq, thiserror::Error)]
#[error("{message}")]
pub struct EngineError {
    pub reason: ErrorReason,
    pub message: String,
}

impl EngineError {
    pub fn new(reason: ErrorReason, message: impl Into<String>) -> Self {
        Self {
            reason,
            message: message.into(),
        }
    }

    pub fn invalid(message: impl Into<String>) -> Self {
        Self::new(ErrorReason::InvalidDocument, message)
    }

    pub fn request(message: impl Into<String>) -> Self {
        Self::new(ErrorReason::InvalidRequest, message)
    }

    pub fn unsupported(message: impl Into<String>) -> Self {
        Self::new(ErrorReason::UnsupportedFormat, message)
    }

    pub fn internal(message: impl Into<String>) -> Self {
        Self::new(ErrorReason::Internal, message)
    }

    pub fn pixels(message: impl Into<String>) -> Self {
        Self::new(ErrorReason::PixelBudget, message)
    }

    pub fn page(page: u32, count: u32) -> Self {
        Self::new(
            ErrorReason::PageOutOfRange,
            format!("page {page} is outside the document's {count} pages"),
        )
    }

    pub fn to_worker(&self) -> WorkerError {
        WorkerError::new(self.reason, self.message.clone())
    }
}

impl From<trenova_imaging_core::ImagingError> for EngineError {
    fn from(err: trenova_imaging_core::ImagingError) -> Self {
        Self::internal(err.to_string())
    }
}

pub type EngineResult<T> = Result<T, EngineError>;
