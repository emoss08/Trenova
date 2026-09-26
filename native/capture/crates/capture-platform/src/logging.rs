//! Daily log files, kept for two weeks.

use std::path::Path;

use tracing_appender::non_blocking::WorkerGuard;
use tracing_subscriber::EnvFilter;

/// How many days of logs are kept.
const LOG_DAYS: usize = 14;
/// Raises or narrows logging, as `RUST_LOG` does: `TRENOVA_CAPTURE_LOG=debug`.
const FILTER_VARIABLE: &str = "TRENOVA_CAPTURE_LOG";

/// Logs to `<dir>\<prefix>.<date>.log`. The guard flushes on drop, so it is
/// held for the life of the process. `None` when the directory is unusable,
/// in which case nothing is logged.
pub fn init(dir: &Path, prefix: &str) -> Option<WorkerGuard> {
    let appender = tracing_appender::rolling::Builder::new()
        .rotation(tracing_appender::rolling::Rotation::DAILY)
        .filename_prefix(prefix)
        .filename_suffix("log")
        .max_log_files(LOG_DAYS)
        .build(dir)
        .ok()?;
    let (writer, guard) = tracing_appender::non_blocking(appender);
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_env(FILTER_VARIABLE).unwrap_or_else(|_| EnvFilter::new("info")),
        )
        .with_writer(writer)
        .with_ansi(false)
        .init();
    Some(guard)
}
