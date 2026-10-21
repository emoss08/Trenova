//! The worker's configuration: an optional TOML file (`--config` or
//! `TRENOVA_IMAGING_CONFIG`) and `TRENOVA_IMAGING__SECTION__KEY` overrides,
//! decoded strictly and validated before anything starts.

use std::net::SocketAddr;
use std::path::PathBuf;
use std::time::Duration;

use serde::Deserialize;
use trenova_service_kit::admission::AdmissionConfig;
use trenova_service_kit::config::{bytes, duration};
use trenova_service_kit::grpc::{Environment, GrpcConfig};
use trenova_service_kit::telemetry::TelemetryConfig;
use trenova_service_kit::{Validate, Validation};

pub const ENV_PREFIX: &str = "TRENOVA_IMAGING";

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Config {
    #[serde(default)]
    pub environment: Environment,
    pub grpc: GrpcConfig,
    #[serde(default)]
    pub metrics: MetricsConfig,
    #[serde(default)]
    pub telemetry: TelemetryConfig,
    /// The admission budget. Defaults: two jobs per child (one running, one
    /// waiting for it) and 75% of the pool's address-space allowance.
    #[serde(default)]
    pub admission: Option<AdmissionConfig>,
    #[serde(default)]
    pub pool: PoolSettings,
    #[serde(default)]
    pub limits: LimitSettings,
    #[serde(default)]
    pub deadlines: DeadlineSettings,
    #[serde(default)]
    pub native: NativeSettings,
    /// How long in-flight calls may run after SIGTERM.
    #[serde(default = "default_grace", deserialize_with = "duration::deserialize")]
    pub drain_grace: Duration,
}

fn default_grace() -> Duration {
    Duration::from_secs(30)
}

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct MetricsConfig {
    #[serde(default = "default_metrics_listen")]
    pub listen: SocketAddr,
}

impl Default for MetricsConfig {
    fn default() -> Self {
        Self {
            listen: default_metrics_listen(),
        }
    }
}

fn default_metrics_listen() -> SocketAddr {
    SocketAddr::from(([0, 0, 0, 0], 9464))
}

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct PoolSettings {
    /// Children, one job each at a time. 0: one per CPU the process may use.
    #[serde(default)]
    pub size: usize,
    #[serde(default = "default_max_jobs")]
    pub max_jobs_per_child: u32,
    #[serde(default = "default_rss", deserialize_with = "bytes::deserialize")]
    pub rss_high_water: u64,
    #[serde(
        default = "default_address_space",
        deserialize_with = "bytes::deserialize"
    )]
    pub address_space: u64,
    #[serde(default = "default_cpu_seconds")]
    pub cpu_seconds: u64,
    #[serde(default = "default_open_files")]
    pub open_files: u64,
    #[serde(
        default = "default_start_timeout",
        deserialize_with = "duration::deserialize"
    )]
    pub start_timeout: Duration,
    /// Refuse to run children without no-new-privs, landlock, seccomp and an
    /// address-space limit. Turning it off is for development machines
    /// whose kernel lacks landlock or seccomp, and is logged at error level
    /// on every child.
    #[serde(default = "default_true")]
    pub require_hardening: bool,
}

impl Default for PoolSettings {
    fn default() -> Self {
        Self {
            size: 0,
            max_jobs_per_child: default_max_jobs(),
            rss_high_water: default_rss(),
            address_space: default_address_space(),
            cpu_seconds: default_cpu_seconds(),
            open_files: default_open_files(),
            start_timeout: default_start_timeout(),
            require_hardening: true,
        }
    }
}

