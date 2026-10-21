//! Logs and traces.
//!
//! Logs are JSON lines on stderr (or text, for a person at a terminal), one
//! object per event with the span fields it happened in. Traces go to an
//! OTLP collector over gRPC when one is configured. The W3C `traceparent`
//! and `tracestate` headers Go's `otelgrpc` sends are taken as each call's
//! parent, so a call from the TMS and the work it causes here are one trace.
//!
//! Nothing a service logs may carry document content; fields are sizes,
//! counts, durations, page numbers and error kinds.

use std::time::Duration;

use opentelemetry::propagation::{Extractor, TextMapPropagator};
use opentelemetry::trace::TracerProvider as _;
use opentelemetry::{Context, KeyValue};
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::Resource;
use opentelemetry_sdk::propagation::TraceContextPropagator;
use opentelemetry_sdk::trace::{Sampler, SdkTracerProvider};
use serde::Deserialize;
use tracing_subscriber::EnvFilter;
use tracing_subscriber::layer::SubscriberExt;
use tracing_subscriber::util::SubscriberInitExt;

#[derive(Clone, Copy, Debug, Default, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum LogFormat {
    #[default]
    Json,
    Text,
}

#[derive(Clone, Debug, Deserialize, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct TelemetryConfig {
    /// An `EnvFilter` directive: `info`, `trenova_imaging=debug,info`.
    #[serde(default = "default_level")]
    pub log_level: String,
    #[serde(default)]
    pub log_format: LogFormat,
    /// The OTLP gRPC endpoint (`http://otel-collector:4317`); no traces are
    /// exported without one.
    #[serde(default)]
    pub otlp_endpoint: Option<String>,
    /// The share of root traces sampled, 0 to 1. A call whose parent was
    /// sampled is always sampled.
    #[serde(default = "default_ratio")]
    pub trace_sample_ratio: f64,
}

impl Default for TelemetryConfig {
    fn default() -> Self {
        Self {
            log_level: default_level(),
            log_format: LogFormat::Json,
            otlp_endpoint: None,
            trace_sample_ratio: default_ratio(),
        }
    }
}

fn default_level() -> String {
    "info".into()
}

fn default_ratio() -> f64 {
    1.0
}

impl TelemetryConfig {
    pub fn validate(&self, prefix: &str, v: &mut crate::Validation) {
        v.check(
            EnvFilter::try_new(&self.log_level).is_ok(),
            &format!("{prefix}.log_level"),
            "is not a valid filter directive",
        );
        v.range(
            self.trace_sample_ratio,
            0.0..=1.0,
            &format!("{prefix}.trace_sample_ratio"),
        );
        if let Some(endpoint) = &self.otlp_endpoint {
            v.check(
                endpoint.starts_with("http://") || endpoint.starts_with("https://"),
                &format!("{prefix}.otlp_endpoint"),
                "must be an http:// or https:// URL",
            );
        }
    }
}

/// Flushes and stops the trace exporter when dropped.
#[derive(Debug)]
pub struct TelemetryGuard {
    provider: Option<SdkTracerProvider>,
}

impl Drop for TelemetryGuard {
    fn drop(&mut self) {
        if let Some(provider) = self.provider.take()
            && let Err(err) = provider.shutdown_with_timeout(Duration::from_secs(5))
        {
            tracing::warn!(error = %err, "trace exporter did not shut down cleanly");
        }
    }
}

#[derive(Debug, thiserror::Error)]
pub enum TelemetryError {
    #[error("the log filter is not valid: {0}")]
    Filter(String),
    #[error("the trace exporter could not be built: {0}")]
    Exporter(String),
    #[error("a global subscriber is already installed: {0}")]
    Install(String),
}

/// Installs the global subscriber and, when configured, the OTLP exporter.
/// Must run inside a tokio runtime.
pub fn init(
    config: &TelemetryConfig,
    service: &'static str,
    version: &'static str,
) -> Result<TelemetryGuard, TelemetryError> {
    let filter = EnvFilter::try_new(&config.log_level)
        .map_err(|err| TelemetryError::Filter(err.to_string()))?;
    opentelemetry::global::set_text_map_propagator(TraceContextPropagator::new());

    let provider = match &config.otlp_endpoint {
        Some(endpoint) => {
            let exporter = opentelemetry_otlp::SpanExporter::builder()
                .with_tonic()
                .with_endpoint(endpoint.clone())
                .with_timeout(Duration::from_secs(5))
                .build()
                .map_err(|err| TelemetryError::Exporter(err.to_string()))?;
            let resource = Resource::builder()
                .with_service_name(service)
                .with_attribute(KeyValue::new("service.version", version))
                .build();
            Some(
                SdkTracerProvider::builder()
                    .with_batch_exporter(exporter)
                    .with_sampler(Sampler::ParentBased(Box::new(Sampler::TraceIdRatioBased(
                        config.trace_sample_ratio,
                    ))))
                    .with_resource(resource)
                    .build(),
            )
        }
        None => None,
    };

    let otel_layer = provider.as_ref().map(|provider| {
        opentelemetry::global::set_tracer_provider(provider.clone());
        tracing_opentelemetry::layer().with_tracer(provider.tracer(service))
    });

    let registry = tracing_subscriber::registry().with(filter).with(otel_layer);
    let installed = match config.log_format {
        LogFormat::Json => registry
            .with(
                tracing_subscriber::fmt::layer()
                    .json()
                    .with_current_span(true)
                    .with_span_list(false)
                    .with_writer(std::io::stderr),
            )
            .try_init(),
        LogFormat::Text => registry
            .with(tracing_subscriber::fmt::layer().with_writer(std::io::stderr))
            .try_init(),
    };
    installed.map_err(|err| TelemetryError::Install(err.to_string()))?;

    Ok(TelemetryGuard { provider })
}

/// The trace context a request's headers carry.
pub fn parent_context(headers: &http::HeaderMap) -> Context {
    TraceContextPropagator::new().extract(&HeaderExtractor(headers))
}

struct HeaderExtractor<'a>(&'a http::HeaderMap);

impl Extractor for HeaderExtractor<'_> {
    fn get(&self, key: &str) -> Option<&str> {
        self.0.get(key).and_then(|value| value.to_str().ok())
    }

    fn keys(&self) -> Vec<&str> {
        self.0.keys().map(http::HeaderName::as_str).collect()
    }
}

#[cfg(test)]
mod tests {
    use opentelemetry::trace::TraceContextExt;

    use super::*;

    #[test]
    fn a_traceparent_header_becomes_the_parent() {
        let mut headers = http::HeaderMap::new();
        headers.insert(
            "traceparent",
            http::HeaderValue::from_static(
                "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
            ),
        );
        let context = parent_context(&headers);
        let span = context.span();
        let span_context = span.span_context();
        assert!(span_context.is_valid());
        assert_eq!(
            span_context.trace_id().to_string(),
            "0af7651916cd43dd8448eb211c80319c"
        );
        assert!(span_context.is_remote());
    }

    #[test]
    fn a_bad_filter_is_a_validation_problem() {
        let mut v = crate::Validation::new();
        TelemetryConfig {
            log_level: "info,[".into(),
            ..TelemetryConfig::default()
        }
        .validate("telemetry", &mut v);
        assert!(!v.is_empty());
    }
}
