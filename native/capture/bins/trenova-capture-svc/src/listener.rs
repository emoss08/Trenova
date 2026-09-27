//! The HTTP side of the Trenova printer: `POST /ipp/print` on loopback.
//!
//! The IPP Class Driver is the only intended client. The listener binds
//! 127.0.0.1, so nothing off the machine reaches it; a browser page could
//! still aim a request at it, so a request with an `Origin` header, or a
//! `Host` other than this loopback address (DNS rebinding), is refused
//! before its body is read. A body is capped, and at most a few documents
//! are converted at once.

use std::convert::Infallible;
use std::io;
use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;

use bytes::Bytes;
use capture_ipp::{JobHandler, Printer};
use http::header::{ALLOW, CONTENT_LENGTH, CONTENT_TYPE, HOST, ORIGIN};
use http::{Method, Request, Response, StatusCode};
use http_body_util::{BodyExt, Full, Limited};
use hyper::body::Incoming;
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper_util::rt::{TokioIo, TokioTimer};
use hyper_util::server::graceful::GracefulShutdown;
use tokio::net::TcpListener;
use tokio::sync::Semaphore;
use tokio_util::sync::CancellationToken;

/// The printer's path, which the Trenova printer's IPP URL names.
pub const IPP_PATH: &str = "/ipp/print";
const IPP_MIME: &str = "application/ipp";
const HEADER_TIMEOUT: Duration = Duration::from_secs(30);
const BODY_TIMEOUT: Duration = Duration::from_secs(300);
const DRAIN_TIMEOUT: Duration = Duration::from_secs(30);

#[derive(Clone, Copy, Debug)]
pub struct Limits {
    /// The largest request, IPP attributes and document together. Raster
    /// is larger than the PDF it becomes, so this is above the server's
    /// print-job limit.
    pub max_request_bytes: usize,
    /// How many documents are converted and stored at once; the rest wait.
    pub concurrent_jobs: usize,
}

impl Default for Limits {
    fn default() -> Self {
        Self {
            max_request_bytes: 512 << 20,
            concurrent_jobs: 2,
        }
    }
}

struct Context<H: JobHandler> {
    printer: Arc<Printer<H>>,
    limits: Limits,
    port: u16,
    jobs: Semaphore,
}

fn plain(status: StatusCode) -> Response<Full<Bytes>> {
    let mut response = Response::new(Full::new(Bytes::from(
        status.canonical_reason().unwrap_or_default(),
    )));
    *response.status_mut() = status;
    response
}

/// Whether `Host` names this listener: a loopback name, and this port when
/// a port is given.
fn loopback_host(host: &str, port: u16) -> bool {
    let (name, given) = match host.rsplit_once(':') {
        Some((name, given))
            if !name.ends_with(':') && given.bytes().all(|b| b.is_ascii_digit()) =>
        {
            (name, Some(given))
        }
        _ => (host, None),
    };
    let loopback = matches!(
        name.to_ascii_lowercase().as_str(),
        "127.0.0.1" | "localhost" | "[::1]"
    );
    loopback && given.is_none_or(|given| given.parse::<u16>() == Ok(port))
}

/// Everything that can be refused before the body is read.
fn check<H: JobHandler>(
    request: &Request<Incoming>,
    context: &Context<H>,
) -> Result<(), StatusCode> {
    if request.uri().path() != IPP_PATH || request.uri().query().is_some() {
        return Err(StatusCode::NOT_FOUND);
    }
    if request.method() != Method::POST {
        return Err(StatusCode::METHOD_NOT_ALLOWED);
    }
    let headers = request.headers();
    if headers.contains_key(ORIGIN) {
        return Err(StatusCode::FORBIDDEN);
    }
    let host = headers
        .get(HOST)
        .and_then(|h| h.to_str().ok())
        .unwrap_or_default();
    if !loopback_host(host, context.port) {
        return Err(StatusCode::FORBIDDEN);
    }
    let ipp = headers
        .get(CONTENT_TYPE)
        .and_then(|h| h.to_str().ok())
        .and_then(|value| value.split(';').next())
        .is_some_and(|mime| mime.trim().eq_ignore_ascii_case(IPP_MIME));
    if !ipp {
        return Err(StatusCode::UNSUPPORTED_MEDIA_TYPE);
    }
    let declared = headers
        .get(CONTENT_LENGTH)
        .and_then(|h| h.to_str().ok())
        .and_then(|value| value.parse::<u64>().ok());
    if declared.is_some_and(|length| length > context.limits.max_request_bytes as u64) {
        return Err(StatusCode::PAYLOAD_TOO_LARGE);
    }
    Ok(())
}

