//! `trenova-imaging`: `serve` runs the worker, `child` is the sandboxed
//! process the worker starts for itself, and `health` asks a running worker
//! whether it is ready (the container healthcheck).

use std::path::PathBuf;
use std::process::ExitCode;

use clap::{Parser, Subcommand};

#[derive(Debug, Parser)]
#[command(
    name = "trenova-imaging",
    version,
    about = "Trenova's document and imaging worker"
)]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Debug, Subcommand)]
enum Command {
    /// Run the worker.
    Serve {
        /// A TOML configuration file; `TRENOVA_IMAGING__*` variables override it.
        #[arg(long, env = "TRENOVA_IMAGING_CONFIG")]
        config: Option<PathBuf>,
    },
    /// The sandboxed child process. Started by `serve`, never by hand.
    #[command(hide = true)]
    Child,
    /// Exit 0 when the worker on this host reports ready.
    Health {
        #[arg(long, default_value = "http://127.0.0.1:9464/readyz")]
        url: String,
    },
}

fn main() -> ExitCode {
    let cli = Cli::parse();
    match cli.command {
        Command::Child => trenova_imaging::child::run(),
        Command::Serve { config } => trenova_imaging::supervisor::run(config),
        Command::Health { url } => trenova_imaging::supervisor::health(&url),
    }
}
