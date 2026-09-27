//! The Trenova Capture scan helper.
//!
//! The agent starts one of these per job, in the bitness of the scanner's
//! driver (`trenova-capture-scan-x64.exe` or `-x86.exe`), and talks to it
//! over its standard input and output with the framed protocol in
//! `capture_protocol::helper`. It loads the driver, runs the one job, and
//! exits. A driver that crashes takes this process down, not the agent, and
//! the pages already sent are safe in the agent's spool. It has no network
//! access and holds no credential. Its log goes to standard error, which the
//! agent writes to its own log.

#[cfg(windows)]
mod run;

use std::process::ExitCode;

fn main() -> ExitCode {
    tracing_subscriber::fmt()
        .with_writer(std::io::stderr)
        .with_ansi(false)
        .with_target(false)
        .init();

    #[cfg(windows)]
    {
        run::run()
    }
    #[cfg(not(windows))]
    {
        tracing::error!("Trenova Capture scans on Windows only");
        ExitCode::from(2)
    }
}
