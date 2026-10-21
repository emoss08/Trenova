//! The Prometheus registry and the plain HTTP listener that serves it.
//!
//! The listener answers `GET /metrics` in the `OpenMetrics` text format,
//! `GET /livez` with 200 while the process runs, and `GET /readyz` with 200
//! while the service says it is ready and 503 otherwise; everything else is
//! 404. It is separate from the gRPC port so a scraper or a container
//! healthcheck needs no client certificate.

use std::net::SocketAddr;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex, PoisonError};

use bytes::Bytes;
use http::{Method, Request, Response, StatusCode, header};
use http_body_util::Full;
use hyper::body::Incoming;
use hyper::service::service_fn;
use hyper_util::rt::TokioIo;
use prometheus_client::encoding::text::encode;
use prometheus_client::registry::Registry;
use tokio::net::TcpListener;

use crate::shutdown::Shutdown;

const CONTENT_TYPE: &str = "application/openmetrics-text; version=1.0.0; charset=utf-8";

/// A registry shared between the code that registers metrics and the
/// listener that encodes them.
#[derive(Clone, Debug, Default)]
pub struct SharedRegistry {
    inner: Arc<Mutex<Registry>>,
}

impl SharedRegistry {
    pub fn new(registry: Registry) -> Self {
        Self {
            inner: Arc::new(Mutex::new(registry)),
        }
    }

    pub fn encode(&self) -> Result<String, std::fmt::Error> {
        let registry = self.inner.lock().unwrap_or_else(PoisonError::into_inner);
        let mut out = String::new();
        encode(&mut out, &registry)?;
        Ok(out)
    }
}

/// Whether the service takes work, as `/readyz` reports it. Starts not ready.
#[derive(Clone, Debug, Default)]
pub struct Readiness {
    ready: Arc<AtomicBool>,
}

impl Readiness {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn set(&self, ready: bool) {
        self.ready.store(ready, Ordering::Release);
    }

    pub fn is_ready(&self) -> bool {
        self.ready.load(Ordering::Acquire)
    }
}

/// Serves `/metrics`, `/livez` and `/readyz` on `addr` until `shutdown` is
/// triggered.
pub async fn serve(
    addr: SocketAddr,
    registry: SharedRegistry,
    readiness: Readiness,
    shutdown: Shutdown,
) -> std::io::Result<()> {
    let listener = TcpListener::bind(addr).await?;
    tracing::info!(%addr, "metrics listening");
    let stopped = shutdown.triggered();
    tokio::pin!(stopped);

    loop {
        let (stream, _) = tokio::select! {
            accepted = listener.accept() => accepted?,
            () = &mut stopped => return Ok(()),
        };
        let registry = registry.clone();
        let readiness = readiness.clone();
        tokio::spawn(async move {
            let service = service_fn(move |request| {
                let registry = registry.clone();
                let ready = readiness.is_ready();
                async move { Ok::<_, std::convert::Infallible>(respond(&request, &registry, ready)) }
            });
            if let Err(err) = hyper::server::conn::http1::Builder::new()
                .serve_connection(TokioIo::new(stream), service)
                .await
            {
                tracing::debug!(error = %err, "metrics connection ended");
            }
        });
    }
}

fn respond(
    request: &Request<Incoming>,
    registry: &SharedRegistry,
    ready: bool,
) -> Response<Full<Bytes>> {
    let response = |status: StatusCode, content_type: &'static str, body: Bytes| {
        let mut response = Response::new(Full::new(body));
        *response.status_mut() = status;
        response.headers_mut().insert(
            header::CONTENT_TYPE,
            header::HeaderValue::from_static(content_type),
        );
        response
    };
    if request.method() != Method::GET {
        return response(StatusCode::METHOD_NOT_ALLOWED, "text/plain", Bytes::new());
    }
    match request.uri().path() {
        "/metrics" => match registry.encode() {
            Ok(text) => response(StatusCode::OK, CONTENT_TYPE, Bytes::from(text)),
            Err(_) => response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "text/plain",
                Bytes::new(),
            ),
        },
        "/livez" => response(StatusCode::OK, "text/plain", Bytes::from_static(b"ok\n")),
        "/readyz" if ready => {
            response(StatusCode::OK, "text/plain", Bytes::from_static(b"ready\n"))
        }
        "/readyz" => response(
            StatusCode::SERVICE_UNAVAILABLE,
            "text/plain",
            Bytes::from_static(b"not ready\n"),
        ),
        _ => response(StatusCode::NOT_FOUND, "text/plain", Bytes::new()),
    }
}

#[cfg(test)]
mod tests {
    use prometheus_client::metrics::counter::Counter;

    use super::*;

    #[test]
    fn the_registry_encodes_what_was_registered() {
        let mut registry = Registry::default();
        let counter: Counter = Counter::default();
        registry.register("jobs", "Jobs run", counter.clone());
        counter.inc();
        let shared = SharedRegistry::new(registry);
        let text = shared.encode().expect("encodes");
        assert!(text.contains("jobs_total 1"));
        assert!(text.ends_with("# EOF\n"));
    }
}
