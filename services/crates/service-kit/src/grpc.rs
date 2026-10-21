//! The gRPC server every service runs: mutual TLS, `grpc.health.v1`,
//! message and stream limits, HTTP/2 keepalives, and a trace span per call
//! whose parent is the caller's W3C trace context. Server reflection is not
//! offered: the contracts are in `services/proto`, and a production server
//! has no reason to describe itself to whoever connects.

use std::future::Future;
use std::net::SocketAddr;
use std::path::PathBuf;
use std::pin::Pin;
use std::task::{Context, Poll};
use std::time::Duration;

use serde::Deserialize;
use tonic::transport::{Certificate, Identity, Server, ServerTlsConfig};
use tower::{Layer, Service};
use tracing::Instrument;
use tracing_opentelemetry::OpenTelemetrySpanExt;

use crate::Validation;

/// Where the service runs, which decides what its configuration must hold.
#[derive(Clone, Copy, Debug, Default, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum Environment {
    #[default]
    Development,
    Production,
}

#[derive(Clone, Debug, Deserialize, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct TlsConfig {
    /// The server's certificate chain, PEM.
    pub cert: PathBuf,
    /// The server's private key, PEM.
    pub key: PathBuf,
    /// The CA client certificates must chain to. With it, a client without
    /// a certificate from this CA is refused during the handshake.
    #[serde(default)]
    pub client_ca: Option<PathBuf>,
}

#[derive(Clone, Debug, Deserialize, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct GrpcConfig {
    pub listen: SocketAddr,
    #[serde(default)]
    pub tls: Option<TlsConfig>,
    /// The largest message the server decodes or encodes.
    #[serde(
        default = "default_max_message",
        deserialize_with = "crate::config::bytes::deserialize"
    )]
    pub max_message_bytes: u64,
    /// Concurrent streams per connection.
    #[serde(default = "default_streams")]
    pub max_concurrent_streams: u32,
    #[serde(
        default = "default_keepalive",
        deserialize_with = "crate::config::duration::deserialize"
    )]
    pub keepalive_interval: Duration,
    #[serde(
        default = "default_keepalive_timeout",
        deserialize_with = "crate::config::duration::deserialize"
    )]
    pub keepalive_timeout: Duration,
}

fn default_max_message() -> u64 {
    4 << 20
}

fn default_streams() -> u32 {
    256
}

fn default_keepalive() -> Duration {
    Duration::from_secs(30)
}

fn default_keepalive_timeout() -> Duration {
    Duration::from_secs(10)
}

impl GrpcConfig {
    /// Checks the server's settings. Production requires mutual TLS.
    pub fn validate(&self, prefix: &str, environment: Environment, v: &mut Validation) {
        v.range(
            self.max_message_bytes,
            64 << 10..=64 << 20,
            &format!("{prefix}.max_message_bytes"),
        );
        v.range(
            self.max_concurrent_streams,
            1..=10_000,
            &format!("{prefix}.max_concurrent_streams"),
        );
        match &self.tls {
            Some(tls) => {
                v.file(&tls.cert, &format!("{prefix}.tls.cert"));
                v.file(&tls.key, &format!("{prefix}.tls.key"));
                match &tls.client_ca {
                    Some(ca) => v.file(ca, &format!("{prefix}.tls.client_ca")),
                    None => v.check(
                        environment != Environment::Production,
                        &format!("{prefix}.tls.client_ca"),
                        "is required in production: callers must present a client certificate",
                    ),
                }
            }
            None => v.check(
                environment != Environment::Production,
                &format!("{prefix}.tls"),
                "is required in production: the link carries tenant documents",
            ),
        }
    }

    pub fn max_message_size(&self) -> usize {
        usize::try_from(self.max_message_bytes).unwrap_or(usize::MAX)
    }