fn default_max_jobs() -> u32 {
    500
}
fn default_rss() -> u64 {
    768 << 20
}
fn default_address_space() -> u64 {
    3 << 30
}
fn default_cpu_seconds() -> u64 {
    3600
}
fn default_open_files() -> u64 {
    64
}
fn default_start_timeout() -> Duration {
    Duration::from_secs(20)
}
fn default_true() -> bool {
    true
}

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct LimitSettings {
    /// The most bytes one call may stream in.
    #[serde(default = "default_max_input", deserialize_with = "bytes::deserialize")]
    pub max_input_bytes: u64,
    /// The most pixels one rendered page may have.
    #[serde(default = "default_max_pixels")]
    pub max_render_pixels: u64,
    /// The most stored pixels a decoded image may have.
    #[serde(default = "default_max_image_pixels")]
    pub max_image_pixels: u64,
    /// The most pages a call that names none processes.
    #[serde(default = "default_max_pages")]
    pub max_pages: u32,
    /// The size of each output chunk.
    #[serde(default = "default_chunk", deserialize_with = "bytes::deserialize")]
    pub chunk_bytes: u64,
    /// The most bytes one capture page may have. A capture call holds no
    /// total: each sub-batch's input is bounded by `max_input_bytes`.
    #[serde(
        default = "default_max_capture_page",
        deserialize_with = "bytes::deserialize"
    )]
    pub max_capture_page_bytes: u64,
    /// Capture pages per child job.
    #[serde(default = "default_capture_batch")]
    pub capture_batch: u32,
    /// Capture jobs one call keeps running at once.
    #[serde(default = "default_capture_parallelism")]
    pub capture_parallelism: usize,
}

impl Default for LimitSettings {
    fn default() -> Self {
        Self {
            max_input_bytes: default_max_input(),
            max_render_pixels: default_max_pixels(),
            max_image_pixels: default_max_image_pixels(),
            max_pages: default_max_pages(),
            chunk_bytes: default_chunk(),
            max_capture_page_bytes: default_max_capture_page(),
            capture_batch: default_capture_batch(),
            capture_parallelism: default_capture_parallelism(),
        }
    }
}

fn default_max_input() -> u64 {
    256 << 20
}
fn default_max_pixels() -> u64 {
    25_000_000
}
fn default_max_image_pixels() -> u64 {
    100_000_000
}
fn default_max_pages() -> u32 {
    2000
}
fn default_chunk() -> u64 {
    256 << 10
}
fn default_max_capture_page() -> u64 {
    20 << 20
}
fn default_capture_batch() -> u32 {
    8
}
fn default_capture_parallelism() -> usize {
    4
}

/// The longest each operation may run when the caller sets no deadline, and
/// the cap on any deadline it does set.
#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DeadlineSettings {
    #[serde(default = "secs::<30>", deserialize_with = "duration::deserialize")]
    pub probe: Duration,
    #[serde(default = "secs::<60>", deserialize_with = "duration::deserialize")]
    pub render: Duration,
    #[serde(default = "secs::<600>", deserialize_with = "duration::deserialize")]
    pub inspect_capture: Duration,
    #[serde(default = "secs::<300>", deserialize_with = "duration::deserialize")]
    pub assemble: Duration,
    #[serde(default = "secs::<60>", deserialize_with = "duration::deserialize")]
    pub normalize_image: Duration,
}

const fn secs<const N: u64>() -> Duration {
    Duration::from_secs(N)
}

impl Default for DeadlineSettings {
    fn default() -> Self {
        Self {
            probe: secs::<30>(),
            render: secs::<60>(),
            inspect_capture: secs::<600>(),
            assemble: secs::<300>(),
            normalize_image: secs::<60>(),
        }
    }
}

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NativeSettings {
    pub pdfium_library: PathBuf,
    /// libheif; empty refuses HEIC and HEIF.
    #[serde(default)]
    pub heif_library: String,
    /// What a child may read once hardened: shared libraries, libheif's
    /// plugins, fonts.
    #[serde(default = "default_read_only")]
    pub read_only_paths: Vec<PathBuf>,
}

impl Default for NativeSettings {
    fn default() -> Self {
        Self {
            pdfium_library: PathBuf::from("/usr/lib/trenova/libpdfium.so"),
            heif_library: "libheif.so.1".into(),
            read_only_paths: default_read_only(),
        }
    }
}

fn default_read_only() -> Vec<PathBuf> {
    [
        "/usr/lib",
        "/usr/lib64",
        "/lib",
        "/lib64",
        "/usr/share/fonts",
    ]
    .iter()
    .map(PathBuf::from)
    .collect()
}

