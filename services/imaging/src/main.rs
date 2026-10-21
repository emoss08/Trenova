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
    /// Issue the private CA and the server and client certificates a
    /// self-hosted install's TMS and worker authenticate each other with.
    Certs {
        #[arg(long, default_value = "/certs/server")]
        server_dir: PathBuf,
        #[arg(long, default_value = "/certs/client")]
        client_dir: PathBuf,
        #[arg(long, default_value = "/certs/ca")]
        ca_dir: PathBuf,
        /// A DNS name the worker is reached by; repeat for several.
        #[arg(long = "server-name", default_value = "imaging")]
        server_names: Vec<String>,
        #[arg(long, default_value_t = 5)]
        years: i32,
        /// Replace an existing set.
        #[arg(long)]
        force: bool,
    },
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
        Command::Certs {
            server_dir,
            client_dir,
            ca_dir,
            server_names,
            years,
            force,
        } => trenova_imaging::certs::run(&trenova_imaging::certs::CertsRequest {
            server_dir,
            client_dir,
            ca_dir,
            server_names,
            years,
            force,
        }),
    }
}