    /// A tonic server configured from these settings.
    pub fn server(&self) -> Result<Server, GrpcError> {
        let mut server = Server::builder()
            .max_concurrent_streams(self.max_concurrent_streams)
            .http2_keepalive_interval(Some(self.keepalive_interval))
            .http2_keepalive_timeout(Some(self.keepalive_timeout))
            .tcp_nodelay(true);
        if let Some(tls) = &self.tls {
            let read = |path: &PathBuf| {
                std::fs::read(path).map_err(|source| GrpcError::Read {
                    path: path.clone(),
                    source,
                })
            };
            let mut config = ServerTlsConfig::new()
                .identity(Identity::from_pem(read(&tls.cert)?, read(&tls.key)?));
            if let Some(ca) = &tls.client_ca {
                config = config
                    .client_ca_root(Certificate::from_pem(read(ca)?))
                    .client_auth_optional(false);
            }
            server = server
                .tls_config(config)
                .map_err(|err| GrpcError::Tls(err.to_string()))?;
        } else {
            tracing::warn!("gRPC is listening without TLS; this is allowed only in development");
        }
        Ok(server)
    }
}

#[derive(Debug, thiserror::Error)]
pub enum GrpcError {
    #[error("read {path}: {source}")]
    Read {
        path: PathBuf,
        source: std::io::Error,
    },
    #[error("the TLS configuration is not valid: {0}")]
    Tls(String),
}

/// A span per call, child of the caller's trace context.
#[derive(Clone, Copy, Debug, Default)]
pub struct TraceLayer;

impl<S> Layer<S> for TraceLayer {
    type Service = TraceService<S>;

    fn layer(&self, inner: S) -> Self::Service {
        TraceService { inner }
    }
}

#[derive(Clone, Debug)]
pub struct TraceService<S> {
    inner: S,
}

impl<S, B> Service<http::Request<B>> for TraceService<S>
where
    S: Service<http::Request<B>> + Send,
    S::Future: Send + 'static,
{
    type Response = S::Response;
    type Error = S::Error;
    type Future = Pin<Box<dyn Future<Output = Result<S::Response, S::Error>> + Send>>;

    fn poll_ready(&mut self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        self.inner.poll_ready(cx)
    }

    fn call(&mut self, request: http::Request<B>) -> Self::Future {
        let path = request.uri().path();
        let (service, method) = path
            .trim_start_matches('/')
            .split_once('/')
            .unwrap_or(("", path));
        let span = tracing::info_span!(
            "grpc.call",
            otel.name = %path.trim_start_matches('/'),
            otel.kind = "server",
            rpc.system = "grpc",
            rpc.service = %service,
            rpc.method = %method,
        );
        let _ = span.set_parent(crate::telemetry::parent_context(request.headers()));
        Box::pin(self.inner.call(request).instrument(span))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn config(tls: Option<TlsConfig>) -> GrpcConfig {
        GrpcConfig {
            listen: "127.0.0.1:0".parse().expect("addr"),
            tls,
            max_message_bytes: default_max_message(),
            max_concurrent_streams: default_streams(),
            keepalive_interval: default_keepalive(),
            keepalive_timeout: default_keepalive_timeout(),
        }
    }

    #[test]
    fn production_refuses_a_server_without_mutual_tls() {
        let mut v = Validation::new();
        config(None).validate("grpc", Environment::Production, &mut v);
        assert_eq!(v.problems().len(), 1);

        let mut v = Validation::new();
        config(None).validate("grpc", Environment::Development, &mut v);
        assert!(v.is_empty());

        let dir = tempfile::tempdir().expect("tempdir");
        let cert = dir.path().join("cert.pem");
        let key = dir.path().join("key.pem");
        std::fs::write(&cert, "x").expect("write");
        std::fs::write(&key, "x").expect("write");
        let mut v = Validation::new();
        config(Some(TlsConfig {
            cert,
            key,
            client_ca: None,
        }))
        .validate("grpc", Environment::Production, &mut v);
        assert!(v.problems()[0].contains("client_ca"));
    }
}
