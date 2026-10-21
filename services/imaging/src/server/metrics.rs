//! The worker's RED metrics: calls by operation, priority and outcome, their
//! duration, and the bytes that flowed in and out.

use prometheus_client::encoding::{EncodeLabelSet, EncodeLabelValue};
use prometheus_client::metrics::counter::Counter;
use prometheus_client::metrics::family::Family;
use prometheus_client::metrics::histogram::{Histogram, exponential_buckets};
use prometheus_client::registry::Registry;
use trenova_service_kit::Priority;

#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq, EncodeLabelValue)]
pub enum PriorityLabel {
    Interactive,
    Batch,
}

impl From<Priority> for PriorityLabel {
    fn from(priority: Priority) -> Self {
        match priority {
            Priority::Interactive => Self::Interactive,
            Priority::Batch => Self::Batch,
        }
    }
}

#[derive(Clone, Debug, Hash, PartialEq, Eq, EncodeLabelSet)]
pub struct CallLabels {
    pub operation: &'static str,
    pub priority: PriorityLabel,
    pub code: &'static str,
}

#[derive(Clone, Debug, Hash, PartialEq, Eq, EncodeLabelSet)]
pub struct DurationLabels {
    pub operation: &'static str,
    pub priority: PriorityLabel,
}

#[derive(Clone, Debug, Hash, PartialEq, Eq, EncodeLabelSet)]
pub struct OperationLabel {
    pub operation: &'static str,
}

#[derive(Debug)]
pub struct ServiceMetrics {
    pub calls: Family<CallLabels, Counter>,
    pub duration: Family<DurationLabels, Histogram>,
    pub bytes_in: Family<OperationLabel, Counter>,
    pub bytes_out: Family<OperationLabel, Counter>,
    pub pages: Family<OperationLabel, Counter>,
}

impl ServiceMetrics {
    pub fn new(registry: &mut Registry) -> Self {
        let metrics = Self {
            calls: Family::default(),
            duration: Family::new_with_constructor(|| {
                Histogram::new(exponential_buckets(0.005, 2.0, 16))
            }),
            bytes_in: Family::default(),
            bytes_out: Family::default(),
            pages: Family::default(),
        };
        let sub = registry.sub_registry_with_prefix("imaging");
        sub.register(
            "calls",
            "Calls by operation, priority and gRPC code",
            metrics.calls.clone(),
        );
        sub.register(
            "call_duration_seconds",
            "Call duration by operation and priority",
            metrics.duration.clone(),
        );
        sub.register(
            "input_bytes",
            "Document bytes received",
            metrics.bytes_in.clone(),
        );
        sub.register(
            "output_bytes",
            "Artifact bytes sent",
            metrics.bytes_out.clone(),
        );
        sub.register(
            "pages",
            "Pages produced or inspected",
            metrics.pages.clone(),
        );
        metrics
    }
}

/// The gRPC code's name, as a label.
pub fn code_label(code: tonic::Code) -> &'static str {
    match code {
        tonic::Code::Ok => "ok",
        tonic::Code::Cancelled => "cancelled",
        tonic::Code::InvalidArgument => "invalid_argument",
        tonic::Code::DeadlineExceeded => "deadline_exceeded",
        tonic::Code::FailedPrecondition => "failed_precondition",
        tonic::Code::OutOfRange => "out_of_range",
        tonic::Code::ResourceExhausted => "resource_exhausted",
        tonic::Code::Unavailable => "unavailable",
        tonic::Code::Internal => "internal",
        _ => "other",
    }
}
