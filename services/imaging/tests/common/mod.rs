//! A worker for tests: the real supervisor and sandbox pool, hardened
//! children running this crate's binary, and a gRPC client to it. Tests skip
//! when `TRENOVA_PDFIUM_LIBRARY` does not name a libpdfium.

#![allow(dead_code, clippy::print_stderr)]

use std::path::PathBuf;

use prometheus_client::registry::Registry;
use tokio_stream::wrappers::TcpListenerStream;
use tonic::transport::Channel;
use trenova_imaging::config::{Config, ENV_PREFIX};
use trenova_imaging::server::ImagingServer;
use trenova_imaging::server::metrics::ServiceMetrics;
use trenova_imaging::supervisor::{child_spec_for, pool_config};
use trenova_proto::imaging::v1::imaging_service_client::ImagingServiceClient;
use trenova_proto::imaging::v1::imaging_service_server::ImagingServiceServer;
use trenova_sandbox::Pool;
use trenova_service_kit::Admission;
use trenova_service_kit::config::{Source, load_from};

pub const CHUNK: usize = 64 << 10;

pub fn fixture(name: &str) -> Vec<u8> {
    let path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("tests/fixtures")
        .join(name);
    std::fs::read(&path).unwrap_or_else(|err| panic!("read {}: {err}", path.display()))
}

pub fn pdfium() -> Option<String> {
    std::env::var("TRENOVA_PDFIUM_LIBRARY")
        .ok()
        .filter(|path| std::path::Path::new(path).is_file())
}

pub struct Worker {
    pub client: ImagingServiceClient<Channel>,
    pub pool: Pool,
}

/// Starts a worker with `pairs` as extra `TRENOVA_IMAGING__*` settings.
pub async fn start(pairs: &[(&str, &str)]) -> Option<Worker> {
    let Some(library) = pdfium() else {
        eprintln!("skipped: TRENOVA_PDFIUM_LIBRARY is not set");
        return None;
    };
    let mut env: Vec<(String, String)> = vec![
        ("TRENOVA_IMAGING__GRPC__LISTEN".into(), "127.0.0.1:0".into()),
        (
            "TRENOVA_IMAGING__NATIVE__PDFIUM_LIBRARY".into(),
            format!("\"{library}\""),
        ),
        ("TRENOVA_IMAGING__POOL__SIZE".into(), "2".into()),
        (
            "TRENOVA_IMAGING__NATIVE__HEIF_LIBRARY".into(),
            format!(
                "\"{}\"",
                std::env::var("TRENOVA_HEIF_LIBRARY").unwrap_or_default()
            ),
        ),
    ];
    env.extend(
        pairs
            .iter()
            .map(|(k, v)| (format!("{ENV_PREFIX}__{k}"), (*v).to_owned())),
    );
    let config: Config = load_from(
        &Source {
            env_prefix: ENV_PREFIX.into(),
            file: None,
        },
        env,
    )
    .expect("the test configuration is valid");

    let mut registry = Registry::default();
    let admission = Admission::new(config.admission(), &mut registry);
    let pool = Pool::start(
        pool_config(&config),
        child_spec_for(
            &config,
            PathBuf::from(env!("CARGO_BIN_EXE_trenova-imaging")),
        ),
        &mut registry,
    )
    .await
    .expect("the pool starts hardened");
    let metrics = ServiceMetrics::new(&mut registry);
    let server = ImagingServer::new(pool.clone(), admission, config.clone(), metrics);

    let listener = tokio::net::TcpListener::bind("127.0.0.1:0")
        .await
        .expect("bind");
    let addr = listener.local_addr().expect("addr");
    let max = config.grpc.max_message_size();
    tokio::spawn(async move {
        tonic::transport::Server::builder()
            .add_service(
                ImagingServiceServer::new(server)
                    .max_decoding_message_size(max)
                    .max_encoding_message_size(max),
            )
            .serve_with_incoming(TcpListenerStream::new(listener))
            .await
            .expect("serves");
    });
    let channel = Channel::from_shared(format!("http://{addr}"))
        .expect("uri")
        .connect()
        .await
        .expect("connects");
    let client = ImagingServiceClient::new(channel).max_decoding_message_size(max);
    Some(Worker { client, pool })
}

/// Splits a document into the chunk messages `wrap` builds.
pub fn chunked<M>(bytes: &[u8], wrap: impl Fn(Vec<u8>) -> M) -> Vec<M> {
    bytes
        .chunks(CHUNK)
        .map(|chunk| wrap(chunk.to_vec()))
        .collect()
}

pub fn reason(status: &tonic::Status) -> String {
    use tonic_types::StatusExt;
    status
        .get_details_error_info()
        .map(|info| info.reason)
        .unwrap_or_default()
}
