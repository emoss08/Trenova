//! Failures as gRPC statuses: a code, a message safe to show, and the
//! google.rpc details the contract promises (an `ErrorInfo` whose reason is
//! the `ErrorReason` name without its prefix, and a `RetryInfo` when the
//! caller should come back).

use std::collections::HashMap;
use std::time::Duration;

use tonic::{Code, Status};
use tonic_types::{ErrorDetails, StatusExt};
use trenova_proto::imaging::v1::{ERROR_DOMAIN, ErrorReason};
use trenova_sandbox::JobError;
use trenova_service_kit::Rejection;

use crate::protocol::WorkerError;

pub fn code_for(reason: ErrorReason) -> Code {
    match reason {
        ErrorReason::InvalidDocument
        | ErrorReason::UnsupportedFormat
        | ErrorReason::InvalidRequest => Code::InvalidArgument,
        ErrorReason::Encrypted => Code::FailedPrecondition,
        ErrorReason::TooLarge | ErrorReason::PixelBudget | ErrorReason::PageOutOfRange => {
            Code::OutOfRange
        }
        ErrorReason::Timeout => Code::DeadlineExceeded,
        ErrorReason::ResourceExhausted => Code::ResourceExhausted,
        ErrorReason::Draining => Code::Unavailable,
        ErrorReason::Internal | ErrorReason::WorkerCrashed | ErrorReason::Unspecified => {
            Code::Internal
        }
    }
}

/// The ErrorInfo reason: the enum value's name without `ERROR_REASON_`.
pub fn reason_name(reason: ErrorReason) -> &'static str {
    reason
        .as_str_name()
        .strip_prefix("ERROR_REASON_")
        .unwrap_or("INTERNAL")
}

pub fn status(
    reason: ErrorReason,
    message: impl Into<String>,
    retry_after: Option<Duration>,
) -> Status {
    let mut details = ErrorDetails::with_error_info(
        reason_name(reason),
        ERROR_DOMAIN,
        HashMap::<String, String>::new(),
    );
    if retry_after.is_some() {
        details.set_retry_info(retry_after);
    }
    Status::with_error_details(code_for(reason), message, details)
}

pub fn invalid(message: impl Into<String>) -> Status {
    status(ErrorReason::InvalidRequest, message, None)
}

pub fn from_worker(error: &WorkerError) -> Status {
    status(error.reason(), error.message.clone(), None)
}

pub fn from_job(error: &JobError) -> Status {
    match error {
        JobError::DeadlineExceeded => status(
            ErrorReason::Timeout,
            "the work did not finish before its deadline",
            None,
        ),
        JobError::Crashed(_) => status(
            ErrorReason::WorkerCrashed,
            "the process working on the document died; the document likely triggers a fault in a native library",
            None,
        ),
        JobError::Protocol(_) => status(
            ErrorReason::WorkerCrashed,
            "the process working on the document misbehaved",
            None,
        ),
        JobError::Cancelled => Status::cancelled("the call was cancelled"),
        JobError::Closed => status(
            ErrorReason::Draining,
            "the worker is shutting down",
            Some(Duration::from_secs(1)),
        ),
        JobError::Spawn(_) => status(
            ErrorReason::ResourceExhausted,
            "the worker could not start a process for the job",
            Some(Duration::from_secs(1)),
        ),
    }
}

pub fn from_rejection(rejection: Rejection) -> Status {
    match rejection {
        Rejection::Exhausted { retry_after } => status(
            ErrorReason::ResourceExhausted,
            "the worker is at capacity",
            Some(retry_after),
        ),
        Rejection::Draining => status(
            ErrorReason::Draining,
            "the worker is shutting down",
            Some(Duration::from_secs(1)),
        ),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_refusal_carries_its_reason_and_retry_delay() {
        let status = from_rejection(Rejection::Exhausted {
            retry_after: Duration::from_millis(750),
        });
        assert_eq!(status.code(), Code::ResourceExhausted);
        let details = status.get_error_details();
        assert_eq!(
            details.error_info().map(|info| info.reason.as_str()),
            Some("RESOURCE_EXHAUSTED")
        );
        assert_eq!(
            details.error_info().map(|info| info.domain.as_str()),
            Some(ERROR_DOMAIN)
        );
        assert_eq!(
            details.retry_info().and_then(|info| info.retry_delay),
            Some(Duration::from_millis(750))
        );
    }

    #[test]
    fn every_reason_has_a_code_and_a_name() {
        for reason in [
            ErrorReason::InvalidDocument,
            ErrorReason::Encrypted,
            ErrorReason::TooLarge,
            ErrorReason::PixelBudget,
            ErrorReason::Timeout,
            ErrorReason::ResourceExhausted,
            ErrorReason::Internal,
            ErrorReason::UnsupportedFormat,
            ErrorReason::PageOutOfRange,
            ErrorReason::InvalidRequest,
            ErrorReason::WorkerCrashed,
            ErrorReason::Draining,
        ] {
            assert!(!reason_name(reason).starts_with("ERROR_REASON"));
            assert_ne!(code_for(reason), Code::Ok);
        }
        assert_eq!(reason_name(ErrorReason::Encrypted), "ENCRYPTED");
    }
}