impl Config {
    pub fn pool_size(&self) -> usize {
        if self.pool.size > 0 {
            return self.pool.size;
        }
        std::thread::available_parallelism().map_or(2, std::num::NonZero::get)
    }

    pub fn admission(&self) -> AdmissionConfig {
        self.admission.clone().unwrap_or_else(|| {
            let size = self.pool_size() as u64;
            AdmissionConfig {
                cost_units: size * 2,
                memory_bytes: self.pool.address_space.saturating_mul(size) / 4 * 3,
                interactive_reserve: 0.25,
                max_wait: Duration::from_millis(250),
                max_waiters: 64,
            }
        })
    }

    pub fn deadline(&self, operation: &str) -> Duration {
        let d = &self.deadlines;
        match operation {
            "probe" => d.probe,
            "render" => d.render,
            "inspect_capture" => d.inspect_capture,
            "assemble" => d.assemble,
            _ => d.normalize_image,
        }
    }
}

impl Validate for Config {
    fn validate(&self, v: &mut Validation) {
        self.grpc.validate("grpc", self.environment, v);
        self.telemetry.validate("telemetry", v);
        if let Some(admission) = &self.admission {
            admission.validate("admission", v);
        }
        v.range(self.pool.size, 0..=256, "pool.size");
        v.range(
            self.pool.max_jobs_per_child,
            1..=100_000,
            "pool.max_jobs_per_child",
        );
        v.range(
            self.pool.address_space,
            256 << 20..=64 << 30,
            "pool.address_space",
        );
        v.check(
            self.pool.rss_high_water < self.pool.address_space,
            "pool.rss_high_water",
            "must be below pool.address_space",
        );
        v.range(self.pool.cpu_seconds, 60..=86_400, "pool.cpu_seconds");
        v.range(self.pool.open_files, 16..=4096, "pool.open_files");
        if self.environment == Environment::Production {
            v.check(
                self.pool.require_hardening,
                "pool.require_hardening",
                "cannot be turned off in production",
            );
        }
        v.range(
            self.limits.max_input_bytes,
            1 << 20..=4 << 30,
            "limits.max_input_bytes",
        );
        v.range(
            self.limits.max_render_pixels,
            1_000_000..=200_000_000,
            "limits.max_render_pixels",
        );
        v.range(
            self.limits.max_image_pixels,
            1_000_000..=400_000_000,
            "limits.max_image_pixels",
        );
        v.range(self.limits.max_pages, 1..=10_000, "limits.max_pages");
        v.range(
            self.limits.chunk_bytes,
            16 << 10..=4 << 20,
            "limits.chunk_bytes",
        );
        v.check(
            self.limits.chunk_bytes + (64 << 10) <= self.grpc.max_message_bytes,
            "limits.chunk_bytes",
            "must leave room for framing inside grpc.max_message_bytes",
        );
        v.range(
            self.limits.max_capture_page_bytes,
            64 << 10..=self.limits.max_input_bytes,
            "limits.max_capture_page_bytes",
        );
        v.range(self.limits.capture_batch, 1..=64, "limits.capture_batch");
        v.range(
            self.limits.capture_parallelism,
            1..=64,
            "limits.capture_parallelism",
        );
        for (name, value) in [
            ("deadlines.probe", self.deadlines.probe),
            ("deadlines.render", self.deadlines.render),
            ("deadlines.inspect_capture", self.deadlines.inspect_capture),
            ("deadlines.assemble", self.deadlines.assemble),
            ("deadlines.normalize_image", self.deadlines.normalize_image),
        ] {
            v.check(
                value >= Duration::from_secs(1) && value <= Duration::from_secs(3600),
                name,
                "must be 1s to 1h",
            );
        }
        v.file(&self.native.pdfium_library, "native.pdfium_library");
        v.check(
            self.drain_grace >= Duration::from_secs(1)
                && self.drain_grace <= Duration::from_secs(600),
            "drain_grace",
            "must be 1s to 10m",
        );
    }
}
