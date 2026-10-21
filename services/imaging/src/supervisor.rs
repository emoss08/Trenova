//! `trenova-imaging serve`: start the pool, serve the contract, report
//! health, and drain on SIGTERM.

use std::io::{Read, Write};
use std::net::TcpStream;
use std::path::PathBuf;
use std::process::ExitCode;
use std::time::Duration;

use prometheus_client::registry::Registry;
use prost::Message;
use trenova_proto::imaging::v1::imaging_service_server::ImagingServiceServer;
use trenova_sandbox::{ChildSpec, HardeningPolicy, Limits, Pool, PoolConfig};
use trenova_service_kit::config::{Source, load};
use trenova_service_kit::grpc::TraceLayer;
use trenova_service_kit::metrics::{Readiness, SharedRegistry, serve as serve_metrics};
use trenova_service_kit::shutdown::drain;
use trenova_service_kit::{Admission, Shutdown, telemetry};

use crate::config::{Config, ENV_PREFIX};
use crate::protocol::ChildInit;
use crate::server::ImagingServer;
use crate::server::metrics::ServiceMetrics;

/// The largest frame a child may send: an output chunk and its framing.
const MAX_FRAME: u32 = 8 << 20;

pub fn run(config_path: Option<PathBuf>) -> ExitCode {
    let config: Config = match load(&Source {
        env_prefix: ENV_PREFIX.into(),
        file: config_path,
    }) {
        Ok(config) => config,
        Err(err) => {
            let _ = writeln!(std::io::stderr(), "trenova-imaging: {err}");
            return ExitCode::from(2);
        }
    };
    let runtime = match tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()
    {
        Ok(runtime) => runtime,
        Err(err) => {
            let _ = writeln!(
                std::io::stderr(),
                "trenova-imaging: start the runtime: {err}"
            );
            return ExitCode::from(2);
        }
    };
    runtime.block_on(async move {
        match Box::pin(serve(config)).await {
            Ok(()) => ExitCode::SUCCESS,
            Err(err) => {
                tracing::error!(error = %err, "the imaging worker stopped");
                let _ = writeln!(std::io::stderr(), "trenova-imaging: {err}");
                ExitCode::FAILURE
            }
        }
    })
}

async fn serve(config: Config) -> Result<(), Box<dyn std::error::Error>> {
    let _telemetry = telemetry::init(
        &config.telemetry,
        "trenova-imaging",
        env!("CARGO_PKG_VERSION"),
    )?;
    let shutdown = Shutdown::new();
    shutdown.trigger_on_signals();

    let mut registry = Registry::default();
    let admission = Admission::new(config.admission(), &mut registry);
    let pool = Pool::start(pool_config(&config), child_spec(&config)?, &mut registry).await?;
    let metrics = ServiceMetrics::new(&mut registry);
    let registry = SharedRegistry::new(registry);
    let readiness = Readiness::new();

    tracing::info!(
        children = pool.size(),
        environment = ?config.environment,
        require_hardening = config.pool.require_hardening,
        "imaging worker started"
    );
    if !config.pool.require_hardening {
        tracing::error!(
            "pool.require_hardening is off: children run untrusted documents without the sandbox's protections"
        );
    }

    let metrics_task = tokio::spawn(serve_metrics(
        config.metrics.listen,
        registry,
        readiness.clone(),
        shutdown.clone(),
    ));

    let (reporter, health) = tonic_health::server::health_reporter();
    let watch = tokio::spawn(watch_health(
        reporter,
        pool.clone(),
        admission.clone(),
        readiness.clone(),
        shutdown.clone(),
    ));

    let max_message = config.grpc.max_message_size();
    let service = ImagingServiceServer::new(ImagingServer::new(
        pool.clone(),
        admission.clone(),
        config.clone(),
        metrics,
    ))
    .max_decoding_message_size(max_message)
    .max_encoding_message_size(max_message);
    let listen = config.grpc.listen;
    tracing::info!(%listen, tls = config.grpc.tls.is_some(), "gRPC listening");
    let stop = {
        let shutdown = shutdown.clone();
        let admission = admission.clone();
        async move {
            shutdown.triggered().await;
            admission.drain();
        }
    };
    let server = config
        .grpc
        .server()?
        .layer(TraceLayer)
        .add_service(health)
        .add_service(service)
        .serve_with_shutdown(listen, stop);

    let mut result = Ok(());
    let finished = drain(&shutdown, config.drain_grace, async {
        result = server.await;
    })
    .await;
    if !finished {
        tracing::warn!("calls were still running when the drain grace ended");
    }
    shutdown.trigger();
    let _ = watch.await;
    pool.close().await;
    let _ = metrics_task.await;
    tracing::info!("imaging worker stopped");
    result.map_err(Into::into)
}