async fn respond<H: JobHandler + 'static>(
    request: Request<Incoming>,
    context: Arc<Context<H>>,
) -> Result<Response<Full<Bytes>>, Infallible> {
    if let Err(status) = check(&request, &context) {
        let mut response = plain(status);
        if status == StatusCode::METHOD_NOT_ALLOWED {
            response
                .headers_mut()
                .insert(ALLOW, http::HeaderValue::from_static("POST"));
        }
        return Ok(response);
    }
    let body = Limited::new(request.into_body(), context.limits.max_request_bytes);
    let body = match tokio::time::timeout(BODY_TIMEOUT, body.collect()).await {
        Err(_) => return Ok(plain(StatusCode::REQUEST_TIMEOUT)),
        Ok(Err(err)) if err.is::<http_body_util::LengthLimitError>() => {
            return Ok(plain(StatusCode::PAYLOAD_TOO_LARGE));
        }
        Ok(Err(_)) => return Ok(plain(StatusCode::BAD_REQUEST)),
        Ok(Ok(collected)) => collected.to_bytes(),
    };

    let Ok(_permit) = context.jobs.acquire().await else {
        return Ok(plain(StatusCode::SERVICE_UNAVAILABLE));
    };
    let printer = Arc::clone(&context.printer);
    let answer = match tokio::task::spawn_blocking(move || printer.handle(&body)).await {
        Ok(answer) => answer,
        Err(err) => {
            tracing::error!(error = %err, "an IPP request failed");
            return Ok(plain(StatusCode::INTERNAL_SERVER_ERROR));
        }
    };
    let mut response = Response::new(Full::new(Bytes::from(answer)));
    response
        .headers_mut()
        .insert(CONTENT_TYPE, http::HeaderValue::from_static(IPP_MIME));
    Ok(response)
}

/// Binds the printer's loopback address.
pub async fn bind(port: u16) -> io::Result<TcpListener> {
    TcpListener::bind(SocketAddr::from(([127, 0, 0, 1], port))).await
}

/// Serves IPP until `shutdown`, then lets requests in flight finish.
pub async fn serve<H: JobHandler + 'static>(
    listener: TcpListener,
    printer: Arc<Printer<H>>,
    limits: Limits,
    shutdown: CancellationToken,
) -> io::Result<()> {
    let port = listener.local_addr()?.port();
    let context = Arc::new(Context {
        printer,
        limits,
        port,
        jobs: Semaphore::new(limits.concurrent_jobs.max(1)),
    });
    let graceful = GracefulShutdown::new();
    tracing::info!(port, "the Trenova printer is listening");

    loop {
        let (stream, peer) = tokio::select! {
            () = shutdown.cancelled() => break,
            accepted = listener.accept() => match accepted {
                Ok(accepted) => accepted,
                Err(err) => {
                    tracing::warn!(error = %err, "could not accept a connection");
                    tokio::time::sleep(Duration::from_millis(100)).await;
                    continue;
                }
            },
        };
        if !peer.ip().is_loopback() {
            continue;
        }
        let context = Arc::clone(&context);
        let service = service_fn(move |request| respond(request, Arc::clone(&context)));
        let connection = http1::Builder::new()
            .timer(TokioTimer::new())
            .header_read_timeout(HEADER_TIMEOUT)
            .serve_connection(TokioIo::new(stream), service);
        let connection = graceful.watch(connection);
        tokio::spawn(async move {
            if let Err(err) = connection.await {
                tracing::debug!(error = %err, "an IPP connection ended with an error");
            }
        });
    }

    tokio::select! {
        () = graceful.shutdown() => {}
        () = tokio::time::sleep(DRAIN_TIMEOUT) => {
            tracing::warn!("stopped with print requests still in flight");
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::loopback_host;

    #[test]
    fn only_this_loopback_address_is_a_valid_host() {
        for host in [
            "127.0.0.1:8631",
            "localhost:8631",
            "LOCALHOST",
            "127.0.0.1",
            "[::1]:8631",
        ] {
            assert!(loopback_host(host, 8631), "{host}");
        }
        for host in [
            "127.0.0.1:80",
            "evil.example:8631",
            "evil.example",
            "",
            "127.0.0.1:x",
            "[::1]:1",
        ] {
            assert!(!loopback_host(host, 8631), "{host}");
        }
    }
}