/// Keeps `grpc.health.v1` and `/readyz` in step: ready while the pool has
/// every child, the budget has room for interactive work, and the worker is
/// not draining.
async fn watch_health(
    reporter: tonic_health::server::HealthReporter,
    pool: Pool,
    admission: Admission,
    readiness: Readiness,
    shutdown: Shutdown,
) {
    let mut was = None;
    loop {
        let ready = !shutdown.is_triggered() && pool.healthy() && !admission.saturated();
        readiness.set(ready);
        if was != Some(ready) {
            let status = if ready {
                tonic_health::ServingStatus::Serving
            } else {
                tonic_health::ServingStatus::NotServing
            };
            reporter
                .set_service_status(
                    <ImagingServiceServer<ImagingServer> as tonic::server::NamedService>::NAME,
                    status,
                )
                .await;
            reporter.set_service_status("", status).await;
            was = Some(ready);
        }
        if shutdown.is_triggered() {
            break;
        }
        tokio::time::sleep(Duration::from_millis(500)).await;
    }
}

/// The sandbox pool the configuration describes.
pub fn pool_config(config: &Config) -> PoolConfig {
    let mut read_only = config.native.read_only_paths.clone();
    if let Some(dir) = config.native.pdfium_library.parent() {
        read_only.push(dir.to_path_buf());
    }
    PoolConfig {
        size: config.pool_size(),
        max_jobs_per_child: config.pool.max_jobs_per_child,
        rss_high_water_bytes: config.pool.rss_high_water,
        limits: Limits {
            address_space_bytes: config.pool.address_space,
            cpu_seconds: config.pool.cpu_seconds,
            file_size_bytes: 0,
            open_files: config.pool.open_files,
        },
        max_frame_bytes: MAX_FRAME,
        start_timeout: config.pool.start_timeout,
        policy: HardeningPolicy {
            read_only,
            read_write: Vec::new(),
            landlock: true,
            seccomp: true,
        },
        require_hardening: config.pool.require_hardening,
    }
}

fn child_spec(config: &Config) -> Result<ChildSpec, std::io::Error> {
    Ok(child_spec_for(config, std::env::current_exe()?))
}

/// The child every pool slot runs: `program child`, told where the native
/// libraries are.
pub fn child_spec_for(config: &Config, program: PathBuf) -> ChildSpec {
    ChildSpec {
        program,
        args: vec!["child".into()],
        init: ChildInit {
            pdfium_library: config.native.pdfium_library.to_string_lossy().into_owned(),
            heif_library: config.native.heif_library.clone(),
            max_frame_bytes: MAX_FRAME,
        }
        .encode_to_vec(),
    }
}

/// `trenova-imaging health`: a plain HTTP GET of the readiness endpoint, so
/// the image needs no curl.
pub fn health(url: &str) -> ExitCode {
    let Some(rest) = url.strip_prefix("http://") else {
        let _ = writeln!(std::io::stderr(), "health: only http:// URLs are checked");
        return ExitCode::from(2);
    };
    let (authority, path) = rest.split_once('/').map_or((rest, "/"), |(a, p)| (a, p));
    let probe = || -> std::io::Result<bool> {
        let mut stream = TcpStream::connect(authority)?;
        stream.set_read_timeout(Some(Duration::from_secs(3)))?;
        stream.set_write_timeout(Some(Duration::from_secs(3)))?;
        write!(
            stream,
            "GET /{} HTTP/1.1\r\nHost: {authority}\r\nConnection: close\r\n\r\n",
            path.trim_start_matches('/')
        )?;
        let mut head = [0u8; 16];
        let read = stream.read(&mut head)?;
        Ok(head[..read].starts_with(b"HTTP/1.1 200"))
    };
    match probe() {
        Ok(true) => ExitCode::SUCCESS,
        Ok(false) => ExitCode::from(1),
        Err(err) => {
            let _ = writeln!(std::io::stderr(), "health: {err}");
            ExitCode::from(1)
        }
    }
}
